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
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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
		Depth:         1,
		Tags:          git.NoTags,
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
