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
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
)

const releaseIDEnvVarName = "_RELEASE_ID"

// commandState holds all necessary information for a command execution.
type commandState struct {
	// ctx provides context for cancellable operations.
	ctx context.Context

	// startTime records when the command began execution. This is used as a
	// consistent timestamp for commands when necessary.
	startTime time.Time

	// workRoot is the base directory for all command operations. The default
	// location is /tmp.
	workRoot string

	// languageRepo is the relevant language-specific Git repository, if
	// applicable.
	languageRepo *gitrepo.Repository

	// pipelineConfig holds the pipeline configuration, loaded from the
	// language repo if present.
	pipelineConfig *statepb.PipelineConfig

	// pipelineState holds the pipeline's current state, loaded from the
	// language repo if present.
	pipelineState *statepb.PipelineState

	// containerConfig provides settings for running containerized commands.
	containerConfig *docker.Docker
}

func cloneOrOpenLanguageRepo(workRoot string) (*gitrepo.Repository, error) {
	var languageRepo *gitrepo.Repository
	if flagRepoRoot != "" && flagRepoUrl != "" {
		return nil, errors.New("do not specify both repo-root and repo-url")
	}
	if flagRepoUrl != "" {
		// Take the last part of the URL as the directory name. It feels very
		// unlikely that will clash with anything else (e.g. "output")
		bits := strings.Split(flagRepoUrl, "/")
		repoName := bits[len(bits)-1]
		repoPath := filepath.Join(workRoot, repoName)
		return gitrepo.NewRepository(&gitrepo.RepositoryOptions{
			Dir:        repoPath,
			MaybeClone: true,
			RemoteURL:  flagRepoUrl,
		})
	}
	if flagRepoRoot == "" {
		languageRepoURL := fmt.Sprintf("https://github.com/googleapis/google-cloud-%s", flagLanguage)
		repoPath := filepath.Join(workRoot, fmt.Sprintf("google-cloud-%s", flagLanguage))
		return gitrepo.NewRepository(&gitrepo.RepositoryOptions{
			Dir:        repoPath,
			MaybeClone: true,
			RemoteURL:  languageRepoURL,
		})
	}
	repoRoot, err := filepath.Abs(flagRepoRoot)
	if err != nil {
		return nil, err
	}
	languageRepo, err = gitrepo.NewRepository(&gitrepo.RepositoryOptions{
		Dir: repoRoot,
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
func createCommandStateForLanguage(ctx context.Context) (*commandState, error) {
	startTime := time.Now()
	workRoot, err := createWorkRoot(startTime)
	if err != nil {
		return nil, err
	}
	repo, err := cloneOrOpenLanguageRepo(workRoot)
	if err != nil {
		return nil, err
	}

	ps, config, err := loadRepoStateAndConfig(repo)
	if err != nil {
		return nil, err
	}

	image := deriveImage(ps)
	containerConfig, err := docker.NewContainerConfig(ctx, workRoot, image, flagSecretsProject, config)
	if err != nil {
		return nil, err
	}

	state := &commandState{
		ctx:             ctx,
		startTime:       startTime,
		workRoot:        workRoot,
		languageRepo:    repo,
		pipelineConfig:  config,
		pipelineState:   ps,
		containerConfig: containerConfig,
	}
	return state, nil
}

func appendResultEnvironmentVariable(workRoot, name, value string) error {
	envFile := flagEnvFile
	if envFile == "" {
		envFile = filepath.Join(workRoot, "env-vars.txt")
	}

	return appendToFile(envFile, fmt.Sprintf("%s=%s\n", name, value))
}

func deriveImage(state *statepb.PipelineState) string {
	if flagImage != "" {
		return flagImage
	}

	defaultRepository := os.Getenv(defaultRepositoryEnvironmentVariable)
	relativeImage := fmt.Sprintf("google-cloud-%s-generator", flagLanguage)

	var tag string
	if state == nil {
		tag = "latest"
	} else {
		tag = state.ImageTag
	}
	if defaultRepository == "" {
		return fmt.Sprintf("%s:%s", relativeImage, tag)
	} else {
		return fmt.Sprintf("%s/%s:%s", defaultRepository, relativeImage, tag)
	}
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

func createWorkRoot(t time.Time) (string, error) {
	if flagWorkRoot != "" {
		slog.Info(fmt.Sprintf("Using specified working directory: %s", flagWorkRoot))
		return flagWorkRoot, nil
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

	slog.Info(fmt.Sprintf("Temporary working directory: %s", path))
	return path, nil
}

// No commit is made if there are no file modifications.
func commitAll(repo *gitrepo.Repository, msg string) error {
	status, err := repo.AddAll()
	if err != nil {
		return err
	}
	if status.IsClean() {
		slog.Info("No modifications to commit.")
		return nil
	}

	return repo.Commit(msg, flagGitUserName, flagGitUserEmail)
}

// Log details of an error which prevents a single API or library from being configured/released, but without
// halting the overall process. Return a brief description to the errors to include in the PR.
// We don't include detailed errors in the PR, as this could reveal sensitive information.
// The action should describe what failed, e.g. "configuring", "building", "generating".
func logPartialError(id string, err error, action string) string {
	slog.Warn(fmt.Sprintf("Error while %s %s: %s", action, id, err))
	return fmt.Sprintf("Error while %s %s", action, id)
}

var librarianCommands = []*cli.Command{
	CmdConfigure,
	CmdGenerate,
	CmdUpdateApis,
	CmdCreateReleasePR,
	CmdUpdateImageTag,
	CmdMergeReleasePR,
	CmdCreateReleaseArtifacts,
	CmdPublishReleaseArtifacts,
	CmdVersion,
}

func formatReleaseTag(libraryID, version string) string {
	return libraryID + "-" + version
}
