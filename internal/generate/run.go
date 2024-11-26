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
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/googleapis/generator/internal/gitrepo"
)

func Run(ctx context.Context, arg ...string) error {
	cfg := &config{}
	cfg, err := parseFlags(cfg, arg)
	if err != nil {
		return err
	}
	googleapisRepo, err := cloneGoogleapis(ctx)
	if err != nil {
		return err
	}
	languageRepo, err := cloneLanguageRepo(ctx, cfg.language)
	if err != nil {
		return err
	}
	return runDocker(googleapisRepo.Dir, languageRepo.Dir, cfg.api)
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

func runCommand(c string, args ...string) error {
	cmd := exec.Command(c, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	slog.Info(strings.Repeat("-", 80))
	slog.Info(cmd.String())
	slog.Info(strings.Repeat("-", 80))
	return cmd.Run()
}
