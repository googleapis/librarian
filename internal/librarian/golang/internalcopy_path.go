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
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

var errInternalCopySymlink = errors.New("internal copy destination contains a symlink")

// validateInternalCopyPaths checks configuration and filesystem destinations
// before cleanup or generation. Symlinks below the library output are not
// supported for copies, even when they point inside the library.
func validateInternalCopyPaths(library *config.Library, outDir string) error {
	if err := validateInternalCopies(library, outDir); err != nil {
		return err
	}
	if !slices.ContainsFunc(library.APIs, func(api *config.API) bool {
		return api.Go != nil && len(api.Go.InternalCopies) != 0
	}) {
		return nil
	}
	root, err := resolveExistingPath(outDir)
	if err != nil {
		return err
	}
	var dirs []string
	for _, api := range library.APIs {
		if api.Go == nil {
			continue
		}
		if api.Go.ImportPath == "" {
			continue
		}
		dir := filepath.Join(repoRootPath(outDir, library.Name), pathFromRepoRoot(library, api.Go.ImportPath))
		resolved, err := resolveExistingPath(dir)
		if err != nil {
			return err
		}
		dirs = append(dirs, resolved)
	}
	for _, api := range library.APIs {
		if api.Go == nil {
			continue
		}
		for _, cp := range api.Go.InternalCopies {
			dir, err := internalCopyDir(library, outDir, cp)
			if err != nil {
				return err
			}
			if err := checkCopySymlinks(outDir, dir); err != nil {
				return err
			}
			resolved, err := resolveExistingPath(dir)
			if err != nil {
				return err
			}
			if resolved == root || !containsPath(root, resolved) {
				return fmt.Errorf("%w: %q resolves to %s, outside %s", errInternalCopyOutsideLibrary, cp.ImportPath, resolved, root)
			}
			for _, other := range dirs {
				if overlapsPath(other, resolved) {
					return fmt.Errorf("%w: internal copy %q resolves to %s, overlapping %s", errInternalCopyOverlap, cp.ImportPath, resolved, other)
				}
			}
			dirs = append(dirs, resolved)
		}
	}
	return nil
}

// resolveExistingPath resolves symlinks in the deepest existing ancestor of
// dir, preserving the suffix that cleanup or generation has not created yet.
func resolveExistingPath(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	var suffix string
	for {
		if _, err := os.Lstat(dir); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return "", err
			}
			suffix = filepath.Join(filepath.Base(dir), suffix)
			dir = filepath.Dir(dir)
			continue
		}
		resolved, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return "", err
		}
		return filepath.Join(resolved, suffix), nil
	}
}

// checkCopySymlinks rejects existing symlinks between the library output and
// the copy. Symlinks in the output root itself or its ancestors are resolved
// separately, allowing repositories reached through a symlink.
func checkCopySymlinks(outDir, dir string) error {
	rel, err := filepath.Rel(outDir, dir)
	if err != nil {
		return err
	}
	for part := range strings.SplitSeq(rel, string(filepath.Separator)) {
		outDir = filepath.Join(outDir, part)
		info, err := os.Lstat(outDir)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: %s", errInternalCopySymlink, outDir)
		}
	}
	return nil
}
