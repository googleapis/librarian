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
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
)

// mockContainerClient is a mock implementation of the ContainerClient interface for testing.

type mockContainerClient struct {
	ContainerClient
	generateCalls int
	buildCalls    int
}

func (m *mockContainerClient) Generate(ctx context.Context, cfg *config.Config, apiRoot, outputDir, generatorInput, libraryID string) error {
	m.generateCalls++
	return nil
}

func (m *mockContainerClient) Build(ctx context.Context, cfg *config.Config, buildDir, libraryID string) error {
	m.buildCalls++
	return nil
}

func TestDetectIfLibraryConfigured(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		apiPath       string
		repoPath      string
		pipelineState *config.PipelineState
		want          bool
		wantErr       bool
	}{
		{
			name:    "no repo specified",
			apiPath: "some/api",
		},
		{
			name:     "api path not in pipeline state",
			apiPath:  "other/api",
			repoPath: "some/repo",
			pipelineState: &config.PipelineState{
				Libraries: []*config.LibraryState{
					{
						ID:       "some-library",
						APIPaths: []string{"some/api"},
					},
				},
			},
		},
		{
			name:     "api path in pipeline state",
			apiPath:  "some/api",
			repoPath: "some/repo",
			pipelineState: &config.PipelineState{
				Libraries: []*config.LibraryState{
					{
						ID:       "some-library",
						APIPaths: []string{"some/api"},
					},
				},
			},
			want: true,
		},
		{
			name:     "pipeline state file does not exist",
			apiPath:  "some/api",
			repoPath: "some/repo",
			wantErr:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var repo *gitrepo.Repository
			if test.repoPath != "" {
				repo = newTestGitRepo(t)
				if test.pipelineState != nil {
					generatorInputDir := filepath.Join(repo.Dir, config.GeneratorInputDir)
					if err := os.MkdirAll(generatorInputDir, 0755); err != nil {
						t.Fatalf("os.MkdirAll(%q, 0755) = %v", generatorInputDir, err)
					}
					stateFile := filepath.Join(generatorInputDir, pipelineStateFile)
					b, err := json.MarshalIndent(test.pipelineState, "", "  ")
					if err != nil {
						t.Fatalf("json.MarshalIndent = %v", err)
					}
					if err := os.WriteFile(stateFile, b, 0644); err != nil {
						t.Fatalf("os.WriteFile(%q, ...) = %v", stateFile, err)
					}
				}
			}

			r := &generateRunner{
				cfg: &config.Config{
					API:  test.apiPath,
					Repo: test.repoPath,
				},
			}
			if repo != nil {
				r.cfg.Repo = repo.Dir
			}

			got, err := r.detectIfLibraryConfigured(context.Background())
			if (err != nil) != test.wantErr {
				t.Errorf("detectIfLibraryConfigured() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("detectIfLibraryConfigured() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunGenerateCommand(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name              string
		apiPath           string
		repo              *gitrepo.Repository
		state             *config.PipelineState
		container         *mockContainerClient
		wantLibraryID     string
		wantErr           bool
		wantGenerateCalls int
	}{
		{
			name:    "works",
			apiPath: "some/api",
			repo:    newTestGitRepo(t),
			state: &config.PipelineState{
				Libraries: []*config.LibraryState{
					{
						ID:       "some-library",
						APIPaths: []string{"some/api"},
					},
				},
			},
			container:         &mockContainerClient{},
			wantLibraryID:     "some-library",
			wantGenerateCalls: 1,
		},
		{
			name:      "missing repo",
			apiPath:   "some/api",
			container: &mockContainerClient{},
			wantErr:   true,
		},
		{
			name:    "library not found in state",
			apiPath: "other/api",
			repo:    newTestGitRepo(t),
			state: &config.PipelineState{
				Libraries: []*config.LibraryState{
					{
						ID:       "some-library",
						APIPaths: []string{"some/api"},
					},
				},
			},
			container: &mockContainerClient{},
			wantErr:   true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r := &generateRunner{
				cfg: &config.Config{
					API:    test.apiPath,
					Source: t.TempDir(),
				},
				repo:            test.repo,
				state:           test.state,
				containerClient: test.container,
			}

			outputDir := t.TempDir()
			gotLibraryID, err := r.runGenerateCommand(context.Background(), outputDir)
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
			name:           "build with library id",
			build:          true,
			libraryID:      "some-library",
			repo:           newTestGitRepo(t),
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
				containerClient: test.container,
			}
			outputDir := t.TempDir()
			if err := r.runBuildCommand(context.Background(), outputDir, test.libraryID); (err != nil) != test.wantErr {
				t.Errorf("runBuildCommand() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.wantBuildCalls, test.container.buildCalls); diff != "" {
				t.Errorf("runBuildCommand() buildCalls mismatch (-want +got):%s", diff)
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
			name:    "missing api flag",
			cfg:     &config.Config{Source: "some/source"},
			wantErr: true,
		},
		{
			name:    "missing source flag",
			cfg:     &config.Config{API: "some/api"},
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: &config.Config{
				API:      "some/api",
				Source:   t.TempDir(),
				Repo:     newTestGitRepo(t).Dir,
				WorkRoot: t.TempDir(),
				Image:    "gcr.io/test/test-image",
			},
		},
		{
			name: "missing image",
			cfg: &config.Config{
				API:      "some/api",
				Source:   t.TempDir(),
				Repo:     "https://github.com/googleapis/librarian.git",
				WorkRoot: t.TempDir(),
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			// We need to create a fake state and config file for the test to pass.
			if test.cfg.Repo != "" && !isUrl(test.cfg.Repo) {
				stateFile := filepath.Join(test.cfg.Repo, pipelineStateFile)
				if err := os.WriteFile(stateFile, []byte("{}"), 0644); err != nil {
					t.Fatalf("os.WriteFile(%q, ...) = %v", stateFile, err)
				}
				configFile := filepath.Join(test.cfg.Repo, pipelineConfigFile)
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
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "initial commit")
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
		name              string
		apiPath           string
		repo              *gitrepo.Repository
		state             *config.PipelineState
		container         *mockContainerClient
		build             bool
		wantErr           bool
		wantGenerateCalls int
		wantBuildCalls    int
	}{
		{
			name:    "regeneration of API",
			apiPath: "some/api",
			repo:    newTestGitRepo(t),
			state: &config.PipelineState{
				Libraries: []*config.LibraryState{
					{
						ID:       "some-library",
						APIPaths: []string{"some/api"},
					},
				},
			},
			container:         &mockContainerClient{},
			build:             true,
			wantGenerateCalls: 1,
			wantBuildCalls:    1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r := &generateRunner{
				cfg: &config.Config{
					API:    test.apiPath,
					Source: t.TempDir(),
					Build:  test.build,
				},
				repo:            test.repo,
				state:           test.state,
				containerClient: test.container,
				workRoot:        t.TempDir(),
			}

			if err := r.run(context.Background()); (err != nil) != test.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.wantGenerateCalls, test.container.generateCalls); diff != "" {
				t.Errorf("run() generateCalls mismatch (-want +got):%s", diff)
			}
			if diff := cmp.Diff(test.wantBuildCalls, test.container.buildCalls); diff != "" {
				t.Errorf("run() buildCalls mismatch (-want +got):%s", diff)
			}
		})
	}
}
