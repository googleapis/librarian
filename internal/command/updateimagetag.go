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
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/googleapis/librarian/internal/container"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
)

var CmdUpdateImageTag = &Command{
	Name:  "update-image-tag",
	Short: "Update the image tag used by a language repo, and regenerating all APIs at the existing commit",
	Run: func(ctx context.Context) error {
		if err := validateLanguage(); err != nil {
			return err
		}
		if err := validatePush(); err != nil {
			return err
		}
		if err := validateRequiredFlag("tag", flagTag); err != nil {
			return err
		}

		startOfRun := time.Now()

		// tmpRoot is a newly-created working directory under /tmp
		// We do any cloning or copying under there.
		tmpRoot, err := createTmpWorkingRoot(startOfRun)
		if err != nil {
			return err
		}

		var apiRepo *gitrepo.Repo
		if flagAPIRoot == "" {
			apiRepo, err = cloneGoogleapis(ctx, tmpRoot)
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
			apiRepo, err = gitrepo.Open(ctx, apiRoot)
			if err != nil {
				return err
			}
			clean, err := gitrepo.IsClean(ctx, apiRepo)
			if err != nil {
				return err
			}
			if !clean {
				return errors.New("api repo must be clean before updating the language image tag")
			}
		}

		var outputDir string
		if flagOutput == "" {
			outputDir = filepath.Join(tmpRoot, "output")
			if err := os.Mkdir(outputDir, 0755); err != nil {
				return err
			}
			slog.Info(fmt.Sprintf("No output directory specified. Defaulting to %s", outputDir))
		} else {
			outputDir, err = filepath.Abs(flagOutput)
			if err != nil {
				return err
			}
		}

		var languageRepo *gitrepo.Repo
		if flagRepoRoot == "" {
			languageRepo, err = cloneLanguageRepo(ctx, flagLanguage, tmpRoot)
			if err != nil {
				return err
			}
		} else {
			repoRoot, err := filepath.Abs(flagRepoRoot)
			if err != nil {
				return err
			}
			languageRepo, err = gitrepo.Open(ctx, repoRoot)
			if err != nil {
				return err
			}
			clean, err := gitrepo.IsClean(ctx, apiRepo)
			if err != nil {
				return err
			}
			if !clean {
				return errors.New("language repo must be clean before update")
			}
		}

		state, err := loadState(languageRepo)
		if err != nil {
			return err
		}

		if state.ImageTag == flagTag {
			return errors.New("specified tag is already in language repo state")
		}
		state.ImageTag = flagTag
		image := deriveImage(state)
		saveState(languageRepo, state)

		// Take a defensive copy of the generator input directory from the language repo.
		generatorInput := filepath.Join(tmpRoot, "generator-input")
		if err := os.CopyFS(generatorInput, os.DirFS(filepath.Join(languageRepo.Dir, "generator-input"))); err != nil {
			return err
		}

		// Perform "generate, clean" on each library.
		for _, library := range state.Libraries {
			err = regenerateLibrary(ctx, apiRepo, languageRepo, generatorInput, image, outputDir, library)
			if err != nil {
				return err
			}
		}

		// Commit any changes
		commitMsg := fmt.Sprintf("chore: update generation image tag to %s", flagTag)
		if err := commitAll(ctx, languageRepo, commitMsg); err != nil {
			return err
		}

		// Build everything at the end. (This is more efficient than building each library with a separate container invocation.)
		slog.Info("Building all libraries.")
		if err := container.BuildLibrary(image, languageRepo.Dir, ""); err != nil {
			return err
		}

		if !flagPush {
			slog.Info("Pushing not specified; update complete.")
			return nil
		}

		return push(ctx, languageRepo, startOfRun, "chore: update generation image tag", "")
	},
}

func regenerateLibrary(ctx context.Context, apiRepo *gitrepo.Repo, languageRepo *gitrepo.Repo, generatorInput string, image string, outputRoot string, library *statepb.LibraryState) error {
	if len(library.ApiPaths) == 0 {
		slog.Info(fmt.Sprintf("Skipping non-generated library: '%s'", library.Id))
		return nil
	}

	// TODO: Handle "no last generated commit"
	if err := gitrepo.Checkout(ctx, apiRepo, library.LastGeneratedCommit); err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Generating '%s'", library.Id))

	// We create an output directory separately for each API.
	outputDir := filepath.Join(outputRoot, library.Id)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	if err := container.GenerateLibrary(ctx, image, apiRepo.Dir, outputDir, generatorInput, library.Id); err != nil {
		return err
	}
	if err := container.Clean(ctx, image, languageRepo.Dir, library.Id); err != nil {
		return err
	}
	if err := os.CopyFS(languageRepo.Dir, os.DirFS(outputDir)); err != nil {
		return err
	}
	if err := gitrepo.ResetHard(ctx, apiRepo); err != nil {
		return err
	}
	return nil
}
