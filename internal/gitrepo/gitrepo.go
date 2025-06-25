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
	"os"
	"slices"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// Repository represents a git repository.
type Repository struct {
	Dir  string
	repo *git.Repository
}

// Commit represents a git commit.
type Commit struct {
	Hash    plumbing.Hash
	Message string
}

// RepositoryOptions are used to configure a [Repository].
type RepositoryOptions struct {
	// Dir is the directory where the repository will reside locally. Required.
	Dir string
	// MaybeClone will try to clone the repository if it does not exist locally.
	// If set to true, RemoteURL must also be set. Optional.
	MaybeClone bool
	// RemoteURL is the URL of the remote repository to clone from. Required if Clone is not CloneOptionNone.
	RemoteURL string
	// CI is the type of Continuous Integration (CI) environment in which
	// the tool is executing.
	CI string
}

// NewRepository provides access to a git repository based on the provided options.
//
// If opts.Clone is CloneOptionNone, it opens an existing repository at opts.Dir.
// If opts.Clone is CloneOptionMaybe, it opens the repository if it exists,
// otherwise it clones from opts.RemoteURL.
// If opts.Clone is CloneOptionAlways, it always clones from opts.RemoteURL.
func NewRepository(opts *RepositoryOptions) (*Repository, error) {
	if opts.Dir == "" {
		return nil, errors.New("gitrepo: dir is required")
	}

	if !opts.MaybeClone {
		return open(opts.Dir)
	}
	slog.Info(fmt.Sprintf("Checking for repository at %q", opts.Dir))
	_, err := os.Stat(opts.Dir)
	if err == nil {
		return open(opts.Dir)
	}
	if os.IsNotExist(err) {
		if opts.RemoteURL == "" {
			return nil, fmt.Errorf("gitrepo: remote URL is required when cloning")
		}
		slog.Info("Repository not found, executing clone")
		return clone(opts.Dir, opts.RemoteURL, opts.CI)
	}
	return nil, fmt.Errorf("failed to check for repository at %q: %w", opts.Dir, err)
}

func open(dir string) (*Repository, error) {
	slog.Info(fmt.Sprintf("Opening repository at %q", dir))
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return nil, err
	}
	return &Repository{
		Dir:  dir,
		repo: repo,
	}, nil
}

func clone(dir, url, ci string) (*Repository, error) {
	slog.Info(fmt.Sprintf("Cloning repository from %q to %q", url, dir))
	options := &git.CloneOptions{
		URL:           url,
		ReferenceName: plumbing.HEAD,
		SingleBranch:  true,
		Tags:          git.AllTags,
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
	return &Repository{
		Dir:  dir,
		repo: repo,
	}, nil
}

// AddAll adds all pending changes from the working tree to the index,
// so that the changes can later be committed.
func (r *Repository) AddAll() (git.Status, error) {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return git.Status{}, err
	}
	err = worktree.AddWithOptions(&git.AddOptions{All: true})
	if err != nil {
		return git.Status{}, err
	}
	return worktree.Status()
}

// Commit creates a new commit with the provided message and author
// information.
func (r *Repository) Commit(msg string, userName, userEmail string) error {
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
	hash, err := worktree.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  userName,
			Email: userEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}

	// Log commit hash (brief) and subject line (first line of commit)
	subject := strings.Split(msg, "\n")[0]
	slog.Info(fmt.Sprintf("Committed %s: '%s'", hash.String()[0:7], subject))
	return nil
}

// HeadHash returns the hash of the head commit in the repository's current branch.
func (r *Repository) HeadHash() (string, error) {
	headRef, err := r.repo.Head()
	if err != nil {
		return "", err
	}
	return headRef.Hash().String(), nil
}

