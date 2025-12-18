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
	"cloud.google.com/go/librarian/internal/config"
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
	return lib
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
