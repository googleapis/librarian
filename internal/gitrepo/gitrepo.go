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

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
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
	repo, err := git.PlainClone(dirpath, false, &git.CloneOptions{
		URL:               repoURL,
		Progress:          os.Stdout,
		ReferenceName:     plumbing.HEAD,
		SingleBranch:      true,
		Depth:             1,
		Tags:              git.NoTags,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})
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
