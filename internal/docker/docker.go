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

// Package docker provides the interface for running language-specific
// Docker containers which conform to the Librarian container contract.
// TODO(https://github.com/googleapis/librarian/issues/330): link to
// the documentation when it's written.
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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/statepb"
)

// Command is the string representation of a command to be passed to the language-specific
// container's entry point as the first argument.
type Command string

// The set of commands passed to the language container, in a single place to avoid typos.
const (
	// CommandGenerateRaw performs raw (unconfigured) generation.
	CommandGenerateRaw Command = "generate-raw"
	// CommandGenerateLibrary performs generation for a configured library.
	CommandGenerateLibrary Command = "generate-library"
	// CommandClean cleans files generated for a library.
	CommandClean Command = "clean"
	// CommandBuildRaw builds the results of generate-raw.
	CommandBuildRaw Command = "build-raw"
	// CommandBuildLibrary builds a library.
	CommandBuildLibrary Command = "build-library"
	// CommandConfigure configures a new API as a library.
	CommandConfigure Command = "configure"
	// CommandPrepareLibraryRelease prepares a repository for the release of a library.
	CommandPrepareLibraryRelease Command = "prepare-library-release"
	// CommandIntegrationTestLibrary runs integration tests on a library.
	CommandIntegrationTestLibrary Command = "integration-test-library"
	// CommandPackageLibrary packages a library's artifacts for publication.
	CommandPackageLibrary Command = "package-library"
	// CommandPublishLibrary publishes a library's artifacts.
	CommandPublishLibrary Command = "publish-library"
)

var networkEnabledCommands = []Command{
	CommandBuildRaw,
	CommandBuildLibrary,
	CommandIntegrationTestLibrary,
	CommandPackageLibrary,
	CommandPublishLibrary,
}

// Docker contains all the information required to run language-specific
// Docker containers.
type Docker struct {
	// The Docker image to run.
	Image string

	// The provider for environment variables, if any.
	env *EnvironmentProvider

	// run runs the docker command.
	run func(args ...string) error
}

// New constructs a Docker instance which will invoke the specified
// Docker image as required to implement language-specific commands,
// providing the container with required environment variables.
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

// GenerateRaw performs generation for an API not configured in a library.
// This does not have any context from a language repo: it requires
// generation purely on the basis of the API specification, which is in
// the subdirectory apiPath of the API specification repo apiRoot, and whatever
// is in the language-specific Docker container. The code is generated
// in the output directory, which is initially empty.
func (c *Docker) GenerateRaw(cfg *config.Config, apiRoot, output, apiPath string) error {
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

	return c.runDocker(cfg, CommandGenerateRaw, mounts, commandArgs)
}

// GenerateLibrary performs generation for an API which is configured as part of a library.
// apiRoot specifies the root directory of the API specification repo,
// output specifies the empty output directory into which the command should
// generate code, and libraryID specifies the ID of the library to generate,
// as configured in the Librarian state file for the repository.
func (c *Docker) GenerateLibrary(cfg *config.Config, apiRoot, output, generatorInput, libraryID string) error {
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
		fmt.Sprintf("--%s=/%s", config.GeneratorInputDir, config.GeneratorInputDir),
		fmt.Sprintf("--library-id=%s", libraryID),
	}
	mounts := []string{
		fmt.Sprintf("%s:/apis", apiRoot),
		fmt.Sprintf("%s:/output", output),
		fmt.Sprintf("%s:/%s", generatorInput, config.GeneratorInputDir),
	}

	return c.runDocker(cfg, CommandGenerateLibrary, mounts, commandArgs)
}

// Clean deletes files within repoRoot which are generated for library
// libraryID, as configured in the Librarian state file for the repository.
func (c *Docker) Clean(cfg *config.Config, repoRoot, libraryID string) error {
	if repoRoot == "" {
		return fmt.Errorf("repoRoot cannot be empty")
	}
	mounts := []string{
		fmt.Sprintf("%s:/repo", repoRoot),
	}
	commandArgs := []string{
		"--repo-root=/repo",
		fmt.Sprintf("--library-id=%s", libraryID),
	}

	return c.runDocker(cfg, CommandClean, mounts, commandArgs)
}

// BuildRaw builds the result of GenerateRaw, which previously generated
// code for apiPath in generatorOutput.
func (c *Docker) BuildRaw(cfg *config.Config, generatorOutput, apiPath string) error {
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

	return c.runDocker(cfg, CommandBuildRaw, mounts, commandArgs)
}

// BuildLibrary builds the library with an ID of libraryID, as configured in
// the Librarian state file for the repository with a root of repoRoot.
func (c *Docker) BuildLibrary(cfg *config.Config, repoRoot, libraryID string) error {
	if repoRoot == "" {
		return fmt.Errorf("repoRoot cannot be empty")
	}
	mounts := []string{
		fmt.Sprintf("%s:/repo", repoRoot),
	}
	commandArgs := []string{
		"--repo-root=/repo",
		"--test=true",
		fmt.Sprintf("--library-id=%s", libraryID),
	}

	return c.runDocker(cfg, CommandBuildLibrary, mounts, commandArgs)
}

