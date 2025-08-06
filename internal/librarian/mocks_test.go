// Copyright 2025 Google LLC
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
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

// mockGitHubClient is a mock implementation of the GitHubClient interface for testing.
type mockGitHubClient struct {
	GitHubClient
	rawContent             []byte
	rawErr                 error
	createPullRequestCalls int
	createPullRequestErr   error
	createdPR              *github.PullRequestMetadata
}

func (m *mockGitHubClient) GetRawContent(ctx context.Context, path, ref string) ([]byte, error) {
	return m.rawContent, m.rawErr
}

func (m *mockGitHubClient) CreatePullRequest(ctx context.Context, repo *github.Repository, remoteBranch, title, body string) (*github.PullRequestMetadata, error) {
	m.createPullRequestCalls++
	if m.createPullRequestErr != nil {
		return nil, m.createPullRequestErr
	}
	return m.createdPR, nil
}

// mockContainerClient is a mock implementation of the ContainerClient interface for testing.
type mockContainerClient struct {
	ContainerClient
	generateCalls       int
	buildCalls          int
	configureCalls      int
	generateErr         error
	buildErr            error
	configureErr        error
	failGenerateForID   string
	requestLibraryID    string
	noBuildResponse     bool
	noConfigureResponse bool
	wantErrorMsg        bool
}

func (m *mockContainerClient) Generate(ctx context.Context, request *docker.GenerateRequest) error {
	m.generateCalls++
	// Write a generate-response.json because it is required by generate
	// command.
	if err := os.MkdirAll(filepath.Join(request.RepoDir, config.LibrarianDir), 0755); err != nil {
		return err
	}

	libraryStr := "{}"
	if m.wantErrorMsg {
		libraryStr = "{error: simulated error message}"
	}
	if err := os.WriteFile(filepath.Join(request.RepoDir, config.LibrarianDir, config.GenerateResponse), []byte(libraryStr), 0755); err != nil {
		return err
	}
	if m.failGenerateForID != "" {
		if request.LibraryID == m.failGenerateForID {
			return m.generateErr
		}
		m.requestLibraryID = request.LibraryID
		return nil
	}
	m.requestLibraryID = request.LibraryID
	return m.generateErr
}

func (m *mockContainerClient) Build(ctx context.Context, request *docker.BuildRequest) error {
	m.buildCalls++
	if m.noBuildResponse {
		return m.buildErr
	}
	// Write a build-response.json because it is required by generate
	// command.
	if err := os.MkdirAll(filepath.Join(request.RepoDir, ".librarian"), 0755); err != nil {
		return err
	}

	libraryStr := "{}"
	if m.wantErrorMsg {
		libraryStr = "{error: simulated error message}"
	}
	if err := os.WriteFile(filepath.Join(request.RepoDir, ".librarian", config.BuildResponse), []byte(libraryStr), 0755); err != nil {
		return err
	}
	return m.buildErr
}

func (m *mockContainerClient) Configure(ctx context.Context, request *docker.ConfigureRequest) (string, error) {
	m.configureCalls++

	if m.noConfigureResponse {
		return "", m.configureErr
	}

	// Write a configure-response.json because it is required by configure
	// command.
	if err := os.MkdirAll(filepath.Join(request.RepoDir, config.LibrarianDir), 0755); err != nil {
		return "", err
	}

	libraryStr := ""
	if m.wantErrorMsg {
		libraryStr = fmt.Sprintf(`{
	"ID": "%s",
  "error": "simulated error message"
}`, request.State.Libraries[0].ID)
	} else {
		libraryStr = fmt.Sprintf(`{
	"ID": "%s"
}`, request.State.Libraries[0].ID)
	}

	if err := os.WriteFile(filepath.Join(request.RepoDir, config.LibrarianDir, config.ConfigureResponse), []byte(libraryStr), 0755); err != nil {
		return "", err
	}
	return "", m.configureErr
}

type MockRepository struct {
	gitrepo.Repository
	Dir          string
	IsCleanValue bool
	IsCleanError error
	AddAllStatus git.Status
	AddAllError  error
	CommitError  error
	RemotesValue []*git.Remote
	RemotesError error
	CommitCalls  int
}

func (m *MockRepository) IsClean() (bool, error) {
	if m.IsCleanError != nil {
		return false, m.IsCleanError
	}
	return m.IsCleanValue, nil
}

func (m *MockRepository) AddAll() (git.Status, error) {
	if m.AddAllError != nil {
		return git.Status{}, m.AddAllError
	}
	return m.AddAllStatus, nil
}

func (m *MockRepository) Commit(msg string) error {
	m.CommitCalls++
	return m.CommitError
}

func (m *MockRepository) Remotes() ([]*git.Remote, error) {
	if m.RemotesError != nil {
		return nil, m.RemotesError
	}
	return m.RemotesValue, nil
}

func (m *MockRepository) GetDir() string {
	return m.Dir
}
