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
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/internal/external"
)

// Publish finds all the crates that should be published, (optionally) runs
// `cargo semver-checks` and (optionally) publishes them.
func Publish(config *config.Release, dryRun bool, skipSemverChecks bool) error {
	if err := PreFlight(config); err != nil {
		return err
	}
	lastTag, err := getLastTag(config)
	if err != nil {
		return err
	}
	if err := matchesBranchPoint(config); err != nil {
		return err
	}
	files, err := filesChangedSince(config, lastTag)
	if err != nil {
		return err
	}
	manifests := map[string]string{}
	for _, manifest := range findCargoManifests(files) {
		names, err := publishedCrate(manifest)
		if err != nil {
			return err
		}
		for _, name := range names {
			manifests[name] = manifest
		}
	}
	slog.Info("computing publication plan with: cargo workspaces plan")
	cmd := exec.Command(cargoExe(config), "workspaces", "plan", "--skip-published")
	if config.RootsPem != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("CARGO_HTTP_CAINFO=%s", config.RootsPem))
	}
	cmd.Dir = "."
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	plannedCrates := strings.Split(string(output), "\n")
	plannedCrates = slices.DeleteFunc(plannedCrates, func(a string) bool { return a == "" })
	changedCrates := slices.Collect(maps.Keys(manifests))
	slices.Sort(plannedCrates)
	slices.Sort(changedCrates)
	if diff := cmp.Diff(changedCrates, plannedCrates); diff != "" && cargoExe(config) != "/bin/echo" {
		return fmt.Errorf("mismatched workspace plan vs. changed crates, probably missing some version bumps (-plan, +changed):\n%s", diff)
	}

	if !skipSemverChecks {
		for name, manifest := range manifests {
			if isNewFile(config, lastTag, manifest) {
				continue
			}
			slog.Info("runnning cargo semver-checks to detect breaking changes", "crate", name)
			if err := external.Run(cargoExe(config), "semver-checks", "--all-features", "-p", name); err != nil {
				return err
			}
		}
	}
	slog.Info("publishing crates with: cargo workspaces publish --skip-published ...")
	args := []string{"workspaces", "publish", "--skip-published", "--publish-interval=60", "--no-git-commit", "--from-git", "skip"}
	if dryRun {
		args = append(args, "--dry-run")
	}
	cmd = exec.Command(cargoExe(config), args...)
	if config.RootsPem != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("CARGO_HTTP_CAINFO=%s", config.RootsPem))
	}
	cmd.Dir = "."
	return cmd.Run()
}
