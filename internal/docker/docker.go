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
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

// Command is the string representation of a command to be passed to the language-specific
// container's entry point as the first argument.
type Command string

// The set of commands passed to the language container, in a single place to avoid typos.
const (
	// CommandBuild builds a library.
	CommandBuild Command = "build"
	// CommandConfigure configures a new API as a library.
	CommandConfigure Command = "configure"
	// CommandGenerate performs generation for a configured library.
	CommandGenerate Command = "generate"
	// CommandReleaseInit performs release for a library.
	CommandReleaseInit Command = "release-init"
)

// Docker contains all the information required to run language-specific
// Docker containers.
type Docker struct {
	// The Docker image to run.
	Image string

	// The user ID to run the container as.
	uid string

	// The group ID to run the container as.
	gid string

	// run runs the docker command.
	run func(args ...string) error
}

// BuildRequest contains all the information required for a language
// container to run the build command.
type BuildRequest struct {
	// HostMount specifies a mount point from the Docker host into the Docker
	// container. The format is "{host-dir}:{local-dir}".
	HostMount string

	// LibraryID specifies the ID of the library to build.
	LibraryID string

	// RepoDir is the local root directory of the language repository.
	RepoDir string

	// State is a pointer to the [config.LibrarianState] struct, representing
	// the overall state of the generation and release pipeline.
	State *config.LibrarianState
}

// ConfigureRequest contains all the information required for a language
// container to run the configure command.
type ConfigureRequest struct {
	// ApiRoot specifies the root directory of the API specification repo.
	ApiRoot string

	// HostMount specifies a mount point from the Docker host into the Docker
	// container. The format is "{host-dir}:{local-dir}".
	HostMount string

	// libraryID specifies the ID of the library to configure.
	LibraryID string

	// RepoDir is the local root directory of the language repository.
	RepoDir string

	// ExistingSourceRoots are existing source roots in the language repository.
	ExistingSourceRoots []string

	// GlobalFiles are global files of the language repository.
	GlobalFiles []string

	// State is a pointer to the [config.LibrarianState] struct, representing
	// the overall state of the generation and release pipeline.
	State *config.LibrarianState
}

// GenerateRequest contains all the information required for a language
// container to run the generate command.
type GenerateRequest struct {
	// ApiRoot specifies the root directory of the API specification repo.
	ApiRoot string

	// HostMount specifies a mount point from the Docker host into the Docker
	// container. The format is "{host-dir}:{local-dir}".
	HostMount string

	// LibraryID specifies the ID of the library to generate.
	LibraryID string

	// Output specifies the empty output directory into which the command should
	// generate code
	Output string

	// RepoDir is the local root directory of the language repository.
	RepoDir string

	// State is a pointer to the [config.LibrarianState] struct, representing
	// the overall state of the generation and release pipeline.
	State *config.LibrarianState
}

// ReleaseInitRequest contains all the information required for a language
// container to run the  init command.
type ReleaseInitRequest struct {
	// Branch is the remote branch of the language repository to use.
	Branch string

	// Commit determines whether to create a commit for the release but not
	// create a pull request. This flag is ignored if Push is set to true.
	Commit bool

	// HostMount is used to remap Docker mount paths when running in environments
	// where Docker containers are siblings (e.g., Kokoro).
	// It specifies a mount point from the Docker host into the Docker container.
	// The format is "{host-dir}:{local-dir}".
	HostMount string

	// LibrarianConfig is a pointer to the [config.LibrarianConfig] struct, holding
	// global files configuration in a language repository.
	LibrarianConfig *config.LibrarianConfig

	// LibraryID specifies the ID of the library to release.
	LibraryID string

	// LibraryVersion specifies the version of the library to release.
	LibraryVersion string

	// Output specifies the empty output directory into which the command should
	// generate code.
	Output string

	// PartialRepoDir is the local root directory of language repository contains
	// files that make up libraries and global files.
	// This is the directory that container can access.
	PartialRepoDir string

	// Push determines whether to push changes to GitHub.
	Push bool

	// State is a pointer to the [config.LibrarianState] struct, representing
	// the overall state of the generation and release pipeline.
	State *config.LibrarianState
}

