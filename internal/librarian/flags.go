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
	"flag"
	"fmt"

	"github.com/googleapis/librarian/internal/config"
)

func addFlagAPI(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.API, "api", "", "path to the API to be configured/generated (e.g., google/cloud/functions/v2)")
}

func addFlagBuild(fs *flag.FlagSet, cfg *config.Config) {
	fs.BoolVar(&cfg.Build, "build", false, "whether to build the generated code")
}

func addFlagHostMount(fs *flag.FlagSet, cfg *config.Config) {
	defaultValue := ""
	fs.StringVar(&cfg.HostMount, "host-mount", defaultValue, "a mount point from Docker host and within the Docker. The format is {host-dir}:{local-dir}.")
}

func addFlagPushConfig(fs *flag.FlagSet, cfg *config.Config) {
	// TODO(https://github.com/googleapis/librarian/issues/724):remove the default for push-config
	defaultValue := "noreply-cloudsdk@google.com,Google Cloud SDK"
	fs.StringVar(&cfg.PushConfig, "push-config", defaultValue, "If specified, will try to create a commit and pull request for the generated changes. The format should be \"{git-email-address},{author-name}\". Also, when this field is specified it is expected a Github token will be provided with push access via the environment variable LIBRARIAN_GITHUB_TOKEN")
}

func addFlagImage(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Image, "image", "", "Container image to run for subcommands. Defaults to the image in the pipeline state.")
}

func addFlagProject(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Project, "project", "", "Project containing Secret Manager secrets.")
}

func addFlagRepo(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Repo, "repo", "",
		"Code repository where the generated code will reside. "+
			"Can be a remote in the format of a remote URL such as "+
			"https://github.com/{owner}/{repo} or a local file path like "+
			"/path/to/repo`. Both absolute and relative paths are supported.")
}

func addFlagSource(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Source, "source", "", "location of googleapis repository. If undefined, googleapis will be cloned to the output")
}

func addFlagWorkRoot(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.WorkRoot, "output", "", "Working directory root. When this is not specified, a working directory will be created in /tmp.")
}

// validateRequiredFlag validates that the flag with the given name has been provided.
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

// validatePushConfigAndGithubTokenCoexist validates that the github token should exist if pushConfig flag is set.
func validatePushConfigAndGithubTokenCoexist(pushConfig, gitHubToken string) error {
	if pushConfig != "" && gitHubToken == "" {
		return fmt.Errorf("GitHub token is required if push config exists")
	}
	return nil
}
