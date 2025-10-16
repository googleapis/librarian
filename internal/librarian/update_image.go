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
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/images"
)

const (
	updateImageCmdName = "update-image"
)

type updateImageRunner struct {
	branch          string
	containerClient ContainerClient
	imagesClient    ImageRegistryClient
	ghClient        GitHubClient
	hostMount       string
	librarianConfig *config.LibrarianConfig
	repo            gitrepo.Repository
	sourceRepo      gitrepo.Repository
	state           *config.LibrarianState
	build           bool
	push            bool
	commit          bool
	image           string
	workRoot        string
}

// ImageRegistryClient is an abstraction around interacting with image.
type ImageRegistryClient interface {
	FindLatest(ctx context.Context, imageName string) (string, error)
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
		build:           cfg.Build,
		commit:          cfg.Commit,
		push:            cfg.Push,
		image:           cfg.Image,
		workRoot:        runner.workRoot,
	}, nil
}

func (r *updateImageRunner) run(ctx context.Context) error {
	imagesClient := r.imagesClient
	if imagesClient == nil {
		slog.Info("no imagesClient provided, defaulting to ArtifactRegistry implementation")
		client, err := images.NewArtifactRegistryClient(ctx)
		if err != nil {
			return err
		}
		defer client.Close()
		imagesClient = client
	}

	// Update `image` entry in state.yaml
	if r.image == "" {
		slog.Info("No image found, looking up latest")
		latestImage, err := imagesClient.FindLatest(ctx, r.state.Image)
		if err != nil {
			slog.Error("Unable to determine latest image to use", "image", r.state.Image)
			return err
		}
		r.image = latestImage
	}

	if r.image == r.state.Image {
		slog.Info("No update to the image, aborting.")
		return nil
	}

	r.state.Image = r.image

	if err := saveLibrarianState(r.repo.GetDir(), r.state); err != nil {
		return err
	}

	// For each library, run generation at the previous commit
	var failedGenerations []*config.LibraryState
	var successfulGenerations []*config.LibraryState
	outputDir := filepath.Join(r.workRoot, "output")
	for _, libraryState := range r.state.Libraries {
		err := r.regenerateSingleLibrary(ctx, libraryState, outputDir)
		if err != nil {
			slog.Error(err.Error(), "library", libraryState.ID, "commit", libraryState.LastGeneratedCommit)
			failedGenerations = append(failedGenerations, libraryState)
			continue
		} else {
			successfulGenerations = append(successfulGenerations, libraryState)
		}
	}
	if len(failedGenerations) > 0 {
		slog.Warn("failed generations", slog.Int("num", len(failedGenerations)))
	}
	slog.Info("successful generations", slog.Int("num", len(successfulGenerations)))

	prBodyBuilder := func() (string, error) {
		return formatUpdateImagePRBody(r.image, failedGenerations)
	}
	commitMessage := fmt.Sprintf("feat: update image to %s", r.image)
	// TODO(#2588): open PR as draft if there are failures
	return commitAndPush(ctx, &commitInfo{
		branch:            r.branch,
		commit:            r.commit,
		commitMessage:     commitMessage,
		prType:            pullRequestUpdateImage,
		ghClient:          r.ghClient,
		pullRequestLabels: []string{},
		push:              r.push,
		languageRepo:      r.repo,
		sourceRepo:        r.sourceRepo,
		state:             r.state,
		workRoot:          r.workRoot,
		failedGenerations: len(failedGenerations),
		prBodyBuilder:     prBodyBuilder,
	})
}

func (r *updateImageRunner) regenerateSingleLibrary(ctx context.Context, libraryState *config.LibraryState, outputDir string) error {
	if len(libraryState.APIs) == 0 {
		slog.Info("library has no APIs; skipping generation", "library", libraryState.ID)
		return nil
	}

	slog.Info("checking out apiSource", "commit", libraryState.LastGeneratedCommit)
	if err := r.sourceRepo.Checkout(libraryState.LastGeneratedCommit); err != nil {
		return fmt.Errorf("error checking out from sourceRepo %w", err)
	}

	if err := generateSingleLibrary(ctx, r.containerClient, r.state, libraryState, r.repo, r.sourceRepo, outputDir); err != nil {
		slog.Error("failed to regenerate a single library", "error", err, "ID", libraryState.ID)
		return err
	}

	if !r.build {
		slog.Info("build not specified, skipping build")
		return nil
	}
	if err := buildSingleLibrary(ctx, r.containerClient, r.state, libraryState, r.repo); err != nil {
		slog.Error("failed to build a single library", "error", err, "ID", libraryState.ID)
		return err
	}

	return nil
}

var updateImageTemplate = template.Must(template.New("updateImage").Parse(`feat: update image to {{.Image}}
{{ if .FailedLibraries }}
## Generation failed for
{{- range .FailedLibraries }}
- {{ . }}
{{- end -}}
{{- end }}
`))

type updateImagePRBody struct {
	Image           string
	FailedLibraries []string
}

func formatUpdateImagePRBody(image string, failedGenerations []*config.LibraryState) (string, error) {
	failedLibraries := make([]string, 0, len(failedGenerations))
	for _, failedGeneration := range failedGenerations {
		failedLibraries = append(failedLibraries, failedGeneration.ID)
	}
	data := &updateImagePRBody{
		Image:           image,
		FailedLibraries: failedLibraries,
	}
	var out bytes.Buffer
	if err := updateImageTemplate.Execute(&out, data); err != nil {
		return "", fmt.Errorf("error executing template %w", err)
	}
	return strings.TrimSpace(out.String()), nil
}
