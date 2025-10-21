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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
	"gopkg.in/yaml.v3"
)

// mockGitHubClient is a mock implementation of the GitHubClient interface for testing.
type mockGitHubClient struct {
	GitHubClient
	rawContent              []byte
	rawErr                  error
	createPullRequestCalls  int
	addLabelsToIssuesCalls  int
	getLabelsCalls          int
	replaceLabelsCalls      int
	searchPullRequestsCalls int
	getPullRequestCalls     int
	createReleaseCalls      int
	createIssueCalls        int
	createTagCalls          int
	createPullRequestErr    error
	addLabelsToIssuesErr    error
	getLabelsErr            error
	replaceLabelsErr        error
	searchPullRequestsErr   error
	getPullRequestErr       error
	createReleaseErr        error
	createIssueErr          error
	createTagErr            error
	createdPR               *github.PullRequestMetadata
	labels                  []string
	pullRequests            []*github.PullRequest
	pullRequest             *github.PullRequest
	createdRelease          *github.RepositoryRelease
	librarianState          *config.LibrarianState
	librarianConfig         *config.LibrarianConfig
}

func (m *mockGitHubClient) GetRawContent(ctx context.Context, path, ref string) ([]byte, error) {
	if path == ".librarian/state.yaml" && m.librarianState != nil {
		return yaml.Marshal(m.librarianState)
	}

	if path == ".librarian/config.yaml" && m.librarianConfig != nil {
		return yaml.Marshal(m.librarianConfig)
	}
	return m.rawContent, m.rawErr
}

func (m *mockGitHubClient) CreatePullRequest(ctx context.Context, repo *github.Repository, remoteBranch, remoteBase, title, body string, isDraft bool) (*github.PullRequestMetadata, error) {
	m.createPullRequestCalls++
	if m.createPullRequestErr != nil {
		return nil, m.createPullRequestErr
	}
	return m.createdPR, nil
}

func (m *mockGitHubClient) AddLabelsToIssue(ctx context.Context, repo *github.Repository, number int, labels []string) error {
	m.addLabelsToIssuesCalls++
	m.labels = append(m.labels, labels...)
	return m.addLabelsToIssuesErr
}

func (m *mockGitHubClient) GetLabels(ctx context.Context, number int) ([]string, error) {
	m.getLabelsCalls++
	return m.labels, m.getLabelsErr
}

func (m *mockGitHubClient) ReplaceLabels(ctx context.Context, number int, labels []string) error {
	m.replaceLabelsCalls++
	return m.replaceLabelsErr
}

func (m *mockGitHubClient) SearchPullRequests(ctx context.Context, query string) ([]*github.PullRequest, error) {
	m.searchPullRequestsCalls++
	return m.pullRequests, m.searchPullRequestsErr
}

func (m *mockGitHubClient) GetPullRequest(ctx context.Context, number int) (*github.PullRequest, error) {
	m.getPullRequestCalls++
	return m.pullRequest, m.getPullRequestErr
}

func (m *mockGitHubClient) CreateRelease(ctx context.Context, tagName, releaseName, body, commitish string) (*github.RepositoryRelease, error) {
	m.createReleaseCalls++
	return m.createdRelease, m.createReleaseErr
}

func (m *mockGitHubClient) CreateIssueComment(ctx context.Context, number int, comment string) error {
	m.createIssueCalls++
	return m.createIssueErr
}

func (m *mockGitHubClient) CreateTag(ctx context.Context, tagName, commitish string) error {
	m.createTagCalls++
	return m.createTagErr
}

