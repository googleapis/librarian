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
	"github.com/googleapis/librarian/internal/githubrepo"
)

// Repo represents a git repository.
type Repo struct {
	Dir  string
	repo *git.Repository
}

// CloneOrOpen provides access to a Git repository.
//
// If a repository already exists at the specified directory path (dirpath),
// it opens and provides access to that repository.
//
// Otherwise, it clones the repository from the given URL (repoURL) and saves it
// to the specified directory path (dirpath).
func CloneOrOpen(dirpath, repoURL string) (*Repo, error) {
	slog.Info(fmt.Sprintf("Cloning %q to %q", repoURL, dirpath))

	_, err := os.Stat(dirpath)
	if err == nil {
		return Open(dirpath)
	}
	if os.IsNotExist(err) {
		return Clone(dirpath, repoURL)
	}
	return nil, err
}

// Clone downloads a copy of a Git repository from repoURL and saves it to the
// specified directory at dirpath.
func Clone(dirpath, repoURL string) (*Repo, error) {
	options := &git.CloneOptions{
		URL:           repoURL,
		ReferenceName: plumbing.HEAD,
		SingleBranch:  true,
		Tags:          git.AllTags,
		// .NET uses submodules for conformance tests.
		// (There may be other examples too.)
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	}
	if ci := os.Getenv("CI"); ci == "" {
		options.Progress = os.Stdout // When not a CI build, output progress.
	}

	repo, err := git.PlainClone(dirpath, false, options)
	if err != nil {
		return nil, err
	}
	return &Repo{
		Dir:  dirpath,
		repo: repo,
	}, nil
}

// Open provides access to a Git repository that exists at dirpath.
func Open(dirpath string) (*Repo, error) {
	repo, err := git.PlainOpen(dirpath)
	if err != nil {
		return nil, err
	}
	return &Repo{
		Dir:  dirpath,
		repo: repo,
	}, nil
}

func AddAll(repo *Repo) (git.Status, error) {
	worktree, err := repo.repo.Worktree()
	if err != nil {
		return git.Status{}, err
	}
	err = worktree.AddWithOptions(&git.AddOptions{All: true})
	if err != nil {
		return git.Status{}, err
	}
	return worktree.Status()
}

