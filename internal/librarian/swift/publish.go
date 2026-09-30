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

package swift

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	"golang.org/x/sync/errgroup"
)

// PublishParams holds parameters for running the Swift Publish function.
type PublishParams struct {
	// Config is the repository configuration.
	Config *config.Config
	// Libraries is an optional list of library names or paths to publish.
	Libraries []string
	// DryRun indicates whether to run publish without pushing.
	DryRun bool
	// DryRunKeepGoing indicates whether to run in dry-run mode without stopping on errors.
	DryRunKeepGoing bool
	// SkipSemverChecks indicates whether to skip semantic versioning checks.
	SkipSemverChecks bool
	// Verbose indicates whether to stream the output of executed commands.
	Verbose bool
	// Force indicates whether to force push to the remote repository.
	Force bool
	// Concurrency is the maximum number of concurrent remote operations (default: 8).
	Concurrency int
	// IgnoredChanges is a list of file paths/patterns to ignore when detecting changed libraries.
	IgnoredChanges []string
	// RemoteURLFormat is an optional template for remote repository URLs (e.g. 'git@github.com:googleapis/{name}.git').
	RemoteURLFormat string
	// Origin is the source commit or branch to split from (default: HEAD).
	Origin string
	// RemoteBranch is the target branch on the remote repository (default: main).
	RemoteBranch string
	// Upstream is the name of the upstream git remote (default: upstream).
	Upstream string
	// RootFiles is the list of root files to preserve (default: LICENSE, CODE_OF_CONDUCT.md, CONTRIBUTING.md).
	RootFiles []string
	// GitExe is the path to the git binary (default: command.Git).
	GitExe string
}

type publishCandidate struct {
	lib       *config.Library
	libDir    string
	repoName  string
	remoteURL string
	tag       string
}

