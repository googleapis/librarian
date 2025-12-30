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

package rustrelease

import (
	"context"

	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/librarian/rust"
	"github.com/googleapis/librarian/internal/sidekick/config"
)

// PreFlight() verifies all the necessary  tools are installed.
func PreFlight(ctx context.Context, config *config.Release) error {
	gitExe := gitExe(config)
	if err := git.GitVersion(ctx, gitExe); err != nil {
		return err
	}
	if err := git.GitRemoteURL(ctx, gitExe, config.Remote); err != nil {
		return err
	}
	if tools, ok := config.Tools["cargo"]; ok {
		var rustTools []rust.Tool
		for _, t := range tools {
			rustTools = append(rustTools, rust.Tool{Name: t.Name, Version: t.Version})
		}
		if err := rust.VerifyCargoTools(ctx, cargoExe(config), rustTools); err != nil {
			return err
		}
	}
	return nil
}

func gitExe(config *config.Release) string {
	if exe, ok := config.Preinstalled["git"]; ok {
		return exe
	}
	return "git"
}

func cargoExe(config *config.Release) string {
	if exe, ok := config.Preinstalled["cargo"]; ok {
		return exe
	}
	return "cargo"
}
