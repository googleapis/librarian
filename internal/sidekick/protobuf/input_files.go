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
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/config"
)

// DetermineInputFiles determines the input files from the source config.
func DetermineInputFiles(source string, cfg config.SourceConfig) ([]string, error) {
	if len(cfg.IncludeList) > 0 && len(cfg.ExcludeList) > 0 {
		return nil, fmt.Errorf("cannot use both `exclude-list` and `include-list` in the source options")
	}

	// Iterate over active roots to find the source directory.
	for _, root := range cfg.ActiveRoots {
		rootPath, err := cfg.Root(root)
		if err != nil {
			return nil, err
		}
		if rootPath == "" {
			continue
		}
		candidate := path.Join(rootPath, source)
		stat, err := os.Stat(candidate)
		if err == nil && stat.IsDir() {
			source = candidate
			break
		}
	}

	files := map[string]bool{}
	if err := findFiles(files, source); err != nil {
		return nil, err
	}
	applyIncludeList(files, source, cfg.IncludeList)
	applyExcludeList(files, source, cfg.ExcludeList)
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

func applyExcludeList(files map[string]bool, sourceDirectory string, excludeList []string) {
	for _, p := range excludeList {
		delete(files, filepath.ToSlash(path.Join(sourceDirectory, p)))
	}
}
