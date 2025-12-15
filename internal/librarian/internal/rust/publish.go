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
	"maps"
	"os"
	stdexec "os/exec"
	"slices"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/change"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

// Publish finds all the crates that should be published, (optionally) runs
// `cargo semver-checks` and (optionally) publishes them.
func Publish(ctx context.Context, cfg *config.Config, dryRun bool, skipSemverChecks bool, lastTag string, files []string) error {
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
	// computing publication plan with cmd
	cmd := stdexec.CommandContext(ctx, cfg.Release.GetExecutablePath("cargo"), "workspaces", "plan", "--skip-published")
	if cfg.Release.RootsPem != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("CARGO_HTTP_CAINFO=%s", cfg.Release.RootsPem))
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
	if diff := cmp.Diff(changedCrates, plannedCrates); diff != "" {
		return fmt.Errorf("mismatched workspace plan vs. changed crates, probably missing some version bumps (-plan, +changed):\n%s", diff)
	}

	if !skipSemverChecks {
		for name, manifest := range manifests {
			if change.IsNewFile(ctx, cfg.Release.GetExecutablePath("git"), lastTag, manifest) {
				continue
			}
			slog.Info("runnning cargo semver-checks to detect breaking changes", "crate", name)
			if err := command.Run(ctx, cfg.Release.GetExecutablePath("cargo"), "semver-checks", "--all-features", "-p", name); err != nil {
				return err
			}
		}
	}
	// publish crates with cargo cmd
	args := []string{"workspaces", "publish", "--skip-published", "--publish-interval=60", "--no-git-commit", "--from-git", "skip"}
	if dryRun {
		args = append(args, "--dry-run")
	}
	cmd = stdexec.CommandContext(ctx, cfg.Release.GetExecutablePath("cargo"), args...)
	if cfg.Release.RootsPem != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("CARGO_HTTP_CAINFO=%s", cfg.Release.RootsPem))
	}
	cmd.Dir = "."
	return cmd.Run()
}

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
