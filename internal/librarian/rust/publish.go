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
	"os"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/sync/errgroup"
)

// semverData holds parameters for running semver checks.
type semverData struct {
	dryRunKeepGoing bool
	manifests       map[string]string
	lastTag         string
	verbose         bool
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
const (
	semverCheckCPUDivisor  = 8
	defaultPublishInterval = 60
	defaultBatchSize       = 5
)

// errSemverCheck is returned when a semver check fails.
var errSemverCheck = errors.New("semver check failed")

// PublishParams holds parameters for running the Publish function.
type PublishParams struct {
	// Config is the repository configuration.
	Config *config.Config
	// DryRun indicates whether to run publish without actually pushing crates.
	DryRun bool
	// DryRunKeepGoing indicates whether to run in dry-run mode without stopping on errors.
	DryRunKeepGoing bool
	// SkipSemverChecks indicates whether to skip semantic versioning checks.
	SkipSemverChecks bool
	// Verbose indicates whether to stream the output of executed commands.
	Verbose bool
	// IgnoredChanges is a list of file paths/patterns to ignore when detecting changed crates.
	IgnoredChanges []string
	// PublishInterval is the number of seconds to wait between publish batches.
	PublishInterval int
	// BatchSize is the maximum number of crates to publish in a single batch.
	BatchSize int
}

// Publish finds all the crates that should be published. It can optionally
// run in dry-run mode, dry-run mode with continue on errors, and/or skip semver checks.
func Publish(ctx context.Context, params PublishParams) error {
	var tools []*config.CargoTool
	if params.Config != nil && params.Config.Tools != nil {
		tools = params.Config.Tools.Cargo
	}
	if err := preFlight(ctx, tools); err != nil {
		return err
	}
	lastTag, err := git.GetLastTag(ctx, command.Git, config.RemoteUpstream, config.BranchMain)
	if err != nil {
		return err
	}
	if err := git.MatchesBranchPoint(ctx, command.Git, config.RemoteUpstream, config.BranchMain); err != nil {
		return err
	}
	files, err := git.FilesChangedSince(ctx, command.Git, lastTag, params.IgnoredChanges)
	if err != nil {
		return err
	}
	return publishCrates(ctx, params, lastTag, files)
}

// publishCrates publishes the crates that have changed.
func publishCrates(ctx context.Context, params PublishParams, lastTag string, files []string) error {
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
	output, err := command.Output(ctx, command.Cargo, "workspaces", "plan", "--skip-published")
	if err != nil {
		return err
	}
	plannedCrates := strings.Split(string(output), "\n")
	plannedCrates = slices.DeleteFunc(plannedCrates, func(a string) bool { return a == "" })
	for _, crate := range plannedCrates {
		if _, ok := manifests[crate]; !ok {
			return fmt.Errorf("unplanned crate %q found in workspace plan", crate)
		}
	}

	if !params.SkipSemverChecks {
		if err := runSemverChecks(ctx, semverData{
			dryRunKeepGoing: params.DryRunKeepGoing,
			manifests:       manifests,
			lastTag:         lastTag,
			verbose:         params.Verbose,
		}); err != nil {
			return err
		}
	}
	interval := params.PublishInterval
	if interval <= 0 {
		interval = defaultPublishInterval
	}
	batchSize := params.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}

	batches, err := batchCrates(plannedCrates, manifests, batchSize)
	if err != nil {
		return err
	}

	for i, batch := range batches {
		for _, crate := range batch {
			if err := publishSingleCrate(ctx, params, crate); err != nil {
				if params.DryRunKeepGoing {
					slog.Warn("publish failed, but continuing due to --keep-going", "crate", crate, "error", err)
					continue
				}
				return err
			}
		}
		if i < len(batches)-1 && interval > 0 {
			slog.Info("waiting between publish batches", "seconds", interval, "completed_batch", i+1, "total_batches", len(batches))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(interval) * time.Second):
			}
		}
	}
	return nil
}

// publishSingleCrate publishes an individual crate using cargo publish.
func publishSingleCrate(ctx context.Context, params PublishParams, crate string) error {
	args := []string{"publish", "-p", crate}
	if params.DryRun || params.DryRunKeepGoing {
		args = append(args, "--dry-run")
	}
	if params.Verbose {
		return command.RunStreaming(ctx, command.Cargo, args...)
	}
	return command.Run(ctx, command.Cargo, args...)
}

type cargoManifestDependencies struct {
	Dependencies      map[string]any `toml:"dependencies"`
	DevDependencies   map[string]any `toml:"dev-dependencies"`
	BuildDependencies map[string]any `toml:"build-dependencies"`
}

type crateInfo struct {
	name string
	deps map[string]bool
}

// crateWorkspaceDeps reads a manifest file and returns workspace crate dependencies wrapped in crateInfo.
func crateWorkspaceDeps(crate string, manifestPath string, workspaceCrates map[string]string) (crateInfo, error) {
	contents, err := os.ReadFile(manifestPath)
	if err != nil {
		return crateInfo{}, err
	}
	var manifest cargoManifestDependencies
	if err := toml.Unmarshal(contents, &manifest); err != nil {
		return crateInfo{}, err
	}
	info := crateInfo{
		name: crate,
		deps: make(map[string]bool),
	}
	addDeps := func(m map[string]any) {
		for name := range m {
			if _, ok := workspaceCrates[name]; ok {
				info.deps[name] = true
			}
		}
	}
	addDeps(manifest.Dependencies)
	addDeps(manifest.DevDependencies)
	addDeps(manifest.BuildDependencies)
	return info, nil
}

// batchCrates splits planned crates into batches of up to batchSize, ensuring no crate in a batch
// depends on another crate in the same batch.
func batchCrates(plannedCrates []string, manifests map[string]string, batchSize int) ([][]string, error) {
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	crateInfos := make(map[string]crateInfo, len(plannedCrates))
	for _, crate := range plannedCrates {
		manifest := manifests[crate]
		info, err := crateWorkspaceDeps(crate, manifest, manifests)
		if err != nil {
			return nil, fmt.Errorf("failed to read dependencies for %s: %w", crate, err)
		}
		crateInfos[crate] = info
	}

	var batches [][]string
	var currBatch []string
	currBatchSet := map[string]bool{}

	for _, crate := range plannedCrates {
		info := crateInfos[crate]
		hasIntraBatchDep := false
		for dep := range info.deps {
			if currBatchSet[dep] {
				hasIntraBatchDep = true
				break
			}
		}
		if len(currBatch) >= batchSize || hasIntraBatchDep {
			if len(currBatch) > 0 {
				batches = append(batches, currBatch)
			}
			currBatch = []string{crate}
			currBatchSet = map[string]bool{crate: true}
		} else {
			currBatch = append(currBatch, crate)
			currBatchSet[crate] = true
		}
	}
	if len(currBatch) > 0 {
		batches = append(batches, currBatch)
	}
	return batches, nil
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
	if git.IsNewFile(ctx, command.Git, semverData.lastTag, manifest) {
		// If the manifest is new, we can skip semver checks, since there is no previous version to compare against.
		return nil
	}
	var err error
	if semverData.verbose {
		err = command.RunStreaming(ctx, command.Cargo, "semver-checks", "--all-features", "-p", name)
	} else {
		err = command.Run(ctx, command.Cargo, "semver-checks", "--all-features", "-p", name)
	}
	if err != nil && semverData.dryRunKeepGoing {
		slog.Warn("semver check failed, but continuing due to --keep-going", "crate", name, "error", err)
		return nil
	}
	return err
}
