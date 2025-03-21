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
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/googleapis/librarian/internal/container"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
)

var CmdUpdateApis = &Command{
	Name:  "update-apis",
	Short: "Update a language repo by regenerating configured APIs with new API specifications",
	Run: func(ctx context.Context) error {

		if err := validateLanguage(); err != nil {
			return err
		}
		if err := validatePush(); err != nil {
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
		hardResetApiRepo := true
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
				hardResetApiRepo = false
				slog.Warn("API repo has modifications, so will not be reset after generation")
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

		image := deriveImage(state)

		// Take a defensive copy of the generator input directory from the language repo.
		generatorInput := filepath.Join(tmpRoot, "generator-input")
		if err := os.CopyFS(generatorInput, os.DirFS(filepath.Join(languageRepo.Dir, "generator-input"))); err != nil {
			return err
		}

		hashBefore, err := gitrepo.HeadHash(ctx, languageRepo)
		if err != nil {
			return err
		}

		// Perform "generate, clean, commit, build" on each library.
		for _, library := range state.Libraries {
			err = updateLibrary(ctx, apiRepo, languageRepo, generatorInput, image, outputDir, state, library)
			if err != nil {
				return err
			}
		}

		// Reset the API repo in case it was changed, but not if it was already dirty before the command.
		if hardResetApiRepo {
			gitrepo.ResetHard(ctx, apiRepo)
		}

		if !flagPush {
			slog.Info("Pushing not specified; update complete.")
			return nil
		}

		hashAfter, err := gitrepo.HeadHash(ctx, languageRepo)
		if err != nil {
			return err
		}
		if hashBefore == hashAfter {
			slog.Info("No changes generated; nothing to push.")
			return nil
		}

		title := fmt.Sprintf("feat: API regeneration: %s", formatTimestamp(startOfRun))
		_, err = push(ctx, languageRepo, startOfRun, title, "")
		return err
	},
}

func updateLibrary(ctx context.Context, apiRepo *gitrepo.Repo, languageRepo *gitrepo.Repo, generatorInput string, image string, outputRoot string, repoState *statepb.PipelineState, library *statepb.LibraryState) error {
	if flagLibraryID != "" && flagLibraryID != library.Id {
		// If flagLibraryID has been passed in, we only act on that library.
		return nil
	}

	if len(library.ApiPaths) == 0 {
		slog.Info(fmt.Sprintf("Skipping non-generated library: '%s'", library.Id))
		return nil
	}

	if library.GenerationAutomationLevel == statepb.AutomationLevel_AUTOMATION_LEVEL_BLOCKED {
		slog.Info(fmt.Sprintf("Skipping generation-blocked library: '%s'", library.Id))
		return nil
	}
	commits, err := gitrepo.GetCommitsForPathsSinceCommit(apiRepo, library.ApiPaths, library.LastGeneratedCommit)
	if err != nil {
		return err
	}
	if len(commits) == 0 {
		slog.Info(fmt.Sprintf("Library '%s' has no changes.", library.Id))
		return nil
	}
	slog.Info(fmt.Sprintf("Generating '%s' with %d new commit(s)", library.Id, len(commits)))

	// Now that we know the API has at least one new API commit, regenerate it, update the state, commit the change and build the output.

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

	library.LastGeneratedCommit = commits[0].Hash.String()
	if err := saveState(languageRepo, repoState); err != nil {
		return err
	}

	// Note that as we've updated the state, we'll definitely have something to commit, even if no
	// generated code changed. This avoids us regenerating no-op changes again and again, and reflects
	// that we really are at the latest state. We could skip the build step here if there are no changes
	// prior to updating the state, but it's probably not worth the additional complexity (and it does
	// no harm to check the code is still "healthy").
	var msg = createCommitMessage(commits)
	if err := commitAll(ctx, languageRepo, msg); err != nil {
		return err
	}

	// Once we've committed, we can build - but then check that nothing has changed afterwards.
	if err := container.BuildLibrary(image, languageRepo.Dir, library.Id); err != nil {
		return err
	}
	clean, err := gitrepo.IsClean(ctx, languageRepo)
	if err != nil {
		return err
	}
	if !clean {
		return fmt.Errorf("building '%s' created changes in the repo", library.Id)
	}
	return nil
}

func createCommitMessage(commits []object.Commit) string {
	const PiperPrefix = "PiperOrigin-RevId: "
	var builder strings.Builder
	piperRevIdLines := []string{}
	sourceLinkLines := []string{}
	// Consume the commits in reverse order, so that they're in normal chronological order,
	// accumulating PiperOrigin-RevId and Source-Link lines.
	for i := len(commits) - 1; i >= 0; i-- {
		commit := commits[i]
		messageLines := strings.Split(commit.Message, "\n")
		sourceLinkLines = append(sourceLinkLines, fmt.Sprintf("Source-Link: https://github.com/googleapis/googleapis/commit/%s", commit.Hash.String()))
		for _, line := range messageLines {
			if strings.HasPrefix(line, PiperPrefix) {
				piperRevIdLines = append(piperRevIdLines, line)
			} else {
				builder.WriteString(line)
				builder.WriteString("\n")
			}

		}
	}
	for _, revIdLine := range piperRevIdLines {
		builder.WriteString(revIdLine)
		builder.WriteString("\n")
	}
	for _, sourceLinkLine := range sourceLinkLines {
		builder.WriteString(sourceLinkLine)
		builder.WriteString("\n")
	}
	return builder.String()
}
