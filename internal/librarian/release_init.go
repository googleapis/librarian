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

	"github.com/googleapis/librarian/internal/conventionalcommits"
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/semver"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
)

type initRunner struct {
	branch          string
	commit          bool
	containerClient ContainerClient
	ghClient        GitHubClient
	image           string
	librarianConfig *config.LibrarianConfig
	library         string
	libraryVersion  string
	partialRepo     string
	push            bool
	repo            gitrepo.Repository
	sourceRepo      gitrepo.Repository
	state           *config.LibrarianState
	workRoot        string
}

func newInitRunner(cfg *config.Config) (*initRunner, error) {
	runner, err := newCommandRunner(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create init runner: %w", err)
	}
	return &initRunner{
		branch:          cfg.Branch,
		commit:          cfg.Commit,
		containerClient: runner.containerClient,
		ghClient:        runner.ghClient,
		image:           runner.image,
		librarianConfig: runner.librarianConfig,
		library:         cfg.Library,
		libraryVersion:  cfg.LibraryVersion,
		partialRepo:     filepath.Join(runner.workRoot, "release-init"),
		push:            cfg.Push,
		repo:            runner.repo,
		sourceRepo:      runner.sourceRepo,
		state:           runner.state,
		workRoot:        runner.workRoot,
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
		branch:         r.branch,
		commit:         r.commit,
		commitMessage:  "chore: create a release",
		ghClient:       r.ghClient,
		library:        r.library,
		libraryVersion: r.libraryVersion,
		prType:         release,
		// Newly created PRs from the `release init` command should have a
		// `release:pending` GitHub tab to be tracked for release.
		pullRequestLabels: []string{"release:pending"},
		push:              r.push,
		repo:              r.repo,
		sourceRepo:        r.sourceRepo,
		state:             r.state,
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
		if r.library != "" {
			if r.library != library.ID {
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
		Branch:          r.branch,
		Commit:          r.commit,
		LibrarianConfig: r.librarianConfig,
		LibraryID:       r.library,
		LibraryVersion:  r.libraryVersion,
		Output:          outputDir,
		PartialRepoDir:  dst,
		Push:            r.push,
		State:           r.state,
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
		if r.library != "" {
			if r.library != library.ID {
				continue
			}
			// Only copy one library to repository.
			if err := copyLibraryFiles(r.state, r.repo.GetDir(), r.library, outputDir); err != nil {
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

	nextVersion, err := r.determineNextVersion(commits, library.Version, library.ID)
	if err != nil {
		return err
	}

	library.Version = nextVersion
	library.ReleaseTriggered = true

	return nil
}

func (r *initRunner) determineNextVersion(commits []*conventionalcommits.ConventionalCommit, currentVersion string, libraryID string) (string, error) {
	// If library version explicitly passed to CLI, use it
	if r.libraryVersion != "" {
		slog.Info("Library version override specified", "currentVersion", currentVersion, "version", r.libraryVersion)
		newVersion := semver.MaxVersion(currentVersion, r.libraryVersion)
		if newVersion == r.libraryVersion {
			return newVersion, nil
		} else {
			slog.Warn("Specified version is not higher than the current version, ignoring override.")
		}
	}

	nextVersionFromCommits, err := NextVersion(commits, currentVersion)
	if err != nil {
		return "", err
	}

	if r.librarianConfig == nil {
		slog.Info("No librarian config")
		return nextVersionFromCommits, nil
	}

	// Look for next_version override from config.yaml
	libraryConfig := r.librarianConfig.LibraryConfigFor(libraryID)
	slog.Info("Looking up library config", "library", libraryID, slog.Any("config", libraryConfig))
	if libraryConfig == nil || libraryConfig.NextVersion == "" {
		return nextVersionFromCommits, nil
	}

	// Compare versions and pick latest
	return semver.MaxVersion(nextVersionFromCommits, libraryConfig.NextVersion), nil
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
