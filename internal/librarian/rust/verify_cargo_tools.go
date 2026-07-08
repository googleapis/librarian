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

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
)

// preFlight performs all the necessary checks before a release.
func preFlight(ctx context.Context, cargoTools []*config.CargoTool) error {
	if err := git.CheckVersion(ctx, command.Git); err != nil {
		return err
	}
	if err := git.CheckRemoteURL(ctx, command.Git, config.RemoteUpstream); err != nil {
		return err
	}
	return cargoPreFlight(ctx, cargoTools)
}

// cargoPreFlight verifies all the necessary cargo tools are installed.
func cargoPreFlight(ctx context.Context, tools []*config.CargoTool) error {
	if err := command.Run(ctx, command.Cargo, "--version"); err != nil {
		return err
	}
	for _, tool := range tools {
		if tool.Version == "" {
			continue
		}
		spec := fmt.Sprintf("%s@%s", tool.Name, tool.Version)
		if err := command.Run(ctx, command.Cargo, "install", "--locked", spec); err != nil {
			return err
		}
	}
	return nil
}
