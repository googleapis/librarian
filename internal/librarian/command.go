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

package librarian

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/docker"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

const (
	generate = "generate"
	release  = "release"
)

var globalPreservePatterns = []string{
	fmt.Sprintf(`^%s(/.*)?$`, regexp.QuoteMeta(config.GeneratorInputDir)), // Preserve the generator-input directory and its contents.
}

// GitHubClientFactory type for creating a GitHubClient.
type GitHubClientFactory func(token string, repo *github.Repository) (GitHubClient, error)

// ContainerClientFactory type for creating a ContainerClient.
type ContainerClientFactory func(workRoot, image, userUID, userGID string) (ContainerClient, error)

type commitInfo struct {
	cfg               *config.Config
	state             *config.LibrarianState
	repo              gitrepo.Repository
	ghClient          GitHubClient
	idToCommits       map[string]string
	failedLibraries   []string
	commitMessage     string
	prType            string
	pullRequestLabels []string
}

type commandRunner struct {
	cfg             *config.Config
	repo            gitrepo.Repository
	sourceRepo      gitrepo.Repository
	state           *config.LibrarianState
	librarianConfig *config.LibrarianConfig
	ghClient        GitHubClient
	containerClient ContainerClient
	workRoot        string
	image           string
}

const defaultAPISourceBranch = "master"

func newCommandRunner(cfg *config.Config, ghClientFactory GitHubClientFactory, containerClientFactory ContainerClientFactory) (*commandRunner, error) {
	// If no GitHub client factory is provided, use the default one.
	if ghClientFactory == nil {
		ghClientFactory = func(token string, repo *github.Repository) (GitHubClient, error) {
			return github.NewClient(token, repo)
		}
	}
	// If no container client factory is provided, use the default one.
	if containerClientFactory == nil {
		containerClientFactory = func(workRoot, image, userUID, userGID string) (ContainerClient, error) {
			return docker.New(workRoot, image, userUID, userGID)
		}
	}

	if cfg.APISource == "" {
		cfg.APISource = "https://github.com/googleapis/googleapis"
	}

	languageRepo, err := cloneOrOpenRepo(cfg.WorkRoot, cfg.Repo, cfg.Branch, cfg.CI, cfg.GitHubToken)
	if err != nil {
		return nil, err
	}

	var sourceRepo gitrepo.Repository
	var sourceRepoDir string
	if cfg.CommandName == generateCmdName {
		sourceRepo, err = cloneOrOpenRepo(cfg.WorkRoot, cfg.APISource, defaultAPISourceBranch, cfg.CI, cfg.GitHubToken)
		if err != nil {
			return nil, err
		}
		sourceRepoDir = sourceRepo.GetDir()
	}
	state, err := loadRepoState(languageRepo, sourceRepoDir)
	if err != nil {
		return nil, err
	}

	librarianConfig, err := loadLibrarianConfig(languageRepo)
	if err != nil {
		return nil, err
	}

	image := deriveImage(cfg.Image, state)

	var gitRepo *github.Repository
	if isURL(cfg.Repo) {
		gitRepo, err = github.ParseRemote(cfg.Repo)
		if err != nil {
			return nil, fmt.Errorf("failed to parse repo url: %w", err)
		}
	} else {
		gitRepo, err = github.FetchGitHubRepoFromRemote(languageRepo)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitHub repo from remote: %w", err)
		}
	}
	ghClient, err := ghClientFactory(cfg.GitHubToken, gitRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	container, err := containerClientFactory(cfg.WorkRoot, image, cfg.UserUID, cfg.UserGID)
	if err != nil {
		return nil, err
	}
	return &commandRunner{
		cfg:             cfg,
		workRoot:        cfg.WorkRoot,
		repo:            languageRepo,
		sourceRepo:      sourceRepo,
		state:           state,
		librarianConfig: librarianConfig,
		image:           image,
		ghClient:        ghClient,
		containerClient: container,
	}, nil
}

