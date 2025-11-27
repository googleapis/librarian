// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package python

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// cleanOutputDirectory deletes everything in the output directory except files listed in keepPaths.
func cleanOutputDirectory(outdir string, keepPaths []string) error {
	// Check if directory exists
	if _, err := os.Stat(outdir); os.IsNotExist(err) {
		return nil
	}

	// Build map of paths to keep (normalized to absolute paths)
	keepMap := make(map[string]bool)
	for _, keepPath := range keepPaths {
		absKeepPath := filepath.Join(outdir, keepPath)
		// Check if any file exists with this prefix
		if hasFileWithPrefix(outdir, keepPath) {
			keepMap[absKeepPath] = true
		}
	}

	// Walk directory and delete everything not in keep list
	entries, err := os.ReadDir(outdir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", outdir, err)
	}

	for _, entry := range entries {
		// Never, ever delete the .git directory. Doing so due to
		// misconfiguration can be awful. Ask me how I know...
		if entry.Name() == ".git" {
			continue
		}
		entryPath := filepath.Join(outdir, entry.Name())

		// Check if this path or any parent should be kept
		if shouldKeep(entryPath, keepMap) {
			continue
		}

		// Delete this entry
		if err := os.RemoveAll(entryPath); err != nil {
			return fmt.Errorf("failed to remove %s: %w", entryPath, err)
		}
	}

	return nil
}

// hasFileWithPrefix checks if any file exists in the directory with the given prefix.
func hasFileWithPrefix(dir, prefix string) bool {
	fullPath := filepath.Join(dir, prefix)

	// Check if exact path exists
	if _, err := os.Stat(fullPath); err == nil {
		return true
	}

	// Check if it's a directory prefix - walk the directory to find files
	parentDir := filepath.Dir(fullPath)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		return false
	}

	// Walk the parent directory to find any files matching the prefix
	found := false
	filepath.Walk(parentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		// Check if this path has the prefix
		if strings.HasPrefix(relPath, prefix) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})

	return found
}

// shouldKeep checks if a path should be kept based on the keep map.
// A path is kept if it exactly matches a keep path, or if it's a parent of a keep path.
func shouldKeep(path string, keepMap map[string]bool) bool {
	// Exact match
	if keepMap[path] {
		return true
	}

	// Check if path is a parent of any keep path
	for keepPath := range keepMap {
		if strings.HasPrefix(keepPath, path+string(filepath.Separator)) {
			return true
		}
	}

	return false
}
