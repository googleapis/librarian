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

import "github.com/googleapis/librarian/internal/sidekick/internal/config"

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
	var packages []string
	for _, manifest := range findCargoManifests(files) {
		names, err := updateManifest(config, lastTag, manifest)
		if err != nil {
			return err
		}
		packages = append(packages, names...)
	}
	_ = packages
	return nil
}
