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
	"os/user"
	"strings"
)

func GenerateRaw(ctx context.Context, image, apiRoot, output, apiPath string) error {
	if image == "" {
		return fmt.Errorf("image cannot be empty")
	}
	if apiRoot == "" {
		return fmt.Errorf("apiRoot cannot be empty")
	}
	if output == "" {
		return fmt.Errorf("output cannot be empty")
	}
	if apiPath == "" {
		return fmt.Errorf("apiPath cannot be empty")
	}
	containerArgs := []string{
		"generate-raw",
		"--api-root=/apis",
		"--output=/output",
		fmt.Sprintf("--api-path=%s", apiPath),
	}
	mounts := []string{
		fmt.Sprintf("%s:/apis", apiRoot),
		fmt.Sprintf("%s:/output", output),
	}
	return runDocker(image, mounts, containerArgs)
}

func GenerateLibrary(ctx context.Context, image, apiRoot, output, generatorInput, libraryID string) error {
	if image == "" {
		return fmt.Errorf("image cannot be empty")
	}
	if apiRoot == "" {
		return fmt.Errorf("apiRoot cannot be empty")
	}
	if output == "" {
		return fmt.Errorf("output cannot be empty")
	}
	if generatorInput == "" {
		return fmt.Errorf("generatorInput cannot be empty")
	}
	if libraryID == "" {
		return fmt.Errorf("libraryID cannot be empty")
	}
	containerArgs := []string{
		"generate-library",
		"--api-root=/apis",
		"--output=/output",
		"--generator-input=/generator-input",
		fmt.Sprintf("--library-id=%s", libraryID),
	}
	mounts := []string{
		fmt.Sprintf("%s:/apis", apiRoot),
		fmt.Sprintf("%s:/output", output),
		fmt.Sprintf("%s:/generator-input", generatorInput),
	}
	return runDocker(image, mounts, containerArgs)
}

func Clean(ctx context.Context, image, repoRoot, libraryID string) error {
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
	if libraryID != "" {
		containerArgs = append(containerArgs, fmt.Sprintf("--library-id=%s", libraryID))
	}
	return runDocker(image, mounts, containerArgs)
}

func BuildRaw(image, generatorOutput, apiPath string) error {
	if image == "" {
		return fmt.Errorf("image cannot be empty")
	}
	if generatorOutput == "" {
		return fmt.Errorf("generatorOutput cannot be empty")
	}
	if apiPath == "" {
		return fmt.Errorf("apiPath cannot be empty")
	}
	mounts := []string{
		fmt.Sprintf("%s:/generator-output", generatorOutput),
	}
	containerArgs := []string{
		"build-raw",
		"--generator-output=/generator-output",
		fmt.Sprintf("--api-path=%s", apiPath),
	}
	return runDocker(image, mounts, containerArgs)
}

func BuildLibrary(image, repoRoot, libraryId string) error {
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
		"build-library",
		"--repo-root=/repo",
	}
	if libraryId != "" {
		containerArgs = append(containerArgs, fmt.Sprintf("--library-id=%s", libraryId))
	}
	return runDocker(image, mounts, containerArgs)
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

func PrepareLibraryRelease(image string, languageRepo string, inputsDirectory string, libId string, releaseVersion string) error {
	if image == "" {
		return fmt.Errorf("image cannot be empty")
	}
	containerArgs := []string{
		"prepare-library-release",
		"--repo-root=/repo",
		fmt.Sprintf("--library-id=%s", libId),
		fmt.Sprintf("--release-notes=/inputs/%s-%s-release-notes.txt", libId, releaseVersion),
		fmt.Sprintf("--version=%s", releaseVersion),
	}
	mounts := []string{
		fmt.Sprintf("%s:/repo", languageRepo),
		fmt.Sprintf("%s:/inputs", inputsDirectory),
	}

	return runDocker(image, mounts, containerArgs)
}

func runDocker(image string, mounts []string, containerArgs []string) error {
	mounts = maybeRelocateMounts(mounts)

	args := []string{
		"run",
		"--rm", // Automatically delete the container after completion
	}
	// Run as the current user in the container - primarily so that any
	// files we create end up being owned by the current user (and easily deletable).
	currentUser, err := user.Current()
	if err != nil {
		return err
	}
	args = append(args, fmt.Sprintf("--user=%s:%s", currentUser.Uid, currentUser.Gid))

	for _, mount := range mounts {
		args = append(args, "-v", mount)
	}

	args = append(args, image)
	args = append(args, containerArgs...)
	return runCommand("docker", args...)
}

func maybeRelocateMounts(mounts []string) []string {
	// When running in Kokoro, we'll be running sibling containers.
	// Make sure we specify the "from" part of the mount as the host directory.
	kokoroHostRootDir := os.Getenv("KOKORO_HOST_ROOT_DIR")
	kokoroRootDir := os.Getenv("KOKORO_ROOT_DIR")
	if kokoroRootDir == "" || kokoroHostRootDir == "" {
		return mounts
	}
	relocatedMounts := []string{}
	for _, mount := range mounts {
		if strings.HasPrefix(mount, kokoroRootDir) {
			mount = strings.Replace(mount, kokoroRootDir, kokoroHostRootDir, 1)
		}
		relocatedMounts = append(relocatedMounts, mount)
	}
	return relocatedMounts
}

func runCommand(c string, args ...string) error {
	cmd := exec.Command(c, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	slog.Info(fmt.Sprintf("=== Docker start %s", strings.Repeat("=", 63)))
	slog.Info(cmd.String())
	slog.Info(strings.Repeat("-", 80))
	err := cmd.Run()
	slog.Info(fmt.Sprintf("=== Docker end %s", strings.Repeat("=", 65)))
	return err
}
