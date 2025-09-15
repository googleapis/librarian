// Copyright 2025 Google LLC
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

package librarian

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/gitrepo"
)

const (
	generateCmdName = "generate"
)

var cmdGenerate = &cli.Command{
	Short:     "generate onboards and generates client library code",
	UsageLine: "librarian generate [flags]",
	Long: `The generate command is the primary tool for all code generation
tasks. It handles both the initial setup of a new library (onboarding) and the
regeneration of existing ones. Librarian works by delegating language-specific
tasks to a container, which is configured in the .librarian/state.yaml file.
Librarian is environment aware and will check if the current directory is the
root of a librarian repository. If you are not executing in such a directory the
'--repo' flag must be provided.

# Onboarding a new library

To configure and generate a new library for the first time, you must specify the
API to be generated and the library it will belong to. Librarian will invoke the
'configure' command in the language container to set up the repository, add the
new library's configuration to the '.librarian/state.yaml' file, and then
proceed with generation.

Example:
  librarian generate --library=secretmanager --api=google/cloud/secretmanager/v1

# Regenerating existing libraries

You can regenerate a single, existing library by specifying either the library
ID or the API path. If no specific library or API is provided, Librarian will
regenerate all libraries listed in '.librarian/state.yaml'. If '--library' or
'--api' is specified the whole library will be regenerated.

Examples:
  # Regenerate a single library by its ID
  librarian generate --library=secretmanager

  # Regenerate a single library by its API path
  librarian generate --api=google/cloud/secretmanager/v1

  # Regenerate all libraries in the repository
  librarian generate

# Workflow and Options:

The generation process involves delegating to the language container's
'generate' command. After the code is generated, the tool cleans the destination
directories and copies the new files into place, according to the configuration
in '.librarian/state.yaml'.

- If the '--build' flag is specified, the 'build' command is also executed in
  the container to compile and validate the generated code.
- If the '--push' flag is provided, the changes are committed to a new branch,
  and a pull request is created on GitHub. Otherwise, the changes are left in
  your local working directory for inspection.

Example with build and push:
  SDK_LIBRARIAN_GITHUB_TOKEN=xxx librarian generate --push --build`,
	Run: func(ctx context.Context, cfg *config.Config) error {
		runner, err := newGenerateRunner(cfg)
		if err != nil {
			return err
		}
		return runner.run(ctx)
	},
}

func init() {
	cmdGenerate.Init()
	fs := cmdGenerate.Flags
	cfg := cmdGenerate.Config

	addFlagAPI(fs, cfg)
	addFlagAPISource(fs, cfg)
	addFlagBuild(fs, cfg)
	addFlagHostMount(fs, cfg)
	addFlagImage(fs, cfg)
	addFlagLibrary(fs, cfg)
	addFlagRepo(fs, cfg)
	addFlagBranch(fs, cfg)
	addFlagWorkRoot(fs, cfg)
	addFlagPush(fs, cfg)
}

type generateRunner struct {
	api             string
	apiSource       string
	branch          string
	build           bool
	commit          bool
	containerClient ContainerClient
	ghClient        GitHubClient
	hostMount       string
	image           string
	library         string
	push            bool
	repo            gitrepo.Repository
	sourceRepo      gitrepo.Repository
	state           *config.LibrarianState
	workRoot        string
}

func newGenerateRunner(cfg *config.Config) (*generateRunner, error) {
	runner, err := newCommandRunner(cfg)
	if err != nil {
		return nil, err
	}
	return &generateRunner{
		api:             cfg.API,
		apiSource:       cfg.APISource,
		build:           cfg.Build,
		commit:          cfg.Commit,
		containerClient: runner.containerClient,
		ghClient:        runner.ghClient,
		hostMount:       cfg.HostMount,
		image:           runner.image,
		library:         cfg.Library,
		push:            cfg.Push,
		repo:            runner.repo,
		sourceRepo:      runner.sourceRepo,
		state:           runner.state,
		workRoot:        runner.workRoot,
	}, nil
}

