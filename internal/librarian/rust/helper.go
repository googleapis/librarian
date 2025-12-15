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
	"os"
	"path"

	"github.com/googleapis/librarian/internal/command"
	"github.com/pelletier/go-toml/v2"
)

// RustHelper interface used for mocking in tests.
type RustHelper interface {
	HelperPrepareCargoWorkspace(ctx context.Context, outputDir string) error
	HelperFormatAndValidateLibrary(ctx context.Context, outputDir string) error
}

// RustHelp struct implements RustCreator interface.
type RustHelp struct {
}

// HelperPrepareCargoWorkspace encaspulates prepareCargoWorkspace command.
func (r *RustHelp) HelperPrepareCargoWorkspace(ctx context.Context, outputDir string) error {
	return PrepareCargoWorkspace(ctx, outputDir)
}

// HelperFormatAndValidateLibrary encaspulates formatAndValidateLibrary command.
func (r *RustHelp) HelperFormatAndValidateLibrary(ctx context.Context, outputDir string) error {
	return FormatAndValidateLibrary(ctx, outputDir)
}

// getPackageName retrieves the packagename from a Cargo.toml file.
func getPackageName(output string) (string, error) {
	cargo := CargoConfig{}
	filename := path.Join(output, "Cargo.toml")
	contents, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", filename, err)
	}
	if err = toml.Unmarshal(contents, &cargo); err != nil {
		return "", fmt.Errorf("error unmarshaling %s: %w", filename, err)
	}
	return cargo.Package.Name, nil
}

// PrepareCargoWorkspace creates a new cargo package in the specified output directory.
func PrepareCargoWorkspace(ctx context.Context, outputDir string) error {
	if err := command.Run(ctx, "cargo", "new", "--vcs", "none", "--lib", outputDir); err != nil {
		return err
	}
	if err := command.Run(ctx, "taplo", "fmt", "Cargo.toml"); err != nil {
		return err
	}
	return nil
}

// formatAndValidateLibrary runs formatter, typos checks, tests  tasks on the specified output directory.
func FormatAndValidateLibrary(ctx context.Context, outputDir string) error {
	manifestPath := path.Join(outputDir, "Cargo.toml")
	if err := command.Run(ctx, "cargo", "fmt", "--manifest-path", manifestPath); err != nil {
		return err
	}
	if err := command.Run(ctx, "cargo", "test", "--manifest-path", manifestPath); err != nil {
		return err
	}
	if err := command.Run(ctx, "env", "RUSTDOCFLAGS=-D warnings", "cargo", "doc", "--manifest-path", manifestPath, "--no-deps"); err != nil {
		return err
	}
	if err := command.Run(ctx, "cargo", "clippy", "--manifest-path", manifestPath, "--", "--deny", "warnings"); err != nil {
		return err
	}
	return addNewFilesToGit(ctx, outputDir)
}

// addNewFilesToGit addes newly created library files and mod files to git to be committed.
func addNewFilesToGit(ctx context.Context, outputDir string) error {
	if err := command.Run(ctx, "git", "add", outputDir); err != nil {
		return err
	}
	return command.Run(ctx, "git", "add", "Cargo.lock", "Cargo.toml")
}

// CargoConfig is the configuration for a cargo package.
type CargoConfig struct {
	Package CargoPackage `toml:"package"`
}

// CargoPackage is a cargo package.
type CargoPackage struct {
	Name string `toml:"name"`
}