func cloneOrOpenRepo(workRoot, repo, branch, ci string, gitPassword string) (*gitrepo.LocalRepository, error) {
	if repo == "" {
		return nil, errors.New("repo must be specified")
	}

	if isURL(repo) {
		// repo is a URL
		// Take the last part of the URL as the directory name. It feels very
		// unlikely that will clash with anything else (e.g. "output")
		repoName := path.Base(strings.TrimSuffix(repo, "/"))
		repoPath := filepath.Join(workRoot, repoName)
		return gitrepo.NewRepository(&gitrepo.RepositoryOptions{
			Dir:          repoPath,
			MaybeClone:   true,
			RemoteURL:    repo,
			RemoteBranch: branch,
			CI:           ci,
			GitPassword:  gitPassword,
		})
	}
	// repo is a directory
	absRepoRoot, err := filepath.Abs(repo)
	if err != nil {
		return nil, err
	}
	githubRepo, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{
		Dir:         absRepoRoot,
		CI:          ci,
		GitPassword: gitPassword,
	})
	if err != nil {
		return nil, err
	}
	cleanRepo, err := githubRepo.IsClean()
	if err != nil {
		return nil, err
	}
	if !cleanRepo {
		return nil, fmt.Errorf("%s repo must be clean", repo)
	}
	return githubRepo, nil
}

func deriveImage(imageOverride string, state *config.LibrarianState) string {
	if imageOverride != "" {
		return imageOverride
	}
	if state == nil {
		return ""
	}
	return state.Image
}

func findLibraryIDByAPIPath(state *config.LibrarianState, apiPath string) string {
	if state == nil {
		return ""
	}
	for _, lib := range state.Libraries {
		for _, api := range lib.APIs {
			if api.Path == apiPath {
				return lib.ID
			}
		}
	}
	return ""
}

func findLibraryByID(state *config.LibrarianState, libraryID string) *config.LibraryState {
	if state == nil {
		return nil
	}
	for _, lib := range state.Libraries {
		if lib.ID == libraryID {
			return lib
		}
	}
	return nil
}

func formatTimestamp(t time.Time) string {
	const yyyyMMddHHmmss = "20060102T150405Z" // Expected format by time library
	return t.Format(yyyyMMddHHmmss)
}

// cleanAndCopyLibrary cleans the files of the given library in repoDir and copies
// the new files from outputDir.
func cleanAndCopyLibrary(state *config.LibrarianState, repoDir, libraryID, outputDir string) error {
	library := findLibraryByID(state, libraryID)
	if library == nil {
		return fmt.Errorf("library %q not found during clean and copy, despite being found in earlier steps", libraryID)
	}

	removePatterns := library.RemoveRegex
	if len(removePatterns) == 0 {
		slog.Info("remove_regex not provided, defaulting to source_roots")
		removePatterns = make([]string, len(library.SourceRoots))
		// For each SourceRoot, create a regex pattern to match the source root
		// directory itself, and any file or subdirectory within it.
		for i, root := range library.SourceRoots {
			removePatterns[i] = fmt.Sprintf("^%s(/.*)?$", regexp.QuoteMeta(root))
		}
	}

	preservePatterns := append(library.PreserveRegex,
		globalPreservePatterns...)

	if err := clean(repoDir, removePatterns, preservePatterns); err != nil {
		return fmt.Errorf("failed to clean library, %s: %w", library.ID, err)
	}

	return copyLibraryFiles(state, repoDir, libraryID, outputDir)
}

// copyLibraryFiles copies the files in state.SourceRoots relative to the src folder to the dest
// folder. It overwrites any existing files.
// If there's no files in the library's SourceRoots under the src directory, no copy will happen.
// If a file is being copied to the library's SourceRoots in the dest folder but the folder does
// not exist, the copy fails.
func copyLibraryFiles(state *config.LibrarianState, dest, libraryID, src string) error {
	library := findLibraryByID(state, libraryID)
	if library == nil {
		return fmt.Errorf("library %q not found", libraryID)
	}
	slog.Info("Copying library files", "id", library.ID, "destination", dest, "source", src)
	for _, srcRoot := range library.SourceRoots {
		dstPath := filepath.Join(dest, srcRoot)
		srcPath := filepath.Join(src, srcRoot)
		files, err := getDirectoryFilenames(srcPath)
		if err != nil {
			return err
		}
		for _, file := range files {
			slog.Info("Copying file", "file", file)
			srcFile := filepath.Join(srcPath, file)
			dstFile := filepath.Join(dstPath, file)
			if err := copyFile(dstFile, srcFile); err != nil {
				return fmt.Errorf("failed to copy file %q for library %s: %w", srcFile, library.ID, err)
			}
		}
	}
	return nil
}

