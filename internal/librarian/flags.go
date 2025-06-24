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

// Environment variables are specified here as they're used for the same sort of purpose as flags...
// ... but see also githubrepo.go
const defaultRepositoryEnvironmentVariable string = "LIBRARIAN_REPOSITORY"

var (
	flagAPIPath              string
	flagAPIRoot              string
	flagArtifactRoot         string
	flagBaselineCommit       string
	flagBranch               string
	flagBuild                bool
	flagEnvFile              string
	flagGitUserEmail         string
	flagGitUserName          string
	flagImage                string
	flagLanguage             string
	flagLibraryID            string
	flagLibraryVersion       string
	flagPush                 bool
	flagReleaseID            string
	flagReleasePRUrl         string
	flagRepoRoot             string
	flagRepoUrl              string
	flagSyncUrlPrefix        string
	flagSecretsProject       string
	flagSkipIntegrationTests string
	flagTag                  string
	flagTagRepoUrl           string
	flagWorkRoot             string
)

func addFlagAPIPath(fs *flag.FlagSet) {
	fs.StringVar(&flagAPIPath, "api-path", "", "path to the API to be configured/generated (e.g., google/cloud/functions/v2)")
}

func addFlagAPIRoot(fs *flag.FlagSet) {
	fs.StringVar(&flagAPIRoot, "api-root", "", "location of googleapis repository. If undefined, googleapis will be cloned to the work-root")
}

func addFlagArtifactRoot(fs *flag.FlagSet) {
	fs.StringVar(&flagArtifactRoot, "artifact-root", "", "Path to root of release artifacts to publish (as created by create-release-artifacts)")
}
func addFlagBaselineCommit(fs *flag.FlagSet) {
	fs.StringVar(&flagBaselineCommit, "baseline-commit", "", "the commit hash that was at HEAD for the language repo when create-release-pr was run")
}

func addFlagBranch(fs *flag.FlagSet) {
	fs.StringVar(&flagBranch, "branch", "main", "repository branch")
}

func addFlagBuild(fs *flag.FlagSet) {
	fs.BoolVar(&flagBuild, "build", false, "whether to build the generated code")
}

func addFlagEnvFile(fs *flag.FlagSet) {
	fs.StringVar(&flagEnvFile, "env-file", "", "full path to the file where the environment variables are stored. Defaults to env-vars.txt within the work-root")
}

func addFlagGitUserEmail(fs *flag.FlagSet) {
	fs.StringVar(&flagGitUserEmail, "git-user-email", "", "Email address to use in Git commits")
}

func addFlagGitUserName(fs *flag.FlagSet) {
	fs.StringVar(&flagGitUserName, "git-user-name", "", "Display name to use in Git commits")
}

func addFlagImage(fs *flag.FlagSet) {
	fs.StringVar(&flagImage, "image", "", "language-specific container to run for subcommands. Defaults to google-cloud-{language}-generator")
}

func addFlagLanguage(fs *flag.FlagSet) {
	fs.StringVar(&flagLanguage, "language", "", "(Required) language for which to configure/generate/release code")
}

func addFlagLibraryID(fs *flag.FlagSet) {
	fs.StringVar(&flagLibraryID, "library-id", "", "The ID of a single library to update")
}

func addFlagLibraryVersion(fs *flag.FlagSet) {
	fs.StringVar(&flagLibraryVersion, "library-version", "", "The version to release (only valid with library-id, only when creating a release PR)")
}

func addFlagPush(fs *flag.FlagSet) {
	fs.BoolVar(&flagPush, "push", false, "push to GitHub if true")
}

func addFlagReleaseID(fs *flag.FlagSet) {
	fs.StringVar(&flagReleaseID, "release-id", "", "The ID of a release PR")
}

func addFlagReleasePRUrl(fs *flag.FlagSet) {
	fs.StringVar(&flagReleasePRUrl, "release-pr-url", "", "The URL of a release PR")
}

func addFlagRepoRoot(fs *flag.FlagSet) {
	fs.StringVar(&flagRepoRoot, "repo-root", "", "Repository root. When this (and repo-url) are not specified, the language repo will be cloned.")
}

func addFlagRepoUrl(fs *flag.FlagSet) {
	fs.StringVar(&flagRepoUrl, "repo-url", "", "Repository URL to clone. If this and repo-root are not specified, the default language repo will be cloned.")
}

func addFlagSecretsProject(fs *flag.FlagSet) {
	fs.StringVar(&flagSecretsProject, "secrets-project", "", "Project containing Secret Manager secrets.")
}

func addFlagSkipIntegrationTests(fs *flag.FlagSet) {
	fs.StringVar(&flagSkipIntegrationTests, "skip-integration-tests", "", "set to a value of b/{explanatory-bug} to skip integration tests")
}

func addFlagSyncUrlPrefix(fs *flag.FlagSet) {
	fs.StringVar(&flagSyncUrlPrefix, "sync-url-prefix", "", "the prefix of the URL to check for commit synchronization; the commit hash will be appended to this")
}

func addFlagTag(fs *flag.FlagSet) {
	fs.StringVar(&flagTag, "tag", "", "new tag for the language-specific container image.")
}

func addFlagTagRepoUrl(fs *flag.FlagSet) {
	fs.StringVar(&flagTagRepoUrl, "tag-repo-url", "", "Repository URL to tag and create releases in. Requires when push is true.")
}

func addFlagWorkRoot(fs *flag.FlagSet) {
	fs.StringVar(&flagWorkRoot, "work-root", "", "Working directory root. When this is not specified, a working directory will be created in /tmp.")
}

func validateSkipIntegrationTests() error {
	if flagSkipIntegrationTests != "" && !strings.HasPrefix(flagSkipIntegrationTests, "b/") {
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

func applyFlags(cfg *config.Config) {
	cfg.APIPath = flagAPIPath
	cfg.APIRoot = flagAPIRoot
	cfg.ArtifactRoot = flagArtifactRoot
	cfg.BaselineCommit = flagBaselineCommit
	cfg.Branch = flagBranch
	cfg.Build = flagBuild
	cfg.EnvFile = flagEnvFile
	cfg.GitUserEmail = flagGitUserEmail
	cfg.GitUserName = flagGitUserName
	cfg.Image = flagImage
	cfg.Language = flagLanguage
	cfg.LibraryID = flagLibraryID
	cfg.LibraryVersion = flagLibraryVersion
	cfg.Push = flagPush
	cfg.ReleaseID = flagReleaseID
	cfg.ReleasePRURL = flagReleasePRUrl
	cfg.RepoRoot = flagRepoRoot
	cfg.RepoURL = flagRepoUrl
	cfg.SyncURLPrefix = flagSyncUrlPrefix
	cfg.SecretsProject = flagSecretsProject
	cfg.SkipIntegrationTests = flagSkipIntegrationTests
	cfg.Tag = flagTag
	cfg.TagRepoURL = flagTagRepoUrl
	cfg.WorkRoot = flagWorkRoot
}
