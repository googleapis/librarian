// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
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
	"github.com/googleapis/librarian/internal/git"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
)

// PreFlight performs all the necessary checks before a release.
func PreFlight(ctx context.Context, preinstalled map[string]string, remote string, cargoTools []config.Tool) error {
	gitExe := command.GetExecutablePath(preinstalled, "git")
	if err := git.GitVersion(ctx, gitExe); err != nil {
		return err
	}
	if err := git.GitRemoteURL(ctx, gitExe, remote); err != nil {
		return err
	}
	return cargoPreFlight(ctx, command.GetExecutablePath(preinstalled, "cargo"), cargoTools)
}

// cargoPreFlight verifies all the necessary cargo tools are installed.
func cargoPreFlight(ctx context.Context, cargoExe string, tools []config.Tool) error {
	if err := command.Run(ctx, cargoExe, "--version"); err != nil {
		return err
	}
	for _, tool := range tools {
		if tool.Version == "" {
			continue
		}
		slog.Info("installing cargo tool", "name", tool.Name, "version", tool.Version)
		spec := fmt.Sprintf("%s@%s", tool.Name, tool.Version)
		if err := command.Run(ctx, cargoExe, "install", "--locked", spec); err != nil {
			return err
		}
	}
	return nil
}

// FromSidekickReleaseConfig converts a sidekick Release config to a librarian Release config.
func FromSidekickReleaseConfig(s *sidekickconfig.Release) *config.Release {
	if s == nil {
		return nil
	}
	tools := map[string][]config.Tool{}
	for k, v := range s.Tools {
		var l []config.Tool
		for _, t := range v {
			l = append(l, config.Tool{Name: t.Name, Version: t.Version})
		}
		tools[k] = l
	}
	return &config.Release{
		Remote:         s.Remote,
		Branch:         s.Branch,
		Preinstalled:   s.Preinstalled,
		IgnoredChanges: s.IgnoredChanges,
		RootsPem:       s.RootsPem,
		Tools:          tools,
	}
}

// ToConfigTools converts a slice of sidekick tools to a slice of librarian tools.
func ToConfigTools(sidekickTools []sidekickconfig.Tool) []config.Tool {
	if sidekickTools == nil {
		return nil
	}
	configTools := make([]config.Tool, len(sidekickTools))
	for i, t := range sidekickTools {
		configTools[i] = config.Tool{Name: t.Name, Version: t.Version}
	}
	return configTools
}
