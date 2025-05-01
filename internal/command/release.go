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
	"strings"

	"github.com/googleapis/librarian/internal/container"
	"github.com/googleapis/librarian/internal/githubrepo"
	"github.com/googleapis/librarian/internal/gitrepo"
)

type LibraryRelease struct {
	LibraryID    string
	ReleaseID    string
	Version      string
	CommitHash   string
	ReleaseNotes string
}

var CmdRelease = &Command{
	Name:  "release",
	Short: "Release libraries from a merged release PR",
	flagFunctions: []func(fs *flag.FlagSet){
		addFlagImage,
		addFlagWorkRoot,
		addFlagLanguage,
		addFlagPush,
		addFlagRepoRoot,
		addFlagRepoUrl,
		addFlagReleaseID,
		addFlagTagRepoUrl,
	},
	maybeGetLanguageRepo: cloneOrOpenLanguageRepo,
	execute: func(ctx *CommandContext) error {
		if err := validateRequiredFlag("release-id", flagReleaseID); err != nil {
			return err
		}

		if err := validatePush(); err != nil {
			return err
		}

		if flagPush && flagTagRepoUrl == "" {
			return errors.New("flag -tag-repo-url is required when -push is true")
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

		if flagPush {
			if err := publishPackages(ctx.containerConfig, outputRoot, releases); err != nil {
				return err
			}
			if err := createRepoReleases(ctx, releases); err != nil {
				return err
			}
		} else {
			slog.Info("Push flag not specified; not publishing packages")
		}
		slog.Info("Release complete.")

		return nil
	},
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

func publishPackages(config *container.ContainerConfig, outputRoot string, releases []LibraryRelease) error {
	for _, release := range releases {
		outputDir := filepath.Join(outputRoot, release.LibraryID)
		if err := container.PublishLibrary(config, outputDir, release.LibraryID, release.Version); err != nil {
			return err
		}
	}
	slog.Info("All packages published.")
	return nil
}

func createRepoReleases(ctx *CommandContext, releases []LibraryRelease) error {
	repoUrl := flagTagRepoUrl

	gitHubRepo, err := githubrepo.ParseUrl(repoUrl)
	if err != nil {
		return err
	}

	for _, release := range releases {
		tag := formatReleaseTag(release.LibraryID, release.Version)
		title := fmt.Sprintf("%s version %s", release.LibraryID, release.Version)
		prerelease := strings.HasPrefix(release.Version, "0.") || strings.Contains(release.Version, "-")
		repoRelease, err := githubrepo.CreateRelease(ctx.ctx, gitHubRepo, tag, release.CommitHash, title, release.ReleaseNotes, prerelease)
		if err != nil {
			return err
		}
		slog.Info(fmt.Sprintf("Created repo release '%s' with tag '%s'", *repoRelease.Name, *repoRelease.TagName))
	}
	slog.Info("All repo releases created.")
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
