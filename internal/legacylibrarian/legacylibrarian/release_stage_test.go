// Copyright 2025 Google LLC
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

package legacylibrarian

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacygitrepo"
	"gopkg.in/yaml.v3"
)

func TestNewStageRunner(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		cfg        *legacyconfig.Config
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "valid config",
			cfg: &legacyconfig.Config{
				API:       "some/api",
				APISource: newTestGitRepo(t).GetDir(),
				Repo:      newTestGitRepo(t).GetDir(),
				WorkRoot:  t.TempDir(),
				Image:     "gcr.io/test/test-image",
			},
		},
		{
			name: "invalid config",
			cfg: &legacyconfig.Config{
				APISource: newTestGitRepo(t).GetDir(),
			},
			wantErr:    true,
			wantErrMsg: "failed to create stage runner",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := newStageRunner(test.cfg)
			if test.wantErr {
				if err == nil {
					t.Fatal("newStageRunner() should return error")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %q, got %q", test.wantErrMsg, err.Error())
				}

				return
			}
			if err != nil {
				t.Errorf("newStageRunner() = %v, want nil", err)
			}
		})
	}
}

func TestStageRun(t *testing.T) {
	t.Parallel()

	mockRepoWithReleasableUnit := &MockRepository{
		Dir: t.TempDir(),
		RemotesValue: []*legacygitrepo.Remote{
			{
				Name: "origin",
				URLs: []string{"https://github.com/googleapis/librarian.git"},
			},
		},
		ChangedFilesInCommitValue: []string{"dir1/file.txt"},
		GetCommitsForPathsSinceTagValue: []*legacygitrepo.Commit{
			{
				Message: "feat: a feature",
			},
		},
	}
	for _, test := range []struct {
		name             string
		containerClient  *mockContainerClient
		dockerStageCalls int
		// TODO: Pass all setup fields to the setupRunner func
		setupRunner func(containerClient *mockContainerClient) *stageRunner
		files       map[string]string
		want        *legacyconfig.LibrarianState
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name:             "run release stage command for all libraries, update librarian state",
			containerClient:  &mockContainerClient{},
			dockerStageCalls: 1,
			setupRunner: func(containerClient *mockContainerClient) *stageRunner {
				return &stageRunner{
					workRoot:        t.TempDir(),
					containerClient: containerClient,
					state: &legacyconfig.LibrarianState{
						Libraries: []*legacyconfig.LibraryState{
							{
								ID:      "another-example-id",
								Version: "1.0.0",
								SourceRoots: []string{
									"dir3",
									"dir4",
								},
								RemoveRegex: []string{
									"dir3",
									"dir4",
								},
							},
							{
								ID:      "example-id",
								Version: "2.0.0",
								SourceRoots: []string{
									"dir1",
									"dir2",
								},
								RemoveRegex: []string{
									"dir1",
									"dir2",
								},
							},
						},
					},
					repo: &MockRepository{
						Dir: t.TempDir(),
						RemotesValue: []*legacygitrepo.Remote{
							{
								Name: "origin",
								URLs: []string{"https://github.com/googleapis/librarian.git"},
							},
						},
						GetCommitsForPathsSinceTagValueByTag: map[string][]*legacygitrepo.Commit{
							"another-example-id-1.0.0": {
								{
									Hash:    plumbing.NewHash("123456"),
									Message: "feat: another new feature",
								},
							},
							"example-id-2.0.0": {
								{
									Hash:    plumbing.NewHash("abcdefg"),
									Message: "feat: a new feature",
								},
							},
						},
						ChangedFilesInCommitValueByHash: map[string][]string{
							plumbing.NewHash("123456").String(): {
								"dir3/file3.txt",
								"dir4/file4.txt",
							},
							plumbing.NewHash("abcdefg").String(): {
								"dir1/file1.txt",
								"dir2/file2.txt",
							},
						},
					},
					librarianConfig: &legacyconfig.LibrarianConfig{},
				}
			},
			files: map[string]string{
				"file1.txt":      "",
				"dir1/file1.txt": "",
				"dir2/file2.txt": "",
				"dir3/file3.txt": "",
				"dir4/file4.txt": "",
			},
			want: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:      "another-example-id",
						Version: "1.1.0", // version is bumped.
						APIs:    []*legacyconfig.API{},
						SourceRoots: []string{
							"dir3",
							"dir4",
						},
						PreserveRegex: []string{},
						RemoveRegex: []string{
							"dir3",
							"dir4",
						},
					},
					{
						ID:      "example-id",
						Version: "2.1.0", // version is bumped.
						APIs:    []*legacyconfig.API{},
						SourceRoots: []string{
							"dir1",
							"dir2",
						},
						PreserveRegex: []string{},
						RemoveRegex: []string{
							"dir1",
							"dir2",
						},
					},
				},
			},
		},
		{
			name:             "run release stage command for one library (library id in cfg)",
			containerClient:  &mockContainerClient{},
			dockerStageCalls: 1,
			setupRunner: func(containerClient *mockContainerClient) *stageRunner {
				return &stageRunner{
					workRoot:        t.TempDir(),
					containerClient: containerClient,
					library:         "example-id",
					state: &legacyconfig.LibrarianState{
						Libraries: []*legacyconfig.LibraryState{
							{
								Version: "1.0.0",
								ID:      "another-example-id",
								SourceRoots: []string{
									"dir3",
									"dir4",
								},
							},
							{
								Version: "2.0.0",
								ID:      "example-id",
								SourceRoots: []string{
									"dir1",
									"dir2",
								},
								RemoveRegex: []string{
									"dir1",
									"dir2",
								},
							},
						},
					},
					repo:            mockRepoWithReleasableUnit,
					librarianConfig: &legacyconfig.LibrarianConfig{},
				}
			},
			files: map[string]string{
				"file1.txt":      "",
				"dir1/file1.txt": "",
				"dir2/file2.txt": "",
			},
			want: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
					{
						Version: "1.0.0",
						ID:      "another-example-id",
						APIs:    []*legacyconfig.API{},
						SourceRoots: []string{
							"dir3",
							"dir4",
						},
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
					},
					{
						Version: "2.1.0", // Version is bumped only for library specified
						ID:      "example-id",
						APIs:    []*legacyconfig.API{},
						SourceRoots: []string{
							"dir1",
							"dir2",
						},
						PreserveRegex: []string{},
						RemoveRegex: []string{
							"dir1",
							"dir2",
						},
					},
				},
			},
		},
		{
			name:             "run release stage command for libraries have the same global files in src roots",
			containerClient:  &mockContainerClient{},
			dockerStageCalls: 1,
			setupRunner: func(containerClient *mockContainerClient) *stageRunner {
				return &stageRunner{
					workRoot:        t.TempDir(),
					containerClient: containerClient,
					state: &legacyconfig.LibrarianState{
						Libraries: []*legacyconfig.LibraryState{
							{
								ID:      "another-example-id",
								Version: "1.0.0",
								SourceRoots: []string{
									"dir3",
									"one/global/example.txt",
								},
								RemoveRegex: []string{
									"dir3",
								},
							},
							{
								ID:      "example-id",
								Version: "2.0.0",
								SourceRoots: []string{
									"dir1",
									"one/global/example.txt",
								},
								RemoveRegex: []string{
									"dir1",
								},
							},
						},
					},
					repo: &MockRepository{
						Dir: t.TempDir(),
						RemotesValue: []*legacygitrepo.Remote{
							{
								Name: "origin",
								URLs: []string{"https://github.com/googleapis/librarian.git"},
							},
						},
						GetCommitsForPathsSinceTagValueByTag: map[string][]*legacygitrepo.Commit{
							"another-example-id-1.0.0": {
								{
									Hash:    plumbing.NewHash("123456"),
									Message: "feat: bump version",
								},
							},
							"example-id-2.0.0": {
								{
									Hash:    plumbing.NewHash("123456"),
									Message: "feat: bump version",
								},
							},
						},
						ChangedFilesInCommitValueByHash: map[string][]string{
							plumbing.NewHash("123456").String(): {
								"one/global/example.txt",
							},
						},
					},
					librarianConfig: &legacyconfig.LibrarianConfig{
						GlobalFilesAllowlist: []*legacyconfig.GlobalFile{
							{
								Path:        "one/global/example.txt",
								Permissions: "read-write",
							},
						},
					},
				}
			},
			files: map[string]string{
				"one/global/example.txt": "",
				"dir1/file1.txt":         "",
				"dir3/file3.txt":         "",
			},
			want: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:      "another-example-id",
						Version: "1.1.0", // version is bumped.
						APIs:    []*legacyconfig.API{},
						SourceRoots: []string{
							"dir3",
							"one/global/example.txt",
						},
						PreserveRegex: []string{},
						RemoveRegex: []string{
							"dir3",
						},
					},
					{
						ID:      "example-id",
						Version: "2.1.0", // version is bumped.
						APIs:    []*legacyconfig.API{},
						SourceRoots: []string{
							"dir1",
							"one/global/example.txt",
						},
						PreserveRegex: []string{},
						RemoveRegex: []string{
							"dir1",
						},
					},
				},
			},
		},
		{
			name:             "run release stage command, skips blocked libraries!",
			containerClient:  &mockContainerClient{},
			dockerStageCalls: 1,
			setupRunner: func(containerClient *mockContainerClient) *stageRunner {
				return &stageRunner{
					workRoot:        t.TempDir(),
					containerClient: containerClient,
					state: &legacyconfig.LibrarianState{
						Libraries: []*legacyconfig.LibraryState{
							{
								ID:          "blocked-example-id",
								Version:     "1.0.0",
								SourceRoots: []string{"dir1"},
							},
							{
								ID:          "example-id",
								Version:     "2.0.0",
								SourceRoots: []string{"dir1"},
							},
						},
					},
					repo: &MockRepository{
						Dir: t.TempDir(),
						RemotesValue: []*legacygitrepo.Remote{
							{
								Name: "origin",
								URLs: []string{"https://github.com/googleapis/librarian.git"},
							},
						},
						GetCommitsForPathsSinceTagValueByTag: map[string][]*legacygitrepo.Commit{
							"blocked-example-id-1.0.0": {
								{
									Hash:    plumbing.NewHash("123456"),
									Message: "feat: another new feature",
								},
							},
							"example-id-2.0.0": {
								{
									Hash:    plumbing.NewHash("abcdefg"),
									Message: "feat: a new feature",
								},
							},
						},
						ChangedFilesInCommitValueByHash: map[string][]string{
							plumbing.NewHash("123456").String(): {
								"dir1/file1.txt",
							},
							plumbing.NewHash("abcdefg").String(): {
								"dir1/file2.txt",
							},
						},
					},
					librarianConfig: &legacyconfig.LibrarianConfig{
						Libraries: []*legacyconfig.LibraryConfig{
							{LibraryID: "blocked-example-id", ReleaseBlocked: true},
							{LibraryID: "example-id"},
						},
					},
				}
			},
			want: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:            "blocked-example-id",
						Version:       "1.0.0", // version is NOT bumped.
						APIs:          []*legacyconfig.API{},
						SourceRoots:   []string{"dir1"},
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
					},
					{
						ID:            "example-id",
						Version:       "2.1.0", // version is bumped.
						APIs:          []*legacyconfig.API{},
						SourceRoots:   []string{"dir1"},
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
					},
				},
			},
		},
		{
			name:             "run release stage command, does not skip blocked library if explicitly specified",
			containerClient:  &mockContainerClient{},
			dockerStageCalls: 1,
			setupRunner: func(containerClient *mockContainerClient) *stageRunner {
				return &stageRunner{
					workRoot:        t.TempDir(),
					containerClient: containerClient,
					// The library is explicitly specified.
					library: "blocked-example-id",
					state: &legacyconfig.LibrarianState{
						Libraries: []*legacyconfig.LibraryState{
							{
								ID:          "blocked-example-id",
								Version:     "1.0.0",
								SourceRoots: []string{"dir1"},
							},
						},
					},
					repo: &MockRepository{
						Dir: t.TempDir(),
						RemotesValue: []*legacygitrepo.Remote{
							{
								Name: "origin",
								URLs: []string{"https://github.com/googleapis/librarian.git"},
							},
						},
						GetCommitsForPathsSinceTagValueByTag: map[string][]*legacygitrepo.Commit{
							"blocked-example-id-1.0.0": {
								{
									Hash:    plumbing.NewHash("123456"),
									Message: "feat: another new feature",
								},
							},
						},
						ChangedFilesInCommitValueByHash: map[string][]string{
							plumbing.NewHash("123456").String(): {
								"dir1/file1.txt",
							},
						},
					},
					librarianConfig: &legacyconfig.LibrarianConfig{
						Libraries: []*legacyconfig.LibraryConfig{
							{LibraryID: "blocked-example-id", ReleaseBlocked: true},
						},
					},
				}
			},
			want: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:            "blocked-example-id",
						Version:       "1.1.0",
						APIs:          []*legacyconfig.API{},
						SourceRoots:   []string{"dir1"},
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
					},
				},
			},
		},
		{
			name:            "run release stage command for one invalid library (invalid library id in cfg)",
			containerClient: &mockContainerClient{},
			setupRunner: func(containerClient *mockContainerClient) *stageRunner {
				return &stageRunner{
					workRoot:        t.TempDir(),
					containerClient: containerClient,
					library:         "does-not-exist",
					state: &legacyconfig.LibrarianState{
						Libraries: []*legacyconfig.LibraryState{
							{
								ID: "another-example-id",
							},
							{
								ID: "example-id",
							},
						},
					},
					repo: &MockRepository{
						Dir: t.TempDir(),
					},
					librarianConfig: &legacyconfig.LibrarianConfig{},
				}
			},
			wantErr:    true,
			wantErrMsg: "unable to find library for release",
		},
		{
			name:             "run release stage command without librarian config (no legacyconfig.yaml file)",
			containerClient:  &mockContainerClient{},
			dockerStageCalls: 1,
			setupRunner: func(containerClient *mockContainerClient) *stageRunner {
				return &stageRunner{
					workRoot:        t.TempDir(),
					containerClient: containerClient,
					library:         "example-id",
					state: &legacyconfig.LibrarianState{
						Libraries: []*legacyconfig.LibraryState{
							{
								Version: "1.0.0",
								ID:      "another-example-id",
								SourceRoots: []string{
									"dir3",
									"dir4",
								},
							},
							{
								Version: "2.0.0",
								ID:      "example-id",
								SourceRoots: []string{
									"dir1",
									"dir2",
								},
								RemoveRegex: []string{
									"dir1",
									"dir2",
								},
							},
						},
					},
					repo: mockRepoWithReleasableUnit,
				}
			},
			files: map[string]string{
				"file1.txt":      "",
				"dir1/file1.txt": "",
				"dir2/file2.txt": "",
			},
			want: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
					{
						Version: "1.0.0",
						ID:      "another-example-id",
						APIs:    []*legacyconfig.API{},
						SourceRoots: []string{
							"dir3",
							"dir4",
						},
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
					},
					{
						Version: "2.1.0",
						ID:      "example-id",
						APIs:    []*legacyconfig.API{},
						SourceRoots: []string{
							"dir1",
							"dir2",
						},
						PreserveRegex: []string{},
						RemoveRegex: []string{
							"dir1",
							"dir2",
						},
					},
				},
			},
		},
		{
			name: "docker command returns error",
			containerClient: &mockContainerClient{
				stageErr: errors.New("simulated init error"),
			},
			// error occurred inside the docker container, there was a single request made to the container
			dockerStageCalls: 1,
			setupRunner: func(containerClient *mockContainerClient) *stageRunner {
				return &stageRunner{
					workRoot:        t.TempDir(),
					containerClient: containerClient,
					state: &legacyconfig.LibrarianState{
						Libraries: []*legacyconfig.LibraryState{
							{
								Version:     "1.0.0",
								ID:          "example-id",
								SourceRoots: []string{"dir1"},
							},
						},
					},
					repo:            mockRepoWithReleasableUnit,
					librarianConfig: &legacyconfig.LibrarianConfig{},
				}
			},
			wantErr:    true,
			wantErrMsg: "simulated init error",
		},
		{
			name: "release response from container contains error message",
			containerClient: &mockContainerClient{
				wantErrorMsg: true,
			},
			// error reported from the docker container, there was a single request made to the container
			dockerStageCalls: 1,
			setupRunner: func(containerClient *mockContainerClient) *stageRunner {
				return &stageRunner{
					workRoot:        t.TempDir(),
					containerClient: containerClient,
					state: &legacyconfig.LibrarianState{
						Libraries: []*legacyconfig.LibraryState{
							{
								Version:     "1.0.0",
								ID:          "example-id",
								SourceRoots: []string{"dir1"},
							},
						},
					},
					repo:            mockRepoWithReleasableUnit,
					librarianConfig: &legacyconfig.LibrarianConfig{},
				}
			},
			wantErr:    true,
			wantErrMsg: "failed with error message: simulated error message",
		},
		{
			name:            "invalid work root",
			containerClient: &mockContainerClient{},
			setupRunner: func(containerClient *mockContainerClient) *stageRunner {
				return &stageRunner{
					workRoot:        "/invalid/path",
					containerClient: containerClient,
					repo: &MockRepository{
						Dir: t.TempDir(),
					},
				}
			},
			wantErr:    true,
			wantErrMsg: "failed to create output dir",
		},
		{
			name:            "failed to get changes from repo when releasing one library",
			containerClient: &mockContainerClient{},
			setupRunner: func(containerClient *mockContainerClient) *stageRunner {
				return &stageRunner{
					workRoot:        t.TempDir(),
					containerClient: containerClient,
					library:         "example-id",
					state: &legacyconfig.LibrarianState{
						Libraries: []*legacyconfig.LibraryState{
							{
								ID:          "example-id",
								SourceRoots: []string{"dir1"},
							},
						},
					},
					repo: &MockRepository{
						Dir:                             t.TempDir(),
						GetCommitsForPathsSinceTagError: errors.New("simulated error when getting commits"),
					},
				}
			},
			wantErr:    true,
			wantErrMsg: "failed to fetch conventional commits for library",
		},
		{
			name:            "failed to get changes from repo when releasing multiple libraries",
			containerClient: &mockContainerClient{},
			setupRunner: func(containerClient *mockContainerClient) *stageRunner {
				return &stageRunner{
					workRoot:        t.TempDir(),
					containerClient: containerClient,
					state: &legacyconfig.LibrarianState{
						Libraries: []*legacyconfig.LibraryState{
							{
								ID:          "example-id",
								SourceRoots: []string{"dir1"},
							},
						},
					},
					repo: &MockRepository{
						Dir:                             t.TempDir(),
						GetCommitsForPathsSinceTagError: errors.New("simulated error when getting commits"),
					},
				}
			},
			wantErr:    true,
			wantErrMsg: "failed to fetch conventional commits for library",
		},
		{
			name:            "single library has no releasable units, no state change",
			containerClient: &mockContainerClient{},
			setupRunner: func(containerClient *mockContainerClient) *stageRunner {
				return &stageRunner{
					workRoot:        os.TempDir(),
					containerClient: containerClient,
					state: &legacyconfig.LibrarianState{
						Libraries: []*legacyconfig.LibraryState{
							{
								Version:     "1.0.0",
								SourceRoots: []string{"dir1"},
							},
						},
					},
					repo: &MockRepository{
						Dir: t.TempDir(),
						RemotesValue: []*legacygitrepo.Remote{
							{
								Name: "origin",
								URLs: []string{"https://github.com/googleapis/librarian.git"},
							},
						},
						ChangedFilesInCommitValue: []string{"file.txt"},
						GetCommitsForPathsSinceTagValue: []*legacygitrepo.Commit{
							{
								Message: "chore: not releasable",
							},
							{
								Message: "test: not releasable",
							},
							{
								Message: "build: not releasable",
							},
						},
					},
					ghClient:        &mockGitHubClient{},
					librarianConfig: &legacyconfig.LibrarianConfig{},
				}
			},
		},
		{
			name:             "release stage has multiple libraries but only one library has a releasable unit",
			containerClient:  &mockContainerClient{},
			dockerStageCalls: 1,
			setupRunner: func(containerClient *mockContainerClient) *stageRunner {
				return &stageRunner{
					workRoot:        os.TempDir(),
					containerClient: containerClient,
					state: &legacyconfig.LibrarianState{
						Libraries: []*legacyconfig.LibraryState{
							{
								Version:     "1.0.0",
								ID:          "another-example-id",
								SourceRoots: []string{"dir1"},
							},
							{
								Version:     "2.0.0",
								ID:          "example-id",
								SourceRoots: []string{"dir1"},
							},
						},
					},
					repo: &MockRepository{
						Dir: t.TempDir(),
						RemotesValue: []*legacygitrepo.Remote{
							{
								Name: "origin",
								URLs: []string{"https://github.com/googleapis/librarian.git"},
							},
						},
						ChangedFilesInCommitValue: []string{"dir1/file.txt"},
						GetCommitsForPathsSinceTagValueByTag: map[string][]*legacygitrepo.Commit{
							"another-example-id-1.0.0": {
								{
									Message: "chore: not releasable",
								},
							},
							"example-id-2.0.0": {
								{
									Message: "feat: a new feature",
								},
							},
						},
					},
					ghClient:        &mockGitHubClient{},
					librarianConfig: &legacyconfig.LibrarianConfig{},
				}
			},
			want: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:            "another-example-id",
						Version:       "1.0.0", // version is NOT bumped.
						APIs:          []*legacyconfig.API{},
						SourceRoots:   []string{"dir1"},
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
					},
					{
						ID:            "example-id",
						Version:       "2.1.0", // version is bumped.
						APIs:          []*legacyconfig.API{},
						SourceRoots:   []string{"dir1"},
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
					},
				},
			},
		},
		{
			name:             "inputted library does not have a releasable unit, version is inputted",
			containerClient:  &mockContainerClient{},
			dockerStageCalls: 1,
			setupRunner: func(containerClient *mockContainerClient) *stageRunner {
				return &stageRunner{
					workRoot:        os.TempDir(),
					containerClient: containerClient,
					library:         "another-example-id", // release only for this library
					libraryVersion:  "3.0.0",
					state: &legacyconfig.LibrarianState{
						Libraries: []*legacyconfig.LibraryState{
							{
								Version:     "1.0.0",
								ID:          "another-example-id",
								SourceRoots: []string{"dir1"},
							},
							{
								Version:     "2.0.0",
								ID:          "example-id",
								SourceRoots: []string{"dir1"},
							},
						},
					},
					repo: &MockRepository{
						Dir: t.TempDir(),
						RemotesValue: []*legacygitrepo.Remote{
							{
								Name: "origin",
								URLs: []string{"https://github.com/googleapis/librarian.git"},
							},
						},
						ChangedFilesInCommitValue: []string{"dir1/file.txt"},
						GetCommitsForPathsSinceTagValueByTag: map[string][]*legacygitrepo.Commit{
							"another-example-id-1.0.0": {
								{
									Message: "chore: not releasable",
								},
							},
							"example-id-2.0.0": {
								{
									Message: "feat: a new feature",
								},
							},
						},
					},
					ghClient:        &mockGitHubClient{},
					librarianConfig: &legacyconfig.LibrarianConfig{},
				}
			},
			want: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
					{
						Version:       "3.0.0",
						ID:            "another-example-id",
						APIs:          []*legacyconfig.API{},
						SourceRoots:   []string{"dir1"},
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
					},
					{
						Version:       "2.0.0",
						ID:            "example-id",
						APIs:          []*legacyconfig.API{},
						SourceRoots:   []string{"dir1"},
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
					},
				},
			},
		},
		{
			name:             "inputted library does not have a releasable unit, no version inputted",
			containerClient:  &mockContainerClient{},
			dockerStageCalls: 0, // version was not inputted, do not trigger a release
			setupRunner: func(containerClient *mockContainerClient) *stageRunner {
				return &stageRunner{
					workRoot:        os.TempDir(),
					containerClient: containerClient,
					library:         "another-example-id", // release only for this library
					state: &legacyconfig.LibrarianState{
						Libraries: []*legacyconfig.LibraryState{
							{
								Version:     "1.0.0",
								ID:          "another-example-id",
								SourceRoots: []string{"dir1"},
							},
							{
								Version:     "2.0.0",
								ID:          "example-id",
								SourceRoots: []string{"dir1"},
							},
						},
					},
					repo: &MockRepository{
						Dir:                       t.TempDir(),
						ChangedFilesInCommitValue: []string{"dir1/file.txt"},
						GetCommitsForPathsSinceTagValueByTag: map[string][]*legacygitrepo.Commit{
							"another-example-id-1.0.0": {
								{
									Message: "chore: not releasable",
								},
							},
							"example-id-2.0.0": {
								{
									Message: "feat: a new feature",
								},
							},
						},
					},
					ghClient:        &mockGitHubClient{},
					librarianConfig: &legacyconfig.LibrarianConfig{},
				}
			},
			wantErr:    true,
			wantErrMsg: "library does not have a releasable unit and will not be released. Use the version flag to force a release for",
		},
		{
			name:             "failed to commit and push",
			containerClient:  &mockContainerClient{},
			dockerStageCalls: 1,
			setupRunner: func(containerClient *mockContainerClient) *stageRunner {
				return &stageRunner{
					workRoot:        os.TempDir(),
					containerClient: containerClient,
					push:            true,
					state: &legacyconfig.LibrarianState{
						Libraries: []*legacyconfig.LibraryState{
							{
								Version:     "1.0.0",
								ID:          "example-id",
								SourceRoots: []string{"dir1"},
							},
						},
					},
					repo: &MockRepository{
						Dir: t.TempDir(),
						RemotesValue: []*legacygitrepo.Remote{
							{
								Name: "origin",
								URLs: []string{"https://github.com/googleapis/librarian.git"},
							},
						},
						ChangedFilesInCommitValue: []string{"dir1/file.txt"},
						GetCommitsForPathsSinceTagValue: []*legacygitrepo.Commit{
							{
								Message: "feat: a feature",
							},
						},
						// This AddAll is used in commitAndPush(). If commitAndPush() is invoked,
						// then this test should error out
						AddAllError: errors.New("unable to add all files"),
					},
					librarianConfig: &legacyconfig.LibrarianConfig{},
				}
			},
			wantErr:    true,
			wantErrMsg: "failed to commit and push",
		},
		{
			name:             "run release stage command with symbolic link",
			containerClient:  &mockContainerClient{},
			dockerStageCalls: 1,
			setupRunner: func(containerClient *mockContainerClient) *stageRunner {
				return &stageRunner{
					workRoot:        t.TempDir(),
					containerClient: containerClient,
					library:         "example-id",
					state: &legacyconfig.LibrarianState{
						Libraries: []*legacyconfig.LibraryState{
							{
								Version: "1.0.0",
								ID:      "example-id",
								SourceRoots: []string{
									"dir1",
								},
							},
						},
					},
					repo:            mockRepoWithReleasableUnit,
					librarianConfig: &legacyconfig.LibrarianConfig{},
				}
			},
			files: map[string]string{
				"dir1/file1.txt": "hello",
			},
			want: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:      "example-id",
						Version: "1.1.0",
						APIs:    []*legacyconfig.API{},
						SourceRoots: []string{
							"dir1",
						},
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			runner := test.setupRunner(test.containerClient)

			// Setup library files before running the command.
			repoDir := runner.repo.GetDir()
			outputDir := filepath.Join(runner.workRoot, "output")
			for path, content := range test.files {
				// Create files in repoDir and outputDir because the run() function
				// will copy files from outputDir to repoDir.
				for _, dir := range []string{repoDir, outputDir} {
					fullPath := filepath.Join(dir, path)
					if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
						t.Fatalf("os.MkdirAll() = %v", err)
					}
					if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
						t.Fatalf("os.WriteFile() = %v", err)
					}
				}
			}
			// Create a symbolic link for the test case with symbolic links.
			if test.name == "run release stage command with symbolic link" {
				if err := os.Symlink(filepath.Join(repoDir, "dir1/file1.txt"),
					filepath.Join(repoDir, "dir1/symlink.txt")); err != nil {
					t.Fatalf("os.Symlink() = %v", err)
				}
			}
			librarianDir := filepath.Join(repoDir, ".librarian")
			if err := os.MkdirAll(librarianDir, 0755); err != nil {
				t.Fatalf("os.MkdirAll() = %v", err)
			}

			// Create the librarian state file.
			stateFile := filepath.Join(repoDir, ".librarian/state.yaml")
			if err := os.MkdirAll(filepath.Dir(stateFile), 0755); err != nil {
				t.Fatalf("os.MkdirAll() = %v", err)
			}
			if err := os.WriteFile(stateFile, []byte{}, 0644); err != nil {
				t.Fatalf("os.WriteFile() = %v", err)
			}

			err := runner.run(t.Context())

			// Check how many times the docker container has been called. If a release is to proceed
			// we expect this to be 1. Otherwise, the dockerStageCalls should be 0. Run this check even
			// if there is an error that is wanted to ensure that a docker request is only made when
			// we want it to.
			if diff := cmp.Diff(test.dockerStageCalls, test.containerClient.stageCalls); diff != "" {
				t.Errorf("docker stage calls mismatch (-want +got):\n%s", diff)
			}

			if test.wantErr {
				if err == nil {
					t.Fatal("run() should return error")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %q, got %q", test.wantErrMsg, err.Error())
				}

				return
			}
			if err != nil {
				t.Errorf("run() failed: %s", err.Error())
			}

			// load librarian state from state.yaml, which should contain updated
			// library state.
			bytes, err := os.ReadFile(filepath.Join(repoDir, ".librarian/state.yaml"))
			if err != nil {
				t.Fatal(err)
			}

			// If there is no release triggered for any library, then the librarian state
			// is not written back. The `want` value for the librarian state is nil
			var got *legacyconfig.LibrarianState
			if err := yaml.Unmarshal(bytes, &got); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("state mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunStageCommand(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		state  *legacyconfig.LibrarianState
		config *legacyconfig.LibrarianConfig
		repo   legacygitrepo.Repository
		client ContainerClient
		want   *legacyconfig.LibrarianState
	}{
		{
			name: "global_file_commits_appear_in_multiple_libraries",
			state: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:      "another-example-id",
						Version: "1.0.0",
						SourceRoots: []string{
							"dir3",
							"one/global/example.txt",
						},
						RemoveRegex: []string{
							"dir3",
						},
					},
					{
						ID:      "example-id",
						Version: "2.0.0",
						SourceRoots: []string{
							"dir1",
							"one/global/example.txt",
						},
						RemoveRegex: []string{
							"dir1",
						},
					},
				},
			},
			config: &legacyconfig.LibrarianConfig{
				GlobalFilesAllowlist: []*legacyconfig.GlobalFile{
					{
						Path:        "one/global/example.txt",
						Permissions: "read-write",
					},
				},
			},
			repo: &MockRepository{
				Dir: t.TempDir(),
				RemotesValue: []*legacygitrepo.Remote{
					{
						Name: "origin",
						URLs: []string{"https://github.com/googleapis/librarian.git"},
					},
				},
				GetCommitsForPathsSinceTagValueByTag: map[string][]*legacygitrepo.Commit{
					"another-example-id-1.0.0": {
						{
							Hash:    plumbing.NewHash("123456"),
							Message: "feat: bump version",
						},
					},
					"example-id-2.0.0": {
						{
							Hash:    plumbing.NewHash("123456"),
							Message: "feat: bump version",
						},
					},
				},
				ChangedFilesInCommitValueByHash: map[string][]string{
					plumbing.NewHash("123456").String(): {
						"one/global/example.txt",
					},
				},
			},
			client: &mockContainerClient{},
			want: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:              "another-example-id",
						Version:         "1.1.0",
						PreviousVersion: "1.0.0",
						SourceRoots: []string{
							"dir3",
							"one/global/example.txt",
						},
						RemoveRegex: []string{
							"dir3",
						},
						Changes: []*legacyconfig.Commit{
							{
								Type:       "feat",
								Subject:    "bump version",
								CommitHash: "1234560000000000000000000000000000000000",
								LibraryIDs: "another-example-id",
							},
						},
						ReleaseTriggered: true,
					},
					{
						ID:              "example-id",
						Version:         "2.1.0",
						PreviousVersion: "2.0.0",
						SourceRoots: []string{
							"dir1",
							"one/global/example.txt",
						},
						RemoveRegex: []string{
							"dir1",
						},
						Changes: []*legacyconfig.Commit{
							{
								Type:       "feat",
								Subject:    "bump version",
								CommitHash: "1234560000000000000000000000000000000000",
								LibraryIDs: "example-id",
							},
						},
						ReleaseTriggered: true,
					},
				},
			},
		},
	} {
		output := t.TempDir()
		for _, globalFile := range test.config.GlobalFilesAllowlist {
			file := filepath.Join(output, globalFile.Path)
			if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, []byte("new content"), 0755); err != nil {
				t.Fatal(err)
			}
		}
		r := &stageRunner{
			repo:            test.repo,
			state:           test.state,
			librarianConfig: test.config,
			containerClient: test.client,
		}
		err := r.runStageCommand(t.Context(), output)
		if err != nil {
			t.Errorf("failed to run runStageCommand(): %q", err.Error())
			return
		}
		if diff := cmp.Diff(test.want, r.state); diff != "" {
			t.Errorf("commit filter mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestProcessLibrary(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name         string
		libraryState *legacyconfig.LibraryState
		repo         legacygitrepo.Repository
		wantErr      bool
		wantErrMsg   string
	}{
		{
			name: "failed to get commit history of one library",
			libraryState: &legacyconfig.LibraryState{
				ID:      "one-id",
				Version: "1.2.3",
			},
			repo: &MockRepository{
				GetCommitsForPathsSinceTagError: errors.New("simulated error when getting commits"),
			},
			wantErr:    true,
			wantErrMsg: "failed to fetch conventional commits for library",
		},
		{
			name: "does not search for git tag for 0.0.0 version",
			libraryState: &legacyconfig.LibraryState{
				ID:      "one-id",
				Version: "0.0.0",
			},
			repo: &MockRepository{},
		},
	} {
		state := &legacyconfig.LibrarianState{
			Libraries: []*legacyconfig.LibraryState{
				test.libraryState,
			},
		}
		r := &stageRunner{
			repo:  test.repo,
			state: state,
		}
		err := r.processLibrary(test.libraryState)
		if test.wantErr {
			if err == nil {
				t.Fatal("processLibrary() should return error")
			}
			if !strings.Contains(err.Error(), test.wantErrMsg) {
				t.Errorf("want error message: %q, got %q", test.wantErrMsg, err.Error())
			}
			return
		}
		if err != nil {
			t.Errorf("failed to run processLibrary(): %q", err.Error())
		}
		if test.libraryState.Version == "0.0.0" && test.repo.(*MockRepository).GetCommitsForPathsSinceTagLastTagName != "" {
			t.Errorf("GetCommitsForPathsSinceTag should be called with an empty tag name for version 0.0.0, got %q", test.repo.(*MockRepository).GetCommitsForPathsSinceTagLastTagName)
		}
	}
}

