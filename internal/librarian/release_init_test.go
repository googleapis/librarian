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
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/gitrepo"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/googleapis/librarian/internal/config"
)

func TestNewInitRunner(t *testing.T) {
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
				API:       "some/api",
				APISource: newTestGitRepo(t).GetDir(),
				Repo:      newTestGitRepo(t).GetDir(),
				WorkRoot:  t.TempDir(),
				Image:     "gcr.io/test/test-image",
			},
		},
		{
			name: "invalid config",
			cfg: &config.Config{
				APISource: newTestGitRepo(t).GetDir(),
			},
			wantErr:    true,
			wantErrMsg: "failed to create init runner",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := newInitRunner(test.cfg)
			if test.wantErr {
				if err == nil {
					t.Error("newInitRunner() should return error")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %q, got %q", test.wantErrMsg, err.Error())
				}

				return
			}
			if err != nil {
				t.Errorf("newInitRunner() = %v, want nil", err)
			}
		})
	}
}

func TestInitRun(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		runner     *initRunner
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "run docker command",
			runner: &initRunner{
				workRoot:        os.TempDir(),
				containerClient: &mockContainerClient{},
				cfg:             &config.Config{},
				state:           &config.LibrarianState{},
			},
		},
		{
			name: "docker command returns error",
			runner: &initRunner{
				workRoot: os.TempDir(),
				containerClient: &mockContainerClient{
					initErr: errors.New("simulated init error"),
				},
				cfg:   &config.Config{},
				state: &config.LibrarianState{},
			},
			wantErr:    true,
			wantErrMsg: "simulated init error",
		},
		{
			name: "invalid work root",
			runner: &initRunner{
				workRoot: "/invalid/path",
			},
			wantErr:    true,
			wantErrMsg: "failed to create output dir",
		},
		{
			name: "failed to get changes from repo",
			runner: &initRunner{
				workRoot:        os.TempDir(),
				containerClient: &mockContainerClient{},
				cfg:             &config.Config{},
				state: &config.LibrarianState{
					Libraries: []*config.LibraryState{
						{
							ID: "example-id",
						},
					},
				},
				repo: &MockRepository{
					GetCommitsForPathsSinceTagError: errors.New("simulated error when getting commits"),
				},
			},
			wantErr:    true,
			wantErrMsg: "failed to fetch conventional commits for library",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.runner.run(context.Background())
			if test.wantErr {
				if err == nil {
					t.Error("run() should return error")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %q, got %q", test.wantErrMsg, err.Error())
				}

				return
			}
			if err != nil {
				t.Errorf("run() got nil runner, want non-nil")
			}
		})
	}
}

