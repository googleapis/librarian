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

func addFlagSource(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Source, "source", "", "location of googleapis repository. If undefined, googleapis will be cloned to the output")
}

func addFlagBranch(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Branch, "branch", "main", "repository branch")
}

func addFlagBuild(fs *flag.FlagSet, cfg *config.Config) {
	fs.BoolVar(&cfg.Build, "build", false, "whether to build the generated code")
}

func addFlagPushConfig(fs *flag.FlagSet, cfg *config.Config) {
	// TODO(https://github.com/googleapis/librarian/issues/724):remove the default for push-config
	fs.StringVar(&cfg.PushConfig, "push-config", "noreply-cloudsdk@google.com,Google Cloud SDK", "The user and email for Git commits, in the format \"user:email\"")
}

func addFlagImage(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Image, "image", "", "Container image to run for subcommands. Defaults to the image in the pipeline state.")
}

func addFlagLibraryID(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.LibraryID, "library-id", "", "The ID of a single library to update")
}

func addFlagRepo(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Repo, "repo", "", "Repository root or URL to clone. If this is not specified, the default language repo will be cloned.")
}

func addFlagProject(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Project, "project", "", "Project containing Secret Manager secrets.")
}

func addFlagTag(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Tag, "tag", "", "new tag for the language-specific container image.")
}

func addFlagWorkRoot(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.WorkRoot, "output", "", "Working directory root. When this is not specified, a working directory will be created in /tmp.")
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
