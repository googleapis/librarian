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
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

var cmdGenerate = &cli.Command{
	Short:     "generate generates client library code for a single API",
	UsageLine: "librarian generate -source=<api-root> -api=<api-path> [flags]",
	Long: `Specify the API repository root and the path within it for the API to generate.
Optional flags can be specified to use a non-default language repository, and to indicate whether or not
to build the generated library.

First, the generation mode is determined by examining the language repository (remotely if
a local clone has not been specified). The Librarian state for the repository is examined to see if the
specified API path is already configured for a library in the repository. If it is, the refined generation
mode is used. Otherwise the raw generation mode is used. These are described separately below.

*Refined generation* is intended to give an accurate result for how an existing library would change when
generated with the new API specification. Generation for this library might include pregeneration or postgeneration
fixes, and the library may include handwritten code, unit tests and integration tests.

The process for refined generation requires the language repo to be cloned (if a local clone hasn't been
specified). Generation then proceeds by executing the following language container commands:
- "generate-library" to generate the source code for the library into an empty directory
- "clean" to clean any previously-generated source code from the language repository
- "build-library" (after copying the newly-generated code into place in the repository)

(The "clean" and "build-library" commands are skipped if the -build flag is not specified.)

The result of the generation is not committed anywhere, but the language repository will be left with any
working tree changes available to be checked. (Changes are not reverted.)


*Raw generation* is intended to give an early indication of whether an API can successfully be generated
as a library, and whether that library can then be built, without any additional information from the language
repo. The language repo is not cloned, but instead the following language container commands are executed:
- "generate-raw" to generate the source code for the library into an empty directory
- "build-raw" (if the -build flag is specified)

There is no "clean" operation or copying of the generated code in raw generation mode, because there is no
other source code to be preserved/cleaned. Instead, the "build-raw" command is provided with the same
output directory that was specified for the "generate-raw" command.
`,
	Run: func(ctx context.Context, cfg *config.Config) error {
		runner, err := newGenerateRunner(cfg)
		if err != nil {
			return err
		}
		return runner.run(ctx)
	},
}

func init() {
	cmdGenerate.Init()
	fs := cmdGenerate.Flags
	cfg := cmdGenerate.Config

	addFlagAPI(fs, cfg)
	addFlagBuild(fs, cfg)
	addFlagHostMount(fs, cfg)
	addFlagImage(fs, cfg)
	addFlagProject(fs, cfg)
	addFlagRepo(fs, cfg)
	addFlagSource(fs, cfg)
	addFlagWorkRoot(fs, cfg)
}

type generateRunner struct {
	cfg      *config.Config
	workRoot string
	repo     *gitrepo.Repository
	state    *config.PipelineState
	config   *config.PipelineConfig
	image    string
}

func newGenerateRunner(cfg *config.Config) (*generateRunner, error) {
	if err := validateRequiredFlag("api", cfg.API); err != nil {
		return nil, err
	}
	if err := validateRequiredFlag("source", cfg.Source); err != nil {
		return nil, err
	}
	workRoot, err := createWorkRoot(time.Now(), cfg.WorkRoot)
	if err != nil {
		return nil, err
	}
	repo, err := cloneOrOpenLanguageRepo(workRoot, cfg.Repo, cfg.CI)
	if err != nil {
		return nil, err
	}
	state, config, err := loadRepoStateAndConfig(repo)
	if err != nil {
		return nil, err
	}
	image, err := deriveImage(cfg.Image, state)
	if err != nil {
		return nil, err
	}
	return &generateRunner{
		cfg:      cfg,
		workRoot: workRoot,
		repo:     repo,
		state:    state,
		config:   config,
		image:    image,
	}, nil
}

