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
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/gitrepo"
)

var cmdGenerate = &cli.Command{
	Short:     "generate generates client library code for a single API",
	UsageLine: "librarian generate -source=<api-root> -api=<api-path> [flags]",
	Long: `Specify the API repository root and the path within it for the API to generate.
Optional flags can be specified to use a non-default language repository, and to indicate whether or not
to build the generated library.

The generate command handles both onboarding new libraries and regenerating existing ones.
The behavior is determined by the provided flags.

**Onboarding a new library:**
To configure and generate a new library, specify both the "-api" and "-library" flags. This process involves:
1. Running the "configure" command in the language container to set up the repository.
2. Adding the new library's configuration to the ".librarian/state.yaml" file.
3. Proceeding with the generation steps below.

**Regenerating existing libraries:**
If only "-api" or "-library" is specified, the command regenerates that single, existing library.
If neither flag is provided, it regenerates all libraries listed in ".librarian/state.yaml".

The generation process for an existing library involves delegating to the language container's 
'generate' command. After generation, the tool cleans the destination directory and copies the 
new files into place, according to the configuration in '.librarian/state.yaml'. 
If the '--build' flag is specified, the 'build' command is also executed.

**Output:**
After generation, if a push configuration is provided (e.g., via the "-push-config" flag), the changes
are committed to a new branch, and a pull request is created. Otherwise, the changes are left in the
local working tree for inspection.`,
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
	addFlagWorkRoot(fs, cfg)
	addFlagPush(fs, cfg)
}

type generateRunner struct {
	cfg             *config.Config
	repo            gitrepo.Repository
	sourceRepo      gitrepo.Repository
	state           *config.LibrarianState
	ghClient        GitHubClient
	containerClient ContainerClient
	workRoot        string
	image           string
}

func newGenerateRunner(cfg *config.Config) (*generateRunner, error) {
	runner, err := newCommandRunner(cfg)
	if err != nil {
		return nil, err
	}
	return &generateRunner{
		cfg:             runner.cfg,
		workRoot:        runner.workRoot,
		repo:            runner.repo,
		sourceRepo:      runner.sourceRepo,
		state:           runner.state,
		image:           runner.image,
		ghClient:        runner.ghClient,
		containerClient: runner.containerClient,
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
		return err
	}
	slog.Info("Code will be generated", "dir", outputDir)

	prBody := ""
	if r.cfg.API != "" || r.cfg.Library != "" {
		libraryID := r.cfg.Library
		if libraryID == "" {
			libraryID = findLibraryIDByAPIPath(r.state, r.cfg.API)
		}
		if err := r.generateSingleLibrary(ctx, libraryID, outputDir); err != nil {
			return err
		}
		prBody += fmt.Sprintf("feat: generated %s\n", libraryID)
	} else {
		for _, library := range r.state.Libraries {
			if err := r.generateSingleLibrary(ctx, library.ID, outputDir); err != nil {
				// TODO(https://github.com/googleapis/librarian/issues/983): record failure and report in PR body when applicable
				slog.Error("failed to generate library", "id", library.ID, "err", err)
				prBody += fmt.Sprintf("%s failed to generate\n", library.ID)
			}
		}
	}

	if err := saveLibrarianState(r.repo.GetDir(), r.state); err != nil {
		return err
	}
	if err := commitAndPush(ctx, r, prBody); err != nil {
		return err
	}
	return nil
}

// generateSingleLibrary manages the generation of a single client library.
//
// It can either configure a new library if the API and library both are specified
// and library not configured in state.yaml yet, or regenerate an existing library
// if a libraryID is provided.
// After ensuring the library is configured, it runs the generation and build commands.
func (r *generateRunner) generateSingleLibrary(ctx context.Context, libraryID, outputDir string) error {
	if r.needsConfigure() {
		slog.Info("library not configured, start initial configuration", "library", r.cfg.Library)
		configuredLibraryID, err := r.runConfigureCommand(ctx)
		if err != nil {
			return err
		}
		libraryID = configuredLibraryID
	}

	generatedLibraryID, err := r.runGenerateCommand(ctx, libraryID, outputDir)
	if err != nil {
		return err
	}

	if err := r.runBuildCommand(ctx, generatedLibraryID); err != nil {
		return err
	}
	if err := r.updateLastGeneratedCommitState(generatedLibraryID); err != nil {
		return err
	}
	return nil
}

