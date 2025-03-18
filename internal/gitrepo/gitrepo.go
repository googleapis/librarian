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
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v69/github"
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
func CloneOrOpen(ctx context.Context, dirpath, repoURL string) (*Repo, error) {
	slog.Info(fmt.Sprintf("Cloning %q to %q", repoURL, dirpath))

	_, err := os.Stat(dirpath)
	if err == nil {
		return Open(ctx, dirpath)
	}
	if os.IsNotExist(err) {
		return Clone(ctx, dirpath, repoURL)
	}
	return nil, err
}

// Clone downloads a copy of a Git repository from repoURL and saves it to the
// specified directory at dirpath.
func Clone(ctx context.Context, dirpath, repoURL string) (*Repo, error) {
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
func Open(ctx context.Context, dirpath string) (*Repo, error) {
	repo, err := git.PlainOpen(dirpath)
	if err != nil {
		return nil, err
	}
	return &Repo{
		Dir:  dirpath,
		repo: repo,
	}, nil
}

func AddAll(ctx context.Context, repo *Repo) (git.Status, error) {
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
func Commit(ctx context.Context, repo *Repo, msg string) error {
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
	commit, err := worktree.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Google Cloud SDK",
			Email: "noreply-cloudsdk@google.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}

	// Log commit object, if enabled
	if slog.Default().Enabled(ctx, slog.LevelInfo.Level()) {
		obj, err := repo.repo.CommitObject(commit)
		if err != nil {
			return err
		}
		slog.Info(fmt.Sprint(obj))
	}
	return nil
}

func HeadHash(ctx context.Context, repo *Repo) (string, error) {
	headRef, err := repo.repo.Head()
	if err != nil {
		return "", err
	}
	return headRef.String(), nil
}

func IsClean(ctx context.Context, repo *Repo) (bool, error) {
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

func ResetHard(ctx context.Context, repo *Repo) error {
	worktree, err := repo.repo.Worktree()
	if err != nil {
		return err
	}
	return worktree.Reset(&git.ResetOptions{Mode: git.HardReset})
}

func PrintStatus(ctx context.Context, repo *Repo) error {
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

// Returns the commits affecting the given path,
// stopping looking at the given commit (which is not included in the results).
// The returned commits are ordered such that the most recent commit is first.
func GetCommitsForPath(repo *Repo, path string, commit string, retrieveAfterTimestamp *time.Time) ([]object.Commit, error) {
	commits := []object.Commit{}
	finalHash := plumbing.NewHash(commit)
	logOptions := git.LogOptions{Order: git.LogOrderCommitterTime}
	if retrieveAfterTimestamp != nil {
		logOptions.Since = retrieveAfterTimestamp
	}
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

		// We perform filtering by finding out if the tree hash for the given
		// path at the commit we're looking at is the same as the tree hash
		// for the commit's parent. This is much, much faster than any other filtering
		// option, it seems.
		parentCommit, err := commit.Parent(0)
		if err != nil {
			return err
		}
		currentTree, err := commit.Tree()
		if err != nil {
			return err
		}
		currentPathEntry, err := currentTree.FindEntry(path)
		if err != nil {
			return err
		}
		parentTree, err := parentCommit.Tree()
		if err != nil {
			return err
		}
		parentPathEntry, err := parentTree.FindEntry(path)
		if err != nil {
			return err
		}

		// If we've found a change, add it to our list of commits.
		if currentPathEntry.Hash != parentPathEntry.Hash {
			commits = append(commits, *commit)
		}

		return nil
	})
	if err != nil && err != ErrStopIterating {
		return nil, err
	}
	return commits, nil
}

// Returns all commits since tagName that contains files in path
func GetCommitsSinceTagForPath(repo *Repo, path, tagName string) ([]object.Commit, error) {
	tagRef, err := repo.repo.Tag(tagName)

	if err != nil {
		return nil, fmt.Errorf("failed to find tag %s: %w", tagName, err)
	}

	tagCommit, err := repo.repo.CommitObject(tagRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit object for tag %s: %w", tagName, err)
	}

	return GetCommitsForPath(repo, path, tagCommit.Hash.String(), &tagCommit.Committer.When)
}

// Creates a branch with the given name in the default remote.
func PushBranch(ctx context.Context, repo *Repo, remoteBranch string, accessToken string) error {
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

// Creates a pull request in the remote repo. At the moment this requires a single remote to be
// configured, which must have a GitHub HTTPS URL. We assume a base branch of "main".
func CreatePullRequest(ctx context.Context, repo *Repo, remoteBranch string, accessToken string, title string, body string) error {
	remotes, err := repo.repo.Remotes()
	if err != nil {
		return err
	}

	if len(remotes) != 1 {
		return fmt.Errorf("can only create a PR with a single remote; number of remotes: %d", len(remotes))
	}

	remoteUrl := remotes[0].Config().URLs[0]
	if !strings.HasPrefix(remoteUrl, "https://github.com/") {
		return fmt.Errorf("remote '%s' is not a GitHub remote", remoteUrl)
	}
	remotePath := remoteUrl[len("https://github.com/"):]
	pathParts := strings.Split(remotePath, "/")
	organization := pathParts[0]
	repoName := pathParts[1]
	repoName = strings.TrimSuffix(repoName, ".git")

	if body == "" {
		body = "Regenerated all changed APIs. See individual commits for details."
	}
	gitHubClient := github.NewClient(nil).WithAuthToken(accessToken)
	newPR := &github.NewPullRequest{
		Title:               &title,
		Head:                &remoteBranch,
		Base:                github.Ptr("main"),
		Body:                github.Ptr(body),
		MaintainerCanModify: github.Ptr(true),
	}

	pr, _, err := gitHubClient.PullRequests.Create(ctx, organization, repoName, newPR)
	if err != nil {
		return err
	}

	fmt.Printf("PR created: %s\n", pr.GetHTMLURL())
	return nil
}

func Checkout(ctx context.Context, repo *Repo, commit string) error {
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
