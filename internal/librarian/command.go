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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
)

func cloneOrOpenLanguageRepo(workRoot, repo, ci string) (*gitrepo.Repository, error) {
	if repo == "" {
		return nil, errors.New("repo must be specified")
	}

	if isUrl(repo) {
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

// createCommandStateForLanguage performs common (but not universal)
// steps for initializing a language repo, obtaining the pipeline state/config
// from it, deriving the container image to use, and creating an appropriate
// ContainerState based on all of the above. This should be used by all commands
// which always have a language repo. Commands which only conditionally use
// language repos should construct the command state themselves.
func createCommandStateForLanguage(workRootOverride, repo, imageOverride, project, ci, uid, gid string) (
	startTime time.Time,
	workRoot string,
	languageRepo *gitrepo.Repository,
	pipelineConfig *statepb.PipelineConfig,
	pipelineState *statepb.PipelineState,
	containerConfig *docker.Docker,
	err error,
) {
	startTime = time.Now()
	workRoot, err = createWorkRoot(startTime, workRootOverride)
	if err != nil {
		return
	}
	languageRepo, err = cloneOrOpenLanguageRepo(workRoot, repo, ci)
	if err != nil {
		return
	}

	pipelineState, pipelineConfig, err = loadRepoStateAndConfig(languageRepo)
	if err != nil {
		return
	}

	image, err := deriveImage(imageOverride, pipelineState)
	if err != nil {
		return
	}
	containerConfig, err = docker.New(workRoot, image, project, uid, gid, pipelineConfig)
	if err != nil {
		return
	}

	return startTime, workRoot, languageRepo, pipelineConfig, pipelineState, containerConfig, nil
}

func appendResultEnvironmentVariable(workRoot, name, value, envFileOverride string) error {
	envFile := envFileOverride
	if envFile == "" {
		envFile = filepath.Join(workRoot, "env-vars.txt")
	}

	return appendToFile(envFile, fmt.Sprintf("%s=%s\n", name, value))
}

func deriveImage(imageOverride string, state *statepb.PipelineState) (string, error) {
	if imageOverride != "" {
		return imageOverride, nil
	}
	if state == nil {
		return "", nil
	}
	// TODO(https://github.com/googleapis/librarian/issues/326):
	// use image from state.yaml when switch to this config file. see go/librarian:cli-reimagined
	if state.ImageTag == "" {
		return "", errors.New("pipeline state does not have image specified and no override was provided")
	}
	return state.ImageTag, nil
}

// Finds a library which includes code generated from the given API path.
// If there are no such libraries, an empty string is returned.
// If there are multiple such libraries, the first match is returned.
func findLibraryIDByApiPath(state *statepb.PipelineState, apiPath string) string {
	for _, library := range state.Libraries {
		if slices.Contains(library.ApiPaths, apiPath) {
			return library.Id
		}
	}
	return ""
}

func findLibraryByID(state *statepb.PipelineState, libraryID string) *statepb.LibraryState {
	for _, library := range state.Libraries {
		if library.Id == libraryID {
			return library
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

// No commit is made if there are no file modifications.
func commitAll(repo *gitrepo.Repository, msg, pushConfig string) error {
	userEmail, userName, err := parsePushConfig(pushConfig)
	if err != nil {
		return err
	}

	status, err := repo.AddAll()
	if err != nil {
		return err
	}
	if status.IsClean() {
		slog.Info("No modifications to commit.")
		return nil
	}

	return repo.Commit(msg, userName, userEmail)
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

func formatReleaseTag(libraryID, version string) string {
	return libraryID + "-" + version
}
