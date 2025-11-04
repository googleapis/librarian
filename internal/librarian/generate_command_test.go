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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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
		setupFunc  func(*config.Config) error
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
				APISource:      "https://github.com/googleapis/googleapis", // This will trigger the clone of googleapis
				APISourceDepth: 1,
				Repo:           newTestGitRepo(t).GetDir(),
				WorkRoot:       t.TempDir(),
				Image:          "gcr.io/test/test-image",
				CommandName:    generateCmdName,
			},
			wantErr:    true,
			wantErrMsg: "repository does not exist",
			setupFunc: func(cfg *config.Config) error {
				// The function will try to clone googleapis into the current work directory.
				// To make it fail, create a non-empty, non-git directory.
				googleapisDir := filepath.Join(cfg.WorkRoot, "googleapis")
				if err := os.MkdirAll(googleapisDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(googleapisDir, "some-file"), []byte("foo"), 0644); err != nil {
					return err
				}
				return nil
			},
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

			// custom setup
			if test.setupFunc != nil {
				if err := test.setupFunc(test.cfg); err != nil {
					t.Fatalf("error in setup %v", err)
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
		name                     string
		api                      string
		library                  string
		state                    *config.LibrarianState
		librarianConfig          *config.LibrarianConfig
		container                *mockContainerClient
		ghClient                 GitHubClient
		build                    bool
		forceShouldGenerateError bool
		wantErr                  bool
		wantErrMsg               string
		wantGenerateCalls        int
		wantBuildCalls           int
		wantConfigureCalls       int
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
			wantErrMsg: "all 1 libraries failed to generate (skipped: 1)",
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
		{
			name: "source_roots_have_same_global_files",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID: "some-library",
						SourceRoots: []string{
							"src/some/path",
							"one/global/example.txt",
						},
						APIs: []*config.API{
							{
								Path: "google/cloud/some",
							},
						},
					},
					{
						ID: "another-library",
						SourceRoots: []string{
							"src/another/path",
							"one/global/example.txt",
						},
						APIs: []*config.API{
							{
								Path: "google/cloud/another",
							},
						},
					},
				},
			},
			librarianConfig: &config.LibrarianConfig{
				GlobalFilesAllowlist: []*config.GlobalFile{
					{
						Path:        "one/global/example.txt",
						Permissions: "read-write",
					},
				},
			},
			container: &mockContainerClient{
				wantLibraryGen: true,
			},
			ghClient:           &mockGitHubClient{},
			build:              true,
			wantGenerateCalls:  2,
			wantBuildCalls:     2,
			wantConfigureCalls: 0,
		},
		// We only have one library to generate, and we force shouldGenerate
		// to fail by making the source repo's HeadHash function fail.
		// As this ends up being all the libraries, the overall result is an error.
		// (Forcing shouldGenerate to fail selectively would be very complicated.)
		{
			name: "shouldGenerate error",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []*config.API{{Path: "some/api"}},
						// We need the LastGeneratedCommit to force shouldGenerate
						// to ask the source repo for the head hash.
						LastGeneratedCommit: "LastGeneratedCommit",
					},
				},
			},
			forceShouldGenerateError: true,
			wantErr:                  true,
			wantErrMsg:               "all 1 libraries failed to generate",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := newTestGitRepoWithState(t, test.state)

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

			if test.forceShouldGenerateError {
				r.sourceRepo = &MockRepository{
					HeadHashError: errors.New("fail"),
				}
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

func TestGenerateSingleLibraryCommand(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		api        string
		library    string
		state      *config.LibrarianState
		container  *mockContainerClient
		ghClient   GitHubClient
		build      bool
		wantErr    bool
		wantErrMsg string
		wantPRType pullRequestType
	}{
		{
			name:    "onboard library returns pullRequestOnboard",
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
			ghClient:   &mockGitHubClient{},
			build:      true,
			wantPRType: pullRequestOnboard,
		},
		{
			name:    "generate existing library returns pullRequestGenerate",
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
			ghClient:   &mockGitHubClient{},
			build:      true,
			wantPRType: pullRequestGenerate,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := newTestGitRepoWithState(t, test.state)
			sourceRepo := newTestGitRepo(t)
			r := &generateRunner{
				api:             test.api,
				library:         test.library,
				build:           test.build,
				repo:            repo,
				sourceRepo:      sourceRepo,
				state:           test.state,
				containerClient: test.container,
				ghClient:        test.ghClient,
				workRoot:        t.TempDir(),
			}

			// Create a service config in api path.
			if test.api != "" {
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
			}

			status, err := r.generateSingleLibrary(context.Background(), r.library, r.workRoot)
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
			if status.prType != test.wantPRType {
				t.Errorf("generateSingleLibrary() prType = %v, want %v", status.prType, test.wantPRType)
			}
		})
	}
}

