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

package ruby

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

var (
	errNoSourceGems = errors.New("no source gems provided to prepareMultiWrapper")
	versionDeclRe   = regexp.MustCompile(`\n(\s+)VERSION = "\d+\.\d+\.\d+"`)
	summaryDeclRe   = regexp.MustCompile(`(?m)gem\.summary(\s*)= "[^\n]+"\n`)
	gemspecDepRe    = regexp.MustCompile(`(?m)(\n  gem\.add_dependency [^\n]+)\nend`)
	readmeLinkRe    = regexp.MustCompile(`^(\[[a-zA-Z0-9_-]+-v\d\w*\]\(https:[^)]+\))[.,]\r?\n?$`)
)

// prepareMultiWrapper combines multiple staged Ruby wrapper gems into a single
// multi-wrapper gem layout.
//
// Rough processing steps:
//  1. Gem resolution: Identifies the primary (mainGem) and secondary (otherGems) constituent gems.
//     The 1st gem in the sourceGems list (corresponding to the 1st entry in the library's apis list)
//     is treated as mainGem, unless another source gem explicitly matches finalGem (in which case
//     it is moved to the front as mainGem).
//  2. Staging main gem: Moves all generated files from stagingDir/<mainGem> into stagingDir root.
//  3. Adjusting primary files (if mainGem != finalGem):
//     When the top-level gem name differs from all sub-gem names (e.g. google-cloud-beyond_corp):
//     - Renames and adjusts the entrypoint, .gemspec, .rubocop.yml, .yardopts, .repo-metadata.json,
//     AUTHENTICATION.md, and README.md.
//     - Disables the version constant in the sub-gem's version.rb.
//     - Generates synthetic version.rb and entrypoint for the suite gem.
//  4. Expanding multi-wrapper assets (for otherGems):
//     - Expands the top-level entrypoint (lib/<finalGem>.rb) to require secondary client modules.
//     - Expands Gemfile local_dependencies to include secondary versioned dependencies.
//     - Expands <finalGem>.gemspec by extracting gem.add_dependency entries from secondary gemspecs.
//     - Expands README.md with client links for secondary APIs.
//  5. Copying minimal files & disabling nested versions:
//     - Copies lib/ and test/ files from stagingDir/<otherGem> into stagingDir/lib and stagingDir/test.
//     - Disables unused nested VERSION constants in otherGems' version.rb files.
//     - Cleans up intermediate staging subdirectories.
func prepareMultiWrapper(stagingDir string, sourceGems []string, prettyName, finalGem string) error {
	if len(sourceGems) == 0 {
		return errNoSourceGems
	}
	gems := slices.Clone(sourceGems)
	if slices.Contains(gems, finalGem) && gems[0] != finalGem {
		idx := slices.Index(gems, finalGem)
		gems = append(gems[:idx], gems[idx+1:]...)
		gems = append([]string{finalGem}, gems...)
	}
	mainGem := gems[0]
	otherGems := gems[1:]
	mainPrettyName, err := readPrettyName(stagingDir, mainGem)
	if err != nil {
		return err
	}
	if prettyName == "" {
		prettyName = mainPrettyName
	}
	if err := copyAllFiles(stagingDir, mainGem); err != nil {
		return err
	}
	if mainGem != finalGem {
		if err := adjustEntrypoint(stagingDir, mainGem, finalGem); err != nil {
			return err
		}
		if err := adjustRepoMetadata(stagingDir, mainGem, finalGem, prettyName); err != nil {
			return err
		}
		if err := adjustRubocopYml(stagingDir, mainGem, finalGem); err != nil {
			return err
		}
		if err := adjustYardopts(stagingDir, mainPrettyName, prettyName); err != nil {
			return err
		}
		if err := adjustGemspec(stagingDir, mainGem, finalGem, prettyName); err != nil {
			return err
		}
		if err := adjustGemfile(stagingDir, mainGem, finalGem); err != nil {
			return err
		}
		if err := adjustAuthenticationMd(stagingDir, mainGem, finalGem); err != nil {
			return err
		}
		if err := adjustReadmeMd(stagingDir, mainGem, finalGem, mainPrettyName, prettyName); err != nil {
			return err
		}
		if err := disableVersion(stagingDir, mainGem); err != nil {
			return err
		}
		if err := createSyntheticVersion(stagingDir, finalGem); err != nil {
			return err
		}
		if err := createSyntheticMain(stagingDir, finalGem); err != nil {
			return err
		}
	}
	if len(otherGems) > 0 {
		if err := expandGemEntrypoint(stagingDir, finalGem, otherGems); err != nil {
			return err
		}
		if err := expandGemfile(stagingDir, otherGems); err != nil {
			return err
		}
		if err := expandGemspecDependencies(stagingDir, finalGem, otherGems); err != nil {
			return err
		}
		if err := expandReadmeMd(stagingDir, mainGem, otherGems); err != nil {
			return err
		}
	}
	for _, name := range otherGems {
		if err := copyMinimalFiles(stagingDir, name); err != nil {
			return err
		}
		if err := disableVersion(stagingDir, name); err != nil {
			return err
		}
	}
	return nil
}

