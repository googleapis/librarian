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
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/docker"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
	"gopkg.in/yaml.v3"
)

// mockContainerClient is a mock implementation of the ContainerClient interface for testing.
type mockContainerClient struct {
	ContainerClient
	generateCalls     int
	buildCalls        int
	configureCalls    int
	generateErr       error
	buildErr          error
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

func (m *mockContainerClient) Configure(ctx context.Context, request *docker.ConfigureRequest) error {
	m.configureCalls++
	return nil
}

func (m *mockGitHubClient) CreatePullRequest(ctx context.Context, repo *github.Repository, remoteBranch, title, body string) (*github.PullRequestMetadata, error) {
	if m.rawErr != nil {
		return nil, m.rawErr
	}
	// Return an empty metadata struct and no error to satisfy the interface.
	return &github.PullRequestMetadata{}, nil
}

func TestRunGenerateCommand(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name              string
		api               string
		pushConfig        string
		repo              *gitrepo.Repository
		state             *config.LibrarianState
		container         *mockContainerClient
		ghClient          GitHubClient
		wantLibraryID     string
		wantErr           bool
		wantGenerateCalls int
	}{
		{
			name:       "works",
			api:        "some/api",
			pushConfig: "xxx@email.com,author",
			repo:       newTestGitRepo(t),
			ghClient:   &mockGitHubClient{},
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []*config.API{{Path: "some/api"}},
					},
				},
			},
			container:         &mockContainerClient{},
			wantLibraryID:     "some-library",
			wantGenerateCalls: 1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r := &generateRunner{
				cfg: &config.Config{
					API:        test.api,
					APISource:  t.TempDir(),
					PushConfig: test.pushConfig,
				},
				repo:            test.repo,
				ghClient:        test.ghClient,
				state:           test.state,
				containerClient: test.container,
			}

			outputDir := t.TempDir()
			gotLibraryID, err := r.runGenerateCommand(context.Background(), "some-library", outputDir)
			if (err != nil) != test.wantErr {
				t.Errorf("runGenerateCommand() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.wantLibraryID, gotLibraryID); diff != "" {
				t.Errorf("runGenerateCommand() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantGenerateCalls, test.container.generateCalls); diff != "" {
				t.Errorf("runGenerateCommand() generateCalls mismatch (-want +got):%s", diff)
			}
		})
	}
}

func TestRunBuildCommand(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name           string
		build          bool
		libraryID      string
		repo           *gitrepo.Repository
		state          *config.LibrarianState
		container      *mockContainerClient
		wantBuildCalls int
		wantErr        bool
	}{
		{
			name:           "build flag not specified",
			build:          false,
			container:      &mockContainerClient{},
			wantBuildCalls: 0,
		},
		{
			name:      "build with library id",
			build:     true,
			libraryID: "some-library",
			repo:      newTestGitRepo(t),
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "some-library",
					},
				},
			},
			container:      &mockContainerClient{},
			wantBuildCalls: 1,
		},
		{
			name:      "build with no library id",
			build:     true,
			container: &mockContainerClient{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r := &generateRunner{
				cfg: &config.Config{
					Build: test.build,
				},
				repo:            test.repo,
				state:           test.state,
				containerClient: test.container,
			}
			err := r.runBuildCommand(context.Background(), test.libraryID)
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantBuildCalls, test.container.buildCalls); diff != "" {
				t.Errorf("runBuildCommand() buildCalls mismatch (-want +got):%s", diff)
			}
		})
	}
}

