// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const dockerfilePath = "../../../cmd/librarian/Dockerfile"

var (
	argPattern = regexp.MustCompile(`(?m)^ARG\s+(\w+)=("[^"]*"|\S+)`)
	// Versions are x.y.z, optionally after a tag prefix such as
	// gapic-generator-v.
	exactVersionPattern = regexp.MustCompile(`^(?:[a-z-]+-v)?\d+\.\d+\.\d+$`)
	// Protobuf releases only have a major and a minor version, such as 29.3.
	protocVersionPattern    = regexp.MustCompile(`^\d+\.\d+$`)
	installPattern          = regexp.MustCompile(`\b(?:cargo install|go install|npm install|pnpm add) [^&|;)]*`)
	pinnedPackagePattern    = regexp.MustCompile(`@v?(?:\$\{\w+_VERSION\}|\d+\.\d+\.\d+)$`)
	downloadPattern         = regexp.MustCompile(`\b(?:curl|wget)\b.*https?://`)
	downloadFilePattern     = regexp.MustCompile(`\s-[oO]\s+(\S+)`)
	gpgVerifyPattern        = regexp.MustCompile(`\bgpg\b[^&|;]*\s--verify\b`)
	verifyPattern           = regexp.MustCompile(`sha(?:256|512)sum -c|` + gpgVerifyPattern.String())
	variablePattern         = regexp.MustCompile(`\$\{?(\w+)`)
	copyFromPattern         = regexp.MustCompile(`--from=(\S+)`)
	librarianInstallPattern = regexp.MustCompile(`\blibrarian install\b`)
	// GPG keys are pinned by their full 40 character fingerprint.
	fingerprintPattern = regexp.MustCompile(`^[0-9A-F]{40}$`)
	// Language runtimes and Composer must come from official releases, not from
	// apt.
	aptRuntimePattern = regexp.MustCompile(`^(?:python3?|php|composer|ruby|nodejs|npm|openjdk|default-jdk|golang|rustc|cargo|swift)(?:[.\d-]|$)`)
)

// runtimeStages lists, for each language image, the runtimes it needs that are
// built in their own stage. The PHP image reuses the Python and Node runtimes.
var runtimeStages = map[string][]string{
	"go":     {"go"},
	"nodejs": {"nodejs"},
	"php":    {"nodejs", "php", "python"},
	"python": {"python"},
	"ruby":   {"ruby"},
	"swift":  {"go", "swift"},
}

// signedRuntimes lists the runtimes that are verified with the publisher's GPG
// signature.
var signedRuntimes = []string{"swift"}

