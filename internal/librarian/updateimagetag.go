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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
)

var cmdUpdateImageTag = &cli.Command{
	Short:     "update-image-tag updates a language repo's image tag and regenerates APIs",
	UsageLine: "librarian update-image-tag -tag=<new tag> [flags]",
	Long: `Specify the the new tag, and optional flags to use non-default repositories, e.g. for testing.
A pull request will only be created if -push is specified, in which case the LIBRARIAN_GITHUB_TOKEN
environment variable must be populated with an access token which has write access to the
language repo in which the pull request will be created.

The update-image-tag command is used to change which tag for the language image is used
for language-specific operations. The most common reasons for doing this are if the code handling
language container commands has changed (e.g. to fix bugs) or because the generator has been updated
to a newer version.

After acquiring the API and language repositories, every library which has any API paths specified
and a last generated commit is regenerated - even if regeneration is otherwise blocked. The API repository
is checked out to the commit at which the library was last regenerated, so that the resulting pull request
*only* contains changes due to updating the image tag.

Regeneration uses the "generate-library" and "clean" language container commands (using the image with the
newly-specified tag), copying the code after the clean operation as normal. The libraries are *not* built
one at a time, however.

If all generation operations are successful, a single commit is created with all the generated code changes and
the state change to indicate the new tag.

After this, the "build-library" command is run, without specifying a library ID.
This means that all configured libraries in the language repository should be rebuilt and unit tested. This
is more efficient than building libraries after regeneration - and coincidentally also checks that libraries
which don't contain generated code still build with the new image tag.

A failure at any point makes the command fail: this command does not support partial success.
(If some libraries can't be regenerated or built with the new image tag, that should be addressed
before using the image for anything.)

If everything has succeeded, and if the -push flag has been specified, a pull request is created in the
language repository, containing the new commit. If the -push flag has not been specified,
the description of the pull request that would have been created is included in the
output of the command. Even if a pull request isn't created, the resulting commit will still be present
in the language repo.
`,
	Run: runUpdateImageTag,
}

func init() {
	cmdUpdateImageTag.Init()
	fs := cmdUpdateImageTag.Flags
	cfg := cmdUpdateImageTag.Config

	addFlagBranch(fs, cfg)
	addFlagPushConfig(fs, cfg)
	addFlagRepo(fs, cfg)
	addFlagProject(fs, cfg)
	addFlagSource(fs, cfg)
	addFlagTag(fs, cfg)
	addFlagWorkRoot(fs, cfg)
}

func runUpdateImageTag(ctx context.Context, cfg *config.Config) error {
	startTime, workRoot, languageRepo, pipelineConfig, pipelineState, containerConfig, err := createCommandStateForLanguage(cfg.WorkRoot, cfg.Repo, cfg.Image, cfg.Project, cfg.CI, cfg.UserUID, cfg.UserGID)
	if err != nil {
		return err
	}
	return updateImageTag(ctx, cfg, startTime, workRoot, languageRepo, pipelineConfig, pipelineState, containerConfig)
}

func updateImageTag(ctx context.Context, cfg *config.Config, startTime time.Time, workRoot string, languageRepo *gitrepo.Repository, pipelineConfig *statepb.PipelineConfig, pipelineState *statepb.PipelineState, containerConfig *docker.Docker) error {
	if err := validateRequiredFlag("tag", cfg.Tag); err != nil {
		return err
	}

	var apiRepo *gitrepo.Repository
	if cfg.Source == "" {
		var err error
		apiRepo, err = cloneGoogleapis(workRoot, cfg.CI)
		if err != nil {
			return err
		}
	} else {
		apiRoot, err := filepath.Abs(cfg.Source)
		slog.Info("Using apiRoot", "api_root", apiRoot)
		if err != nil {
			slog.Info("Error retrieving apiRoot", "err", err)
			return err
		}
		apiRepo, err = gitrepo.NewRepository(&gitrepo.RepositoryOptions{
			Dir: apiRoot,
			CI:  cfg.CI,
		})
		if err != nil {
			return err
		}
		clean, err := apiRepo.IsClean()
		if err != nil {
			return err
		}
		if !clean {
			return errors.New("api repo must be clean before updating the language image tag")
		}
	}

	outputDir := filepath.Join(workRoot, "output")
	if err := os.Mkdir(outputDir, 0755); err != nil {
		return err
	}
	slog.Info("Code will be generated", "dir", outputDir)

	if pipelineState.ImageTag == cfg.Tag {
		return errors.New("specified tag is already in language repo state")
	}
	// Derive the new image to use, and save it in the context.
	pipelineState.ImageTag = cfg.Tag
	image, err := deriveImage(cfg.Image, pipelineState)
	if err != nil {
		return err
	}
	containerConfig.Image = image
	if err := savePipelineState(languageRepo, pipelineState); err != nil {
		return err
	}

	// Take a defensive copy of the generator input directory from the language repo.
	generatorInput := filepath.Join(workRoot, config.GeneratorInputDir)
	if err := os.CopyFS(generatorInput, os.DirFS(filepath.Join(languageRepo.Dir, config.GeneratorInputDir))); err != nil {
		return err
	}

	// Perform "generate, clean" on each library.
	for _, library := range pipelineState.Libraries {
		err := regenerateLibrary(ctx, cfg, apiRepo, generatorInput, outputDir, library, languageRepo, containerConfig)
		if err != nil {
			return err
		}
	}

	// Commit any changes
	commitMsg := fmt.Sprintf("chore: update generation image tag to %s", cfg.Tag)
	if err := commitAll(languageRepo, commitMsg,
		cfg.PushConfig); err != nil {
		return err
	}

	// Build everything at the end. (This is more efficient than building each library with a separate container invocation.)
	slog.Info("Building all libraries.")
	if err := containerConfig.BuildLibrary(ctx, cfg, languageRepo.Dir, ""); err != nil {
		return err
	}

	// The PullRequestContent for update-image-tag is slightly different to others, but we
	// can massage it into a similar state.
	prContent := new(PullRequestContent)
	addSuccessToPullRequest(prContent, "Regenerated all libraries with new image tag.")
	_, err = createPullRequest(ctx, prContent, "chore: update generation image tag", "", "update-image-tag", cfg.GitHubToken, cfg.Push, startTime, languageRepo, pipelineConfig)
	return err
}

func regenerateLibrary(ctx context.Context, cfg *config.Config, apiRepo *gitrepo.Repository, generatorInput string, outputRoot string, library *statepb.LibraryState, languageRepo *gitrepo.Repository, containerConfig *docker.Docker) error {
	if len(library.ApiPaths) == 0 {
		slog.Info("Skipping non-generated library", "id", library.Id)
		return nil
	}

	// TODO(https://github.com/googleapis/librarian/issues/341): handle "no last generated commit"
	if err := apiRepo.Checkout(library.LastGeneratedCommit); err != nil {
		return err
	}

	slog.Info("Generating library", "id", library.Id)

	// We create an output directory separately for each API.
	outputDir := filepath.Join(outputRoot, library.Id)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	if err := containerConfig.GenerateLibrary(ctx, cfg, apiRepo.Dir, outputDir, generatorInput, library.Id); err != nil {
		return err
	}
	if err := containerConfig.Clean(ctx, cfg, languageRepo.Dir, library.Id); err != nil {
		return err
	}
	if err := os.CopyFS(languageRepo.Dir, os.DirFS(outputDir)); err != nil {
		return err
	}
	if err := apiRepo.CleanWorkingTree(); err != nil {
		return err
	}
	return nil
}
