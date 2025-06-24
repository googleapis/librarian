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
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/githubrepo"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
)

var cmdGenerate = &cli.Command{
	Short: "generate generates client library code for a single API",
	Usage: "librarian generate -api-root=<api-root> -api-path=<api-path> -language=<language> [flags]",
	Long: `Specify the language, the API repository root and the path within it for the API to generate.
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
		if err := validateRequiredFlag("api-path", flagAPIPath); err != nil {
			return err
		}
		if err := validateRequiredFlag("api-root", flagAPIRoot); err != nil {
			return err
		}
		return runGenerate(ctx, cfg)
	},
}

func init() {
	cmdGenerate.SetFlags([]func(fs *flag.FlagSet){
		addFlagImage,
		addFlagWorkRoot,
		addFlagAPIPath,
		addFlagAPIRoot,
		addFlagLanguage,
		addFlagBuild,
		addFlagRepoRoot,
		addFlagRepoUrl,
		addFlagSecretsProject,
	})
}

func runGenerate(ctx context.Context, cfg *config.Config) error {
	libraryConfigured, err := detectIfLibraryConfigured(cfg.APIPath, cfg.RepoURL, cfg.RepoRoot, cfg.GitHubToken)
	if err != nil {
		return err
	}

	// TODO(https://github.com/googleapis/librarian/issues/482): pass in
	// flagWorkroot explicitly.
	startTime := time.Now()
	workRoot, err := createWorkRoot(startTime)
	if err != nil {
		return err
	}

	var (
		repo   *gitrepo.Repository
		ps     *statepb.PipelineState
		config *statepb.PipelineConfig
	)
	// We only clone/open the language repo and use the state within it
	// if the requested API is configured as a library.
	if libraryConfigured {
		repo, err = cloneOrOpenLanguageRepo(workRoot)
		if err != nil {
			return err
		}

		ps, config, err = loadRepoStateAndConfig(repo)
		if err != nil {
			return err
		}
	}

	image := deriveImage(ps)
	containerConfig, err := docker.New(ctx, workRoot, image, cfg.SecretsProject, config)
	if err != nil {
		return err
	}

	state := &commandState{
		ctx:             ctx,
		startTime:       startTime,
		workRoot:        workRoot,
		languageRepo:    repo,
		pipelineConfig:  config,
		pipelineState:   ps,
		containerConfig: containerConfig,
	}
	return executeGenerate(state, cfg.APIPath, cfg.APIRoot, cfg.Build)
}

func executeGenerate(state *commandState, apiPath, apiRoot string, build bool) error {
	outputDir := filepath.Join(state.workRoot, "output")
	if err := os.Mkdir(outputDir, 0755); err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Code will be generated in %s", outputDir))

	libraryID, err := runGenerateCommand(state, apiRoot, apiPath, outputDir)
	if err != nil {
		return err
	}
	if build {
		if libraryID != "" {
			slog.Info("Build requested in the context of refined generation; cleaning and copying code to the local language repo before building.")
			if err := state.containerConfig.Clean(state.languageRepo.Dir, libraryID); err != nil {
				return err
			}
			if err := os.CopyFS(state.languageRepo.Dir, os.DirFS(outputDir)); err != nil {
				return err
			}
			if err := state.containerConfig.BuildLibrary(state.languageRepo.Dir, libraryID); err != nil {
				return err
			}
		} else if err := state.containerConfig.BuildRaw(outputDir, apiPath); err != nil {
			return err
		}
	}
	return nil
}

// Checks if the library exists in the remote pipeline state, if so use GenerateLibrary command
// otherwise use GenerateRaw command.
// In case of non fatal error when looking up library, we will fallback to GenerateRaw command
// and log the error.
// If refined generation is used, the context's languageRepo field will be populated and the
// library ID will be returned; otherwise, an empty string will be returned.
func runGenerateCommand(state *commandState, apiRoot, apiPath, outputDir string) (string, error) {
	apiRoot, err := filepath.Abs(apiRoot)
	if err != nil {
		return "", err
	}

	// If we've got a language repo, it's because we've already found a library for the
	// specified API, configured in the repo.
	if state.languageRepo != nil {
		libraryID := findLibraryIDByApiPath(state.pipelineState, apiPath)
		if libraryID == "" {
			return "", errors.New("bug in Librarian: Library not found during generation, despite being found in earlier steps")
		}
		generatorInput := filepath.Join(state.languageRepo.Dir, "generator-input")
		slog.Info(fmt.Sprintf("Performing refined generation for library %s", libraryID))
		return libraryID, state.containerConfig.GenerateLibrary(apiRoot, outputDir, generatorInput, libraryID)
	} else {
		slog.Info(fmt.Sprintf("No matching library found (or no repo specified); performing raw generation for %s", apiPath))
		return "", state.containerConfig.GenerateRaw(apiRoot, outputDir, apiPath)
	}
}

// detectIfLibraryConfigured returns whether or not a library has been configured for
// the requested API (as specified in apiPath). This is done by checking the local
// pipeline state if repoRoot has been specified, or the remote pipeline state (just
// by fetching the single file) if flatRepoUrl has been specified. If neither the repo
// root not the repo url has been specified, we always perform raw generation.
func detectIfLibraryConfigured(apiPath, repoURL, repoRoot, gitHubToken string) (bool, error) {
	if repoURL == "" && repoRoot == "" {
		slog.Warn("repo url and root are not specified, cannot check if library exists")
		return false, nil
	}
	if repoRoot != "" && repoURL != "" {
		return false, errors.New("do not specify both repo-root and repo-url")
	}

	// Attempt to load the pipeline state either locally or from the repo URL
	var (
		pipelineState *statepb.PipelineState
		err           error
	)
	if repoRoot != "" {
		pipelineState, err = loadPipelineStateFile(filepath.Join(repoRoot, "generator-input", pipelineStateFile))
		if err != nil {
			return false, err
		}
	} else {
		languageRepoMetadata, err := githubrepo.ParseUrl(repoURL)
		if err != nil {
			slog.Warn("failed to parse", "repo url:", repoURL, "error", err)
			return false, err
		}
		pipelineState, err = fetchRemotePipelineState(context.Background(), languageRepoMetadata, "HEAD", gitHubToken)
		if err != nil {
			return false, err
		}
	}
	// If the library doesn't exist, we don't use the repo at all.
	libraryID := findLibraryIDByApiPath(pipelineState, apiPath)
	if libraryID == "" {
		slog.Info(fmt.Sprintf("API path %s not configured in repo", apiPath))
		return false, nil
	}

	slog.Info("API configured", "path", apiPath, "library", libraryID)
	return true, nil
}
