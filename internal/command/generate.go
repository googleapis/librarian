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

package command

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/container"
)

var CmdGenerate = &Command{
	Name:  "generate",
	Short: "Generate client library code for an API.",
	Run:   runGenerate,
	flagFunctions: []func(fs *flag.FlagSet){
		addFlagImage,
		addFlagWorkRoot,
		addFlagAPIPath,
		addFlagAPIRoot,
		addFlagLanguage,
		addFlagBuild,
		addFlagRepoRoot,
		addFlagRepoUrl,
		addFlagSecretsProject,
	},
}

func runGenerate(ctx context.Context) error {
	state, err := createContainerForLanguage(ctx)
	if err != nil {
		return err
	}
	return executeGenerate(state)
}

func executeGenerate(state *commandState) error {
	if err := validateRequiredFlag("api-path", flagAPIPath); err != nil {
		return err
	}
	if err := validateRequiredFlag("api-root", flagAPIRoot); err != nil {
		return err
	}

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