// returns an error if there is nothing to commit
func Commit(repo *Repo, msg string, userName, userEmail string) error {
	worktree, err := repo.repo.Worktree()
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
	if userName == "" {
		userName = "Google Cloud SDK"
	}
	if userEmail == "" {
		userEmail = "noreply-cloudsdk@google.com"
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

func HeadHash(repo *Repo) (string, error) {
	headRef, err := repo.repo.Head()
	if err != nil {
		return "", err
	}
	return headRef.Hash().String(), nil
}

func IsClean(repo *Repo) (bool, error) {
	worktree, err := repo.repo.Worktree()
	if err != nil {
		return false, err
	}
	status, err := worktree.Status()
	if err != nil {
		return false, err
	}

	return status.IsClean(), nil
}

func PrintStatus(repo *Repo) error {
	worktree, err := repo.repo.Worktree()
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

// Returns the commits affecting any of the given paths, stopping looking at the
// given commit (which is not included in the results).
// The returned commits are ordered such that the most recent commit is first.
// If sinceCommit is not provided, all commits are searched. Otherwise, if no commit
// matching sinceCommit is found, an error is returned.
func GetCommitsForPathsSinceCommit(repo *Repo, paths []string, sinceCommit string) ([]object.Commit, error) {
	if len(paths) == 0 {
		return nil, errors.New("no paths to check for commits")
	}
	commits := []object.Commit{}
	finalHash := plumbing.NewHash(sinceCommit)
	logOptions := git.LogOptions{Order: git.LogOrderCommitterTime}
	logIterator, err := repo.repo.Log(&logOptions)
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
		// for the commit's parent. This is much, much faster than any other filtering
		// option, it seems. In theory we should be able to remember our "current"
		// commit for each path, but that's likely to be significantly more complex.
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
				commits = append(commits, *commit)
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

// Returns the hash for a path at a given commit, or an empty string if the path
// (file or directory) did not exist.
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

// Returns all commits since tagName that contains files in path.
// If tagName is empty, all commits for the given paths are returned.
func GetCommitsForPathsSinceTag(repo *Repo, paths []string, tagName string) ([]object.Commit, error) {
	var hash string
	if tagName == "" {
		hash = ""
	} else {
		tagRef, err := repo.repo.Tag(tagName)
		if err != nil {
			return nil, fmt.Errorf("failed to find tag %s: %w", tagName, err)
		}

		tagCommit, err := repo.repo.CommitObject(tagRef.Hash())
		if err != nil {
			return nil, fmt.Errorf("failed to get commit object for tag %s: %w", tagName, err)
		}
		hash = tagCommit.Hash.String()
	}
	return GetCommitsForPathsSinceCommit(repo, paths, hash)
}

// Returns all commits with the given release ID, i.e. where the commit message contains a line of
// Librarian-Release-Id: <release-id>. These commits are expected to be contiguous, from head,
// with all commits having a single parent.
func GetCommitsForReleaseID(repo *Repo, releaseID string) ([]object.Commit, error) {
	releaseIDLine := fmt.Sprintf("Librarian-Release-ID: %s", releaseID)
	commits := []object.Commit{}

	headRef, err := repo.repo.Head()
	if err != nil {
		return nil, err
	}
	headCommit, err := repo.repo.CommitObject(headRef.Hash())
	if err != nil {
		return nil, err
	}

	// Iterate from the head via parents, until we find a commit that doesn't
	// have our expected line in the message.
	candidateCommit := headCommit
	for {
		messageLines := strings.Split(candidateCommit.Message, "\n")
		gotReleaseID := slices.Contains(messageLines, releaseIDLine)
		if !gotReleaseID {
			break
		}

		commits = append(commits, *candidateCommit)

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

// Creates a branch with the given name in the default remote.
func PushBranch(repo *Repo, remoteBranch string, accessToken string) error {
	headRef, err := repo.repo.Head()
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
	return repo.repo.Push(&pushOptions)
}

// CleanWorkingTree Drops any local changes NOT committed, but keeps any local commits
func CleanWorkingTree(repo *Repo) error {
	worktree, err := repo.repo.Worktree()
	if err != nil {
		return err
	}
	if err = worktree.Reset(&git.ResetOptions{Mode: git.HardReset}); err != nil {
		return err
	}
	return worktree.Clean(&git.CleanOptions{Dir: true})
}

// Drop any local changes, and also reset to the parent of the current head commit
func CleanAndRevertHeadCommit(repo *Repo) error {
	headRef, err := repo.repo.Head()
	if err != nil {
		return err
	}
	headCommit, err := repo.repo.CommitObject(headRef.Hash())
	if err != nil {
		return err
	}
	if headCommit.NumParents() != 1 {
		return errors.New("head commit has multiple parents")
	}
	parentCommit, err := headCommit.Parent(0)
	if err != nil {
		return err
	}
	worktree, err := repo.repo.Worktree()
	if err != nil {
		return err
	}
	if err = worktree.Reset(&git.ResetOptions{Mode: git.HardReset, Commit: parentCommit.Hash}); err != nil {
		return err
	}
	return worktree.Clean(&git.CleanOptions{Dir: true})
}

func Checkout(repo *Repo, commit string) error {
	worktree, err := repo.repo.Worktree()
	if err != nil {
		return err
	}
	hash := plumbing.NewHash(commit)
	checkoutOptions := git.CheckoutOptions{
		Hash: hash,
	}
	return worktree.Checkout(&checkoutOptions)
}

// Parses the GitHub repo name from the remote for this repository.
// There must only be a single remote with a GitHub URL (as the first URL), in order to provide an
// unambiguous result.
// Remotes without any URLs, or where the first URL does not start with https://github.com/ are ignored.
func GetGitHubRepoFromRemote(repo *Repo) (githubrepo.GitHubRepo, error) {
	remotes, err := repo.repo.Remotes()
	if err != nil {
		return githubrepo.GitHubRepo{}, err
	}
	gitHubRemoteNames := []string{}
	gitHubUrl := ""
	for _, remote := range remotes {
		urls := remote.Config().URLs
		if len(urls) > 0 && strings.HasPrefix(urls[0], "https://github.com/") {
			gitHubRemoteNames = append(gitHubRemoteNames, remote.Config().Name)
			gitHubUrl = urls[0]
		}
	}

	if len(gitHubRemoteNames) == 0 {
		return githubrepo.GitHubRepo{}, fmt.Errorf("no GitHub remotes found")
	}

	if len(gitHubRemoteNames) > 1 {
		joinedRemoteNames := strings.Join(gitHubRemoteNames, ", ")
		return githubrepo.GitHubRepo{}, fmt.Errorf("can only determine the GitHub repo with a single matching remote; GitHub remotes in repo: %s", joinedRemoteNames)
	}
	return githubrepo.ParseUrl(gitHubUrl)
}
