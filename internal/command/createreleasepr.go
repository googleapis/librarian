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
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/googleapis/librarian/internal/container"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var CmdCreateReleasePR = &Command{
	Name:  "create-release-pr",
	Short: "Generate a PR for release",
	Run: func(ctx context.Context) error {
		if err := validateLanguage(); err != nil {
			return err
		}
		if err := validatePush(); err != nil {
			return err
		}
		languageRepo, inputDirectory, err := setupReleasePrFolders(ctx)
		if err != nil {
			return err
		}

		pipelineState, err := loadState(languageRepo)
		if err != nil {
			slog.Info(fmt.Sprintf("Error loading pipeline state: %s", err))
			return err
		}

		if flagImage == "" {
			flagImage = deriveImage(pipelineState)
		}

		prDescription, errorsInGeneration, err := generateReleaseCommitForEachLibrary(ctx, languageRepo.Dir, languageRepo, inputDirectory, pipelineState)
		if err != nil {
			return err
		}

		return generateReleasePr(ctx, languageRepo, prDescription, errorsInGeneration)
	},
}

func setupReleasePrFolders(ctx context.Context) (*gitrepo.Repo, string, error) {
	startOfRun := time.Now()
	tmpRoot, err := createTmpWorkingRoot(startOfRun)
	if err != nil {
		return nil, "", err
	}
	var languageRepo *gitrepo.Repo
	if flagRepoRoot == "" {
		languageRepo, err = cloneLanguageRepo(ctx, flagLanguage, tmpRoot)
		if err != nil {
			return nil, "", err
		}
	} else {
		languageRepo, err = gitrepo.Open(ctx, flagRepoRoot)
		if err != nil {
			return nil, "", err
		}
	}

	inputDir := filepath.Join(tmpRoot, "inputs")
	if err := os.Mkdir(inputDir, 0755); err != nil {
		slog.Error("Failed to create input directory")
		return nil, "", err
	}

	return languageRepo, inputDir, nil
}

func generateReleasePr(ctx context.Context, repo *gitrepo.Repo, prDescription string, errorsInGeneration bool) error {
	if prDescription != "" {
		prNumber, err := push(ctx, repo, time.Now(), "chore(main): release", "Release "+prDescription)
		if err != nil {
			slog.Warn(fmt.Sprintf("Received error trying to create release PR: '%s'", err))
			return err
		}
		if errorsInGeneration {
			gitHubAccessToken := os.Getenv(gitHubTokenEnvironmentVariable)
			err = gitrepo.AddLabelToPr(ctx, repo, prNumber, gitHubAccessToken, "do-not-merge")
			if err != nil {
				slog.Warn(fmt.Sprintf("Received error trying to add label to PR: '%s'", err))
				return nil //TODO: check if its okay to ignore
			}
		}
	}
	return nil
}

/*
this goes through each library in pipeline state and checks if any new commits have been added to that library since previous commit tag
*/
func generateReleaseCommitForEachLibrary(ctx context.Context, repoPath string, repo *gitrepo.Repo, inputDirectory string, pipelineState *statepb.PipelineState) (string, bool, error) {
	libraries := pipelineState.Libraries
	var prDescription string
	var errorsInGeneration []string
	for _, library := range libraries {
		if library.GenerationAutomationLevel == statepb.AutomationLevel_AUTOMATION_LEVEL_BLOCKED {
			slog.Info(fmt.Sprintf("Skipping release-blocked library: '%s'", library.Id))
			continue
		}
		var commitMessages []*CommitMessage

		previousReleaseTag := library.Id + "-" + library.CurrentVersion
		allSourcePaths := append(pipelineState.CommonLibrarySourcePaths, library.SourcePaths...)
		commits, err := gitrepo.GetCommitsForPathsSinceTag(repo, allSourcePaths, previousReleaseTag)
		if err != nil {
			logErrorAndAppendToErrorList(fmt.Sprintf("%s: unable to retrieve commits since previous release tag %s", library.Id, previousReleaseTag), errorsInGeneration)
			continue
		}
		for _, commit := range commits {
			commitMessages = append(commitMessages, ParseCommit(commit))
		}

		if len(commitMessages) > 0 && isReleaseWorthy(commitMessages, library.Id) {
			releaseVersion, err := calculateNextVersion(library)
			if err != nil {
				errorsInGeneration = logErrorAndAppendToErrorList(fmt.Sprintf("%s: unable to calculate next version %s", library.Id, err), errorsInGeneration)
				continue
			}

			releaseNotes := formatReleaseNotes(commitMessages)
			if err = createReleaseNotesFile(inputDirectory, library.Id, releaseVersion, releaseNotes); err != nil {
				errorsInGeneration = logErrorAndAppendToErrorList(fmt.Sprintf("%s: unable to create release notes file %s", library.Id, err), errorsInGeneration)
				continue
			}

			if err := container.PrepareLibraryRelease(flagImage, repoPath, inputDirectory, library.Id, releaseVersion); err != nil {
				errorsInGeneration = logErrorAndAppendToErrorList(fmt.Sprintf("%s: prepare-library-release error %s", library.Id, err), errorsInGeneration)
				continue
			}
			//TODO: make this configurable so we don't have to run per library
			if !flagSkipBuild {
				if err := container.BuildLibrary(flagImage, repoPath, library.Id); err != nil {
					errorsInGeneration = logErrorAndAppendToErrorList(fmt.Sprintf("%s: build/test failed with error %s", library.Id, err), errorsInGeneration)
					continue
				}
			}

			// Update the pipeline state to record what we've released and when.
			library.CurrentVersion = releaseVersion
			library.LastReleasedCommit = library.LastGeneratedCommit
			library.ReleaseTimestamp = timestamppb.Now()

			if err = saveState(repo, pipelineState); err != nil {
				errorsInGeneration = logErrorAndAppendToErrorList(fmt.Sprintf("%s: unable to save pipeline state %s", library.Id, err), errorsInGeneration)
				err := gitrepo.CleanWorkingTree(repo)
				if err != nil {
					isClean, err2 := gitrepo.IsClean(ctx, repo)
					if err2 != nil || !isClean {
						slog.Error(fmt.Sprintf("working directory is in a bad state aborting process: '%s'", err))
						return "", true, err
					}
				}
				continue
			}

			//TODO: add extra meta data what is this
			prDescription += fmt.Sprintf("Release library: %s version %s\n", library.Id, releaseVersion)
			libraryReleaseCommitDesc := fmt.Sprintf("Release library: %s version %s\n\n", library.Id, releaseVersion)

			err = commitAll(ctx, repo, libraryReleaseCommitDesc+releaseNotes)
			if err != nil {
				//TODO: need to work out the different states: no commit happened vs need to rollback commit
				errorsInGeneration = logErrorAndAppendToErrorList(fmt.Sprintf("%s: unable to create release commit %s", library.Id, err), errorsInGeneration)
				continue
			}
		}
	}
	if (errorsInGeneration != nil) && len(errorsInGeneration) > 0 {
		prDescription = fmt.Sprintf("There were errors found in creating this release PR:%s\n%s\n", strings.Join(errorsInGeneration, "\n"), prDescription)
	}
	return prDescription, (errorsInGeneration == nil || len(errorsInGeneration) == 0), nil
}

