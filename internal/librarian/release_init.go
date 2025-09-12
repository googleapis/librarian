// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package librarian

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/docker"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
)

// cmdInit is the command for the `release init` subcommand.
var cmdInit = &cli.Command{
	Short:     "init initiates a release by creating a release pull request.",
	UsageLine: "librarian release init [flags]",
	Long: `The 'release init' command is the primary entry point for initiating
a new release. It automates the creation of a release pull request by parsing
conventional commits, determining the next semantic version for each library,
and generating a changelog. Librarian is environment aware and will check if the
current directory is the root of a librarian repository. If you are not
executing in such a directory the '--repo' flag must be provided.

This command scans the git history since the last release, identifies changes
(feat, fix, BREAKING CHANGE), and calculates the appropriate version bump
according to semver rules. It then delegates all language-specific file
modifications, such as updating a CHANGELOG.md or bumping the version in a pom.xml, 
to the configured language-specific container.

By default, 'release init' leaves the changes in your local working directory
for inspection. Use the '--push' flag to automatically commit the changes to
a new branch and create a pull request on GitHub. The '--commit' flag may be
used to create a local commit without creating a pull request; this flag is
ignored if '--push' is also specified.

Examples:
  # Create a release PR for all libraries with pending changes.
  librarian release init --push

  # Create a release PR for a single library.
  librarian release init --library=secretmanager --push

  # Manually specify a version for a single library, overriding the calculation.
  librarian release init --library=secretmanager --library-version=2.0.0 --push`,
	Run: func(ctx context.Context, cfg *config.Config) error {
		runner, err := newInitRunner(cfg)
		if err != nil {
			return err
		}
		return runner.run(ctx)
	},
}

func init() {
	cmdInit.Init()
	fs := cmdInit.Flags
	cfg := cmdInit.Config

	addFlagCommit(fs, cfg)
	addFlagPush(fs, cfg)
	addFlagImage(fs, cfg)
	addFlagLibrary(fs, cfg)
	addFlagLibraryVersion(fs, cfg)
	addFlagRepo(fs, cfg)
	addFlagBranch(fs, cfg)
	addFlagWorkRoot(fs, cfg)
}

type initRunner struct {
	cfg             *config.Config
	repo            gitrepo.Repository
	sourceRepo      gitrepo.Repository
	state           *config.LibrarianState
	librarianConfig *config.LibrarianConfig
	ghClient        GitHubClient
	containerClient ContainerClient
	workRoot        string
	partialRepo     string
	image           string
}

func newInitRunner(cfg *config.Config) (*initRunner, error) {
	runner, err := newCommandRunner(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create init runner: %w", err)
	}
	return &initRunner{
		cfg:             runner.cfg,
		repo:            runner.repo,
		sourceRepo:      runner.sourceRepo,
		state:           runner.state,
		librarianConfig: runner.librarianConfig,
		ghClient:        runner.ghClient,
		containerClient: runner.containerClient,
		workRoot:        runner.workRoot,
		partialRepo:     filepath.Join(runner.workRoot, "release-init"),
		image:           runner.image,
	}, nil
}

func (r *initRunner) run(ctx context.Context) error {
	outputDir := filepath.Join(r.workRoot, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %s", outputDir)
	}
	slog.Info("Initiating a release", "dir", outputDir)
	if err := r.runInitCommand(ctx, outputDir); err != nil {
		return err
	}

	if err := saveLibrarianState(r.repo.GetDir(), r.state); err != nil {
		return err
	}

	commitInfo := &commitInfo{
		cfg:           r.cfg,
		state:         r.state,
		repo:          r.repo,
		sourceRepo:    r.sourceRepo,
		ghClient:      r.ghClient,
		commitMessage: "chore: create a release",
		prType:        release,
		// Newly created PRs from the `release init` command should have a
		// `release:pending` GitHub tab to be tracked for release.
		pullRequestLabels: []string{"release:pending"},
	}
	if err := commitAndPush(ctx, commitInfo); err != nil {
		return fmt.Errorf("failed to commit and push: %w", err)
	}

	return nil
}

