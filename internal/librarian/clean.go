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

package librarian

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
)

// checkAndClean removes all files in dir except those in keep. The keep list
// should contain paths relative to dir. It returns an error if any file
// in keep does not exist.
func checkAndClean(dir string, keep []string) error {
	keepSet, err := checkKeep(dir, keep)
	if err != nil {
		return err
	}
	return clean(dir, keepSet)
}

// checkKeep validates the given directory and returns a set of files to keep.
// It ensures that the provided directory exists and is a directory.
// It also verifies that all files specified in 'keep' exist within 'dir'.
func checkKeep(dir string, keep []string) (map[string]bool, error) {
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
			return nil, fmt.Errorf("keep file %q does not exist", k)
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

// clean removes files from dir that are not in keepSet.
func clean(dir string, keepSet map[string]bool) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if keepSet[rel] {
			return nil
		}
		return os.Remove(path)
	})
}

// TODO(https://github.com/googleapis/librarian/issues/4001): move this function
// to internal/librarian/golang when the logic is deviate from checkAndClean.
func cleanGo(library *config.Library) (*config.Library, error) {
	libraryDir := filepath.Join(library.Output, library.Name)
	if err := checkAndClean(libraryDir, library.Keep); err != nil {
		return nil, err
	}
	snippetDir := filepath.Join(library.Output, "internal", "generated", "snippets", library.Name)
	if err := checkAndClean(snippetDir, nil); err != nil {
		return nil, err
	}
	return library, nil
}
