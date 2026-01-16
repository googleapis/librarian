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
	"path"

	"github.com/googleapis/librarian/internal/command"
)

// Create creates a cargo workspace, runs the provided generation function, and
// validates the library.
//
// TODO(https://github.com/googleapis/librarian/issues/3219): generateFn can be
// removed once sidekick.rustGenerate is deprecated.
func Create(ctx context.Context, outputDir string, generateFn func(context.Context) error) error {
	if err := command.Run(ctx, "cargo", "--version"); err != nil {
		return fmt.Errorf("got an error trying to run `cargo --version`, the instructions on https://www.rust-lang.org/learn/get-started may solve this problem: %w", err)
	}
	if err := command.Run(ctx, "taplo", "--version"); err != nil {
		return fmt.Errorf("got an error trying to run `taplo --version`, please install using `cargo install taplo-cli`: %w", err)
	}
	if err := command.Run(ctx, "cargo", "new", "--vcs", "none", "--lib", outputDir); err != nil {
		return err
	}
	if err := command.Run(ctx, "taplo", "fmt", "Cargo.toml"); err != nil {
		return err
	}
	if err := generateFn(ctx); err != nil {
		return err
	}

	manifestPath := path.Join(outputDir, "Cargo.toml")
	if err := command.Run(ctx, "cargo", "fmt", "--manifest-path", manifestPath); err != nil {
		return err
	}
	if err := command.Run(ctx, "cargo", "test", "--manifest-path", manifestPath); err != nil {
		return err
	}
	return command.RunWithEnv(ctx, map[string]string{"RUSTDOCFLAGS": "-D warnings"}, "cargo", "doc", "--manifest-path", manifestPath, "--no-deps")
}