// New constructs a Docker instance which will invoke the specified
// Docker image as required to implement language-specific commands,
// providing the container with required environment variables.
func New(workRoot, image, uid, gid string) (*Docker, error) {
	docker := &Docker{
		Image: image,
		uid:   uid,
		gid:   gid,
	}
	docker.run = func(args ...string) error {
		return docker.runCommand("docker", args...)
	}
	return docker, nil
}

// Generate performs generation for an API which is configured as part of a
// library.
func (c *Docker) Generate(ctx context.Context, request *GenerateRequest) error {
	reqFilePath := filepath.Join(request.RepoDir, config.LibrarianDir, config.GenerateRequest)
	if err := writeLibraryState(request.State, request.LibraryID, reqFilePath); err != nil {
		return err
	}
	defer func() {
		if b, err := os.ReadFile(reqFilePath); err == nil {
			slog.Debug("generate request", "content", string(b))
		}
		err := os.Remove(reqFilePath)
		if err != nil {
			slog.Warn("fail to remove file", slog.String("name", reqFilePath), slog.Any("err", err))
		}
	}()

	commandArgs := []string{
		"--librarian=/librarian",
		"--input=/input",
		"--output=/output",
		"--source=/source",
	}

	generatorInput := filepath.Join(request.RepoDir, config.GeneratorInputDir)
	librarianDir := filepath.Join(request.RepoDir, config.LibrarianDir)
	mounts := []string{
		fmt.Sprintf("%s:/librarian", librarianDir),
		fmt.Sprintf("%s:/input", generatorInput),
		fmt.Sprintf("%s:/output", request.Output),
		fmt.Sprintf("%s:/source:ro", request.ApiRoot), // readonly volume
	}
	return c.runDocker(ctx, request.HostMount, CommandGenerate, mounts, commandArgs)
}

// Build builds the library with an ID of libraryID, as configured in
// the Librarian state file for the repository with a root of repoRoot.
func (c *Docker) Build(ctx context.Context, request *BuildRequest) error {
	reqFilePath := filepath.Join(request.RepoDir, config.LibrarianDir, config.BuildRequest)
	if err := writeLibraryState(request.State, request.LibraryID, reqFilePath); err != nil {
		return err
	}
	defer func() {
		if b, err := os.ReadFile(reqFilePath); err == nil {
			slog.Debug("build request", "content", string(b))
		}
		err := os.Remove(reqFilePath)
		if err != nil {
			slog.Warn("fail to remove file", slog.String("name", reqFilePath), slog.Any("err", err))
		}
	}()

	librarianDir := filepath.Join(request.RepoDir, config.LibrarianDir)
	mounts := []string{
		fmt.Sprintf("%s:/librarian", librarianDir),
		fmt.Sprintf("%s:/repo", request.RepoDir),
	}
	commandArgs := []string{
		"--librarian=/librarian",
		"--repo=/repo",
	}

	return c.runDocker(ctx, request.HostMount, CommandBuild, mounts, commandArgs)
}

// Configure configures an API within a repository, either adding it to an
// existing library or creating a new library.
//
// Returns the configured library id if the command succeeds.
func (c *Docker) Configure(ctx context.Context, request *ConfigureRequest) (string, error) {
	reqFilePath := filepath.Join(request.RepoDir, config.LibrarianDir, config.ConfigureRequest)
	if err := writeLibrarianState(request.State, reqFilePath); err != nil {
		return "", err
	}
	defer func() {
		if b, err := os.ReadFile(reqFilePath); err == nil {
			slog.Debug("configure request", "content", string(b))
		}
		err := os.Remove(reqFilePath)
		if err != nil {
			slog.Warn("fail to remove file", slog.String("name", reqFilePath), slog.Any("err", err))
		}
	}()
	commandArgs := []string{
		"--librarian=/librarian",
		"--input=/input",
		"--repo=/repo",
		"--source=/source",
	}
	generatorInput := filepath.Join(request.RepoDir, config.GeneratorInputDir)
	librarianDir := filepath.Join(request.RepoDir, config.LibrarianDir)
	mounts := []string{
		fmt.Sprintf("%s:/librarian", librarianDir),
		fmt.Sprintf("%s:/input", generatorInput),
		fmt.Sprintf("%s:/source:ro", request.ApiRoot), // readonly volume
	}
	// Mount existing source roots as a readonly volume.
	for _, sourceRoot := range request.ExistingSourceRoots {
		mounts = append(mounts, fmt.Sprintf("%s/%s:/repo/%s:ro", request.RepoDir, sourceRoot, sourceRoot))
	}
	// Mount global files as a readonly volume.
	for _, globalFile := range request.GlobalFiles {
		mounts = append(mounts, fmt.Sprintf("%s/%s:/repo/%s:ro", request.RepoDir, globalFile, globalFile))
	}

	if err := c.runDocker(ctx, request.HostMount, CommandConfigure, mounts, commandArgs); err != nil {
		return "", err
	}

	return request.LibraryID, nil
}

