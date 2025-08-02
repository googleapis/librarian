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
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Repository defines the interface for git repository operations.
type Repository interface {
	AddAll() (git.Status, error)
	Commit(msg string) error
	IsClean() (bool, error)
	Remotes() ([]*git.Remote, error)
	GetDir() string
}

// LocalRepository represents a git repository.
type LocalRepository struct {
	Dir  string
	repo *git.Repository
}

// Commit represents a git commit.
type Commit struct {
	Hash    plumbing.Hash
	Message string
}

// RepositoryOptions are used to configure a [LocalRepository].
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
func NewRepository(opts *RepositoryOptions) (*LocalRepository, error) {
	if opts.Dir == "" {
		return nil, errors.New("gitrepo: dir is required")
	}

	if !opts.MaybeClone {
		return open(opts.Dir)
	}
	slog.Info("Checking for repository", "dir", opts.Dir)
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

func open(dir string) (*LocalRepository, error) {
	slog.Info("Opening repository", "dir", dir)
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return nil, err
	}
	return &LocalRepository{
		Dir:  dir,
		repo: repo,
	}, nil
}

func clone(dir, url, ci string) (*LocalRepository, error) {
	slog.Info("Cloning repository", "url", url, "dir", dir)
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
	return &LocalRepository{
		Dir:  dir,
		repo: repo,
	}, nil
}

// AddAll adds all pending changes from the working tree to the index,
// so that the changes can later be committed.
func (r *LocalRepository) AddAll() (git.Status, error) {
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
func (r *LocalRepository) Commit(msg string) error {
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

// Remotes returns the remotes within the repository.
func (r *LocalRepository) Remotes() ([]*git.Remote, error) {
	return r.repo.Remotes()
}

// GetDir returns the directory of the repository.
func (r *LocalRepository) GetDir() string {
	return r.Dir
}
