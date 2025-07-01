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
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
)

// LibraryRelease contains information about the release of a single
// library.
type LibraryRelease struct {
	// The ID of the library being released.
	LibraryID string
	// The release ID of the PR containing this release
	// (as specified in commit messages and the PR description).
	ReleaseID string
	// The version of the library being released.
	Version string
	// The hash of the commit which should be tagged when the library is released.
	CommitHash string
	// The release notes for the library release, to be included in the GitHub
	// release.
	ReleaseNotes string
}

var cmdCreateReleaseArtifacts = &cli.Command{
	Short:     "create-release-artifacts creates release artifacts from a merged release PR",
	UsageLine: "librarian create-release-artifacts -release-id=<id> [flags]",
	Long: `Specify the release ID, and optional flags to use non-default repositories, e.g. for testing.
The release ID is specified in the the release PR and in each commit within it, in a line starting "Librarian-Release-ID: ".

After acquiring the language repository, the repository is scanned backwards from the head commit to find
commits belonging to the single release for which the command is creating artifacts. The head commit is not required to
belong to the release, but the commits for the release are expected to be contiguous.
(In other words, once the command has found a commit that *is* part of the release, when it encounters a commit which
*isn't* part of the release, it assumes it's found all the relevant commits.) If this phase doesn't find any commits,
the command fails.

The command creates a root output folder for all the release artifacts.

The commits are examined to determine the libraries which are being released. For each library, the following steps
are taken:
- The language repository is checked out at the commit associated with that library's release.
- The container commands of "build-library", "integration-test-library" and "package-library" are run. The last
  of these places the release artifacts in an empty folder created by the CLI command. (Each library has a separate
  subfolder of the root output folder.)

Finally, metadata files for the Librarian state and config, and the libraries that are being released, is copied
into the root output folder for use in the "publish-release-artifacts" command.

This command does not create any pull requests. Any failure is considered fatal for this command: if one library
fails its integration tests for example, the whole job fails. This is to avoid a situation where a release is half-published.
The command can safely be rerun, e.g. if a service outage caused a failure and the release can be expected to succeed
if retried.
`,
	Run: runCreateReleaseArtifacts,
}

func init() {
	cmdCreateReleaseArtifacts.Init()
	fs := cmdCreateReleaseArtifacts.Flags
	cfg := cmdCreateReleaseArtifacts.Config

	addFlagImage(fs, cfg)
	addFlagProject(fs, cfg)
	addFlagReleaseID(fs, cfg)
	addFlagRepo(fs, cfg)
	addFlagSkipIntegrationTests(fs, cfg)
	addFlagWorkRoot(fs, cfg)
}

func runCreateReleaseArtifacts(ctx context.Context, cfg *config.Config) error {
	state, err := createCommandStateForLanguage(cfg.WorkRoot, cfg.Repo,
		cfg.Image, cfg.Project, cfg.CI, cfg.UserUID, cfg.UserGID)
	if err != nil {
		return err
	}
	return createReleaseArtifactsImpl(ctx, state, cfg)
}

func createReleaseArtifactsImpl(ctx context.Context, state *commandState, cfg *config.Config) error {
	if err := validateSkipIntegrationTests(cfg.SkipIntegrationTests); err != nil {
		return err
	}
	if err := validateRequiredFlag("release-id", cfg.ReleaseID); err != nil {
		return err
	}
	outputRoot := filepath.Join(state.workRoot, "output")
	if err := os.Mkdir(outputRoot, 0755); err != nil {
		return err
	}
	slog.Info("Packages will be created", "dir", outputRoot)

	releases, err := parseCommitsForReleases(state.languageRepo, cfg.ReleaseID)
	if err != nil {
		return err
	}

	for _, release := range releases {
		if err := buildTestPackageRelease(ctx, state, cfg, outputRoot, release); err != nil {
			return err
		}
	}

	if err := copyMetadataFiles(state, outputRoot, releases); err != nil {
		return err
	}

	slog.Info("Release artifact creation complete", "artifact_root", outputRoot)
	return nil
}

