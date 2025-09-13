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

	"github.com/go-git/go-git/v5/plumbing"
	"gopkg.in/yaml.v3"

	"github.com/googleapis/librarian/internal/conventionalcommits"

	"github.com/go-git/go-git/v5"

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
	gitStatus := make(git.Status)
	gitStatus["file.txt"] = &git.FileStatus{Worktree: git.Modified}
	for _, test := range []struct {
		name       string
		runner     *initRunner
		files      map[string]string
		want       *config.LibrarianState
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "run release init command for all libraries, update librarian state",
			runner: &initRunner{
				workRoot:        t.TempDir(),
				containerClient: &mockContainerClient{},
				state: &config.LibrarianState{
					Libraries: []*config.LibraryState{
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
					GetCommitsForPathsSinceTagValueByTag: map[string][]*gitrepo.Commit{
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
				librarianConfig: &config.LibrarianConfig{},
				partialRepo:     t.TempDir(),
			},
			files: map[string]string{
				"file1.txt":      "",
				"dir1/file1.txt": "",
				"dir2/file2.txt": "",
				"dir3/file3.txt": "",
				"dir4/file4.txt": "",
			},
			want: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:      "another-example-id",
						Version: "1.1.0", // version is bumped.
						APIs:    []*config.API{},
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
						APIs:    []*config.API{},
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
			name: "run release init command for one library",
			runner: &initRunner{
				workRoot:        t.TempDir(),
				containerClient: &mockContainerClient{},
				library:         "example-id",
				state: &config.LibrarianState{
					Libraries: []*config.LibraryState{
						{
							ID: "another-example-id",
							SourceRoots: []string{
								"dir3",
								"dir4",
							},
						},
						{
							ID: "example-id",
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
				},
				librarianConfig: &config.LibrarianConfig{},
				partialRepo:     t.TempDir(),
			},
			files: map[string]string{
				"file1.txt":      "",
				"dir1/file1.txt": "",
				"dir2/file2.txt": "",
			},
			want: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:   "another-example-id",
						APIs: []*config.API{},
						SourceRoots: []string{
							"dir3",
							"dir4",
						},
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
					},
					{
						ID:   "example-id",
						APIs: []*config.API{},
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
			name: "run release init command without librarian config",
			runner: &initRunner{
				workRoot:        t.TempDir(),
				containerClient: &mockContainerClient{},
				library:         "example-id",
				state: &config.LibrarianState{
					Libraries: []*config.LibraryState{
						{
							ID: "another-example-id",
							SourceRoots: []string{
								"dir3",
								"dir4",
							},
						},
						{
							ID: "example-id",
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
				},
				partialRepo: t.TempDir(),
			},
			files: map[string]string{
				"file1.txt":      "",
				"dir1/file1.txt": "",
				"dir2/file2.txt": "",
			},
			want: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:   "another-example-id",
						APIs: []*config.API{},
						SourceRoots: []string{
							"dir3",
							"dir4",
						},
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
					},
					{
						ID:   "example-id",
						APIs: []*config.API{},
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
			runner: &initRunner{
				workRoot: t.TempDir(),
				containerClient: &mockContainerClient{
					initErr: errors.New("simulated init error"),
				},
				state: &config.LibrarianState{},
				repo: &MockRepository{
					Dir: t.TempDir(),
				},
				partialRepo:     t.TempDir(),
				librarianConfig: &config.LibrarianConfig{},
			},
			wantErr:    true,
			wantErrMsg: "simulated init error",
		},
		{
			name: "release response contains error message",
			runner: &initRunner{
				workRoot: t.TempDir(),
				containerClient: &mockContainerClient{
					wantErrorMsg: true,
				},
				state: &config.LibrarianState{},
				repo: &MockRepository{
					Dir: t.TempDir(),
				},
				partialRepo:     t.TempDir(),
				librarianConfig: &config.LibrarianConfig{},
			},
			wantErr:    true,
			wantErrMsg: "failed with error message: simulated error message",
		},
		{
			name: "invalid work root",
			runner: &initRunner{
				workRoot: "/invalid/path",
				repo: &MockRepository{
					Dir: t.TempDir(),
				},
			},
			wantErr:    true,
			wantErrMsg: "failed to create output dir",
		},
		{
			name: "failed to get changes from repo when releasing one library",
			runner: &initRunner{
				workRoot:        t.TempDir(),
				containerClient: &mockContainerClient{},
				library:         "example-id",
				state: &config.LibrarianState{
					Libraries: []*config.LibraryState{
						{
							ID: "example-id",
						},
					},
				},
				repo: &MockRepository{
					Dir:                             t.TempDir(),
					GetCommitsForPathsSinceTagError: errors.New("simulated error when getting commits"),
				},
				partialRepo: t.TempDir(),
			},
			wantErr:    true,
			wantErrMsg: "failed to fetch conventional commits for library",
		},
		{
			name: "failed to get changes from repo when releasing multiple libraries",
			runner: &initRunner{
				workRoot:        t.TempDir(),
				containerClient: &mockContainerClient{},
				state: &config.LibrarianState{
					Libraries: []*config.LibraryState{
						{
							ID: "example-id",
						},
					},
				},
				repo: &MockRepository{
					Dir:                             t.TempDir(),
					GetCommitsForPathsSinceTagError: errors.New("simulated error when getting commits"),
				},
				partialRepo: t.TempDir(),
			},
			wantErr:    true,
			wantErrMsg: "failed to fetch conventional commits for library",
		},
		{
			name: "failed to commit and push",
			runner: &initRunner{
				workRoot:        os.TempDir(),
				containerClient: &mockContainerClient{},
				push:            true,
				state: &config.LibrarianState{
					Libraries: []*config.LibraryState{
						{
							ID: "example-id",
						},
					},
				},
				repo: &MockRepository{
					Dir:          t.TempDir(),
					AddAllStatus: gitStatus,
					RemotesValue: []*git.Remote{}, // No remotes
				},
				librarianConfig: &config.LibrarianConfig{},
				partialRepo:     t.TempDir(),
			},
			wantErr:    true,
			wantErrMsg: "failed to commit and push",
		},
		{
			name: "failed to make partial repo",
			runner: &initRunner{
				workRoot: t.TempDir(),
				repo: &MockRepository{
					Dir: t.TempDir(),
				},
				partialRepo: "/invalid/path",
			},
			wantErr:    true,
			wantErrMsg: "failed to make directory",
		},
		{
			name: "run release init command with symbolic link",
			runner: &initRunner{
				workRoot:        t.TempDir(),
				containerClient: &mockContainerClient{},
				library:         "example-id",
				state: &config.LibrarianState{
					Libraries: []*config.LibraryState{
						{
							ID: "example-id",
							SourceRoots: []string{
								"dir1",
							},
						},
					},
				},
				repo: &MockRepository{
					Dir: t.TempDir(),
				},
				librarianConfig: &config.LibrarianConfig{},
				partialRepo:     t.TempDir(),
			},
			files: map[string]string{
				"dir1/file1.txt": "hello",
			},
			want: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:   "example-id",
						APIs: []*config.API{},
						SourceRoots: []string{
							"dir1",
						},
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
					},
				},
			},
		},
		{
			name: "copy library files returns error",
			runner: &initRunner{
				workRoot:        t.TempDir(),
				containerClient: &mockContainerClient{},
				library:         "example-id",
				state: &config.LibrarianState{
					Libraries: []*config.LibraryState{
						{
							ID: "example-id",
							SourceRoots: []string{
								"dir1",
							},
						},
					},
				},
				repo: &MockRepository{
					Dir: t.TempDir(),
				},
				librarianConfig: &config.LibrarianConfig{},
				partialRepo:     t.TempDir(),
			},
			files: map[string]string{
				"dir1/file1.txt": "hello",
			},
			wantErr:    true,
			wantErrMsg: "failed to copy file",
		},
		{
			name: "copy library files returns error (no library id in cfg)",
			runner: &initRunner{
				workRoot:        t.TempDir(),
				containerClient: &mockContainerClient{},
				state: &config.LibrarianState{
					Libraries: []*config.LibraryState{
						{
							ID: "example-id",
							SourceRoots: []string{
								"dir1",
							},
						},
					},
				},
				repo: &MockRepository{
					Dir: t.TempDir(),
				},
				librarianConfig: &config.LibrarianConfig{},
				partialRepo:     t.TempDir(),
			},
			files: map[string]string{
				"dir1/file1.txt": "hello",
			},
			wantErr:    true,
			wantErrMsg: "failed to copy file",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Setup library files before running the command.
			repoDir := test.runner.repo.GetDir()
			outputDir := filepath.Join(test.runner.workRoot, "output")
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
			if strings.HasPrefix(test.name, "copy library files returns error") {
				// Make the directory non-writable so that the copy operations fail.
				if err := os.Chmod(test.runner.partialRepo, 0555); err != nil {
					t.Fatalf("os.Chmod() = %v", err)
				}
			}
			// Create a symbolic link for the test case with symbolic links.
			if test.name == "run release init command with symbolic link" {
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
				t.Errorf("run() failed: %s", err.Error())
			}
			// load librarian state from state.yaml, which should contain updated
			// library state.
			bytes, err := os.ReadFile(filepath.Join(repoDir, ".librarian/state.yaml"))
			if err != nil {
				t.Fatal(err)
			}

			var got *config.LibrarianState
			if err := yaml.Unmarshal(bytes, &got); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
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
		libraryVersion  string
		library         *config.LibraryState
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
			libraryVersion: "2.0.0",
			library: &config.LibraryState{
				ID:      "one-id",
				Version: "1.2.3",
				SourceRoots: []string{
					"one/path",
					"two/path",
				},
			},
			want: &config.LibraryState{
				ID:              "one-id",
				Version:         "2.0.0",
				PreviousVersion: "1.2.3",
				SourceRoots: []string{
					"one/path",
					"two/path",
				},
				Changes: []*conventionalcommits.ConventionalCommit{
					{
						Type:      "fix",
						Subject:   "change a typo",
						LibraryID: "one-id",
						Footers:   map[string]string{},
					},
					{
						Type:      "feat",
						Subject:   "add a config file",
						Body:      "This is the body.",
						LibraryID: "one-id",
						Footers:   map[string]string{"PiperOrigin-RevId": "12345"},
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
			},
			library: &config.LibraryState{
				ID:      "one-id",
				Version: "1.2.3",
				SourceRoots: []string{
					"one/path",
					"two/path",
				},
			},
			want: &config.LibraryState{
				ID:              "one-id",
				Version:         "2.0.0",
				PreviousVersion: "1.2.3",
				SourceRoots: []string{
					"one/path",
					"two/path",
				},
				Changes: []*conventionalcommits.ConventionalCommit{
					{
						Type:      "feat",
						Subject:   "add another config file",
						Body:      "This is the body",
						LibraryID: "one-id",
						Footers: map[string]string{
							"BREAKING CHANGE": "this is a breaking change",
						},
						IsBreaking: true,
					},
					{
						Type:       "feat",
						Subject:    "change a typo",
						LibraryID:  "one-id",
						Footers:    map[string]string{},
						IsBreaking: true,
					},
				},
				ReleaseTriggered: true,
			},
		},
		{
			name: "failed to get commit history of one library",
			library: &config.LibraryState{
				ID:      "one-id",
				Version: "1.2.3",
				SourceRoots: []string{
					"one/path",
					"two/path",
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
			r := &initRunner{
				libraryVersion: test.libraryVersion,
				repo:           test.repo,
			}
			var err error
			if test.repo != nil {
				err = r.updateLibrary(test.library)
			} else {
				repo := setupRepoForGetCommits(t, test.pathAndMessages, test.tags)
				r.repo = repo
				err = r.updateLibrary(test.library)
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
			if diff := cmp.Diff(test.want, test.library, cmpopts.IgnoreFields(conventionalcommits.ConventionalCommit{}, "SHA", "When")); diff != "" {
				t.Errorf("state mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCopyGlobalAllowlist(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name              string
		cfg               *config.LibrarianConfig
		files             []string
		copied            []string
		skipped           []string
		doNotCreateOutput bool // do not create files in output dir.
		wantErr           bool
		wantErrMsg        string
		copyReadOnly      bool
	}{
		{
			name: "copied all global allowlist",
			cfg: &config.LibrarianConfig{
				GlobalFilesAllowlist: []*config.GlobalFile{
					{
						Path:        "one/path/example.txt",
						Permissions: "read-write",
					},
					{
						Path:        "another/path/example.txt",
						Permissions: "write-only",
					},
				},
			},
			files: []string{
				"one/path/example.txt",
				"another/path/example.txt",
				"ignored/path/example.txt",
			},
			copied: []string{
				"one/path/example.txt",
				"another/path/example.txt",
			},
			skipped: []string{
				"ignored/path/example.txt",
			},
		},
		{
			name: "read only file is not copied",
			cfg: &config.LibrarianConfig{
				GlobalFilesAllowlist: []*config.GlobalFile{
					{
						Path:        "one/path/example.txt",
						Permissions: "read-write",
					},
					{
						Path:        "another/path/example.txt",
						Permissions: "read-only",
					},
				},
			},
			files: []string{
				"one/path/example.txt",
				"another/path/example.txt",
				"ignored/path/example.txt",
			},
			copied: []string{
				"one/path/example.txt",
			},
			skipped: []string{
				"another/path/example.txt",
				"ignored/path/example.txt",
			},
		},
		{
			name: "repo doesn't have the global file",
			cfg: &config.LibrarianConfig{
				GlobalFilesAllowlist: []*config.GlobalFile{
					{
						Path:        "one/path/example.txt",
						Permissions: "read-write",
					},
					{
						Path:        "another/path/example.txt",
						Permissions: "read-only",
					},
				},
			},
			files: []string{
				"another/path/example.txt",
				"ignored/path/example.txt",
			},
			wantErr:    true,
			wantErrMsg: "failed to lstat file",
		},
		{
			name: "output doesn't have the global file",
			cfg: &config.LibrarianConfig{
				GlobalFilesAllowlist: []*config.GlobalFile{
					{
						Path:        "one/path/example.txt",
						Permissions: "read-write",
					},
				},
			},
			files: []string{
				"one/path/example.txt",
			},
			doNotCreateOutput: true,
			wantErr:           true,
			wantErrMsg:        "failed to copy global file",
		},
		{
			name:         "copies read-only files",
			copyReadOnly: true,
			cfg: &config.LibrarianConfig{
				GlobalFilesAllowlist: []*config.GlobalFile{
					{
						Path:        "one/path/example.txt",
						Permissions: "read-write",
					},
					{
						Path:        "another/path/example.txt",
						Permissions: "read-only",
					},
				},
			},
			files: []string{
				"one/path/example.txt",
				"another/path/example.txt",
				"ignored/path/example.txt",
			},
			copied: []string{
				"one/path/example.txt",
				"another/path/example.txt",
			},
			skipped: []string{
				"ignored/path/example.txt",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			output := t.TempDir()
			repo := t.TempDir()
			for _, oneFile := range test.files {
				// Create files in repo directory.
				file := filepath.Join(repo, oneFile)
				dir := filepath.Dir(file)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Error(err)
				}

				err := os.WriteFile(file, []byte("old content"), 0755)
				if err != nil {
					t.Error(err)
				}

				if test.doNotCreateOutput {
					continue
				}

				// Create files in output directory.
				file = filepath.Join(output, oneFile)
				dir = filepath.Dir(file)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Error(err)
				}

				err = os.WriteFile(file, []byte("new content"), 0755)
				if err != nil {
					t.Error(err)
				}
			}

			err := copyGlobalAllowlist(test.cfg, repo, output, test.copyReadOnly)

			if test.wantErr {
				if err == nil {
					t.Error("cleanAndCopyGlobalAllowlist() should return error")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %q, got %q", test.wantErrMsg, err.Error())
				}

				return
			}
			if err != nil {
				t.Errorf("failed to run cleanAndCopyGlobalAllowlist(): %q", err.Error())
			}

			for _, wantFile := range test.copied {
				got, err := os.ReadFile(filepath.Join(repo, wantFile))
				if err != nil {
					return
				}
				if diff := cmp.Diff("new content", string(got)); diff != "" {
					t.Errorf("state mismatch (-want +got):\n%s in %s", diff, wantFile)
				}
			}
			// Make sure the skipped files are not changed.
			for _, skippedFile := range test.skipped {
				got, err := os.ReadFile(filepath.Join(repo, skippedFile))
				if err != nil {
					return
				}
				if diff := cmp.Diff("old content", string(got)); diff != "" {
					t.Errorf("state mismatch (-want +got):\n%s in %s", diff, skippedFile)
				}
			}
		})
	}
}

func TestDetermineNextVersion(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name            string
		commits         []*conventionalcommits.ConventionalCommit
		currentVersion  string
		libraryID       string
		config          *config.Config
		librarianConfig *config.LibrarianConfig
		wantVersion     string
		wantErr         bool
		wantErrMsg      string
	}{
		{
			name: "from commits",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "feat"},
			},
			config: &config.Config{
				Library: "some-library",
			},
			libraryID: "some-library",
			librarianConfig: &config.LibrarianConfig{
				Libraries: []*config.LibraryConfig{},
			},
			currentVersion: "1.0.0",
			wantVersion:    "1.1.0",
			wantErr:        false,
		},
		{
			name: "with CLI override version",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "feat"},
			},
			config: &config.Config{
				Library:        "some-library",
				LibraryVersion: "1.2.3",
			},
			libraryID: "some-library",
			librarianConfig: &config.LibrarianConfig{
				Libraries: []*config.LibraryConfig{
					&config.LibraryConfig{
						LibraryID:   "some-library",
						NextVersion: "2.3.4",
					},
				},
			},
			currentVersion: "1.0.0",
			wantVersion:    "1.2.3",
			wantErr:        false,
		},
		{
			name: "with CLI override version cannot revert version",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "feat"},
			},
			config: &config.Config{
				Library:        "some-library",
				LibraryVersion: "1.2.3",
			},
			libraryID: "some-library",
			librarianConfig: &config.LibrarianConfig{
				Libraries: []*config.LibraryConfig{
					&config.LibraryConfig{
						LibraryID: "some-library",
					},
				},
			},
			currentVersion: "2.4.0",
			wantVersion:    "2.5.0",
			wantErr:        false,
		},
		{
			name: "with config.yaml override version",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "feat"},
			},
			config: &config.Config{
				Library: "some-library",
			},
			libraryID: "some-library",
			librarianConfig: &config.LibrarianConfig{
				Libraries: []*config.LibraryConfig{
					&config.LibraryConfig{
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
			name: "with outdated config.yaml override version",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "feat"},
			},
			config: &config.Config{
				Library: "some-library",
			},
			libraryID: "some-library",
			librarianConfig: &config.LibrarianConfig{
				Libraries: []*config.LibraryConfig{
					&config.LibraryConfig{
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
			runner := &initRunner{
				libraryVersion:  test.config.LibraryVersion,
				librarianConfig: test.librarianConfig,
			}
			got, err := runner.determineNextVersion(test.commits, test.currentVersion, test.libraryID)
			if test.wantErr {
				if err == nil {
					t.Error("determineNextVersion() should return error")
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
