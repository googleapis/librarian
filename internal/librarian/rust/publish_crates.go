// Copyright 2026 Google LLC
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
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
)

// PublishCrates publishes the crates that have changed.
func PublishCrates(ctx context.Context, config *config.Release, dryRun bool, skipSemverChecks bool, lastTag string, files []string) error {
	manifests := map[string]string{}
	for _, manifest := range FindCargoManifests(files) {
		names, err := PublishedCrate(manifest)
		if err != nil {
			return err
		}
		for _, name := range names {
			manifests[name] = manifest
		}
	}
	slog.Info("computing publication plan with: cargo workspaces plan")
	cargoPath := command.GetExecutablePath(config.Preinstalled, "cargo")
	cmd := exec.CommandContext(ctx, cargoPath, "workspaces", "plan", "--skip-published")
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
	if diff := cmp.Diff(changedCrates, plannedCrates); diff != "" && cargoPath != "/bin/echo" {
		return fmt.Errorf("mismatched workspace plan vs. changed crates, probably missing some version bumps (-plan, +changed):\n%s", diff)
	}

	crateSummary := slices.Collect(maps.Keys(manifests))
	totalCrates := len(crateSummary)
	crateSummary = crateSummary[0:min(20, totalCrates)]
	slog.Info(fmt.Sprintf("there are %d crates in need of publishing, summary=%v", totalCrates, crateSummary))

	if !skipSemverChecks {
		gitPath := command.GetExecutablePath(config.Preinstalled, "git")
		for name, manifest := range manifests {
			if git.IsNewFile(ctx, gitPath, lastTag, manifest) {
				continue
			}
			slog.Info("running cargo semver-checks to detect breaking changes", "crate", name)
			if err := command.Run(ctx, cargoPath, "semver-checks", "--all-features", "-p", name); err != nil {
				return err
			}
		}
	}
	slog.Info("publishing crates with: cargo workspaces publish --skip-published ...")
	args := []string{"workspaces", "publish", "--skip-published", "--publish-interval=60", "--no-git-commit", "--from-git", "skip"}
	if dryRun {
		args = append(args, "--dry-run")
	}
	cmd = exec.CommandContext(ctx, cargoPath, args...)
	if config.RootsPem != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("CARGO_HTTP_CAINFO=%s", config.RootsPem))
	}
	cmd.Dir = "."
	return cmd.Run()
}

// Publish finds all the crates that should be published, (optionally) runs
// `cargo semver-checks` and (optionally) publishes them.
func Publish(ctx context.Context, config *config.Release, dryRun bool, skipSemverChecks bool) error {
	if err := PreFlight(ctx, config.Preinstalled, config.Remote, config.Tools["cargo"]); err != nil {
		return err
	}
	gitPath := command.GetExecutablePath(config.Preinstalled, "git")
	lastTag, err := git.GetLastTag(ctx, gitPath, config.Remote, config.Branch)
	if err != nil {
		return err
	}
	if err := git.MatchesBranchPoint(ctx, gitPath, config.Remote, config.Branch); err != nil {
		return err
	}
	files, err := git.FilesChangedSince(ctx, lastTag, gitPath, config.IgnoredChanges)
	if err != nil {
		return err
	}
	return PublishCrates(ctx, config, dryRun, skipSemverChecks, lastTag, files)
}
