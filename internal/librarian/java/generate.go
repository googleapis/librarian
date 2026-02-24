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

// Package java provides Java specific functionality for librarian.
package java

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/googleapis/librarian/internal/config"
)

var itTestRegexp = regexp.MustCompile(`src/test/java/com/google/cloud/.*/v.*/it/IT.*Test\.java$`)

// GenerateLibraries generates all the given libraries in sequence.
func GenerateLibraries(ctx context.Context, libraries []*config.Library, googleapisDir string) error {
	for _, library := range libraries {
		if err := generate(ctx, library, googleapisDir); err != nil {
			return err
		}
	}
	return nil
}

// generate generates a Java client library.
func generate(ctx context.Context, library *config.Library, googleapisDir string) error {
	if len(library.APIs) == 0 {
		return fmt.Errorf("no apis configured for library %q", library.Name)
	}
	fmt.Printf("to be implemented with: %v, %v, %v", ctx, library.Name, googleapisDir)
	return nil
}

// Format formats a Java client library using google-java-format.
func Format(ctx context.Context, library *config.Library) error {
	files, err := collectJavaFiles(library.Output)
	if err != nil {
		return fmt.Errorf("failed to find java files for formatting: %w", err)
	}
	if len(files) == 0 {
		return nil
	}

	if _, err := exec.LookPath("google-java-format"); err != nil {
		return fmt.Errorf("google-java-format not found in PATH: %w", err)
	}

	args := append([]string{"--replace"}, files...)
	cmd := exec.CommandContext(ctx, "google-java-format", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("formatting failed: %w", err)
	}
	return nil
}

func collectJavaFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".java" {
			return nil
		}
		// exclude samples/snippets/generated
		if strings.Contains(path, filepath.Join("samples", "snippets", "generated")) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

// Clean removes files in the library's output directory that are not in the keep list.
// It targets patterns like proto-*, grpc-*, and the main GAPIC module.
func Clean(library *config.Library) error {
	libraryName := library.Name
	if !strings.HasPrefix(libraryName, "google-cloud-") {
		libraryName = "google-cloud-" + libraryName
	}

	patterns := []string{
		fmt.Sprintf("proto-%s-*", libraryName),
		fmt.Sprintf("grpc-%s-*", libraryName),
		libraryName,
		filepath.Join("samples", "snippets", "generated"),
	}

	keepSet := make(map[string]bool)
	for _, k := range library.Keep {
		keepSet[k] = true
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(library.Output, pattern))
		if err != nil {
			return err
		}
		for _, match := range matches {
			if err := cleanPath(match, library.Output, keepSet); err != nil {
				return err
			}
		}
	}
	return nil
}

func cleanPath(targetPath, root string, keepSet map[string]bool) error {
	var dirs []string
	err := filepath.WalkDir(targetPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			rel, _ := filepath.Rel(root, path)
			if keepSet[rel] {
				return filepath.SkipDir
			}
			dirs = append(dirs, path)
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if keepSet[rel] || itTestRegexp.MatchString(filepath.ToSlash(rel)) {
			return nil
		}
		// Bypass clirr-ignored-differences.xml files as they are generated once and manually maintained.
		if d.Name() == "clirr-ignored-differences.xml" {
			return nil
		}
		return os.Remove(path)
	})
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Remove empty directories in reverse order (bottom-up).
	for i := len(dirs) - 1; i >= 0; i-- {
		d := dirs[i]
		rel, err := filepath.Rel(root, d)
		if err != nil {
			return err
		}
		if !keepSet[rel] {
			if err := os.Remove(d); err != nil && !os.IsNotExist(err) && !isDirNotEmpty(err) {
				return err
			}
		}
	}
	return nil
}

// isDirNotEmpty returns true if err indicates the directory is not empty.
func isDirNotEmpty(err error) bool {
	return errors.Is(err, syscall.ENOTEMPTY) || errors.Is(err, syscall.EEXIST)
}
