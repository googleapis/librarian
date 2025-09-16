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
	"log/slog"
	"slices"

	"github.com/googleapis/librarian/internal/sidekick/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/internal/external"
)

// BumpVersions finds all the crates that need a version bump and performs the
// bump, changing both the Cargo.toml and sidekick.toml files.
func BumpVersions(config *config.Release) error {
	if err := PreFlight(config); err != nil {
		return err
	}
	lastTag, err := getLastTag(config)
	if err != nil {
		return err
	}
	files, err := filesChangedSince(config, lastTag)
	if err != nil {
		return err
	}
	var crates []string
	for _, manifest := range findCargoManifests(files) {
		names, err := updateManifest(config, lastTag, manifest)
		if err != nil {
			return err
		}
		crates = append(crates, names...)
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
		if err := external.Run(cargoExe(config), "semver-checks", "--all-features", "-p", name); err != nil {
			return err
		}
	}
	return nil
}

func containsSemverChecks(a config.Tool) bool {
	return a.Name == "cargo-semver-checks"
}
