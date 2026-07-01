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

package java

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	// Matches lowercase/digit followed by uppercase (e.g., "FooBar" -> "Foo Bar").
	camelCaseRegexp = regexp.MustCompile(`([a-z0-9])([A-Z])`)

	// reTitle matches a "sample-metadata:" marker followed immediately by a "title:" line on the next comment line.
	reTitle = regexp.MustCompile(`sample-metadata:\s*\n\s*(?://|#)\s*title:\s*(.*)`)

	// errMissingTitle indicates the "title:" line is missing immediately following "sample-metadata:".
	errMissingTitle = errors.New("missing title line immediately following sample-metadata")

	// errEmptyTitle indicates the extracted title value is empty.
	errEmptyTitle = errors.New("title value cannot be empty")

	// errEmptyDir indicates the provided directory path is empty.
	errEmptyDir = errors.New("dir cannot be empty")
)

// codeSample represents a discovered Java code sample along with its derived title.
type codeSample struct {
	Title string
	File  string
}

// extractSamples locates production Java sample files and returns parsed codeSample structs
// containing display titles and relative paths for README rendering.
func extractSamples(dir string) ([]codeSample, error) {
	if dir == "" {
		return nil, errEmptyDir
	}
	files, err := collectSampleFiles(dir)
	if err != nil {
		return nil, err
	}
	var samples []codeSample
	for _, file := range files {
		sample, err := parseCodeSample(dir, file)
		if err != nil {
			return nil, err
		}
		samples = append(samples, *sample)
	}
	return samples, nil
}

// collectSampleFiles recursively scans dir/samples for Java production files.
func collectSampleFiles(dir string) ([]string, error) {
	samplesDir := filepath.Join(dir, "samples")
	if _, err := os.Stat(samplesDir); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to stat samples directory: %w", err)
	}
	var files []string
	err := filepath.WalkDir(samplesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if isProductionSample(rel) {
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk samples directory: %w", err)
	}
	return files, nil
}

// parseCodeSample reads a Java sample file and constructs a codeSample struct with its title and relative path.
func parseCodeSample(dir, file string) (*codeSample, error) {
	// Derive default title by stripping extension and converting CamelCase to space-separated words.
	base := strings.TrimSuffix(filepath.Base(file), ".java")
	title := decamelize(base)
	titleOverride, err := extractTitle(filepath.Join(dir, file))
	if err != nil {
		return nil, fmt.Errorf("failed to extract title from %s: %w", file, err)
	}
	if titleOverride != "" {
		title = titleOverride
	}
	return &codeSample{
		Title: title,
		// Normalize path separators to forward slashes for Markdown links in README.
		File: filepath.ToSlash(file),
	}, nil
}

// decamelize converts CamelCase string to space-separated string (e.g. "CamelCase" -> "Camel Case").
func decamelize(value string) string {
	return strings.TrimSpace(camelCaseRegexp.ReplaceAllString(value, `$1 $2`))
}

// isProductionSample reports whether the given path represents a production Java source file
// located under a standard "src/main/java" path.
func isProductionSample(path string) bool {
	slashed := filepath.ToSlash(path)
	return strings.HasSuffix(slashed, ".java") &&
		(strings.HasPrefix(slashed, "src/main/java/") || strings.Contains(slashed, "/src/main/java/"))
}

// extractTitle reads a file from disk and extracts the title override from Java comment blocks.
// It expects a "title:" line to immediately follow the "sample-metadata:" marker.
// Returns an error if the file cannot be read, or if the marker is present but the title line
// is missing, malformed, or empty.
func extractTitle(filePath string) (string, error) {
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	if !bytes.Contains(contentBytes, []byte("sample-metadata:")) {
		return "", nil
	}
	content := string(contentBytes)
	matches := reTitle.FindStringSubmatch(content)
	if len(matches) < 2 {
		return "", errMissingTitle
	}
	// Trim surrounding whitespace, quotes, and carriage returns.
	title := strings.Trim(matches[1], " \t\"'\r\n")
	if title == "" {
		return "", errEmptyTitle
	}
	return title, nil
}

// toCamelCase converts snake_case, kebab-case, or space-separated strings into CamelCase identifiers.
func toCamelCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	var sb strings.Builder
	for _, p := range parts {
		r, size := utf8.DecodeRuneInString(p)
		sb.WriteRune(unicode.ToUpper(r))
		sb.WriteString(p[size:])
	}
	return sb.String()
}

// parseGroupIDArtifactID extracts GroupID and ArtifactID from a Maven distribution name.
func parseGroupIDArtifactID(distributionName string) (string, string) {
	groupID, artifactID, _ := strings.Cut(distributionName, ":")
	return groupID, artifactID
}

// parseRepoShortName extracts the short repository name from the full repo path.
func parseRepoShortName(repo string) string {
	if repo == "" {
		return ""
	}
	if i := strings.LastIndexByte(repo, '/'); i >= 0 {
		return repo[i+1:]
	}
	return repo
}