func TestSetReleaseTrigger(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name           string
		library        *config.LibraryState
		libraryID      string
		libraryVersion string
		trigger        bool
		want           *config.LibraryState
	}{
		{
			name: "set trigger for a library",
			library: &config.LibraryState{
				ID:      "one-example-id",
				Version: "1.0.0",
			},
			trigger: true,
			want: &config.LibraryState{
				ID:               "one-example-id",
				Version:          "1.0.0",
				ReleaseTriggered: true,
			},
		},
		{
			name: "set trigger for one library and override version",
			library: &config.LibraryState{
				ID:      "one-example-id",
				Version: "1.0.0",
			},
			trigger:        true,
			libraryVersion: "2.0.0",
			want: &config.LibraryState{
				ID:               "one-example-id",
				Version:          "2.0.0",
				ReleaseTriggered: true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			setReleaseTrigger(test.library, test.libraryVersion, test.trigger)
			if diff := cmp.Diff(test.want, test.library); diff != "" {
				t.Errorf("state mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUpdateLibrary(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name            string
		pathAndMessages []pathAndMessage
		tags            []string
		runner          *initRunner
		state           *config.LibrarianState
		repo            gitrepo.Repository
		want            *config.LibraryState
		wantErr         bool
		wantErrMsg      string
	}{
		{
			name: "update a library",
			pathAndMessages: []pathAndMessage{
				{
					path:    "non-related/path/example.txt",
					message: "chore: initial commit",
				},
				{
					path:    "one/path/example.txt",
					message: "feat: add a config file\n\nThis is the body.\n\nPiperOrigin-RevId: 12345",
				},
				{
					path:    "one/path/example.txt",
					message: "fix: change a typo",
				},
				{
					path:    "another/path/example.txt",
					message: "fix: another commit",
				},
			},
			tags: []string{
				"one-id-1.2.3",
			},
			runner: &initRunner{
				cfg: &config.Config{
					LibraryVersion: "2.0.0",
				},
			},
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:      "another-id",
						Version: "2.3.4",
						SourceRoots: []string{
							"another/path",
						},
					},
					{
						ID:      "one-id",
						Version: "1.2.3",
						SourceRoots: []string{
							"one/path",
							"two/path",
						},
					},
				},
			},
			want: &config.LibraryState{
				ID:      "one-id",
				Version: "2.0.0",
				SourceRoots: []string{
					"one/path",
					"two/path",
				},
				Changes: []*config.Change{
					{
						Type:    "fix",
						Subject: "change a typo",
					},
					{
						Type:    "feat",
						Subject: "add a config file",
						Body:    "This is the body.",
						ClNum:   "12345",
					},
				},
				ReleaseTriggered: true,
			},
		},
		{
			name: "get breaking changes of one library",
			pathAndMessages: []pathAndMessage{
				{
					path:    "non-related/path/example.txt",
					message: "chore: initial commit",
				},
				{
					path:    "one/path/example.txt",
					message: "feat!: change a typo",
				},
				{
					path:    "one/path/config.txt",
					message: "feat: add another config file\n\nThis is the body\n\nBREAKING CHANGE: this is a breaking change",
				},
			},
			tags: []string{
				"one-id-1.2.3",
				"another-id-2.3.4",
			},
			runner: &initRunner{cfg: &config.Config{}},
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:      "another-id",
						Version: "2.3.4",
						SourceRoots: []string{
							"another/path",
						},
					},
					{
						ID:      "one-id",
						Version: "1.2.3",
						SourceRoots: []string{
							"one/path",
							"two/path",
						},
					},
				},
			},
			want: &config.LibraryState{
				ID:      "one-id",
				Version: "1.2.3",
				SourceRoots: []string{
					"one/path",
					"two/path",
				},
				Changes: []*config.Change{
					{
						Type:    "feat!",
						Subject: "add another config file",
						Body:    "This is the body",
					},
					{
						Type:    "feat!",
						Subject: "change a typo",
					},
				},
				ReleaseTriggered: true,
			},
		},
		{
			name:   "failed to get commit history of one library",
			runner: &initRunner{cfg: &config.Config{}},
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:      "another-id",
						Version: "2.3.4",
						SourceRoots: []string{
							"another/path",
						},
					},
					{
						ID:      "one-id",
						Version: "1.2.3",
						SourceRoots: []string{
							"one/path",
							"two/path",
						},
					},
				},
			},
			repo: &MockRepository{
				GetCommitsForPathsSinceTagError: errors.New("simulated error when getting commits"),
			},
			wantErr:    true,
			wantErrMsg: "failed to fetch conventional commits for library",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var err error
			if test.repo != nil {
				test.runner.repo = test.repo
				err = updateLibrary(test.runner, test.state, 1)
			} else {
				test.runner.repo = setupRepoForGetCommits(t, test.pathAndMessages, test.tags)
				err = updateLibrary(test.runner, test.state, 1)
			}

			if test.wantErr {
				if err == nil {
					t.Error("getChangesOf() should return error")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %q, got %q", test.wantErrMsg, err.Error())
				}

				return
			}
			if err != nil {
				t.Errorf("failed to run getChangesOf(): %q", err.Error())
			}
			if diff := cmp.Diff(test.want, test.state.Libraries[1], cmpopts.IgnoreFields(config.Change{}, "CommitHash")); diff != "" {
				t.Errorf("state mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
