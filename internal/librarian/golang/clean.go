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

package golang

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/googleapis/librarian/internal/config"
)

var (
	rootFiles = []string{"README.md", "internal/version.go"}
	// TODO(https://github.com/googleapis/librarian/issues/4217), document each file about
	// what are matched and why it is necessary.
	// Separate generated files to filename and filename suffix allow us to match
	// the files as accurate as possible.
	generatedClientFiles = []string{
		".repo-metadata.json",
		"auxiliary.go",
		"auxiliary_go123.go",
		"doc.go",
		"gapic_metadata.json",
		"helpers.go",
		"operations.go",
	}
	generatedClientFileSuffixes = []string{
		".pb.go",
		"_client.go",
		"_client_example_go123_test.go",
		"_client_example_test.go",
	}
)

// Clean cleans up a Go library and its associated snippets.
func Clean(library *config.Library) error {
	libraryDir := library.Output
	keepSet, err := check(libraryDir, library.Keep)
	if err != nil {
		return err
	}

	if err := cleanRootFiles(libraryDir, keepSet); err != nil {
		return err
	}
	if err := cleanClientDirectory(library, libraryDir, keepSet); err != nil {
		return err
	}
	return nil
}

// check validates the given directory and returns a set of files to keep.
// It ensures that the provided directory exists and is a directory.
// It also verifies that all files specified in 'keep' exist within 'dir'.
func check(dir string, keep []string) (map[string]bool, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot access output directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", dir)
	}
	keepSet := make(map[string]bool)
	for _, k := range keep {
		path := filepath.Join(dir, k)
		if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("error keeping %s: %w", k, err)
		}
		// Effectively get a canonical relative path. While in most cases
		// this will be equal to k, it might not be - in particular,
		// on Windows the directory separator in paths returned by Rel
		// will be a backslash.
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil, err
		}
		keepSet[rel] = true
	}
	return keepSet, nil
}

// cleanRootFiles removes predefined root files from the library directory unless
// they are explicitly marked to be kept.
func cleanRootFiles(libraryDir string, keepSet map[string]bool) error {
	for _, rootFile := range rootFiles {
		// Handwritten/veneer libraries may have handwritten root files, README.md for example,
		// defined in the keep list.
		// Skip cleaning these files.
		if keepSet[rootFile] {
			continue
		}
		rootFilePath := filepath.Join(libraryDir, rootFile)
		if err := os.Remove(rootFilePath); err != nil {
			if errors.Is(err, syscall.ENOENT) {
				// The file doesn't exist during deletion, it's fine to ignore this error.
				continue
			}
			return err
		}
	}
	return nil
}

// cleanClientDirectory walks through each API directory in the library and
// removes generated Go client files and snippets.
func cleanClientDirectory(library *config.Library, libraryDir string, keepSet map[string]bool) error {
	for _, api := range library.APIs {
		goAPI := findGoAPI(library, api.Path)
		if goAPI == nil {
			return fmt.Errorf("error finding goAPI associated with API %s: %w", api.Path, errGoAPINotFound)
		}
		repoRoot := repoRootPath(libraryDir, library.Name)
		relClientPath := clientPathFromRepoRoot(library, goAPI)
		clientPath := filepath.Join(repoRoot, relClientPath)
		if err := cleanGeneratedClientFiles(clientPath, libraryDir, keepSet); err != nil {
			return err
		}
		snippetDir := snippetDirectory(repoRoot, relClientPath)
		if err := os.RemoveAll(snippetDir); err != nil {
			return err
		}
	}
	return nil
}

func cleanGeneratedClientFiles(clientPath, libraryDir string, keepSet map[string]bool) error {
	// clientPath doesn't exist, which means this is a new library, skip cleaning.
	if _, err := os.Stat(clientPath); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return filepath.WalkDir(clientPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(libraryDir, path)
		if err != nil {
			return err
		}
		// Some libraries may have a non-generated file that has one of the suffixes in generatedClientFileSuffixes,
		// e.g., iam_policy_client.go.
		// These files will be listed in the keep configuration, so we need to check and potentially skip cleaning.
		if keepSet[relPath] {
			return nil
		}
		for _, file := range generatedClientFiles {
			if d.Name() == file {
				return os.Remove(path)
			}
		}
		for _, file := range generatedClientFileSuffixes {
			if strings.HasSuffix(filepath.Base(path), file) {
				return os.Remove(path)
			}
		}
		return nil
	})
}