// run executes the library generation process.
//
// It determines whether to generate a single library or all configured libraries based on the
// command-line flags. If an API or library is specified, it generates a single library. Otherwise,
// it iterates through all libraries defined in the state and generates them.
func (r *generateRunner) run(ctx context.Context) error {
	outputDir := filepath.Join(r.workRoot, "output")
	if err := os.Mkdir(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to make output directory, %s: %w", outputDir, err)
	}
	// The last generated commit is changed after library generation,
	// use this map to keep the mapping from library id to commit sha before the
	// generation since we need these commits to create pull request body.
	idToCommits := make(map[string]string, 0)
	var failedLibraries []string
	if r.api != "" || r.library != "" {
		libraryID := r.library
		if libraryID == "" {
			libraryID = findLibraryIDByAPIPath(r.state, r.api)
		}
		oldCommit, err := r.generateSingleLibrary(ctx, libraryID, outputDir)
		if err != nil {
			return err
		}
		idToCommits[libraryID] = oldCommit
	} else {
		succeededGenerations := 0
		failedGenerations := 0
		for _, library := range r.state.Libraries {
			oldCommit, err := r.generateSingleLibrary(ctx, library.ID, outputDir)
			if err != nil {
				slog.Error("failed to generate library", "id", library.ID, "err", err)
				failedLibraries = append(failedLibraries, library.ID)
				failedGenerations++
			} else {
				// Only add the mapping if library generation is successful so that
				// failed library will not appear in generation PR body.
				idToCommits[library.ID] = oldCommit
				succeededGenerations++
			}
		}

		slog.Info(
			"generation statistics",
			"all", len(r.state.Libraries),
			"successes", succeededGenerations,
			"failures", failedGenerations)
		if failedGenerations > 0 && failedGenerations == len(r.state.Libraries) {
			return fmt.Errorf("all %d libraries failed to generate", failedGenerations)
		}
	}

	if err := saveLibrarianState(r.repo.GetDir(), r.state); err != nil {
		return err
	}

	commitInfo := &commitInfo{
		branch:          r.branch,
		commit:          r.commit,
		commitMessage:   "feat: generate libraries",
		failedLibraries: failedLibraries,
		ghClient:        r.ghClient,
		idToCommits:     idToCommits,
		prType:          generate,
		push:            r.push,
		repo:            r.repo,
		sourceRepo:      r.sourceRepo,
		state:           r.state,
	}

	return commitAndPush(ctx, commitInfo)
}

// generateSingleLibrary manages the generation of a single client library.
//
// The single library generation executes as follows:
//
// 1. Configure the library, if the library is not configured in the state.yaml.
//
// 2. Generate the library.
//
// 3. Build the library.
//
// 4. Update the last generated commit.
//
// Returns the last generated commit before the generation and error, if any.
func (r *generateRunner) generateSingleLibrary(ctx context.Context, libraryID, outputDir string) (string, error) {
	if r.needsConfigure() {
		slog.Info("library not configured, start initial configuration", "library", r.library)
		configuredLibraryID, err := r.runConfigureCommand(ctx)
		if err != nil {
			return "", err
		}
		libraryID = configuredLibraryID
	}

	// At this point, we should have a library in the state.
	libraryState := findLibraryByID(r.state, libraryID)
	if libraryState == nil {
		return "", fmt.Errorf("library %q not configured yet, generation stopped", libraryID)
	}
	lastGenCommit := libraryState.LastGeneratedCommit

	if len(libraryState.APIs) == 0 {
		slog.Info("library has no APIs; skipping generation", "library", libraryID)
		return "", nil
	}

	// For each library, create a separate output directory. This avoids
	// libraries interfering with each other, and makes it easier to see what
	// was generated for each library when debugging.
	libraryOutputDir := filepath.Join(outputDir, libraryID)
	if err := os.MkdirAll(libraryOutputDir, 0755); err != nil {
		return "", err
	}

	generatedLibraryID, err := r.runGenerateCommand(ctx, libraryID, libraryOutputDir)
	if err != nil {
		return "", err
	}

	if err := r.runBuildCommand(ctx, generatedLibraryID); err != nil {
		return "", err
	}

	if err := r.updateLastGeneratedCommitState(generatedLibraryID); err != nil {
		return "", err
	}

	return lastGenCommit, nil
}

func (r *generateRunner) needsConfigure() bool {
	return r.api != "" && r.library != "" && findLibraryByID(r.state, r.library) == nil
}

func (r *generateRunner) updateLastGeneratedCommitState(libraryID string) error {
	hash, err := r.sourceRepo.HeadHash()
	if err != nil {
		return err
	}
	for _, l := range r.state.Libraries {
		if l.ID == libraryID {
			l.LastGeneratedCommit = hash
			break
		}
	}
	return nil
}

