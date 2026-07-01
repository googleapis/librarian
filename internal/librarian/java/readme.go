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
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	openSnippetRegex  = regexp.MustCompile(`\[START ([a-zA-Z0-9_-]+)\]`)
	closeSnippetRegex = regexp.MustCompile(`\[END ([a-zA-Z0-9_-]+)\]`)
	openExcludeRegex  = regexp.MustCompile(`\[START_EXCLUDE\]`)
	closeExcludeRegex = regexp.MustCompile(`\[END_EXCLUDE\]`)

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

	// errEmptyFile indicates an empty file path was provided.
	errEmptyFile = errors.New("file cannot be empty")
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

// collectSnippetFiles recursively scans dir/samples for Java and XML files containing snippets.
func collectSnippetFiles(dir string) ([]string, error) {
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
		if d.IsDir() {
			// Skip unit test directories and generated snippet output directories.
			if d.Name() == "test" || (d.Name() == "generated" && filepath.Base(filepath.Dir(path)) == "snippets") {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".java" || ext == ".xml" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk samples directory: %w", err)
	}
	return files, nil
}

// extractSnippetsFromFile parses a single file to return a map of tagged code snippets.
// Code between [START <name>] and [END <name>] tags is captured line by line.
// Any code inside [START_EXCLUDE] and [END_EXCLUDE] blocks is omitted.
// Example:
//
//	Input file content:
//	  // [START my_snippet]
//	  void run() {
//	    // [START_EXCLUDE]
//	    secretInit();
//	    // [END_EXCLUDE]
//	    doWork();
//	  }
//	  // [END my_snippet]
//
//	Resulting map entry for "my_snippet":
//	  void run() {
//	    doWork();
//	  }
func extractSnippetsFromFile(file string) (map[string][]string, error) {
	if file == "" {
		return nil, errEmptyFile
	}
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", file, err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	// 10 MB sanity limit to protect system memory.
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
	// Scan through file line by line and capture all open snippets.
	// More than one open block might exist for a given line, so we
	// need openSnippets to track currently active snippet blocks by name
	snippetLines := make(map[string][]string)
	openSnippets := make(map[string]bool)
	excluding := false
	for scanner.Scan() {
		line := scanner.Text()
		// Check for exclusion blocks first; code within EXCLUDE tags is completely skipped.
		if openExcludeRegex.MatchString(line) {
			excluding = true
			continue
		}
		if closeExcludeRegex.MatchString(line) {
			excluding = false
			continue
		}
		if excluding {
			continue
		}
		// Check for snippet start/end tags. Tag lines themselves are not saved.
		openMatch := openSnippetRegex.FindStringSubmatch(line)
		closeMatch := closeSnippetRegex.FindStringSubmatch(line)
		if len(openMatch) > 1 {
			name := openMatch[1]
			openSnippets[name] = true
			if _, exists := snippetLines[name]; !exists {
				snippetLines[name] = nil
			}
			continue
		}
		if len(closeMatch) > 1 {
			delete(openSnippets, closeMatch[1])
			continue
		}
		// Append this line of code to every snippet block currently open.
		for s := range openSnippets {
			snippetLines[s] = append(snippetLines[s], line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed scanning file %s: %w", file, err)
	}
	return snippetLines, nil
}
