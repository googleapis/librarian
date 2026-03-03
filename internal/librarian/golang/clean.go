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

	"github.com/googleapis/librarian/internal/config"
)

var (
	rootFiles = []string{"README.md", "internal/version.go"}
	// TODO(https://github.com/googleapis/librarian/issues/4217), document each file about
	// what are matched and why it is necessary.
	generatedClientFiles = []string{
		".repo-metadata.json",
		".pb.go",
		"auxiliary.go",
		"auxiliary_go123.go",
		"_client.go",
		"_client_example_go123_test.go",
		"_client_example_test.go",
		"doc.go",
		"gapic_metadata.json",
		"helpers.go",
		"operations.go",
	}
)

// Clean cleans up a Go library and its associated snippets.
func Clean(library *config.Library) error {
	libraryDir := filepath.Join(library.Output, library.Name)
	keepSet, err := check(libraryDir, library.Keep)
	if err != nil {
		return err
	}

	if err := cleanRootFiles(libraryDir, keepSet); err != nil {
		return err
	}
	if err := cleanClientDirectory(library); err != nil {
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
		rootFile := filepath.Join(libraryDir, rootFile)
		// Some library may not have the root file, README.md for example, this is rare,
		// but we should not fail the clean in this case.
		if _, err := os.Stat(rootFile); os.IsNotExist(err) {
			continue
		}
		if err := os.Remove(rootFile); err != nil {
			return err
		}
	}
	return nil
}

// cleanClientDirectory walks through each API directory in the library and
// removes generated Go client files.
func cleanClientDirectory(library *config.Library) error {
	for _, api := range library.APIs {
		goAPI := findGoAPI(library, api.Path)
		if goAPI == nil {
			return fmt.Errorf("could not find Go API associated with %s: %w", api.Path, errGoAPINotFound)
		}
		clientPath := filepath.Join(library.Output, goAPI.ImportPath)
		// clientPath doesn't exist, which means this is a new library, skip cleaning.
		if _, err := os.Stat(clientPath); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err := filepath.WalkDir(clientPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			for _, file := range generatedClientFiles {
				if !strings.HasSuffix(filepath.Base(path), file) {
					continue
				}
				return os.Remove(path)
			}
			return nil
		}); err != nil {
			return err
		}

		snippetDir := snippetDirectory(library.Output, goAPI.ImportPath)
		if err := os.RemoveAll(snippetDir); err != nil {
			return err
		}
	}
	return nil
}
