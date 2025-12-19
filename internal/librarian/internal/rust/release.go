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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/semver"
	rustrelease "github.com/googleapis/librarian/internal/sidekick/rust_release"
	"github.com/pelletier/go-toml/v2"
)

var errLibraryNotFound = errors.New("library not found")

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

	if name == "" {
		for _, library := range cfg.Libraries {
			if err := releaseLibrary(library, cfg); err != nil {
				return nil, err
			}
		}
	} else {
		library, err := libraryByName(cfg, name)
		if err != nil {
			return nil, fmt.Errorf("library %q not found", name)
		}
		if err := releaseLibrary(library, cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// libraryByName returns a library with the given name from the config.
func libraryByName(c *config.Config, name string) (*config.Library, error) {
	if c.Libraries == nil {
		return nil, errLibraryNotFound
	}
	for _, library := range c.Libraries {
		if library.Name == name {
			return library, nil
		}
	}
	return nil, errLibraryNotFound
}

func releaseLibrary(library *config.Library, cfg *config.Config) error {
	output, err := deriveSrcPath(library, cfg)
	if err != nil {
		return err
	}
	cargoFile := filepath.Join(output, "Cargo.toml")
	cargoContents, err := os.ReadFile(cargoFile)
	if err != nil {
		return err
	}
	var manifest cargoManifest
	if err := toml.Unmarshal(cargoContents, &manifest); err != nil {
		return err
	}
	if manifest.Package == nil {
		return nil
	}
	newVersion, err := semver.DeriveNextOptions{
		BumpVersionCore:       true,
		DowngradePreGAChanges: true,
	}.DeriveNext(semver.Minor, manifest.Package.Version)
	if err != nil {
		return err
	}
	if err := rustrelease.UpdateCargoVersion(cargoFile, newVersion); err != nil {
		return err
	}
	library.Version = newVersion
	return nil
}

func deriveSrcPath(libCfg *config.Library, cfg *config.Config) (string, error) {
	if libCfg.Output != "" {
		return libCfg.Output, nil
	}
	libSrcDir := ""
	if len(libCfg.Channels) > 0 && libCfg.Channels[0].Path != "" {
		libSrcDir = libCfg.Channels[0].Path
	} else {
		libSrcDir = strings.ReplaceAll(libCfg.Name, "-", "/")
		if cfg.Default == nil {
			return "", nil
		}
	}
	return DefaultOutput(libSrcDir, cfg.Default.Output), nil

}