func (r *initRunner) runInitCommand(ctx context.Context, outputDir string) error {
	dst := r.partialRepo
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("failed to make directory: %w", err)
	}

	src := r.repo.GetDir()
	for _, library := range r.state.Libraries {
		if r.cfg.Library != "" {
			if r.cfg.Library != library.ID {
				continue
			}

			// Only update one library with the given library ID.
			if err := r.updateLibrary(library); err != nil {
				return err
			}
			if err := copyLibraryFiles(r.state, dst, library.ID, src); err != nil {
				return err
			}

			break
		}

		// Update all libraries.
		if err := r.updateLibrary(library); err != nil {
			return err
		}
		if err := copyLibraryFiles(r.state, dst, library.ID, src); err != nil {
			return err
		}
	}

	if err := copyLibrarianDir(dst, src); err != nil {
		return fmt.Errorf("failed to copy librarian dir from %s to %s: %w", src, dst, err)
	}

	if err := copyGlobalAllowlist(r.librarianConfig, dst, src, true); err != nil {
		return fmt.Errorf("failed to copy global allowlist  from %s to %s: %w", src, dst, err)
	}

	initRequest := &docker.ReleaseInitRequest{
		Cfg:             r.cfg,
		State:           r.state,
		LibrarianConfig: r.librarianConfig,
		LibraryID:       r.cfg.Library,
		LibraryVersion:  r.cfg.LibraryVersion,
		Output:          outputDir,
		PartialRepoDir:  dst,
	}

	if err := r.containerClient.ReleaseInit(ctx, initRequest); err != nil {
		return err
	}

	// Read the response file.
	if _, err := readLibraryState(
		filepath.Join(initRequest.PartialRepoDir, config.LibrarianDir, config.ReleaseInitResponse)); err != nil {
		return err
	}

	for _, library := range r.state.Libraries {
		if r.cfg.Library != "" {
			if r.cfg.Library != library.ID {
				continue
			}
			// Only copy one library to repository.
			if err := copyLibraryFiles(r.state, r.repo.GetDir(), r.cfg.Library, outputDir); err != nil {
				return err
			}

			break
		}

		// Copy all libraries to repository.
		if err := copyLibraryFiles(r.state, r.repo.GetDir(), library.ID, outputDir); err != nil {
			return err
		}
	}

	return copyGlobalAllowlist(r.librarianConfig, r.repo.GetDir(), outputDir, false)
}

// updateLibrary updates the given library in the following way:
//
// 1. Update the library's previous version.
//
// 2. Get the library's commit history in the given git repository.
//
// 3. Override the library version if libraryVersion is not empty.
//
// 4. Set the library's release trigger to true.
func (r *initRunner) updateLibrary(library *config.LibraryState) error {
	// Update the previous version, we need this value when creating release note.
	library.PreviousVersion = library.Version
	commits, err := GetConventionalCommitsSinceLastRelease(r.repo, library)
	if err != nil {
		return fmt.Errorf("failed to fetch conventional commits for library, %s: %w", library.ID, err)
	}

	library.Changes = commits
	if len(library.Changes) == 0 {
		slog.Info("Skip releasing library since no eligible change is found", "library", library.ID)
		return nil
	}

	nextVersion, err := NextVersion(commits, library.Version, r.cfg.LibraryVersion)
	if err != nil {
		return err
	}

	library.Version = nextVersion
	library.ReleaseTriggered = true

	return nil
}

// copyGlobalAllowlist copies files in the global file allowlist excluding
//
//	read-only files and copies global files from src.
func copyGlobalAllowlist(cfg *config.LibrarianConfig, dst, src string, copyReadOnly bool) error {
	if cfg == nil {
		slog.Info("librarian config is not setup, skip copying global allowlist")
		return nil
	}
	slog.Info("Copying global allowlist files", "destination", dst, "source", src)
	for _, globalFile := range cfg.GlobalFilesAllowlist {
		if globalFile.Permissions == config.PermissionReadOnly && !copyReadOnly {
			slog.Debug("skipping read-only file", "path", globalFile.Path)
			continue
		}
		srcPath := filepath.Join(src, globalFile.Path)
		dstPath := filepath.Join(dst, globalFile.Path)
		if err := copyFile(dstPath, srcPath); err != nil {
			return fmt.Errorf("failed to copy global file %s from %s: %w", dstPath, srcPath, err)
		}
	}
	return nil
}

func copyLibrarianDir(dst, src string) error {
	return os.CopyFS(
		filepath.Join(dst, config.LibrarianDir),
		os.DirFS(filepath.Join(src, config.LibrarianDir)))
}
