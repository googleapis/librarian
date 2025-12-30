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

// Package rustrelease implements the release automation logic for Rust crates.
package rustrelease

import (
	"context"
	"log/slog"
	"slices"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/librarian/rust"
	"github.com/googleapis/librarian/internal/sidekick/config"
)

// BumpVersions finds all the crates that need a version bump and performs the
// bump, changing both the Cargo.toml and sidekick.toml files.
func BumpVersions(ctx context.Context, config *config.Release) error {
	if err := PreFlight(ctx, config); err != nil {
		return err
	}
	lastTag, err := git.GetLastTag(ctx, gitExe(config), config.Remote, config.Branch)
	if err != nil {
		return err
	}
	files, err := git.FilesChangedSince(ctx, lastTag, gitExe(config), config.IgnoredChanges)
	if err != nil {
		return err
	}
	var crates []string
	manifests := rust.FindCargoManifests(files)
	for _, manifest := range manifests {
		newVersion, name, err := rust.UpdateManifest(gitExe(config), lastTag, manifest)
		if err != nil {
			return err
		}
		if newVersion != "" {
			if err := updateSidekickConfig(manifest, newVersion); err != nil {
				return err
			}
			crates = append(crates, name)
		}
	}
	if tools, ok := config.Tools["cargo"]; ok {
		if !slices.ContainsFunc(tools, containsSemverChecks) {
			return nil
		}
	} else {
		return nil
	}
	for _, name := range crates {
		slog.Info("runnning cargo semver-checks", "crate", name)
		if err := command.Run(ctx, cargoExe(config), "semver-checks", "--all-features", "-p", name); err != nil {
			return err
		}
	}
	return nil
}

func containsSemverChecks(a config.Tool) bool {
	return a.Name == "cargo-semver-checks"
}
