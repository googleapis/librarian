// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package librarianops

import (
	"context"
	"fmt"
	"os"
)

func cleanupForEmptyDir(newWorkingDir string, repoDir string) {
	os.Chdir(newWorkingDir)
	os.RemoveAll(repoDir)
}

func setupEnvironmentForEmptyDir(ctx context.Context, repoName string) (string, func(), error) {
	originalWorkingDir, err := os.Getwd()
	if err != nil {
		return "", nil, fmt.Errorf("getting current working directory: %w", err)
	}

	repoDir, err := os.MkdirTemp("", "librarianops-"+repoName+"-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp directory: %w", err)
	}

	if err := cloneRepoInDir(ctx, repoName, repoDir); err != nil {
		cleanupForEmptyDir(originalWorkingDir, repoDir)
		return "", nil, fmt.Errorf("clone repo in directory %q: %w", repoDir, err)
	}
	if err := os.Chdir(repoDir); err != nil {
		cleanupForEmptyDir(originalWorkingDir, repoDir)
		return "", nil, fmt.Errorf("changing to repo directory %q: %w", repoDir, err)
	}
	return repoDir, func() {
		cleanupForEmptyDir(originalWorkingDir, repoDir)
	}, nil
}

func setupEnvironmentForDir(repoDir string) (string, func(), error) {
	currentWorkingDir, err := os.Getwd()
	if err != nil {
		return "", nil, fmt.Errorf("getting current working directory: %w", err)
	}

	if err := os.Chdir(repoDir); err != nil {
		os.Chdir(currentWorkingDir)
		return "", nil, fmt.Errorf("changing to repo directory %q: %w", repoDir, err)
	}
	return repoDir, func() { os.Chdir(currentWorkingDir) }, nil
}

// setupEnvironment sets up the environment for a librarianops command.
// If repoDir is empty, it creates a temporary directory and clones the repo.
// The returned func() must be called to restore the working directory and
// clean up temporary resources.
func setupEnvironment(ctx context.Context, repoDir string, repoName string) (string, func(), error) {
	if repoDir == "" {
		return setupEnvironmentForEmptyDir(ctx, repoName)
	}
	return setupEnvironmentForDir(repoDir)
}
