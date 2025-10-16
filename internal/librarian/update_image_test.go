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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

func TestNewUpdateImageRunner(t *testing.T) {
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
				CommandName: updateImageCmdName,
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
				CommandName: updateImageCmdName,
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
				CommandName: updateImageCmdName,
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
				CommandName: updateImageCmdName,
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
				CommandName:    updateImageCmdName,
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
				CommandName:    updateImageCmdName,
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
				CommandName: updateImageCmdName,
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

			r, err := newUpdateImageRunner(test.cfg)
			if test.wantErr {
				if err == nil {
					t.Fatalf("newUpdateImageRunner() error = %v, wantErr %v", err, test.wantErr)
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Fatalf("want error message: %s, got: %s", test.wantErrMsg, err.Error())
				}

				return
			}

			if err != nil {
				t.Fatalf("newUpdateImageRunner() got error: %v", err)
			}

			if r.branch == "" {
				t.Errorf("newUpdateImageRunner() branch is not set")
			}

			if r.ghClient == nil {
				t.Errorf("newUpdateImageRunner() ghClient is nil")
			}
			if r.containerClient == nil {
				t.Errorf("newUpdateImageRunner() containerClient is nil")
			}
			if r.repo == nil {
				t.Errorf("newUpdateImageRunner() repo is nil")
			}
			if r.sourceRepo == nil {
				t.Errorf("newUpdateImageRunner() sourceRepo is nil")
			}
		})
	}
}

