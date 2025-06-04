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
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/container"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
)

var CmdUpdateImageTag = &Command{
	Name:  "update-image-tag",
	Short: "Update a language repo's image tag and regenerate APIs.",
	flagFunctions: []func(fs *flag.FlagSet){
		addFlagWorkRoot,
		addFlagAPIRoot,
		addFlagBranch,
		addFlagGitUserEmail,
		addFlagGitUserName,
		addFlagLanguage,
		addFlagPush,
		addFlagRepoRoot,
		addFlagRepoUrl,
		addFlagSecretsProject,
		addFlagTag,
	},
	maybeGetLanguageRepo:    cloneOrOpenLanguageRepo,
	maybeLoadStateAndConfig: loadRepoStateAndConfig,
	execute: func(state *commandState) error {
		if err := validatePush(); err != nil {
			return err
		}
		if err := validateRequiredFlag("tag", flagTag); err != nil {
			return err
		}

		var apiRepo *gitrepo.Repo
		if flagAPIRoot == "" {
			var err error
			apiRepo, err = cloneGoogleapis(state.workRoot)
			if err != nil {
				return err
			}
		} else {
			apiRoot, err := filepath.Abs(flagAPIRoot)
			slog.Info(fmt.Sprintf("Using apiRoot: %s", apiRoot))
			if err != nil {
				slog.Info(fmt.Sprintf("Error retrieving apiRoot: %s", err))
				return err
			}
			apiRepo, err = gitrepo.Open(apiRoot)
			if err != nil {
				return err
			}
			clean, err := gitrepo.IsClean(apiRepo)
			if err != nil {
				return err
			}
			if !clean {
				return errors.New("api repo must be clean before updating the language image tag")
			}
		}

		outputDir := filepath.Join(state.workRoot, "output")
		if err := os.Mkdir(outputDir, 0755); err != nil {
			return err
		}
		slog.Info(fmt.Sprintf("Code will be generated in %s", outputDir))

		ps := state.pipelineState
		languageRepo := state.languageRepo

		if ps.ImageTag == flagTag {
			return errors.New("specified tag is already in language repo state")
		}
		// Derive the new image to use, and save it in the context.
		ps.ImageTag = flagTag
		state.containerConfig.Image = deriveImage(ps)
		savePipelineState(state)

		// Take a defensive copy of the generator input directory from the language repo.
		generatorInput := filepath.Join(state.workRoot, "generator-input")
		if err := os.CopyFS(generatorInput, os.DirFS(filepath.Join(languageRepo.Dir, "generator-input"))); err != nil {
			return err
		}

		// Perform "generate, clean" on each library.
		for _, library := range ps.Libraries {
			err := regenerateLibrary(state, apiRepo, generatorInput, outputDir, library)
			if err != nil {
				return err
			}
		}

		// Commit any changes
		commitMsg := fmt.Sprintf("chore: update generation image tag to %s", flagTag)
		if err := commitAll(languageRepo, commitMsg); err != nil {
			return err
		}

		// Build everything at the end. (This is more efficient than building each library with a separate container invocation.)
		slog.Info("Building all libraries.")
		if err := container.BuildLibrary(state.containerConfig, languageRepo.Dir, ""); err != nil {
			return err
		}

		// The PullRequestContent for update-image-tag is slightly different to others, but we
		// can massage it into a similar state.
		prContent := new(PullRequestContent)
		addSuccessToPullRequest(prContent, "Regenerated all libraries with new image tag.")
		_, err := createPullRequest(state, prContent, "chore: update generation image tag", "", "update-image-tag")
		return err
	},
}

func regenerateLibrary(state *commandState, apiRepo *gitrepo.Repo, generatorInput string, outputRoot string, library *statepb.LibraryState) error {
	containerConfig := state.containerConfig
	languageRepo := state.languageRepo

	if len(library.ApiPaths) == 0 {
		slog.Info(fmt.Sprintf("Skipping non-generated library: '%s'", library.Id))
		return nil
	}

	// TODO: Handle "no last generated commit"
	if err := gitrepo.Checkout(apiRepo, library.LastGeneratedCommit); err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Generating '%s'", library.Id))

	// We create an output directory separately for each API.
	outputDir := filepath.Join(outputRoot, library.Id)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	if err := container.GenerateLibrary(containerConfig, apiRepo.Dir, outputDir, generatorInput, library.Id); err != nil {
		return err
	}
	if err := container.Clean(containerConfig, languageRepo.Dir, library.Id); err != nil {
		return err
	}
	if err := os.CopyFS(languageRepo.Dir, os.DirFS(outputDir)); err != nil {
		return err
	}
	if err := gitrepo.CleanWorkingTree(apiRepo); err != nil {
		return err
	}
	return nil
}
