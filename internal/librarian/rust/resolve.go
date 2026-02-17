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
func ResolveDependencies(ctx context.Context, cfg *config.Config, lib *config.Library, sources *source.Sources) (*config.Config, error) {
	if len(lib.APIs) == 0 {
		return cfg, nil
	}
	externalPackages, err := findExternalPackages(lib, sources)
	if err != nil {
		return nil, err
	}
	return resolveExternalPackages(cfg, lib, externalPackages), nil
}

func findExternalPackages(lib *config.Library, sources *source.Sources) (map[string]bool, error) {
	// Only resolve dependencies for the first API in the library.
	// This is consistent with how the Rust generator works.
	modelConfig, err := libraryToModelConfig(lib, lib.APIs[0], sources)
	if err != nil {
		return nil, fmt.Errorf("failed to create model config: %w", err)
	}
	model, err := parser.CreateModel(modelConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}
	// Identify the packages owned by the current library.
	ownedPackages := map[string]bool{}
	for _, api := range lib.APIs {
		ownedPackages[toPackageName(api.Path)] = true
	}
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
		return nil, fmt.Errorf("failed to find dependencies: %w", err)
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

	return externalPackages, nil
}

func resolveExternalPackages(cfg *config.Config, lib *config.Library, externalPackages map[string]bool) *config.Config {
	if len(externalPackages) == 0 {
		return cfg
	}
	if lib.Rust == nil {
		lib.Rust = &config.RustCrate{}
	}
	// Map Protobuf packages to Rust crates.
	for pkg := range externalPackages {
		if isDependencyPresent(pkg, lib, cfg) {
			continue
		}
		// Check other libraries in the config.
		for _, other := range cfg.Libraries {
			if other == lib {
				continue
			}
			// Check if either the library name or the
			// first API path corresponds to the package.
			var apiPathMatches bool
			if len(other.APIs) > 0 {
				apiPathMatches = toPackageName(other.APIs[0].Path) == pkg
			}
			libNameMatches := derivePackageName(other.Name) == pkg
			if apiPathMatches || libNameMatches {
				lib.Rust.PackageDependencies = append(lib.Rust.PackageDependencies, &config.RustPackageDependency{
					Name:    other.Name,
					Package: other.Name,
					Source:  pkg,
				})
				break
			}
		}
	}
	return cfg
}

func isDependencyPresent(pkg string, lib *config.Library, cfg *config.Config) bool {
	check := func(deps []*config.RustPackageDependency) bool {
		return slices.ContainsFunc(deps, func(d *config.RustPackageDependency) bool {
			return d.Source == pkg
		})
	}
	if lib.Rust != nil && check(lib.Rust.PackageDependencies) {
		return true
	}
	if cfg.Default != nil && cfg.Default.Rust != nil && check(cfg.Default.Rust.PackageDependencies) {
		return true
	}
	return false
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