func readPrettyName(stagingDir, gemName string) (string, error) {
	data, err := os.ReadFile(filepath.Join(stagingDir, gemName, ".repo-metadata.json"))
	if err != nil {
		return "", err
	}
	var meta struct {
		NamePretty string `json:"name_pretty"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", fmt.Errorf("parsing .repo-metadata.json for %s: %w", gemName, err)
	}
	return meta.NamePretty, nil
}

func copyAllFiles(stagingDir, fromGem string) error {
	fromDir := filepath.Join(stagingDir, fromGem)
	entries, err := os.ReadDir(fromDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		src := filepath.Join(fromDir, entry.Name())
		dst := filepath.Join(stagingDir, entry.Name())
		if err := os.Rename(src, dst); err != nil {
			return err
		}
	}
	return os.RemoveAll(fromDir)
}

func adjustEntrypoint(stagingDir, mainGem, finalGem string) error {
	srcPath := filepath.Join(stagingDir, "lib", mainGem+".rb")
	contentBytes, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	contentBytes = bytes.ReplaceAll(contentBytes, []byte(`"`+makePath(mainGem)+`"`), []byte(`"`+makePath(finalGem)+`"`))
	contentBytes = bytes.ReplaceAll(contentBytes, []byte(makeConstant(mainGem)+"::VERSION"), []byte(makeConstant(finalGem)+"::VERSION"))
	dstPath := filepath.Join(stagingDir, "lib", finalGem+".rb")
	if err := os.WriteFile(dstPath, contentBytes, 0o644); err != nil {
		return err
	}
	return os.Remove(srcPath)
}

func adjustRepoMetadata(stagingDir, mainGem, finalGem, prettyName string) error {
	metaPath := filepath.Join(stagingDir, ".repo-metadata.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return err
	}
	var meta map[string]any
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("parsing %s: %w", metaPath, err)
	}
	if clientDoc, ok := meta["client_documentation"].(string); ok {
		meta["client_documentation"] = strings.ReplaceAll(clientDoc, mainGem+"/latest", finalGem+"/latest")
	}
	meta["distribution_name"] = finalGem
	meta["name_pretty"] = prettyName
	out, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling %s: %w", metaPath, err)
	}
	return os.WriteFile(metaPath, append(out, '\n'), 0o644)
}

func adjustRubocopYml(stagingDir, mainGem, finalGem string) error {
	path := filepath.Join(stagingDir, ".rubocop.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	data = bytes.ReplaceAll(data, []byte(`"`+mainGem+`.gemspec"`), []byte(`"`+finalGem+`.gemspec"`))
	data = bytes.ReplaceAll(data, []byte(`"lib/`+mainGem+`.rb"`), []byte(`"lib/`+finalGem+`.rb"`))
	return os.WriteFile(path, data, 0o644)
}

func adjustYardopts(stagingDir, mainPrettyName, prettyName string) error {
	path := filepath.Join(stagingDir, ".yardopts")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	data = bytes.ReplaceAll(data, []byte(`"`+mainPrettyName+`"`), []byte(`"`+prettyName+`"`))
	return os.WriteFile(path, data, 0o644)
}

func adjustGemspec(stagingDir, mainGem, finalGem, prettyName string) error {
	srcPath := filepath.Join(stagingDir, mainGem+".gemspec")
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	data = bytes.Replace(data, []byte(`"lib/`+makePath(mainGem)+`/version"`), []byte(`"lib/`+makePath(finalGem)+`/version"`), 1)
	data = bytes.ReplaceAll(data, []byte(`"`+mainGem+`"`), []byte(`"`+finalGem+`"`))
	data = bytes.ReplaceAll(data, []byte(makeConstant(mainGem)+"::VERSION"), []byte(makeConstant(finalGem)+"::VERSION"))
	data = summaryDeclRe.ReplaceAll(data, []byte(`gem.summary${1}= "API client library for the `+prettyName+`"`+"\n"))
	dstPath := filepath.Join(stagingDir, finalGem+".gemspec")
	if err := os.WriteFile(dstPath, data, 0o644); err != nil {
		return err
	}
	return os.Remove(srcPath)
}

func adjustGemfile(stagingDir, mainGem, finalGem string) error {
	path := filepath.Join(stagingDir, "Gemfile")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	data = bytes.ReplaceAll(data, []byte(`"`+mainGem+`.gemspec"`), []byte(`"`+finalGem+`.gemspec"`))
	return os.WriteFile(path, data, 0o644)
}

func adjustAuthenticationMd(stagingDir, mainGem, finalGem string) error {
	path := filepath.Join(stagingDir, "AUTHENTICATION.md")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	wordRe := regexp.MustCompile(`([^\w-])` + regexp.QuoteMeta(mainGem) + `([^\w-])`)
	data = wordRe.ReplaceAll(data, []byte(`${1}`+finalGem+`${2}`))
	data = bytes.ReplaceAll(data, []byte(`"`+makePath(mainGem)+`"`), []byte(`"`+makePath(finalGem)+`"`))
	return os.WriteFile(path, data, 0o644)
}

func adjustReadmeMd(stagingDir, mainGem, finalGem, mainPrettyName, prettyName string) error {
	path := filepath.Join(stagingDir, "README.md")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	wordRe := regexp.MustCompile(`([^\w-])` + regexp.QuoteMeta(mainGem) + `([^\w-])`)
	data = wordRe.ReplaceAll(data, []byte(`${1}`+finalGem+`${2}`))
	data = bytes.ReplaceAll(data, []byte(mainPrettyName), []byte(prettyName))
	return os.WriteFile(path, data, 0o644)
}

func disableVersion(stagingDir, gemName string) error {
	path := filepath.Join(stagingDir, "lib", makePath(gemName), "version.rb")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	data = versionDeclRe.ReplaceAll(data, []byte("\n${1}# @private Unused\n${1}VERSION = \"\""))
	return os.WriteFile(path, data, 0o644)
}

func createSyntheticVersion(stagingDir, finalGem string) error {
	path := filepath.Join(stagingDir, "lib", makePath(finalGem), "version.rb")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString(fileHeader())
	modules := strings.Split(finalGem, "-")
	for i, mod := range modules {
		indent := strings.Repeat("  ", i)
		fmt.Fprintf(&sb, "%smodule %s\n", indent, pascalize(mod))
	}
	indent := strings.Repeat("  ", len(modules))
	fmt.Fprintf(&sb, "%sVERSION = \"0.0.1\"\n", indent)
	for i := range slices.Backward(modules) {
		indent := strings.Repeat("  ", i)
		fmt.Fprintf(&sb, "%send\n", indent)
	}
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

func createSyntheticMain(stagingDir, finalGem string) error {
	path := filepath.Join(stagingDir, "lib", makePath(finalGem)+".rb")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf("%srequire %q\n", fileHeader(), "lib/"+finalGem)
	return os.WriteFile(path, []byte(content), 0o644)
}

func expandGemEntrypoint(stagingDir, finalGem string, otherGems []string) error {
	path := filepath.Join(stagingDir, "lib", finalGem+".rb")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, gemName := range otherGems {
		line := fmt.Sprintf("require %q unless defined? %s::VERSION\n", makePath(gemName), makeConstant(gemName))
		if _, err := f.WriteString(line); err != nil {
			return err
		}
	}
	return nil
}

func expandGemfile(stagingDir string, otherGems []string) error {
	path := filepath.Join(stagingDir, "Gemfile")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	localDepsRe := regexp.MustCompile(`local_dependencies = \[(.*)\]`)
	match := localDepsRe.FindStringSubmatch(string(data))
	if len(match) < 2 {
		return nil
	}
	deps := parseArrayLiteral(match[1])
	for _, gemName := range otherGems {
		otherPath := filepath.Join(stagingDir, gemName, "Gemfile")
		otherData, err := os.ReadFile(otherPath)
		if err != nil {
			continue
		}
		otherMatch := localDepsRe.FindStringSubmatch(string(otherData))
		if len(otherMatch) >= 2 {
			deps = append(deps, parseArrayLiteral(otherMatch[1])...)
		}
	}
	var uniqueDeps []string
	seen := make(map[string]bool)
	for _, dep := range deps {
		if dep != "" && !seen[dep] {
			seen[dep] = true
			uniqueDeps = append(uniqueDeps, dep)
		}
	}
	content := localDepsRe.ReplaceAllString(string(data), fmt.Sprintf("local_dependencies = [%s]", strings.Join(uniqueDeps, ", ")))
	return os.WriteFile(path, []byte(content), 0o644)
}

func parseArrayLiteral(s string) []string {
	var items []string
	for item := range strings.SplitSeq(s, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func expandGemspecDependencies(stagingDir, finalGem string, otherGems []string) error {
	var lines []string
	for _, gemName := range otherGems {
		gemspecPath := filepath.Join(stagingDir, gemName, gemName+".gemspec")
		data, err := os.ReadFile(gemspecPath)
		if err != nil {
			return err
		}
		prefix := `  gem.add_dependency "` + gemName
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.HasPrefix(line, prefix) {
				lines = append(lines, line)
			}
		}
	}
	if len(lines) == 0 {
		return nil
	}
	targetGemspec := filepath.Join(stagingDir, finalGem+".gemspec")
	data, err := os.ReadFile(targetGemspec)
	if err != nil {
		return err
	}
	replacement := fmt.Sprintf("${1}\n%s\nend", strings.Join(lines, "\n"))
	content := gemspecDepRe.ReplaceAllString(string(data), replacement)
	return os.WriteFile(targetGemspec, []byte(content), 0o644)
}