func logErrorAndAppendToErrorList(message string, errorsInGeneration []string) []string {
	slog.Warn(message)
	errorsInGeneration = append(errorsInGeneration, message)
	return errorsInGeneration
}

func formatReleaseNotes(commitMessages []*CommitMessage) string {
	features := []string{}
	docs := []string{}
	fixes := []string{}

	// TODO: Deduping (same message across multiple commits)
	// TODO: Breaking changes
	// TODO: Use the source links etc
	for _, commitMessage := range commitMessages {
		features = append(features, commitMessage.Features...)
		docs = append(docs, commitMessage.Docs...)
		fixes = append(fixes, commitMessage.Fixes...)
	}

	var builder strings.Builder

	maybeAppendReleaseNotesSection(&builder, "New features", features)
	maybeAppendReleaseNotesSection(&builder, "Bug fixes", fixes)
	maybeAppendReleaseNotesSection(&builder, "Documentation improvements", docs)

	if builder.Len() == 0 {
		// TODO: Work out something rather better than this...
		builder.WriteString("No specific release notes.")
	}
	return builder.String()
}

func createReleaseNotesFile(inputDirectory, libraryId, releaseVersion, releaseNotes string) error {
	path := filepath.Join(inputDirectory, fmt.Sprintf("%s-%s-release-notes.txt", libraryId, releaseVersion))

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	_, err = file.WriteString(releaseNotes)
	if err != nil {
		return err
	}
	return nil
}

func maybeAppendReleaseNotesSection(builder *strings.Builder, description string, lines []string) {
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(builder, "### %s\n\n", description)
	for _, line := range lines {
		fmt.Fprintf(builder, "- %s\n", line)
	}
	builder.WriteString("\n")
}

func calculateNextVersion(library *statepb.LibraryState) (string, error) {
	if library.NextVersion != "" {
		return library.NextVersion, nil
	}
	current, err := semver.StrictNewVersion(library.CurrentVersion)
	if err != nil {
		return "", err
	}
	var next *semver.Version
	prerelease := current.Prerelease()
	if prerelease != "" {
		nextPrerelease, err := calculateNextPrerelease(prerelease)
		if err != nil {
			return "", err
		}
		next = semver.New(current.Major(), current.Minor(), current.Patch(), nextPrerelease, "")
	} else {
		next = semver.New(current.Major(), current.Minor()+1, current.Patch(), "", "")
	}
	return next.String(), nil
}

// Match trailing digits in the prerelease part, then parse those digits as an integer.
// Increment the integer, then format it again - keeping as much of the existing prerelease as is
// required to end up with a string longer-than-or-equal to the original.
// If there are no trailing digits, fail.
// Note: this assumes the prerelease is purely ASCII.
func calculateNextPrerelease(prerelease string) (string, error) {
	digits := 0
	for i := len(prerelease) - 1; i >= 0; i-- {
		c := prerelease[i]
		if c < '0' || c > '9' {
			break
		} else {
			digits++
		}
	}
	if digits == 0 {
		return "", fmt.Errorf("unable to create next prerelease from '%s'", prerelease)
	}
	currentSuffix := prerelease[len(prerelease)-digits:]
	currentNumber, err := strconv.Atoi(currentSuffix)
	if err != nil {
		return "", err
	}
	nextSuffix := strconv.Itoa(currentNumber + 1)
	if len(nextSuffix) < len(currentSuffix) {
		nextSuffix = strings.Repeat("0", len(currentSuffix)-len(nextSuffix)) + nextSuffix
	}
	return prerelease[:(len(prerelease)-digits)] + nextSuffix, nil
}

func isReleaseWorthy(messages []*CommitMessage, libraryId string) bool {
	for _, message := range messages {
		// TODO: Work out why we can't call message.IsReleaseWorthy(libraryId)
		if IsReleaseWorthy(message, libraryId) {
			return true
		}
	}
	return false
}
