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
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

const (
	defaultAPISourceBranch  = "master"
	prBodyFile              = "pr-body.txt"
	failedGenerationComment = `One or more libraries have failed to generate, please review PR description for a list of failed libraries.
For each failed library, open a ticket in that library’s repository and then you may resolve this comment and merge.
`
)

var errBuilderNotProvided = fmt.Errorf("no prBodyBuilder provided")

type pullRequestType int

const (
	pullRequestUnspecified pullRequestType = iota
	pullRequestOnboard
	pullRequestGenerate
	pullRequestRelease
	pullRequestUpdateImage
)

// String returns the string representation of a pullRequestType.
// It returns unknown if the type is not a recognized constant.
func (t pullRequestType) String() string {
	names := map[pullRequestType]string{
		pullRequestUnspecified: "unspecified",
		pullRequestOnboard:     "onboard",
		pullRequestGenerate:    "generate",
		pullRequestRelease:     "release",
		pullRequestUpdateImage: "update image",
	}
	if name, ok := names[t]; ok {
		return name
	}
	return "unspecified"
}

var globalPreservePatterns = []string{
	fmt.Sprintf(`^%s(/.*)?$`, regexp.QuoteMeta(config.GeneratorInputDir)), // Preserve the generator-input directory and its contents.
}

// GitHubClient is an abstraction over the GitHub client.
type GitHubClient interface {
	GetRawContent(ctx context.Context, path, ref string) ([]byte, error)
	CreatePullRequest(ctx context.Context, repo *github.Repository, remoteBranch, remoteBase, title, body string, isDraft bool) (*github.PullRequestMetadata, error)
	AddLabelsToIssue(ctx context.Context, repo *github.Repository, number int, labels []string) error
	GetLabels(ctx context.Context, number int) ([]string, error)
	ReplaceLabels(ctx context.Context, number int, labels []string) error
	SearchPullRequests(ctx context.Context, query string) ([]*github.PullRequest, error)
	GetPullRequest(ctx context.Context, number int) (*github.PullRequest, error)
	CreateRelease(ctx context.Context, tagName, name, body, commitish string) (*github.RepositoryRelease, error)
	CreateIssueComment(ctx context.Context, number int, comment string) error
	CreateTag(ctx context.Context, tag, commitish string) error
}

// ContainerClient is an abstraction over the Docker client.
type ContainerClient interface {
	Build(ctx context.Context, request *docker.BuildRequest) error
	Configure(ctx context.Context, request *docker.ConfigureRequest) (string, error)
	Generate(ctx context.Context, request *docker.GenerateRequest) error
	ReleaseStage(ctx context.Context, request *docker.ReleaseStageRequest) error
}

type commitInfo struct {
	// branch is the base branch of the created pull request.
	branch string
	// commit declares whether to create a commit.
	commit bool
	// commitMessage is used as the message on the actual git commit.
	commitMessage string
	// ghClient is used to interact with the GitHub API.
	ghClient GitHubClient
	// prType is an enum for which type of librarian pull request we are creating.
	prType pullRequestType
	// pullRequestLabels is a list of labels to add to the created pull request.
	pullRequestLabels []string
	// push declares whether to push the commits to GitHub.
	push bool
	// languageRepo is the git repository containing the language-specific libraries.
	languageRepo gitrepo.Repository
	// sourceRepo is the git repository containing the source protos.
	sourceRepo gitrepo.Repository
	// state is the librarian state.yaml contents.
	state *config.LibrarianState
	// workRoot is the directory that we stage code changes in.
	workRoot string
	// failedGenerations is the number of generations that failed.
	failedGenerations int
	// api is the api path of a library, only set this value during api onboarding.
	api string
	// library is the ID of a library, only set this value during api onboarding.
	library string
	// prBodyBuilder is a callback function for building the pull request body
	prBodyBuilder func() (string, error)
	// isDraft declares whether to create the pull request as a draft.
	isDraft bool
}

type commandRunner struct {
	repo            gitrepo.Repository
	sourceRepo      gitrepo.Repository
	state           *config.LibrarianState
	librarianConfig *config.LibrarianConfig
	ghClient        GitHubClient
	containerClient ContainerClient
	image           string
	workRoot        string
}