func TestRunConfigureCommand(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name               string
		api                string
		apiSource          string
		repo               *gitrepo.Repository
		state              *config.LibrarianState
		container          *mockContainerClient
		wantConfigureCalls int
		wantErr            bool
	}{
		{
			name: "configures library",
			api:  "some/api",
			repo: newTestGitRepo(t),
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []*config.API{{Path: "some/api"}},
					},
				},
			},
			container:          &mockContainerClient{},
			wantConfigureCalls: 1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			sourcePath := test.apiSource
			if sourcePath == "" {
				sourcePath = t.TempDir()
			}
			r := &generateRunner{
				cfg: &config.Config{
					API:       test.api,
					APISource: sourcePath,
				},
				repo:            test.repo,
				state:           test.state,
				containerClient: test.container,
			}

			_, err := r.runConfigureCommand(context.Background())
			if (err != nil) != test.wantErr {
				t.Errorf("runConfigureCommand() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.wantConfigureCalls, test.container.configureCalls); diff != "" {
				t.Errorf("runConfigureCommand() configureCalls mismatch (-want +got):%s", diff)
			}
		})
	}
}

func TestNewGenerateRunner(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name:    "missing repo flag",
			cfg:     &config.Config{API: "some/api"},
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: &config.Config{
				API:       "some/api",
				APISource: t.TempDir(),
				Repo:      newTestGitRepo(t).Dir,
				WorkRoot:  t.TempDir(),
				Image:     "gcr.io/test/test-image",
			},
		},
		{
			name: "missing image",
			cfg: &config.Config{

				API:       "some/api",
				APISource: t.TempDir(),
				Repo:      "https://github.com/googleapis/librarian.git",
				WorkRoot:  t.TempDir(),
			},
			wantErr: true,
		},
		{
			name: "push config without github token",
			cfg: &config.Config{
				API:        "some/api",
				APISource:  "some/source",
				PushConfig: "test@example.com,Test User",
			},
			wantErr: true,
		},
		{
			name: "push config with github token is valid",
			cfg: &config.Config{
				API:         "some/api",
				APISource:   t.TempDir(),
				Repo:        newTestGitRepo(t).Dir,
				WorkRoot:    t.TempDir(),
				Image:       "gcr.io/test/test-image",
				PushConfig:  "test@example.com,Test User",
				GitHubToken: "gh-token",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			// We need to create a fake state and config file for the test to pass.
			if test.cfg.Repo != "" && !isURL(test.cfg.Repo) {
				stateFile := filepath.Join(test.cfg.Repo, config.LibrarianDir, pipelineStateFile)

				if err := os.MkdirAll(filepath.Dir(stateFile), 0755); err != nil {
					t.Fatalf("os.MkdirAll() = %v", err)
				}
				state := &config.LibrarianState{
					Image: "some/image:v1.2.3",
					Libraries: []*config.LibraryState{
						{
							ID:          "some-library",
							APIs:        []*config.API{{Path: "some/api", ServiceConfig: "api_config.yaml"}},
							SourcePaths: []string{"src/a"},
						},
					},
				}
				b, err := yaml.Marshal(state)
				if err != nil {
					t.Fatalf("yaml.Marshal() = %v", err)
				}
				if err := os.WriteFile(stateFile, b, 0644); err != nil {
					t.Fatalf("os.WriteFile(%q, ...) = %v", stateFile, err)
				}
				configFile := filepath.Join(test.cfg.Repo, config.LibrarianDir, pipelineConfigFile)
				if err := os.WriteFile(configFile, []byte("{}"), 0644); err != nil {
					t.Fatalf("os.WriteFile(%q, ...) = %v", configFile, err)
				}
				runGit(t, test.cfg.Repo, "add", ".")
				runGit(t, test.cfg.Repo, "commit", "-m", "add config")
			}

			_, err := newGenerateRunner(test.cfg)
			if (err != nil) != test.wantErr {
				t.Errorf("newGenerateRunner() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// newTestGitRepo creates a new git repository in a temporary directory.
func newTestGitRepo(t *testing.T) *gitrepo.Repository {
	t.Helper()
	dir := t.TempDir()
	remoteURL := "https://github.com/googleapis/librarian.git"
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "initial commit")
	runGit(t, dir, "remote", "add", "origin", remoteURL)
	repo, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{Dir: dir})
	if err != nil {
		t.Fatalf("gitrepo.Open(%q) = %v", dir, err)
	}
	return repo
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}

func TestGenerateRun(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name               string
		api                string
		repo               *gitrepo.Repository
		state              *config.LibrarianState
		container          *mockContainerClient
		ghClient           GitHubClient
		pushConfig         string
		build              bool
		wantErr            bool
		wantGenerateCalls  int
		wantBuildCalls     int
		wantConfigureCalls int
	}{
		{
			name: "generation of API",
			api:  "some/api",
			repo: newTestGitRepo(t),
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []*config.API{{Path: "some/api"}},
					},
				},
			},
			container:          &mockContainerClient{},
			ghClient:           &mockGitHubClient{},
			pushConfig:         "xxx@email.com,author",
			build:              true,
			wantGenerateCalls:  1,
			wantBuildCalls:     1,
			wantConfigureCalls: 0,
		},
		{
			name: "symlink in output",
			api:  "some/api",
			repo: newTestGitRepo(t),
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []*config.API{{Path: "some/api"}},
					},
				},
			},
			container:         &mockContainerClient{},
			build:             true,
			wantGenerateCalls: 1,
			wantErr:           true,
		},
		{
			name: "generate error",
			api:  "some/api",
			repo: newTestGitRepo(t),
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []*config.API{{Path: "some/api"}},
					},
				},
			},
			container:  &mockContainerClient{generateErr: errors.New("generate error")},
			ghClient:   &mockGitHubClient{},
			pushConfig: "xxx@email.com,author",
			build:      true,
			wantErr:    true,
		},
		{
			name: "build error",
			api:  "some/api",
			repo: newTestGitRepo(t),
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []*config.API{{Path: "some/api"}},
					},
				},
			},
			container:  &mockContainerClient{buildErr: errors.New("build error")},
			ghClient:   &mockGitHubClient{},
			pushConfig: "xxx@email.com,author",
			build:      true,
			wantErr:    true,
		},
		{
			name: "generate all, partial failure does not halt execution",
			repo: newTestGitRepo(t),
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "lib1",
						APIs: []*config.API{{Path: "some/api1"}},
					},
					{
						ID:   "lib2",
						APIs: []*config.API{{Path: "some/api2"}},
					},
				},
			},
			container: &mockContainerClient{
				failGenerateForID: "lib1",
				generateErr:       errors.New("generate error"),
			},
			ghClient:          &mockGitHubClient{},
			build:             true,
			wantGenerateCalls: 2,
			wantBuildCalls:    1,
		},
		{
			name: "commit and push error",
			api:  "some/api",
			repo: newTestGitRepo(t),
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []*config.API{{Path: "some/api"}},
					},
				},
			},
			container:  &mockContainerClient{},
			ghClient:   &mockGitHubClient{rawErr: errors.New("commit and push error")},
			pushConfig: "xxx@email.com,author",
			build:      true,
			wantErr:    true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r := &generateRunner{
				cfg: &config.Config{
					API:        test.api,
					PushConfig: test.pushConfig,
					APISource:  t.TempDir(),
					Build:      test.build,
				},
				repo:            test.repo,
				state:           test.state,
				containerClient: test.container,
				ghClient:        test.ghClient,
				workRoot:        t.TempDir(),
			}

			// Create a symlink in the output directory to trigger an error.
			if test.name == "symlink in output" {
				outputDir := filepath.Join(r.workRoot, "output")
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					t.Fatalf("os.MkdirAll() = %v", err)
				}
				if err := os.Symlink("target", filepath.Join(outputDir, "symlink")); err != nil {
					t.Fatalf("os.Symlink() = %v", err)
				}
			}

			err := r.run(context.Background())
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantGenerateCalls, test.container.generateCalls); diff != "" {
				t.Errorf("run() generateCalls mismatch (-want +got):%s", diff)
			}
			if diff := cmp.Diff(test.wantBuildCalls, test.container.buildCalls); diff != "" {
				t.Errorf("run() buildCalls mismatch (-want +got):%s", diff)
			}
			if diff := cmp.Diff(test.wantConfigureCalls, test.container.configureCalls); diff != "" {
				t.Errorf("run() configureCalls mismatch (-want +got):%s", diff)
			}
		})
	}
}

