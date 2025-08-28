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
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
)

func TestRunGenerateCommand(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name              string
		api               string
		repo              gitrepo.Repository
		state             *config.LibrarianState
		container         *mockContainerClient
		ghClient          GitHubClient
		wantLibraryID     string
		wantErr           bool
		wantGenerateCalls int
	}{
		{
			name:     "works",
			api:      "some/api",
			repo:     newTestGitRepo(t),
			ghClient: &mockGitHubClient{},
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
		{
			name:     "works with no response",
			api:      "some/api",
			repo:     newTestGitRepo(t),
			ghClient: &mockGitHubClient{},
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []*config.API{{Path: "some/api"}},
					},
				},
			},
			container: &mockContainerClient{
				noGenerateResponse: true,
			},
			wantLibraryID:     "some-library",
			wantGenerateCalls: 1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r := &generateRunner{
				cfg: &config.Config{
					API:       test.api,
					APISource: t.TempDir(),
				},
				repo:            test.repo,
				sourceRepo:      newTestGitRepo(t),
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
		repo           gitrepo.Repository
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
		{
			name:      "build with no response",
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
			container: &mockContainerClient{
				noBuildResponse: true,
			},
			wantBuildCalls: 1,
		},
		{
			name:      "build with error response in response",
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
			container: &mockContainerClient{
				wantErrorMsg: true,
			},
			wantErr: true,
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
		repo               gitrepo.Repository
		state              *config.LibrarianState
		container          *mockContainerClient
		wantConfigureCalls int
		wantErr            bool
		wantErrMsg         string
	}{
		{
			name: "configures library successfully",
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
		{
			name: "configures library with non-existent api source",
			api:  "non-existent-dir/api",
			repo: newTestGitRepo(t),
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []*config.API{{Path: "non-existent-dir/api"}},
					},
				},
			},
			container:          &mockContainerClient{},
			wantConfigureCalls: 1,
			wantErr:            true,
			wantErrMsg:         "failed to read dir",
		},
		{
			name: "configures library with error message in response",
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
			container: &mockContainerClient{
				wantErrorMsg: true,
			},
			wantConfigureCalls: 1,
			wantErr:            true,
			wantErrMsg:         "failed with error message",
		},
		{
			name: "configures library with no response",
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
			container: &mockContainerClient{
				noConfigureResponse: true,
			},
			wantConfigureCalls: 1,
			wantErr:            true,
			wantErrMsg:         "no response file for configure container command",
		},
		{
			name: "configures library without initial version",
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
			container: &mockContainerClient{
				noInitVersion: true,
			},
			wantConfigureCalls: 1,
		},
		{
			name: "configure command failed",
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
			container: &mockContainerClient{
				configureErr:        errors.New("simulated configure command error"),
				noConfigureResponse: true,
			},
			wantConfigureCalls: 1,
			wantErr:            true,
			wantErrMsg:         "simulated configure command error",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			sourcePath := t.TempDir()
			cfg := &config.Config{
				API:       test.api,
				APISource: sourcePath,
			}
			r := &generateRunner{
				cfg:             cfg,
				repo:            test.repo,
				state:           test.state,
				containerClient: test.container,
			}

			// Create a service config
			if err := os.MkdirAll(filepath.Join(cfg.APISource, test.api), 0755); err != nil {
				t.Fatal(err)
			}

			data := []byte("type: google.api.Service")
			if err := os.WriteFile(filepath.Join(cfg.APISource, test.api, "example_service_v2.yaml"), data, 0755); err != nil {
				t.Fatal(err)
			}

			if test.name == "configures library with non-existent api source" {
				// This test verifies the scenario of no service config is found
				// in api path.
				if err := os.RemoveAll(filepath.Join(cfg.APISource)); err != nil {
					t.Fatal(err)
				}
			}

			_, err := r.runConfigureCommand(context.Background())

			if test.wantErr {
				if err == nil {
					t.Errorf("runConfigureCommand() should return error")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("runConfigureCommand() err = %v, want error containing %q", err, test.wantErrMsg)
				}

				return
			}

			if err != nil {
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
			name: "valid config",
			cfg: &config.Config{
				API:         "some/api",
				APISource:   newTestGitRepo(t).GetDir(),
				Repo:        newTestGitRepo(t).GetDir(),
				WorkRoot:    t.TempDir(),
				Image:       "gcr.io/test/test-image",
				CommandName: generateCmdName,
			},
		},
		{
			name: "invalid api source",
			cfg: &config.Config{
				API:         "some/api",
				APISource:   t.TempDir(), // Not a git repo
				Repo:        newTestGitRepo(t).GetDir(),
				WorkRoot:    t.TempDir(),
				Image:       "gcr.io/test/test-image",
				CommandName: generateCmdName,
			},
			wantErr: true,
		},
		{
			name: "missing image",
			cfg: &config.Config{
				API:         "some/api",
				APISource:   t.TempDir(),
				Repo:        "https://github.com/googleapis/librarian.git",
				WorkRoot:    t.TempDir(),
				CommandName: generateCmdName,
			},
			wantErr: true,
		},
		{
			name: "valid config with github token",
			cfg: &config.Config{
				API:         "some/api",
				APISource:   newTestGitRepo(t).GetDir(),
				Repo:        newTestGitRepo(t).GetDir(),
				WorkRoot:    t.TempDir(),
				Image:       "gcr.io/test/test-image",
				GitHubToken: "gh-token",
				CommandName: generateCmdName,
			},
		},
		{
			name: "empty API source",
			cfg: &config.Config{
				API:         "some/api",
				APISource:   "", // This will trigger the clone of googleapis
				Repo:        newTestGitRepo(t).GetDir(),
				WorkRoot:    t.TempDir(),
				Image:       "gcr.io/test/test-image",
				CommandName: generateCmdName,
			},
		},
		{
			name: "clone googleapis fails",
			cfg: &config.Config{
				API:         "some/api",
				APISource:   "", // This will trigger the clone of googleapis
				Repo:        newTestGitRepo(t).GetDir(),
				WorkRoot:    t.TempDir(),
				Image:       "gcr.io/test/test-image",
				CommandName: generateCmdName,
			},
			wantErr: true,
		},
		{
			name: "valid config with local repo",
			cfg: &config.Config{
				API:         "some/api",
				APISource:   newTestGitRepo(t).GetDir(),
				Repo:        newTestGitRepo(t).GetDir(),
				WorkRoot:    t.TempDir(),
				Image:       "gcr.io/test/test-image",
				CommandName: generateCmdName,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if test.cfg.APISource == "" && test.cfg.WorkRoot != "" {
				if test.name == "clone googleapis fails" {
					// The function will try to clone googleapis into the current work directory.
					// To make it fail, create a non-empty, non-git directory.
					googleapisDir := filepath.Join(test.cfg.WorkRoot, "googleapis")
					if err := os.MkdirAll(googleapisDir, 0755); err != nil {
						t.Fatalf("os.MkdirAll() = %v", err)
					}
					if err := os.WriteFile(filepath.Join(googleapisDir, "some-file"), []byte("foo"), 0644); err != nil {
						t.Fatalf("os.WriteFile() = %v", err)
					}
				} else {
					// The function will try to clone googleapis into the current work directory.
					// To prevent a real clone, we can pre-create a fake googleapis repo.
					googleapisDir := filepath.Join(test.cfg.WorkRoot, "googleapis")
					if err := os.MkdirAll(googleapisDir, 0755); err != nil {
						t.Fatalf("os.MkdirAll() = %v", err)
					}
					runGit(t, googleapisDir, "init")
					runGit(t, googleapisDir, "config", "user.email", "test@example.com")
					runGit(t, googleapisDir, "config", "user.name", "Test User")
					if err := os.WriteFile(filepath.Join(googleapisDir, "README.md"), []byte("test"), 0644); err != nil {
						t.Fatalf("os.WriteFile: %v", err)
					}
					runGit(t, googleapisDir, "add", "README.md")
					runGit(t, googleapisDir, "commit", "-m", "initial commit")
				}
			}

			r, err := newGenerateRunner(test.cfg)
			if (err != nil) != test.wantErr {
				t.Errorf("newGenerateRunner() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr {
				return
			}
			if r.ghClient == nil {
				t.Errorf("newGenerateRunner() ghClient is nil")
			}
			if r.containerClient == nil {
				t.Errorf("newGenerateRunner() containerClient is nil")
			}
			if r.repo == nil {
				t.Errorf("newGenerateRunner() repo is nil")
			}
			if r.sourceRepo == nil {
				t.Errorf("newGenerateRunner() sourceRepo is nil")
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
		repo               gitrepo.Repository
		state              *config.LibrarianState
		container          *mockContainerClient
		ghClient           GitHubClient
		build              bool
		wantErr            bool
		wantErrMsg         string
		wantGenerateCalls  int
		wantBuildCalls     int
		wantConfigureCalls int
	}{
		{
			name:    "generate single library including initial configuration",
			api:     "some/api",
			library: "some-library",
			repo:    newTestGitRepo(t),
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
			},
			container: &mockContainerClient{
				wantLibraryGen: true,
				configureLibraryPaths: []string{
					"src/a",
				},
			},
			ghClient:           &mockGitHubClient{},
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
						SourceRoots: []string{
							"src/a",
						},
					},
				},
			},
			container: &mockContainerClient{
				wantLibraryGen: true,
			},
			ghClient:           &mockGitHubClient{},
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
						SourceRoots: []string{
							"src/a",
						},
					},
				},
			},
			container: &mockContainerClient{
				wantLibraryGen: true,
			},
			ghClient:           &mockGitHubClient{},
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
						SourceRoots: []string{
							"src/a",
						},
					},
				},
			},
			container: &mockContainerClient{
				wantLibraryGen: true,
			},
			ghClient:           &mockGitHubClient{},
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
			build:      true,
			wantErr:    true,
			wantErrMsg: "not configured yet, generation stopped",
		},
		{
			name:    "generate single existing library with error message in response",
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
			container: &mockContainerClient{
				wantErrorMsg: true,
			},
			ghClient:           &mockGitHubClient{},
			wantGenerateCalls:  1,
			wantConfigureCalls: 0,
			wantErr:            true,
			wantErrMsg:         "failed with error message",
		},
		{
			name: "generate all libraries configured in state",
			repo: newTestGitRepo(t),
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "library1",
						APIs: []*config.API{{Path: "some/api1"}},
						SourceRoots: []string{
							"src/a",
						},
					},
					{
						ID:   "library2",
						APIs: []*config.API{{Path: "some/api2"}},
						SourceRoots: []string{
							"src/b",
						},
					},
				},
			},
			container: &mockContainerClient{
				wantLibraryGen: true,
			},
			ghClient:          &mockGitHubClient{},
			build:             true,
			wantGenerateCalls: 2,
			wantBuildCalls:    2,
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
			build:      true,
			wantErr:    true,
			wantErrMsg: "not configured yet, generation stopped",
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
			wantErrMsg:        "failed to make output directory",
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
			build:      true,
			wantErr:    true,
			wantErrMsg: "generate error",
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
						SourceRoots: []string{
							"src/a",
						},
					},
				},
			},
			container: &mockContainerClient{
				buildErr:       errors.New("build error"),
				wantLibraryGen: true,
			},
			ghClient:   &mockGitHubClient{},
			build:      true,
			wantErr:    true,
			wantErrMsg: "build error",
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
						SourceRoots: []string{
							"src/a",
						},
					},
					{
						ID:   "lib2",
						APIs: []*config.API{{Path: "some/api2"}},
						SourceRoots: []string{
							"src/b",
						},
					},
				},
			},
			container: &mockContainerClient{
				wantLibraryGen:    true,
				failGenerateForID: "lib1",
				generateErrForID:  errors.New("generate error"),
			},
			ghClient:          &mockGitHubClient{},
			build:             true,
			wantGenerateCalls: 2,
			wantBuildCalls:    1,
		},
		{
			name: "generate skips libraries with no APIs",
			repo: newTestGitRepo(t),
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID: "some-library",
					},
				},
			},
			container:          &mockContainerClient{},
			ghClient:           &mockGitHubClient{},
			build:              true,
			wantGenerateCalls:  0,
			wantBuildCalls:     0,
			wantConfigureCalls: 0,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				API:       test.api,
				Library:   test.library,
				APISource: t.TempDir(),
				Build:     test.build,
			}

			r := &generateRunner{
				cfg:             cfg,
				repo:            test.repo,
				sourceRepo:      newTestGitRepo(t),
				state:           test.state,
				containerClient: test.container,
				ghClient:        test.ghClient,
				workRoot:        t.TempDir(),
			}

			// Create a service config in api path.
			if err := os.MkdirAll(filepath.Join(cfg.APISource, test.api), 0755); err != nil {
				t.Fatal(err)
			}
			data := []byte("type: google.api.Service")
			if err := os.WriteFile(filepath.Join(cfg.APISource, test.api, "example_service_v2.yaml"), data, 0755); err != nil {
				t.Fatal(err)
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

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message %s, got %s", test.wantErrMsg, err.Error())
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

func TestUpdateLastGeneratedCommitState(t *testing.T) {
	t.Parallel()
	sourceRepo := newTestGitRepo(t)
	hash, err := sourceRepo.HeadHash()
	if err != nil {
		t.Fatal(err)
	}
	r := &generateRunner{
		sourceRepo: sourceRepo,
		state: &config.LibrarianState{
			Libraries: []*config.LibraryState{
				{
					ID: "some-library",
				},
			},
		},
	}
	if err := r.updateLastGeneratedCommitState("some-library"); err != nil {
		t.Fatal(err)
	}
	if r.state.Libraries[0].LastGeneratedCommit != hash {
		t.Errorf("updateState() got = %v, want %v", r.state.Libraries[0].LastGeneratedCommit, hash)
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

func TestUpdateChangesSinceLastGeneration(t *testing.T) {
	t.Parallel()
	hash1 := plumbing.NewHash("1234567")
	hash2 := plumbing.NewHash("abcdefg")
	for _, test := range []struct {
		name       string
		libraryID  string
		libraries  []*config.LibraryState
		repo       gitrepo.Repository
		want       *config.LibraryState
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:      "update changes in a library",
			libraryID: "another-id",
			libraries: []*config.LibraryState{
				{
					ID: "example-d",
				},
				{
					ID:                  "another-id",
					LastGeneratedCommit: "fake-sha",
					APIs: []*config.API{
						{
							Path: "api/one/path",
						},
						{
							Path: "api/another/path",
						},
					},
				},
			},
			repo: &MockRepository{
				GetCommitsForPathsSinceLastGenByPath: map[string][]*gitrepo.Commit{
					"api/one/path": {
						{
							Message: "feat: new feature\n\nThis is body.\n\nPiperOrigin-RevId: 98765",
							Hash:    hash1,
						},
					},
					"api/another/path": {
						{
							Message: "fix: a bug fix\n\nThis is another body.\n\nPiperOrigin-RevId: 573342",
							Hash:    hash2,
						},
					},
				},
				ChangedFilesInCommitValueByHash: map[string][]string{
					hash1.String(): {
						"api/one/path/file.txt",
						"api/another/path/example.txt",
					},
					hash2.String(): {
						"api/one/path/another-file.txt",
						"api/another/path/another-example.txt",
					},
				},
			},
			want: &config.LibraryState{
				ID:                  "another-id",
				LastGeneratedCommit: "fake-sha",
				APIs: []*config.API{
					{
						Path: "api/one/path",
					},
					{
						Path: "api/another/path",
					},
				},
				Changes: []*config.Change{
					{
						Type:       "feat",
						Subject:    "new feature",
						Body:       "This is body.",
						ClNum:      "98765",
						CommitHash: hash1.String(),
					},
					{
						Type:       "fix",
						Subject:    "a bug fix",
						Body:       "This is another body.",
						ClNum:      "573342",
						CommitHash: hash2.String(),
					},
				},
			},
		},
		{
			name:      "empty last generated commit",
			libraryID: "another-id",
			libraries: []*config.LibraryState{
				{
					ID: "example-d",
				},
				{
					ID: "another-id",
					APIs: []*config.API{
						{
							Path: "api/one/path",
						},
						{
							Path: "api/another/path",
						},
					},
				},
			},
			repo: &MockRepository{
				// Set this error to verify the function under test will not
				// fetch the commits.
				GetCommitsForPathsSinceLastGenError: errors.New("simulated error"),
			},
			want: &config.LibraryState{
				ID: "another-id",
				APIs: []*config.API{
					{
						Path: "api/one/path",
					},
					{
						Path: "api/another/path",
					},
				},
				Changes: []*config.Change{},
			},
		},
		{
			name:      "failed to get conventional commits",
			libraryID: "another-id",
			libraries: []*config.LibraryState{
				{
					ID: "example-d",
				},
				{
					ID:                  "another-id",
					LastGeneratedCommit: "fake-sha",
				},
			},
			repo: &MockRepository{
				GetCommitsForPathsSinceLastGenError: errors.New("simulated error"),
			},
			wantErr:    true,
			wantErrMsg: "failed to fetch conventional commits for library",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			runner := &generateRunner{
				sourceRepo: test.repo,
				state: &config.LibrarianState{
					Libraries: test.libraries,
				},
			}
			err := runner.updateChangesSinceLastGeneration(test.libraryID)
			if test.wantErr {
				if err == nil {
					t.Error("updateChangesSinceLastGeneration() should fail")
				}
				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message %s, got %s", test.wantErrMsg, err.Error())
				}

				return
			}

			if err != nil {
				t.Errorf("updateChangesSinceLastGeneration() failed: %q", err)
			}

			if diff := cmp.Diff(test.want, runner.state.Libraries[1]); diff != "" {
				t.Errorf("updateChangesSinceLastGeneration() dirs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
