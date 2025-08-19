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
	"log/slog"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/docker"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

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

func newCommandRunner(cfg *config.Config) (*commandRunner, error) {
	if cfg.APISource == "" {
		cfg.APISource = "https://github.com/googleapis/googleapis"
	}
	languageRepo, err := cloneOrOpenRepo(cfg.WorkRoot, cfg.Repo, cfg.CI)
	if err != nil {
		return nil, err
	}

	var sourceRepo gitrepo.Repository
	var sourceRepoDir string
	if cfg.CommandName != tagAndReleaseCmdName {
		sourceRepo, err = cloneOrOpenRepo(cfg.WorkRoot, cfg.APISource, cfg.CI)
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
		gitRepo, err = github.ParseURL(cfg.Repo)
		if err != nil {
			return nil, fmt.Errorf("failed to parse repo url: %w", err)
		}
	} else {
		gitRepo, err = github.FetchGitHubRepoFromRemote(languageRepo)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitHub repo from remote: %w", err)
		}
	}
	ghClient, err := github.NewClient(cfg.GitHubToken, gitRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	container, err := docker.New(cfg.WorkRoot, image, cfg.UserUID, cfg.UserGID)
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

func cloneOrOpenRepo(workRoot, repo, ci string) (*gitrepo.LocalRepository, error) {
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
			Dir:        repoPath,
			MaybeClone: true,
			RemoteURL:  repo,
			CI:         ci,
		})
	}
	// repo is a directory
	absRepoRoot, err := filepath.Abs(repo)
	if err != nil {
		return nil, err
	}
	githubRepo, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{
		Dir: absRepoRoot,
		CI:  ci,
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

// commitAndPush creates a commit and push request to GitHub for the generated
// changes.
// It uses the GitHub client to create a PR with the specified branch, title, and
// description to the repository.
func commitAndPush(ctx context.Context, r *generateRunner, commitMessage string) error {
	if !r.cfg.Push {
		slog.Info("Push flag is not specified, skipping")
		return nil
	}
	// Ensure we have a GitHub repository
	gitHubRepo, err := github.FetchGitHubRepoFromRemote(r.repo)
	if err != nil {
		return err
	}

	status, err := r.repo.AddAll()
	if err != nil {
		return err
	}
	if status.IsClean() {
		slog.Info("No changes to commit, skipping commit and push.")
		return nil
	}

	// TODO: get correct language for message (https://github.com/googleapis/librarian/issues/885)
	if err := r.repo.Commit(commitMessage); err != nil {
		return err
	}

	// Create a new branch, set title and message for the PR.
	datetimeNow := formatTimestamp(time.Now())
	titlePrefix := "Librarian pull request"
	branch := fmt.Sprintf("librarian-%s", datetimeNow)
	title := fmt.Sprintf("%s: %s", titlePrefix, datetimeNow)

	if _, err = r.ghClient.CreatePullRequest(ctx, gitHubRepo, branch, title, commitMessage); err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}
	return nil
}
