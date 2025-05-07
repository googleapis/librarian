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
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/container"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/utils"
)

type LibraryRelease struct {
	LibraryID    string
	ReleaseID    string
	Version      string
	CommitHash   string
	ReleaseNotes string
}

var CmdCreateReleaseArtifacts = &Command{
	Name:  "create-release-artifacts",
	Short: "Create release artifacts from a merged release PR",
	flagFunctions: []func(fs *flag.FlagSet){
		addFlagImage,
		addFlagWorkRoot,
		addFlagLanguage,
		addFlagRepoRoot,
		addFlagRepoUrl,
		addFlagReleaseID,
	},
	maybeGetLanguageRepo: cloneOrOpenLanguageRepo,
	execute:              createReleaseArtifactsImpl,
}

func createReleaseArtifactsImpl(ctx *CommandContext) error {
	if err := validateRequiredFlag("release-id", flagReleaseID); err != nil {
		return err
	}

	outputRoot := filepath.Join(ctx.workRoot, "output")
	if err := os.Mkdir(outputRoot, 0755); err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Packages will be created in %s", outputRoot))

	releases, err := parseCommitsForReleases(ctx.languageRepo, flagReleaseID)
	if err != nil {
		return err
	}

	for _, release := range releases {
		if err := buildTestPackageRelease(ctx, outputRoot, release); err != nil {
			return err
		}
	}

	// Save the release details in the output directory, so that's all way need later.
	releasesJson, err := json.Marshal(releases)
	if err != nil {
		return err
	}
	utils.CreateAndWriteBytesToFile(filepath.Join(outputRoot, "releases.json"), releasesJson)

	slog.Info(fmt.Sprintf("Release artifact creation complete. Artifact root: %s", outputRoot))
	return nil
}

func buildTestPackageRelease(ctx *CommandContext, outputRoot string, release LibraryRelease) error {
	containerConfig := ctx.containerConfig
	languageRepo := ctx.languageRepo

	if err := gitrepo.Checkout(languageRepo, release.CommitHash); err != nil {
		return err
	}
	if err := container.BuildLibrary(containerConfig, languageRepo.Dir, release.LibraryID); err != nil {
		return err
	}
	if err := container.IntegrationTestLibrary(containerConfig, languageRepo.Dir, release.LibraryID); err != nil {
		return err
	}
	outputDir := filepath.Join(outputRoot, release.LibraryID)
	if err := os.Mkdir(outputDir, 0755); err != nil {
		return err
	}
	if err := container.PackageLibrary(containerConfig, languageRepo.Dir, release.LibraryID, outputDir); err != nil {
		return err
	}
	return nil
}

func parseCommitsForReleases(repo *gitrepo.Repo, releaseID string) ([]LibraryRelease, error) {
	commits, err := gitrepo.GetCommitsForReleaseID(repo, releaseID)
	if err != nil {
		return nil, err
	}
	releases := []LibraryRelease{}
	for _, commit := range commits {
		release, err := parseCommitMessageForRelease(commit.Message, commit.Hash.String())
		if err != nil {
			return nil, err
		}
		releases = append(releases, *release)
	}
	return releases, nil
}

func parseCommitMessageForRelease(message, hash string) (*LibraryRelease, error) {
	messageLines := strings.Split(message, "\n")

	// Remove the expected "title and blank line" (as we'll have a release title).
	// We're fairly conservative about this - if the commit message has been manually
	// changed, we'll leave it as it is.
	if len(messageLines) > 0 && strings.HasPrefix(messageLines[0], "Release library:") {
		messageLines = messageLines[1:]
		if len(messageLines) > 0 && messageLines[0] == "" {
			messageLines = messageLines[1:]
		}
	}

	libraryID, err := findMetadataValue("Librarian-Release-Library", messageLines)
	if err != nil {
		return nil, err
	}
	version, err := findMetadataValue("Librarian-Release-Version", messageLines)
	if err != nil {
		return nil, err
	}
	releaseID, err := findMetadataValue("Librarian-Release-ID", messageLines)
	if err != nil {
		return nil, err
	}
	releaseNotesLines := []string{}
	for _, line := range messageLines {
		if !strings.HasPrefix(line, "Librarian-Release") {
			releaseNotesLines = append(releaseNotesLines, line)
		}
	}
	releaseNotes := strings.Join(releaseNotesLines, "\n")
	return &LibraryRelease{
		LibraryID:    libraryID,
		Version:      version,
		ReleaseNotes: releaseNotes,
		CommitHash:   hash,
		ReleaseID:    releaseID,
	}, nil
}

func findMetadataValue(key string, lines []string) (string, error) {
	prefix := key + ": "
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			return line[len(prefix):], nil
		}
	}
	return "", fmt.Errorf("unable to find metadata value for key '%s'", key)
}
