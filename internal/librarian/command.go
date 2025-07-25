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
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

func cloneOrOpenLanguageRepo(workRoot, repo, ci string) (*gitrepo.Repository, error) {
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
	languageRepo, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{
		Dir: absRepoRoot,
		CI:  ci,
	})
	if err != nil {
		return nil, err
	}
	clean, err := languageRepo.IsClean()
	if err != nil {
		return nil, err
	}
	if !clean {
		return nil, errors.New("language repo must be clean")
	}
	return languageRepo, nil
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

func createWorkRoot(t time.Time, workRootOverride string) (string, error) {
	if workRootOverride != "" {
		slog.Info("Using specified working directory", "dir", workRootOverride)
		return workRootOverride, nil
	}

	path := filepath.Join(os.TempDir(), fmt.Sprintf("librarian-%s", formatTimestamp(t)))

	_, err := os.Stat(path)
	switch {
	case os.IsNotExist(err):
		if err := os.Mkdir(path, 0755); err != nil {
			return "", fmt.Errorf("unable to create temporary working directory '%s': %w", path, err)
		}
	case err == nil:
		return "", fmt.Errorf("temporary working directory already exists: %s", path)
	default:
		return "", fmt.Errorf("unable to check directory '%s': %w", path, err)
	}

	slog.Info("Temporary working directory", "dir", path)
	return path, nil
}

// commitAndPush creates a commit and push request to Github for the generated changes.
// It uses the GitHub client to create a PR with the specified branch, title, and description to the repository.
func commitAndPush(ctx context.Context, repo *gitrepo.Repository, ghClient GitHubClient, pushConfig string) error {
	if pushConfig == "" {
		slog.Info("PushConfig flag not specified, skipping")
		return nil
	}
	// Ensure we have a GitHub repository
	gitHubRepo, err := github.FetchGitHubRepoFromRemote(repo)
	if err != nil {
		return err
	}

	userEmail, userName, err := parsePushConfig(pushConfig)
	if err != nil {
		return err
	}
	if _, err = repo.AddAll(); err != nil {
		return err
	}

	// TODO: get correct language for message (https://github.com/googleapis/librarian/issues/885)
	message := "Changes in this PR"
	repo.Commit(message, userName, userEmail)

	// Create a new branch, set title and message for the PR.
	datetimeNow := formatTimestamp(time.Now())
	titlePrefix := "Librarian pull request"
	branch := fmt.Sprintf("librarian-%s", datetimeNow)
	title := fmt.Sprintf("%s: %s", titlePrefix, datetimeNow)

	_, err = ghClient.CreatePullRequest(ctx, gitHubRepo, branch, title, message)
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}
	return nil
}

func parsePushConfig(pushConfig string) (string, string, error) {
	parts := strings.Split(pushConfig, ",")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid pushConfig format: expected 'email,user', got %q", pushConfig)
	}
	userEmail := parts[0]
	userName := parts[1]
	return userEmail, userName, nil
}
