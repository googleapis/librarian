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
	"path"
	"strings"

	cmd "github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/librarian/rust"
	"github.com/googleapis/librarian/internal/sidekick/config"
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

	// As part of migration to librarian we removed call to typos, so specifically call it here.
	if err := cmd.Run(ctx, "typos", "--version"); err != nil {
		return fmt.Errorf("got an error trying to run `typos --version`, please install using `cargo install typos-cli`: %w", err)
	}

	if err := rust.VerifyRustTools(ctx); err != nil {
		return err
	}

	if err := rust.PrepareCargoWorkspace(ctx, cmdLine.Output); err != nil {
		return err
	}
	slog.Info("generating new library code and adding it to git")
	if err := generate(ctx, rootConfig, cmdLine); err != nil {
		return err
	}
	return PostGenerate(ctx, cmdLine.Output)
}

// PostGenerate runs post-generation tasks on the specified output directory.
func PostGenerate(ctx context.Context, outdir string) error {
	if err := rust.FormatAndValidateLibrary(ctx, outdir); err != nil {
		return nil
	}
	slog.Info("running `typos` on new client library")
	if err := cmd.Run(ctx, "typos"); err != nil {
		slog.Info("please manually add the typos to `.typos.toml` and fix the problem upstream")
		return err
	}
	return nil
}
