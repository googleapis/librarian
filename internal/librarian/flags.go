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
	"github.com/googleapis/librarian/internal/config"
)

func addFlagAPI(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.API, "api", "", "path to the API to be configured/generated (e.g., google/cloud/functions/v2)")
}

func addFlagAPISource(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.APISource, "api-source", "https://github.com/googleapis/googleapis", "location of googleapis repository. If undefined, googleapis will be cloned to the output")
}

func addFlagBuild(fs *flag.FlagSet, cfg *config.Config) {
	fs.BoolVar(&cfg.Build, "build", false, "whether to build the generated code")
}

func addFlagHostMount(fs *flag.FlagSet, cfg *config.Config) {
	defaultValue := ""
	fs.StringVar(&cfg.HostMount, "host-mount", defaultValue, "a mount point from Docker host and within the Docker. The format is {host-dir}:{local-dir}.")
}

func addFlagImage(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Image, "image", "", "Container image to run for subcommands. Defaults to the image in the pipeline state.")
}

func addFlagLibrary(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Library, "library", "", "The ID of a single library to update")
}

func addFlagRepo(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Repo, "repo", "",
		`Code repository where the generated code will reside.
			Can be a remote in the format of a remote URL such as 
			https://github.com/{owner}/{repo} or a local file path like 
			/path/to/repo. Both absolute and relative paths are supported.
			If not specified, will try to detect if the current working 
			directory is configured as a language repository.`)
}

func addFlagWorkRoot(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.WorkRoot, "output", "", "Working directory root. When this is not specified, a working directory will be created in /tmp.")
}