func TestGenerateScenarios(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name               string
		api                string
		library            string
		repo               *gitrepo.Repository
		state              *config.LibrarianState
		container          *mockContainerClient
		ghClient           GitHubClient
		pushConfig         string
		build              bool
		wantErr            bool
		wantGenerateCalls  int
		wantBuildCalls     int
		wantConfigureCalls int
	}{
		{
			name:               "generate single library including initial configuration",
			api:                "some/api",
			library:            "some-library",
			repo:               newTestGitRepo(t),
			state:              &config.LibrarianState{Image: "gcr.io/test/image:v1.2.3"},
			container:          &mockContainerClient{},
			ghClient:           &mockGitHubClient{},
			pushConfig:         "xxx@email.com,author",
			build:              true,
			wantGenerateCalls:  1,
			wantBuildCalls:     1,
			wantConfigureCalls: 1,
		},
		{
			name:    "generate single existing library by library id",
			library: "some-library",
			repo:    newTestGitRepo(t),
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []*config.API{{Path: "some/api"}},
					},
				},
			},
			container:          &mockContainerClient{},
			ghClient:           &mockGitHubClient{},
			pushConfig:         "xxx@email.com,author",
			build:              true,
			wantGenerateCalls:  1,
			wantBuildCalls:     1,
			wantConfigureCalls: 0,
		},
		{
			name: "generate single existing library by api",
			api:  "some/api",
			repo: newTestGitRepo(t),
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []*config.API{{Path: "some/api"}},
					},
				},
			},
			container:          &mockContainerClient{},
			ghClient:           &mockGitHubClient{},
			pushConfig:         "xxx@email.com,author",
			build:              true,
			wantGenerateCalls:  1,
			wantBuildCalls:     1,
			wantConfigureCalls: 0,
		},
		{
			name:    "generate single existing library with library id and api",
			api:     "some/api",
			library: "some-library",
			repo:    newTestGitRepo(t),
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []*config.API{{Path: "some/api"}},
					},
				},
			},
			container:          &mockContainerClient{},
			ghClient:           &mockGitHubClient{},
			pushConfig:         "xxx@email.com,author",
			build:              true,
			wantGenerateCalls:  1,
			wantBuildCalls:     1,
			wantConfigureCalls: 0,
		},
		{
			name:    "generate single existing library with invalid library id should fail",
			library: "some-not-configured-library",
			repo:    newTestGitRepo(t),
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []*config.API{{Path: "some/api"}},
					},
				},
			},
			container:  &mockContainerClient{},
			ghClient:   &mockGitHubClient{},
			pushConfig: "xxx@email.com,author",
			build:      true,
			wantErr:    true,
		},
		{
			name: "generate all libraries configured in state",
			repo: newTestGitRepo(t),
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{ID: "library1", APIs: []*config.API{{Path: "some/api1"}}},
					{ID: "library2", APIs: []*config.API{{Path: "some/api2"}}},
				},
			},
			container:         &mockContainerClient{},
			ghClient:          &mockGitHubClient{},
			pushConfig:        "xxx@email.com,author",
			build:             true,
			wantGenerateCalls: 2,
			wantBuildCalls:    2,
		},
		{
			name:    "generate single library, corrupted library id",
			library: "non-existent-library",
			repo:    newTestGitRepo(t),
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []*config.API{{Path: "some/api"}},
					},
				},
			},
			container:  &mockContainerClient{},
			ghClient:   &mockGitHubClient{},
			pushConfig: "xxx@email.com,author",
			build:      true,
			wantErr:    true,
		},
		{
			name: "generate single library, corrupted api",
			api:  "corrupted/api/path",
			repo: newTestGitRepo(t),
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []*config.API{{Path: "some/api"}},
					},
				},
			},
			container:  &mockContainerClient{},
			ghClient:   &mockGitHubClient{},
			pushConfig: "xxx@email.com,author",
			build:      true,
			wantErr:    true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r := &generateRunner{
				cfg: &config.Config{
					API:        test.api,
					Library:    test.library,
					PushConfig: test.pushConfig,
					APISource:  t.TempDir(),
					Build:      test.build,
				},
				repo:            test.repo,
				state:           test.state,
				containerClient: test.container,
				ghClient:        test.ghClient,
				workRoot:        t.TempDir(),
			}

			err := r.run(context.Background())
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantGenerateCalls, test.container.generateCalls); diff != "" {
				t.Errorf("%s: run() generateCalls mismatch (-want +got):%s", test.name, diff)
			}
			if diff := cmp.Diff(test.wantBuildCalls, test.container.buildCalls); diff != "" {
				t.Errorf("%s: run() buildCalls mismatch (-want +got):%s", test.name, diff)
			}
			if diff := cmp.Diff(test.wantConfigureCalls, test.container.configureCalls); diff != "" {
				t.Errorf("%s: run() configureCalls mismatch (-want +got):%s", test.name, diff)
			}
		})
	}
}

