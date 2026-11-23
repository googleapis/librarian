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
	argPattern = regexp.MustCompile(`(?m)^ARG\s+(\w+)=(\S+)`)
	// Versions are x.y.z, optionally after a tag prefix such as
	// gapic-generator-v.
	exactVersionPattern = regexp.MustCompile(`^(?:[a-z-]+-v)?\d+\.\d+\.\d+$`)
	// Protobuf releases only have a major and a minor version, such as 29.3.
	protocVersionPattern = regexp.MustCompile(`^\d+\.\d+$`)
	installPattern       = regexp.MustCompile(`\b(?:cargo install|go install|npm install|pnpm add) [^&|;)]*`)
	pinnedPackagePattern = regexp.MustCompile(`@v?(?:\$\{\w+_VERSION\}|\d+\.\d+\.\d+)$`)
	downloadPattern      = regexp.MustCompile(`\b(?:curl|wget)\b.*https?://`)
	downloadFilePattern  = regexp.MustCompile(`\s-[oO]\s+(\S+)`)
	verifyPattern        = regexp.MustCompile(`sha(?:256|512)sum -c|gpg --verify`)
	variablePattern      = regexp.MustCompile(`\$\{?(\w+)`)
)

// TestSupportedLanguages checks that every language image stage in the
// Dockerfile is built. Helper stages, such as php-runtime, have a hyphen in
// their name.
func TestSupportedLanguages(t *testing.T) {
	var want []string
	for _, instruction := range dockerfileInstructions(t) {
		_, stage, ok := parseFrom(instruction)
		if !ok || stage == "build" || stage == "base" || strings.Contains(stage, "-") {
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
	for _, match := range argPattern.FindAllStringSubmatch(readDockerfile(t), -1) {
		name, value := match[1], match[2]
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