// The publish-release-artifacts stage will need bits of metadata:
// - The releases we're creating
// - The pipeline config
// - (Just in case) The pipeline state
// The pipeline config and state files are copied by checking out the commit of the last
// release, which should effectively be the tip of the release PR.
func copyMetadataFiles(state *commandState, outputRoot string, releases []LibraryRelease) error {
	releasesJson, err := json.Marshal(releases)
	if err != nil {
		return err
	}
	if err := createAndWriteBytesToFile(filepath.Join(outputRoot, "releases.json"), releasesJson); err != nil {
		return err
	}

	languageRepo := state.languageRepo
	finalRelease := releases[len(releases)-1]
	if err := languageRepo.Checkout(finalRelease.CommitHash); err != nil {
		return err
	}
	sourceStateFile := filepath.Join(languageRepo.Dir, config.GeneratorInputDir, pipelineStateFile)
	destStateFile := filepath.Join(outputRoot, pipelineStateFile)
	if err := copyFile(sourceStateFile, destStateFile); err != nil {
		return err
	}

	sourceConfigFile := filepath.Join(languageRepo.Dir, config.GeneratorInputDir, pipelineConfigFile)
	destConfigFile := filepath.Join(outputRoot, pipelineConfigFile)
	if err := copyFile(sourceConfigFile, destConfigFile); err != nil {
		return err
	}
	return nil
}

func copyFile(sourcePath, destPath string) error {
	bytes, err := readAllBytesFromFile(sourcePath)
	if err != nil {
		return err
	}
	return createAndWriteBytesToFile(destPath, bytes)
}

func buildTestPackageRelease(ctx context.Context, state *commandState, cfg *config.Config, outputRoot string, release LibraryRelease) error {
	cc := state.containerConfig
	languageRepo := state.languageRepo

	if err := languageRepo.Checkout(release.CommitHash); err != nil {
		return err
	}
	if err := cc.BuildLibrary(ctx, cfg, languageRepo.Dir, release.LibraryID); err != nil {
		return err
	}
	if cfg.SkipIntegrationTests != "" {
		slog.Info("Skipping integration tests", "bug", cfg.SkipIntegrationTests)
	} else if err := cc.IntegrationTestLibrary(ctx, cfg, languageRepo.Dir, release.LibraryID); err != nil {
		return err
	}
	outputDir := filepath.Join(outputRoot, release.LibraryID)
	if err := os.Mkdir(outputDir, 0755); err != nil {
		return err
	}
	if err := cc.PackageLibrary(ctx, cfg, languageRepo.Dir, release.LibraryID, outputDir); err != nil {
		return err
	}
	return nil
}

func parseCommitsForReleases(repo *gitrepo.Repository, releaseID string) ([]LibraryRelease, error) {
	commits, err := repo.GetCommitsForReleaseID(releaseID)
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
	if len(messageLines) > 0 && strings.HasPrefix(messageLines[0], "chore: Release library ") {
		messageLines = messageLines[1:]
		if len(messageLines) > 0 && messageLines[0] == "" {
			messageLines = messageLines[1:]
		}
	}

	libraryID, err := findMetadataValue("Librarian-Release-Library", messageLines, hash)
	if err != nil {
		return nil, err
	}
	version, err := findMetadataValue("Librarian-Release-Version", messageLines, hash)
	if err != nil {
		return nil, err
	}
	releaseID, err := findMetadataValue("Librarian-Release-ID", messageLines, hash)
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

func findMetadataValue(key string, lines []string, hash string) (string, error) {
	prefix := key + ": "
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			return line[len(prefix):], nil
		}
	}
	return "", fmt.Errorf("unable to find metadata value for key '%s' in commit %s", key, hash)
}
