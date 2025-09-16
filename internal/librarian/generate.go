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
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

const (
	generateCmdName = "generate"
)

type generateRunner struct {
	api             string
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
	librarianConfig *config.LibrarianConfig
	workRoot        string
}

const defaultAPISourceBranch = "master"

func newGenerateRunner(cfg *config.Config) (*generateRunner, error) {
	languageRepo, err := cloneOrOpenRepo(cfg.WorkRoot, cfg.Repo, cfg.APISourceDepth, cfg.Branch, cfg.CI, cfg.GitHubToken)
	if err != nil {
		return nil, err
	}
	sourceRepo, err := cloneOrOpenRepo(cfg.WorkRoot, cfg.APISource, cfg.APISourceDepth, defaultAPISourceBranch, cfg.CI, cfg.GitHubToken)
	if err != nil {
		return nil, err
	}
	sourceRepoDir := sourceRepo.GetDir()
	state, err := loadRepoState(languageRepo, sourceRepoDir)
	if err != nil {
		return nil, err
	}

	image := deriveImage(cfg.Image, state)
	container, err := docker.New(cfg.WorkRoot, image, cfg.UserUID, cfg.UserGID)
	if err != nil {
		return nil, err
	}

	ghClient, err := newGitHubClient(cfg.Repo, cfg.GitHubToken, languageRepo)
	if err != nil {
		return nil, err
	}

	return &generateRunner{
		api:             cfg.API,
		branch:          cfg.Branch,
		build:           cfg.Build,
		commit:          cfg.Commit,
		containerClient: container,
		ghClient:        ghClient,
		hostMount:       cfg.HostMount,
		image:           image,
		librarianConfig: cfg.LibrarianConfig,
		library:         cfg.Library,
		push:            cfg.Push,
		repo:            languageRepo,
		sourceRepo:      sourceRepo,
		state:           state,
		workRoot:        cfg.WorkRoot,
	}, nil
}

