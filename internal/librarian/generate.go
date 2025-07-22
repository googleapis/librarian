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
	"time"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

var cmdGenerate = &cli.Command{
	Short:     "generate generates client library code for a single API",
	UsageLine: "librarian generate -source=<api-root> -api=<api-path> [flags]",
	Long: `Specify the API repository root and the path within it for the API to generate.
Optional flags can be specified to use a non-default language repository, and to indicate whether or not
to build the generated library.

First, the generation mode is determined by examining the language repository (remotely if
a local clone has not been specified). The Librarian state for the repository is examined to see if the
specified API path is already configured for a library in the repository. If it is, the refined generation
mode is used. Otherwise the raw generation mode is used. These are described separately below.

*Refined generation* is intended to give an accurate result for how an existing library would change when
generated with the new API specification. Generation for this library might include pregeneration or postgeneration
fixes, and the library may include handwritten code, unit tests and integration tests.

The process for refined generation requires the language repo to be cloned (if a local clone hasn't been
specified). Generation then proceeds by executing the following language container commands:
- "generate-library" to generate the source code for the library into an empty directory
- "clean" to clean any previously-generated source code from the language repository
- "build-library" (after copying the newly-generated code into place in the repository)

(The "clean" and "build-library" commands are skipped if the -build flag is not specified.)

The result of the generation is not committed anywhere, but the language repository will be left with any
working tree changes available to be checked. (Changes are not reverted.)


*Raw generation* is intended to give an early indication of whether an API can successfully be generated
as a library, and whether that library can then be built, without any additional information from the language
repo. The language repo is not cloned, but instead the following language container commands are executed:
- "generate-raw" to generate the source code for the library into an empty directory
- "build-raw" (if the -build flag is specified)

There is no "clean" operation or copying of the generated code in raw generation mode, because there is no
other source code to be preserved/cleaned. Instead, the "build-raw" command is provided with the same
output directory that was specified for the "generate-raw" command.
`,
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
	addFlagBuild(fs, cfg)
	addFlagHostMount(fs, cfg)
	addFlagPushConfig(fs, cfg)
	addFlagImage(fs, cfg)
	addFlagProject(fs, cfg)
	addFlagRepo(fs, cfg)
	addFlagSource(fs, cfg)
	addFlagWorkRoot(fs, cfg)
}

type generateRunner struct {
	cfg             *config.Config
	repo            *gitrepo.Repository
	state           *config.LibrarianState
	config          *config.PipelineConfig
	ghClient        GitHubClient
	containerClient ContainerClient
	workRoot        string
	image           string
}

func newGenerateRunner(cfg *config.Config) (*generateRunner, error) {
	if err := validateRequiredFlag("repo", cfg.Repo); err != nil {
		return nil, err
	}
	if err := validatePushConfigAndGithubTokenCoexist(cfg.PushConfig, cfg.GitHubToken); err != nil {
		return nil, err
	}
	workRoot, err := createWorkRoot(time.Now(), cfg.WorkRoot)
	if err != nil {
		return nil, err
	}
	repo, err := cloneOrOpenLanguageRepo(workRoot, cfg.Repo, cfg.CI)
	if err != nil {
		return nil, err
	}
	state, pipelineConfig, err := loadRepoStateAndConfig(repo, cfg.Source)
	if err != nil {
		return nil, err
	}
	image := deriveImage(cfg.Image, state)

	var ghClient GitHubClient
	if isUrl(cfg.Repo) {
		// repo is a URL
		languageRepo, err := github.ParseURL(cfg.Repo)
		if err != nil {
			return nil, fmt.Errorf("failed to parse repo url: %w", err)
		}
		ghClient, err = github.NewClient(cfg.GitHubToken, languageRepo)
		if err != nil {
			return nil, fmt.Errorf("failed to create GitHub client: %w", err)
		}
	}
	container, err := docker.New(workRoot, image, cfg.Project, cfg.UserUID, cfg.UserGID, pipelineConfig)
	if err != nil {
		return nil, err
	}
	return &generateRunner{
		cfg:             cfg,
		workRoot:        workRoot,
		repo:            repo,
		state:           state,
		config:          pipelineConfig,
		image:           image,
		ghClient:        ghClient,
		containerClient: container,
	}, nil
}

func (r *generateRunner) run(ctx context.Context) error {
	outputDir := filepath.Join(r.workRoot, "output")
	if err := os.Mkdir(outputDir, 0755); err != nil {
		return err
	}
	slog.Info("Code will be generated", "dir", outputDir)

	configured, err := r.detectIfLibraryConfigured(ctx)
	if err != nil {
		return err
	}

	if !configured {
		if err := r.runConfigureCommand(ctx); err != nil {
			return err
		}
	}

	libraryID, err := r.runGenerateCommand(ctx, outputDir)
	if err != nil {
		return err
	}

	if err := r.runBuildCommand(ctx, libraryID); err != nil {
		return err
	}

	// Commit and Push to GitHub.
	if err := commitAndPush(ctx, r.repo, r.ghClient, r.cfg.PushConfig); err != nil {
		return err
	}
	return nil
}