// TestSupportedLanguages checks that every language image stage in the
// Dockerfile is built. Helper stages, such as php-runtime, have a hyphen in
// their name.
func TestSupportedLanguages(t *testing.T) {
	var want []string
	for stage := range dockerfileStages(t) {
		if stage == "build" || stage == "base" || strings.Contains(stage, "-") {
			continue
		}
		want = append(want, stage)
	}
	slices.Sort(want)
	got := slices.Sorted(slices.Values(supportedLanguages))
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

// TestDockerfileVersionsArePinned checks that every FOO_VERSION argument is an
// exact x.y.z version, or x.y for protoc.
func TestDockerfileVersionsArePinned(t *testing.T) {
	for name, value := range dockerfileArgDefaults(t) {
		if !strings.HasSuffix(name, "_VERSION") {
			continue
		}
		pattern := exactVersionPattern
		if strings.HasSuffix(name, "PROTOC_VERSION") {
			pattern = protocVersionPattern
		}
		if !pattern.MatchString(value) {
			t.Errorf("%s=%s is not an exact version", name, value)
		}
	}
}

// TestDockerfileInstallsArePinned checks that every package installed with
// cargo, go, npm or pnpm is pinned to an exact version. Flags and local paths
// are skipped.
func TestDockerfileInstallsArePinned(t *testing.T) {
	for _, instruction := range dockerfileInstructions(t) {
		for _, install := range installPattern.FindAllString(instruction, -1) {
			for _, packageSpec := range strings.Fields(install)[2:] {
				if strings.HasPrefix(packageSpec, "-") || strings.HasPrefix(packageSpec, ".") ||
					strings.HasPrefix(packageSpec, "/") || packageSpec == `"$PWD"` {
					continue
				}
				if !pinnedPackagePattern.MatchString(packageSpec) {
					t.Errorf("unpinned install: %s", strings.TrimSpace(install))
				}
			}
		}
	}
}

// TestDockerfileDownloadsAreVerified checks that every file downloaded in a
// RUN instruction uses HTTPS, is saved to a file, and that same file is
// verified with a checksum or a signature.
func TestDockerfileDownloadsAreVerified(t *testing.T) {
	for _, instruction := range dockerfileInstructions(t) {
		if !strings.HasPrefix(instruction, "RUN ") {
			continue
		}
		if strings.Contains(instruction, "http://") {
			t.Errorf("insecure download: %s", instruction)
		}
		commands := strings.Split(instruction, "&&")
		for _, command := range commands {
			if !downloadPattern.MatchString(command) {
				continue
			}
			match := downloadFilePattern.FindStringSubmatch(command)
			if match == nil || !slices.ContainsFunc(commands, func(other string) bool {
				return verifyPattern.MatchString(other) && strings.Contains(other, match[1])
			}) {
				t.Errorf("unverified download: %s", strings.TrimSpace(command))
			}
		}
	}
}

// TestDockerfileStageArgsAreDeclared checks that every global ARG used in a
// stage is declared in that stage or a parent stage. Otherwise it expands to
// an empty string.
func TestDockerfileStageArgsAreDeclared(t *testing.T) {
	globalArgs := map[string]bool{}
	stageArgs := map[string]map[string]bool{}
	var stage string
	for _, instruction := range dockerfileInstructions(t) {
		fields := strings.Fields(instruction)
		parent, name, isFrom := parseFrom(instruction)
		switch {
		case isFrom:
			stage = name
			stageArgs[stage] = map[string]bool{}
			for arg := range stageArgs[parent] {
				stageArgs[stage][arg] = true
			}
		case fields[0] == "ARG":
			name, _, _ := strings.Cut(fields[1], "=")
			if stage == "" {
				globalArgs[name] = true
			} else {
				stageArgs[stage][name] = true
			}
		case stage != "":
			for _, match := range variablePattern.FindAllStringSubmatch(instruction, -1) {
				if globalArgs[match[1]] && !stageArgs[stage][match[1]] {
					t.Errorf("stage %s uses undeclared ARG %s", stage, match[1])
				}
			}
		}
	}
}

// TestDockerfileRuntimesAreNotFromApt checks that no language runtime is
// installed with apt, whose versions are not pinned. Packages given through
// ARGs are expanded from their defaults.
func TestDockerfileRuntimesAreNotFromApt(t *testing.T) {
	defaults := dockerfileArgDefaults(t)
	for _, instruction := range dockerfileInstructions(t) {
		for command := range strings.SplitSeq(instruction, "&&") {
			_, packages, found := strings.Cut(command, "apt-get install")
			if !found {
				continue
			}
			packages = os.Expand(packages, func(name string) string { return defaults[name] })
			for packageName := range strings.FieldsSeq(packages) {
				if aptRuntimePattern.MatchString(packageName) {
					t.Errorf("runtime installed with apt: %s", packageName)
				}
			}
		}
	}
}

// TestDockerfileDoesNotRunLibrarianInstall checks that no image runs
// librarian install at build time. The images only provide language runtimes;
// the presubmit installs tools from the repository's librarian.yaml.
func TestDockerfileDoesNotRunLibrarianInstall(t *testing.T) {
	for _, instruction := range dockerfileInstructions(t) {
		if strings.HasPrefix(instruction, "RUN ") && librarianInstallPattern.MatchString(instruction) {
			t.Errorf("librarian install at build time: %s", instruction)
		}
	}
}

// TestDockerfileRuntimeStages checks that each runtime has its own
// <runtime>-runtime stage, and that every language stage needing it uses that
// stage instead of installing it again. The runtime stage must not depend on
// the librarian binary, so a librarian or tool change never rebuilds the
// runtime.
func TestDockerfileRuntimeStages(t *testing.T) {
	stages := dockerfileStages(t)
	for language, runtimes := range runtimeStages {
		t.Run(language, func(t *testing.T) {
			for _, runtime := range runtimes {
				runtimeStage := runtime + "-runtime"
				if _, ok := stages[runtimeStage]; !ok {
					t.Fatalf("missing stage %s", runtimeStage)
				}
				if !stageUses(stages, language, runtimeStage) {
					t.Errorf("stage %s does not use %s", language, runtimeStage)
				}
				if stageUses(stages, runtimeStage, "build") {
					t.Errorf("stage %s depends on the build stage", runtimeStage)
				}
			}
		})
	}
}

// TestDockerfileSignedRuntimes checks that each signed runtime is verified with
// gpg in its runtime stage, and that the keys come from a verified download or
// are fetched by their full fingerprint.
func TestDockerfileSignedRuntimes(t *testing.T) {
	stages := dockerfileStages(t)
	defaults := dockerfileArgDefaults(t)
	for _, runtime := range signedRuntimes {
		t.Run(runtime, func(t *testing.T) {
			stage := runtime + "-runtime"
			index := slices.IndexFunc(stages[stage].instructions, gpgVerifyPattern.MatchString)
			if index < 0 {
				t.Fatalf("stage %s does not run gpg --verify", stage)
			}
			instruction := os.Expand(stages[stage].instructions[index], func(name string) string { return defaults[name] })
			if !strings.Contains(instruction, "--import") && !strings.Contains(instruction, "--recv-keys") {
				t.Errorf("stage %s does not import gpg keys", stage)
			}
			for command := range strings.SplitSeq(instruction, "&&") {
				_, keys, found := strings.Cut(command, "--recv-keys")
				if !found {
					continue
				}
				for key := range strings.FieldsSeq(keys) {
					if !fingerprintPattern.MatchString(key) {
						t.Errorf("gpg key %s is not a full fingerprint", key)
					}
				}
			}
		})
	}
}

// dockerfileStage is a build stage of the Dockerfile.
type dockerfileStage struct {
	parent       string
	instructions []string
}

// dockerfileStages returns the build stages of the Dockerfile by name.
func dockerfileStages(t *testing.T) map[string]dockerfileStage {
	t.Helper()
	stages := map[string]dockerfileStage{}
	var name string
	for _, instruction := range dockerfileInstructions(t) {
		if parent, stage, ok := parseFrom(instruction); ok {
			name = stage
			stages[name] = dockerfileStage{parent: parent}
			continue
		}
		if name != "" {
			stage := stages[name]
			stage.instructions = append(stage.instructions, instruction)
			stages[name] = stage
		}
	}
	return stages
}

// stageUses reports whether a stage is built from, or copies from, the other
// stage, directly or through the stages it uses.
func stageUses(stages map[string]dockerfileStage, name, other string) bool {
	stage, ok := stages[name]
	if !ok {
		return false
	}
	sources := []string{stage.parent}
	for _, instruction := range stage.instructions {
		for _, match := range copyFromPattern.FindAllStringSubmatch(instruction, -1) {
			sources = append(sources, match[1])
		}
	}
	for _, source := range sources {
		if source == other || stageUses(stages, source, other) {
			return true
		}
	}
	return false
}

// parseFrom returns the image and stage name of a FROM instruction, skipping
// flags such as --platform.
func parseFrom(instruction string) (image, stage string, ok bool) {
	fields := slices.DeleteFunc(strings.Fields(instruction), func(field string) bool {
		return strings.HasPrefix(field, "--")
	})
	if len(fields) != 4 || fields[0] != "FROM" || !strings.EqualFold(fields[2], "AS") {
		return "", "", false
	}
	return fields[1], fields[3], true
}

// dockerfileArgDefaults returns the default value of each ARG, without quotes.
func dockerfileArgDefaults(t *testing.T) map[string]string {
	t.Helper()
	defaults := map[string]string{}
	for _, match := range argPattern.FindAllStringSubmatch(readDockerfile(t), -1) {
		defaults[match[1]] = strings.Trim(match[2], `"`)
	}
	return defaults
}

func readDockerfile(t *testing.T) string {
	t.Helper()
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

// dockerfileInstructions returns the Dockerfile instructions, with line
// continuations joined and comments removed.
func dockerfileInstructions(t *testing.T) []string {
	t.Helper()
	var instructions []string
	var current strings.Builder
	for line := range strings.SplitSeq(readDockerfile(t), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		joined, continued := strings.CutSuffix(line, "\\")
		current.WriteString(strings.TrimSpace(joined) + " ")
		if !continued {
			instructions = append(instructions, strings.TrimSpace(current.String()))
			current.Reset()
		}
	}
	return instructions
}
