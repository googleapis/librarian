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

package dart

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

func libraryOutput(lib *config.Library, defaults *config.Default) string {
	if lib.Output != "" {
		return lib.Output
	}
	var defaultOut string
	if defaults != nil {
		defaultOut = defaults.Output
	}
	return filepath.Join(defaultOut, lib.Name)
}

// getDeps returns the dependencies and versions for the given libraries as found in the GitHub repository.
//
// For example:
//
//		getDeps(..., [google_cloud_protobuf, google_cloud_logging_type, google_cloud_logging]) =>
//		({"google_cloud_protobuf": [], "google_cloud_logging_type": ["google_cloud_protobuf"],
//		 "google_cloud_logging": ["google_cloud_logging_type"]}
//
//		 {"google_cloud_protobuf": "1.2.3", "google_cloud_logging_type": "4.5.6", "google_cloud_logging": "5.6.7"},
//
//	  nil)
func getDeps(ctx context.Context, libraries []*config.Library) (map[string][]string, map[string]string, error) {
	output, err := command.Output(ctx, command.Dart, "pub", "deps", "--json")
	if err != nil {
		return nil, nil, err
	}
	var data struct {
		Packages []struct {
			Name         string   `json:"name"`
			Version      string   `json:"version"`
			Dependencies []string `json:"dependencies"`
		} `json:"packages"`
	}
	if err := json.Unmarshal([]byte(output), &data); err != nil {
		return nil, nil, fmt.Errorf("failed to parse dart pub deps output: %w", err)
	}

	libNames := make(map[string]bool)
	for _, lib := range libraries {
		libNames[lib.Name] = true
	}

	depsMap := make(map[string][]string)
	versionsMap := make(map[string]string)
	for _, pkg := range data.Packages {
		if !libNames[pkg.Name] {
			continue
		}
		versionsMap[pkg.Name] = pkg.Version
		var deps []string
		for _, dep := range pkg.Dependencies {
			if libNames[dep] {
				deps = append(deps, dep)
			}
		}
		slices.Sort(deps)
		depsMap[pkg.Name] = deps
	}

	return depsMap, versionsMap, nil
}

// sortByDeps performs a topological sort of the libraries by their dependencies such that dependencies
// always appear before the libraries that depend on them.
func sortByDeps(libraryByName map[string]*config.Library, deps map[string][]string) ([]string, error) {
	inDegree := make(map[string]int)
	dependents := make(map[string][]string)

	for name := range libraryByName {
		pkgDeps := deps[name]
		inDegree[name] = len(pkgDeps)
		for _, dep := range pkgDeps {
			dependents[dep] = append(dependents[dep], name)
		}
	}

	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	slices.Sort(queue)

	var sorted []string
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		sorted = append(sorted, curr)

		for _, dep := range dependents[curr] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
		slices.Sort(queue)
	}

	if len(sorted) < len(libraryByName) {
		return nil, fmt.Errorf("cycle detected in dependency DAG")
	}

	return sorted, nil
}
