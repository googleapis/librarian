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
	"fmt"
	"log/slog"

	"github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/sidekick/external"
)

// PreFlight() verifies all the necessary tools are installed.
func PreFlight(config *config.Release) error {
	if err := external.Run(gitExe(config), "--version"); err != nil {
		return err
	}
	if err := external.Run(cargoExe(config), "--version"); err != nil {
		return err
	}
	if err := external.Run(gitExe(config), "remote", "get-url", config.Remote); err != nil {
		return err
	}
	tools, ok := config.Tools["cargo"]
	if !ok {
		return nil
	}
	for _, tool := range tools {
		slog.Info("installing cargo tool", "name", tool.Name, "version", tool.Version)
		spec := fmt.Sprintf("%s@%s", tool.Name, tool.Version)
		if err := external.Run(cargoExe(config), "install", "--locked", spec); err != nil {
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
