// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package rust provides Rust specific functionality for librarian.
package rust

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	rustrelease "github.com/googleapis/librarian/internal/sidekick/rust_release"
	"github.com/pelletier/go-toml/v2"
)

type cargoPackage struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
}

type cargoManifest struct {
	Package *cargoPackage `toml:"package"`
}

// ReleaseAll bumps versions for all Cargo.toml files and updates librarian.yaml.
func ReleaseAll(cfg *config.Config) (*config.Config, error) {
	return release(cfg, "")
}

// ReleaseLibrary bumps the version for a specific library and updates librarian.yaml.
func ReleaseLibrary(cfg *config.Config, name string) (*config.Config, error) {
	return release(cfg, name)
}

func release(cfg *config.Config, name string) (*config.Config, error) {
	shouldRelease := func(pkgName string) bool {
		// If name is the empty string, release everything.
		if name == "" {
			return true
		}
		if name == pkgName {
			return true
		}
		return false
	}

	var found bool
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "Cargo.toml" {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var manifest cargoManifest
		if err := toml.Unmarshal(contents, &manifest); err != nil {
			return err
		}
		if manifest.Package == nil {
			return nil
		}
		if !shouldRelease(manifest.Package.Name) {
			return nil
		}

		found = true
		newVersion, err := rustrelease.BumpPackageVersion(manifest.Package.Version)
		if err != nil {
			return err
		}
		if err := rustrelease.UpdateCargoVersion(path, newVersion); err != nil {
			return err
		}
		library, err := cfg.LibraryByName(name)
		if err != nil {
			return err
		}
		library.Version = newVersion
		return nil
	})
	if err != nil {
		return nil, err
	}
	if name != "" && !found {
		return nil, fmt.Errorf("library %q not found", name)
	}
	return cfg, nil
}
