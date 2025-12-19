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

// Package rust provides Rust specific functionality for librarian.
package rust

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/semver"
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

func ReleaseLibrary(cfg *config.Config, library *config.Library) (*config.Config, error) {
	srcPath := deriveSrcPath(library, cfg)
	cargoFile := filepath.Join(srcPath, "Cargo.toml")
	cargoContents, err := os.ReadFile(cargoFile)
	if err != nil {
		return nil, err
	}
	var manifest cargoManifest
	if err := toml.Unmarshal(cargoContents, &manifest); err != nil {
		return nil, err
	}
	if manifest.Package == nil {
		return nil, err
	}
	newVersion, err := semver.DeriveNextOptions{
		BumpVersionCore:       true,
		DowngradePreGAChanges: true,
	}.DeriveNext(semver.Minor, manifest.Package.Version)
	if err != nil {
		return nil, err
	}
	if err := rustrelease.UpdateCargoVersion(cargoFile, newVersion); err != nil {
		return nil, err
	}
	library.Version = newVersion
	return cfg, err
}

func deriveSrcPath(libCfg *config.Library, cfg *config.Config) string {
	if libCfg.Output != "" {
		return libCfg.Output
	}
	libSrcDir := ""
	if len(libCfg.Channels) > 0 && libCfg.Channels[0].Path != "" {
		libSrcDir = libCfg.Channels[0].Path
	} else {
		libSrcDir = strings.ReplaceAll(libCfg.Name, "-", "/")
		if cfg.Default == nil {
			return ""
		}
	}
	return DefaultOutput(libSrcDir, cfg.Default.Output)

}
