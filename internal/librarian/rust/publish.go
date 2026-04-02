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
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"runtime"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	"golang.org/x/sync/errgroup"
)

// semverData holds parameters for running semver checks.
type semverData struct {
	dryRunKeepGoing bool
	manifests       map[string]string
	lastTag         string
	cargoPath       string
	gitPath         string
}

// semverCheckCPUDivisor scales the concurrency limit based on available CPUs to balance
// throughput against resource contention.
//
// Why a limit?
// `cargo semver-checks` is internally multithreaded during the compilation phase.
// Running it completely unbounded, or even 1:1 with CPU cores, can cause severe CPU
// thrashing and RAM exhaustion, as multiple instances of the Rust compiler
// compete for the same physical cores and memory bandwidth.
//
// Why a divisor of 8?
// Performance testing on 64-core workstations revealed a "sweet spot":
// Running 8 concurrent jobs (64 cores / 8) reduced execution time from ~2 hours
// down to ~17 minutes. Pushing concurrency higher yielded negligible gains (e.g.,
// 15 mins at 16-way) but massively increased system load and OOM (Out Of Memory) risks.
//
// By using a divisor instead of a hard cap, we dynamically apply this optimal 1/8th
// ratio across varied hardware. This prevents smaller CI runners or local dev machines
// from being overwhelmed while still safely maximizing throughput on larger workstations.
const semverCheckCPUDivisor = 8

// errSemverCheck is returned when a semver check fails.
var errSemverCheck = errors.New("semver check failed")

// Publish finds all the crates that should be published. It can optionally
// run in dry-run mode, dry-run mode with continue on errors, and/or skip semver checks.
func Publish(ctx context.Context, cfg *config.Release, dryRun, dryRunKeepGoing, skipSemverChecks bool) error {
	if err := preFlight(ctx, cfg.Preinstalled, cfg.Tools[command.Cargo]); err != nil {
		return err
	}
	gitExe := command.GetExecutablePath(cfg.Preinstalled, command.Git)
	lastTag, err := git.GetLastTag(ctx, gitExe, config.RemoteUpstream, config.BranchMain)
	if err != nil {
		return err
	}
	if err := git.MatchesBranchPoint(ctx, gitExe, config.RemoteUpstream, config.BranchMain); err != nil {
		return err
	}
	files, err := git.FilesChangedSince(ctx, gitExe, lastTag, cfg.IgnoredChanges)
	if err != nil {
		return err
	}
	return publishCrates(ctx, cfg, dryRun, dryRunKeepGoing, skipSemverChecks, lastTag, files)
}

// publishCrates publishes the crates that have changed.
func publishCrates(ctx context.Context, cfg *config.Release, dryRun, dryRunKeepGoing, skipSemverChecks bool, lastTag string, files []string) error {
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
	cargoPath := command.GetExecutablePath(cfg.Preinstalled, command.Cargo)
	output, err := command.Output(ctx, cargoPath, "workspaces", "plan", "--skip-published")
	if err != nil {
		return err
	}
	plannedCrates := strings.Split(string(output), "\n")
	plannedCrates = slices.DeleteFunc(plannedCrates, func(a string) bool { return a == "" })
	if !isMockCargo(cargoPath) {
		for _, crate := range plannedCrates {
			if _, ok := manifests[crate]; !ok {
				return fmt.Errorf("unplanned crate %q found in workspace plan", crate)
			}
		}
	}

	crateSummary := slices.Collect(maps.Keys(manifests))
	totalCrates := len(crateSummary)
	crateSummary = crateSummary[0:min(20, totalCrates)]
	slog.Info(fmt.Sprintf("there are %d crates in need of publishing, summary=%v", totalCrates, crateSummary))

	if !skipSemverChecks {
		gitPath := command.GetExecutablePath(cfg.Preinstalled, command.Git)
		if err := runSemverChecks(ctx, semverData{
			dryRunKeepGoing: dryRunKeepGoing,
			manifests:       manifests,
			lastTag:         lastTag,
			cargoPath:       cargoPath,
			gitPath:         gitPath,
		}); err != nil {
			return err
		}
	}
	slog.Info("publishing crates with: cargo workspaces publish --skip-published ...")
	args := []string{"workspaces", "publish", "--skip-published", "--publish-interval=60", "--no-git-commit", "--from-git", "skip"}
	if dryRunKeepGoing {
		args = append(args, "--dry-run", "--keep-going")
	} else if dryRun {
		args = append(args, "--dry-run")
	}
	return command.Run(ctx, cargoPath, args...)
}

// runSemverChecks iterates through manifests and runs semver checks for each.
func runSemverChecks(ctx context.Context, semverData semverData) error {
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(max(runtime.NumCPU()/semverCheckCPUDivisor, 1))
	for name, manifest := range semverData.manifests {
		group.Go(func() error {
			if err := semverCheck(ctx, semverData, name, manifest); err != nil {
				return fmt.Errorf("%s: %w: %v", name, errSemverCheck, err)
			}
			return nil
		})
	}
	return group.Wait()
}

// semverCheck runs semver checks for a specific crate.
func semverCheck(ctx context.Context, semverData semverData, name string, manifest string) error {
	if git.IsNewFile(ctx, semverData.gitPath, semverData.lastTag, manifest) {
		// If the manifest is new, we can skip semver checks, since there is no previous version to compare against.
		return nil
	}
	err := command.Run(ctx, semverData.cargoPath, "semver-checks", "--all-features", "-p", name)
	if err != nil && semverData.dryRunKeepGoing {
		slog.Warn("semver check failed, but continuing due to --keep-going", "crate", name, "error", err)
		return nil
	}
	return err
}

func isMockCargo(path string) bool {
	return path == "/bin/echo"
}
