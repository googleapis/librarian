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
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
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

// Publish iterates through every library in librarian.yaml, checks if the version
// has already been published to the target split repository, and performs a repo split
// and push if needed.
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

	for _, lib := range params.Config.Libraries {
		if lib.SkipRelease || lib.Version == "" {
			continue
		}

		libDir := packageDirectory(lib, params.Config.Default)
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

		tagExists, err := git.RemoteTagExists(ctx, gitExe, remoteURL, tag)
		if err != nil {
			if params.DryRunKeepGoing {
				slog.Error("failed to check remote tags, but continuing due to --keep-going", "library", lib.Name, "remote", remoteURL, "error", err)
				continue
			}
			return fmt.Errorf("failed to check remote tags for %s on %s: %w", lib.Name, remoteURL, err)
		}

		if tagExists {
			slog.Info("version already tagged on remote repository, skipping", "library", lib.Name, "version", tag, "remote", remoteURL)
			continue
		}

		slog.Info("splitting repository for library", "library", lib.Name, "path", libDir, "version", tag, "remote", remoteURL)

		splitSHA, err := Split(ctx, SplitParams{
			TargetDir: libDir,
			Origin:    origin,
			RootFiles: rootFiles,
			GitExe:    gitExe,
		})
		if err != nil {
			if params.DryRunKeepGoing {
				slog.Error("failed to split library, but continuing due to --keep-going", "library", lib.Name, "error", err)
				continue
			}
			return fmt.Errorf("failed to split %s: %w", lib.Name, err)
		}

		if params.DryRun || params.DryRunKeepGoing {
			slog.Info("[DRY-RUN] Would push to remote", "library", lib.Name, "remote", remoteURL, "branch", remoteBranch, "sha", splitSHA, "tag", tag)
			continue
		}

		if err := git.PushBranch(ctx, gitExe, remoteURL, splitSHA, remoteBranch, true); err != nil {
			if params.DryRunKeepGoing {
				slog.Error("failed to push branch, but continuing due to --keep-going", "library", lib.Name, "remote", remoteURL, "error", err)
				continue
			}
			return fmt.Errorf("failed to push branch for %s to %s: %w", lib.Name, remoteURL, err)
		}

		if err := git.PushRefToTag(ctx, gitExe, remoteURL, splitSHA, tag, true); err != nil {
			if params.DryRunKeepGoing {
				slog.Error("failed to push tag, but continuing due to --keep-going", "library", lib.Name, "remote", remoteURL, "error", err)
				continue
			}
			return fmt.Errorf("failed to push tag %s for %s to %s: %w", tag, lib.Name, remoteURL, err)
		}

		slog.Info("successfully published library", "library", lib.Name, "version", tag, "remote", remoteURL)
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
	}
	defaultOut := ""
	if defaults != nil {
		defaultOut = defaults.Output
	}
	return DefaultOutput(apiPath, defaultOut)
}

func packageDirectory(lib *config.Library, defaults *config.Default) string {
	dir := libraryOutput(lib, defaults)
	if dir == "" {
		return ""
	}
	current := dir
	for current != "." && current != "/" && current != "" {
		if _, err := os.Stat(filepath.Join(current, "Package.swift")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return dir
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