// IsClean reports whether the working tree has no uncommitted changes.
func (r *Repository) IsClean() (bool, error) {
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

// PrintStatus prints the status of the working tree of the repository, similar
// to running "git status" from the command line.
func (r *Repository) PrintStatus() error {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return err
	}

	status, err := worktree.Status()
	if err != nil {
		return err
	}

	if status.IsClean() {
		slog.Info("git status: No modifications found.")
		return nil
	}

	var staged []string
	for path, file := range status {
		switch file.Staging {
		case git.Added:
			staged = append(staged, fmt.Sprintf("  A %s", path))
		case git.Modified:
			staged = append(staged, fmt.Sprintf("  M %s", path))
		case git.Deleted:
			staged = append(staged, fmt.Sprintf("  D %s", path))
		}
	}
	if len(staged) > 0 {
		slog.Info(fmt.Sprintf("git status: Staged Changes\n%s", strings.Join(staged, "\n")))
	}

	var notStaged []string
	for path, file := range status {
		switch file.Worktree {
		case git.Untracked:
			notStaged = append(notStaged, fmt.Sprintf("  ? %s", path))
		case git.Modified:
			notStaged = append(notStaged, fmt.Sprintf("  M %s", path))
		case git.Deleted:
			notStaged = append(notStaged, fmt.Sprintf("  D %s", path))
		}
	}
	if len(notStaged) > 0 {
		slog.Info(fmt.Sprintf("git status: Unstaged Changes\n%s", strings.Join(notStaged, "\n")))
	}

	return nil
}

// GetCommitsForPathsSinceCommit returns the commits affecting any of the given
// paths, stopping looking at the given commit (which is not included in the
// results).
// The returned commits are ordered such that the most recent commit is first.
//
// If sinceCommit is not provided, all commits are searched; otherwise, if no
// commit matching sinceCommit is found, an error is returned.
func (r *Repository) GetCommitsForPathsSinceCommit(paths []string, sinceCommit string) ([]*Commit, error) {
	if len(paths) == 0 {
		return nil, errors.New("no paths to check for commits")
	}
	commits := []*Commit{}
	finalHash := plumbing.NewHash(sinceCommit)
	logOptions := git.LogOptions{Order: git.LogOrderCommitterTime}
	logIterator, err := r.repo.Log(&logOptions)
	if err != nil {
		return nil, err
	}
	// Sentinel "error" - this can be replaced using LogOptions.To when that's available.
	var ErrStopIterating = fmt.Errorf("fake error to stop iterating")
	err = logIterator.ForEach(func(commit *object.Commit) error {
		if commit.Hash == finalHash {
			return ErrStopIterating
		}

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
			currentPathHash, err := getHashForPathOrEmpty(commit, candidatePath)
			if err != nil {
				return err
			}
			parentPathHash, err := getHashForPathOrEmpty(parentCommit, candidatePath)
			if err != nil {
				return err
			}
			// If we've found a change (including a path being added or removed),
			// add it to our list of commits and proceed to the next commit.
			if currentPathHash != parentPathHash {

				commits = append(commits, &Commit{
					Hash:    commit.Hash,
					Message: commit.Message,
				})
				return nil
			}
		}

		return nil
	})
	if err != nil && err != ErrStopIterating {
		return nil, err
	}
	if sinceCommit != "" && err != ErrStopIterating {
		return nil, fmt.Errorf("did not find commit %s when iterating", sinceCommit)
	}
	return commits, nil
}

// getHashForPathOrEmpty returns the hash for a path at a given commit, or an
// empty string if the path (file or directory) did not exist.
func getHashForPathOrEmpty(commit *object.Commit, path string) (string, error) {
	tree, err := commit.Tree()
	if err != nil {
		return "", err
	}
	treeEntry, err := tree.FindEntry(path)
	if err == object.ErrEntryNotFound || err == object.ErrDirectoryNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return treeEntry.Hash.String(), nil
}

// GetCommitsForPathsSinceTag returns all commits since tagName that contains
// files in paths.
//
// If tagName is empty, all commits for the given paths are returned.
func (r *Repository) GetCommitsForPathsSinceTag(paths []string, tagName string) ([]*Commit, error) {
	var hash string
	if tagName == "" {
		hash = ""
	} else {
		tagRef, err := r.repo.Tag(tagName)
		if err != nil {
			return nil, fmt.Errorf("failed to find tag %s: %w", tagName, err)
		}

		tagCommit, err := r.repo.CommitObject(tagRef.Hash())
		if err != nil {
			return nil, fmt.Errorf("failed to get commit object for tag %s: %w", tagName, err)
		}
		hash = tagCommit.Hash.String()
	}
	return r.GetCommitsForPathsSinceCommit(paths, hash)
}