func newGitHubClient(repo, token string, languageRepo *gitrepo.LocalRepository) (_ GitHubClient, err error) {
	var gitRepo *github.Repository
	if isURL(repo) {
		gitRepo, err = github.ParseRemote(repo)
		if err != nil {
			return nil, fmt.Errorf("failed to parse repo url: %w", err)
		}
	} else {
		gitRepo, err = github.FetchGitHubRepoFromRemote(languageRepo)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitHub repo from remote: %w", err)
		}
	}
	return github.NewClient(token, gitRepo), nil
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
	idToCommits := make(map[string]string)
	var failedLibraries []string
	failedGenerations := 0
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
		blockedGenerations := 0
		for _, library := range r.state.Libraries {
			if r.librarianConfig != nil {
				libConfig := r.librarianConfig.LibraryConfigFor(library.ID)
				if libConfig != nil && libConfig.GenerateBlocked {
					slog.Info("library has generate_blocked, skipping", "id", library.ID)
					blockedGenerations++
					continue
				}
			}
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
			"blocked", blockedGenerations,
			"failures", failedGenerations)
		if failedGenerations > 0 && failedGenerations+blockedGenerations == len(r.state.Libraries) {
			return fmt.Errorf("all %d libraries failed to generate (blocked: %d)",
				failedGenerations, blockedGenerations)
		}
	}

	if err := saveLibrarianState(r.repo.GetDir(), r.state); err != nil {
		return err
	}

	commitInfo := &commitInfo{
		branch:            r.branch,
		commit:            r.commit,
		commitMessage:     "feat: generate libraries",
		failedLibraries:   failedLibraries,
		ghClient:          r.ghClient,
		idToCommits:       idToCommits,
		prType:            generate,
		push:              r.push,
		repo:              r.repo,
		sourceRepo:        r.sourceRepo,
		state:             r.state,
		workRoot:          r.workRoot,
		failedGenerations: failedGenerations,
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
	safeLibraryDirectory := getSafeDirectoryName(libraryID)
	if r.needsConfigure() {
		slog.Info("library not configured, start initial configuration", "library", r.library)
		configureOutputDir := filepath.Join(outputDir, safeLibraryDirectory, "configure")
		if err := os.MkdirAll(configureOutputDir, 0755); err != nil {
			return "", err
		}
		configuredLibraryID, err := r.runConfigureCommand(ctx, configureOutputDir)
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
	libraryOutputDir := filepath.Join(outputDir, safeLibraryDirectory)
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
		ApiRoot:   apiRoot,
		HostMount: r.hostMount,
		LibraryID: libraryID,
		Output:    outputDir,
		RepoDir:   r.repo.GetDir(),
		State:     r.state,
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
		HostMount: r.hostMount,
		LibraryID: libraryID,
		RepoDir:   r.repo.GetDir(),
		State:     r.state,
	}
	slog.Info("Performing build for library", "id", libraryID)
	if containerErr := r.containerClient.Build(ctx, buildRequest); containerErr != nil {
		if restoreErr := r.restoreLibrary(libraryID); restoreErr != nil {
			return errors.Join(containerErr, restoreErr)
		}

		return errors.Join(containerErr)
	}

	// Read the library state from the response.
	if _, responseErr := readLibraryState(
		filepath.Join(buildRequest.RepoDir, config.LibrarianDir, config.BuildResponse)); responseErr != nil {
		if restoreErr := r.restoreLibrary(libraryID); restoreErr != nil {
			return errors.Join(responseErr, restoreErr)
		}

		return responseErr
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
func (r *generateRunner) runConfigureCommand(ctx context.Context, outputDir string) (string, error) {

	apiRoot, err := filepath.Abs(r.sourceRepo.GetDir())
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
		apiRoot); err != nil {
		return "", err
	}

	var globalFiles []string
	if r.librarianConfig != nil {
		globalFiles = r.librarianConfig.GetGlobalFiles()
	}

	configureRequest := &docker.ConfigureRequest{
		ApiRoot:             apiRoot,
		HostMount:           r.hostMount,
		LibraryID:           r.library,
		Output:              outputDir,
		RepoDir:             r.repo.GetDir(),
		GlobalFiles:         globalFiles,
		ExistingSourceRoots: r.getExistingSrc(r.library),
		State:               r.state,
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

	if err := copyLibraryFiles(r.state, r.repo.GetDir(), libraryState.ID, outputDir); err != nil {
		return "", err
	}

	if err := copyGlobalAllowlist(r.librarianConfig, r.repo.GetDir(), outputDir, false); err != nil {
		return "", err
	}

	return libraryState.ID, nil
}

func (r *generateRunner) restoreLibrary(libraryID string) error {
	// At this point, we should have a library in the state.
	library := findLibraryByID(r.state, libraryID)
	if err := r.repo.Restore(library.SourceRoots); err != nil {
		return err
	}

	return r.repo.CleanUntracked(library.SourceRoots)
}

// getExistingSrc returns source roots as-is of a given library ID, if the source roots exist in the language repo.
func (r *generateRunner) getExistingSrc(libraryID string) []string {
	library := findLibraryByID(r.state, libraryID)
	var existingSrc []string
	for _, src := range library.SourceRoots {
		relPath := filepath.Join(r.repo.GetDir(), src)
		if _, err := os.Stat(relPath); err == nil {
			existingSrc = append(existingSrc, src)
		}
	}

	return existingSrc
}

func setAllAPIStatus(state *config.LibrarianState, status string) {
	for _, library := range state.Libraries {
		for _, api := range library.APIs {
			api.Status = status
		}
	}
}

// getSafeDirectoryName returns a directory name which doesn't contain slashes
// based on a library ID. This avoids cases where a library ID contains
// slashes but we want generateSingleLibrary to create a directory which
// is not a subdirectory of some other directory. For example, if there
// are library IDs of "pubsub" and "pubsub/v2" we don't want to create
// "output/pubsub/v2" and then "output/pubsub" later. This function does
// not protect against malicious library IDs, e.g. ".", ".." or deliberate
// collisions (e.g. "pubsub/v2" and "pubsub-slash-v2").
//
// The exact implementation may change over time - nothing should rely on this.
// The current implementation simply replaces any slashes with "-slash-".
func getSafeDirectoryName(libraryID string) string {
	return strings.ReplaceAll(libraryID, "/", "-slash-")
}
