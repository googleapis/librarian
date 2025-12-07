// Copyright 2024 Google LLC
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

package sidekick

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"

	cmd "github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/sidekick/config"
	toml "github.com/pelletier/go-toml/v2"
)

func init() {
	newCommand(
		"sidekick rust-generate",
		"Runs the generator for the first time for a client library assuming the target is the Rust monorepo.",
		`
Runs the generator for the first time for a client library.

Uses the configuration provided in the command line arguments, and saving it in
a .sidekick.toml file in the output directory.

Uses the conventions in the Rust monorepo to determine the source and output
directories from the name of the service config YAML file.
`,
		cmdSidekick,
		rustGenerate,
	)
}

// rustGenerate takes some state and applies it to a template to create a client
// library.
func rustGenerate(ctx context.Context, rootConfig *config.Config, cmdLine *CommandLine) error {
	if cmdLine.SpecificationSource == "" {
		cmdLine.SpecificationSource = path.Dir(cmdLine.ServiceConfig)
	}
	if cmdLine.Output == "" {
		cmdLine.Output = path.Join("src/generated", strings.TrimPrefix(cmdLine.SpecificationSource, "google/"))
	}

	if err := VerifyRustTools(ctx); err != nil {
		return err
	}

	if err := PrepareCargoWorkspace(ctx, cmdLine.Output); err != nil {
		return err
	}
	slog.Info("generating new library code and adding it to git")
	if err := generate(ctx, rootConfig, cmdLine); err != nil {
		return err
	}
	return PostGenerate(ctx, cmdLine.Output)
}

func getPackageName(output string) (string, error) {
	cargo := CargoConfig{}
	filename := path.Join(output, "Cargo.toml")
	if contents, err := os.ReadFile(filename); err == nil {
		err = toml.Unmarshal(contents, &cargo)
		if err != nil {
			return "", fmt.Errorf("error reading %s: %w", filename, err)
		}
	}
	// Ignore errors reading the top-level file.
	return cargo.Package.Name, nil
}

// VerifyRustTools verifies that all required Rust tools are installed.
func VerifyRustTools(ctx context.Context) error {
	if err := cmd.Run(ctx, "cargo", "--version"); err != nil {
		return fmt.Errorf("got an error trying to run `cargo --version`, the instructions on https://www.rust-lang.org/learn/get-started may solve this problem: %w", err)
	}
	if err := cmd.Run(ctx, "taplo", "--version"); err != nil {
		return fmt.Errorf("got an error trying to run `taplo --version`, please install using `cargo install taplo-cli`: %w", err)
	}
	if err := cmd.Run(ctx, "typos", "--version"); err != nil {
		return fmt.Errorf("got an error trying to run `typos --version`, please install using `cargo install typos-cli`: %w", err)
	}
	if err := cmd.Run(ctx, "git", "--version"); err != nil {
		return fmt.Errorf("got an error trying to run `git --version`, the instructions on https://github.com/git-guides/install-git may solve this problem: %w", err)
	}
	return nil
}

// PrepareCargoWorkspace creates a new cargo package in the specified output directory.
func PrepareCargoWorkspace(ctx context.Context, outputDir string) error {
	slog.Info("preparing cargo workspace to get new package")
	if err := cmd.Run(ctx, "cargo", "new", "--vcs", "none", "--lib", outputDir); err != nil {
		return err
	}
	if err := cmd.Run(ctx, "taplo", "fmt", "Cargo.toml"); err != nil {
		return err
	}
	return nil
}

// PostGenerate runs post-generation tasks on the specified output directory.
func PostGenerate(ctx context.Context, outdir string) error {
	if err := cmd.Run(ctx, "cargo", "fmt"); err != nil {
		return err
	}
	if err := cmd.Run(ctx, "git", "add", outdir); err != nil {
		return err
	}
	packagez, err := getPackageName(outdir)
	if err != nil {
		return err
	}
	slog.Info("generated new client library", "package", packagez)
	slog.Info("running `cargo test` on new client library")
	if err := cmd.Run(ctx, "cargo", "test", "--package", packagez); err != nil {
		return err
	}
	slog.Info("running `cargo doc` on new client library")
	if err := cmd.Run(ctx, "env", "RUSTDOCFLAGS=-D warnings", "cargo", "doc", "--package", packagez, "--no-deps"); err != nil {
		return err
	}
	slog.Info("running `cargo clippy` on new client library")
	if err := cmd.Run(ctx, "cargo", "clippy", "--package", packagez, "--", "--deny", "warnings"); err != nil {
		return err
	}
	slog.Info("running `typos` on new client library")
	if err := cmd.Run(ctx, "typos"); err != nil {
		slog.Info("please manually add the typos to `.typos.toml` and fix the problem upstream")
		return err
	}
	return cmd.Run(ctx, "git", "add", "Cargo.lock", "Cargo.toml")
}

// CargoConfig is the configuration for a cargo package.
type CargoConfig struct {
	Package CargoPackage // `toml:"package"`
}

// CargoPackage is a cargo package.
type CargoPackage struct {
	Name string // `toml:"name"`
}