func TestGetExistingSrc(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		libraryID string
		paths     []string
		want      []string
	}{
		{
			name:      "all_source_paths_existed",
			libraryID: "some-library",
			paths: []string{
				"a/path",
				"another/path",
			},
			want: []string{
				"a/path",
				"another/path",
			},
		},
		{
			name:      "one_source_paths_existed",
			libraryID: "some-library",
			paths: []string{
				"a/path",
			},
			want: []string{
				"a/path",
			},
		},
		{
			name:      "no_source_paths_existed",
			libraryID: "some-library",
			want:      nil,
		},
		{
			name:      "no_library_existed",
			libraryID: "another-library",
			want:      nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repo := newTestGitRepo(t)
			state := &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "some-library",
						SourceRoots: []string{
							"a/path",
							"another/path",
						},
					},
				},
			}
			for _, path := range test.paths {
				relPath := filepath.Join(repo.GetDir(), path)
				if err := os.MkdirAll(relPath, 0755); err != nil {
					t.Fatal(err)
				}
			}

			r := &generateRunner{
				repo:  repo,
				state: state,
			}

			got := r.getExistingSrc(test.libraryID)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("getExistingSrc() mismatch (-want +got):%s", diff)
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

func TestShouldGenerate(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name              string
		config            *config.LibrarianConfig
		state             *config.LibrarianState
		generateUnchanged bool
		sourceRepo        gitrepo.Repository
		libraryIDToTest   string
		want              bool
		wantErr           bool
	}{
		// Tests that don't get as far as checking for hashes.
		// (The mock repo will fail if we do get that far.)
		{
			name: "generation blocked",
			config: &config.LibrarianConfig{
				Libraries: []*config.LibraryConfig{
					{
						LibraryID:       "TestLibrary",
						GenerateBlocked: true,
					},
				},
			},
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:                  "TestLibrary",
						APIs:                []*config.API{{Path: "google/cloud/test"}},
						LastGeneratedCommit: "LastGeneratedHash",
					},
				},
			},
			sourceRepo: &MockRepository{
				HeadHashError: errors.New("Shouldn't get as far as checking head"),
			},
			libraryIDToTest: "TestLibrary",
			want:            false,
		},
		{
			name: "library has no APIs",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "TestLibrary",
						// This may be present even if it's meaningless.
						LastGeneratedCommit: "LastGeneratedCommit",
					},
				},
			},
			sourceRepo: &MockRepository{
				HeadHashError: errors.New("Shouldn't get as far as checking head"),
			},
			libraryIDToTest: "TestLibrary",
			want:            false,
		},
		{
			name: "generateUnchanged specified",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:                  "TestLibrary",
						APIs:                []*config.API{{Path: "google/cloud/test"}},
						LastGeneratedCommit: "LastGeneratedCommit",
					},
				},
			},
			generateUnchanged: true,
			sourceRepo: &MockRepository{
				HeadHashError: errors.New("Shouldn't get as far as checking head"),
			},
			libraryIDToTest: "TestLibrary",
			want:            true,
		},
		{
			name: "no LastGeneratedCommit",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:   "TestLibrary",
						APIs: []*config.API{{Path: "google/cloud/test"}},
					},
				},
			},
			sourceRepo: &MockRepository{
				HeadHashError: errors.New("Shouldn't get as far as checking head"),
			},
			libraryIDToTest: "TestLibrary",
			want:            true,
		},
		{
			name: "error from HeadHash",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:                  "TestLibrary",
						APIs:                []*config.API{{Path: "google/cloud/test"}},
						LastGeneratedCommit: "LastGeneratedCommit",
					},
				},
			},
			sourceRepo: &MockRepository{
				HeadHashError: errors.New("Can't get head commit"),
			},
			libraryIDToTest: "TestLibrary",
			wantErr:         true,
		},
		// Tests that do perform hash checking.
		{
			name: "error from GetHashForPath",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:                  "TestLibrary",
						APIs:                []*config.API{{Path: "google/cloud/test"}},
						LastGeneratedCommit: "LastGeneratedCommit",
					},
				},
			},
			sourceRepo: &MockRepository{
				HeadHashValue:       "HeadCommit",
				GetHashForPathError: errors.New("Can't get hash for path"),
			},
			libraryIDToTest: "TestLibrary",
			wantErr:         true,
		},
		{
			name: "config present but generation not blocked",
			config: &config.LibrarianConfig{
				Libraries: []*config.LibraryConfig{
					{
						LibraryID:       "OtherLibrary",
						GenerateBlocked: true,
					},
					{
						LibraryID: "TestLibrary",
						// Just to have some reason to make it configured...
						ReleaseBlocked: true,
					}},
			},
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:                  "TestLibrary",
						APIs:                []*config.API{{Path: "google/cloud/test"}},
						LastGeneratedCommit: "LastGeneratedCommit",
					},
				},
			},
			sourceRepo: &MockRepository{
				HeadHashValue: "HeadCommit",
				GetHashForPathValue: map[string]string{
					"LastGeneratedCommit:google/cloud/test": "hash1",
					"HeadCommit:google/cloud/test":          "hash2",
				},
			},
			libraryIDToTest: "TestLibrary",
			want:            true,
		},
		{
			name: "API hasn't changed",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:                  "TestLibrary",
						APIs:                []*config.API{{Path: "google/cloud/test"}},
						LastGeneratedCommit: "LastGeneratedCommit",
					},
				},
			},
			sourceRepo: &MockRepository{
				HeadHashValue: "HeadCommit",
				GetHashForPathValue: map[string]string{
					"LastGeneratedCommit:google/cloud/test": "hash",
					"HeadCommit:google/cloud/test":          "hash",
				},
			},
			libraryIDToTest: "TestLibrary",
			want:            false,
		},
		{
			name: "one API hasn't changed, one has",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "TestLibrary",
						APIs: []*config.API{
							{Path: "google/cloud/test1"},
							{Path: "google/cloud/test2"},
						},
						LastGeneratedCommit: "LastGeneratedCommit",
					},
				},
			},
			sourceRepo: &MockRepository{
				HeadHashValue: "HeadCommit",
				GetHashForPathValue: map[string]string{
					"LastGeneratedCommit:google/cloud/test1": "hash1",
					"HeadCommit:google/cloud/test1":          "hash1",
					"LastGeneratedCommit:google/cloud/test2": "hash2a",
					"HeadCommit:google/cloud/test2":          "hash2b",
				},
			},
			libraryIDToTest: "TestLibrary",
			want:            true,
		},
		{
			name: "second call to GetHashForPath fails",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:                  "TestLibrary",
						APIs:                []*config.API{{Path: "google/cloud/test"}},
						LastGeneratedCommit: "LastGeneratedCommit",
					},
				},
			},
			sourceRepo: &MockRepository{
				HeadHashValue: "HeadCommit",
				GetHashForPathValue: map[string]string{
					"LastGeneratedCommit:google/cloud/test": "hash",
					// Entry which deliberately returns an error
					"HeadCommit:google/cloud/test": "error",
				},
			},
			libraryIDToTest: "TestLibrary",
			wantErr:         true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := &generateRunner{
				generateUnchanged: test.generateUnchanged,
				librarianConfig:   test.config,
				state:             test.state,
				sourceRepo:        test.sourceRepo,
			}
			library := test.state.LibraryByID(test.libraryIDToTest)
			got, err := r.shouldGenerate(library)
			if test.wantErr != (err != nil) {
				t.Fatalf("shouldGenerate() error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Errorf("shouldGenerate() got = %v, want %v", got, test.want)
			}
		})
	}
}

