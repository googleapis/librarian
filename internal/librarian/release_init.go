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

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
)

const (
	KeyClNum = "PiperOrigin-RevId"
)

// cmdInit is the command for the `release init` subcommand.
var cmdInit = &cli.Command{
	Short:     "init initiates a release by creating a release pull request.",
	UsageLine: "librarian release init [arguments]",
	Long: `The release init command is the primary entry point for initiating a release.
It orchestrates the process of parsing commits, determining new versions, generating
a changelog, and creating a release pull request.`,
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
	state           *config.LibrarianState
	librarianConfig *config.LibrarianConfig
	ghClient        GitHubClient
	containerClient ContainerClient
	workRoot        string
	partialRepo     string
	image           string
}

func newInitRunner(cfg *config.Config) (*initRunner, error) {
	runner, err := newCommandRunner(cfg, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create init runner: %w", err)
	}
	return &initRunner{
		cfg:             runner.cfg,
		workRoot:        runner.workRoot,
		repo:            runner.repo,
		partialRepo:     filepath.Join(runner.workRoot, "release-init"),
		state:           runner.state,
		librarianConfig: runner.librarianConfig,
		image:           runner.image,
		ghClient:        runner.ghClient,
		containerClient: runner.containerClient,
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

	commitInfo := &commitInfo{
		cfg:           r.cfg,
		state:         r.state,
		repo:          r.repo,
		ghClient:      r.ghClient,
		commitMessage: "chore: create a release",
		prType:        release,
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
			if err := updateLibrary(r.repo, library, r.cfg.LibraryVersion); err != nil {
				return err
			}
			if err := copyLibrary(dst, src, library); err != nil {
				return err
			}

			break
		}

		// Update all libraries.
		if err := updateLibrary(r.repo, library, r.cfg.LibraryVersion); err != nil {
			return err
		}
		if err := copyLibrary(dst, src, library); err != nil {
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
// 1. Get the library's commit history in the given git repository.
//
// 2. Override the library version if libraryVersion is not empty.
//
// 3. Set the library's release trigger to true.
func updateLibrary(repo gitrepo.Repository, library *config.LibraryState, libraryVersion string) error {
	commits, err := GetConventionalCommitsSinceLastRelease(repo, library)
	if err != nil {
		return fmt.Errorf("failed to fetch conventional commits for library, %s: %w", library.ID, err)
	}

	library.Changes = coerceLibraryChanges(commits)
	if len(library.Changes) == 0 {
		slog.Info("Skip releasing library since no eligible change is found", "library", library.ID)
		return nil
	}

	nextVersion, err := NextVersion(commits, library.Version, libraryVersion)
	if err != nil {
		return err
	}

	library.Version = nextVersion
	library.ReleaseTriggered = true

	return nil
}

func coerceLibraryChanges(commits []*conventionalcommits.ConventionalCommit) []*config.Change {
	changes := make([]*config.Change, 0)
	for _, commit := range commits {
		clNum := ""
		if cl, ok := commit.Footers[KeyClNum]; ok {
			clNum = cl
		}

		changeType := getChangeType(commit)
		changes = append(changes, &config.Change{
			Type:       changeType,
			Subject:    commit.Description,
			Body:       commit.Body,
			ClNum:      clNum,
			CommitHash: commit.SHA,
		})
	}

	return changes
}

// getChangeType gets the type of the commit, adding an escalation mark (!) if
// it is a breaking change.
func getChangeType(commit *conventionalcommits.ConventionalCommit) string {
	changeType := commit.Type
	if commit.IsBreaking {
		changeType = changeType + "!"
	}

	return changeType
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