// mockContainerClient is a mock implementation of the ContainerClient interface for testing.
type mockContainerClient struct {
	ContainerClient
	generateCalls  int
	buildCalls     int
	configureCalls int
	initCalls      int
	generateErr    error
	buildErr       error
	configureErr   error
	initErr        error
	// Set this value if you want an error when
	// generate a library with a specific id.
	failGenerateForID string
	// Set this value if you want an error when
	// generate a library with a specific id.
	generateErrForID error
	// Set this value if you want an error when
	// build a library with a specific id.
	failBuildForID string
	// Set this value if you want an error when
	// build a library with a specific id.
	buildErrForID       error
	requestLibraryID    string
	noBuildResponse     bool
	noConfigureResponse bool
	noGenerateResponse  bool
	noReleaseResponse   bool
	noInitVersion       bool
	wantErrorMsg        bool
	// Set this value if you want library files
	// to be generated in source roots.
	wantLibraryGen bool
	// Set this value if you want the configure-response
	// has library source roots and remove regex.
	configureLibraryPaths []string
}

func (m *mockContainerClient) Build(ctx context.Context, request *docker.BuildRequest) error {
	m.buildCalls++
	if m.noBuildResponse {
		return m.buildErr
	}
	// Write a build-response.json unless we're configured not to.
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

	if m.failBuildForID != "" {
		if request.LibraryID == m.failBuildForID {
			return m.buildErrForID
		}
	}

	return m.buildErr
}

func (m *mockContainerClient) Configure(ctx context.Context, request *docker.ConfigureRequest) (string, error) {
	m.configureCalls++

	if m.noConfigureResponse {
		return "", m.configureErr
	}

	// Write a configure-response.json unless we're configured not to.
	if err := os.MkdirAll(filepath.Join(request.RepoDir, config.LibrarianDir), 0755); err != nil {
		return "", err
	}
	for _, library := range request.State.Libraries {
		needConfigure := false
		for _, oneApi := range library.APIs {
			if oneApi.Status == "new" {
				needConfigure = true
			}
		}

		if !needConfigure {
			continue
		}

		if !m.noInitVersion {
			library.Version = "0.1.0"
		}

		// Configure source root and remove regex.
		if len(m.configureLibraryPaths) != 0 {
			library.SourceRoots = make([]string, len(m.configureLibraryPaths))
			copy(library.SourceRoots, m.configureLibraryPaths)

			library.RemoveRegex = make([]string, len(m.configureLibraryPaths))
			copy(library.RemoveRegex, m.configureLibraryPaths)
		}

		if m.wantErrorMsg {
			library.ErrorMessage = "simulated error message"
		}

		b, err := json.Marshal(library)
		if err != nil {
			return "", err
		}

		if err := os.WriteFile(filepath.Join(request.RepoDir, config.LibrarianDir, config.ConfigureResponse), b, 0755); err != nil {
			return "", err
		}
	}

	return "", m.configureErr
}

func (m *mockContainerClient) Generate(ctx context.Context, request *docker.GenerateRequest) error {
	m.generateCalls++

	if m.noGenerateResponse {
		return m.generateErr
	}

	// // Write a generate-response.json unless we're configured not to.
	if err := os.MkdirAll(filepath.Join(request.RepoDir, config.LibrarianDir), 0755); err != nil {
		return err
	}

	library := &config.LibraryState{}
	library.ID = request.LibraryID
	if m.wantErrorMsg {
		library.ErrorMessage = "simulated error message"
	}
	b, err := json.MarshalIndent(library, "", " ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(request.RepoDir, config.LibrarianDir, config.GenerateResponse), b, 0755); err != nil {
		return err
	}

	if m.failGenerateForID != "" {
		if request.LibraryID == m.failGenerateForID {
			return m.generateErrForID
		}
	}

	m.requestLibraryID = request.LibraryID
	if m.wantLibraryGen {
		for _, library := range request.State.Libraries {
			if request.LibraryID != library.ID {
				continue
			}

			for _, src := range library.SourceRoots {
				srcPath := filepath.Join(request.Output, src)
				if err := os.MkdirAll(srcPath, 0755); err != nil {
					return err
				}
				if _, err := os.Create(filepath.Join(srcPath, "example.txt")); err != nil {
					return err
				}
			}
		}
	}

	return m.generateErr
}