func TestAddAPIToLibrary(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name          string
		initialState  *config.LibrarianState
		libraryID     string
		apiPath       string
		expectedState *config.LibrarianState
	}{
		{
			name: "add api to existing library",
			initialState: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "lib1",
						APIs: []*config.API{
							{Path: "api1"},
						},
					},
				},
			},
			libraryID: "lib1",
			apiPath:   "api2",
			expectedState: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "lib1",
						APIs: []*config.API{
							{Path: "api1"},
							{Path: "api2", Status: config.StatusNew},
						},
					},
				},
			},
		},
		{
			name: "add api to new library",
			initialState: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "lib1",
						APIs: []*config.API{
							{Path: "api1"},
						},
					},
				},
			},
			libraryID: "lib2",
			apiPath:   "api2",
			expectedState: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "lib1",
						APIs: []*config.API{
							{Path: "api1"},
						},
					},
					{
						ID: "lib2",
						APIs: []*config.API{
							{Path: "api2", Status: config.StatusNew},
						},
					},
				},
			},
		},
		{
			name: "add existing api to existing library",
			initialState: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "lib1",
						APIs: []*config.API{
							{Path: "api1"},
						},
					},
				},
			},
			libraryID: "lib1",
			apiPath:   "api1",
			expectedState: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "lib1",
						APIs: []*config.API{
							{Path: "api1"},
						},
					},
				},
			},
		},
		{
			name:         "add api to empty state",
			initialState: &config.LibrarianState{},
			libraryID:    "lib1",
			apiPath:      "api1",
			expectedState: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "lib1",
						APIs: []*config.API{
							{Path: "api1", Status: config.StatusNew},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			addAPIToLibrary(tc.initialState, tc.libraryID, tc.apiPath)
			if diff := cmp.Diff(tc.expectedState, tc.initialState); diff != "" {
				t.Errorf("addAPIToLibrary() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNeedsConfigure(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		api     string
		library string
		state   *config.LibrarianState
		want    bool
	}{
		{
			name:    "api and library set, library does not exist",
			api:     "some/api",
			library: "some-library",
			state:   &config.LibrarianState{},
			want:    true,
		},
		{
			name:    "api and library set library exists no api path in state yaml",
			api:     "some/api",
			library: "some-library",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "some-library",
					},
				},
			},
			want: true,
		},
		{
			name:    "api and library set library exists different api path in state yaml",
			api:     "some/api",
			library: "some-library",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "some-library",
						APIs: []*config.API{
							{Path: "another/api"},
						},
					},
				},
			},
			want: true,
		},
		{
			name:    "api not set",
			api:     "",
			library: "some-library",
			state:   &config.LibrarianState{},
			want:    false,
		},
		{
			name:    "library not set",
			api:     "some/api",
			library: "",
			state:   &config.LibrarianState{},
			want:    false,
		},
		{
			name:    "api and library not set",
			api:     "",
			library: "",
			state:   &config.LibrarianState{},
			want:    false,
		},
		{
			name:    "api and library set, library and api exist",
			api:     "some/api",
			library: "some-library",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "some-library",
						APIs: []*config.API{
							{Path: "some/api"},
						},
					},
				},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &generateRunner{
				api:     tc.api,
				library: tc.library,
				state:   tc.state,
			}
			got := r.needsConfigure()
			if got != tc.want {
				t.Errorf("needsConfigure() = %v, want %v", got, tc.want)
			}
		})
	}
}
