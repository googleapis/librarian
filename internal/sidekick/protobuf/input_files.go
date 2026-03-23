// Copyright 2025 Google LLC
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

// Package protobuf provides utilities for handling protobuf files.
package protobuf

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/sources"
)

// DetermineInputFiles determines the input files from the source config.
func DetermineInputFiles(source string, cfg *sources.SourceConfig) ([]string, error) {

	source = cfg.ResolveDir(source)

	files := map[string]bool{}
	if err := findFiles(files, source); err != nil {
		return nil, err
	}
	var includeList []string
	if cfg != nil {
		includeList = cfg.IncludeList
	}
	applyIncludeList(files, source, includeList)
	var list []string
	for name, ok := range files {
		if ok {
			list = append(list, name)
		}
	}
	sort.Strings(list)
	return list, nil
}

func findFiles(files map[string]bool, source string) error {
	const maxDepth = 1
	source = filepath.ToSlash(source)
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		path = filepath.ToSlash(path)
		depth := strings.Count(strings.TrimPrefix(path, source), "/")
		if info.IsDir() && depth >= maxDepth {
			return filepath.SkipDir
		}
		if depth > maxDepth {
			return nil
		}
		if filepath.Ext(path) == ".proto" {
			files[path] = true
		}
		return nil
	})
}

func applyIncludeList(files map[string]bool, sourceDirectory string, includeList []string) {
	if len(includeList) == 0 {
		return
	}
	// Ignore any discovered paths, only the paths from the include list apply.
	clear(files)
	for _, p := range includeList {
		files[filepath.ToSlash(path.Join(sourceDirectory, p))] = true
	}
}
