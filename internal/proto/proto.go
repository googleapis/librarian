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

// Package proto provides helper functions for working with protobuf files.
package proto

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var (
	ErrNotFound = errors.New("not found")
	// nonRecursivePaths is a set of paths where proto gathering should not be recursive.
	nonRecursivePaths = map[string]bool{
		"google/api":   true,
		"google/cloud": true,
	}
)

// Gather returns a sorted list of proto files in the given root directory,
// ensuring that subpackage protos (e.g., in a "schema" directory) are included
// in the generation.
//
// recursion is disabled for certain base paths in nonRecursivePaths.
func Gather(root, relPath string) ([]string, error) {
	var protos []string
	recursive := !nonRecursivePaths[filepath.ToSlash(relPath)]
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if !recursive && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && filepath.Ext(path) == ".proto" {
			protos = append(protos, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(protos)
	return protos, nil
}

// Search looks for a regex in the first .proto file in the API directory.
// Returns the value of the first match found, and a boolean indicating if a match was found.
func Search(googleapisDir, apiPath string, regex *regexp.Regexp) (string, bool, error) {
	file, err := searchForProto(googleapisDir, apiPath)
	if err != nil {
		return "", false, err
	}
	f, err := os.Open(file)
	if err != nil {
		return "", false, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Ignore comments.
		if strings.HasPrefix(line, "//") {
			continue
		}
		if matches := regex.FindStringSubmatch(line); len(matches) > 1 {
			return matches[1], true, nil
		}
	}
	if scanner.Err() != nil {
		return "", false, scanner.Err()
	}
	return "", false, nil
}

// searchForProto finds the first .proto file in the API directory.
func searchForProto(googleapisDir, apiPath string) (string, error) {
	dir := filepath.Join(googleapisDir, apiPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".proto" {
			return filepath.Join(dir, entry.Name()), nil
		}
	}
	return "", fs.ErrNotExist
}
