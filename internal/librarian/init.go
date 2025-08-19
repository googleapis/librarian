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

	addFlagAPISource(fs, cfg)
	addFlagPush(fs, cfg)
	addFlagImage(fs, cfg)
	addFlagLibrary(fs, cfg)
	addFlagLibraryVersion(fs, cfg)
	addFlagRepo(fs, cfg)
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
	image           string
}

func newInitRunner(cfg *config.Config) (*initRunner, error) {
	runner, err := newCommandRunner(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create init runner: %w", err)
	}
	return &initRunner{
		cfg:             runner.cfg,
		workRoot:        runner.workRoot,
		repo:            runner.repo,
		sourceRepo:      runner.sourceRepo,
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
	return r.runInitCommand(ctx, outputDir)
}

func (r *initRunner) runInitCommand(ctx context.Context, outputDir string) error {
	setReleaseTrigger(r.state, r.cfg.Library, r.cfg.LibraryVersion, true)
	initRequest := &docker.ReleaseInitRequest{
		Cfg:            r.cfg,
		State:          r.state,
		LibraryID:      r.cfg.Library,
		LibraryVersion: r.cfg.LibraryVersion,
		Output:         outputDir,
	}
	return r.containerClient.ReleaseInit(ctx, initRequest)
}

// setReleaseTrigger sets the release trigger for the library with a non-empty
// libraryID and override the version, if provided; or for all libraries if
// the libraryID is empty.
func setReleaseTrigger(state *config.LibrarianState, libraryID, libraryVersion string, trigger bool) {
	for _, library := range state.Libraries {
		if libraryID != "" {
			// Only set the trigger for one library.
			if libraryID != library.ID {
				// Set other libraries with an opposite value.
				library.ReleaseTriggered = !trigger
				continue
			}

			if libraryVersion != "" {
				library.Version = libraryVersion
			}
			library.ReleaseTriggered = trigger

			break
		}
		// Set the trigger for all libraries.
		library.ReleaseTriggered = trigger
	}
}
