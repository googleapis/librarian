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

package rust

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sidekick/source"
)

// ResolveDependencies automatically resolves Protobuf dependencies for a Rust library.
func ResolveDependencies(ctx context.Context, cfg *config.Config, lib *config.Library, sources *source.Sources) error {
	if len(lib.APIs) == 0 {
		return nil
	}

	// We only resolve dependencies for the first API in the library.
	// This is consistent with how the Rust generator works.
	modelConfig, err := libraryToModelConfig(lib, lib.APIs[0], sources)
	if err != nil {
		return fmt.Errorf("failed to create model config: %w", err)
	}
	model, err := parser.CreateModel(modelConfig)
	if err != nil {
		return fmt.Errorf("failed to create model: %w", err)
	}

	// Identify the packages owned by the current library.
	ownedPackages := map[string]bool{}
	for _, api := range lib.APIs {
		ownedPackages[toPackageName(api.Path)] = true
	}
	// Also add packages from the model in case they differ.
	for _, s := range model.Services {
		ownedPackages[s.Package] = true
	}
	for _, m := range model.Messages {
		ownedPackages[m.Package] = true
	}
	for _, e := range model.Enums {
		ownedPackages[e.Package] = true
	}

	// Identify all dependencies.
	var targetIDs []string
	for _, s := range model.Services {
		targetIDs = append(targetIDs, s.ID)
	}
	for _, m := range model.Messages {
		targetIDs = append(targetIDs, m.ID)
	}
	for _, e := range model.Enums {
		targetIDs = append(targetIDs, e.ID)
	}

	allDeps, err := api.FindDependencies(model, targetIDs)
	if err != nil {
		return fmt.Errorf("failed to find dependencies: %w", err)
	}

	// Map dependencies back to Protobuf packages.
	externalPackages := map[string]bool{}
	for id := range allDeps {
		var pkg string
		if s, ok := model.State.ServiceByID[id]; ok {
			pkg = s.Package
		} else if m, ok := model.State.MessageByID[id]; ok {
			pkg = m.Package
		} else if e, ok := model.State.EnumByID[id]; ok {
			pkg = e.Package
		}
		if pkg != "" && !ownedPackages[pkg] {
			externalPackages[pkg] = true
		}
	}

	if len(externalPackages) == 0 {
		return nil
	}

	if lib.Rust == nil {
		lib.Rust = &config.RustCrate{}
	}

	// Map Protobuf packages to Rust crates.
	for pkg := range externalPackages {
		// Skip if already present in the library or in the defaults.
		isDependencyPresent := func(deps []*config.RustPackageDependency) bool {
			return slices.ContainsFunc(deps, func(d *config.RustPackageDependency) bool {
				return d.Source == pkg
			})
		}
		if isDependencyPresent(lib.Rust.PackageDependencies) {
			continue
		}
		if cfg.Default != nil && cfg.Default.Rust != nil && isDependencyPresent(cfg.Default.Rust.PackageDependencies) {
			continue
		}

		// Check other libraries in the config.
		for _, other := range cfg.Libraries {
			if other == lib {
				continue
			}
			// This is a heuristic: we check if either the library name or the
			// first API path corresponds to the package.
			// e.g. "google-cloud-secretmanager-v1" -> "google.cloud.secretmanager.v1"
			// e.g. "google/cloud/secretmanager/v1" -> "google.cloud.secretmanager.v1"
			var matchPkg string
			if len(other.APIs) > 0 {
				matchPkg = toPackageName(other.APIs[0].Path)
			} else {
				matchPkg = derivePackageName(other.Name)
			}

			if matchPkg == pkg {
				lib.Rust.PackageDependencies = append(lib.Rust.PackageDependencies, &config.RustPackageDependency{
					Name:    other.Name,
					Package: other.Name,
					Source:  pkg,
				})
				break
			}
		}
	}
	return nil
}

// derivePackageName derives a Protobuf package name from a library name.
// For example: google-cloud-secretmanager-v1 -> google.cloud.secretmanager.v1.
func derivePackageName(name string) string {
	return toPackageName(DeriveAPIPath(name))
}

// toPackageName converts an API path to a Protobuf package name.
// For example: google/cloud/secretmanager/v1 -> google.cloud.secretmanager.v1.
func toPackageName(path string) string {
	return strings.ReplaceAll(path, "/", ".")
}
