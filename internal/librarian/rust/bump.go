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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/semver"
)

var (
	errMissingVersion = errors.New("must provide version")
)

// Bump checks if a version bump is required and performs it.
// It returns without error if no bump is needed (version already updated since lastTag).
func Bump(ctx context.Context, library *config.Library, output, version, gitExe, lastTag string) error {
	if version == "" {
		return errMissingVersion
	}
	cargoFile := filepath.Join(output, "Cargo.toml")
	needed, err := shouldBumpManifestVersion(ctx, gitExe, lastTag, cargoFile)
	if err != nil {
		return err
	}
	if !needed {
		return nil
	}
	return writeVersion(library, output, version)
}

func writeVersion(library *config.Library, output, versionString string) error {
	// validate version before writing to Cargo.toml
	version, err := semver.Parse(versionString)
	if err != nil {
		return err
	}
	cargoFile := filepath.Join(output, "Cargo.toml")
	_, err = os.Stat(cargoFile)
	switch {
	case err != nil && !os.IsNotExist(err):
		return err
	case os.IsNotExist(err):
		cargo := fmt.Sprintf(`[package]
name                   = "%s"
version                = "%s"
edition                = "2021"
`, library.Name, version.String())
		if err := os.WriteFile(cargoFile, []byte(cargo), 0644); err != nil {
			return err
		}
	default:
		if err := updateCargoVersion(cargoFile, version); err != nil {
			return err
		}
	}

	// Update the workspace manifest if it exists.
	if err := updateWorkspaceVersion("Cargo.toml", library.Name, version); err != nil && !os.IsNotExist(err) {
		return err
	}
	library.Version = version.String()
	return nil
}
