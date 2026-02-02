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

package librarian

import (
	"fmt"
	"maps"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

// fillDefaults populates empty library fields from the provided defaults.
func fillDefaults(lib *config.Library, d *config.Default) *config.Library {
	if d == nil {
		return lib
	}
	if lib.Output == "" {
		lib.Output = d.Output
	}
	if lib.ReleaseLevel == "" {
		lib.ReleaseLevel = d.ReleaseLevel
	}
	if lib.Transport == "" {
		lib.Transport = d.Transport
	}
	if d.Rust != nil {
		return fillRust(lib, d)
	}
	if d.Dart != nil {
		return fillDart(lib, d)
	}
	return lib
}

// fillRust populates empty Rust-specific fields in lib from the provided default.
func fillRust(lib *config.Library, d *config.Default) *config.Library {
	if lib.Rust == nil {
		lib.Rust = &config.RustCrate{}
	}
	lib.Rust.PackageDependencies = mergePackageDependencies(
		d.Rust.PackageDependencies,
		lib.Rust.PackageDependencies,
	)
	if len(lib.Rust.DisabledRustdocWarnings) == 0 {
		lib.Rust.DisabledRustdocWarnings = d.Rust.DisabledRustdocWarnings
	}
	if lib.Rust.GenerateSetterSamples == "" {
		lib.Rust.GenerateSetterSamples = d.Rust.GenerateSetterSamples
	}
	if lib.Rust.GenerateRpcSamples == "" {
		lib.Rust.GenerateRpcSamples = d.Rust.GenerateRpcSamples
	}
	for _, mod := range lib.Rust.Modules {
		if mod.GenerateSetterSamples == "" {
			mod.GenerateSetterSamples = lib.Rust.GenerateSetterSamples
		}
		if mod.GenerateRpcSamples == "" {
			mod.GenerateRpcSamples = lib.Rust.GenerateRpcSamples
		}
	}
	return lib
}

func fillDart(lib *config.Library, d *config.Default) *config.Library {
	if lib.Version == "" {
		lib.Version = d.Dart.Version
	}
	if lib.Dart == nil {
		lib.Dart = &config.DartPackage{}
	}
	if lib.Dart.APIKeysEnvironmentVariables == "" {
		lib.Dart.APIKeysEnvironmentVariables = d.Dart.APIKeysEnvironmentVariables
	}
	if lib.Dart.IssueTrackerURL == "" {
		lib.Dart.IssueTrackerURL = d.Dart.IssueTrackerURL
	}
	lib.Dart.Packages = mergeMaps(lib.Dart.Packages, d.Dart.Packages)
	lib.Dart.Prefixes = mergeMaps(lib.Dart.Prefixes, d.Dart.Prefixes)
	lib.Dart.Protos = mergeMaps(lib.Dart.Protos, d.Dart.Protos)
	lib.Dart.Dependencies = mergeDartDependencies(lib.Dart.Dependencies, d.Dart.Dependencies)
	return lib
}

// mergeDartDependencies merges library dependencies with default dependencies.
// Duplicate dependencies in defaults will be ignored.
func mergeDartDependencies(libDeps, defaultDeps string) string {
	seen := make(map[string]bool)
	var deps []string
	for _, dep := range strings.Split(libDeps, ",") {
		dep = strings.TrimSpace(dep)
		if dep == "" {
			continue
		}
		seen[dep] = true
		deps = append(deps, dep)
	}
	for _, dep := range strings.Split(defaultDeps, ",") {
		dep = strings.TrimSpace(dep)
		if dep == "" || seen[dep] {
			continue
		}
		deps = append(deps, dep)
	}
	return strings.Join(deps, ",")
}

// mergePackageDependencies merges default and library package dependencies,
// with library dependencies taking precedence for duplicates.
func mergePackageDependencies(defaults, lib []*config.RustPackageDependency) []*config.RustPackageDependency {
	seen := make(map[string]bool)
	var result []*config.RustPackageDependency
	for _, dep := range lib {
		seen[dep.Name] = true
		result = append(result, dep)
	}
	for _, dep := range defaults {
		if seen[dep.Name] {
			continue
		}
		copied := *dep
		result = append(result, &copied)
	}
	return result
}

// libraryOutput returns the output path for a library. If the library has an
// explicit output path, it returns that. Otherwise, it computes the default
// output path based on the api path and default configuration.
func libraryOutput(language string, lib *config.Library, defaults *config.Default) string {
	if lib.Output != "" {
		return lib.Output
	}
	if lib.Veneer {
		// Veneers require explicit output, so return empty if not set.
		return ""
	}
	apiPath := deriveAPIPath(language, lib.Name)
	if len(lib.APIs) > 0 && lib.APIs[0].Path != "" {
		apiPath = lib.APIs[0].Path
	}
	defaultOut := ""
	if defaults != nil {
		defaultOut = defaults.Output
	}
	return defaultOutput(language, lib.Name, apiPath, defaultOut)
}

// applyDefaults applies language-specific derivations and fills defaults.
func applyDefaults(language string, lib *config.Library, defaults *config.Default) (*config.Library, error) {
	if len(lib.APIs) == 0 {
		lib.APIs = append(lib.APIs, &config.API{})
	}
	if !lib.Veneer {
		for _, api := range lib.APIs {
			if api.Path == "" {
				api.Path = deriveAPIPath(language, lib.Name)
			}
		}
	}
	if lib.Output == "" {
		if lib.Veneer {
			return nil, fmt.Errorf("veneer %q requires an explicit output path", lib.Name)
		}
		lib.Output = defaultOutput(language, lib.Name, lib.APIs[0].Path, defaults.Output)
	}
	return fillDefaults(lib, defaults), nil
}

// mergeMaps merges key-values of src and dst maps.
// When a key in src is already present in dst, the value in dst will NOT be overwritten
// by the value associated with the key in src.
func mergeMaps(dst, src map[string]string) map[string]string {
	if dst == nil {
		dst = make(map[string]string)
		maps.Copy(dst, src)
		return dst
	}

	res := make(map[string]string)
	for key, value := range src {
		res[key] = value
	}
	for key, value := range dst {
		res[key] = value
	}
	return res
}
