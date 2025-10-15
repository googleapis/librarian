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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
)

func TestNewGenerateRunner(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		cfg        *config.Config
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "valid config",
			cfg: &config.Config{
				API:         "some/api",
				APISource:   newTestGitRepo(t).GetDir(),
				Branch:      "test-branch",
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
			wantErr:    true,
			wantErrMsg: "repository does not exist",
		},
		{
			name: "missing image",
			cfg: &config.Config{
				API:         "some/api",
				APISource:   t.TempDir(),
				Branch:      "test-branch",
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
				Branch:      "test-branch",
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
				API:            "some/api",
				APISource:      "https://github.com/googleapis/googleapis", // This will trigger the clone of googleapis
				APISourceDepth: 1,
				Branch:         "test-branch",
				Repo:           newTestGitRepo(t).GetDir(),
				WorkRoot:       t.TempDir(),
				Image:          "gcr.io/test/test-image",
				CommandName:    generateCmdName,
			},
		},
		{
			name: "clone googleapis fails",
			cfg: &config.Config{
				API:            "some/api",
				APISource:      "", // This will trigger the clone of googleapis
				APISourceDepth: 1,
				Repo:           newTestGitRepo(t).GetDir(),
				WorkRoot:       t.TempDir(),
				Image:          "gcr.io/test/test-image",
				CommandName:    generateCmdName,
			},
			wantErr:    true,
			wantErrMsg: "repo must be specified",
		},
		{
			name: "valid config with local repo",
			cfg: &config.Config{
				API:         "some/api",
				APISource:   newTestGitRepo(t).GetDir(),
				Branch:      "test-branch",
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
			if test.wantErr {
				if err == nil {
					t.Fatalf("newGenerateRunner() error = %v, wantErr %v", err, test.wantErr)
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Fatalf("want error message: %s, got: %s", test.wantErrMsg, err.Error())
				}

				return
			}

			if err != nil {
				t.Fatalf("newGenerateRunner() got error: %v", err)
			}

			if r.branch == "" {
				t.Errorf("newGenerateRunner() branch is not set")
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

func TestRunConfigureCommand(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name               string
		api                string
		repo               gitrepo.Repository
		state              *config.LibrarianState
		librarianConfig    *config.LibrarianConfig
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
			name: "configure_library_without_global_files_in_output",
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
			librarianConfig: &config.LibrarianConfig{
				GlobalFilesAllowlist: []*config.GlobalFile{
					{
						Path: "a/path/example.txt",
					},
				},
			},
			container:          &mockContainerClient{},
			wantConfigureCalls: 1,
			wantErr:            true,
			wantErrMsg:         "failed to copy global file",
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
			outputDir := t.TempDir()
			r := &generateRunner{
				api:             test.api,
				repo:            test.repo,
				sourceRepo:      newTestGitRepo(t),
				state:           test.state,
				librarianConfig: test.librarianConfig,
				containerClient: test.container,
			}

			// Create a service config
			if err := os.MkdirAll(filepath.Join(r.sourceRepo.GetDir(), test.api), 0755); err != nil {
				t.Fatal(err)
			}

			data := []byte("type: google.api.Service")
			if err := os.WriteFile(filepath.Join(r.sourceRepo.GetDir(), test.api, "example_service_v2.yaml"), data, 0755); err != nil {
				t.Fatal(err)
			}

			if test.name == "configures library with non-existent api source" {
				// This test verifies the scenario of no service config is found
				// in api path.
				if err := os.RemoveAll(filepath.Join(r.sourceRepo.GetDir())); err != nil {
					t.Fatal(err)
				}
			}

			_, err := r.runConfigureCommand(context.Background(), outputDir)

			if test.wantErr {
				if err == nil {
					t.Fatal("runConfigureCommand() should return error")
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

func TestGenerateScenarios(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name               string
		api                string
		library            string
		state              *config.LibrarianState
		librarianConfig    *config.LibrarianConfig
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
			name:    "generate_single_library_including_initial_configuration",
			api:     "some/api",
			library: "some-library",
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
			name:    "generate_single_library_with_librarian_config",
			api:     "some/api",
			library: "some-library",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
			},
			container: &mockContainerClient{
				wantLibraryGen: true,
				configureLibraryPaths: []string{
					"src/a",
				},
			},
			librarianConfig: &config.LibrarianConfig{
				GlobalFilesAllowlist: []*config.GlobalFile{
					{
						Path:        "a/path/example.txt",
						Permissions: "read-only",
					},
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
			name: "generate skips blocked libraries",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "google.cloud.texttospeech.v1",
						APIs: []*config.API{{Path: "google/cloud/texttospeech/v1"}},
					},
					{
						ID:   "google.cloud.vision.v1",
						APIs: []*config.API{{Path: "google/cloud/vision/v1"}},
					},
				},
			},
			librarianConfig: &config.LibrarianConfig{
				Libraries: []*config.LibraryConfig{
					{LibraryID: "google.cloud.texttospeech.v1"},
					{LibraryID: "google.cloud.vision.v1", GenerateBlocked: true},
				},
			},
			container: &mockContainerClient{
				wantLibraryGen: true,
			},
			ghClient:          &mockGitHubClient{},
			build:             true,
			wantGenerateCalls: 1,
			wantBuildCalls:    1,
		},
		{
			name:    "generate runs blocked libraries if explicitly requested",
			library: "google.cloud.vision.v1",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "google.cloud.texttospeech.v1",
						APIs: []*config.API{{Path: "google/cloud/texttospeech/v1"}},
					},
					{
						ID:   "google.cloud.vision.v1",
						APIs: []*config.API{{Path: "google/cloud/vision/v1"}},
					},
				},
			},
			librarianConfig: &config.LibrarianConfig{
				Libraries: []*config.LibraryConfig{
					{LibraryID: "google.cloud.texttospecech.v1"},
					{LibraryID: "google.cloud.vision.v1", GenerateBlocked: true},
				},
			},
			container: &mockContainerClient{
				wantLibraryGen: true,
			},
			ghClient:          &mockGitHubClient{},
			build:             true,
			wantGenerateCalls: 1,
			wantBuildCalls:    1,
		},
		{
			name: "generate skips a blocked library and the rest fail. should report error",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "google.cloud.texttospeech.v1",
						APIs: []*config.API{{Path: "google/cloud/texttospeech/v1"}},
					},
					{
						ID:   "google.cloud.vision.v1",
						APIs: []*config.API{{Path: "google/cloud/vision/v1"}},
					},
				},
			},
			librarianConfig: &config.LibrarianConfig{
				Libraries: []*config.LibraryConfig{
					{LibraryID: "google.cloud.texttospeech.v1"},
					{LibraryID: "google.cloud.vision.v1", GenerateBlocked: true},
				},
			},
			container:  &mockContainerClient{generateErr: errors.New("generate error")},
			ghClient:   &mockGitHubClient{},
			build:      true,
			wantErr:    true,
			wantErrMsg: "all 1 libraries failed to generate (blocked: 1)",
		},
		{
			name: "generate all, all fail should report error",
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
				},
			},
			container: &mockContainerClient{
				failGenerateForID: "lib1",
				generateErrForID:  errors.New("generate error"),
			},
			ghClient:          &mockGitHubClient{},
			build:             true,
			wantErr:           true,
			wantErrMsg:        "all 1 libraries failed to generate",
			wantGenerateCalls: 1,
			wantBuildCalls:    0,
		},
		{
			name: "generate skips libraries with no APIs",
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
			repo := newTestGitRepoWithState(t, test.state, true)

			r := &generateRunner{
				api:             test.api,
				library:         test.library,
				build:           test.build,
				repo:            repo,
				sourceRepo:      newTestGitRepo(t),
				state:           test.state,
				librarianConfig: test.librarianConfig,
				containerClient: test.container,
				ghClient:        test.ghClient,
				workRoot:        t.TempDir(),
			}

			// Create a service config in api path.
			if err := os.MkdirAll(filepath.Join(r.sourceRepo.GetDir(), test.api), 0755); err != nil {
				t.Fatal(err)
			}
			data := []byte("type: google.api.Service")
			if err := os.WriteFile(filepath.Join(r.sourceRepo.GetDir(), test.api, "example_service_v2.yaml"), data, 0755); err != nil {
				t.Fatal(err)
			}

			// Commit the service config file because configure command needs
			// to find the piper id associated with the commit message.
			if err := r.sourceRepo.AddAll(); err != nil {
				t.Fatal(err)
			}
			message := "feat: add an api\n\nPiperOrigin-RevId: 123456"
			if err := r.sourceRepo.Commit(message); err != nil {
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
					t.Fatalf("%s should return error", test.name)
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

func TestFormatGenerationPRBody(t *testing.T) {
	t.Parallel()

	today := time.Now()
	hash1 := plumbing.NewHash("1234567890abcdef")
	hash2 := plumbing.NewHash("fedcba0987654321")
	librarianVersion := cli.Version()

	for _, test := range []struct {
		name            string
		state           *config.LibrarianState
		sourceRepo      gitrepo.Repository
		languageRepo    gitrepo.Repository
		idToCommits     map[string]string
		failedLibraries []string
		api             string
		library         string
		apiOnboarding   bool
		want            string
		wantErr         bool
		wantErrPhrase   string
	}{
		{
			// This test verifies that only changed libraries appear in the pull request
			// body.
			name: "multiple libraries generation",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
						APIs: []*config.API{
							{
								Path: "path/to",
							},
						},
					},
					{
						ID:          "another-library",
						SourceRoots: []string{"path/to"},
						APIs: []*config.API{
							{
								Path: "path/to",
							},
						},
					},
				},
			},
			sourceRepo: &MockRepository{
				RemotesValue: []*gitrepo.Remote{{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}}},
				GetCommitByHash: map[string]*gitrepo.Commit{
					"1234567890": {
						Hash: plumbing.NewHash("1234567890"),
						When: time.UnixMilli(200),
					},
					"abcdefg": {
						Hash: plumbing.NewHash("abcdefg"),
						When: time.UnixMilli(300),
					},
				},
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*gitrepo.Commit{
					"1234567890": {
						{
							Message: "fix: a bug fix\n\nThis is another body.\n\nPiperOrigin-RevId: 573342",
							Hash:    hash2,
							When:    today.Add(time.Hour),
						},
					},
					"abcdefg": {}, // no new commits since commit "abcdefg".
				},
				ChangedFilesInCommitValueByHash: map[string][]string{
					hash2.String(): {
						"path/to/file",
					},
				},
			},
			languageRepo: &MockRepository{
				IsCleanValue:              true,
				HeadHashValue:             "5678",
				ChangedFilesInCommitValue: []string{"path/to/a.go"},
			},
			idToCommits: map[string]string{
				"one-library":     "1234567890",
				"another-library": "abcdefg",
			},
			failedLibraries: []string{},
			want: fmt.Sprintf(`BEGIN_COMMIT_OVERRIDE

BEGIN_NESTED_COMMIT
fix: a bug fix
This is another body.

PiperOrigin-RevId: 573342
Library-IDs: one-library
Source-link: [googleapis/googleapis@fedcba09](https://github.com/googleapis/googleapis/commit/fedcba09)
END_NESTED_COMMIT

END_COMMIT_OVERRIDE

This pull request is generated with proto changes between
[googleapis/googleapis@abcdef00](https://github.com/googleapis/googleapis/commit/abcdef0000000000000000000000000000000000)
(exclusive) and
[googleapis/googleapis@fedcba09](https://github.com/googleapis/googleapis/commit/fedcba0987654321000000000000000000000000)
(inclusive).

Librarian Version: %s
Language Image: %s`,
				librarianVersion, "go:1.21"),
		},
		{
			name: "group_commit_messages",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
						APIs: []*config.API{
							{
								Path: "path/to",
							},
						},
					},
					{
						ID:          "another-library",
						SourceRoots: []string{"path/to"},
						APIs: []*config.API{
							{
								Path: "path/to",
							},
						},
					},
				},
			},
			sourceRepo: &MockRepository{
				RemotesValue: []*gitrepo.Remote{{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}}},
				GetCommitByHash: map[string]*gitrepo.Commit{
					"1234567890": {
						Hash: plumbing.NewHash("1234567890"),
						When: time.UnixMilli(200),
					},
					"abcdefg": {
						Hash: plumbing.NewHash("abcdefg"),
						When: time.UnixMilli(300),
					},
				},
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*gitrepo.Commit{
					"1234567890": {
						{
							Message: "fix: a bug fix\n\nThis is another body.\n\nPiperOrigin-RevId: 573342",
							Hash:    hash2,
							When:    today.Add(time.Hour),
						},
					},
					"abcdefg": {
						{
							Message: "fix: a bug fix\n\nThis is another body.\n\nPiperOrigin-RevId: 573342",
							Hash:    hash2,
							When:    today.Add(time.Hour),
						},
					},
				},
				ChangedFilesInCommitValueByHash: map[string][]string{
					hash2.String(): {
						"path/to/file",
					},
				},
			},
			languageRepo: &MockRepository{
				IsCleanValue:              true,
				HeadHashValue:             "5678",
				ChangedFilesInCommitValue: []string{"path/to/a.go"},
			},
			idToCommits: map[string]string{
				"one-library":     "1234567890",
				"another-library": "abcdefg",
			},
			failedLibraries: []string{},
			want: fmt.Sprintf(`BEGIN_COMMIT_OVERRIDE

BEGIN_NESTED_COMMIT
fix: a bug fix
This is another body.

PiperOrigin-RevId: 573342
Library-IDs: one-library,another-library
Source-link: [googleapis/googleapis@fedcba09](https://github.com/googleapis/googleapis/commit/fedcba09)
END_NESTED_COMMIT

END_COMMIT_OVERRIDE

This pull request is generated with proto changes between
[googleapis/googleapis@abcdef00](https://github.com/googleapis/googleapis/commit/abcdef0000000000000000000000000000000000)
(exclusive) and
[googleapis/googleapis@fedcba09](https://github.com/googleapis/googleapis/commit/fedcba0987654321000000000000000000000000)
(inclusive).

Librarian Version: %s
Language Image: %s`,
				librarianVersion, "go:1.21"),
		},
		{
			name: "multiple libraries generation with failed libraries",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
						APIs: []*config.API{
							{
								Path: "path/to",
							},
						},
					},
					{
						ID:          "another-library",
						SourceRoots: []string{"path/to"},
						APIs: []*config.API{
							{
								Path: "path/to",
							},
						},
					},
				},
			},
			sourceRepo: &MockRepository{
				RemotesValue: []*gitrepo.Remote{{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}}},
				GetCommitByHash: map[string]*gitrepo.Commit{
					"1234567890": {
						Hash: plumbing.NewHash("1234567890"),
						When: time.UnixMilli(200),
					},
					"abcdefg": {
						Hash: plumbing.NewHash("abcdefg"),
						When: time.UnixMilli(300),
					},
				},
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*gitrepo.Commit{
					"1234567890": {
						{
							Message: "fix: a bug fix\n\nThis is another body.\n\nPiperOrigin-RevId: 573342",
							Hash:    hash2,
							When:    today.Add(time.Hour),
						},
					},
					"abcdefg": {}, // no new commits since commit "abcdefg".
				},
				ChangedFilesInCommitValueByHash: map[string][]string{
					hash2.String(): {
						"path/to/file",
					},
				},
			},
			languageRepo: &MockRepository{
				IsCleanValue:              true,
				HeadHashValue:             "5678",
				ChangedFilesInCommitValue: []string{"path/to/a.go"},
			},
			idToCommits: map[string]string{
				"one-library":     "1234567890",
				"another-library": "abcdefg",
			},
			failedLibraries: []string{
				"failed-library-a",
				"failed-library-b",
			},
			want: fmt.Sprintf(`BEGIN_COMMIT_OVERRIDE

BEGIN_NESTED_COMMIT
fix: a bug fix
This is another body.

PiperOrigin-RevId: 573342
Library-IDs: one-library
Source-link: [googleapis/googleapis@fedcba09](https://github.com/googleapis/googleapis/commit/fedcba09)
END_NESTED_COMMIT

END_COMMIT_OVERRIDE

This pull request is generated with proto changes between
[googleapis/googleapis@abcdef00](https://github.com/googleapis/googleapis/commit/abcdef0000000000000000000000000000000000)
(exclusive) and
[googleapis/googleapis@fedcba09](https://github.com/googleapis/googleapis/commit/fedcba0987654321000000000000000000000000)
(inclusive).

Librarian Version: %s
Language Image: %s

## Generation failed for
- failed-library-a
- failed-library-b`,
				librarianVersion, "go:1.21"),
		},
		{
			name: "single library generation",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
						APIs: []*config.API{
							{
								Path: "path/to",
							},
						},
					},
				},
			},
			sourceRepo: &MockRepository{
				RemotesValue: []*gitrepo.Remote{{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}}},
				GetCommitByHash: map[string]*gitrepo.Commit{
					"1234567890": {
						Hash: plumbing.NewHash("1234567890"),
						When: time.UnixMilli(200),
					},
				},
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*gitrepo.Commit{
					"1234567890": {
						{
							Message: "feat: new feature\n\nThis is body.\n\nPiperOrigin-RevId: 98765",
							Hash:    hash1,
							When:    today,
						},
						{
							Message: "fix: a bug fix\n\nThis is another body.\n\nPiperOrigin-RevId: 573342",
							Hash:    hash2,
							When:    today.Add(time.Hour),
						},
					},
				},
				ChangedFilesInCommitValueByHash: map[string][]string{
					hash1.String(): {
						"path/to/file",
						"path/to/another/file",
					},
					hash2.String(): {
						"path/to/file",
					},
				},
			},
			languageRepo: &MockRepository{
				IsCleanValue:              true,
				HeadHashValue:             "5678",
				ChangedFilesInCommitValue: []string{"path/to/a.go"},
			},
			idToCommits: map[string]string{
				"one-library": "1234567890",
			},
			failedLibraries: []string{},
			want: fmt.Sprintf(`BEGIN_COMMIT_OVERRIDE

BEGIN_NESTED_COMMIT
fix: a bug fix
This is another body.

PiperOrigin-RevId: 573342
Library-IDs: one-library
Source-link: [googleapis/googleapis@fedcba09](https://github.com/googleapis/googleapis/commit/fedcba09)
END_NESTED_COMMIT

BEGIN_NESTED_COMMIT
feat: new feature
This is body.

PiperOrigin-RevId: 98765
Library-IDs: one-library
Source-link: [googleapis/googleapis@12345678](https://github.com/googleapis/googleapis/commit/12345678)
END_NESTED_COMMIT

END_COMMIT_OVERRIDE

This pull request is generated with proto changes between
[googleapis/googleapis@12345678](https://github.com/googleapis/googleapis/commit/1234567890000000000000000000000000000000)
(exclusive) and
[googleapis/googleapis@fedcba09](https://github.com/googleapis/googleapis/commit/fedcba0987654321000000000000000000000000)
(inclusive).

Librarian Version: %s
Language Image: %s`,
				librarianVersion, "go:1.21"),
		},
		{
			name: "no conventional commit is found since last generation",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
						// Intentionally set this value to verify the test can pass.
						LastGeneratedCommit: "randomCommit",
						APIs: []*config.API{
							{
								Path: "path/to",
							},
						},
					},
				},
			},
			sourceRepo: &MockRepository{
				RemotesValue:   []*gitrepo.Remote{{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}}},
				GetCommitError: errors.New("simulated get commit error"),
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*gitrepo.Commit{
					"1234567890": {
						{
							Message: "feat: new feature\n\nThis is body.\n\nPiperOrigin-RevId: 98765",
							Hash:    hash1,
							When:    today,
						},
						{
							Message: "fix: a bug fix\n\nThis is another body.\n\nPiperOrigin-RevId: 573342",
							Hash:    hash2,
							When:    today.Add(time.Hour),
						},
					},
				},
				ChangedFilesInCommitValueByHash: map[string][]string{
					hash1.String(): {
						"path/to/file",
						"path/to/another/file",
					},
					hash2.String(): {
						"path/to/file",
					},
				},
			},
			languageRepo: &MockRepository{
				IsCleanValue:              true,
				HeadHashValue:             "5678",
				ChangedFilesInCommitValue: []string{"path/to/a.go"},
			},
			idToCommits: map[string]string{
				"one-library": "1234567890",
			},
			wantErr:       true,
			wantErrPhrase: "failed to find the start commit",
		},
		{
			name: "no conventional commits since last generation",
			state: &config.LibrarianState{
				Image:     "go:1.21",
				Libraries: []*config.LibraryState{{ID: "one-library", SourceRoots: []string{"path/to"}}},
			},
			sourceRepo: &MockRepository{},
			languageRepo: &MockRepository{
				HeadHashValue:             "5678",
				ChangedFilesInCommitValue: []string{"path/to/a.go"},
			},
			idToCommits: map[string]string{
				"one-library": "",
			},
			want: "No commit is found since last generation",
		},
		{
			name: "failed to get language repo changes commits",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
					},
				},
			},
			sourceRepo: &MockRepository{},
			languageRepo: &MockRepository{
				IsCleanError: errors.New("simulated error"),
			},
			idToCommits: map[string]string{
				"one-library": "1234567890",
			},
			wantErr:       true,
			wantErrPhrase: "failed to fetch changes in language repo",
		},
		{
			name: "failed to get conventional commits",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
					},
				},
			},
			sourceRepo: &MockRepository{
				GetCommitsForPathsSinceLastGenError: errors.New("simulated error"),
			},
			languageRepo: &MockRepository{
				IsCleanValue:              true,
				HeadHashValue:             "5678",
				ChangedFilesInCommitValue: []string{"path/to/a.go"},
			},
			idToCommits: map[string]string{
				"one-library": "1234567890",
			},
			wantErr:       true,
			wantErrPhrase: "failed to fetch conventional commits for library",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := &generationPRRequest{
				sourceRepo:      test.sourceRepo,
				languageRepo:    test.languageRepo,
				state:           test.state,
				idToCommits:     test.idToCommits,
				failedLibraries: test.failedLibraries,
			}
			got, err := formatGenerationPRBody(req)
			if test.wantErr {
				if err == nil {
					t.Fatalf("%s should return error", test.name)
				}
				if !strings.Contains(err.Error(), test.wantErrPhrase) {
					t.Errorf("formatGenerationPRBody() returned error %q, want to contain %q", err.Error(), test.wantErrPhrase)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("formatGenerationPRBody() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatOnboardPRBody(t *testing.T) {
	t.Parallel()
	librarianVersion := cli.Version()

	for _, test := range []struct {
		name          string
		state         *config.LibrarianState
		sourceRepo    gitrepo.Repository
		api           string
		library       string
		want          string
		wantErr       bool
		wantErrPhrase string
	}{
		{
			name: "onboarding_new_api",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
						APIs: []*config.API{
							{
								Path:          "path/to",
								ServiceConfig: "library_v1.yaml",
							},
						},
					},
				},
			},
			sourceRepo: &MockRepository{
				GetLatestCommitByPath: map[string]*gitrepo.Commit{
					"path/to/library_v1.yaml": {
						Message: "feat: new feature\n\nThis is body.\n\nPiperOrigin-RevId: 98765",
					},
				},
			},
			api:     "path/to",
			library: "one-library",
			want: fmt.Sprintf(`feat: onboard a new library

PiperOrigin-RevId: 98765
Library-IDs: one-library
Librarian Version: %s
Language Image: %s`,
				librarianVersion, "go:1.21"),
		},
		{
			name: "no_latest_commit_during_api_onboarding",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
						APIs: []*config.API{
							{
								Path:          "path/to",
								ServiceConfig: "library_v1.yaml",
							},
						},
					},
				},
			},
			sourceRepo: &MockRepository{
				GetLatestCommitError: errors.New("no latest commit"),
			},
			api:           "path/to",
			library:       "one-library",
			wantErr:       true,
			wantErrPhrase: "no latest commit",
		},
		{
			name: "latest_commit_does_not_contain_piper_during_api_onboarding",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
						APIs: []*config.API{
							{
								Path:          "path/to",
								ServiceConfig: "library_v1.yaml",
							},
						},
					},
				},
			},
			sourceRepo: &MockRepository{
				GetLatestCommitByPath: map[string]*gitrepo.Commit{
					"path/to/library_v1.yaml": {
						Message: "feat: new feature\n\nThis is body.",
					},
				},
			},
			api:           "path/to",
			library:       "one-library",
			wantErr:       true,
			wantErrPhrase: errPiperNotFound.Error(),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := &onboardPRRequest{
				sourceRepo: test.sourceRepo,
				state:      test.state,
				api:        test.api,
				library:    test.library,
			}
			got, err := formatOnboardPRBody(req)
			if test.wantErr {
				if err == nil {
					t.Fatalf("%s should return error", test.name)
				}
				if !strings.Contains(err.Error(), test.wantErrPhrase) {
					t.Errorf("formatOnboardPRBody() returned error %q, want to contain %q", err.Error(), test.wantErrPhrase)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("formatOnboardPRBody() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindLatestCommit(t *testing.T) {
	t.Parallel()

	today := time.Now()
	hash1 := plumbing.NewHash("1234567890abcdef")
	hash2 := plumbing.NewHash("fedcba0987654321")
	hash3 := plumbing.NewHash("ghfgsfgshfsdf232")
	for _, test := range []struct {
		name          string
		state         *config.LibrarianState
		repo          gitrepo.Repository
		idToCommits   map[string]string
		want          *gitrepo.Commit
		wantErr       bool
		wantErrPhrase string
	}{
		{
			name: "find the last generated commit",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "one-library",
					},
					{
						ID: "another-library",
					},
					{
						ID: "yet-another-library",
					},
					{
						ID: "skipped-library",
					},
				},
			},
			repo: &MockRepository{
				GetCommitByHash: map[string]*gitrepo.Commit{
					hash1.String(): {
						Hash:    hash1,
						Message: "this is a message",
						When:    today.Add(time.Hour),
					},
					hash2.String(): {
						Hash:    hash2,
						Message: "this is another message",
						When:    today.Add(2 * time.Hour).Add(time.Minute),
					},
					hash3.String(): {
						Hash:    hash3,
						Message: "yet another message",
						When:    today.Add(2 * time.Hour),
					},
				},
			},
			idToCommits: map[string]string{
				"one-library":         hash1.String(),
				"another-library":     hash2.String(),
				"yet-another-library": hash3.String(),
			},
			want: &gitrepo.Commit{
				Hash:    hash2,
				Message: "this is another message",
				When:    today.Add(2 * time.Hour).Add(time.Minute),
			},
		},
		{
			name: "failed to find last generated commit",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "one-library",
					},
				},
			},
			repo: &MockRepository{
				GetCommitError: errors.New("simulated error"),
			},
			idToCommits: map[string]string{
				"one-library": "1234567890",
			},
			wantErr:       true,
			wantErrPhrase: "can't find last generated commit for",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := findLatestGenerationCommit(test.repo, test.state, test.idToCommits)
			if test.wantErr {
				if err == nil {
					t.Fatalf("%s should return error", test.name)
				}
				if !strings.Contains(err.Error(), test.wantErrPhrase) {
					t.Errorf("findLatestCommit() returned error %q, want to contain %q", err.Error(), test.wantErrPhrase)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("findLatestCommit() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
func TestGroupByPiperID(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		commits []*gitrepo.ConventionalCommit
		want    []*gitrepo.ConventionalCommit
	}{
		{
			name: "group_commits_with_same_piper_id_and_subject",
			commits: []*gitrepo.ConventionalCommit{
				{
					LibraryID: "library-1",
					Subject:   "one subject",
					Footers: map[string]string{
						"PiperOrigin-RevId": "123456",
					},
				},
				{
					LibraryID: "library-2",
					Subject:   "a different subject",
					Footers: map[string]string{
						"PiperOrigin-RevId": "123456",
					},
				},
				{
					LibraryID: "library-3",
					Subject:   "the same subject",
					Footers: map[string]string{
						"PiperOrigin-RevId": "987654",
					},
				},
				{
					LibraryID: "library-4",
					Subject:   "the same subject",
					Footers: map[string]string{
						"PiperOrigin-RevId": "987654",
					},
				},
				{
					LibraryID: "library-5",
				},
				{
					LibraryID: "library-6",
					Footers: map[string]string{
						"random-key": "random-value",
					},
				},
			},
			want: []*gitrepo.ConventionalCommit{
				{
					LibraryID: "library-1",
					Subject:   "one subject",
					Footers: map[string]string{
						"PiperOrigin-RevId": "123456",
						"Library-IDs":       "library-1",
					},
				},
				{
					LibraryID: "library-2",
					Subject:   "a different subject",
					Footers: map[string]string{
						"PiperOrigin-RevId": "123456",
						"Library-IDs":       "library-2",
					},
				},
				{
					LibraryID: "library-3",
					Subject:   "the same subject",
					Footers: map[string]string{
						"PiperOrigin-RevId": "987654",
						"Library-IDs":       "library-3,library-4",
					},
				},
				{
					LibraryID: "library-5",
					Footers: map[string]string{
						"Library-IDs": "library-5",
					},
				},
				{
					LibraryID: "library-6",
					Footers: map[string]string{
						"random-key":  "random-value",
						"Library-IDs": "library-6",
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := groupByIDAndSubject(test.commits)
			// We don't care the order in the slice but sorting makes the test deterministic.
			opts := cmpopts.SortSlices(func(a, b *gitrepo.ConventionalCommit) bool {
				return a.LibraryID < b.LibraryID
			})
			if diff := cmp.Diff(test.want, got, opts); diff != "" {
				t.Errorf("groupByIDAndSubject() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
