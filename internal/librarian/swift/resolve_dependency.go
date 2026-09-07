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

package swift

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/iancoleman/strcase"
)

// ResolveDependencyVersions resolves missing versions for dependencies of a Swift library
// using matching library declarations from the configuration.
func ResolveDependencyVersions(cfg *config.Config, library *config.Library) {
	if cfg == nil || library == nil || library.Swift == nil || len(library.Swift.Dependencies) == 0 {
		return
	}
	for i := range library.Swift.Dependencies {
		dep := &library.Swift.Dependencies[i]
		if dep.Version != "" {
			continue
		}
		target := findLibraryForDependency(cfg.Libraries, dep)
		if target == nil {
			continue
		}
		if target.Version != "" {
			dep.Version = target.Version
		} else if cfg.Default != nil && cfg.Default.Swift != nil && cfg.Default.Swift.DefaultVersion != "" {
			dep.Version = cfg.Default.Swift.DefaultVersion
		}
	}
}

func findLibraryForDependency(libraries []*config.Library, dep *config.SwiftDependency) *config.Library {
	for _, lib := range libraries {
		if lib.Swift != nil && lib.Swift.LibraryNameOverride != "" && strings.EqualFold(lib.Swift.LibraryNameOverride, dep.Name) {
			return lib
		}
	}
	for _, lib := range libraries {
		if lib.Name == dep.Name || strings.EqualFold(lib.Name, dep.Name) {
			return lib
		}
		camel := strcase.ToCamel(lib.Name)
		if camel == dep.Name || strings.EqualFold(camel, dep.Name) {
			return lib
		}
	}
	if dep.URL != "" {
		repo := repoNameFromURL(dep.URL)
		trimmedRepo := strings.TrimPrefix(repo, "swift-")
		for _, lib := range libraries {
			if lib.Name == repo || lib.Name == trimmedRepo {
				return lib
			}
			if lib.Swift != nil && (strings.EqualFold(lib.Swift.LibraryNameOverride, repo) || strings.EqualFold(lib.Swift.LibraryNameOverride, trimmedRepo)) {
				return lib
			}
			camelTrimmed := strcase.ToCamel(trimmedRepo)
			if camelTrimmed == dep.Name || strings.EqualFold(camelTrimmed, dep.Name) {
				return lib
			}
			if lib.Output != "" && (outputPathContains(lib.Output, repo) || outputPathContains(lib.Output, trimmedRepo)) {
				return lib
			}
		}
	}
	if dep.Path != "" {
		pathBase := filepath.Base(dep.Path)
		trimmedPathBase := strings.TrimPrefix(pathBase, "swift-")
		for _, lib := range libraries {
			if lib.Name == pathBase || lib.Name == trimmedPathBase {
				return lib
			}
			if lib.Output != "" && (filepath.Clean(lib.Output) == filepath.Clean(dep.Path) || strings.HasPrefix(lib.Output, dep.Path) || strings.HasPrefix(dep.Path, lib.Output)) {
				return lib
			}
		}
	}
	if dep.ApiPackage != "" {
		for _, lib := range libraries {
			for _, api := range lib.APIs {
				if strings.ReplaceAll(api.Path, "/", ".") == dep.ApiPackage {
					return lib
				}
			}
			if strings.ReplaceAll(lib.Name, "-", ".") == dep.ApiPackage {
				return lib
			}
		}
	}
	return nil
}

func repoNameFromURL(rawURL string) string {
	source := strings.TrimSuffix(rawURL, ".git")
	source = strings.Trim(source, "/")
	idx := strings.LastIndex(source, "/")
	if idx == -1 {
		return source
	}
	return source[idx+1:]
}

func outputPathContains(output, target string) bool {
	clean := filepath.Clean(output)
	return slices.Contains(strings.Split(clean, string(filepath.Separator)), target)
}