// Publish discovers internal dependencies among Swift libraries, checks if their versions
// have already been published, splits their commit histories in parallel, and pushes them
// in topological dependency order.
func Publish(ctx context.Context, params PublishParams) error {
	gitExe := params.GitExe
	if gitExe == "" {
		gitExe = command.Git
	}
	upstream := params.Upstream
	if upstream == "" {
		upstream = config.RemoteUpstream
	}

	if err := git.MatchesBranchPoint(ctx, gitExe, upstream, config.BranchMain); err != nil {
		if params.DryRunKeepGoing {
			slog.Error("Branch point check failed, but continuing due to --keep-going", "error", err)
		} else {
			return err
		}
	}

	if params.Config == nil {
		return nil
	}

	origin := params.Origin
	if origin == "" {
		origin = "HEAD"
	}
	remoteBranch := params.RemoteBranch
	if remoteBranch == "" {
		remoteBranch = config.BranchMain
	}
	rootFiles := params.RootFiles
	if len(rootFiles) == 0 {
		rootFiles = DefaultRootFiles
	}
	concurrency := params.Concurrency
	if concurrency <= 0 {
		concurrency = 8
	}

	var eligibleLibs []*config.Library
	var candidates []publishCandidate

	for _, lib := range params.Config.Libraries {
		if lib.SkipRelease || lib.Version == "" {
			continue
		}

		libDir := libraryPackageDirectory(lib, params.Config.Default)
		if len(params.Libraries) > 0 && !matchLibrary(params.Libraries, lib, libDir) {
			continue
		}

		if libDir == "" {
			if params.DryRunKeepGoing {
				slog.Error("library directory is empty, skipping", "library", lib.Name)
				continue
			}
			return fmt.Errorf("library %s has no output directory configured", lib.Name)
		}

		if _, err := os.Stat(libDir); err != nil {
			if params.DryRunKeepGoing {
				slog.Error("library directory not found, but continuing due to --keep-going", "library", lib.Name, "path", libDir, "error", err)
				continue
			}
			return fmt.Errorf("library directory %s does not exist: %w", libDir, err)
		}

		repoName := SplitRepoName(libDir)
		remoteURL := FormatRemoteURL(params.RemoteURLFormat, params.Config.Repo, repoName)
		tag := lib.Version

		eligibleLibs = append(eligibleLibs, lib)
		candidates = append(candidates, publishCandidate{
			lib:       lib,
			libDir:    libDir,
			repoName:  repoName,
			remoteURL: remoteURL,
			tag:       tag,
		})
	}

	if len(candidates) == 0 {
		return nil
	}

	deps, err := buildDependencyGraph(params.Config, eligibleLibs)
	if err != nil {
		if params.DryRunKeepGoing {
			slog.Error("failed to build dependency graph, but continuing due to --keep-going", "error", err)
		} else {
			return err
		}
	}

	levels, err := topologicalLevels(eligibleLibs, deps)
	if err != nil {
		if params.DryRunKeepGoing {
			slog.Error("topological sort failed, but continuing due to --keep-going", "error", err)
			levels = [][]*config.Library{eligibleLibs}
		} else {
			return err
		}
	}

	// Phase 1: Parallel Pre-Check (git.RemoteTagExists across all candidates)
	needsPublish := make([]bool, len(candidates))
	checkGroup, checkCtx := errgroup.WithContext(ctx)
	checkGroup.SetLimit(max(concurrency*2, 16))

	for i, c := range candidates {
		checkGroup.Go(func() error {
			tagExists, err := git.RemoteTagExists(checkCtx, gitExe, c.remoteURL, c.tag)
			if err != nil {
				if params.DryRunKeepGoing {
					slog.Error("failed to check remote tags, but continuing due to --keep-going", "library", c.lib.Name, "remote", c.remoteURL, "error", err)
					return nil
				}
				return fmt.Errorf("failed to check remote tags for %s on %s: %w", c.lib.Name, c.remoteURL, err)
			}
			if tagExists {
				slog.Info("version already tagged on remote repository, skipping", "library", c.lib.Name, "version", c.tag, "remote", c.remoteURL)
				return nil
			}
			needsPublish[i] = true
			return nil
		})
	}
	if err := checkGroup.Wait(); err != nil {
		return err
	}

	var toPublish []publishCandidate
	toPublishMap := make(map[string]publishCandidate)
	for i, c := range candidates {
		if needsPublish[i] {
			toPublish = append(toPublish, c)
			toPublishMap[c.lib.Name] = c
		}
	}

	if len(toPublish) == 0 {
		slog.Info("all eligible libraries are already tagged on remote repositories")
		return nil
	}

	// Phase 2: Parallel Local History Split (swift.Split across unpublished libraries)
	var rootEntries []string
	if len(rootFiles) > 0 {
		var err error
		rootEntries, err = getRootEntries(ctx, gitExe, origin, rootFiles)
		if err != nil {
			if params.DryRunKeepGoing {
				slog.Error("failed to get root entries, but continuing due to --keep-going", "error", err)
			} else {
				return err
			}
		}
	}

	splitSHAs := make(map[string]string)
	var splitMu sync.Mutex

	splitGroup, splitCtx := errgroup.WithContext(ctx)
	splitGroup.SetLimit(runtime.NumCPU())

	for _, c := range toPublish {
		splitGroup.Go(func() error {
			slog.Info("splitting repository for library", "library", c.lib.Name, "path", c.libDir, "version", c.tag, "remote", c.remoteURL)
			splitSHA, err := Split(splitCtx, SplitParams{
				TargetDir:   c.libDir,
				Origin:      origin,
				RootFiles:   rootFiles,
				RootEntries: rootEntries,
				GitExe:      gitExe,
			})
			if err != nil {
				if params.DryRunKeepGoing {
					slog.Error("failed to split library, but continuing due to --keep-going", "library", c.lib.Name, "error", err)
					return nil
				}
				return fmt.Errorf("failed to split %s: %w", c.lib.Name, err)
			}
			splitMu.Lock()
			splitSHAs[c.lib.Name] = splitSHA
			splitMu.Unlock()
			return nil
		})
	}
	if err := splitGroup.Wait(); err != nil {
		return err
	}

	// Phase 3: Topological Parallel Push (Level by Level)
	for levelIdx, level := range levels {
		var levelCandidates []publishCandidate
		for _, lib := range level {
			if c, ok := toPublishMap[lib.Name]; ok {
				splitMu.Lock()
				_, hasSHA := splitSHAs[lib.Name]
				splitMu.Unlock()
				if hasSHA {
					levelCandidates = append(levelCandidates, c)
				}
			}
		}
		if len(levelCandidates) == 0 {
			continue
		}

		slog.Info("pushing topological level", "level", levelIdx, "count", len(levelCandidates))

		if params.DryRun || params.DryRunKeepGoing {
			for _, c := range levelCandidates {
				splitMu.Lock()
				sha := splitSHAs[c.lib.Name]
				splitMu.Unlock()
				slog.Info("[DRY-RUN] Would push to remote", "library", c.lib.Name, "remote", c.remoteURL, "branch", remoteBranch, "sha", sha, "tag", c.tag)
			}
			continue
		}

		pushGroup, pushCtx := errgroup.WithContext(ctx)
		pushGroup.SetLimit(concurrency)

		for _, c := range levelCandidates {
			splitMu.Lock()
			sha := splitSHAs[c.lib.Name]
			splitMu.Unlock()

			pushGroup.Go(func() error {
				if err := git.PushBranchAndTag(pushCtx, gitExe, c.remoteURL, sha, remoteBranch, c.tag, params.Force); err != nil {
					if params.DryRunKeepGoing {
						slog.Error("failed to push, but continuing due to --keep-going", "library", c.lib.Name, "remote", c.remoteURL, "error", err)
						return nil
					}
					return fmt.Errorf("failed to push %s to %s: %w", c.lib.Name, c.remoteURL, err)
				}
				slog.Info("successfully published library", "library", c.lib.Name, "version", c.tag, "remote", c.remoteURL)
				return nil
			})
		}
		if err := pushGroup.Wait(); err != nil {
			return err
		}
	}

	return nil
}