// Configure configures an API within a repository, either adding it to an
// existing library or creating a new library. The API is indicated by the
// apiPath directory within apiRoot, and the container is provided with the
// generatorInput directory to record the results of configuration. The
// library code is not generated.
func (c *Docker) Configure(cfg *config.Config, apiRoot, apiPath, generatorInput string) error {
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
		fmt.Sprintf("--%s=/%s", config.GeneratorInputDir, config.GeneratorInputDir),
		fmt.Sprintf("--api-path=%s", apiPath),
	}
	mounts := []string{
		fmt.Sprintf("%s:/apis", apiRoot),
		fmt.Sprintf("%s:/%s", generatorInput, config.GeneratorInputDir),
	}

	return c.runDocker(cfg, CommandConfigure, mounts, commandArgs)
}

// PrepareLibraryRelease prepares the repository languageRepo for the release of a library with
// ID libraryID within repoRoot, with version releaseVersion. Release notes
// are expected to be present within inputsDirectory, in a file named
// `{libraryID}-{releaseVersion}-release-notes.txt`.
func (c *Docker) PrepareLibraryRelease(cfg *config.Config, repoRoot, inputsDirectory, libraryID, releaseVersion string) error {
	commandArgs := []string{
		"--repo-root=/repo",
		fmt.Sprintf("--library-id=%s", libraryID),
		fmt.Sprintf("--release-notes=/inputs/%s-%s-release-notes.txt", libraryID, releaseVersion),
		fmt.Sprintf("--version=%s", releaseVersion),
	}
	mounts := []string{
		fmt.Sprintf("%s:/repo", repoRoot),
		fmt.Sprintf("%s:/inputs", inputsDirectory),
	}

	return c.runDocker(cfg, CommandPrepareLibraryRelease, mounts, commandArgs)
}

// IntegrationTestLibrary runs the integration tests for a library with ID libraryID within repoRoot.
func (c *Docker) IntegrationTestLibrary(cfg *config.Config, repoRoot, libraryID string) error {
	commandArgs := []string{
		"--repo-root=/repo",
		fmt.Sprintf("--library-id=%s", libraryID),
	}
	mounts := []string{
		fmt.Sprintf("%s:/repo", repoRoot),
	}

	return c.runDocker(cfg, CommandIntegrationTestLibrary, mounts, commandArgs)
}

// PackageLibrary packages release artifacts for a library with ID libraryID within repoRoot,
// creating the artifacts within outputDir.
func (c *Docker) PackageLibrary(cfg *config.Config, repoRoot, libraryID, outputDir string) error {
	commandArgs := []string{
		"--repo-root=/repo",
		"--output=/output",
		fmt.Sprintf("--library-id=%s", libraryID),
	}
	mounts := []string{
		fmt.Sprintf("%s:/repo", repoRoot),
		fmt.Sprintf("%s:/output", outputDir),
	}

	return c.runDocker(cfg, CommandPackageLibrary, mounts, commandArgs)
}

// PublishLibrary publishes release artifacts for a library with ID libraryID and version releaseVersion
// to package managers, documentation sites etc. The artifacts will previously have been
// created by PackageLibrary.
func (c *Docker) PublishLibrary(cfg *config.Config, outputDir, libraryID, releaseVersion string) error {
	commandArgs := []string{
		"--package-output=/output",
		fmt.Sprintf("--library-id=%s", libraryID),
		fmt.Sprintf("--version=%s", releaseVersion),
	}
	mounts := []string{
		fmt.Sprintf("%s:/output", outputDir),
	}

	return c.runDocker(cfg, CommandPublishLibrary, mounts, commandArgs)
}

func (c *Docker) runDocker(cfg *config.Config, command Command, mounts []string, commandArgs []string) (err error) {
	if c.Image == "" {
		return fmt.Errorf("image cannot be empty")
	}

	mounts = maybeRelocateMounts(cfg, mounts)

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
		defer func() {
			cerr := os.Remove(c.env.tmpFile)
			if err == nil {
				err = cerr
			}
		}()
	}
	if !slices.Contains(networkEnabledCommands, command) {
		args = append(args, "--network=none")
	}
	args = append(args, c.Image)
	args = append(args, string(command))
	args = append(args, commandArgs...)
	return c.run(args...)
}

func maybeRelocateMounts(cfg *config.Config, mounts []string) []string {
	// When running in Kokoro, we'll be running sibling containers.
	// Make sure we specify the "from" part of the mount as the host directory.
	if cfg.DockerMountRootDir == "" || cfg.DockerHostRootDir == "" {
		return mounts
	}
	relocatedMounts := []string{}
	for _, mount := range mounts {
		if strings.HasPrefix(mount, cfg.DockerMountRootDir) {
			mount = strings.Replace(mount, cfg.DockerMountRootDir, cfg.DockerHostRootDir, 1)
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
