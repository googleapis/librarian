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
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
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

	// The user ID to run the container as.
	uid string

	// The group ID to run the container as.
	gid string

	// run runs the docker command.
	run func(args ...string) error
}

// New constructs a Docker instance which will invoke the specified
// Docker image as required to implement language-specific commands,
// providing the container with required environment variables.
func New(workRoot, image, secretsProject, uid, gid string, pipelineConfig *config.PipelineConfig) (*Docker, error) {
	envProvider := newEnvironmentProvider(workRoot, secretsProject, pipelineConfig)
	docker := &Docker{
		Image: image,
		env:   envProvider,
		uid:   uid,
		gid:   gid,
	}
	docker.run = func(args ...string) error {
		return docker.runCommand("docker", args...)
	}
	return docker, nil
}

// GenerateRaw performs generation for an API not configured in a library.
// This does not have any context from a language repo: it requires
// generation purely on the basis of the API specification, which is in
// the subdirectory apiPath of the API specification repo apiRoot, and whatever
// is in the language-specific Docker container. The code is generated
// in the output directory, which is initially empty.
func (c *Docker) GenerateRaw(ctx context.Context, cfg *config.Config, apiRoot, output, apiPath string) error {
	commandArgs := []string{
		"--source=/apis",
		"--output=/output",
		fmt.Sprintf("--api=%s", apiPath),
	}
	mounts := []string{
		fmt.Sprintf("%s:/apis", apiRoot),
		fmt.Sprintf("%s:/output", output),
	}

	return c.runDocker(ctx, cfg, CommandGenerateRaw, mounts, commandArgs)
}

// GenerateLibrary performs generation for an API which is configured as part of a library.
// apiRoot specifies the root directory of the API specification repo,
// output specifies the empty output directory into which the command should
// generate code, and libraryID specifies the ID of the library to generate,
// as configured in the Librarian state file for the repository.
func (c *Docker) GenerateLibrary(ctx context.Context, cfg *config.Config, apiRoot, output, generatorInput, libraryID string) error {
	commandArgs := []string{
		"--source=/apis",
		"--output=/output",
		fmt.Sprintf("--%s=/%s", config.GeneratorInputDir, config.GeneratorInputDir),
		fmt.Sprintf("--library-id=%s", libraryID),
	}
	mounts := []string{
		fmt.Sprintf("%s:/apis", apiRoot),
		fmt.Sprintf("%s:/output", output),
		fmt.Sprintf("%s:/%s", generatorInput, config.GeneratorInputDir),
	}

	return c.runDocker(ctx, cfg, CommandGenerateLibrary, mounts, commandArgs)
}

// Clean deletes files within repoRoot which are generated for library
// libraryID, as configured in the Librarian state file for the repository.
func (c *Docker) Clean(ctx context.Context, cfg *config.Config, repoRoot, libraryID string) error {
	mounts := []string{
		fmt.Sprintf("%s:/repo", repoRoot),
	}
	commandArgs := []string{
		"--repo-root=/repo",
		fmt.Sprintf("--library-id=%s", libraryID),
	}

	return c.runDocker(ctx, cfg, CommandClean, mounts, commandArgs)
}

// BuildRaw builds the result of GenerateRaw, which previously generated
// code for apiPath in generatorOutput.
func (c *Docker) BuildRaw(ctx context.Context, cfg *config.Config, generatorOutput, apiPath string) error {
	mounts := []string{
		fmt.Sprintf("%s:/generator-output", generatorOutput),
	}
	commandArgs := []string{
		"--generator-output=/generator-output",
		fmt.Sprintf("--api=%s", apiPath),
	}

	return c.runDocker(ctx, cfg, CommandBuildRaw, mounts, commandArgs)
}

// BuildLibrary builds the library with an ID of libraryID, as configured in
// the Librarian state file for the repository with a root of repoRoot.
func (c *Docker) BuildLibrary(ctx context.Context, cfg *config.Config, repoRoot, libraryID string) error {
	mounts := []string{
		fmt.Sprintf("%s:/repo", repoRoot),
	}
	commandArgs := []string{
		"--repo-root=/repo",
		"--test=true",
		fmt.Sprintf("--library-id=%s", libraryID),
	}

	return c.runDocker(ctx, cfg, CommandBuildLibrary, mounts, commandArgs)
}

