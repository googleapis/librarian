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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
)

var (
	// errTextNotFound is returned when the target text or pattern is not found in the file.
	errTextNotFound = errors.New("text not found")

	// errEmptyOriginal is returned when the original text to replace is empty.
	errEmptyOriginal = errors.New("original text to replace cannot be empty")

	// errEmptyPattern is returned when the regex pattern to replace is empty.
	errEmptyPattern = errors.New("regex pattern cannot be empty")

	// errUnsupportedMethodAction is returned when a method operation action is not supported.
	errUnsupportedMethodAction = errors.New("unsupported method operation action")

	// errSameSourceAndDestination is returned when Src and Dst resolve to the same path.
	errSameSourceAndDestination = errors.New("src and dst must be different")
)

// Apply executes all configured post-processing operations against outDir in sequential order.
func Apply(outDir string, cfg *config.Postprocess) error {
	if cfg == nil {
		return nil
	}
	if err := CopyFiles(outDir, cfg.CopyFile); err != nil {
		return fmt.Errorf("failed to copy files: %w", err)
	}
	if err := RemoveFiles(outDir, cfg.RemoveFile); err != nil {
		return fmt.Errorf("failed to remove files: %w", err)
	}
	if err := ReplaceAll(outDir, cfg.Replace); err != nil {
		return fmt.Errorf("failed to replace all: %w", err)
	}
	if err := ReplaceRegexAll(outDir, cfg.ReplaceRegex); err != nil {
		return fmt.Errorf("failed to replace regex all: %w", err)
	}
	if err := ApplyMethodOperations(outDir, cfg.MethodOperations); err != nil {
		return fmt.Errorf("failed to apply method operations: %w", err)
	}
	return nil
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
	return os.WriteFile(path, newContent, 0o644)
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
	return os.WriteFile(path, newContent, 0o644)
}

// CopyFiles copies files specified by copyConfigs from src to dst inside outDir.
func CopyFiles(outDir string, copyConfigs []config.CopyConfig) error {
	for _, c := range copyConfigs {
		srcAbs := filepath.Join(outDir, c.Src)
		dstAbs := filepath.Join(outDir, c.Dst)
		if srcAbs == dstAbs {
			return fmt.Errorf("invalid copy config for %s: %w", c.Src, errSameSourceAndDestination)
		}
		if err := os.MkdirAll(filepath.Dir(dstAbs), 0o755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", c.Dst, err)
		}
		if err := filesystem.CopyFile(srcAbs, dstAbs); err != nil {
			return fmt.Errorf("failed to copy file from %s to %s: %w", c.Src, c.Dst, err)
		}
	}
	return nil
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

// ReplaceAll applies exact text replacements specified by replaceConfigs across matching files in outDir.
func ReplaceAll(outDir string, replaceConfigs []config.ReplaceConfig) error {
	for _, r := range replaceConfigs {
		if err := applyToFiles(outDir, r.Path, func(file string) error {
			info, err := os.Stat(file)
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if err := Replace(file, r.Original, r.Replacement); err != nil {
				return fmt.Errorf("failed to apply replacement in %s: %w", file, err)
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

// ReplaceRegexAll applies regex replacements specified by replaceRegexConfigs across matching files in outDir.
func ReplaceRegexAll(outDir string, replaceRegexConfigs []config.ReplaceRegexConfig) error {
	for _, r := range replaceRegexConfigs {
		if err := applyToFiles(outDir, r.Path, func(file string) error {
			info, err := os.Stat(file)
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if err := ReplaceRegex(file, r.Pattern, r.Replacement); err != nil {
				return fmt.Errorf("failed to apply regex replacement in %s: %w", file, err)
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

// ApplyMethodOperations executes method operations across matching files in outDir.
func ApplyMethodOperations(outDir string, methodOperations []config.MethodOperation) error {
	for _, mo := range methodOperations {
		if err := applyToFiles(outDir, mo.Path, func(file string) error {
			switch mo.Action {
			case "delete":
				if err := DeleteMethod(file, mo.FuncName, "java"); err != nil {
					return fmt.Errorf("failed to delete method %q in %s: %w", mo.FuncName, file, err)
				}
			case "duplicate":
				if err := DuplicateMethod(file, mo.FuncName, mo.NewName, "java"); err != nil {
					return fmt.Errorf("failed to duplicate method %q in %s: %w", mo.FuncName, file, err)
				}
			case "deprecate":
				if err := DeprecateMethod(file, mo.FuncName, mo.DeprecationMessage, "java"); err != nil {
					return fmt.Errorf("failed to deprecate method %q in %s: %w", mo.FuncName, file, err)
				}
			default:
				return fmt.Errorf("%w: %q", errUnsupportedMethodAction, mo.Action)
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

// applyToFiles executes action on files matching pathPattern under outDir.
// Note: Uses [filepath.Glob] (* only, ** is not supported).
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
