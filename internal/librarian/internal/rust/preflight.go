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

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

// PreFlight() verifies all the necessary rust tools are installed.
func PreFlight(ctx context.Context, cfg *config.Release) error {
	if err := command.Run(ctx, cfg.GetExecutablePath("cargo"), "--version"); err != nil {
		return err
	}

	tools, ok := cfg.Tools["cargo"]
	if !ok {
		return nil
	}
	for _, tool := range tools {
		slog.Info("installing cargo tool", "name", tool.Name, "version", tool.Version)
		spec := fmt.Sprintf("%s@%s", tool.Name, tool.Version)
		if err := command.Run(ctx, cfg.GetExecutablePath("cargo"), "install", "--locked", spec); err != nil {
			return err
		}
	}
	return nil
}
