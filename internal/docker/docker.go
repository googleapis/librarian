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

package docker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/user"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/statepb"
)

type Command string

// The set of commands passed to the language container, in a single place to avoid typos.
const (
	CommandGenerateRaw            Command = "generate-raw"
	CommandGenerateLibrary        Command = "generate-library"
	CommandClean                  Command = "clean"
	CommandBuildRaw               Command = "build-raw"
	CommandBuildLibrary           Command = "build-library"
	CommandConfigure              Command = "configure"
	CommandPrepareLibraryRelease  Command = "prepare-library-release"
	CommandIntegrationTestLibrary Command = "integration-test-library"
	CommandPackageLibrary         Command = "package-library"
	CommandPublishLibrary         Command = "publish-library"
)

var networkEnabledCommands = []Command{
	CommandBuildRaw,
	CommandBuildLibrary,
	CommandIntegrationTestLibrary,
	CommandPackageLibrary,
	CommandPublishLibrary,
}

type Docker struct {
	// The Docker image to run.
	Image string

	// The provider for environment variables, if any.
	env *EnvironmentProvider

	// run runs the docker command.
	run func(args ...string) error
}

func New(ctx context.Context, workRoot, image, secretsProject string, pipelineConfig *statepb.PipelineConfig) (*Docker, error) {
	envProvider, err := newEnvironmentProvider(ctx, workRoot, secretsProject, pipelineConfig)
	if err != nil {
		return nil, err
	}
	return &Docker{
		Image: image,
		env:   envProvider,
		run: func(args ...string) error {
			return runCommand("docker", args...)
		},
	}, nil
}

func (c *Docker) GenerateRaw(apiRoot, output, apiPath string) error {
	if apiRoot == "" {
		return fmt.Errorf("apiRoot cannot be empty")
	}
	if output == "" {
		return fmt.Errorf("output cannot be empty")
	}
	if apiPath == "" {
		return fmt.Errorf("apiPath cannot be empty")
	}
	commandArgs := []string{
		"--api-root=/apis",
		"--output=/output",
		fmt.Sprintf("--api-path=%s", apiPath),
	}
	mounts := []string{
		fmt.Sprintf("%s:/apis", apiRoot),
		fmt.Sprintf("%s:/output", output),
	}
	return c.runDocker(CommandGenerateRaw, mounts, commandArgs)
}