func TestClean(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name             string
		files            map[string]string
		setup            func(t *testing.T, tmpDir string)
		symlinks         map[string]string
		removePatterns   []string
		preservePatterns []string
		wantRemaining    []string
		wantErr          bool
	}{
		{
			name: "remove all",
			files: map[string]string{
				"file1.txt": "",
				"file2.txt": "",
			},
			removePatterns: []string{".*\\.txt"},
			wantRemaining:  []string{"."},
		},
		{
			name: "preserve all",
			files: map[string]string{
				"file1.txt": "",
				"file2.txt": "",
			},
			removePatterns:   []string{".*"},
			preservePatterns: []string{".*"},
			wantRemaining:    []string{".", "file1.txt", "file2.txt"},
		},
		{
			name: "remove some",
			files: map[string]string{
				"foo/file1.txt": "",
				"foo/file2.txt": "",
				"bar/file3.txt": "",
			},
			removePatterns: []string{"foo/.*"},
			wantRemaining:  []string{".", "bar", "bar/file3.txt", "foo"},
		},
		{
			name: "invalid remove pattern",
			files: map[string]string{
				"file1.txt": "",
			},
			removePatterns: []string{"["}, // Invalid regex
			wantErr:        true,
		},
		{
			name: "invalid preserve pattern",
			files: map[string]string{
				"file1.txt": "",
			},
			removePatterns:   []string{".*"},
			preservePatterns: []string{"["}, // Invalid regex
			wantErr:          true,
		},
		{
			name: "remove symlink",
			files: map[string]string{
				"file1.txt": "content",
			},
			symlinks: map[string]string{
				"symlink_to_file1": "file1.txt",
			},
			removePatterns: []string{"symlink_to_file1"},
			wantRemaining:  []string{".", "file1.txt"},
		},
		{
			name: "remove file symlinked to",
			files: map[string]string{
				"file1.txt": "content",
			},
			symlinks: map[string]string{
				"symlink_to_file1": "file1.txt",
			},
			removePatterns: []string{"file1.txt"},
			// The symlink should remain, even though it's now broken, because
			// it was not targeted for removal.
			wantRemaining: []string{".", "symlink_to_file1"},
		},
		{
			name: "remove directory",
			files: map[string]string{
				"dir/file1.txt": "",
				"dir/file2.txt": "",
			},
			removePatterns: []string{"dir"},
			wantRemaining:  []string{"."},
		},
		{
			name: "preserve file not matching remove pattern",
			files: map[string]string{
				"file1.txt": "",
				"file2.log": "",
			},
			removePatterns: []string{".*\\.txt"},
			wantRemaining:  []string{".", "file2.log"},
		},
		{
			name: "remove file fails on permission error",
			files: map[string]string{
				"readonlydir/file.txt": "content",
			},
			setup: func(t *testing.T, tmpDir string) {
				// Make the directory read-only to cause os.Remove to fail.
				readOnlyDir := filepath.Join(tmpDir, "readonlydir")
				if err := os.Chmod(readOnlyDir, 0555); err != nil {
					t.Fatalf("os.Chmod() = %v", err)
				}
				// Register a cleanup function to restore permissions so TempDir can be removed.
				t.Cleanup(func() {
					_ = os.Chmod(readOnlyDir, 0755)
				})
			},
			removePatterns: []string{"readonlydir/file.txt"},
			wantRemaining:  []string{".", "readonlydir", "readonlydir/file.txt"},
			wantErr:        true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			for path, content := range test.files {
				fullPath := filepath.Join(tmpDir, path)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatalf("os.MkdirAll() = %v", err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatalf("os.WriteFile() = %v", err)
				}
			}
			for link, target := range test.symlinks {
				linkPath := filepath.Join(tmpDir, link)
				if err := os.Symlink(target, linkPath); err != nil {
					t.Fatalf("os.Symlink() = %v", err)
				}
			}
			if test.setup != nil {
				test.setup(t, tmpDir)
			}
			err := clean(tmpDir, test.removePatterns, test.preservePatterns)
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			remainingPaths, err := allPaths(tmpDir)
			if err != nil {
				t.Fatalf("allPaths() = %v", err)
			}
			sort.Strings(test.wantRemaining)
			sort.Strings(remainingPaths)
			if diff := cmp.Diff(test.wantRemaining, remainingPaths); diff != "" {
				t.Errorf("clean() remaining files mismatch (-want +got):%s", diff)
			}

		})
	}
}

