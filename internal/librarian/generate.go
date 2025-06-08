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
	"github.com/googleapis/librarian/internal/container"
	"github.com/googleapis/librarian/internal/githubrepo"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
)

var CmdGenerate = &cli.Command{
	Name:  "generate",
	Short: "Generate client library code for an API.",
	Usage: "TODO(https://github.com/googleapis/librarian/issues/237): add documentation",
	Long:  "TODO(https://github.com/googleapis/librarian/issues/237): add documentation",
	Run:   runGenerate,
}

func init() {
	CmdGenerate.SetFlags([]func(fs *flag.FlagSet){
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

func runGenerate(ctx context.Context) error {
	if err := validateRequiredFlag("api-path", flagAPIPath); err != nil {
		return err
	}
	if err := validateRequiredFlag("api-root", flagAPIRoot); err != nil {
		return err
	}

	startTime := time.Now()
	workRoot, err := createWorkRoot(startTime)
	if err != nil {
		return err
	}
	libraryConfigured, err := detectIfLibraryConfigured()
	if err != nil {
		return err
	}

	var (
		repo   *gitrepo.Repo
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
	containerConfig, err := container.NewContainerConfig(ctx, workRoot, image, flagSecretsProject, config)
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
	return executeGenerate(state)
}

func executeGenerate(state *commandState) error {
	outputDir := filepath.Join(state.workRoot, "output")
	if err := os.Mkdir(outputDir, 0755); err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Code will be generated in %s", outputDir))

	libraryID, err := runGenerateCommand(state, outputDir)
	if err != nil {
		return err
	}
	if flagBuild {
		if libraryID != "" {
			slog.Info("Build requested in the context of refined generation; cleaning and copying code to the local language repo before building.")
			if err := container.Clean(state.containerConfig, state.languageRepo.Dir, libraryID); err != nil {
				return err
			}
			if err := os.CopyFS(state.languageRepo.Dir, os.DirFS(outputDir)); err != nil {
				return err
			}
			if err := container.BuildLibrary(state.containerConfig, state.languageRepo.Dir, libraryID); err != nil {
				return err
			}
		} else if err := container.BuildRaw(state.containerConfig, outputDir, flagAPIPath); err != nil {
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
func runGenerateCommand(state *commandState, outputDir string) (string, error) {
	apiRoot, err := filepath.Abs(flagAPIRoot)
	if err != nil {
		return "", err
	}

	// If we've got a language repo, it's because we've already found a library for the
	// specified API, configured in the repo.
	if state.languageRepo != nil {
		libraryID := findLibraryIDByApiPath(state.pipelineState, flagAPIPath)
		if libraryID == "" {
			return "", errors.New("bug in Librarian: Library not found during generation, despite being found in earlier steps")
		}
		generatorInput := filepath.Join(state.languageRepo.Dir, "generator-input")
		slog.Info(fmt.Sprintf("Performing refined generation for library %s", libraryID))
		return libraryID, container.GenerateLibrary(state.containerConfig, apiRoot, outputDir, generatorInput, libraryID)
	} else {
		slog.Info(fmt.Sprintf("No matching library found (or no repo specified); performing raw generation for %s", flagAPIPath))
		return "", container.GenerateRaw(state.containerConfig, apiRoot, outputDir, flagAPIPath)
	}
}

// detectIfLibraryConfigured returns whether or not a library has been configured for
// the requested API (as specified in flagAPIPath). This is done by checking the local
// pipeline state if flagRepoRoot has been specified, or the remote pipeline state (just
// by fetching the single file) if flatRepoUrl has been specified. If neither the repo
// root not the repo url has been specified, we always perform raw generation.
func detectIfLibraryConfigured() (bool, error) {
	if flagRepoUrl == "" && flagRepoRoot == "" {
		slog.Warn("repo url and root are not specified, cannot check if library exists")
		return false, nil
	}

	if flagRepoRoot != "" && flagRepoUrl != "" {
		return false, errors.New("do not specify both repo-root and repo-url")
	}

	// Attempt to load the pipeline state either locally or from the repo URL
	var pipelineState *statepb.PipelineState
	var err error
	if flagRepoRoot != "" {
		pipelineState, err = loadPipelineStateFile(filepath.Join(flagRepoRoot, "generator-input", pipelineStateFile))
	} else {
		var languageRepoMetadata githubrepo.GitHubRepo
		languageRepoMetadata, err = githubrepo.ParseUrl(flagRepoUrl)
		if err != nil {
			slog.Warn("failed to parse", "repo url:", flagRepoUrl, "error", err)
			return false, err
		}
		pipelineState, err = fetchRemotePipelineState(context.Background(), languageRepoMetadata, "HEAD")
	}

	if err != nil {
		return false, err
	}

	// If the library doesn't exist, we don't use the repo at all.
	libraryID := findLibraryIDByApiPath(pipelineState, flagAPIPath)
	if libraryID == "" {
		slog.Info(fmt.Sprintf("API path %s not configured in repo", flagAPIPath))
		return false, nil
	}

	slog.Info(fmt.Sprintf("API path %s configured in repo library %s", flagAPIPath, libraryID))
	return true, nil
}
