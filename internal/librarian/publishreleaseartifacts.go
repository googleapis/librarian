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
	"flag"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/container"
	"github.com/googleapis/librarian/internal/githubrepo"
	"github.com/googleapis/librarian/internal/utils"
)

var CmdPublishReleaseArtifacts = &cli.Command{
	Name:  "publish-release-artifacts",
	Short: "Publish (previously-created) release artifacts to package managers.",
	Run:   runPublishReleaseArtifacts,
	FlagFunctions: []func(fs *flag.FlagSet){
		addFlagArtifactRoot,
		addFlagImage,
		addFlagWorkRoot,
		addFlagLanguage,
		addFlagSecretsProject,
		addFlagTagRepoUrl,
	},
}

func runPublishReleaseArtifacts(ctx context.Context) error {
	// Load the state and config from the artifact directory. These will have been created by create-release-artifacts.
	ps, err := loadPipelineStateFile(filepath.Join(flagArtifactRoot, pipelineStateFile))
	if err != nil {
		return err
	}

	config, err := loadPipelineConfigFile(filepath.Join(flagArtifactRoot, pipelineConfigFile))
	if err != nil {
		return err
	}
	image := deriveImage(ps)

	startTime := time.Now()
	workRoot, err := createWorkRoot(startTime)
	if err != nil {
		return err
	}

	containerConfig, err := container.NewContainerConfig(ctx, workRoot, image, flagSecretsProject, config)
	if err != nil {
		return err
	}
	return publishReleaseArtifacts(ctx, containerConfig)
}

func publishReleaseArtifacts(ctx context.Context, containerConfig *container.ContainerConfig) error {
	if err := validateRequiredFlag("artifact-root", flagArtifactRoot); err != nil {
		return err
	}

	if err := validateRequiredFlag("tag-repo-url", flagTagRepoUrl); err != nil {
		return err
	}

	releasesJson, err := utils.ReadAllBytesFromFile(filepath.Join(flagArtifactRoot, "releases.json"))
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
	gitHubRepo, err := githubrepo.ParseUrl(flagTagRepoUrl)
	if err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Publishing packages for %d libraries", len(releases)))

	if err := publishPackages(containerConfig, flagArtifactRoot, releases); err != nil {
		return err
	}
	if err := createRepoReleases(ctx, releases, gitHubRepo); err != nil {
		return err
	}
	slog.Info("Release complete.")

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

func createRepoReleases(ctx context.Context, releases []LibraryRelease, gitHubRepo githubrepo.GitHubRepo) error {
	for _, release := range releases {
		tag := formatReleaseTag(release.LibraryID, release.Version)
		title := fmt.Sprintf("%s version %s", release.LibraryID, release.Version)
		prerelease := strings.HasPrefix(release.Version, "0.") || strings.Contains(release.Version, "-")
		repoRelease, err := githubrepo.CreateRelease(ctx, gitHubRepo, tag, release.CommitHash, title, release.ReleaseNotes, prerelease)
		if err != nil {
			return err
		}
		slog.Info(fmt.Sprintf("Created repo release '%s' with tag '%s'", *repoRelease.Name, *repoRelease.TagName))
	}
	slog.Info("All repo releases created.")
	return nil
}
