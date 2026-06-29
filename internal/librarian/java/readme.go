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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
)

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
	content := string(contentBytes)
	if !strings.Contains(content, "sample-metadata:") {
		return "", nil
	}
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
