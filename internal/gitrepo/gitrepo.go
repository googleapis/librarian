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

// Package gitrepo provides operations on git repos.
package gitrepo

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	httpAuth "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// Repository defines the interface for git repository operations.
type Repository interface {
	AddAll() error
	Commit(msg string) error
	IsClean() (bool, error)
	Remotes() ([]*Remote, error)
	GetDir() string
	HeadHash() (string, error)
	ChangedFilesInCommit(commitHash string) ([]string, error)
	ChangedFiles() ([]string, error)
	GetCommit(commitHash string) (*Commit, error)
	GetLatestCommit(path string) (*Commit, error)
	GetCommitsForPathsSinceTag(paths []string, tagName string) ([]*Commit, error)
	GetCommitsForPathsSinceCommit(paths []string, sinceCommit string) ([]*Commit, error)
	CreateBranchAndCheckout(name string) error
	Push(branchName string) error
	Restore(paths []string) error
	CleanUntracked(paths []string) error
	pushRefSpec(refSpec string) error
	Checkout(commitHash string) error
	GetHashForPath(commitHash, path string) (string, error)
}

const RootPath = "."

// LocalRepository represents a git repository.
type LocalRepository struct {
	Dir         string
	repo        *git.Repository
	gitPassword string
}

// Commit represents a git commit.
type Commit struct {
	Hash    plumbing.Hash
	Message string
	When    time.Time
}

// Remote represent a git remote.
type Remote struct {
	Name string
	URLs []string
}

// RepositoryOptions are used to configure a [LocalRepository].
type RepositoryOptions struct {
	// Dir is the directory where the repository will reside locally. Required.
	Dir string
	// MaybeClone will try to clone the repository if it does not exist locally.
	// If set to true, RemoteURL and RemoteBranch must also be set. Optional.
	MaybeClone bool
	// RemoteURL is the URL of the remote repository to clone from. Required if MaybeClone is set to true.
	RemoteURL string
	// RemoteBranch is the remote branch to clone. Required if MaybeClone is set to true.
	RemoteBranch string
	// CI is the type of Continuous Integration (CI) environment in which
	// the tool is executing.
	CI string
	// GitPassword is used for HTTP basic auth.
	GitPassword string
	// Depth controls the cloning depth if the repository needs to be cloned.
	Depth int
}

// NewRepository provides access to a git repository based on the provided options.
//
// If opts.Clone is CloneOptionNone, it opens an existing repository at opts.Dir.
// If opts.Clone is CloneOptionMaybe, it opens the repository if it exists,
// otherwise it clones from opts.RemoteURL.
// If opts.Clone is CloneOptionAlways, it always clones from opts.RemoteURL.
func NewRepository(opts *RepositoryOptions) (*LocalRepository, error) {
	repo, err := newRepositoryWithoutUser(opts)
	if err != nil {
		return repo, err
	}
	repo.gitPassword = opts.GitPassword
	return repo, nil
}

func newRepositoryWithoutUser(opts *RepositoryOptions) (*LocalRepository, error) {
	if opts.Dir == "" {
		return nil, errors.New("gitrepo: dir is required")
	}

	if !opts.MaybeClone {
		return open(opts.Dir)
	}
	slog.Info("checking for repository", "dir", opts.Dir)
	_, err := os.Stat(opts.Dir)
	if err == nil {
		return open(opts.Dir)
	}
	if os.IsNotExist(err) {
		if opts.RemoteURL == "" {
			return nil, fmt.Errorf("gitrepo: remote URL is required when cloning")
		}
		if opts.RemoteBranch == "" {
			return nil, fmt.Errorf("gitrepo: remote branch is required when cloning")
		}
		slog.Info("repository not found, executing clone")
		return clone(opts.Dir, opts.RemoteURL, opts.RemoteBranch, opts.CI, opts.Depth)
	}
	return nil, fmt.Errorf("failed to check for repository at %q: %w", opts.Dir, err)
}

