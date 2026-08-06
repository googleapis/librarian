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
	"os"
	"path/filepath"
	"slices"
)

var (
	// nonRecursivePaths is a set of paths where proto gathering should not be recursive.
	nonRecursivePaths = map[string]bool{
		"google/api":   true,
		"google/cloud": true,
		"google/rpc":   true,
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
