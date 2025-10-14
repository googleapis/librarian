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
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/docker"
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

// generationStatus represents the result of a single library generation.
type generationStatus struct {
	// oldCommit is the SHA of the previously generated version of the library.
	oldCommit string
	prType    pullRequestType
}

func newGenerateRunner(cfg *config.Config) (*generateRunner, error) {
	runner, err := newCommandRunner(cfg)
	if err != nil {
		return nil, err
	}
	return &generateRunner{
		api:             cfg.API,
		branch:          cfg.Branch,
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
		librarianConfig: runner.librarianConfig,
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
	idToCommits := make(map[string]string)
	var failedLibraries []string
	failedGenerations := 0
	prType := pullRequestGenerate
	if r.api != "" || r.library != "" {
		libraryID := r.library
		if libraryID == "" {
			libraryID = findLibraryIDByAPIPath(r.state, r.api)
		}
		status, err := r.generateSingleLibrary(ctx, libraryID, outputDir)
		if err != nil {
			return err
		}
		idToCommits[libraryID] = status.oldCommit
		prType = status.prType
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
			status, err := r.generateSingleLibrary(ctx, library.ID, outputDir)
			if err != nil {
				slog.Error("failed to generate library", "id", library.ID, "err", err)
				failedLibraries = append(failedLibraries, library.ID)
				failedGenerations++
			} else {
				// Only add the mapping if library generation is successful so that
				// failed library will not appear in generation PR body.
				idToCommits[library.ID] = status.oldCommit
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

	var prBodyBuilder func() (string, error)
	switch prType {
	case pullRequestGenerate:
		prBodyBuilder = func() (string, error) {
			req := &generationPRRequest{
				sourceRepo:      r.sourceRepo,
				languageRepo:    r.repo,
				state:           r.state,
				idToCommits:     idToCommits,
				failedLibraries: failedLibraries,
			}
			return formatGenerationPRBody(req)
		}
	case pullRequestOnboard:
		prBodyBuilder = func() (string, error) {
			req := &onboardPRRequest{
				sourceRepo: r.sourceRepo,
				state:      r.state,
				api:        r.api,
				library:    r.library,
			}
			return formatOnboardPRBody(req)
		}
	default:
		return fmt.Errorf("unexpected prType %s", prType)
	}

	commitInfo := &commitInfo{
		branch:            r.branch,
		commit:            r.commit,
		commitMessage:     "feat: generate libraries",
		ghClient:          r.ghClient,
		prType:            prType,
		push:              r.push,
		languageRepo:      r.repo,
		sourceRepo:        r.sourceRepo,
		state:             r.state,
		workRoot:          r.workRoot,
		api:               r.api,
		library:           r.library,
		failedGenerations: failedGenerations,
		prBodyBuilder:     prBodyBuilder,
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
// 4. Update the last generated commit or initial piper id if the library needs configure.
func (r *generateRunner) generateSingleLibrary(ctx context.Context, libraryID, outputDir string) (*generationStatus, error) {
	safeLibraryDirectory := getSafeDirectoryName(libraryID)
	prType := pullRequestGenerate
	if r.needsConfigure() {
		slog.Info("library not configured, start initial configuration", "library", r.library)
		configureOutputDir := filepath.Join(outputDir, safeLibraryDirectory, "configure")
		if err := os.MkdirAll(configureOutputDir, 0755); err != nil {
			return nil, err
		}
		configuredLibraryID, err := r.runConfigureCommand(ctx, configureOutputDir)
		if err != nil {
			return nil, err
		}

		prType = pullRequestGenerate
		libraryID = configuredLibraryID
	}

	// At this point, we should have a library in the state.
	libraryState := findLibraryByID(r.state, libraryID)
	if libraryState == nil {
		return nil, fmt.Errorf("library %q not configured yet, generation stopped", libraryID)
	}
	lastGenCommit := libraryState.LastGeneratedCommit

	if len(libraryState.APIs) == 0 {
		slog.Info("library has no APIs; skipping generation", "library", libraryID)
		return &generationStatus{
			oldCommit: "",
			prType:    prType,
		}, nil
	}

	if err := generateSingleLibrary(ctx, r.containerClient, r.state, libraryState, r.repo, r.sourceRepo, outputDir); err != nil {
		return nil, err
	}

	if r.build {
		if err := buildSingleLibrary(ctx, r.containerClient, r.state, libraryState, r.repo); err != nil {
			return nil, err
		}
	}

	if err := r.updateLastGeneratedCommitState(libraryID); err != nil {
		return nil, err
	}

	return &generationStatus{
		oldCommit: lastGenCommit,
		prType:    prType,
	}, nil
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

type generationPRRequest struct {
	sourceRepo      gitrepo.Repository
	languageRepo    gitrepo.Repository
	state           *config.LibrarianState
	idToCommits     map[string]string
	failedLibraries []string
}

type onboardPRRequest struct {
	sourceRepo gitrepo.Repository
	state      *config.LibrarianState
	api        string
	library    string
}

type generationPRBody struct {
	StartSHA         string
	EndSHA           string
	LibrarianVersion string
	ImageVersion     string
	Commits          []*gitrepo.ConventionalCommit
	FailedLibraries  []string
}

type onboardingPRBody struct {
	ImageVersion     string
	LibrarianVersion string
	LibraryID        string
	PiperID          string
}

// formatGenerationPRBody creates the body of a generation pull request.
// Only consider libraries whose ID appears in idToCommits.
func formatGenerationPRBody(request *generationPRRequest) (string, error) {
	var allCommits []*gitrepo.ConventionalCommit
	languageRepoChanges, err := languageRepoChangedFiles(request.languageRepo)
	if err != nil {
		return "", fmt.Errorf("failed to fetch changes in language repo: %w", err)
	}
	for _, library := range request.state.Libraries {
		lastGenCommit, ok := request.idToCommits[library.ID]
		if !ok {
			continue
		}
		// If nothing has changed that would be significant in a release for this library,
		// we don't look at the API changes either.
		if !shouldIncludeForRelease(languageRepoChanges, library.SourceRoots, library.ReleaseExcludePaths) {
			continue
		}

		commits, err := getConventionalCommitsSinceLastGeneration(request.sourceRepo, library, lastGenCommit)
		if err != nil {
			return "", fmt.Errorf("failed to fetch conventional commits for library, %s: %w", library.ID, err)
		}
		allCommits = append(allCommits, commits...)
	}

	if len(allCommits) == 0 {
		return "No commit is found since last generation", nil
	}

	startCommit, err := findLatestGenerationCommit(request.sourceRepo, request.state, request.idToCommits)
	if err != nil {
		return "", fmt.Errorf("failed to find the start commit: %w", err)
	}
	// Even though startCommit might be nil, it shouldn't happen in production
	// because this function will return early if no conventional commit is found
	// since last generation.
	startSHA := startCommit.Hash.String()
	groupedCommits := groupByIDAndSubject(allCommits)
	// Sort the slice by commit time in reverse order,
	// so that the latest commit appears first.
	sort.Slice(groupedCommits, func(i, j int) bool {
		return groupedCommits[i].When.After(groupedCommits[j].When)
	})
	endSHA := groupedCommits[0].CommitHash
	librarianVersion := cli.Version()
	data := &generationPRBody{
		StartSHA:         startSHA,
		EndSHA:           endSHA,
		LibrarianVersion: librarianVersion,
		ImageVersion:     request.state.Image,
		Commits:          groupedCommits,
		FailedLibraries:  request.failedLibraries,
	}
	var out bytes.Buffer
	if err := genBodyTemplate.Execute(&out, data); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return strings.TrimSpace(out.String()), nil
}

// languageRepoChangedFiles returns the paths of files changed in the repo as part
// of the current librarian run - either in the head commit if the repo is clean,
// or the outstanding changes otherwise.
func languageRepoChangedFiles(languageRepo gitrepo.Repository) ([]string, error) {
	clean, err := languageRepo.IsClean()
	if err != nil {
		return nil, err
	}
	if clean {
		headHash, err := languageRepo.HeadHash()
		if err != nil {
			return nil, err
		}
		return languageRepo.ChangedFilesInCommit(headHash)
	}
	// The commit or push flag is not set, get all locally changed files.
	return languageRepo.ChangedFiles()
}

// formatOnboardPRBody creates the body of an onboarding pull request.
func formatOnboardPRBody(request *onboardPRRequest) (string, error) {
	piperID, err := getPiperID(request.state, request.sourceRepo, request.api, request.library)
	if err != nil {
		return "", err
	}

	data := &onboardingPRBody{
		LibrarianVersion: cli.Version(),
		ImageVersion:     request.state.Image,
		LibraryID:        request.library,
		PiperID:          piperID,
	}

	var out bytes.Buffer
	if err := onboardingBodyTemplate.Execute(&out, data); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return strings.TrimSpace(out.String()), nil
}

// getPiperID extracts the Piper ID from the commit message that onboarded the API.
func getPiperID(state *config.LibrarianState, sourceRepo gitrepo.Repository, apiPath, library string) (string, error) {
	libraryState := findLibraryByID(state, library)
	serviceYaml := ""
	for _, api := range libraryState.APIs {
		if api.Path == apiPath {
			serviceYaml = api.ServiceConfig
			break
		}
	}

	initialCommit, err := sourceRepo.GetLatestCommit(filepath.Join(apiPath, serviceYaml))
	if err != nil {
		return "", err
	}

	id, err := findPiperIDFrom(initialCommit, library)
	if err != nil {
		return "", err
	}

	slog.Info("found piper id in the commit message", "piperID", id)
	return id, nil
}

func findPiperIDFrom(commit *gitrepo.Commit, libraryID string) (string, error) {
	commits, err := gitrepo.ParseCommits(commit, libraryID)
	if err != nil {
		return "", err
	}

	if len(commits) == 0 || commits[0].Footers == nil {
		return "", errPiperNotFound
	}

	id, ok := commits[0].Footers["PiperOrigin-RevId"]
	if !ok {
		return "", errPiperNotFound
	}

	return id, nil
}
