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
		name        string
		runner      *initRunner
		files       map[string]string
		outputFiles map[string]string
		wantFiles   map[string]string
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name: "run release init command for one library",
			runner: &initRunner{
				workRoot:        t.TempDir(),
				containerClient: &mockContainerClient{},
				cfg: &config.Config{
					Library: "example-id",
				},
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
		},
		{
			name: "run release init command should not remove existing files",
			runner: &initRunner{
				workRoot:        filepath.Join(t.TempDir(), "work-root"),
				containerClient: &mockContainerClient{},
				cfg: &config.Config{
					Library: "example-id",
				},
				state: &config.LibrarianState{
					Libraries: []*config.LibraryState{
						{
							ID: "example-id",
							SourceRoots: []string{
								"dir1",
							},
							RemoveRegex: []string{"dir1/.*"},
						},
					},
				},
				repo: &MockRepository{
					Dir: filepath.Join(t.TempDir(), "repo"),
				},
				librarianConfig: &config.LibrarianConfig{},
			},
			files: map[string]string{
				"dir1/should_keep.txt": "content",
			},
			outputFiles: map[string]string{
				"dir1/new_file.txt": "new content",
			},
			wantFiles: map[string]string{
				"dir1/should_keep.txt": "content",
				"dir1/new_file.txt":    "new content",
			},
		},
		{
			name: "run release init command without librarian config",
			runner: &initRunner{
				workRoot:        t.TempDir(),
				containerClient: &mockContainerClient{},
				cfg: &config.Config{
					Library: "example-id",
				},
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
		},
		{
			name: "run release init command for all libraries",
			runner: &initRunner{
				workRoot:        t.TempDir(),
				containerClient: &mockContainerClient{},
				cfg:             &config.Config{},
				state: &config.LibrarianState{
					Libraries: []*config.LibraryState{
						{
							ID: "another-example-id",
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
				"dir3/file3.txt": "",
				"dir4/file4.txt": "",
			},
		},
		{
			name: "docker command returns error",
			runner: &initRunner{
				workRoot: t.TempDir(),
				containerClient: &mockContainerClient{
					initErr: errors.New("simulated init error"),
				},
				cfg:   &config.Config{},
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
				cfg: &config.Config{
					Library: "example-id",
				},
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
				cfg:             &config.Config{},
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
			librarianDir := filepath.Join(repoDir, ".librarian")
			if err := os.MkdirAll(librarianDir, 0755); err != nil {
				t.Fatalf("os.MkdirAll() = %v", err)
			}

			outputDir := filepath.Join(test.runner.workRoot, "output")
			if test.outputFiles != nil {
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					t.Fatalf("os.MkdirAll() for output = %v", err)
				}
				for path, content := range test.outputFiles {
					fullPath := filepath.Join(outputDir, path)
					if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
						t.Fatalf("os.MkdirAll() for output file = %v", err)
					}
					if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
						t.Fatalf("os.WriteFile() for output file = %v", err)
					}
				}
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

			if test.wantFiles != nil {
				for path, wantContent := range test.wantFiles {
					fullPath := filepath.Join(repoDir, path)
					gotContent, err := os.ReadFile(fullPath)
					if err != nil {
						t.Errorf("os.ReadFile(%q) got error: %v", fullPath, err)
					}
					if string(gotContent) != wantContent {
						t.Errorf("content of %q is %q, want %q", path, string(gotContent), wantContent)
					}
				}
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

func TestCleanAndCopyGlobalAllowlist(t *testing.T) {
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
			wantErrMsg: "failed to remove global file",
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

			err := cleanAndCopyGlobalAllowlist(test.cfg, repo, output)

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