func TestFilterCommitsByLibraryID(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		commits   []*legacygitrepo.ConventionalCommit
		LibraryID string
		want      []*legacygitrepo.ConventionalCommit
	}{
		{
			name: "commits_all_match_libraryID",
			commits: []*legacygitrepo.ConventionalCommit{
				{
					LibraryID: "library-one",
					Type:      "feat",
				},
				{
					LibraryID: "library-one",
					Type:      "chore",
				},
				{
					LibraryID: "library-one",
					Type:      "deps",
				},
			},
			LibraryID: "library-one",
			want: []*legacygitrepo.ConventionalCommit{
				{
					LibraryID: "library-one",
					Type:      "feat",
				},
				{
					LibraryID: "library-one",
					Type:      "chore",
				},
				{
					LibraryID: "library-one",
					Type:      "deps",
				},
			},
		},
		{
			name: "some_commits_match_libraryID",
			commits: []*legacygitrepo.ConventionalCommit{
				{
					LibraryID: "library-one",
					Type:      "feat",
				},
				{
					LibraryID: "library-two",
					Type:      "chore",
				},
				{
					LibraryID: "library-three",
					Type:      "deps",
				},
			},
			LibraryID: "library-one",
			want: []*legacygitrepo.ConventionalCommit{
				{
					LibraryID: "library-one",
					Type:      "feat",
				},
			},
		},
		{
			name: "some_commits_have_library_id_in_footer",
			commits: []*legacygitrepo.ConventionalCommit{
				{
					LibraryID: "library-one",
					Type:      "feat",
					Footers: map[string]string{
						"Library-IDs": "library-one,library-two",
					},
				},
				{
					LibraryID: "library-two",
					Type:      "chore",
					Footers: map[string]string{
						"Library-IDs": "library-one,library-two",
					},
				},
				{
					LibraryID: "library-three",
					Type:      "deps",
				},
			},
			LibraryID: "library-one",
			want: []*legacygitrepo.ConventionalCommit{
				{
					LibraryID: "library-one",
					Type:      "feat",
					Footers: map[string]string{
						"Library-IDs": "library-one,library-two",
					},
				},
				{
					LibraryID: "library-two",
					Type:      "chore",
					Footers: map[string]string{
						"Library-IDs": "library-one,library-two",
					},
				},
			},
		},
		{
			name: "some_commits_have_library_id_that_is_prefix_of_another",
			commits: []*legacygitrepo.ConventionalCommit{
				{
					LibraryID: "library-one",
					Type:      "feat",
					Footers: map[string]string{
						"Library-IDs": "library-one,library-one_suffix",
					},
				},
				{
					LibraryID: "library-one-suffix",
					Type:      "chore",
					Footers: map[string]string{
						"Library-IDs": "library-one-suffix",
					},
				},
			},
			LibraryID: "library-one",
			want: []*legacygitrepo.ConventionalCommit{
				{
					LibraryID: "library-one",
					Type:      "feat",
					Footers: map[string]string{
						"Library-IDs": "library-one,library-one_suffix",
					},
				},
			},
		},
		{
			name: "no_commits_match_libraryID",
			commits: []*legacygitrepo.ConventionalCommit{
				{
					LibraryID: "library-one",
					Type:      "feat",
				},
				{
					LibraryID: "library-two",
					Type:      "chore",
				},
				{
					LibraryID: "library-three",
					Type:      "deps",
				},
			},
			LibraryID: "invalid-library",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := filterCommitsByLibraryID(test.commits, test.LibraryID)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("commit filter mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUpdateLibrary(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name           string
		libraryState   *legacyconfig.LibraryState
		library        string // this is the `--library` input
		libraryVersion string // this is the `--version` input
		commits        []*legacygitrepo.ConventionalCommit
		want           *legacyconfig.LibraryState
		wantErr        bool
		wantErrMsg     string
	}{
		{
			name: "update a library, automatic version calculation",
			libraryState: &legacyconfig.LibraryState{
				ID:      "one-id",
				Version: "1.2.3",
			},
			commits: []*legacygitrepo.ConventionalCommit{
				{
					Type:    "fix",
					Subject: "change a typo",
				},
				{
					Type:    "feat",
					Subject: "add a config file",
					Body:    "This is the body.",
					Footers: map[string]string{"PiperOrigin-RevId": "12345"},
				},
			},
			want: &legacyconfig.LibraryState{
				ID:              "one-id",
				Version:         "1.3.0",
				PreviousVersion: "1.2.3",
				Changes: []*legacyconfig.Commit{
					{
						Type:       "fix",
						Subject:    "change a typo",
						LibraryIDs: "one-id",
					},
					{
						Type:          "feat",
						Subject:       "add a config file",
						Body:          "This is the body.",
						PiperCLNumber: "12345",
						LibraryIDs:    "one-id",
					},
				},
				ReleaseTriggered: true,
			},
		},
		{
			name: "update a library with releasable units, valid version inputted",
			libraryState: &legacyconfig.LibraryState{
				ID:      "one-id",
				Version: "1.2.3",
			},
			libraryVersion: "5.0.0",
			commits: []*legacygitrepo.ConventionalCommit{
				{
					Type:    "fix",
					Subject: "change a typo",
				},
				{
					Type:    "feat",
					Subject: "add a config file",
					Body:    "This is the body.",
					Footers: map[string]string{"PiperOrigin-RevId": "12345"},
				},
			},
			want: &legacyconfig.LibraryState{
				ID:              "one-id",
				Version:         "5.0.0", // Use the `--version` value`
				PreviousVersion: "1.2.3",
				Changes: []*legacyconfig.Commit{
					{
						Type:       "fix",
						Subject:    "change a typo",
						LibraryIDs: "one-id",
					},
					{
						Type:          "feat",
						Subject:       "add a config file",
						Body:          "This is the body.",
						PiperCLNumber: "12345",
						LibraryIDs:    "one-id",
					},
				},
				ReleaseTriggered: true,
			},
		},
		{
			name: "update a library with releasable units, invalid version inputted",
			libraryState: &legacyconfig.LibraryState{
				ID:      "one-id",
				Version: "1.2.3",
			},
			libraryVersion: "1.0.0",
			commits: []*legacygitrepo.ConventionalCommit{
				{
					Type:    "fix",
					Subject: "change a typo",
				},
				{
					Type:    "feat",
					Subject: "add a config file",
					Body:    "This is the body.",
					Footers: map[string]string{"PiperOrigin-RevId": "12345"},
				},
			},
			wantErr:    true,
			wantErrMsg: "inputted version is not SemVer greater than the current version. Set a version SemVer greater than current than",
		},
		{
			name: "update a library with library ids in footer",
			libraryState: &legacyconfig.LibraryState{
				ID:      "one-id",
				Version: "1.2.3",
			},
			commits: []*legacygitrepo.ConventionalCommit{
				{
					Type:    "feat",
					Subject: "add a config file",
					Body:    "This is the body.",
					Footers: map[string]string{"Library-IDs": "a,b,c"},
				},
			},
			want: &legacyconfig.LibraryState{
				ID:              "one-id",
				Version:         "1.3.0",
				PreviousVersion: "1.2.3",
				Changes: []*legacyconfig.Commit{
					{
						Type:       "feat",
						Subject:    "add a config file",
						Body:       "This is the body.",
						LibraryIDs: "a,b,c",
					},
				},
				ReleaseTriggered: true,
			},
		},
		{
			name: "library has breaking changes",
			libraryState: &legacyconfig.LibraryState{
				ID:      "one-id",
				Version: "1.2.3",
			},
			commits: []*legacygitrepo.ConventionalCommit{
				{
					Type:    "feat",
					Subject: "add another config file",
					Body:    "This is the body",
					Footers: map[string]string{
						"BREAKING CHANGE": "this is a breaking change",
					},
					IsBreaking: true,
				},
				{
					Type:       "feat",
					Subject:    "change a typo",
					IsBreaking: true,
				},
			},
			want: &legacyconfig.LibraryState{
				ID:              "one-id",
				Version:         "2.0.0",
				PreviousVersion: "1.2.3",
				Changes: []*legacyconfig.Commit{
					{
						Type:       "feat",
						Subject:    "add another config file",
						Body:       "This is the body",
						LibraryIDs: "one-id",
					},
					{
						Type:       "feat",
						Subject:    "change a typo",
						LibraryIDs: "one-id",
					},
				},
				ReleaseTriggered: true,
			},
		},
		{
			name: "library has no changes",
			libraryState: &legacyconfig.LibraryState{
				ID:      "one-id",
				Version: "1.2.3",
			},
			want: &legacyconfig.LibraryState{
				ID:      "one-id",
				Version: "1.2.3",
			},
		},
		{
			name: "library has no releasable units and is inputted for release without a version",
			libraryState: &legacyconfig.LibraryState{
				ID:      "one-id",
				Version: "1.2.3",
			},
			library: "one-id",
			commits: []*legacygitrepo.ConventionalCommit{
				{
					Type:    "chore",
					Subject: "a chore",
				},
			},
			wantErr:    true,
			wantErrMsg: "library does not have a releasable unit and will not be released. Use the version flag to force a release for",
		},
		{
			name: "library has no releasable units and is inputted for release with a specific version",
			libraryState: &legacyconfig.LibraryState{
				ID:      "one-id",
				Version: "1.2.3",
			},
			library:        "one-id",
			libraryVersion: "5.0.0",
			commits: []*legacygitrepo.ConventionalCommit{
				{
					Type:    "chore",
					Subject: "a chore",
				},
			},
			want: &legacyconfig.LibraryState{
				ID:               "one-id",
				PreviousVersion:  "1.2.3",
				Version:          "5.0.0", // Use the `--version` override value
				ReleaseTriggered: true,
				Changes: []*legacyconfig.Commit{
					{
						Type:       "chore",
						Subject:    "a chore",
						LibraryIDs: "one-id",
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := &stageRunner{
				library:        test.library,
				libraryVersion: test.libraryVersion,
			}
			err := r.updateLibrary(test.libraryState, test.commits)

			if test.wantErr {
				if err == nil {
					t.Fatal("updateLibrary() should return error")
				}
				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %q, got %q", test.wantErrMsg, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("failed to run updateLibrary(): %q", err.Error())
			}
			if diff := cmp.Diff(test.want, test.libraryState); diff != "" {
				t.Errorf("state mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDetermineNextVersion(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name            string
		commits         []*legacygitrepo.ConventionalCommit
		currentVersion  string
		libraryID       string
		config          *legacyconfig.Config
		librarianConfig *legacyconfig.LibrarianConfig
		wantVersion     string
		wantErr         bool
		wantErrMsg      string
	}{
		{
			name: "from commits",
			commits: []*legacygitrepo.ConventionalCommit{
				{Type: "feat"},
			},
			config: &legacyconfig.Config{
				Library: "some-library",
			},
			libraryID: "some-library",
			librarianConfig: &legacyconfig.LibrarianConfig{
				Libraries: []*legacyconfig.LibraryConfig{},
			},
			currentVersion: "1.0.0",
			wantVersion:    "1.1.0",
			wantErr:        false,
		},
		{
			name: "with legacyconfig.yaml override version",
			commits: []*legacygitrepo.ConventionalCommit{
				{Type: "feat"},
			},
			config: &legacyconfig.Config{
				Library: "some-library",
			},
			libraryID: "some-library",
			librarianConfig: &legacyconfig.LibrarianConfig{
				Libraries: []*legacyconfig.LibraryConfig{
					&legacyconfig.LibraryConfig{
						LibraryID:   "some-library",
						NextVersion: "2.3.4",
					},
				},
			},
			currentVersion: "1.0.0",
			wantVersion:    "2.3.4",
			wantErr:        false,
		},
		{
			name: "with outdated legacyconfig.yaml override version",
			commits: []*legacygitrepo.ConventionalCommit{
				{Type: "feat"},
			},
			config: &legacyconfig.Config{
				Library: "some-library",
			},
			libraryID: "some-library",
			librarianConfig: &legacyconfig.LibrarianConfig{
				Libraries: []*legacyconfig.LibraryConfig{
					&legacyconfig.LibraryConfig{
						LibraryID:   "some-library",
						NextVersion: "2.3.4",
					},
				},
			},
			currentVersion: "2.4.0",
			wantVersion:    "2.5.0",
			wantErr:        false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			runner := &stageRunner{
				libraryVersion:  test.config.LibraryVersion,
				librarianConfig: test.librarianConfig,
			}
			got, err := runner.determineNextVersion(test.commits, test.currentVersion, test.libraryID)
			if test.wantErr {
				if err == nil {
					t.Fatal("determineNextVersion() should return error")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %q, got %q", test.wantErrMsg, err.Error())
				}

				return
			}
			if diff := cmp.Diff(test.wantVersion, got); diff != "" {
				t.Errorf("state mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