func TestSortDirsByDepth(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		dirs []string
		want []string
	}{
		{
			name: "simple case",
			dirs: []string{
				"a/b",
				"short-dir",
				"a/b/c",
				"a",
			},
			want: []string{
				"a/b/c",
				"a/b",
				"short-dir",
				"a",
			},
		},
		{
			name: "empty",
			dirs: []string{},
			want: []string{},
		},
		{
			name: "single dir",
			dirs: []string{"a"},
			want: []string{"a"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sortDirsByDepth(tc.dirs)
			if diff := cmp.Diff(tc.want, tc.dirs); diff != "" {
				t.Errorf("sortDirsByDepth() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAllPaths(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		setup       func(t *testing.T, tmpDir string)
		wantPaths   []string
		wantErr     bool
		errorString string
	}{
		{
			name: "success",
			setup: func(t *testing.T, tmpDir string) {
				files := []string{
					"file1.txt",
					"dir1/file2.txt",
					"dir1/dir2/file3.txt",
				}
				for _, file := range files {
					path := filepath.Join(tmpDir, file)
					if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
						t.Fatalf("os.MkdirAll() = %v", err)
					}
					if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
						t.Fatalf("os.WriteFile() = %v", err)
					}
				}
			},
			wantPaths: []string{
				".",
				"dir1",
				"dir1/dir2",
				"dir1/dir2/file3.txt",
				"dir1/file2.txt",
				"file1.txt",
			},
		},
		{
			name: "unreadable directory",
			setup: func(t *testing.T, tmpDir string) {
				unreadableDir := filepath.Join(tmpDir, "unreadable")
				if err := os.Mkdir(unreadableDir, 0755); err != nil {
					t.Fatalf("os.Mkdir() = %v", err)
				}

				// Make the directory unreadable to trigger an error in filepath.WalkDir.
				if err := os.Chmod(unreadableDir, 0000); err != nil {
					t.Fatalf("os.Chmod() = %v", err)
				}
				// Schedule cleanup to restore permissions so TempDir can be removed.
				t.Cleanup(func() {
					_ = os.Chmod(unreadableDir, 0755)
				})
			},
			wantErr:     true,
			errorString: "unreadable",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if test.setup != nil {
				test.setup(t, tmpDir)
			}

			paths, err := allPaths(tmpDir)
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			// Sort both slices to ensure consistent comparison.
			sort.Strings(paths)
			sort.Strings(test.wantPaths)

			if diff := cmp.Diff(test.wantPaths, paths); diff != "" {
				t.Errorf("allPaths() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFilterPaths(t *testing.T) {
	t.Parallel()
	paths := []string{
		"foo/file1.txt",
		"foo/file2.log",
		"bar/file3.txt",
		"bar/file4.log",
	}
	regexps := []*regexp.Regexp{
		regexp.MustCompile(`^foo/.*\.txt$`),
		regexp.MustCompile(`^bar/.*`),
	}

	filtered := filterPaths(paths, regexps)

	wantFiltered := []string{
		"foo/file1.txt",
		"bar/file3.txt",
		"bar/file4.log",
	}

	sort.Strings(filtered)
	sort.Strings(wantFiltered)

	if diff := cmp.Diff(wantFiltered, filtered); diff != "" {
		t.Errorf("filterPaths() mismatch (-want +got):%s", diff)
	}
}

func TestDeriveFinalPathsToRemove(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name             string
		files            map[string]string
		removePatterns   []string
		preservePatterns []string
		wantToRemove     []string
		wantErr          bool
	}{
		{
			name: "remove all txt files, preserve nothing",
			files: map[string]string{
				"file1.txt":      "",
				"dir1/file2.txt": "",
				"dir2/file3.log": "",
			},
			removePatterns:   []string{`.*\.txt`},
			preservePatterns: []string{},
			wantToRemove:     []string{"file1.txt", "dir1/file2.txt"},
		},
		{
			name: "remove all files, preserve log files",
			files: map[string]string{
				"file1.txt":      "",
				"dir1/file2.txt": "",
				"dir2/file3.log": "",
			},
			removePatterns:   []string{".*"},
			preservePatterns: []string{`.*\.log`},
			wantToRemove:     []string{".", "dir1", "dir2", "file1.txt", "dir1/file2.txt"},
		},
		{
			name: "remove files in dir1, preserve nothing",
			files: map[string]string{
				"file1.txt":      "",
				"dir1/file2.txt": "",
				"dir1/file3.log": "",
				"dir2/file4.txt": "",
			},
			removePatterns:   []string{`dir1/.*`},
			preservePatterns: []string{},
			wantToRemove:     []string{"dir1/file2.txt", "dir1/file3.log"},
		},
		{
			name: "remove all, preserve files in dir2",
			files: map[string]string{
				"file1.txt":      "",
				"dir1/file2.txt": "",
				"dir2/file3.txt": "",
			},
			removePatterns:   []string{".*"},
			preservePatterns: []string{`dir2/.*`},
			wantToRemove:     []string{".", "dir1", "dir2", "file1.txt", "dir1/file2.txt"},
		},
		{
			name:             "no files",
			files:            map[string]string{},
			removePatterns:   []string{".*"},
			preservePatterns: []string{},
			wantToRemove:     []string{"."},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			for path, content := range test.files {
				fullPath := filepath.Join(tmpDir, path)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatalf("os.MkdirAll() = %v", err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatalf("os.WriteFile() = %v", err)
				}
			}

			gotToRemove, err := deriveFinalPathsToRemove(tmpDir, test.removePatterns, test.preservePatterns)
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			sort.Strings(gotToRemove)
			sort.Strings(test.wantToRemove)

			if diff := cmp.Diff(test.wantToRemove, gotToRemove); diff != "" {
				t.Errorf("deriveFinalPathsToRemove() toRemove mismatch in %s (-want +got):\n%s", test.name, diff)
			}
		})
	}
}

func TestSeparateFilesAndDirs(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		setup     func(t *testing.T, tmpDir string)
		paths     []string
		wantFiles []string
		wantDirs  []string
		wantErr   bool
	}{
		{
			name: "mixed files, dirs, and non-existent path",
			setup: func(t *testing.T, tmpDir string) {
				files := []string{"file1.txt", "dir1/file2.txt"}
				dirs := []string{"dir1", "dir2"}
				for _, file := range files {
					path := filepath.Join(tmpDir, file)
					if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
						t.Fatalf("os.MkdirAll() = %v", err)
					}
					if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
						t.Fatalf("os.WriteFile() = %v", err)
					}
				}
				for _, dir := range dirs {
					if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
						t.Fatalf("os.MkdirAll() = %v", err)
					}
				}
			},
			paths:     []string{"file1.txt", "dir1/file2.txt", "dir1", "dir2", "non-existent-file"},
			wantFiles: []string{"file1.txt", "dir1/file2.txt"},
			wantDirs:  []string{"dir1", "dir2"},
		},
		{
			name:    "stat error",
			paths:   []string{strings.Repeat("a", 300)},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if test.setup != nil {
				test.setup(t, tmpDir)
			}

			gotFiles, gotDirs, err := separateFilesAndDirs(tmpDir, test.paths)
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			sort.Strings(gotFiles)
			sort.Strings(gotDirs)
			sort.Strings(test.wantFiles)
			sort.Strings(test.wantDirs)

			if diff := cmp.Diff(test.wantFiles, gotFiles); diff != "" {
				t.Errorf("separateFilesAndDirs() files mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantDirs, gotDirs); diff != "" {
				t.Errorf("separateFilesAndDirs() dirs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCompileRegexps(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		patterns []string
		wantErr  bool
	}{
		{
			name: "valid patterns",
			patterns: []string{
				`^foo.*`,
				`\\.txt$`,
			},
			wantErr: false,
		},
		{
			name:     "empty patterns",
			patterns: []string{},
			wantErr:  false,
		},
		{
			name: "invalid pattern",
			patterns: []string{
				`[`,
			},
			wantErr: true,
		},
		{
			name: "mixed valid and invalid patterns",
			patterns: []string{
				`^foo.*`,
				`[`,
			},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			regexps, err := compileRegexps(tc.patterns)
			if (err != nil) != tc.wantErr {
				t.Fatalf("compileRegexps() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr {
				if len(regexps) != len(tc.patterns) {
					t.Errorf("compileRegexps() len = %d, want %d", len(regexps), len(tc.patterns))
				}
			}
		})
	}
}

func TestCleanAndCopyLibrary(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		libraryID   string
		state       *config.LibrarianState
		repo        *gitrepo.Repository
		outputDir   string
		setup       func(t *testing.T, r *generateRunner, outputDir string)
		wantErr     bool
		errContains string
	}{
		{
			name:      "library not found",
			libraryID: "non-existent-library",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "some-library",
					},
				},
			},
			repo:    newTestGitRepo(t),
			wantErr: true,
		},
		{
			name:      "clean fails",
			libraryID: "some-library",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:          "some-library",
						RemoveRegex: []string{"["}, // Invalid regex
					},
				},
			},
			repo:    newTestGitRepo(t),
			wantErr: true,
		},
		{
			name:      "copy fails on symlink",
			libraryID: "some-library",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "some-library",
					},
				},
			},
			repo: newTestGitRepo(t),
			setup: func(t *testing.T, r *generateRunner, outputDir string) {
				// Create a symlink in the output directory to trigger an error.
				if err := os.Symlink("target", filepath.Join(outputDir, "symlink")); err != nil {
					t.Fatalf("os.Symlink() = %v", err)
				}
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r := &generateRunner{
				state: test.state,
				repo:  test.repo,
			}
			outputDir := t.TempDir()
			if test.setup != nil {
				test.setup(t, r, outputDir)
			}
			err := r.cleanAndCopyLibrary(test.libraryID, outputDir)
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
