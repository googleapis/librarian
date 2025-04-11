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

package command

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

// Environment variables are specified here as they're used for the same sort of purpose as flags...
const gitHubTokenEnvironmentVariable string = "LIBRARIAN_GITHUB_TOKEN"
const defaultRepositoryEnvironmentVariable string = "LIBRARIAN_REPOSITORY"

var (
	flagAPIPath        string
	flagAPIRoot        string
	flagBranch         string
	flagBuild          bool
	flagEnvFile        string
	flagGitUserEmail   string
	flagGitUserName    string
	flagImage          string
	flagLanguage       string
	flagLibraryID      string
	flagPush           bool
	flagReleaseID      string
	flagRepoRoot       string
	flagRepoUrl        string
	flagSecretsProject string
	flagSkipBuild      bool
	flagTag            string
	flagTagRepoUrl     string
	flagWorkRoot       string
)

func addFlagAPIPath(fs *flag.FlagSet) {
	fs.StringVar(&flagAPIPath, "api-path", "", "(Required) path api-root to the API to be generated (e.g., google/cloud/functions/v2)")
}

func addFlagAPIRoot(fs *flag.FlagSet) {
	fs.StringVar(&flagAPIRoot, "api-root", "", "location of googleapis repository. If undefined, googleapis will be cloned to the work-root")
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
	fs.StringVar(&flagLanguage, "language", "", "(Required) language to generate code for")
}

func addFlagLibraryID(fs *flag.FlagSet) {
	fs.StringVar(&flagLibraryID, "library-id", "", "The ID of a single library to update")
}

func addFlagPush(fs *flag.FlagSet) {
	fs.BoolVar(&flagPush, "push", false, "push to GitHub if true")
}

func addFlagReleaseID(fs *flag.FlagSet) {
	fs.StringVar(&flagReleaseID, "release-id", "", "The ID of a release PR")
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

func addFlagSkipBuild(fs *flag.FlagSet) {
	fs.BoolVar(&flagSkipBuild, "skipBuild", false, "when create release PR if this is set to true do not perform build/integration tests")
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

var supportedLanguages = map[string]bool{
	"cpp":    false,
	"dotnet": true,
	"go":     false,
	"java":   false,
	"node":   false,
	"php":    false,
	"python": false,
	"ruby":   false,
	"rust":   false,
	"all":    false,
}

func validatePush() error {
	if flagPush && os.Getenv(gitHubTokenEnvironmentVariable) == "" {
		return errors.New("no GitHub token supplied for push")
	}
	return nil
}

func validateLanguage() error {
	if !supportedLanguages[flagLanguage] {
		return fmt.Errorf("invalid -language flag specified: %q", flagLanguage)
	}
	return nil
}

// Validate that the flag with the given name has been provided.
// TODO: Rework how we add flags so that these can be validated before we even
// start executing the command. (At least for simple cases where a flag is required;
// note that this isn't always going to be the same for all commands for one flag.)
func validateRequiredFlag(name, value string) error {
	if value == "" {
		return fmt.Errorf("required flag -%s not specified", name)
	}
	return nil
}
