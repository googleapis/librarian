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
	"log/slog"
	"os"
	"path"

	"github.com/googleapis/librarian/internal/command"
	"github.com/pelletier/go-toml/v2"
)

// RustCreator interface used for mocking in tests.
type RustCreator interface {
	PrepareCargoWorkspace(ctx context.Context, outputDir string) error
	FormatAndValidateLibrary(ctx context.Context, outputDir string) error
}

// RustCreate struct implements RustCreator interface.
type RustCreate struct {
}

// PrepareCargoWorkspace encaspulates PrepareCargoWorkspace command.
func (r *RustCreate) PrepareCargoWorkspace(ctx context.Context, outputDir string) error {
	return prepareCargoWorkspace(ctx, outputDir)
}

// FormatAndValidateLibrary encaspulates PrepareCargoWorkspace command.
func (r *RustCreate) FormatAndValidateLibrary(ctx context.Context, outputDir string) error {
	return formatAndValidateLibrary(ctx, outputDir)
}

// getPackageName retrieves the packagename from a Cargo.toml file.
func getPackageName(output string) (string, error) {
	cargo := CargoConfig{}
	filename := path.Join(output, "Cargo.toml")
	if contents, err := os.ReadFile(filename); err == nil {
		err = toml.Unmarshal(contents, &cargo)
		if err != nil {
			return "", fmt.Errorf("error reading %s: %w", filename, err)
		}
	}
	return cargo.Package.Name, nil
}

// PrepareCargoWorkspace creates a new cargo package in the specified output directory.
func prepareCargoWorkspace(ctx context.Context, outputDir string) error {
	if err := command.Run(ctx, "cargo", "new", "--vcs", "none", "--lib", outputDir); err != nil {
		return err
	}
if err := command.Run(ctx, "taplo", "fmt", path.Join(outputDir, "Cargo.toml")); err != nil {
		return err
	}
	return nil
}

// formatAndValidateLibrary runs formatter, typos checks, tests  tasks on the specified output directory.
func formatAndValidateLibrary(ctx context.Context, outputDir string) error {
	if err := command.Run(ctx, "cargo", "fmt"); err != nil {
		return err
	}
	packagez, err := getPackageName(outputDir)
	if err != nil {
		return err
	}
	if err := command.Run(ctx, "cargo", "test", "--package", packagez); err != nil {
		return err
	}
	if err := command.Run(ctx, "env", "RUSTDOCFLAGS=-D warnings", "cargo", "doc", "--package", packagez, "--no-deps"); err != nil {
		return err
	}
	if err := command.Run(ctx, "cargo", "clippy", "--package", packagez, "--", "--deny", "warnings"); err != nil {
		return err
	}
	if err := command.Run(ctx, "typos"); err != nil {
		slog.Info("please manually add the typos to `.typos.toml` and fix the problem upstream")
		return err
	}
	return addNewFilesToGit(ctx, outputDir)
}

// addNewFilesToGit addes newly created library files and mod files to git to be committed.
func addNewFilesToGit(ctx context.Context, outputDir string) error {
	if err := command.Run(ctx, "git", "add", outputDir); err != nil {
		return err
	}
	return command.Run(ctx, "git", "add", "cargo.lock", "cargo.toml")
}

// CargoConfig is the configuration for a cargo package.
type CargoConfig struct {
	Package CargoPackage // `toml:"package"`
}

// CargoPackage is a cargo package.
type CargoPackage struct {
	Name string // `toml:"name"`
}
