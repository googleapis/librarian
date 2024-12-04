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

package generate

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/generator/internal/gitrepo"
)

var (
	apiFlag      string
	languageFlag string
)

func generatorGenerateCommand() *command {
	c := &command{
		name:  "generate",
		short: "Generate a new client library",
		usage: "generator generate [arguments]",
		flags: flag.NewFlagSet("generate", flag.ContinueOnError),
		run:   generate,
	}
	c.flags.StringVar(&apiFlag, "api", "", "name of API inside googleapis")
	c.flags.Func("language", "language to generate", func(language string) error {
		if !supportedLanguages[language] {
			return fmt.Errorf("invalid -language flag specified: %q", language)
		}
		languageFlag = language
		return nil
	})
	c.flags.Usage = constructUsage(c.flags, c.short, c.usage, c.commands, true)
	return c
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
}

func generate(ctx context.Context) error {
	googleapisRepo, err := cloneGoogleapis(ctx)
	if err != nil {
		return err
	}
	languageRepo, err := cloneLanguageRepo(ctx, languageFlag)
	if err != nil {
		return err
	}
	return runDocker(googleapisRepo.Dir, languageRepo.Dir, apiFlag)
}

const googleapisURL = "https://github.com/googleapis/googleapis"

func cloneGoogleapis(ctx context.Context) (*gitrepo.Repo, error) {
	repoPath := filepath.Join(os.TempDir(), "/generator-googleapis")
	return gitrepo.CloneOrOpen(ctx, repoPath, googleapisURL)
}

func cloneLanguageRepo(ctx context.Context, language string) (*gitrepo.Repo, error) {
	languageRepoURL := fmt.Sprintf("https://github.com/googleapis/google-cloud-%s", language)
	repoPath := filepath.Join(os.TempDir(), fmt.Sprintf("/generator-google-cloud-%s", language))
	return gitrepo.CloneOrOpen(ctx, repoPath, languageRepoURL)
}

const dotnetImageTag = "picard"

func runDocker(googleapisDir, languageDir, api string) error {
	args := []string{
		"run",
		"-v", fmt.Sprintf("%s:/apis", googleapisDir),
		"-v", fmt.Sprintf("%s:/output", languageDir),
		dotnetImageTag,
		"--command=update",
		"--api-root=/apis",
		fmt.Sprintf("--api=%s", api),
		"--output-root=/output",
	}
	return runCommand("docker", args...)
}