func open(dir string) (*LocalRepository, error) {
	slog.Info("opening repository", "dir", dir)
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return nil, err
	}

	return &LocalRepository{
		Dir:  dir,
		repo: repo,
	}, nil
}

func clone(dir, url, branch, ci string, depth int) (*LocalRepository, error) {
	slog.Info("cloning repository", "url", url, "dir", dir)
	options := &git.CloneOptions{
		URL:           url,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
		Tags:          git.AllTags,
		Depth:         depth,
		// .NET uses submodules for conformance tests.
		// (There may be other examples too.)
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	}
	if ci == "" {
		options.Progress = os.Stdout // When not a CI build, output progress.
	}

	repo, err := git.PlainClone(dir, false, options)
	if err != nil {
		return nil, err
	}
	return &LocalRepository{
		Dir:  dir,
		repo: repo,
	}, nil
}

// AddAll adds all pending changes from the working tree to the index,
// so that the changes can later be committed.
func (r *LocalRepository) AddAll() error {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return err
	}
	err = worktree.AddWithOptions(&git.AddOptions{All: true})
	if err != nil {
		return err
	}
	return nil
}

// Commit creates a new commit with the provided message and author
// information.
func (r *LocalRepository) Commit(msg string) error {
	slog.Info("committing", "message", msg)
	worktree, err := r.repo.Worktree()
	if err != nil {
		return err
	}

	status, err := worktree.Status()
	if err != nil {
		return err
	}
	if status.IsClean() {
		return fmt.Errorf("no modifications to commit")
	}
	// The author of the commit will be read from git config.
	hash, err := worktree.Commit(msg, &git.CommitOptions{})
	if err != nil {
		return err
	}

	// Log commit hash (brief) and subject line (first line of commit)
	subject := strings.Split(msg, "\n")[0]
	slog.Info(fmt.Sprintf("Committed %s: '%s'", hash.String()[0:7], subject))
	return nil
}

// IsClean reports whether the working tree has no uncommitted changes.
func (r *LocalRepository) IsClean() (bool, error) {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return false, err
	}
	status, err := worktree.Status()
	if err != nil {
		return false, err
	}

	return status.IsClean(), nil
}

// ChangedFiles returns a list of files that have been modified, added, or deleted
// in the working tree, including both staged and unstaged changes.
func (r *LocalRepository) ChangedFiles() ([]string, error) {
	slog.Debug("getting changed files")
	worktree, err := r.repo.Worktree()
	if err != nil {
		return nil, err
	}
	status, err := worktree.Status()
	if err != nil {
		return nil, err
	}
	var changedFiles []string
	for file, fileStatus := range status {
		if fileStatus.Staging != git.Unmodified || fileStatus.Worktree != git.Unmodified {
			changedFiles = append(changedFiles, file)
		}
	}
	return changedFiles, nil
}

// Remotes returns the remotes within the repository.
func (r *LocalRepository) Remotes() ([]*Remote, error) {
	gitRemotes, err := r.repo.Remotes()
	if err != nil {
		return nil, err
	}
	var remotes []*Remote
	for _, remote := range gitRemotes {
		remotes = append(remotes, &Remote{Name: remote.Config().Name, URLs: remote.Config().URLs})
	}

	return remotes, nil
}

// HeadHash returns hash of the commit for the repository's HEAD.
func (r *LocalRepository) HeadHash() (string, error) {
	ref, err := r.repo.Head()
	if err != nil {
		return "", err
	}
	return ref.Hash().String(), nil
}

// GetDir returns the directory of the repository.
func (r *LocalRepository) GetDir() string {
	return r.Dir
}

// GetCommit returns a commit for the given commit hash.
func (r *LocalRepository) GetCommit(commitHash string) (*Commit, error) {
	commit, err := r.repo.CommitObject(plumbing.NewHash(commitHash))
	if err != nil {
		return nil, err
	}

	return &Commit{
		Hash:    commit.Hash,
		Message: commit.Message,
		When:    commit.Author.When,
	}, nil
}

