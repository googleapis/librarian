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
	"flag"
)

var (
	flagAPIPath     string
	flagAPIRoot     string
	flagBranch      string
	flagBuild       bool
	flagGitHubToken string
	flagImage       string
	flagLanguage    string
	flagOutput      string
	flagPush        bool
	flagRepoRoot    string
	flagWorkRoot    string
)

func addFlagAPIPath(fs *flag.FlagSet) {
	fs.StringVar(&flagAPIPath, "api-path", "", "(Required) path api-root to the API to be generated (e.g., google/cloud/functions/v2)")
}

func addFlagAPIRoot(fs *flag.FlagSet) {
	fs.StringVar(&flagAPIRoot, "api-root", "", "location of googleapis repository. If undefined, googleapis will be cloned to /tmp")
}

func addFlagBranch(fs *flag.FlagSet) {
	fs.StringVar(&flagBranch, "branch", "main", "repository branch")
}

func addFlagBuild(fs *flag.FlagSet) {
	fs.BoolVar(&flagBuild, "build", false, "whether to build the generated code")
}

func addFlagGitHubToken(fs *flag.FlagSet) {
	fs.StringVar(&flagGitHubToken, "github-token", "", "GitHub access token")
}

func addFlagImage(fs *flag.FlagSet) {
	fs.StringVar(&flagImage, "image", "", "language-specific container to run for subcommands. Defaults to google-cloud-{language}-generator")
}

func addFlagLanguage(fs *flag.FlagSet) {
	fs.StringVar(&flagLanguage, "language", "", "(Required) language to generate code for")
}

func addFlagOutput(fs *flag.FlagSet) {
	fs.StringVar(&flagOutput, "output", "", "directory where generated code will be written")
}

func addFlagPush(fs *flag.FlagSet) {
	fs.BoolVar(&flagPush, "push", false, "push to GitHub if true")
}

func addFlagRepoRoot(fs *flag.FlagSet) {
	fs.StringVar(&flagRepoRoot, "repo-root", "", "Repository root. When this is not specified, the language repo will be cloned.")
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