// FormatRemoteURL constructs the remote repository URL for a library.
func FormatRemoteURL(format, repo, name string) string {
	if format != "" {
		return strings.ReplaceAll(format, "{name}", name)
	}
	org := "googleapis"
	if repo != "" {
		parts := strings.Split(repo, "/")
		if len(parts) > 0 && parts[0] != "" {
			org = parts[0]
		}
	}
	return fmt.Sprintf("git@github.com:%s/%s.git", org, name)
}

func libraryOutput(lib *config.Library, defaults *config.Default) string {
	if lib.Output != "" {
		return lib.Output
	}
	if IsMixedLibrary(lib) {
		return ""
	}
	apiPath := ""
	if len(lib.APIs) > 0 && lib.APIs[0].Path != "" {
		apiPath = lib.APIs[0].Path
	} else if lib.Name != "" {
		apiPath = strings.ReplaceAll(lib.Name, "-", "/")
	}
	defaultOut := "generated"
	if defaults != nil && defaults.Output != "" {
		defaultOut = defaults.Output
	}
	return DefaultOutput(apiPath, defaultOut)
}

func libraryPackageDirectory(lib *config.Library, defaults *config.Default) string {
	return PackageDirectory(libraryOutput(lib, defaults))
}

// SplitRepoName derives the split repository name for a library directory.
// For example, packages/wkt -> swift-wkt, generated/google-rpc -> swift-google-rpc.
func SplitRepoName(libDir string) string {
	if libDir == "" {
		return ""
	}
	base := filepath.Base(libDir)
	if base == "." || base == "/" {
		return ""
	}
	if !strings.HasPrefix(base, "swift-") {
		return "swift-" + base
	}
	return base
}

func matchLibrary(targets []string, lib *config.Library, pkgDir string) bool {
	cleanDir := filepath.Clean(pkgDir)
	baseDir := filepath.Base(cleanDir)
	repoName := SplitRepoName(pkgDir)
	for _, target := range targets {
		cleanTarget := filepath.Clean(target)
		if target == lib.Name || cleanTarget == cleanDir || cleanTarget == baseDir || target == pkgDir || target == lib.Output || target == repoName {
			return true
		}
		if lib.Swift != nil && (target == lib.Swift.LibraryNameOverride || target == lib.Swift.PackageNameOverride) {
			return true
		}
	}
	return false
}
