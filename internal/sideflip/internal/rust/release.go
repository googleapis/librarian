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

package rust

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sideflip/config"
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

// BumpVersions bumps versions for all Cargo.toml files and updates
// librarian.yaml.
func BumpVersions(ctx context.Context, cfg *config.Config, configPath string) error {
	if cfg.Versions == nil {
		cfg.Versions = make(map[string]string)
	}
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
		newVersion, err := rustrelease.BumpPackageVersion(manifest.Package.Version)
		if err != nil {
			return err
		}

		lines := strings.Split(string(contents), "\n")
		idx := slices.IndexFunc(lines, func(a string) bool {
			return strings.HasPrefix(a, "version ")
		})
		if idx == -1 {
			return fmt.Errorf("no version line found in %s", path)
		}
		lines[idx] = fmt.Sprintf(`version                = "%s"`, newVersion)
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644); err != nil {
			return err
		}
		cfg.Versions[manifest.Package.Name] = newVersion
		return nil
	})

	if err != nil {
		return err
	}
	return cfg.Write(configPath)
}
