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
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
)

type updateImageRunner struct {
	branch          string
	containerClient ContainerClient
	ghClient        GitHubClient
	hostMount       string
	librarianConfig *config.LibrarianConfig
	repo            gitrepo.Repository
	sourceRepo      gitrepo.Repository
	state           *config.LibrarianState
	generate        bool
	build           bool
	push            bool
	commit          bool
	image           string
	workRoot        string
}

func newUpdateImageRunner(cfg *config.Config) (*updateImageRunner, error) {
	runner, err := newCommandRunner(cfg)
	if err != nil {
		return nil, err
	}
	return &updateImageRunner{
		branch:          cfg.Branch,
		containerClient: runner.containerClient,
		ghClient:        runner.ghClient,
		hostMount:       cfg.HostMount,
		librarianConfig: runner.librarianConfig,
		repo:            runner.repo,
		sourceRepo:      runner.sourceRepo,
		state:           runner.state,
		generate:        true,
		build:           cfg.Build,
		commit:          cfg.Commit,
		push:            cfg.Push,
		image:           cfg.Image,
		workRoot:        runner.workRoot,
	}, nil
}

func (r *updateImageRunner) run(ctx context.Context) error {
	// Update `image` entry in state.yaml
	if r.image == "" {
		slog.Info("No image found, looking up latest")
		latestImage, err := findLatestImage(r.state.Image)
		if err != nil {
			slog.Error("Unable to determine latest image to use", "image", r.state.Image)
			return err
		}
		r.image = latestImage
	}
	r.state.Image = r.image

	if err := saveLibrarianState(r.repo.GetDir(), r.state); err != nil {
		return err
	}

	commitMessage := fmt.Sprintf("chore: update image to %s", r.image)
	committed, err := commit(r.repo, commitMessage)
	if err != nil {
		return err
	}
	if !committed {
		slog.Info("No update to the image, aborting.")
		return nil
	}

	// For each library, run generation at the previous commit
	failedGenerations := make([]*config.LibraryState, 0)
	outputDir := filepath.Join(r.workRoot, "output")
	for _, libraryState := range r.state.Libraries {
		err := r.regenerateSingleLibrary(ctx, libraryState, outputDir)
		if err != nil {
			slog.Error(err.Error(), "library", libraryState.ID, "commit", libraryState.LastGeneratedCommit)
			failedGenerations = append(failedGenerations, libraryState)
			continue
		}
	}
	slog.Warn("failed generations", slog.Int("num", len(failedGenerations)))

	return nil
}

func findLatestImage(currentImage string) (string, error) {
	slog.Warn("findLatestImage is not yet implemented.")
	return currentImage, nil
}

func (r *updateImageRunner) regenerateSingleLibrary(ctx context.Context, libraryState *config.LibraryState, outputDir string) error {
	slog.Info("checking out apiSource", "commit", libraryState.LastGeneratedCommit)
	if err := r.sourceRepo.Checkout(libraryState.LastGeneratedCommit); err != nil {
		return fmt.Errorf("error checking out from sourceRepo %w", err)
	}

	if r.generate {
		if err := generateSingleLibrary(ctx, r.containerClient, r.state, libraryState, r.repo, r.sourceRepo, outputDir); err != nil {
			slog.Error("failed to regenerate a single library", "ID", libraryState.ID)
			return err
		}
	}

	if r.build {
		if err := buildSingleLibrary(ctx, r.containerClient, r.state, libraryState, r.repo); err != nil {
			slog.Error("failed to build a single library", "ID", libraryState.ID)
			return err
		}
	}

	return nil
}