func (c *Docker) GenerateLibrary(apiRoot, output, generatorInput, libraryID string) error {
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
	commandArgs := []string{
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
	return c.runDocker(CommandGenerateLibrary, mounts, commandArgs)
}

func (c *Docker) Clean(repoRoot, libraryID string) error {
	if repoRoot == "" {
		return fmt.Errorf("repoRoot cannot be empty")
	}
	mounts := []string{
		fmt.Sprintf("%s:/repo", repoRoot),
	}
	commandArgs := []string{
		"--repo-root=/repo",
	}
	if libraryID != "" {
		commandArgs = append(commandArgs, fmt.Sprintf("--library-id=%s", libraryID))
	}
	return c.runDocker(CommandClean, mounts, commandArgs)
}

func (c *Docker) BuildRaw(generatorOutput, apiPath string) error {
	if generatorOutput == "" {
		return fmt.Errorf("generatorOutput cannot be empty")
	}
	if apiPath == "" {
		return fmt.Errorf("apiPath cannot be empty")
	}
	mounts := []string{
		fmt.Sprintf("%s:/generator-output", generatorOutput),
	}
	commandArgs := []string{
		"--generator-output=/generator-output",
		fmt.Sprintf("--api-path=%s", apiPath),
	}
	return c.runDocker(CommandBuildRaw, mounts, commandArgs)
}

func (c *Docker) BuildLibrary(repoRoot, libraryId string) error {
	if repoRoot == "" {
		return fmt.Errorf("repoRoot cannot be empty")
	}
	mounts := []string{
		fmt.Sprintf("%s:/repo", repoRoot),
	}
	commandArgs := []string{
		"--repo-root=/repo",
		"--test=true",
	}
	if libraryId != "" {
		commandArgs = append(commandArgs, fmt.Sprintf("--library-id=%s", libraryId))
	}
	return c.runDocker(CommandBuildLibrary, mounts, commandArgs)
}

func (c *Docker) Configure(apiRoot, apiPath, generatorInput string) error {
	if apiRoot == "" {
		return fmt.Errorf("apiRoot cannot be empty")
	}
	if apiPath == "" {
		return fmt.Errorf("apiPath cannot be empty")
	}
	if generatorInput == "" {
		return fmt.Errorf("generatorInput cannot be empty")
	}
	commandArgs := []string{
		"--api-root=/apis",
		"--generator-input=/generator-input",
		fmt.Sprintf("--api-path=%s", apiPath),
	}
	mounts := []string{
		fmt.Sprintf("%s:/apis", apiRoot),
		fmt.Sprintf("%s:/generator-input", generatorInput),
	}
	return c.runDocker(CommandConfigure, mounts, commandArgs)
}

func (c *Docker) PrepareLibraryRelease(languageRepo, inputsDirectory, libId, releaseVersion string) error {
	commandArgs := []string{
		"--repo-root=/repo",
		fmt.Sprintf("--library-id=%s", libId),
		fmt.Sprintf("--release-notes=/inputs/%s-%s-release-notes.txt", libId, releaseVersion),
		fmt.Sprintf("--version=%s", releaseVersion),
	}
	mounts := []string{
		fmt.Sprintf("%s:/repo", languageRepo),
		fmt.Sprintf("%s:/inputs", inputsDirectory),
	}

	return c.runDocker(CommandPrepareLibraryRelease, mounts, commandArgs)
}

func (c *Docker) IntegrationTestLibrary(languageRepo, libId string) error {
	commandArgs := []string{
		"--repo-root=/repo",
		fmt.Sprintf("--library-id=%s", libId),
	}
	mounts := []string{
		fmt.Sprintf("%s:/repo", languageRepo),
	}

	return c.runDocker(CommandIntegrationTestLibrary, mounts, commandArgs)
}

func (c *Docker) PackageLibrary(languageRepo, libId, outputDir string) error {
	commandArgs := []string{
		"--repo-root=/repo",
		"--output=/output",
		fmt.Sprintf("--library-id=%s", libId),
	}
	mounts := []string{
		fmt.Sprintf("%s:/repo", languageRepo),
		fmt.Sprintf("%s:/output", outputDir),
	}

	return c.runDocker(CommandPackageLibrary, mounts, commandArgs)
}

func (c *Docker) PublishLibrary(outputDir, libId, libVersion string) error {
	commandArgs := []string{
		"--package-output=/output",
		fmt.Sprintf("--library-id=%s", libId),
		fmt.Sprintf("--version=%s", libVersion),
	}
	mounts := []string{
		fmt.Sprintf("%s:/output", outputDir),
	}

	return c.runDocker(CommandPublishLibrary, mounts, commandArgs)
}

func (c *Docker) runDocker(command Command, mounts []string, commandArgs []string) error {
	if c.Image == "" {
		return fmt.Errorf("image cannot be empty")
	}

	mounts = maybeRelocateMounts(mounts)

	args := []string{
		"run",
		"--rm", // Automatically delete the container after completion
	}

	for _, mount := range mounts {
		args = append(args, "-v", mount)
	}
	if c.env != nil {
		if err := c.env.writeEnvironmentFile(string(command)); err != nil {
			return err
		}
		args = append(args, "--env-file")
		args = append(args, c.env.tmpFile)
		defer deleteEnvironmentFile(c.env)
	}
	if !slices.Contains(networkEnabledCommands, command) {
		args = append(args, "--network=none")
	}
	args = append(args, c.Image)
	args = append(args, string(command))
	args = append(args, commandArgs...)
	return c.run(args...)
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
	// Run as the current user in the container - primarily so that any files
	// we create end up being owned by the current user (and easily deletable).
	//
	// TODO(https://github.com/googleapis/librarian/issues/590):
	// temporarily lives here to support testing; move to config
	currentUser, err := user.Current()
	if err != nil {
		return err
	}
	args = append(args, fmt.Sprintf("--user=%s:%s", currentUser.Uid, currentUser.Gid))

	cmd := exec.Command(c, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	slog.Info(fmt.Sprintf("=== Docker start %s", strings.Repeat("=", 63)))
	slog.Info(cmd.String())
	slog.Info(strings.Repeat("-", 80))
	err = cmd.Run()
	slog.Info(fmt.Sprintf("=== Docker end %s", strings.Repeat("=", 65)))
	return err
}