func getDirectoryFilenames(dir string) ([]string, error) {
	if _, err := os.Stat(dir); err != nil {
		// Skip dirs that don't exist
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var fileNames []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			relativePath, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			fileNames = append(fileNames, relativePath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return fileNames, nil
}

// commitAndPush creates a commit and push request to GitHub for the generated changes.
// It uses the GitHub client to create a PR with the specified branch, title, and
// description to the repository.
func commitAndPush(ctx context.Context, info *commitInfo) error {
	cfg := info.cfg
	if !cfg.Push && !cfg.Commit {
		slog.Info("Push flag and Commit flag are not specified, skipping committing")
		return nil
	}

	repo := info.repo
	status, err := repo.AddAll()
	if err != nil {
		return err
	}
	if status.IsClean() {
		slog.Info("No changes to commit, skipping commit and push.")
		return nil
	}

	datetimeNow := formatTimestamp(time.Now())
	branch := fmt.Sprintf("librarian-%s", datetimeNow)
	if err := repo.CreateBranchAndCheckout(branch); err != nil {
		return err
	}

	if err := repo.Commit(info.commitMessage); err != nil {
		return err
	}

	if err := repo.Push(branch); err != nil {
		return err
	}

	if !cfg.Push {
		slog.Info("Push flag is not specified, skipping pull request creation")
		return nil
	}

	// Ensure we have a GitHub repository
	gitHubRepo, err := github.FetchGitHubRepoFromRemote(repo)
	if err != nil {
		return err
	}

	title := fmt.Sprintf("Librarian %s pull request: %s", info.prType, datetimeNow)
	prBody, err := createPRBody(info)
	if err != nil {
		return fmt.Errorf("failed to create pull request body: %w", err)
	}

	pullRequestMetadata, err := info.ghClient.CreatePullRequest(ctx, gitHubRepo, branch, cfg.Branch, title, prBody)
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	return addLabelsToPullRequest(ctx, info.ghClient, info.pullRequestLabels, pullRequestMetadata)
}

// addLabelsToPullRequest adds a list of labels to a single pull request (specified by the id number).
// Should only be called on a valid Github pull request.
// Passing in `nil` for labels will no-op and an empty list for labels will clear all labels on the PR.
// TODO: Consolidate the params to a potential PullRequestInfo struct.
func addLabelsToPullRequest(ctx context.Context, ghClient GitHubClient, pullRequestLabels []string, prMetadata *github.PullRequestMetadata) error {
	// Do not update if there are aren't labels provided
	if pullRequestLabels == nil {
		return nil
	}
	// Github API treats Issues and Pull Request the same
	// https://docs.github.com/en/rest/issues/labels#add-labels-to-an-issue
	if err := ghClient.AddLabelsToIssue(ctx, prMetadata.Repo, prMetadata.Number, pullRequestLabels); err != nil {
		return fmt.Errorf("failed to add labels to pull request: %w", err)
	}
	return nil
}

func createPRBody(info *commitInfo) (string, error) {
	switch info.prType {
	case generate:
		return formatGenerationPRBody(info.repo, info.state, info.idToCommits, info.failedLibraries)
	case release:
		return formatReleaseNotes(info.repo, info.state)
	default:
		return "", fmt.Errorf("unrecognized pull request type: %s", info.prType)
	}
}

func copyFile(dst, src string) (err error) {
	lstat, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("failed to lstat file: %q: %w", src, err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to make directory for %q: %w", dst, err)
	}

	if lstat.Mode()&os.ModeSymlink == os.ModeSymlink {
		linkTarget, err := os.Readlink(src)
		if err != nil {
			return fmt.Errorf("failed to read link: %q: %w", src, err)
		}
		// Remove existing file at dst if it exists. os.Symlink will fail otherwise.
		if _, err := os.Lstat(dst); err == nil {
			if err := os.Remove(dst); err != nil {
				return fmt.Errorf("failed to remove existing file at destination: %q: %w", dst, err)
			}
		}
		return os.Symlink(linkTarget, dst)
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open file: %q: %w", src, err)
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create file: %s", dst)
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)

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