// GetLatestCommit returns the latest commit of the given path in the repository.
func (r *LocalRepository) GetLatestCommit(path string) (*Commit, error) {
	slog.Info("retrieving the latest commit", "path", path)
	opt := &git.LogOptions{
		Order:    git.LogOrderCommitterTime,
		FileName: &path,
	}
	log, err := r.repo.Log(opt)
	if err != nil {
		return nil, err
	}

	commit, err := log.Next()
	if err != nil {
		return nil, err
	}

	return &Commit{
		Hash:    commit.Hash,
		Message: commit.Message,
		When:    commit.Author.When,
	}, nil
}

// GetCommitsForPathsSinceTag returns all commits since tagName that contains
// files in paths.
//
// If tagName empty, all commits for the given paths are returned.
func (r *LocalRepository) GetCommitsForPathsSinceTag(paths []string, tagName string) ([]*Commit, error) {
	var hash string
	if tagName == "" {
		return r.GetCommitsForPathsSinceCommit(paths, "")
	}
	tagRef, err := r.repo.Tag(tagName)
	if err != nil {
		return nil, fmt.Errorf("failed to find tag %s: %w", tagName, err)
	}

	tagCommit, err := r.repo.CommitObject(tagRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit object for tag %s: %w", tagName, err)
	}
	hash = tagCommit.Hash.String()

	return r.GetCommitsForPathsSinceCommit(paths, hash)
}

// GetCommitsForPathsSinceCommit returns the commits affecting any of the given
// paths, stopping looking at the given commit (which is not included in the
// results).
// The returned commits are ordered such that the most recent commit is first.
//
// If sinceCommit is not provided, all commits are searched; otherwise, if no
// commit matching sinceCommit is found, an error is returned.
func (r *LocalRepository) GetCommitsForPathsSinceCommit(paths []string, sinceCommit string) ([]*Commit, error) {
	if len(paths) == 0 {
		return nil, errors.New("no paths to check for commits")
	}
	var commits []*Commit
	finalHash := plumbing.NewHash(sinceCommit)
	logOptions := git.LogOptions{Order: git.LogOrderCommitterTime}
	logIterator, err := r.repo.Log(&logOptions)
	if err != nil {
		return nil, err
	}
	// Sentinel "error" - this can be replaced using LogOptions.To when that's available.
	ErrStopIterating := fmt.Errorf("iteration done")
	err = logIterator.ForEach(func(commit *object.Commit) error {
		if commit.Hash == finalHash {
			return ErrStopIterating
		}
		// Skips the initial commit as it has no parents.
		// This is a known limitation that should be addressed in the future.
		// Skip any commit with multiple parents. We shouldn't see this
		// as we don't use merge commits.
		if commit.NumParents() != 1 {
			return nil
		}
		parentCommit, err := commit.Parent(0)
		if err != nil {
			return err
		}

		// We perform filtering by finding out if the tree hash for the given
		// path at the commit we're looking at is the same as the tree hash
		// for the commit's parent.
		// This is much, much faster than any other filtering option, it seems.
		// In theory, we should be able to remember our "current" commit for each
		// path, but that's likely to be significantly more complex.
		for _, candidatePath := range paths {
			matching, err := commitMatchesPath(candidatePath, commit, parentCommit)
			if err != nil {
				return err
			}
			// If we've found a change (including a path being added or removed),
			// add it to our list of commits and proceed to the next commit.
			if matching {
				commits = append(commits, &Commit{
					Hash:    commit.Hash,
					Message: commit.Message,
					When:    commit.Author.When,
				})
				return nil
			}
		}

		return nil
	})
	if err != nil && !errors.Is(err, ErrStopIterating) {
		return nil, err
	}
	if sinceCommit != "" && !errors.Is(err, ErrStopIterating) {
		return nil, fmt.Errorf("did not find commit %s when iterating", sinceCommit)
	}
	return commits, nil
}