func expandReadmeMd(stagingDir, mainGem string, otherGems []string) error {
	var links []string
	for _, gemName := range otherGems {
		readmePath := filepath.Join(stagingDir, gemName, "README.md")
		data, err := os.ReadFile(readmePath)
		if err != nil {
			continue
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			trimmed := strings.TrimRight(line, "\r") + "\n"
			if m := readmeLinkRe.FindStringSubmatch(trimmed); len(m) >= 2 {
				links = append(links, m[1])
				break
			}
		}
	}
	if len(links) == 0 {
		return nil
	}
	joinedLinks := strings.Join(links, ",\n")
	readmePath := filepath.Join(stagingDir, "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	targetRe := regexp.MustCompile(`(?m)(\n\[` + regexp.QuoteMeta(mainGem) + `-v\d\w*\]\(https:[^)]+\))\.\n`)
	content := targetRe.ReplaceAllString(string(data), fmt.Sprintf("${1},\n%s.\n", joinedLinks))
	return os.WriteFile(readmePath, []byte(content), 0o644)
}

func copyMinimalFiles(stagingDir, gemName string) error {
	srcDir := filepath.Join(stagingDir, gemName)
	for _, sub := range []string{"lib", "test"} {
		subDir := filepath.Join(srcDir, sub)
		err := filepath.WalkDir(subDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".rb") {
				return nil
			}
			rel, err := filepath.Rel(srcDir, path)
			if err != nil {
				return err
			}
			parts := strings.Split(filepath.ToSlash(rel), "/")
			if len(parts) < 3 {
				return nil
			}
			dst := filepath.Join(stagingDir, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			return os.Rename(path, dst)
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return os.RemoveAll(srcDir)
}

func makePath(gemName string) string {
	return strings.ReplaceAll(gemName, "-", "/")
}

func makeConstant(gemName string) string {
	parts := strings.Split(gemName, "-")
	var constantParts []string
	for _, part := range parts {
		constantParts = append(constantParts, pascalize(part))
	}
	return strings.Join(constantParts, "::")
}

func pascalize(str string) string {
	parts := strings.Split(str, "_")
	var result strings.Builder
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		result.WriteString(strings.ToUpper(part[:1]) + strings.ToLower(part[1:]))
	}
	return result.String()
}

func fileHeader() string {
	return fmt.Sprintf(`# frozen_string_literal: true

# Copyright %d Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

`, time.Now().Year())
}