func (r *generateRunner) needsConfigure() bool {
	return r.cfg.API != "" && r.cfg.Library != "" && findLibraryByID(r.state, r.cfg.Library) == nil
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
	if findLibraryByID(r.state, libraryID) == nil {
		return "", fmt.Errorf("library %q not configured yet, generation stopped", libraryID)
	}
	apiRoot, err := filepath.Abs(r.sourceRepo.GetDir())
	if err != nil {
		return "", err
	}

	generateRequest := &docker.GenerateRequest{
		Cfg:       r.cfg,
		State:     r.state,
		ApiRoot:   apiRoot,
		LibraryID: libraryID,
		Output:    outputDir,
		RepoDir:   r.repo.GetDir(),
	}
	slog.Info("Performing generation for library", "id", libraryID)
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

	return libraryID, nil
}

// runBuildCommand orchestrates the building of an API library using a containerized
// environment.
//
// The `outputDir` parameter specifies the target directory where the built artifacts
// should be placed.
func (r *generateRunner) runBuildCommand(ctx context.Context, libraryID string) error {
	if !r.cfg.Build {
		slog.Info("Build flag not specified, skipping")
		return nil
	}
	if libraryID == "" {
		slog.Warn("Cannot perform build, missing library ID")
		return nil
	}

	buildRequest := &docker.BuildRequest{
		Cfg:       r.cfg,
		State:     r.state,
		LibraryID: libraryID,
		RepoDir:   r.repo.GetDir(),
	}
	slog.Info("Build requested for library", "id", libraryID)
	if err := r.containerClient.Build(ctx, buildRequest); err != nil {
		return err
	}

	// Read the library state from the response.
	_, err := readLibraryState(
		filepath.Join(buildRequest.RepoDir, config.LibrarianDir, config.BuildResponse),
	)

	return err
}

// clean removes files and directories from a root directory based on remove and preserve patterns.
//
// It first determines the paths to remove by applying the removePatterns and then excluding any paths
// that match the preservePatterns. It then separates the remaining paths into files and directories and
// removes them, ensuring that directories are removed last.
//
// This logic is ported from owlbot logic: https://github.com/googleapis/repo-automation-bots/blob/12dad68640960290910b660e4325630c9ace494b/packages/owl-bot/src/copy-code.ts#L1027
func clean(rootDir string, removePatterns, preservePatterns []string) error {
	slog.Info("cleaning directory", "path", rootDir)
	finalPathsToRemove, err := deriveFinalPathsToRemove(rootDir, removePatterns, preservePatterns)
	if err != nil {
		return err
	}

	filesToRemove, dirsToRemove, err := separateFilesAndDirs(rootDir, finalPathsToRemove)
	if err != nil {
		return err
	}

	// Remove files first, then directories.
	for _, file := range filesToRemove {
		slog.Info("removing file", "path", file)
		if err := os.Remove(filepath.Join(rootDir, file)); err != nil {
			return err
		}
	}

	sortDirsByDepth(dirsToRemove)

	for _, dir := range dirsToRemove {
		slog.Info("removing directory", "path", dir)
		if err := os.Remove(filepath.Join(rootDir, dir)); err != nil {
			// It's possible the directory is not empty due to preserved files.
			slog.Warn("failed to remove directory, it may not be empty", "dir", dir, "err", err)
		}
	}

	return nil
}

// sortDirsByDepth sorts directories by depth (descending) to remove children first.
func sortDirsByDepth(dirs []string) {
	slices.SortFunc(dirs, func(a, b string) int {
		return strings.Count(b, string(filepath.Separator)) - strings.Count(a, string(filepath.Separator))
	})
}

