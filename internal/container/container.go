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
	"strings"
)

func Generate(ctx context.Context, image, apiRoot, output, generatorInput, apiPath string) error {
	return runGenerate(image, apiRoot, output, generatorInput, apiPath)
}

func Clean(ctx context.Context, image, repoRoot, apiPath string) error {
	return runClean(image, repoRoot, apiPath)
}

func Build(ctx context.Context, image, repoRoot, apiPath string) error {
	return runBuild(image, repoRoot, apiPath)
}

func Configure(ctx context.Context, language, apiRoot, apiPath, generatorInput string) error {
	return runCommand("echo", "configure not implemented")
}

func runGenerate(image, apiRoot, output, generatorInput, apiPath string) error {
	if image == "" {
		return fmt.Errorf("image cannot be empty")
	}
	if apiRoot == "" {
		return fmt.Errorf("apiRoot cannot be empty")
	}
	if output == "" {
		return fmt.Errorf("output cannot be empty")
	}
	if generatorInput == "" && apiPath == "" {
		return fmt.Errorf("apiPath and generatorInput can't both be empty")
	}
	var containerArgs []string
	containerArgs = append(containerArgs,
		"generate",
		"--api-root=/apis",
		"--output=/output")
	var mounts []string
	mounts = append(mounts,
		fmt.Sprintf("%s:/apis", apiRoot),
		fmt.Sprintf("%s:/output", output),
	)

	if generatorInput != "" {
		mounts = append(mounts, fmt.Sprintf("%s:/generator-input", generatorInput))
		containerArgs = append(containerArgs, "--generator-input=/generator-input")
	}
	if apiPath != "" {
		containerArgs = append(containerArgs, fmt.Sprintf("--api-path=%s", apiPath))
	}
	return runDocker(image, mounts, containerArgs)
}

func runClean(image, repoRoot, apiPath string) error {
	if image == "" {
		return fmt.Errorf("image cannot be empty")
	}
	if repoRoot == "" {
		return fmt.Errorf("repoRoot cannot be empty")
	}
	mounts := []string{fmt.Sprintf("%s:/repo", repoRoot)}
	var containerArgs []string
	containerArgs = append(containerArgs,
		"clean",
		"--repo-root=/repo",
	)
	if apiPath != "" {
		containerArgs = append(containerArgs, fmt.Sprintf("--api-path=%s", apiPath))
	}
	return runDocker(image, mounts, containerArgs)
}

func runBuild(image, repoRoot, apiPath string) error {
	if image == "" {
		return fmt.Errorf("image cannot be empty")
	}
	if repoRoot == "" {
		return fmt.Errorf("repoRoot cannot be empty")
	}
	mounts := []string{fmt.Sprintf("%s:/repo", repoRoot)}
	var containerArgs []string
	containerArgs = append(containerArgs,
		"build",
		"--repo-root=/repo",
	)
	if apiPath != "" {
		containerArgs = append(containerArgs, fmt.Sprintf("--api-path=%s", apiPath))
	}
	return runDocker(image, mounts, containerArgs)
}

func runDocker(image string, mounts []string, containerArgs []string) error {
	var args []string
	args = append(args, "run")
	for _, mount := range mounts {
		args = append(args, "-v", mount)
	}
	args = append(args, image)
	args = append(args, containerArgs...)
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