func commitMatchesPath(path string, commit *object.Commit, parentCommit *object.Commit) (bool, error) {
	if path == RootPath {
		return true, nil
	}
	currentPathHash, err := getHashForPath(commit, path)
	if err != nil {
		return false, err
	}
	parentPathHash, err := getHashForPath(parentCommit, path)
	if err != nil {
		return false, err
	}
	return currentPathHash != parentPathHash, nil
}

// getHashForPath returns the hash for a path at a given commit, or an
// empty string if the path (file or directory) did not exist.
func getHashForPath(commit *object.Commit, path string) (string, error) {
	tree, err := commit.Tree()
	if err != nil {
		return "", err
	}
	treeEntry, err := tree.FindEntry(path)
	if errors.Is(err, object.ErrEntryNotFound) || errors.Is(err, object.ErrDirectoryNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return treeEntry.Hash.String(), nil
}

// ChangedFilesInCommit returns the files changed in the given commit.
func (r *LocalRepository) ChangedFilesInCommit(commitHash string) ([]string, error) {
	slog.Debug("getting changed files in commit", "hash", commitHash)
	commit, err := r.repo.CommitObject(plumbing.NewHash(commitHash))
	if err != nil {
		return nil, fmt.Errorf("failed to get commit object for hash %s: %w", commitHash, err)
	}

	var fromTree *object.Tree
	var toTree *object.Tree

	toTree, err = commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree for commit %s: %w", commitHash, err)
	}

	if commit.NumParents() == 0 {
		fromTree = &object.Tree{} // Empty tree for initial commit
	} else {
		parent, err := commit.Parent(0)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent for commit %s: %w", commitHash, err)
		}
		fromTree, err = parent.Tree()
		if err != nil {
			return nil, fmt.Errorf("failed to get parent tree for commit %s: %w", commitHash, err)
		}
	}

	changes, err := fromTree.Diff(toTree)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff for commit %s: %w", commitHash, err)
	}
	var files []string
	for _, change := range changes {
		from := change.From.Name
		to := change.To.Name
		// Deletion or modification
		if from != "" {
			files = append(files, from)
		}
		// Insertion or rename
		if to != "" && from != to {
			files = append(files, to)
		}
	}
	return files, nil
}

// CreateBranchAndCheckout creates a new git branch and checks out the
// branch in the local git repository.
func (r *LocalRepository) CreateBranchAndCheckout(name string) error {
	slog.Info("creating branch and checking out", "name", name)
	worktree, err := r.repo.Worktree()
	if err != nil {
		return err
	}
	return worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(name),
		Create: true,
		Keep:   true,
	})
}

// Push pushes the local branch to the origin remote.
func (r *LocalRepository) Push(branchName string) error {
	// https://stackoverflow.com/a/75727620
	refSpec := fmt.Sprintf("+refs/heads/%s:refs/heads/%s", branchName, branchName)
	slog.Info("pushing changes", "branch name", branchName, slog.Any("refspec", refSpec))
	return r.pushRefSpec(refSpec)
}

// DeleteBranch deletes a branch on the origin remote.
func (r *LocalRepository) DeleteBranch(branchName string) error {
	refSpec := fmt.Sprintf(":refs/heads/%s", branchName)
	return r.pushRefSpec(refSpec)
}

func (r *LocalRepository) pushRefSpec(refSpec string) error {
	slog.Info("pushing changes", "refSpec", refSpec)

	// Check for the configured URI for the `origin` remote.
	// If there are multiple URLs, the first one is selected.
	var remoteURI string
	remotes, err := r.Remotes()
	if err != nil {
		return err
	}
	for _, remote := range remotes {
		if remote.Name == "origin" {
			if len(remote.URLs) > 0 {
				remoteURI = remote.URLs[0]
			}
		}
	}

	useSSH := canUseSSH(remoteURI)
	// While cloning a public repo does not require any authCreds, pushing
	// to the repo requires authentication and verification of identity
	auth, err := r.authCreds(useSSH)
	if err != nil {
		return err
	}
	if err := r.repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{config.RefSpec(refSpec)},
		Auth:       auth,
	}); err != nil {
		return err
	}
	slog.Info("successfully pushed changes", "refSpec", refSpec)
	return nil
}

