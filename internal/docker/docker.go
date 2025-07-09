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
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

// Command is the string representation of a command to be passed to the language-specific
// container's entry point as the first argument.
type Command string

// The set of commands passed to the language container, in a single place to avoid typos.
const (
	// CommandGenerate performs generation for a configured library.
	CommandGenerate Command = "generate"
	// CommandBuild builds a library.
	CommandBuild Command = "build"
	// CommandConfigure configures a new API as a library.
	CommandConfigure Command = "configure"
)

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

// Generate performs generation for an API which is configured as part of a library.
// apiRoot specifies the root directory of the API specification repo,
// output specifies the empty output directory into which the command should
// generate code, and libraryID specifies the ID of the library to generate,
// as configured in the Librarian state file for the repository.
func (c *Docker) Generate(ctx context.Context, cfg *config.Config, apiRoot, output, generatorInput, libraryID string) error {
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

	return c.runDocker(ctx, cfg, CommandGenerate, mounts, commandArgs)
}

// Build builds the library with an ID of libraryID, as configured in
// the Librarian state file for the repository with a root of repoRoot.
func (c *Docker) Build(ctx context.Context, cfg *config.Config, repoRoot, libraryID string) error {
	mounts := []string{
		fmt.Sprintf("%s:/repo", repoRoot),
	}
	commandArgs := []string{
		"--repo-root=/repo",
		"--test=true",
		fmt.Sprintf("--library-id=%s", libraryID),
	}

	return c.runDocker(ctx, cfg, CommandBuild, mounts, commandArgs)
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
	args = append(args, c.Image)
	args = append(args, string(command))
	args = append(args, commandArgs...)
	return c.run(args...)
}

func maybeRelocateMounts(cfg *config.Config, mounts []string) []string {
	// When running in Kokoro, we'll be running sibling containers.
	// Make sure we specify the "from" part of the mount as the host directory.
	if cfg.HostMount == "" {
		return mounts
	}

	relocatedMounts := []string{}
	hostMount := strings.Split(cfg.HostMount, ":")
	for _, mount := range mounts {
		if strings.HasPrefix(mount, hostMount[0]) {
			mount = strings.Replace(mount, hostMount[0], hostMount[1], 1)
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