// Configure configures an API within a repository, either adding it to an
// existing library or creating a new library. The API is indicated by the
// apiPath directory within apiRoot, and the container is provided with the
// generatorInput directory to record the results of configuration. The
// library code is not generated.
func (c *Docker) Configure(ctx context.Context, cfg *config.Config, apiRoot, apiPath, generatorInput string) error {
	commandArgs := []string{
		"--source=/apis",
		fmt.Sprintf("--%s=/%s", config.GeneratorInputDir, config.GeneratorInputDir),
		fmt.Sprintf("--api=%s", apiPath),
	}
	mounts := []string{
		fmt.Sprintf("%s:/apis", apiRoot),
		fmt.Sprintf("%s:/%s", generatorInput, config.GeneratorInputDir),
	}

	return c.runDocker(ctx, cfg, CommandConfigure, mounts, commandArgs)
}

// PrepareLibraryRelease prepares the repository languageRepo for the release of a library with
// ID libraryID within repoRoot, with version releaseVersion. Release notes
// are expected to be present within inputsDirectory, in a file named
// `{libraryID}-{releaseVersion}-release-notes.txt`.
func (c *Docker) PrepareLibraryRelease(ctx context.Context, cfg *config.Config, repoRoot, inputsDirectory, libraryID, releaseVersion string) error {
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

	return c.runDocker(ctx, cfg, CommandPrepareLibraryRelease, mounts, commandArgs)
}

// IntegrationTestLibrary runs the integration tests for a library with ID libraryID within repoRoot.
func (c *Docker) IntegrationTestLibrary(ctx context.Context, cfg *config.Config, repoRoot, libraryID string) error {
	commandArgs := []string{
		"--repo-root=/repo",
		fmt.Sprintf("--library-id=%s", libraryID),
	}
	mounts := []string{
		fmt.Sprintf("%s:/repo", repoRoot),
	}

	return c.runDocker(ctx, cfg, CommandIntegrationTestLibrary, mounts, commandArgs)
}

// PackageLibrary packages release artifacts for a library with ID libraryID within repoRoot,
// creating the artifacts within output.
func (c *Docker) PackageLibrary(ctx context.Context, cfg *config.Config, repoRoot, libraryID, output string) error {
	commandArgs := []string{
		"--repo-root=/repo",
		"--output=/output",
		fmt.Sprintf("--library-id=%s", libraryID),
	}
	mounts := []string{
		fmt.Sprintf("%s:/repo", repoRoot),
		fmt.Sprintf("%s:/output", output),
	}

	return c.runDocker(ctx, cfg, CommandPackageLibrary, mounts, commandArgs)
}

// PublishLibrary publishes release artifacts for a library with ID libraryID and version releaseVersion
// to package managers, documentation sites etc. The artifacts will previously have been
// created by PackageLibrary.
func (c *Docker) PublishLibrary(ctx context.Context, cfg *config.Config, output, libraryID, releaseVersion string) error {
	commandArgs := []string{
		"--package-output=/output",
		fmt.Sprintf("--library-id=%s", libraryID),
		fmt.Sprintf("--version=%s", releaseVersion),
	}
	mounts := []string{
		fmt.Sprintf("%s:/output", output),
	}

	return c.runDocker(ctx, cfg, CommandPublishLibrary, mounts, commandArgs)
}

func (c *Docker) runDocker(ctx context.Context, cfg *config.Config, command Command, mounts []string, commandArgs []string) (err error) {
	mounts = maybeRelocateMounts(cfg, mounts)

	args := []string{
		"run",
		"--rm", // Automatically delete the container after completion
	}

	for _, mount := range mounts {
		args = append(args, "-v", mount)
	}

	// Run as the current user in the container - primarily so that any files
	// we create end up being owned by the current user (and easily deletable).
	if c.uid != "" && c.gid != "" {
		args = append(args, "--user", fmt.Sprintf("%s:%s", c.uid, c.gid))
	}

	if c.env != nil {
		if err := c.env.writeEnvironmentFile(ctx, string(command)); err != nil {
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

func (c *Docker) runCommand(cmdName string, args ...string) error {
	cmd := exec.Command(cmdName, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	slog.Info(fmt.Sprintf("=== Docker start %s", strings.Repeat("=", 63)))
	slog.Info(cmd.String())
	slog.Info(strings.Repeat("-", 80))
	err := cmd.Run()
	slog.Info(fmt.Sprintf("=== Docker end %s", strings.Repeat("=", 65)))
	return err
}