func newCommandRunner(cfg *config.Config) (*commandRunner, error) {
	languageRepo, err := cloneOrOpenRepo(cfg.WorkRoot, cfg.Repo, cfg.APISourceDepth, cfg.Branch, cfg.CI, cfg.GitHubToken)
	if err != nil {
		return nil, err
	}

	var (
		sourceRepo    gitrepo.Repository
		sourceRepoDir string
	)

	// If APISource is set, checkout the protos repository.
	if cfg.APISource != "" {
		sourceRepo, err = cloneOrOpenRepo(cfg.WorkRoot, cfg.APISource, cfg.APISourceDepth, defaultAPISourceBranch, cfg.CI, cfg.GitHubToken)
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

	gitHubRepo, err := GetGitHubRepository(cfg, languageRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub repository: %w", err)
	}

	ghClient := github.NewClient(cfg.GitHubToken, gitHubRepo)
	container, err := docker.New(cfg.WorkRoot, image, &docker.DockerOptions{
		UserUID:   cfg.UserUID,
		UserGID:   cfg.UserGID,
		HostMount: cfg.HostMount,
	})
	if err != nil {
		return nil, err
	}
	return &commandRunner{
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

func cloneOrOpenRepo(workRoot, repo string, depth int, branch, ci string, gitPassword string) (*gitrepo.LocalRepository, error) {
	if repo == "" {
		return nil, fmt.Errorf("repo must be specified")
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
			Depth:        depth,
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

func formatTimestamp(t time.Time) string {
	const yyyyMMddHHmmss = "20060102T150405Z" // Expected format by time library
	return t.Format(yyyyMMddHHmmss)
}

// cleanAndCopyLibrary cleans the files of the given library in repoDir and copies
// the new files from outputDir.
func cleanAndCopyLibrary(state *config.LibrarianState, repoDir, libraryID, outputDir string) error {
	library := state.LibraryByID(libraryID)
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

	preservePatterns := append(library.PreserveRegex, globalPreservePatterns...)

	if err := clean(repoDir, library.SourceRoots, removePatterns, preservePatterns); err != nil {
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
	library := state.LibraryByID(libraryID)
	if library == nil {
		return fmt.Errorf("library %q not found", libraryID)
	}
	slog.Info("copying library files", "id", library.ID, "destination", dest, "source", src)
	for _, srcRoot := range library.SourceRoots {
		dstPath := filepath.Join(dest, srcRoot)
		srcPath := filepath.Join(src, srcRoot)
		files, err := getDirectoryFilenames(srcPath)
		if err != nil {
			return err
		}
		for _, file := range files {
			slog.Debug("copying file", "file", file)
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
	if !info.push && !info.commit {
		slog.Info("push flag and Commit flag are not specified, skipping committing")
		return writePRBody(info)
	}

	repo := info.languageRepo
	if err := repo.AddAll(); err != nil {
		return err
	}
	isClean, err := repo.IsClean()
	if err != nil {
		return err
	}

	if isClean {
		slog.Info("no changes to commit, skipping commit and push.")
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

	if !info.push {
		slog.Info("push flag is not specified, skipping pull request creation")
		return writePRBody(info)
	}

	if err := repo.Push(branch); err != nil {
		return err
	}

	gitHubRepo, err := GetGitHubRepositoryFromGitRepo(info.languageRepo)
	if err != nil {
		return fmt.Errorf("failed to get GitHub repository: %w", err)
	}

	title := fmt.Sprintf("chore: librarian %s pull request: %s", info.prType, datetimeNow)
	prBody, err := info.prBodyBuilder()
	if err != nil {
		return fmt.Errorf("failed to create pull request body: %w", err)
	}

	pullRequestMetadata, err := info.ghClient.CreatePullRequest(ctx, gitHubRepo, branch, info.branch, title, prBody, info.isDraft)
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	if info.failedGenerations != 0 {
		if err := info.ghClient.CreateIssueComment(ctx, pullRequestMetadata.Number, failedGenerationComment); err != nil {
			return fmt.Errorf("failed to add pull request comment: %w", err)
		}
	}

	return addLabelsToPullRequest(ctx, info.ghClient, info.pullRequestLabels, pullRequestMetadata)
}

// writePRBody attempts to log the body of a PR that would have been created if the
// -push flag had been specified. This logs any errors and returns them to the
// caller.
func writePRBody(info *commitInfo) error {
	if info.prBodyBuilder == nil {
		return errBuilderNotProvided
	}

	prBody, err := info.prBodyBuilder()
	if err != nil {
		slog.Warn("unable to create PR body", "error", err)
		return err
	}
	// Note: we can't accurately predict whether a PR would have been created,
	// as we're not checking whether the repo is clean or not. The intention is to be
	// as light-touch as possible.
	fullPath := filepath.Join(info.workRoot, prBodyFile)
	// Ensure that "cat [path-to-pr-body.txt]" gives useful output.
	prBody = prBody + "\n"
	err = os.WriteFile(fullPath, []byte(prBody), 0644)
	if err != nil {
		slog.Warn("unable to save PR body", "error", err)
		return err
	}
	slog.Info("wrote body of pull request that might have been created", "file", fullPath)
	return nil
}

// addLabelsToPullRequest adds a list of labels to a single pull request (specified by the id number).
// Should only be called on a valid Github pull request.
// Passing in `nil` for labels will no-op and an empty list for labels will clear all labels on the PR.
// TODO: Consolidate the params to a potential PullRequestInfo struct.
func addLabelsToPullRequest(ctx context.Context, ghClient GitHubClient, pullRequestLabels []string, prMetadata *github.PullRequestMetadata) error {
	// Do not update if there aren't labels provided
	if pullRequestLabels == nil {
		return nil
	}
	// GitHub API treats Issues and Pull Request the same
	// https://docs.github.com/en/rest/issues/labels#add-labels-to-an-issue
	if err := ghClient.AddLabelsToIssue(ctx, prMetadata.Repo, prMetadata.Number, pullRequestLabels); err != nil {
		return fmt.Errorf("failed to add labels to pull request: %w", err)
	}
	return nil
}

// copyGlobalAllowlist copies files in the global file allowlist from src to dst.
func copyGlobalAllowlist(cfg *config.LibrarianConfig, dst, src string, copyReadOnly bool) error {
	if cfg == nil {
		slog.Info("librarian config is not setup, skip copying global allowlist")
		return nil
	}
	slog.Info("copying global allowlist files", "destination", dst, "source", src)
	for _, globalFile := range cfg.GlobalFilesAllowlist {
		if globalFile.Permissions == config.PermissionReadOnly && !copyReadOnly {
			slog.Debug("skipping read-only file", "path", globalFile.Path)
			continue
		}

		srcPath := filepath.Join(src, globalFile.Path)
		if _, err := os.Lstat(srcPath); os.IsNotExist(err) {
			slog.Info("skip copying a non-existent global allowlist file", "source", srcPath)
			continue
		}
		dstPath := filepath.Join(dst, globalFile.Path)
		if err := copyFile(dstPath, srcPath); err != nil {
			return fmt.Errorf("failed to copy global file %s from %s: %w", dstPath, srcPath, err)
		}
	}
	return nil
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

// clean removes files and directories from source roots based on remove and preserve patterns.
// Limit the possible files when cleaning to those in source roots (not rootDir) as regex patterns
// for preserve and remove should ONLY impact source root files.
//
// It first determines the paths to remove by applying the removePatterns and then excluding any paths
// that match the preservePatterns. It then separates the remaining paths into files and directories and
// removes them, ensuring that directories are removed last.
//
// This logic is ported from owlbot logic: https://github.com/googleapis/repo-automation-bots/blob/12dad68640960290910b660e4325630c9ace494b/packages/owl-bot/src/copy-code.ts#L1027
func clean(rootDir string, sourceRoots, removePatterns, preservePatterns []string) error {
	slog.Info("cleaning directories", "source roots", sourceRoots)

	// relPaths contains a list of files in source root's relative paths from rootDir. The
	// regex patterns for preserve and remove apply to a source root's relative path
	var relPaths []string
	for _, sourceRoot := range sourceRoots {
		sourceRootPath := filepath.Join(rootDir, sourceRoot)
		// If a source root does not exist, log a warning and searching the other source roots.
		// TODO: Consider not calling clean if it's a first time generation
		if _, err := os.Lstat(sourceRootPath); err != nil {
			if os.IsNotExist(err) {
				slog.Warn("unable to find source root. It may be an initial generation request", "source root", sourceRoot)
				continue
			}
			// For any other error (permissions, I/O, etc.)
			slog.Error("error trying to clean source root", "source root", sourceRoot, "error", err)
			return err
		}
		sourceRootPaths, err := findSubDirRelPaths(rootDir, sourceRootPath)
		if err != nil {
			// Log a warning and continue processing other source roots. There may be other files
			// that can be cleaned up.
			slog.Warn("unable to search for files in a source root", "source root", sourceRoot, "error", err)
			continue
		}
		if len(sourceRootPaths) == 0 {
			slog.Info("source root does not contain any files", "source root", sourceRoot)
		}
		relPaths = append(relPaths, sourceRootPaths...)
	}

	if len(relPaths) == 0 {
		slog.Info("there are no files to be cleaned in source roots", "source roots", sourceRoots)
		return nil
	}

	pathsToRemove, err := filterPathsForRemoval(relPaths, removePatterns, preservePatterns)
	if err != nil {
		return err
	}

	// prepend the rootDir to each path to ensure that os.Remove can find the file
	var paths []string
	for _, path := range pathsToRemove {
		paths = append(paths, filepath.Join(rootDir, path))
	}

	filesToRemove, dirsToRemove, err := separateFilesAndDirs(paths)
	if err != nil {
		return err
	}

	// Remove files first, then directories.
	for _, file := range filesToRemove {
		slog.Debug("removing file", "path", file)
		if err := os.Remove(file); err != nil {
			return err
		}
	}

	// Sort to remove the child directories first
	slices.SortFunc(dirsToRemove, func(a, b string) int {
		return strings.Count(b, string(filepath.Separator)) - strings.Count(a, string(filepath.Separator))
	})

	for _, dir := range dirsToRemove {
		slog.Debug("removing directory", "path", dir)
		if err := os.Remove(dir); err != nil {
			// It's possible the directory is not empty due to preserved files.
			slog.Warn("failed to remove directory, it may not be empty", "dir", dir, "err", err)
		}
	}

	return nil
}

// findSubDirRelPaths walks the subDir tree returns a slice of all file and directory paths
// relative to the dir. This is repeated for all nested directories. subDir must be under
// or the same as dir.
func findSubDirRelPaths(dir, subDir string) ([]string, error) {
	dirRelPath, err := filepath.Rel(dir, subDir)
	if err != nil {
		return nil, fmt.Errorf("cannot establish the relationship between %s and %s: %w", dir, subDir, err)
	}
	// '..' signifies that the subDir exists outside of dir
	if strings.HasPrefix(dirRelPath, "..") {
		return nil, fmt.Errorf("subDir is not nested within the dir: %s, %s", subDir, dir)
	}

	var paths []string
	err = filepath.WalkDir(subDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// error is ignored as we have confirmed that subDir is child or equal to rootDir
		relPath, _ := filepath.Rel(dir, path)
		// Special case when subDir is equal to dir. Drop the "." as it references itself
		if relPath != "." {
			paths = append(paths, relPath)
		}
		return nil
	})
	return paths, err
}

// filterPathsByRegex returns a new slice containing only the paths from the input slice
// that match at least one of the provided regular expressions.
func filterPathsByRegex(paths []string, regexps []*regexp.Regexp) []string {
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

// filterPathsForRemoval determines the list of paths to be removed. The logic runs as follows:
// 1. paths that match any removePatterns are marked for removal
// 2. paths that match the preservePatterns are kept (even if they match removePatterns)
// Paths that match both are kept as preserve has overrides.
func filterPathsForRemoval(paths, removePatterns, preservePatterns []string) ([]string, error) {
	removeRegexps, err := compileRegexps(removePatterns)
	if err != nil {
		return nil, err
	}
	preserveRegexps, err := compileRegexps(preservePatterns)
	if err != nil {
		return nil, err
	}

	pathsToRemove := filterPathsByRegex(paths, removeRegexps)
	pathsToPreserve := filterPathsByRegex(pathsToRemove, preserveRegexps)

	// map for a quick lookup for any preserve paths
	preserveMap := make(map[string]bool)
	for _, p := range pathsToPreserve {
		preserveMap[p] = true
	}
	finalPathsToRemove := slices.DeleteFunc(pathsToRemove, func(path string) bool {
		return preserveMap[path]
	})
	return finalPathsToRemove, nil
}

// separateFilesAndDirs takes a list of paths and categorizes them into files
// and directories. It uses os.Lstat to avoid following symlinks, treating them
// as files. Paths that do not exist are silently ignored.
func separateFilesAndDirs(paths []string) ([]string, []string, error) {
	var filePaths, dirPaths []string
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			// The file or directory may have already been removed.
			if errors.Is(err, os.ErrNotExist) {
				slog.Warn("unable to find path", "path", path)
				continue
			}
			// For any other error (permissions, I/O, etc.)
			return nil, nil, fmt.Errorf("failed to stat path %q: %w", path, err)

		}
		if info.IsDir() {
			dirPaths = append(dirPaths, path)
		} else {
			filePaths = append(filePaths, path)
		}
	}
	return filePaths, dirPaths, nil
}

func isURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}