func (r *generateRunner) run(ctx context.Context) error {
	outputDir := filepath.Join(r.workRoot, "output")
	if err := os.Mkdir(outputDir, 0755); err != nil {
		return err
	}
	slog.Info("Code will be generated", "dir", outputDir)

	_, err := r.detectIfLibraryConfigured(ctx)
	if err != nil {
		return err
	}
	containerConfig, err := docker.New(r.workRoot, r.image, r.cfg.Project, r.cfg.UserUID, r.cfg.UserGID, r.config)
	if err != nil {
		return err
	}

	// TODO(https://github.com/googleapis/librarian/issues/815)

	libraryID, err := r.runGenerateCommand(ctx, outputDir, r.state, containerConfig)
	if err != nil {
		return err
	}

	if r.cfg.Build {
		if libraryID != "" {
			slog.Info("Build requested in the context of refined generation; cleaning and copying code to the local language repo before building.")
			// TODO(https://github.com/googleapis/librarian/issues/775)
			if err := os.CopyFS(r.repo.Dir, os.DirFS(outputDir)); err != nil {
				return err
			}
			if err := containerConfig.Build(ctx, r.cfg, r.repo.Dir, libraryID); err != nil {
				return err
			}
		}
		slog.Warn("Cannot perform build, missing library ID")
	}
	return nil
}

// runGenerateCommand checks if the library exists in the remote pipeline state, if so use GenerateLibrary command
// otherwise use GenerateRaw command.
// In case of non-fatal error when looking up library, we will fall back to GenerateRaw command
// and log the error.
// If refined generation is used, the context's languageRepo field will be populated and the
// library ID will be returned; otherwise, an empty string will be returned.
func (r *generateRunner) runGenerateCommand(ctx context.Context, outputDir string, state *config.PipelineState, containerConfig *docker.Docker) (string, error) {
	apiRoot, err := filepath.Abs(r.cfg.Source)
	if err != nil {
		return "", err
	}

	// If we've got a language repo, it's because we've already found a library for the
	// specified API, configured in the repo.
	if r.repo != nil {
		libraryID := findLibraryIDByAPIPath(state, r.cfg.API)
		if libraryID == "" {
			return "", errors.New("bug in Librarian: Library not found during generation, despite being found in earlier steps")
		}
		generatorInput := filepath.Join(r.repo.Dir, config.GeneratorInputDir)
		slog.Info("Performing refined generation for library", "id", libraryID)
		return libraryID, containerConfig.Generate(ctx, r.cfg, apiRoot, outputDir, generatorInput, libraryID)
	}
	slog.Info("No matching library found (or no repo specified)", "path", r.cfg.API)
	return "", fmt.Errorf("library not found")
}

// detectIfLibraryConfigured returns whether a library has been configured for
// the requested API (as specified in apiPath). This is done by checking the local
// pipeline state if repoRoot has been specified, or the remote pipeline state (just
// by fetching the single file) if flatRepoUrl has been specified. If neither the repo
// root not the repo url has been specified, we always perform raw generation.
func (r *generateRunner) detectIfLibraryConfigured(ctx context.Context) (bool, error) {
	apiPath, repo, gitHubToken := r.cfg.API, r.cfg.Repo, r.cfg.GitHubToken
	if repo == "" {
		slog.Warn("repo is not specified, cannot check if library exists")
		return false, nil
	}

	// Attempt to load the pipeline state either locally or from the repo URL
	var (
		pipelineState *config.PipelineState
		err           error
	)
	if !isUrl(repo) {
		// repo is a directory
		pipelineState, err = loadPipelineStateFile(filepath.Join(repo, config.GeneratorInputDir, pipelineStateFile))
		if err != nil {
			return false, err
		}
	} else {
		// repo is a URL
		languageRepoMetadata, err := github.ParseUrl(repo)
		if err != nil {
			slog.Warn("failed to parse", "repo url:", repo, "error", err)
			return false, err
		}
		pipelineState, err = fetchRemotePipelineState(ctx, languageRepoMetadata, "HEAD", gitHubToken)
		if err != nil {
			return false, err
		}
	}
	// If the library doesn't exist, we don't use the repo at all.
	libraryID := findLibraryIDByAPIPath(pipelineState, apiPath)
	if libraryID == "" {
		slog.Info("API path not configured in repo", "path", apiPath)
		return false, nil
	}

	slog.Info("API configured", "path", apiPath, "library", libraryID)
	return true, nil
}
