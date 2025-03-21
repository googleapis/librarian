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
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/googleapis/librarian/internal/container"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
)

var CmdCreateReleasePR = &Command{
	Name:  "create-release-pr",
	Short: "Generate a PR for release",
	Run: func(ctx context.Context) error {
		err := checkFlags()
		if err != nil {
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

		prDescription, err := generateReleaseCommitForEachLibrary(ctx, languageRepo.Dir, languageRepo, inputDirectory, pipelineState)
		if err != nil {
			return err
		}

		return generateReleasePr(ctx, languageRepo, prDescription)
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

func checkFlags() error {
	if !supportedLanguages[flagLanguage] {
		return fmt.Errorf("invalid -language flag specified: %q", flagLanguage)
	}
	return nil
}

func generateReleasePr(ctx context.Context, repo *gitrepo.Repo, prDescription string) error {
	if prDescription != "" {
		err := push(ctx, repo, time.Now(), "chore(main): release", "Release "+prDescription)
		if err != nil {
			slog.Info(fmt.Sprintf("Received error trying to create release PR: '%s'", err))
			return err
		}
	}
	return nil
}

/*
this goes through each library in pipeline state and checks if any new commits have been added to that library since previous commit tag
*/
func generateReleaseCommitForEachLibrary(ctx context.Context, repoPath string, repo *gitrepo.Repo, inputDirectory string, pipelineState *statepb.PipelineState) (string, error) {
	libraries := pipelineState.LibraryReleaseStates
	var prDescription string
	var lastGeneratedCommit object.Commit
	for _, library := range libraries {
		var commitMessages []string
		//TODO: need to add common paths as well as refactor to see if can check all paths at 1 x
		for _, sourcePath := range library.SourcePaths {
			//TODO: figure out generic logic
			previousReleaseTag := library.Id + "-" + library.CurrentVersion
			commits, err := gitrepo.GetCommitsSinceTagForPath(repo, sourcePath, previousReleaseTag)
			if err != nil {
				slog.Error(fmt.Sprintf("Error searching commits: %s", err))
				//TODO update PR description with this data and mark as not humanly resolvable
			}
			for _, commit := range commits {
				commitMessages = append(commitMessages, commit.Message)
			}
			if len(commitMessages) > 0 {
				lastGeneratedCommit = commits[len(commits)-1]
			}
		}

		if len(commitMessages) > 0 && isReleaseWorthy(commitMessages) {
			releaseVersion, err := calculateNextVersion(library)
			if err != nil {
				return "", err
			}

			releaseNotes, err := createReleaseNotes(library, commitMessages, inputDirectory, releaseVersion)
			if err != nil {
				return "", err
			}

			if err := container.UpdateReleaseMetadata(flagImage, repoPath, inputDirectory, library.Id, releaseVersion); err != nil {
				slog.Info(fmt.Sprintf("Received error running container: '%s'", err))
				//TODO: log in release PR
				continue
			}
			//TODO: make this configurable so we don't have to run per library
			if !flagSkipBuild {
				if err := container.BuildLibrary(flagImage, repoPath, library.Id); err != nil {
					slog.Info(fmt.Sprintf("Received error running container: '%s'", err))
					continue
					//TODO: log in release PR
				}
			}

			//TODO: add extra meta data what is this
			prDescription += fmt.Sprintf("Release library: %s version %s\n", library.Id, releaseVersion)

			libraryReleaseCommitDesc := fmt.Sprintf("Release library: %s version %s\n", library.Id, releaseVersion)

			updateLibraryMetadata(library.Id, releaseVersion, lastGeneratedCommit.Hash.String(), pipelineState)

			err = saveState(repo, pipelineState)
			if err != nil {
				return "", err
			}

			err = createLibraryReleaseCommit(ctx, repo, libraryReleaseCommitDesc+releaseNotes)
			if err != nil {
				//TODO: need to revert the changes made to state for this library/reload from last commit
			}
		}
	}
	return prDescription, nil
}

func createLibraryReleaseCommit(ctx context.Context, repo *gitrepo.Repo, releaseNotes string) error {
	_, err := gitrepo.AddAll(ctx, repo)
	if err != nil {
		slog.Info(fmt.Sprintf("Error adding files: %s", err))
		return err
		//TODO update PR description with this data and mark as not humanly resolvable
	}
	if err := gitrepo.Commit(ctx, repo, releaseNotes); err != nil {
		slog.Info(fmt.Sprintf("Received error trying to commit: '%s'", err))
		return err
		//TODO update PR description with this data and mark as not humanly resolvable
	}
	return nil
}

// TODO: update with actual logic
func createReleaseNotes(library *statepb.LibraryReleaseState, commitMessages []string, inputDirectory string, releaseVersion string) (string, error) {
	var releaseNotes string

	for _, commitMessage := range commitMessages {
		releaseNotes += fmt.Sprintf("%s\n", commitMessage)
	}

	path := filepath.Join(inputDirectory, fmt.Sprintf("%s-%s-release-notes.txt", library.Id, releaseVersion))

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	_, err = file.WriteString(releaseNotes)
	if err != nil {
		return "", err
	}
	return releaseNotes, nil
}

func calculateNextVersion(library *statepb.LibraryReleaseState) (string, error) {
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

func updateLibraryMetadata(libraryId string, releaseVersion string, lastGeneratedCommit string, pipelineState *statepb.PipelineState) {
	for i, library := range pipelineState.LibraryReleaseStates {
		if library.Id == libraryId {
			pipelineState.LibraryReleaseStates[i].CurrentVersion = releaseVersion
			for _, apis := range library.Apis {
				//TODO is this logic correct? we update all apis in library with same commit id?
				for i := 0; i < len(pipelineState.ApiGenerationStates); i++ {
					apiGeneratedState := pipelineState.ApiGenerationStates[i]
					if apiGeneratedState.Id == apis.ApiId {
						pipelineState.ApiGenerationStates[i].LastGeneratedCommit = lastGeneratedCommit
					}
				}
			}
			break
		}
	}
}

func isReleaseWorthy(messages []string) bool {
	for _, str := range messages {
		if strings.Contains(strings.ToLower(str), "feat") {
			return true
		}
	}
	return false
}
