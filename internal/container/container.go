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

func Generate(ctx context.Context, image, apiRoot, output, generatorInput, apiPath string) error {
	return runGenerate(image, apiRoot, output, generatorInput, apiPath)
}

func Clean(ctx context.Context, image, repoRoot, apiPath string) error {
	return runClean(image, repoRoot, apiPath)
}

func Build(ctx context.Context, image, rootOptionName, root, apiPath string) error {
	return runBuild(image, rootOptionName, root, apiPath)
}

func Configure(ctx context.Context, image, apiRoot, apiPath, generatorInput string) error {
	if image == "" {
		return fmt.Errorf("image cannot be empty")
	}
	if apiRoot == "" {
		return fmt.Errorf("apiRoot cannot be empty")
	}
	if apiPath == "" {
		return fmt.Errorf("apiPath cannot be empty")
	}
	if generatorInput == "" {
		return fmt.Errorf("generatorInput cannot be empty")
	}
	containerArgs := []string{
		"configure",
		"--api-root=/apis",
		"--generator-input=/generator-input",
		fmt.Sprintf("--api-path=%s", apiPath),
	}
	mounts := []string{
		fmt.Sprintf("%s:/apis", apiRoot),
		fmt.Sprintf("%s:/generator-input", generatorInput),
	}
	return runDocker(image, mounts, containerArgs)
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
	containerArgs := []string{
		"generate",
		"--api-root=/apis",
		"--output=/output",
	}
	mounts := []string{
		fmt.Sprintf("%s:/apis", apiRoot),
		fmt.Sprintf("%s:/output", output),
	}

	if generatorInput != "" {
		mounts = append(mounts, fmt.Sprintf("%s:/generator-input", generatorInput))
		containerArgs = append(containerArgs, "--generator-input=/generator-input")
	}
	if apiPath != "" {
		containerArgs = append(containerArgs, fmt.Sprintf("--api-path=%s", apiPath))
	}

	// _, err := getCommand("ls", "-la", generatorInput)
	// if err != nil {
	// 	return err
	// }
	// runDockerWithEntrypointOverride(image, mounts, "id", []string{})
	// runDockerWithEntrypointOverride(image, mounts, "ls", []string{"-la"})
	// runDockerWithEntrypointOverride(image, mounts, "ls", []string{"-la", "/generator-input"})
	sharedDir := os.Getenv("KOKORO_ARTIFACTS_DIR")
	sharedFile := filepath.Join(sharedDir, "orpheus.txt")
	getCommand("cat", sharedFile)
	mounts = append(mounts, fmt.Sprintf("%s:%s", sharedDir, sharedDir))
	runDockerWithEntrypointOverride(image, mounts, "cat", []string{sharedFile})
	return runDocker(image, mounts, containerArgs)
}

func runClean(image, repoRoot, apiPath string) error {
	if image == "" {
		return fmt.Errorf("image cannot be empty")
	}
	if repoRoot == "" {
		return fmt.Errorf("repoRoot cannot be empty")
	}
	mounts := []string{
		fmt.Sprintf("%s:/repo", repoRoot),
	}
	containerArgs := []string{
		"clean",
		"--repo-root=/repo",
	}
	if apiPath != "" {
		containerArgs = append(containerArgs, fmt.Sprintf("--api-path=%s", apiPath))
	}
	return runDocker(image, mounts, containerArgs)
}

func runBuild(image, rootName, root, apiPath string) error {
	if image == "" {
		return fmt.Errorf("image cannot be empty")
	}
	if rootName == "" {
		return fmt.Errorf("rootName cannot be empty")
	}
	if root == "" {
		return fmt.Errorf("root cannot be empty")
	}
	mounts := []string{
		fmt.Sprintf("%s:/%s", root, rootName),
	}
	containerArgs := []string{
		"build",
		fmt.Sprintf("--%s=/%s", rootName, rootName),
	}
	if apiPath != "" {
		containerArgs = append(containerArgs, fmt.Sprintf("--api-path=%s", apiPath))
	}
	return runDocker(image, mounts, containerArgs)
}

func runDockerWithEntrypointOverride(image string, mounts []string, entrypoint string, entrypointArgs []string) error {
	args := []string{
		"run",
	}
	for _, mount := range mounts {
		args = append(args, "-v", mount)
	}

	uid, err := uid()
	if err != nil {
		return err
	}
	gid, err := gid()
	if err != nil {
		return err
	}

	args = append(args, "--user", fmt.Sprintf("%s:%s", uid, gid))
	args = append(args, "--entrypoint", entrypoint)
	args = append(args, image)
	args = append(args, entrypointArgs...)
	return runCommand("docker", args...)
}

func runDocker(image string, mounts []string, containerArgs []string) error {
	args := []string{
		"run",
	}
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

func uid() (string, error) {
	return getCommand("id", "-u")
}
func gid() (string, error) {
	return getCommand("id", "-g")
}
func getCommand(c string, args ...string) (string, error) {
	cmd := exec.Command(c, args...)
	slog.Info(cmd.String())

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	result := string(out[:len(out)-1])
	slog.Info(result)
	return result, nil
}