// GetCommitsForReleaseID returns all commits with the given release ID,
// i.e. where the commit message contains a line of `Librarian-Release-Id: <release-id>`.
// These commits are expected to be contiguous, from head,
// with all commits having a single parent.
func (r *Repository) GetCommitsForReleaseID(releaseID string) ([]*Commit, error) {
	releaseIDLine := fmt.Sprintf("Librarian-Release-ID: %s", releaseID)
	commits := []*Commit{}

	headRef, err := r.repo.Head()
	if err != nil {
		return nil, err
	}
	headCommit, err := r.repo.CommitObject(headRef.Hash())
	if err != nil {
		return nil, err
	}

	// Iterate from the head via parents:
	// - First until we find a commit that *does* have our expected release ID line
	// - Then until we find a commit that *doesn't* have our expected line
	// This way we don't require that the repo HEAD is the merged release PR.
	candidateCommit := headCommit
	for {
		messageLines := strings.Split(candidateCommit.Message, "\n")
		gotReleaseID := slices.Contains(messageLines, releaseIDLine)
		// If we now don't have the release ID, but we did before, we're done.
		if !gotReleaseID && len(commits) > 0 {
			break
		}

		if gotReleaseID {
			commits = append(commits, &Commit{
				Hash:    candidateCommit.Hash,
				Message: candidateCommit.Message,
			})
		}

		if candidateCommit.NumParents() != 1 {
			return nil, fmt.Errorf("aborted finding release PR commits; commit %s has multiple parents", candidateCommit.Hash.String())
		}
		candidateCommit, err = candidateCommit.Parent(0)
		if err != nil {
			return nil, err
		}
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("did not find any commits with release ID %s", releaseID)
	}
	// Present the commits in forward-chronological order.
	slices.Reverse(commits)
	return commits, nil
}

// PushBranch creates a branch with the given name in the default remote.
func (r *Repository) PushBranch(remoteBranch string, accessToken string) error {
	headRef, err := r.repo.Head()
	if err != nil {
		return err
	}
	auth := http.BasicAuth{
		Username: "Ignored",
		Password: accessToken,
	}
	refFrom := headRef.Name().String()
	refTo := fmt.Sprintf("refs/heads/%s", remoteBranch)
	refSpec := config.RefSpec(fmt.Sprintf("%s:%s", refFrom, refTo))
	pushOptions := git.PushOptions{
		RefSpecs: []config.RefSpec{refSpec},
		Auth:     &auth,
	}

	slog.Info(fmt.Sprintf("Pushing to branch %s", remoteBranch))
	return r.repo.Push(&pushOptions)
}

// CleanWorkingTree Drops any local changes NOT committed, but keeps any local commits
func (r *Repository) CleanWorkingTree() error {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return err
	}
	if err = worktree.Reset(&git.ResetOptions{Mode: git.HardReset}); err != nil {
		return err
	}
	return worktree.Clean(&git.CleanOptions{Dir: true})
}

// CleanAndRevertHeadCommit drops any local changes, and also reset to the
// parent of the current head commit.  This is a special case of
// CleanAndRevertCommits where the count is 1.
func (r *Repository) CleanAndRevertHeadCommit() error {
	return r.CleanAndRevertCommits(1)
}

// CleanAndRevertCommits the specified number of commits in the repository,
// by counting commits backwards from the head and then doing a hard reset
// for the target commit, then a clean.
func (r *Repository) CleanAndRevertCommits(count int) error {
	headRef, err := r.repo.Head()
	if err != nil {
		return err
	}
	headCommit, err := r.repo.CommitObject(headRef.Hash())
	if err != nil {
		return err
	}
	targetCommit := headCommit
	for range count {
		if targetCommit.NumParents() != 1 {
			return fmt.Errorf("commit %s has multiple parents", targetCommit.Hash.String())
		}
		var err error
		targetCommit, err = targetCommit.Parent(0)
		if err != nil {
			return err
		}
	}
	worktree, err := r.repo.Worktree()
	if err != nil {
		return err
	}
	if err = worktree.Reset(&git.ResetOptions{Mode: git.HardReset, Commit: targetCommit.Hash}); err != nil {
		return err
	}
	return worktree.Clean(&git.CleanOptions{Dir: true})
}

// Checkout checks out the given commit in the repository.
func (r *Repository) Checkout(commit string) error {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return err
	}
	hash := plumbing.NewHash(commit)
	checkoutOptions := git.CheckoutOptions{
		Hash: hash,
	}
	return worktree.Checkout(&checkoutOptions)
}

// Remotes returns the remotes within the repository.
func (r *Repository) Remotes() ([]*git.Remote, error) {
	return r.repo.Remotes()
}