func TestUpdateImageRunnerRun(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name                       string
		imagesClient               *mockImagesClient
		containerClient            *mockContainerClient
		ghClient                   *mockGitHubClient
		state                      *config.LibrarianState
		image                      string
		build                      bool
		commit                     bool
		push                       bool
		wantErr                    bool
		wantErrMsg                 string
		wantFindLatestCalls        int
		wantGenerateCalls          int
		wantBuildCalls             int
		wantCheckoutCalls          int
		wantCreatePullRequestCalls int
		wantCreateIssueCalls       int
		wantCommitMsg              string
		checkoutError              error
	}{
		{
			name:  "specific image",
			image: "some-image",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "lib1",
						APIs: []*config.API{{Path: "some/api1"}},
						SourceRoots: []string{
							"src/a",
						},
						LastGeneratedCommit: "abcd1234",
					},
				},
			},
			containerClient:     &mockContainerClient{},
			imagesClient:        &mockImagesClient{},
			ghClient:            &mockGitHubClient{},
			wantFindLatestCalls: 0,
			wantGenerateCalls:   1,
			wantBuildCalls:      0, // no -build flag
			wantCheckoutCalls:   1,
		},
		{
			name:  "no change image",
			image: "gcr.io/test/image:v1.2.3",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "lib1",
						APIs: []*config.API{{Path: "some/api1"}},
						SourceRoots: []string{
							"src/a",
						},
						LastGeneratedCommit: "abcd1234",
					},
				},
			},
			containerClient:     &mockContainerClient{},
			imagesClient:        &mockImagesClient{},
			ghClient:            &mockGitHubClient{},
			wantFindLatestCalls: 0,
			wantGenerateCalls:   0,
			wantBuildCalls:      0,
			wantCheckoutCalls:   0,
		},
		{
			name: "finds latest image",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "lib1",
						APIs: []*config.API{{Path: "some/api1"}},
						SourceRoots: []string{
							"src/a",
						},
						LastGeneratedCommit: "abcd1234",
					},
				},
			},
			containerClient: &mockContainerClient{},
			imagesClient: &mockImagesClient{
				latestImage: "gcr.io/test/image@sha256:abc123",
			},
			ghClient:            &mockGitHubClient{},
			wantFindLatestCalls: 1,
			wantGenerateCalls:   1,
			wantBuildCalls:      0, // no -build flag
			wantCheckoutCalls:   1,
		},
		{
			name: "finds image error",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "lib1",
						APIs: []*config.API{{Path: "some/api1"}},
						SourceRoots: []string{
							"src/a",
						},
						LastGeneratedCommit: "abcd1234",
					},
				},
			},
			containerClient: &mockContainerClient{},
			imagesClient: &mockImagesClient{
				err: fmt.Errorf("some lookup error"),
			},
			ghClient:            &mockGitHubClient{},
			wantFindLatestCalls: 1,
			wantGenerateCalls:   0,
			wantBuildCalls:      0,
			wantCheckoutCalls:   0,
			wantErr:             true,
			wantErrMsg:          "some lookup error",
		},
		{
			name: "runs build",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "lib1",
						APIs: []*config.API{{Path: "some/api1"}},
						SourceRoots: []string{
							"src/a",
						},
						LastGeneratedCommit: "abcd1234",
					},
				},
			},
			containerClient: &mockContainerClient{},
			imagesClient: &mockImagesClient{
				latestImage: "gcr.io/test/image@sha256:abc123",
			},
			ghClient:            &mockGitHubClient{},
			build:               true,
			wantFindLatestCalls: 1,
			wantGenerateCalls:   1,
			wantBuildCalls:      1,
			wantCheckoutCalls:   1,
		},
		{
			name: "updates multiple",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "lib1",
						APIs: []*config.API{{Path: "some/api1"}},
						SourceRoots: []string{
							"src/a",
						},
						LastGeneratedCommit: "abcd1234",
					},
					{
						ID:   "lib2",
						APIs: []*config.API{{Path: "some/api2"}},
						SourceRoots: []string{
							"src/b",
						},
						LastGeneratedCommit: "abcd1235",
					},
				},
			},
			containerClient: &mockContainerClient{},
			imagesClient: &mockImagesClient{
				latestImage: "gcr.io/test/image@sha256:abc123",
			},
			ghClient:            &mockGitHubClient{},
			build:               true,
			wantFindLatestCalls: 1,
			wantGenerateCalls:   2,
			wantBuildCalls:      2,
			wantCheckoutCalls:   2,
		},
		{
			name: "skips libraries without APIs",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "lib1",
						APIs: []*config.API{{Path: "some/api1"}},
						SourceRoots: []string{
							"src/a",
						},
						LastGeneratedCommit: "abcd1234",
					},
					{
						ID:   "lib2",
						APIs: []*config.API{},
						SourceRoots: []string{
							"src/b",
						},
						LastGeneratedCommit: "abcd1235",
					},
				},
			},
			containerClient: &mockContainerClient{},
			imagesClient: &mockImagesClient{
				latestImage: "gcr.io/test/image@sha256:abc123",
			},
			ghClient:            &mockGitHubClient{},
			build:               true,
			wantFindLatestCalls: 1,
			wantGenerateCalls:   1,
			wantBuildCalls:      1,
			wantCheckoutCalls:   1,
		},
		{
			name: "partial generate success",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "lib1",
						APIs: []*config.API{{Path: "some/api1"}},
						SourceRoots: []string{
							"src/a",
						},
						LastGeneratedCommit: "abcd1234",
					},
					{
						ID:   "lib2",
						APIs: []*config.API{{Path: "some/api2"}},
						SourceRoots: []string{
							"src/b",
						},
						LastGeneratedCommit: "abcd1235",
					},
				},
			},
			containerClient: &mockContainerClient{
				failGenerateForID: "lib1",
				generateErrForID:  fmt.Errorf("error generating lib1"),
			},
			imagesClient: &mockImagesClient{
				latestImage: "gcr.io/test/image@sha256:abc123",
			},
			ghClient:            &mockGitHubClient{},
			build:               true,
			wantFindLatestCalls: 1,
			wantGenerateCalls:   2,
			wantBuildCalls:      1, // build for failed generate should not run
			wantCheckoutCalls:   2,
		},
		{
			name: "partial build success",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "lib1",
						APIs: []*config.API{{Path: "some/api1"}},
						SourceRoots: []string{
							"src/a",
						},
						LastGeneratedCommit: "abcd1234",
					},
					{
						ID:   "lib2",
						APIs: []*config.API{{Path: "some/api2"}},
						SourceRoots: []string{
							"src/b",
						},
						LastGeneratedCommit: "abcd1235",
					},
				},
			},
			containerClient: &mockContainerClient{
				failBuildForID: "lib1",
				buildErrForID:  fmt.Errorf("error building lib1"),
			},
			imagesClient: &mockImagesClient{
				latestImage: "gcr.io/test/image@sha256:abc123",
			},
			ghClient:            &mockGitHubClient{},
			build:               true,
			wantFindLatestCalls: 1,
			wantGenerateCalls:   2,
			wantBuildCalls:      2,
			wantCheckoutCalls:   2,
		},
		{
			name: "checkout error",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "lib1",
						APIs: []*config.API{{Path: "some/api1"}},
						SourceRoots: []string{
							"src/a",
						},
						LastGeneratedCommit: "abcd1234",
					},
					{
						ID:   "lib2",
						APIs: []*config.API{{Path: "some/api2"}},
						SourceRoots: []string{
							"src/b",
						},
						LastGeneratedCommit: "abcd1235",
					},
				},
			},
			containerClient: &mockContainerClient{},
			imagesClient: &mockImagesClient{
				latestImage: "gcr.io/test/image@sha256:abc123",
			},
			ghClient:            &mockGitHubClient{},
			build:               true,
			wantFindLatestCalls: 1,
			wantGenerateCalls:   0,
			wantBuildCalls:      0,
			wantCheckoutCalls:   2,
			checkoutError:       fmt.Errorf("some checkout error"),
		},
		{
			name: "updates multiple with commit",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "lib1",
						APIs: []*config.API{{Path: "some/api1"}},
						SourceRoots: []string{
							"src/a",
						},
						LastGeneratedCommit: "abcd1234",
					},
					{
						ID:   "lib2",
						APIs: []*config.API{{Path: "some/api2"}},
						SourceRoots: []string{
							"src/b",
						},
						LastGeneratedCommit: "abcd1235",
					},
				},
			},
			containerClient: &mockContainerClient{},
			imagesClient: &mockImagesClient{
				latestImage: "gcr.io/test/image@sha256:abc123",
			},
			ghClient:            &mockGitHubClient{},
			build:               true,
			commit:              true,
			wantFindLatestCalls: 1,
			wantGenerateCalls:   2,
			wantBuildCalls:      2,
			wantCheckoutCalls:   2,
			wantCommitMsg:       "feat: update image to gcr.io/test/image@sha256:abc123",
		},
		{
			name: "push failure",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "lib1",
						APIs: []*config.API{{Path: "some/api1"}},
						SourceRoots: []string{
							"src/a",
						},
						LastGeneratedCommit: "abcd1234",
					},
					{
						ID:   "lib2",
						APIs: []*config.API{{Path: "some/api2"}},
						SourceRoots: []string{
							"src/b",
						},
						LastGeneratedCommit: "abcd1235",
					},
				},
			},
			containerClient: &mockContainerClient{},
			imagesClient: &mockImagesClient{
				latestImage: "gcr.io/test/image@sha256:abc123",
			},
			ghClient: &mockGitHubClient{
				createPullRequestErr: fmt.Errorf("some API error"),
			},
			build:                      true,
			commit:                     true,
			push:                       true,
			wantFindLatestCalls:        1,
			wantGenerateCalls:          2,
			wantBuildCalls:             2,
			wantCheckoutCalls:          2,
			wantCreatePullRequestCalls: 1,
			wantCommitMsg:              "feat: update image to gcr.io/test/image@sha256:abc123",
			wantErr:                    true,
			wantErrMsg:                 "some API error",
		},
		{
			name: "updates multiple with push",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "lib1",
						APIs: []*config.API{{Path: "some/api1"}},
						SourceRoots: []string{
							"src/a",
						},
						LastGeneratedCommit: "abcd1234",
					},
					{
						ID:   "lib2",
						APIs: []*config.API{{Path: "some/api2"}},
						SourceRoots: []string{
							"src/b",
						},
						LastGeneratedCommit: "abcd1235",
					},
				},
			},
			containerClient: &mockContainerClient{},
			imagesClient: &mockImagesClient{
				latestImage: "gcr.io/test/image@sha256:abc123",
			},
			ghClient: &mockGitHubClient{
				createdPR: &github.PullRequestMetadata{
					Number: 1234,
					Repo: &github.Repository{
						Owner: "googleapis",
						Name:  "google-cloud-go",
					},
				},
			},
			build:                      true,
			commit:                     true,
			push:                       true,
			wantFindLatestCalls:        1,
			wantGenerateCalls:          2,
			wantBuildCalls:             2,
			wantCheckoutCalls:          2,
			wantCreatePullRequestCalls: 1,
			wantCommitMsg:              "feat: update image to gcr.io/test/image@sha256:abc123",
		},
		{
			name: "partial updates with push",
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "lib1",
						APIs: []*config.API{{Path: "some/api1"}},
						SourceRoots: []string{
							"src/a",
						},
						LastGeneratedCommit: "abcd1234",
					},
					{
						ID:   "lib2",
						APIs: []*config.API{{Path: "some/api2"}},
						SourceRoots: []string{
							"src/b",
						},
						LastGeneratedCommit: "abcd1235",
					},
				},
			},
			containerClient: &mockContainerClient{
				failBuildForID: "lib1",
				buildErrForID:  fmt.Errorf("error building lib1"),
			},
			imagesClient: &mockImagesClient{
				latestImage: "gcr.io/test/image@sha256:abc123",
			},
			ghClient: &mockGitHubClient{
				createdPR: &github.PullRequestMetadata{
					Number: 1234,
					Repo: &github.Repository{
						Owner: "googleapis",
						Name:  "google-cloud-go",
					},
				},
			},
			build:                      true,
			commit:                     true,
			push:                       true,
			wantFindLatestCalls:        1,
			wantGenerateCalls:          2,
			wantBuildCalls:             2,
			wantCheckoutCalls:          2,
			wantCreatePullRequestCalls: 1,
			wantCreateIssueCalls:       1,
			wantCommitMsg:              "feat: update image to gcr.io/test/image@sha256:abc123",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testRepo := newTestGitRepoWithState(t, test.state)
			repo := &MockRepository{
				Dir: testRepo.GetDir(),
				RemotesValue: []*gitrepo.Remote{
					{
						Name: "origin",
						URLs: []string{"https://github.com/googleapis/google-cloud-go.git"},
					},
				},
			}
			sourceRepo := &MockRepository{
				CheckoutError: test.checkoutError,
			}
			r := &updateImageRunner{
				branch:          "main",
				build:           test.build,
				commit:          test.commit,
				push:            test.push,
				image:           test.image,
				containerClient: test.containerClient,
				imagesClient:    test.imagesClient,
				ghClient:        test.ghClient,
				state:           test.state,
				workRoot:        t.TempDir(),
				repo:            repo,
				sourceRepo:      sourceRepo,
			}

			err := r.run(t.Context())

			if test.wantErr {
				if err == nil {
					t.Fatalf("%s should return error", test.name)
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message %s, got %s", test.wantErrMsg, err.Error())
				}
				return
			} else {
				if err != nil {
					t.Fatal(err)
				}
			}

			if diff := cmp.Diff(test.wantGenerateCalls, test.containerClient.generateCalls); diff != "" {
				t.Errorf("%s: run() generateCalls mismatch (-want +got):%s", test.name, diff)
			}
			if diff := cmp.Diff(test.wantBuildCalls, test.containerClient.buildCalls); diff != "" {
				t.Errorf("%s: run() buildCalls mismatch (-want +got):%s", test.name, diff)
			}
			if diff := cmp.Diff(test.wantFindLatestCalls, test.imagesClient.findLatestCalls); diff != "" {
				t.Errorf("%s: run() findLatestCalls mismatch (-want +got):%s", test.name, diff)
			}
			if diff := cmp.Diff(test.wantCheckoutCalls, sourceRepo.CheckoutCalls); diff != "" {
				t.Errorf("%s: run() checkoutCalls mismatch (-want +got):%s", test.name, diff)
			}
			if diff := cmp.Diff(test.wantCreatePullRequestCalls, test.ghClient.createPullRequestCalls); diff != "" {
				t.Errorf("%s: run() createPullRequestCalls mismatch (-want +got):%s", test.name, diff)
			}
			if diff := cmp.Diff(test.wantCreateIssueCalls, test.ghClient.createIssueCalls); diff != "" {
				t.Errorf("%s: run() createIssueCalls mismatch (-want +got):%s", test.name, diff)
			}

			if test.wantCommitMsg != "" {
				if diff := cmp.Diff(test.wantCommitMsg, repo.LastCommitMessage); diff != "" {
					t.Errorf("%s: run() commit message mismatch (-want +got):%s", test.name, diff)
				}
			}
		})
	}
}
