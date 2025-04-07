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
)

type LibraryRelease struct {
	LibraryID    string
	Version      string
	CommitHash   string
	ReleaseNotes string
}

var CmdRelease = &Command{
	Name:  "release",
	Short: "Release libraries from a merged release PR",
	Run: func(ctx context.Context) error {
		if err := validateLanguage(); err != nil {
			return err
		}
		if err := validateRequiredFlag("release-id", flagReleaseID); err != nil {
			return err
		}
		if err := validateRequiredFlag("repo-root", flagRepoRoot); err != nil {
			return err
		}

		tmpRoot, err := createTmpWorkingRoot(time.Now())
		if err != nil {
			return err
		}

		outputRoot := filepath.Join(tmpRoot, "output")
		if err := os.Mkdir(outputRoot, 0755); err != nil {
			return err
		}
		slog.Info(fmt.Sprintf("Packages will be created in %s", outputRoot))

		repoRoot, err := filepath.Abs(flagRepoRoot)
		if err != nil {
			return err
		}
		languageRepo, err := gitrepo.Open(repoRoot)
		if err != nil {
			return err
		}
		clean, err := gitrepo.IsClean(languageRepo)
		if err != nil {
			return err
		}
		if !clean {
			return errors.New("language repo must be clean before releasing")
		}

		if flagImage == "" {
			pipelineState, err := loadState(languageRepo)
			if err != nil {
				slog.Info(fmt.Sprintf("Error loading pipeline state: %s", err))
				return err
			}
			flagImage = deriveImage(pipelineState)
		}

		releases, err := parseCommitsForReleases(languageRepo, flagReleaseID)
		if err != nil {
			return err
		}

		for _, release := range releases {
			if err := buildTestPackageRelease(flagImage, outputRoot, languageRepo, release); err != nil {
				return err
			}
		}

		if err := publishPackages(flagImage, outputRoot, releases); err != nil {
			return err
		}
		slog.Info("Release complete.")

		return nil
	},
}

func buildTestPackageRelease(image, outputRoot string, languageRepo *gitrepo.Repo, release LibraryRelease) error {
	if err := gitrepo.Checkout(languageRepo, release.CommitHash); err != nil {
		return err
	}
	if err := container.BuildLibrary(image, languageRepo.Dir, release.LibraryID); err != nil {
		return err
	}
	if err := container.IntegrationTestLibrary(image, languageRepo.Dir, release.LibraryID); err != nil {
		return err
	}
	outputDir := filepath.Join(outputRoot, release.LibraryID)
	if err := os.Mkdir(outputRoot, 0755); err != nil {
		return err
	}
	if err := container.PackageLibrary(image, languageRepo.Dir, release.LibraryID, outputDir); err != nil {
		return err
	}
	return nil
}

func publishPackages(image, outputRoot string, releases []LibraryRelease) error {
	for _, release := range releases {
		slog.Info(fmt.Sprintf("Would create GitHub release for %s", release.LibraryID))
	}
	slog.Info(fmt.Sprintf("Would public packages with image %s and output root %s", image, outputRoot))
	return errors.New("publishing releases isn't implemented yet")
}

func parseCommitsForReleases(repo *gitrepo.Repo, releaseID string) ([]LibraryRelease, error) {
	commits, err := gitrepo.GetCommitsForReleaseID(repo, releaseID)
	if err != nil {
		return nil, err
	}
	releases := []LibraryRelease{}
	for _, commit := range commits {
		release, err := parseCommitMessageForRelease(commit)
		if err != nil {
			return nil, err
		}
		releases = append(releases, *release)
	}
	return releases, nil
}

func parseCommitMessageForRelease(commit object.Commit) (*LibraryRelease, error) {
	messageLines := strings.Split(commit.Message, "\n")
	libraryID, err := findMetadataValue("Librarian-Release-Library", messageLines)
	if err != nil {
		return nil, err
	}
	version, err := findMetadataValue("Librarian-Release-Version", messageLines)
	if err != nil {
		return nil, err
	}
	releaseNotesLines := []string{}
	for _, line := range messageLines {
		if !strings.HasPrefix("Librarian-Release", line) {
			releaseNotesLines = append(releaseNotesLines, line)
		}
	}
	releaseNotes := strings.Join(releaseNotesLines, "\n")
	return &LibraryRelease{
		LibraryID:    libraryID,
		Version:      version,
		ReleaseNotes: releaseNotes,
		CommitHash:   commit.Hash.String(),
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