// ReleaseInit initiates a release for a given language repository.
func (c *Docker) ReleaseInit(ctx context.Context, request *ReleaseInitRequest) error {
	reqFilePath := filepath.Join(request.PartialRepoDir, config.LibrarianDir, config.ReleaseInitRequest)
	if err := writeLibrarianState(request.State, reqFilePath); err != nil {
		return err
	}
	defer func() {
		if b, err := os.ReadFile(reqFilePath); err == nil {
			slog.Debug("release init request", "content", string(b))
		}
		err := os.Remove(reqFilePath)
		if err != nil {
			slog.Warn("fail to remove file", slog.String("name", reqFilePath), slog.Any("err", err))
		}
	}()
	commandArgs := []string{
		"--librarian=/librarian",
		"--repo=/repo",
		"--output=/output",
	}

	librarianDir := filepath.Join(request.PartialRepoDir, config.LibrarianDir)
	mounts := []string{
		fmt.Sprintf("%s:/librarian", librarianDir),
		fmt.Sprintf("%s:/repo:ro", request.PartialRepoDir), // readonly volume
		fmt.Sprintf("%s:/output", request.Output),
	}

	if err := c.runDocker(ctx, request.HostMount, CommandReleaseInit, mounts, commandArgs); err != nil {
		return err
	}

	return nil
}

func (c *Docker) runDocker(_ context.Context, hostMount string, command Command, mounts []string, commandArgs []string) (err error) {
	mounts = maybeRelocateMounts(hostMount, mounts)
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

	args = append(args, c.Image)
	args = append(args, string(command))
	args = append(args, commandArgs...)
	return c.run(args...)
}

func maybeRelocateMounts(hostMount string, mounts []string) []string {
	// When running in Kokoro, we'll be running sibling containers.
	// Make sure we specify the "from" part of the mount as the host directory.
	if hostMount == "" {
		return mounts
	}

	relocatedMounts := []string{}
	hostMountParts := strings.Split(hostMount, ":")
	for _, mount := range mounts {
		if strings.HasPrefix(mount, hostMountParts[0]) {
			mount = strings.Replace(mount, hostMountParts[0], hostMountParts[1], 1)
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

func writeLibraryState(state *config.LibrarianState, libraryID, jsonFilePath string) error {
	if err := os.MkdirAll(filepath.Dir(jsonFilePath), 0755); err != nil {
		return fmt.Errorf("failed to make directory: %w", err)
	}
	jsonFile, err := os.Create(jsonFilePath)
	if err != nil {
		return fmt.Errorf("failed to create JSON file: %w", err)
	}
	defer jsonFile.Close()

	for _, library := range state.Libraries {
		if library.ID != libraryID {
			continue
		}

		data, err := json.MarshalIndent(library, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal state to JSON: %w", err)
		}
		_, err = jsonFile.Write(data)
		if err != nil {
			return fmt.Errorf("failed to write generate request JSON file: %w", err)
		}
	}

	return nil
}

func writeLibrarianState(state *config.LibrarianState, jsonFilePath string) error {
	if err := os.MkdirAll(filepath.Dir(jsonFilePath), 0755); err != nil {
		return fmt.Errorf("failed to make directory: %w", err)
	}
	jsonFile, err := os.Create(jsonFilePath)
	if err != nil {
		return fmt.Errorf("failed to create JSON file: %w", err)
	}
	defer jsonFile.Close()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state to JSON: %w", err)
	}

	_, err = jsonFile.Write(data)

	return err
}
