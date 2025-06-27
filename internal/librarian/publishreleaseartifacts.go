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
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/github"
)

var cmdPublishReleaseArtifacts = &cli.Command{
	Short:     "publish-release-artifacts publishes (previously-created) release artifacts to package managers and documentation sites",
	UsageLine: "librarian publish-release-artifacts -language=<language> -artifact-root=<artifact-root> -tag-repo-url=<repo-url> [flags]",
	Long: `Specify the language, the root output directory created by create-release-artifacts, and
the GitHub repository in which to create tags/releases.

The command first loads the metadata created by create-release-artifacts. This
includes all the relevant state and configuration, as well as which libraries are being released (including
the version number, release notes, and the commit to tag for each library).

The command iterates over the libraries being released, calling the language container "publish-library"
command for each library, passing in the directory in which the artifacts for that library have been created.

The command then iterates over all the libraries again, creating tags with appropriate release notes in
GitHub.

If any operation fails, the whole command fails immediately. This means that on failure we can have
inconsistent states of:
- Some packages being published but not all
- All packages being published, but not all tags being created (potentially none)

However, if *any* tags are created, that means *all* packages have already been published. If package publication
for the language being released ignores republication errors, and if no tags have been created,
the command can be rerun to resolve partial publication. (Currently the command will fail if it attempts to
create a tag which already exists.)
`,
	Run: runPublishReleaseArtifacts,
}

func init() {
	cmdPublishReleaseArtifacts.Init()
	fs := cmdPublishReleaseArtifacts.Flags
	cfg := cmdPublishReleaseArtifacts.Config

	addFlagArtifactRoot(fs, cfg)
	addFlagImage(fs, cfg)
	addFlagWorkRoot(fs, cfg)
	addFlagLanguage(fs, cfg)
	addFlagSecretsProject(fs, cfg)
	addFlagTagRepoUrl(fs, cfg)
}

func runPublishReleaseArtifacts(ctx context.Context, cfg *config.Config) error {
	if err := validateRequiredFlag("artifact-root", cfg.ArtifactRoot); err != nil {
		return err
	}
	// Load the state and config from the artifact directory. These will have been created by create-release-artifacts.
	ps, err := loadPipelineStateFile(filepath.Join(cfg.ArtifactRoot, pipelineStateFile))
	if err != nil {
		return err
	}

	pipelineConfig, err := loadPipelineConfigFile(filepath.Join(cfg.ArtifactRoot, pipelineConfigFile))
	if err != nil {
		return err
	}

	image := deriveImage(cfg.Language, cfg.Image, cfg.LibrarianRepository, ps)

	startTime := time.Now()
	workRoot, err := createWorkRoot(startTime, cfg.WorkRoot)
	if err != nil {
		return err
	}

	containerConfig, err := docker.New(workRoot, image, cfg.SecretsProject, cfg.UserUID, cfg.UserGID, pipelineConfig)
	if err != nil {
		return err
	}
	return publishReleaseArtifacts(ctx, containerConfig, cfg)
}

func publishReleaseArtifacts(ctx context.Context, containerConfig *docker.Docker, cfg *config.Config) error {
	if err := validateRequiredFlag("tag-repo-url", cfg.TagRepoURL); err != nil {
		return err
	}

	releasesJson, err := readAllBytesFromFile(filepath.Join(cfg.ArtifactRoot, "releases.json"))
	if err != nil {
		return err
	}
	var releases []LibraryRelease
	if err := json.Unmarshal(releasesJson, &releases); err != nil {
		return err
	}

	if len(releases) == 0 {
		return errors.New("no releases to publish")
	}

	// Load the pipeline config from the commit of the first release, using the tag repo, then
	// update our context to use it for the container config.
	gitHubRepo, err := github.ParseUrl(cfg.TagRepoURL)
	if err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Publishing packages for %d libraries", len(releases)))

	if err := publishPackages(ctx, containerConfig, cfg, releases); err != nil {
		return err
	}
	if err := createRepoReleases(ctx, releases, gitHubRepo, cfg.GitHubToken); err != nil {
		return err
	}
	slog.Info("Release complete.")

	return nil
}

func publishPackages(ctx context.Context, config *docker.Docker, cfg *config.Config, releases []LibraryRelease) error {
	for _, release := range releases {
		outputDir := filepath.Join(cfg.ArtifactRoot, release.LibraryID)
		if err := config.PublishLibrary(ctx, cfg, outputDir, release.LibraryID, release.Version); err != nil {
			return err
		}
	}
	slog.Info("All packages published.")
	return nil
}

func createRepoReleases(ctx context.Context, releases []LibraryRelease, gitHubRepo *github.Repository, gitHubToken string) error {
	ghClient, err := github.NewClient(gitHubToken)
	if err != nil {
		return err
	}
	for _, release := range releases {
		tag := formatReleaseTag(release.LibraryID, release.Version)
		title := fmt.Sprintf("%s version %s", release.LibraryID, release.Version)
		prerelease := strings.HasPrefix(release.Version, "0.") || strings.Contains(release.Version, "-")
		repoRelease, err := ghClient.CreateRelease(ctx, gitHubRepo, tag, release.CommitHash, title, release.ReleaseNotes, prerelease)
		if err != nil {
			return err
		}
		slog.Info(fmt.Sprintf("Created repo release '%s' with tag '%s'", *repoRelease.Name, *repoRelease.TagName))
	}
	slog.Info("All repo releases created.")
	return nil
}