func (m *mockContainerClient) ReleaseInit(ctx context.Context, request *docker.ReleaseInitRequest) error {
	m.initCalls++
	if m.noReleaseResponse {
		return m.initErr
	}
	// Write a release-init-response.json unless we're configured not to.
	if err := os.MkdirAll(filepath.Join(request.RepoDir, ".librarian"), 0755); err != nil {
		return err
	}

	library := &config.LibraryState{}
	if m.wantErrorMsg {
		library.ErrorMessage = "simulated error message"
	}
	b, err := json.MarshalIndent(library, "", " ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(request.RepoDir, ".librarian", config.ReleaseInitResponse), b, 0755); err != nil {
		return err
	}
	return m.initErr
}

type MockRepository struct {
	gitrepo.Repository
	Dir                                    string
	IsCleanValue                           bool
	IsCleanError                           error
	AddAllError                            error
	CommitError                            error
	RemotesValue                           []*gitrepo.Remote
	RemotesError                           error
	CommitCalls                            int
	LastCommitMessage                      string
	GetCommitError                         error
	GetLatestCommitError                   error
	GetCommitByHash                        map[string]*gitrepo.Commit
	GetCommitsForPathsSinceTagValue        []*gitrepo.Commit
	GetCommitsForPathsSinceTagValueByTag   map[string][]*gitrepo.Commit
	GetCommitsForPathsSinceTagError        error
	GetCommitsForPathsSinceTagLastTagName  string
	GetCommitsForPathsSinceLastGenValue    []*gitrepo.Commit
	GetCommitsForPathsSinceLastGenByCommit map[string][]*gitrepo.Commit
	GetCommitsForPathsSinceLastGenByPath   map[string][]*gitrepo.Commit
	GetLatestCommitByPath                  map[string]*gitrepo.Commit
	GetCommitsForPathsSinceLastGenError    error
	ChangedFilesInCommitValue              []string
	ChangedFilesInCommitValueByHash        map[string][]string
	ChangedFilesInCommitError              error
	ChangedFilesValue                      []string
	ChangedFilesError                      error
	CreateBranchAndCheckoutError           error
	PushCalls                              int
	PushError                              error
	RestoreError                           error
	HeadHashValue                          string
	HeadHashError                          error
	CheckoutCalls                          int
	CheckoutError                          error
	GetHashForPathError                    error
	// GetHashForPathValue is a map where each key is of the form "commitHash:path",
	// and the value is the hash to return. Every requested entry must be populated.
	// If the value is "error", an error is returned instead. (This is useful when some
	// calls must be successful, and others must fail.)
	GetHashForPathValue map[string]string
}

func (m *MockRepository) HeadHash() (string, error) {
	if m.HeadHashError != nil {
		return "", m.HeadHashError
	}
	return m.HeadHashValue, nil
}

func (m *MockRepository) IsClean() (bool, error) {
	if m.IsCleanError != nil {
		return false, m.IsCleanError
	}
	return m.IsCleanValue, nil
}

func (m *MockRepository) AddAll() error {
	if m.AddAllError != nil {
		return m.AddAllError
	}
	return nil
}

func (m *MockRepository) Commit(msg string) error {
	m.CommitCalls++
	m.LastCommitMessage = msg
	return m.CommitError
}

func (m *MockRepository) Remotes() ([]*gitrepo.Remote, error) {
	if m.RemotesError != nil {
		return nil, m.RemotesError
	}
	return m.RemotesValue, nil
}

func (m *MockRepository) GetDir() string {
	return m.Dir
}

func (m *MockRepository) GetCommit(commitHash string) (*gitrepo.Commit, error) {
	if m.GetCommitError != nil {
		return nil, m.GetCommitError
	}

	if m.GetCommitByHash != nil {
		if commit, ok := m.GetCommitByHash[commitHash]; ok {
			return commit, nil
		}
	}

	return nil, errors.New("should not reach here")
}

func (m *MockRepository) GetLatestCommit(path string) (*gitrepo.Commit, error) {
	if m.GetLatestCommitError != nil {
		return nil, m.GetLatestCommitError
	}

	if m.GetLatestCommitByPath != nil {
		if commit, ok := m.GetLatestCommitByPath[path]; ok {
			return commit, nil
		}
	}

	return nil, errors.New("should not reach here")
}

