// Copyright 2025 Google LLC
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

package legacylibrarian

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacygitrepo"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyimages"
)

const (
	updateImageCmdName = "update-image"
)

type updateImageRunner struct {
	branch                 string
	containerClient        ContainerClient
	imagesClient           ImageRegistryClient
	ghClient               GitHubClient
	librarianConfig        *legacyconfig.LibrarianConfig
	repo                   legacygitrepo.Repository
	sourceRepo             legacygitrepo.Repository
	state                  *legacyconfig.LibrarianState
	build                  bool
	push                   bool
	commit                 bool
	image                  string
	workRoot               string
	test                   bool
	libraryToTest          string
	checkUnexpectedChanges bool
}

// ImageRegistryClient is an abstraction around interacting with image.
type ImageRegistryClient interface {
	FindLatest(ctx context.Context, imageName string) (string, error)
}

func newUpdateImageRunner(cfg *legacyconfig.Config) (*updateImageRunner, error) {
	runner, err := newCommandRunner(cfg)
	if err != nil {
		return nil, err
	}
	return &updateImageRunner{
		branch:                 cfg.Branch,
		containerClient:        runner.containerClient,
		ghClient:               runner.ghClient,
		librarianConfig:        runner.librarianConfig,
		repo:                   runner.repo,
		sourceRepo:             runner.sourceRepo,
		state:                  runner.state,
		build:                  cfg.Build,
		commit:                 cfg.Commit,
		push:                   cfg.Push,
		image:                  cfg.Image,
		workRoot:               runner.workRoot,
		test:                   cfg.Test,
		libraryToTest:          cfg.LibraryToTest,
		checkUnexpectedChanges: cfg.CheckUnexpectedChanges,
	}, nil
}

func (r *updateImageRunner) run(ctx context.Context) error {
	imagesClient := r.imagesClient
	if imagesClient == nil {
		slog.Info("no imagesClient provided, defaulting to ArtifactRegistry implementation")
		client, err := legacyimages.NewArtifactRegistryClient(ctx)
		if err != nil {
			return err
		}
		defer client.Close()
		imagesClient = client
	}

	// Update `image` entry in state.yaml
	if r.image == "" {
		slog.Info("no image found, looking up latest")
		latestImage, err := imagesClient.FindLatest(ctx, r.state.Image)
		if err != nil {
			slog.Error("unable to determine latest image to use", "image", r.state.Image)
			return err
		}
		r.image = latestImage
	}

	if r.image == r.state.Image {
		slog.Info("no update to the image; assuming diagnostic run")
	}

	r.state.Image = r.image

	if err := saveLibrarianState(r.repo.GetDir(), r.state); err != nil {
		return err
	}

	// For each library, run generation at the previous commit
	var failedGenerations []*legacyconfig.LibraryState
	var successfulGenerations []*legacyconfig.LibraryState
	var skippedGenerationsCount int
	sourceHead, err := r.sourceRepo.HeadHash()
	if err != nil {
		return err
	}
	outputDir := filepath.Join(r.workRoot, "output")
	timings := map[string]time.Duration{}
	for _, libraryState := range r.state.Libraries {
		if r.librarianConfig.IsGenerationBlocked(libraryState.ID) {
			slog.Debug("skipping generation for library due to generate_blocked", "library", libraryState.ID)
			skippedGenerationsCount++
			continue
		}
		startTime := time.Now()
		err := r.regenerateSingleLibrary(ctx, libraryState, outputDir)
		if err != nil {
			slog.Error(err.Error(), "library", libraryState.ID, "commit", libraryState.LastGeneratedCommit)
			failedGenerations = append(failedGenerations, libraryState)
			continue
		} else {
			successfulGenerations = append(successfulGenerations, libraryState)
		}
		timings[libraryState.ID] = time.Since(startTime)
	}
	if len(failedGenerations) > 0 {
		slog.Warn("failed generations", "num", len(failedGenerations))
	}
	slog.Info(
		"generation statistics",
		"all", len(r.state.Libraries),
		"successes", len(successfulGenerations),
		"skipped", skippedGenerationsCount,
		"failures", len(failedGenerations))
	if err := writeTiming(r.workRoot, timings); err != nil {
		return err
	}

	// Restore api source repo
	if err := r.sourceRepo.Checkout(sourceHead); err != nil {
		slog.Error(err.Error(), "repository", r.sourceRepo, "HEAD", sourceHead)
	}
	if r.test {
		slog.Info("running container tests")
		testRunner := &testGenerateRunner{
			library:                r.libraryToTest,
			repo:                   r.repo,
			sourceRepo:             r.sourceRepo,
			state:                  r.state,
			librarianConfig:        r.librarianConfig,
			workRoot:               r.workRoot,
			containerClient:        r.containerClient,
			checkUnexpectedChanges: r.checkUnexpectedChanges,
			branchesToDelete:       []string{},
		}
		if err := runContainerGenerateTest(ctx, r.repo, sourceHead, testRunner); err != nil {
			return fmt.Errorf("container generate test failed: %w", err)
		}
	}
	prBodyBuilder := func() (string, error) {
		return formatUpdateImagePRBody(r.image, failedGenerations)
	}
	commitMessage := fmt.Sprintf("feat: update image to %s", r.image)
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
		isDraft:           len(failedGenerations) > 0,
	})
}

func (r *updateImageRunner) regenerateSingleLibrary(ctx context.Context, libraryState *legacyconfig.LibraryState, outputDir string) error {
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

// runContainerGenerateTest creates a temporary commit to ensure a clean
// repo state for the test runner, runs the tests, and then soft-resets
// the commit to leave the working directory in its original dirty state
// for the final commit operation.
func runContainerGenerateTest(ctx context.Context, repo legacygitrepo.Repository, sourceHead string, testRunner *testGenerateRunner) error {
	slog.Debug("creating temporary commit for testing")
	committed := true
	if err := repo.AddAll(); err != nil {
		return fmt.Errorf("failed to stage changes for temporary commit: %w", err)
	}

	// Commit the generated changes so the repo is clean for the test runner.
	if err := repo.Commit("chore: temporary commit for update-image test"); err != nil {
		if !errors.Is(err, legacygitrepo.ErrNoModificationsToCommit) {
			return fmt.Errorf("failed to create temporary commit for test: %w", err)
		}
		slog.Debug("no changes to commit for test, proceeding without temporary commit")
		committed = false
	}

	if err := testRunner.runTests(ctx, sourceHead); err != nil {
		// If tests fail, leave the temporary commit in place for diagnostics and return the error.
		return fmt.Errorf("failure in container generate test: %w", err)
	}

	// If tests pass and temporary commit was made, reset it to restore the dirty state for the final commit.
	if committed {
		slog.Debug("tests passed, resetting temporary commit")
		if err := repo.ResetSoft("HEAD~1"); err != nil {
			return fmt.Errorf("failed to reset temporary commit after successful test: %w", err)
		}
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

func formatUpdateImagePRBody(image string, failedGenerations []*legacyconfig.LibraryState) (string, error) {
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
