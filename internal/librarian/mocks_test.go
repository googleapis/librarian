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
	"github.com/go-git/go-git/v5"
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
	generateCalls     int
	buildCalls        int
	configureCalls    int
	generateErr       error
	buildErr          error
	configureErr      error
	failGenerateForID string
}

func (m *mockContainerClient) Generate(ctx context.Context, request *docker.GenerateRequest) error {
	m.generateCalls++
	if m.failGenerateForID != "" {
		if request.LibraryID == m.failGenerateForID {
			return m.generateErr
		}
		return nil
	}
	return m.generateErr
}

func (m *mockContainerClient) Build(ctx context.Context, request *docker.BuildRequest) error {
	m.buildCalls++
	return m.buildErr
}

func (m *mockContainerClient) Configure(ctx context.Context, request *docker.ConfigureRequest) (string, error) {
	m.configureCalls++
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
