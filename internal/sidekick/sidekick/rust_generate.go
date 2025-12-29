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
	"log/slog"
	"path"
	"strings"

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

	slog.Info("generating new library code and adding it to git")
	return rust.Create(ctx, cmdLine.Output, func(ctx context.Context) error {
		return generate(ctx, rootConfig, cmdLine)
	})
}
