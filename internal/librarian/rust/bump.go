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

	"github.com/googleapis/librarian/internal/config"
)

var (
	errMissingVersion = errors.New("version must not be empty")
)

// Bump bumps version for Cargo.toml files and updates librarian config version
// for a library.
func Bump(library *config.Library, version string) (*config.Library, error) {
	if version == "" {
		return nil, errMissingVersion
	}

	cargoFile := filepath.Join(library.Output, "Cargo.toml")
	_, err := os.Stat(cargoFile)
	switch {
	case err != nil && !os.IsNotExist(err):
		return nil, err
	case os.IsNotExist(err):
		cargo := fmt.Sprintf(`[package]
name                   = "%s"
version                = "%s"
edition                = "2021"
`, library.Name, version)
		if err := os.WriteFile(cargoFile, []byte(cargo), 0644); err != nil {
			return nil, err
		}
	default:
		if err := updateCargoVersion(cargoFile, version); err != nil {
			return nil, err
		}
	}

	library.Version = version
	return library, nil
}
