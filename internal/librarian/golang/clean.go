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
	"regexp"

	"github.com/googleapis/librarian/internal/config"
)

var (
	generatedRegex = []*regexp.Regexp{
		regexp.MustCompile(`auxiliary\.go$`),
		regexp.MustCompile(`auxiliary_go123\.go$`),
		regexp.MustCompile(`doc\.go$`),
		regexp.MustCompile(`.*_client\.go$`),
		regexp.MustCompile(`.*_client_example_go123_test\.go$`),
		regexp.MustCompile(`.*_client_example_test\.go$`),
		regexp.MustCompile(`gapic_metadata\.json$`),
		regexp.MustCompile(`helpers\.go$`),
		regexp.MustCompile(`.*\.pb\.go$`),
		regexp.MustCompile(`.*/internal/generated/snippets/.*$`),
	}
)

// Clean cleans up a Go library and its associated snippets.
func Clean(library *config.Library) (*config.Library, error) {
	libraryDir := filepath.Join(library.Output, library.Name)
	keepSet, err := check(libraryDir, library.Keep)
	if err != nil {
		return nil, err
	}
	var nestedModule string
	if library.Go != nil {
		nestedModule = library.Go.NestedModule
	}
	if err := clean(libraryDir, nestedModule, keepSet); err != nil {
		return nil, err
	}
	snippetDir := filepath.Join(library.Output, "internal", "generated", "snippets", library.Name)
	if err := clean(snippetDir, nestedModule, nil); err != nil {
		return nil, err
	}
	return library, nil
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

// clean recursively removes files in dir that are not in keepSet.
// If nestedModule is non-empty, any directory with that name is skipped.
func clean(dir, nestedModule string, keepSet map[string]bool) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == nestedModule {
				return fs.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		isGenerated := false
		for _, re := range generatedRegex {
			if re.MatchString(path) {
				isGenerated = true
				break
			}
		}
		if keepSet[rel] || !isGenerated {
			return nil
		}
		return os.Remove(path)
	})
}