// runGenerateCommand attempts to perform generation for an API. It then cleans the
// destination directory and copies the newly generated files into it.
//
// If successful, it returns the ID of the generated library; otherwise, it
// returns an empty string and an error.
func (r *generateRunner) runGenerateCommand(ctx context.Context, outputDir string) (string, error) {
	apiRoot, err := filepath.Abs(r.cfg.Source)
	if err != nil {
		return "", err
	}

	// If we've got a language repo, it's because we've already found a library for the
	// specified API, configured in the repo.
	if r.repo != nil {
		libraryID := findLibraryIDByAPIPath(r.state, r.cfg.API)
		if libraryID == "" {
			return "", errors.New("bug in Librarian: Library not found during generation, despite being found in earlier steps")
		}

		generateRequest := &docker.GenerateRequest{
			Cfg:       r.cfg,
			State:     r.state,
			ApiRoot:   apiRoot,
			LibraryID: libraryID,
			Output:    outputDir,
			RepoDir:   r.repo.Dir,
		}
		slog.Info("Performing refined generation for library", "id", libraryID)
		if err := r.containerClient.Generate(ctx, generateRequest); err != nil {
			return "", err
		}

		if err := r.cleanAndCopyLibrary(libraryID, outputDir); err != nil {
			return "", err
		}

		return libraryID, nil
	}

	slog.Info("No matching library found (or no repo specified)", "path", r.cfg.API)

	return "", fmt.Errorf("library not found")
}

func (r *generateRunner) cleanAndCopyLibrary(libraryID, outputDir string) error {
	library := findLibraryByID(r.state, libraryID)
	if library == nil {
		return fmt.Errorf("library %q not found during clean and copy, despite being found in earlier steps", libraryID)
	}
	if err := clean(r.repo.Dir, library.RemoveRegex, library.PreserveRegex); err != nil {
		return err
	}
	// os.CopyFS in Go1.24 returns error when copying from a symbolic link
	// https://github.com/golang/go/blob/9d828e80fa1f3cc52de60428cae446b35b576de8/src/os/dir.go#L143-L144
	if err := os.CopyFS(r.repo.Dir, os.DirFS(outputDir)); err != nil {
		return err
	}
	slog.Info("Library updated", "id", libraryID)
	return nil
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
		RepoDir:   r.repo.Dir,
	}
	slog.Info("Build requested in the context of refined generation; cleaning and copying code to the local language repo before building.")
	return r.containerClient.Build(ctx, buildRequest)
}

// clean removes files and directories from a root directory based on remove and preserve patterns.
//
// It first determines the paths to remove by applying the removePatterns and then excluding any paths
// that match the preservePatterns. It then separates the remaining paths into files and directories and
// removes them, ensuring that directories are removed last.
//
// This logic is ported from owlbot logic: https://github.com/googleapis/repo-automation-bots/blob/12dad68640960290910b660e4325630c9ace494b/packages/owl-bot/src/copy-code.ts#L1027
func clean(rootDir string, removePatterns, preservePatterns []string) error {
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
		slog.Info("Removing file", "path", file)
		if err := os.Remove(filepath.Join(rootDir, file)); err != nil {
			return err
		}
	}

	sortDirsByDepth(dirsToRemove)

	for _, dir := range dirsToRemove {
		slog.Info("Removing directory", "path", dir)
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
// This step prepares a language repository for a new library by adding
// necessary metadata or build configurations. It finds the library's ID
// from the runner's state, gathers all paths and settings, and then delegates
// the execution to the container client.
func (r *generateRunner) runConfigureCommand(ctx context.Context) error {
	if r.cfg.API == "" {
		return errors.New("API flag not specified for new library configuration")
	}
	apiRoot, err := filepath.Abs(r.cfg.Source)
	if err != nil {
		return err
	}

	// Configuration requires a language repository to modify. If one isn't specified or
	// found, we cannot proceed.
	if r.repo == nil {
		slog.Error("No language repository specified; cannot run configure.", "api", r.cfg.API)
		return errors.New("a language repository must be specified to run configure")
	}

	libraryID := findLibraryIDByAPIPath(r.state, r.cfg.API)
	if libraryID == "" {
		return errors.New("bug in Librarian: Library not found during configuration, despite being found in earlier steps")
	}

	configureRequest := &docker.ConfigureRequest{
		Cfg:       r.cfg,
		State:     r.state,
		ApiRoot:   apiRoot,
		LibraryID: libraryID,
		RepoDir:   r.repo.Dir,
	}
	slog.Info("Performing configuration for library", "id", libraryID)
	return r.containerClient.Configure(ctx, configureRequest)
}

// detectIfLibraryConfigured returns whether a library has been configured for
// the requested API (as specified in apiPath). This is done by checking the local
// pipeline state if repoRoot has been specified, or the remote pipeline state (just
// by fetching the single file) if flatRepoUrl has been specified. If neither the repo
// root not the repo url has been specified, we always perform raw generation.
func (r *generateRunner) detectIfLibraryConfigured(ctx context.Context) (bool, error) {
	apiPath, repo, source := r.cfg.API, r.cfg.Repo, r.cfg.Source
	if repo == "" {
		slog.Warn("repo is not specified, cannot check if library exists")
		return false, nil
	}

	// Attempt to load the pipeline state either locally or from the repo URL
	var (
		pipelineState *config.LibrarianState
		err           error
	)
	if isUrl(repo) {
		pipelineState, err = fetchRemoteLibrarianState(ctx, r.ghClient, "HEAD", source)
		if err != nil {
			return false, err
		}
	} else {
		// repo is a directory
		pipelineState, err = loadLibrarianStateFile(filepath.Join(repo, config.LibrarianDir, pipelineStateFile), source)
		if err != nil {
			return false, err
		}
	}
	// If the library doesn't exist, we don't use the repo at all.
	libraryID := findLibraryIDByAPIPath(pipelineState, apiPath)
	if libraryID == "" {
		slog.Info("API path not configured in repo", "path", apiPath)
		return false, nil
	}

	slog.Info("API configured", "path", apiPath, "library", libraryID)
	return true, nil
}