func (m *MockRepository) GetCommitsForPathsSinceTag(paths []string, tagName string) ([]*gitrepo.Commit, error) {
	m.GetCommitsForPathsSinceTagLastTagName = tagName
	if m.GetCommitsForPathsSinceTagError != nil {
		return nil, m.GetCommitsForPathsSinceTagError
	}
	if m.GetCommitsForPathsSinceTagValueByTag != nil {
		if commits, ok := m.GetCommitsForPathsSinceTagValueByTag[tagName]; ok {
			return commits, nil
		}
	}
	return m.GetCommitsForPathsSinceTagValue, nil
}

func (m *MockRepository) GetCommitsForPathsSinceCommit(paths []string, sinceCommit string) ([]*gitrepo.Commit, error) {
	if m.GetCommitsForPathsSinceLastGenError != nil {
		return nil, m.GetCommitsForPathsSinceLastGenError
	}

	if m.GetCommitsForPathsSinceLastGenByCommit != nil {
		if commits, ok := m.GetCommitsForPathsSinceLastGenByCommit[sinceCommit]; ok {
			return commits, nil
		}
	}

	if m.GetCommitsForPathsSinceLastGenByPath != nil {
		allCommits := make([]*gitrepo.Commit, 0)
		for _, path := range paths {
			if commits, ok := m.GetCommitsForPathsSinceLastGenByPath[path]; ok {
				allCommits = append(allCommits, commits...)
			}
		}

		return allCommits, nil
	}
	return m.GetCommitsForPathsSinceLastGenValue, nil
}

func (m *MockRepository) ChangedFilesInCommit(hash string) ([]string, error) {
	if m.ChangedFilesInCommitError != nil {
		return nil, m.ChangedFilesInCommitError
	}
	if m.ChangedFilesInCommitValueByHash != nil {
		if files, ok := m.ChangedFilesInCommitValueByHash[hash]; ok {
			return files, nil
		}
	}
	return m.ChangedFilesInCommitValue, nil
}

func (m *MockRepository) ChangedFiles() ([]string, error) {
	if m.ChangedFilesError != nil {
		return nil, m.ChangedFilesError
	}
	return m.ChangedFilesValue, nil
}

func (m *MockRepository) CreateBranchAndCheckout(name string) error {
	if m.CreateBranchAndCheckoutError != nil {
		return m.CreateBranchAndCheckoutError
	}
	return nil
}

func (m *MockRepository) Push(name string) error {
	m.PushCalls++
	if m.PushError != nil {
		return m.PushError
	}
	return nil
}

func (m *MockRepository) Restore(paths []string) error {
	return m.RestoreError
}

func (m *MockRepository) CleanUntracked(paths []string) error {
	return nil
}

func (m *MockRepository) Checkout(commitHash string) error {
	m.CheckoutCalls++
	if m.CheckoutError != nil {
		return m.CheckoutError
	}
	return nil
}

// mockImagesClient is a mock implementation of the ImageRegistryClient interface for testing.
type mockImagesClient struct {
	latestImage     string
	err             error
	findLatestCalls int
}

func (m *mockImagesClient) FindLatest(ctx context.Context, imageName string) (string, error) {
	m.findLatestCalls++
	return m.latestImage, m.err
}

func (m *MockRepository) GetHashForPath(commitHash, path string) (string, error) {
	if m.GetHashForPathError != nil {
		return "", m.GetHashForPathError
	}
	if m.GetHashForPathValue != nil {
		key := commitHash + ":" + path
		if hash, ok := m.GetHashForPathValue[key]; ok {
			if hash == "error" {
				return "", errors.New("deliberate error from GetHashForPath")
			}
			return hash, nil
		}

	}
	return "", fmt.Errorf("should not reach here: GetHashForPath called with unhandled input (commitHash: %q, path: %q)", commitHash, path)
}
