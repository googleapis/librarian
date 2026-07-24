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

package ruby

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
)

var (
	errNotADirectory = errors.New("output path is not a directory")

	// oneTimeGeneratedRootFiles is the list of files generated only once upon initial library creation.
	oneTimeGeneratedRootFiles = []string{
		"CHANGELOG.md",
	}
	// generatedRootFiles is the list of specific root files generated for Ruby client gems.
	generatedRootFiles = []string{
		"AUTHENTICATION.md",
		"Gemfile",
		"Gemfile.lock",
		"LICENSE",
		"LICENSE.md",
		"README.md",
		"Rakefile",
		"gapic_metadata.json",
		".gitignore",
		".repo-metadata.json",
		".rubocop.yml",
		".toys.rb",
		".yardopts",
	}
	// generatedFileExtensions is the list of file extensions for root generated files.
	generatedFileExtensions = []string{
		".gemspec",
	}
	// generatedDirectories is the list of subdirectories containing code/tests/docs generated for Ruby client gems.
	generatedDirectories = []string{
		"lib",
		"proto_docs",
		"snippets",
		"test",
	}
)

// Clean removes generated files and directories from beneath the given library's
// output directory. If the output directory does not currently exist, this
// function is a no-op.
func Clean(library *config.Library) error {
	dir := library.Output
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("cannot access output directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %q", errNotADirectory, dir)
	}
	keepSet := buildKeepSet(library.Keep)
	if err := cleanGeneratedRootFiles(dir, keepSet); err != nil {
		return err
	}
	return cleanGeneratedDirectories(dir, keepSet)
}

// buildKeepSet builds a set of relative paths to keep from the given keep list.
func buildKeepSet(keep []string) map[string]bool {
	keepSet := make(map[string]bool)
	for _, keepPath := range keep {
		cleaned := filepath.ToSlash(filepath.Clean(keepPath))
		keepSet[cleaned] = true
	}
	for _, file := range oneTimeGeneratedRootFiles {
		keepSet[file] = true
	}
	return keepSet
}

// cleanGeneratedRootFiles removes generated root files from the library directory.
func cleanGeneratedRootFiles(dir string, keepSet map[string]bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		relSlash := filepath.ToSlash(name)
		if isKept(relSlash, keepSet) {
			continue
		}
		if isGeneratedRootFile(name) {
			if err := os.Remove(filepath.Join(dir, name)); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					// The file doesn't exist during deletion, it's fine to ignore this error.
					continue
				}
				return err
			}
		}
	}
	return nil
}

// cleanGeneratedDirectories removes generated subdirectories beneath the library output directory.
func cleanGeneratedDirectories(dir string, keepSet map[string]bool) error {
	for _, subDir := range generatedDirectories {
		subDirPath := filepath.Join(dir, subDir)
		if err := cleanSubdirectory(dir, subDirPath, keepSet); err != nil {
			return err
		}
	}
	return nil
}

func isGeneratedRootFile(name string) bool {
	if slices.Contains(generatedRootFiles, name) {
		return true
	}
	for _, ext := range generatedFileExtensions {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// cleanSubdirectory walks the given subdirectory, removing non-kept files
// and cleaning up empty directories bottom-up using [filesystem.RemoveEmptyDirs].
// If subDirPath does not exist, it returns nil as a no-op.
func cleanSubdirectory(libraryDir, subDirPath string, keepSet map[string]bool) error {
	err := filepath.WalkDir(subDirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(libraryDir, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)
		if isKept(relSlash, keepSet) {
			return nil
		}
		return os.Remove(path)
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	keepFunc := func(rel string) bool {
		return isKept(rel, keepSet)
	}
	return filesystem.RemoveEmptyDirs(subDirPath, libraryDir, keepFunc)
}

// isKept returns true if the specified relative path or any of its parent
// directories is present in keepSet.
func isKept(relSlash string, keepSet map[string]bool) bool {
	currentPath := relSlash
	for currentPath != "." {
		if keepSet[currentPath] {
			return true
		}
		parent := filepath.Dir(currentPath)
		if parent == currentPath {
			break
		}
		currentPath = parent
	}
	return false
}
