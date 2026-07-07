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

// Package postprocessing provides tools for the YAML-based postprocessing workflow.
package postprocessing

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/filesystem"
)

var (
	// errTextNotFound is returned when the target text or pattern is not found in the file.
	errTextNotFound = errors.New("text not found")

	// errEmptyOriginal is returned when the original text to replace is empty.
	errEmptyOriginal = errors.New("original text to replace cannot be empty")

	// errEmptyPattern is returned when the regex pattern to replace is empty.
	errEmptyPattern = errors.New("regex pattern cannot be empty")
)

// CopyFile copies a single file from the src path to the dst path.
// It acts as a wrapper around filesystem.CopyFile to provide a unified
// interface for all postprocessing file operations.
func CopyFile(src, dst string) error {
	return filesystem.CopyFile(src, dst)
}

// Replace finds and replaces exact text in a file.
// It returns an error if the target file does not exist or if the text is not found.
func Replace(path, original, replacement string) error {
	if original == "" {
		return errEmptyOriginal
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	oldBytes := []byte(original)
	if !bytes.Contains(content, oldBytes) {
		return fmt.Errorf("%w: %q in file %s", errTextNotFound, original, path)
	}
	newContent := bytes.ReplaceAll(content, oldBytes, []byte(replacement))
	return os.WriteFile(path, newContent, 0644)
}

// ReplaceRegex finds and replaces text in a file using a regular expression.
// It returns an error if the target file does not exist or if the pattern matches no text.
func ReplaceRegex(path, pattern, replacement string) error {
	if pattern == "" {
		return errEmptyPattern
	}
	// Default to multiline mode so ^ and $ match per-line.
	if !strings.HasPrefix(pattern, "(?") {
		pattern = "(?m)" + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !re.Match(content) {
		return fmt.Errorf("%w: pattern %q in file %s", errTextNotFound, pattern, path)
	}
	newContent := re.ReplaceAll(content, []byte(replacement))
	return os.WriteFile(path, newContent, 0644)
}

// RemoveFiles removes all files in outDir matching the given patterns (exact paths or globs).
func RemoveFiles(outDir string, removePatterns []string) error {
	for _, rem := range removePatterns {
		if err := applyToFiles(outDir, rem, os.Remove); err != nil {
			return err
		}
	}
	return nil
}

// applyToFiles executes action on files matching pathPattern under outDir.
// Note: Uses filepath.Glob (* only, ** is not supported).
func applyToFiles(outDir string, pathPattern string, action func(string) error) error {
	files, err := filepath.Glob(filepath.Join(outDir, pathPattern))
	if err != nil {
		return fmt.Errorf("failed to resolve glob for %s: %w", pathPattern, err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no files match pattern %q in %s: %w", pathPattern, outDir, fs.ErrNotExist)
	}
	// Reverse sort so children are processed before parent directories.
	slices.Sort(files)
	slices.Reverse(files)
	for _, file := range files {
		if err := action(file); err != nil {
			return err
		}
	}
	return nil
}
