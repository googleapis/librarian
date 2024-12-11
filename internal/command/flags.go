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
	"fmt"
)

var (
	flagAPIPath        string
	flagAPIRoot        string
	flagBranch         string
	flagGeneratorInput string
	flagGitHubToken    string
	flagLanguage       string
	flagOutput         string
	flagPush           bool
)

func addFlagAPIRoot(fs *flag.FlagSet) {
	fs.StringVar(&flagAPIRoot, "api-root", "", "location of googleapis repository")
}

func addFlagAPIPath(fs *flag.FlagSet) {
	fs.StringVar(&flagAPIPath, "api-path", "", "path api-root to the API to be generated (e.g., google/cloud/functions/v2)")
}

func addFlagBranch(fs *flag.FlagSet) {
	fs.StringVar(&flagBranch, "branch", "main", "repository branch")
}

func addFlagLanguage(fs *flag.FlagSet) {
	fs.StringVar(&flagLanguage, "language", "", "language to generate code for")
	fs.Func("language", "", func(language string) error {
		if !supportedLanguages[language] {
			return fmt.Errorf("invalid -language flag specified: %q", language)
		}
		flagLanguage = language
		return nil
	})
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

func addFlagOutput(fs *flag.FlagSet) {
	fs.StringVar(&flagOutput, "output", "", "directory where generated code will be written")
}

func addFlagPush(fs *flag.FlagSet) {
	fs.BoolVar(&flagPush, "push", false, "push to GitHub if true")
}

func addFlagGitHubToken(fs *flag.FlagSet) {
	fs.StringVar(&flagGitHubToken, "github-token", "", "GitHub access token")
}

func addFlagGeneratorInput(fs *flag.FlagSet) {
	fs.StringVar(&flagGeneratorInput, "generator-input", "", "generator-input within the clone we've just created")
}