// canUseSSH returns if the remote URI can connect via https ssh. It attempts to
// automatically determine the type and returns false as a default if it's unable
// to make a determination.
func canUseSSH(remoteURI string) bool {
	// First, try to parse it as a standard URL
	// e.g. "https://github.com/golang/go.git", "ssh://git@github.com/golang/go.git"
	parsedURL, err := url.Parse(remoteURI)
	if err == nil && parsedURL.Scheme != "" {
		// Check the scheme directly
		switch parsedURL.Scheme {
		case "https":
			return false
		case "ssh":
			return true
		}
	}

	// If parsing fails or the scheme is not standard, check for the `user@hostname`
	// SSH syntax (e.g., "git@github.com:user/repo.git"). This format doesn't
	// have a "://" and contains a ":"
	if !strings.Contains(remoteURI, "://") && strings.Contains(remoteURI, ":") {
		return true
	}

	return false
}

// authCreds returns the configured AuthMethod to used to pushing to the
// remote repository. The useSSH determines if Basic Auth or SSH is used.
func (r *LocalRepository) authCreds(useSSH bool) (transport.AuthMethod, error) {
	if useSSH {
		slog.Info("authenticating with SSH")
		// This is the generic `git` username when cloning via SSH. It is the value
		// that exists before the URL. e.g. git@github.com:googleapis/librarian.git
		auth, err := ssh.DefaultAuthBuilder("git")
		if err != nil {
			return nil, err
		}
		return auth, nil
	}
	slog.Info("authenticating with basic auth")
	return &httpAuth.BasicAuth{
		// GitHub's authentication needs the username set to a non-empty value, but
		// it does not need to match the token
		Username: "cloud-sdk-librarian",
		Password: r.gitPassword,
	}, nil
}

// Restore restores changes in the working tree, leaving staged area untouched.
// Note that untracked files, if any, are not touched.
//
// Wrap git operations in exec, because [git.Worktree.Restore] does not support
// this operation.
func (r *LocalRepository) Restore(paths []string) error {
	args := []string{"restore"}
	args = append(args, paths...)
	slog.Info("restoring uncommitted changes", "paths", strings.Join(paths, ","))
	cmd := exec.Command("git", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Dir = r.Dir
	return cmd.Run()
}

// CleanUntracked removes untracked files within the given paths.
func (r *LocalRepository) CleanUntracked(paths []string) error {
	slog.Info("cleaning untracked files", "paths", strings.Join(paths, ","))
	worktree, err := r.repo.Worktree()
	if err != nil {
		return err
	}

	status, err := worktree.Status()
	if err != nil {
		return err
	}

	// Remove an untracked file if it lives within one of the given paths.
	for _, path := range paths {
		for file, fileStatue := range status {
			if !strings.Contains(file, path) || fileStatue.Worktree != git.Untracked {
				continue
			}

			relPath := filepath.Join(r.Dir, file)
			if err := os.Remove(relPath); err != nil {
				return fmt.Errorf("failed to remove untracked file, %s: %q", relPath, err)
			}
		}
	}

	return nil
}

// Checkout checks out the local repository at the provided git SHA.
func (r *LocalRepository) Checkout(commitSha string) error {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return err
	}
	return worktree.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(commitSha),
	})
}

// GetHashForPath returns a tree hash for the specified path,
// at the given commit in this repository. If the path does not exist
// at the commit, an empty string is returned rather than an error,
// as the purpose of this function is to allow callers to determine changes
// in the tree. (A path going from missing to anything else, or vice versa,
// indicates a change. A path being missing at two different commits is not a change.)
func (r *LocalRepository) GetHashForPath(commitHash, path string) (string, error) {
	// This public function just delegates to the internal function that uses a Commit
	// object instead of the hash.
	commit, err := r.repo.CommitObject(plumbing.NewHash(commitHash))
	if err != nil {
		return "", err
	}
	return getHashForPath(commit, path)
}
