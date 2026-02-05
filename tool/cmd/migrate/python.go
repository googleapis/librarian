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

package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
)

// buildPythonLibraries builds a set of librarian libraries from legacylibrarian
// libraries and the googleapis directory used to find settings in service
// config files, BUILD.bazel files etc.
func buildPythonLibraries(input *MigrationInput) ([]*config.Library, error) {
	var libraries []*config.Library
	// No need to use legacyconfig.LibraryConfig - the only thing in
	// the python config is a single global file entry.

	for _, libState := range input.librarianState.Libraries {
		library := &config.Library{
			Name:    libState.ID,
			Version: libState.Version,
		}
		if libState.APIs != nil {
			library.APIs = toAPIs(libState.APIs)
		}
		// Convert "preserve" regexes into "keep" paths, sorted for ease
		// of testing.
		keep, err := transformPreserveToKeep(input.repoPath, libState)
		if err != nil {
			return nil, err
		}
		slices.Sort(keep)
		library.Keep = keep

		libraries = append(libraries, library)
	}
	return libraries, nil
}

// transformPreserveToKeep converts the "preserve" entries in a legacylibrarian
// state file into "keep" entries in a new library config. Differences:
//   - Preserve entries are repo-root-relative; keep entries are
//     library-output-relative.
//   - Preserve entries are regular expressions; keep entries are just filenames.
//   - Preserve entries don't have to match anything; keep entries must exist.
//
// transformPreserveToKeep finds all files which would *currently* be kept, and
// creates a keep entry for each of them.
func transformPreserveToKeep(rootDir string, libState *legacyconfig.LibraryState) ([]string, error) {
	if len(libState.PreserveRegex) == 0 {
		return nil, nil
	}
	if len(libState.SourceRoots) != 1 {
		return nil, fmt.Errorf("cannot migrate %s with %d source roots and %d preserve regexes", libState.ID, len(libState.SourceRoots), len(libState.PreserveRegex))
	}
	sourceRoot := libState.SourceRoots[0]

	// relPaths contains a list of files in the source root, but relative to
	// rootDir.
	relPaths, err := findSubDirRelPaths(rootDir, filepath.Join(rootDir, sourceRoot))
	if err != nil {
		return nil, err
	}

	// Apply all the preserve regular expressions to all the files we've found,
	// adding each match to the keep list.
	preserveRegexps, err := compileRegexps(libState.PreserveRegex)
	if err != nil {
		return nil, err
	}
	var keepPaths []string
	for _, path := range filterPathsByRegex(relPaths, preserveRegexps) {
		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			// This should not happen given the logic in findSubDirRelPaths
			return nil, fmt.Errorf("could not make path %q relative to %q: %w", path, sourceRoot, err)
		}
		keepPaths = append(keepPaths, relative)
	}
	return keepPaths, nil
}

// compileRegexps takes a slice of string patterns and compiles each one into a
// regular expression. It returns a slice of compiled regexps or an error if any
// pattern is invalid.
// This function is copied from legacylibrarian. Neither this code nor
// legacylibrarian is expected to be long-lived, so it's not worth refactoring
// for reuse without duplication.
func compileRegexps(patterns []string) ([]*regexp.Regexp, error) {
	var regexps []*regexp.Regexp
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %q: %w", pattern, err)
		}
		regexps = append(regexps, re)
	}
	return regexps, nil
}

// findSubDirRelPaths walks the subDir tree returns a slice of all file and
// directory paths relative to the dir. This is repeated for all nested
// directories. subDir must be under or the same as dir.
// This function is copied from legacylibrarian. Neither this code nor
// legacylibrarian is expected to be long-lived, so it's not worth refactoring
// for reuse without duplication.
func findSubDirRelPaths(dir, subDir string) ([]string, error) {
	dirRelPath, err := filepath.Rel(dir, subDir)
	if err != nil {
		return nil, fmt.Errorf("cannot establish the relationship between %s and %s: %w", dir, subDir, err)
	}
	// '..' signifies that the subDir exists outside of dir
	if strings.HasPrefix(dirRelPath, "..") {
		return nil, fmt.Errorf("subDir is not nested within the dir: %s, %s", subDir, dir)
	}

	var paths []string
	err = filepath.WalkDir(subDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// error is ignored as we have confirmed that subDir is child or equal to rootDir
		relPath, _ := filepath.Rel(dir, path)
		// Special case when subDir is equal to dir. Drop the "." as it references itself
		if relPath != "." {
			paths = append(paths, relPath)
		}
		return nil
	})
	return paths, err
}

// filterPathsByRegex returns a new slice containing only the paths from the
// input slice that match at least one of the provided regular expressions.
// This function is copied from legacylibrarian. Neither this code nor
// legacylibrarian is expected to be long-lived, so it's not worth refactoring
// for reuse without duplication.
func filterPathsByRegex(paths []string, regexps []*regexp.Regexp) []string {
	var filtered []string
	for _, path := range paths {
		for _, re := range regexps {
			if re.MatchString(path) {
				filtered = append(filtered, path)
				break
			}
		}
	}
	return filtered
}