// allPaths walks the directory tree rooted at rootDir and returns a slice of all
// file and directory paths, relative to rootDir.
func allPaths(rootDir string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		paths = append(paths, relPath)
		return nil
	})
	return paths, err
}

// filterPaths returns a new slice containing only the paths from the input slice
// that match at least one of the provided regular expressions.
func filterPaths(paths []string, regexps []*regexp.Regexp) []string {
	var filtered []string
	for _, path := range paths {
		for _, re := range regexps {
			if re.MatchString(path) {
				filtered = append(filtered, path)
				break
			}
		}
	}
	return filtered
}

// deriveFinalPathsToRemove determines the final set of paths to be removed. It
// starts with all paths under rootDir, filters them based on removePatterns,
// and then excludes any paths that match preservePatterns.
func deriveFinalPathsToRemove(rootDir string, removePatterns, preservePatterns []string) ([]string, error) {
	removeRegexps, err := compileRegexps(removePatterns)
	if err != nil {
		return nil, err
	}
	preserveRegexps, err := compileRegexps(preservePatterns)
	if err != nil {
		return nil, err
	}

	allPaths, err := allPaths(rootDir)
	if err != nil {
		return nil, err
	}

	pathsToRemove := filterPaths(allPaths, removeRegexps)
	pathsToPreserve := filterPaths(pathsToRemove, preserveRegexps)

	// delete pathsToPreserve from pathsToRemove.
	pathsToDelete := make(map[string]bool)
	for _, p := range pathsToPreserve {
		pathsToDelete[p] = true
	}
	finalPathsToRemove := slices.DeleteFunc(pathsToRemove, func(path string) bool {
		return pathsToDelete[path]
	})
	return finalPathsToRemove, nil
}

// separateFilesAndDirs takes a list of paths and categorizes them into files
// and directories. It uses os.Lstat to avoid following symlinks, treating them
// as files. Paths that do not exist are silently ignored.
func separateFilesAndDirs(rootDir string, paths []string) ([]string, []string, error) {
	var files, dirs []string
	for _, path := range paths {
		info, err := os.Lstat(filepath.Join(rootDir, path))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				// The file or directory may have already been removed.
				continue
			}
			// For any other error (permissions, I/O, etc.)
			return nil, nil, fmt.Errorf("failed to stat path %q: %w", path, err)

		}
		if info.IsDir() {
			dirs = append(dirs, path)
		} else {
			files = append(files, path)
		}
	}
	return files, dirs, nil
}

// compileRegexps takes a slice of string patterns and compiles each one into a
// regular expression. It returns a slice of compiled regexps or an error if any
// pattern is invalid.
func compileRegexps(patterns []string) ([]*regexp.Regexp, error) {
	var regexps []*regexp.Regexp
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %q: %w", pattern, err)
		}
		regexps = append(regexps, re)
	}
	return regexps, nil
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

	apiRoot, err := filepath.Abs(r.cfg.APISource)
	if err != nil {
		return "", err
	}

	setAllAPIStatus(r.state, config.StatusExisting)
	// Record to state, not write to state.yaml
	r.state.Libraries = append(r.state.Libraries, &config.LibraryState{
		ID:   r.cfg.Library,
		APIs: []*config.API{{Path: r.cfg.API, Status: config.StatusNew}},
	})

	if err := populateServiceConfigIfEmpty(
		r.state,
		r.cfg.APISource); err != nil {
		return "", err
	}

	configureRequest := &docker.ConfigureRequest{
		Cfg:       r.cfg,
		State:     r.state,
		ApiRoot:   apiRoot,
		LibraryID: r.cfg.Library,
		RepoDir:   r.repo.GetDir(),
	}
	slog.Info("Performing configuration for library", "id", r.cfg.Library)
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
		slog.Info("library doesn't receive a version, apply the default version", "id", r.cfg.Library)
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
