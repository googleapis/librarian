// Copyright 2024 Google LLC
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
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

func addFlagAPI(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.API, "api", "", "path to the API to be configured/generated (e.g., google/cloud/functions/v2)")
}

func addFlagSource(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Source, "source", "", "location of googleapis repository. If undefined, googleapis will be cloned to the output")
}

func addFlagArtifactRoot(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.ArtifactRoot, "artifact-root", "", "Path to root of release artifacts to publish (as created by create-release-artifacts)")
}

func addFlagBaselineCommit(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.BaselineCommit, "baseline-commit", "", "the commit hash that was at HEAD for the language repo when create-release-pr was run")
}

func addFlagBranch(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Branch, "branch", "main", "repository branch")
}

func addFlagBuild(fs *flag.FlagSet, cfg *config.Config) {
	fs.BoolVar(&cfg.Build, "build", false, "whether to build the generated code")
}

func addFlagEnvFile(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.EnvFile, "env-file", "", "full path to the file where the environment variables are stored. Defaults to env-vars.txt within the output")
}

func addFlagGitUserEmail(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.GitUserEmail, "git-user-email", "noreply-cloudsdk@google.com", "Email address to use in Git commits")
}

func addFlagGitUserName(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.GitUserName, "git-user-name", "Google Cloud SDK", "Display name to use in Git commits")
}

func addFlagImage(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Image, "image", "", "Container image to run for subcommands. Defaults to the image in the pipeline state.")
}

func addFlagLibraryID(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.LibraryID, "library-id", "", "The ID of a single library to update")
}

func addFlagLibraryVersion(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.LibraryVersion, "library-version", "", "The version to release (only valid with library-id, only when creating a release PR)")
}

func addFlagReleaseID(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.ReleaseID, "release-id", "", "The ID of a release PR")
}

func addFlagReleasePRUrl(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.ReleasePRURL, "release-pr-url", "", "The URL of a release PR")
}

func addFlagRepo(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Repo, "repo", "", "Repository root or URL to clone. If this is not specified, the default language repo will be cloned.")
}

func addFlagProject(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Project, "project", "", "Project containing Secret Manager secrets.")
}

func addFlagSkipIntegrationTests(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.SkipIntegrationTests, "skip-integration-tests", "", "set to a value of b/{explanatory-bug} to skip integration tests")
}

func addFlagSyncUrlPrefix(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.SyncURLPrefix, "sync-url-prefix", "", "the prefix of the URL to check for commit synchronization; the commit hash will be appended to this")
}

func addFlagTag(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Tag, "tag", "", "new tag for the language-specific container image.")
}

func addFlagTagRepoUrl(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.TagRepoURL, "tag-repo-url", "", "Repository URL to tag and create releases in. Requires when push is true.")
}

func addFlagWorkRoot(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.WorkRoot, "output", "", "Working directory root. When this is not specified, a working directory will be created in /tmp.")
}

func validateSkipIntegrationTests(skipIntegrationTests string) error {
	if skipIntegrationTests != "" && !strings.HasPrefix(skipIntegrationTests, "b/") {
		return errors.New("skipping integration tests requires a bug to be specified, e.g. -skip-integration-tests=b/12345")
	}
	return nil
}

// Validate that the flag with the given name has been provided.
// TODO(https://github.com/googleapis/librarian/issues/488): add support for required string flags
// We should rework how we add flags so that these can be validated before we even
// start executing the command. (At least for simple cases where a flag is required;
// note that this isn't always going to be the same for all commands for one flag.)
func validateRequiredFlag(name, value string) error {
	if value == "" {
		return fmt.Errorf("required flag -%s not specified", name)
	}
	return nil
}