// runGenerateCommand attempts to perform generation for an API. It then cleans the
// destination directory and copies the newly generated files into it.
//
// If successful, it returns the ID of the generated library; otherwise, it
// returns an empty string and an error.
func (r *generateRunner) runGenerateCommand(ctx context.Context, libraryID, outputDir string) (string, error) {
	apiRoot, err := filepath.Abs(r.sourceRepo.GetDir())
	if err != nil {
		return "", err
	}

	generateRequest := &docker.GenerateRequest{
		Cfg:       &config.Config{HostMount: r.hostMount},
		State:     r.state,
		ApiRoot:   apiRoot,
		LibraryID: libraryID,
		Output:    outputDir,
		RepoDir:   r.repo.GetDir(),
	}
	slog.Info("Performing generation for library", "id", libraryID, "outputDir", outputDir)
	if err := r.containerClient.Generate(ctx, generateRequest); err != nil {
		return "", err
	}

	// Read the library state from the response.
	if _, err := readLibraryState(
		filepath.Join(generateRequest.RepoDir, config.LibrarianDir, config.GenerateResponse)); err != nil {
		return "", err
	}

	if err := cleanAndCopyLibrary(r.state, r.repo.GetDir(), libraryID, outputDir); err != nil {
		return "", err
	}

	slog.Info("Generation succeeds", "id", libraryID)
	return libraryID, nil
}

// runBuildCommand orchestrates the building of an API library using a containerized
// environment.
//
// The `outputDir` parameter specifies the target directory where the built artifacts
// should be placed.
func (r *generateRunner) runBuildCommand(ctx context.Context, libraryID string) error {
	if !r.build {
		slog.Info("Build flag not specified, skipping")
		return nil
	}
	if libraryID == "" {
		slog.Warn("Cannot perform build, missing library ID")
		return nil
	}

	buildRequest := &docker.BuildRequest{
		Cfg:       &config.Config{HostMount: r.hostMount},
		State:     r.state,
		LibraryID: libraryID,
		RepoDir:   r.repo.GetDir(),
	}
	slog.Info("Performing build for library", "id", libraryID)
	if err := r.containerClient.Build(ctx, buildRequest); err != nil {
		return err
	}

	// Read the library state from the response.
	if _, err := readLibraryState(
		filepath.Join(buildRequest.RepoDir, config.LibrarianDir, config.BuildResponse)); err != nil {
		return err
	}

	slog.Info("Build succeeds", "id", libraryID)
	return nil
}

// runConfigureCommand executes the container's "configure" command for an API.
//
// This function performs the following steps:
//
// 1. Constructs a request for the language-specific container, including the API
// root, library ID, and repository directory.
//
// 2. Populates a service configuration if one is missing.
//
// 3. Delegates the configuration task to the container's `Configure` command.
//
// 4. Reads the updated library state from the `configure-response.json` file
// generated by the container.
//
// 5. Updates the in-memory librarian state with the new configuration.
//
// 6. Writes the complete, updated librarian state back to the `state.yaml` file
// in the repository.
//
// If successful, it returns the ID of the newly configured library; otherwise,
// it returns an empty string and an error.
func (r *generateRunner) runConfigureCommand(ctx context.Context) (string, error) {

	apiRoot, err := filepath.Abs(r.apiSource)
	if err != nil {
		return "", err
	}

	setAllAPIStatus(r.state, config.StatusExisting)
	// Record to state, not write to state.yaml
	r.state.Libraries = append(r.state.Libraries, &config.LibraryState{
		ID:   r.library,
		APIs: []*config.API{{Path: r.api, Status: config.StatusNew}},
	})

	if err := populateServiceConfigIfEmpty(
		r.state,
		r.apiSource); err != nil {
		return "", err
	}

	configureRequest := &docker.ConfigureRequest{
		Cfg:       &config.Config{HostMount: r.hostMount},
		State:     r.state,
		ApiRoot:   apiRoot,
		LibraryID: r.library,
		RepoDir:   r.repo.GetDir(),
	}
	slog.Info("Performing configuration for library", "id", r.library)
	if _, err := r.containerClient.Configure(ctx, configureRequest); err != nil {
		return "", err
	}

	// Read the new library state from the response.
	libraryState, err := readLibraryState(
		filepath.Join(r.repo.GetDir(), config.LibrarianDir, config.ConfigureResponse),
	)
	if err != nil {
		return "", err
	}
	if libraryState == nil {
		return "", errors.New("no response file for configure container command")
	}

	if libraryState.Version == "" {
		slog.Info("library doesn't receive a version, apply the default version", "id", r.library)
		libraryState.Version = "0.0.0"
	}

	// Update the library state in the librarian state.
	for i, library := range r.state.Libraries {
		if library.ID != libraryState.ID {
			continue
		}
		r.state.Libraries[i] = libraryState
	}

	return libraryState.ID, nil
}

func setAllAPIStatus(state *config.LibrarianState, status string) {
	for _, library := range state.Libraries {
		for _, api := range library.APIs {
			api.Status = status
		}
	}
}
