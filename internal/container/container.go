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

package container

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Generate(ctx context.Context, language, apiRoot, apiPath, output, generatorInput string) error {
	languageDir := filepath.Join(output, fmt.Sprintf("google-cloud-%s", language))
	return runGenerate(apiRoot, languageDir, apiPath)
}

func Clean(ctx context.Context, language, repoRoot, apiPath string) error {
	return runCommand("echo", "clean not implemented")
}

func Build(ctx context.Context, language, repoRoot, apiPath string) error {
	return runCommand("echo", "build not implemented")
}

func Configure(ctx context.Context, language, apiRoot, apiPath, generatorInput string) error {
	return runCommand("echo", "configure not implemented")
}

const dotnetImageTag = "picard"

func runGenerate(googleapisDir, languageDir, apiPath string) error {
	if apiPath == "" {
		return fmt.Errorf("apiPath cannot be empty")
	}
	args := []string{
		"run",
		"-v", fmt.Sprintf("%s:/apis", googleapisDir),
		"-v", fmt.Sprintf("%s:/output", languageDir),
		dotnetImageTag,
		"generate",
		"--api-root=/apis",
		fmt.Sprintf("--api-path=%s", apiPath),
		"--output=/output",
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
