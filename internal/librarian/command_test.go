// Copyright 2024 Google LLC
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
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	gogitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

func TestCommandUsage(t *testing.T) {
	for _, c := range CmdLibrarian.Commands {
		t.Run(c.Name(), func(t *testing.T) {
			parts := strings.Fields(c.UsageLine)
			// The first word should always be "librarian".
			if parts[0] != "librarian" {
				t.Errorf("invalid usage text: %q (the first word should be `librarian`)", c.UsageLine)
			}
			if !strings.Contains(c.UsageLine, c.Name()) {
				t.Errorf("invalid usage text: %q (should contain command name %q)", c.UsageLine, c.Name())
			}
		})
	}
}

func TestFindLibraryByID(t *testing.T) {
	lib1 := &config.LibraryState{ID: "lib1"}
	lib2 := &config.LibraryState{ID: "lib2"}
	stateWithLibs := &config.LibrarianState{
		Libraries: []*config.LibraryState{lib1, lib2},
	}
	stateNoLibs := &config.LibrarianState{
		Libraries: []*config.LibraryState{},
	}

	for _, test := range []struct {
		name      string
		state     *config.LibrarianState
		libraryID string
		want      *config.LibraryState
	}{
		{
			name:      "found first library",
			state:     stateWithLibs,
			libraryID: "lib1",
			want:      lib1,
		},
		{
			name:      "found second library",
			state:     stateWithLibs,
			libraryID: "lib2",
			want:      lib2,
		},
		{
			name:      "not found",
			state:     stateWithLibs,
			libraryID: "lib3",
			want:      nil,
		},
		{
			name:      "empty libraries slice",
			state:     stateNoLibs,
			libraryID: "lib1",
			want:      nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := findLibraryByID(test.state, test.libraryID)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("findLibraryByID() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeriveImage(t *testing.T) {
	for _, test := range []struct {
		name          string
		imageOverride string
		state         *config.LibrarianState
		want          string
	}{
		{
			name:          "with image override, nil state",
			imageOverride: "my/custom-image:v1",
			state:         nil,
			want:          "my/custom-image:v1",
		},
		{
			name:          "with image override, non-nil state",
			imageOverride: "my/custom-image:v1",
			state:         &config.LibrarianState{Image: "gcr.io/foo/bar:v1.2.3"},
			want:          "my/custom-image:v1",
		},
		{
			name:          "no override, nil state",
			imageOverride: "",
			state:         nil,
			want:          "",
		},
		{
			name:          "no override, with state",
			imageOverride: "",
			state:         &config.LibrarianState{Image: "gcr.io/foo/bar:v1.2.3"},
			want:          "gcr.io/foo/bar:v1.2.3",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := deriveImage(test.imageOverride, test.state)

			if got != test.want {
				t.Errorf("deriveImage() = %q, want %q", got, test.want)
			}
		})
	}
}

// newTestGitRepoWithCommit creates a new git repository with an initial commit.
// If dir is empty, a new temporary directory is created.
// It returns the path to the repository directory.
func newTestGitRepoWithCommit(t *testing.T, dir string) string {
	t.Helper()
	if dir == "" {
		dir = t.TempDir()
	} else {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", dir, err)
		}
	}
	for _, args := range [][]string{
		{"init"},
		{"config", "user.name", "tester"},
		{"config", "user.email", "tester@example.com"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}

	filePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	for _, args := range [][]string{
		{"add", "README.md"},
		{"commit", "-m", "initial commit"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	return dir
}

func TestCloneOrOpenLanguageRepo(t *testing.T) {
	workRoot := t.TempDir()

	cleanRepoPath := newTestGitRepoWithCommit(t, "")
	dirtyRepoPath := newTestGitRepoWithCommit(t, "")
	if err := os.WriteFile(filepath.Join(dirtyRepoPath, "untracked.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	notARepoPath := t.TempDir()

	for _, test := range []struct {
		name    string
		repo    string
		ci      string
		wantErr bool
		check   func(t *testing.T, repo *gitrepo.LocalRepository)
		setup   func(t *testing.T, workRoot string) func()
	}{
		{
			name: "with clean repoRoot",
			repo: cleanRepoPath,
			check: func(t *testing.T, repo *gitrepo.LocalRepository) {
				absWantDir, _ := filepath.Abs(cleanRepoPath)
				if repo.Dir != absWantDir {
					t.Errorf("repo.Dir got %q, want %q", repo.Dir, absWantDir)
				}
			},
		},
		{
			name: "with repoURL with trailing slash",
			repo: "https://github.com/googleapis/google-cloud-go/",
			setup: func(t *testing.T, workRoot string) func() {
				// The expected directory name is `google-cloud-go`.
				repoPath := filepath.Join(workRoot, "google-cloud-go")
				newTestGitRepoWithCommit(t, repoPath)
				return func() {
					if err := os.RemoveAll(repoPath); err != nil {
						t.Errorf("os.RemoveAll(%q) = %v; want nil", repoPath, err)
					}
				}
			},
			check: func(t *testing.T, repo *gitrepo.LocalRepository) {
				wantDir := filepath.Join(workRoot, "google-cloud-go")
				if repo.Dir != wantDir {
					t.Errorf("repo.Dir got %q, want %q", repo.Dir, wantDir)
				}
			},
		},
		{
			name:    "no repoRoot or repoURL",
			wantErr: true,
		},
		{
			name:    "with dirty repoRoot",
			repo:    dirtyRepoPath,
			wantErr: true,
		},
		{
			name:    "with repoRoot that is not a repo",
			repo:    notARepoPath,
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var cleanup func()
			if test.setup != nil {
				cleanup = test.setup(t, workRoot)
			}
			defer func() {
				if cleanup != nil {
					cleanup()
				}
			}()

			repo, err := cloneOrOpenRepo(workRoot, test.repo, test.ci, "")
			if test.wantErr {
				if err == nil {
					t.Error("cloneOrOpenLanguageRepo() expected an error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("cloneOrOpenLanguageRepo() got unexpected error: %v", err)
				return
			}
			if test.check != nil {
				if repo == nil {
					t.Fatal("cloneOrOpenLanguageRepo() returned nil repo but no error")
				}
				test.check(t, repo)
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
		repo        gitrepo.Repository
		outputDir   string
		setup       func(t *testing.T, outputDir string)
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
			repo:        newTestGitRepo(t),
			wantErr:     true,
			errContains: "not found during clean and copy",
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
			repo:        newTestGitRepo(t),
			wantErr:     true,
			errContains: "failed to clean library",
		},
		{
			name:      "copy fails on symlink",
			libraryID: "some-library",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "some-library",
						SourceRoots: []string{
							"symlink",
						},
					},
				},
			},
			repo: newTestGitRepo(t),
			setup: func(t *testing.T, outputDir string) {
				// Create a symlink in the output directory to trigger an error.
				if err := os.Symlink("target", filepath.Join(outputDir, "symlink")); err != nil {
					t.Fatalf("os.Symlink() = %v", err)
				}
			},
			wantErr:     true,
			errContains: "failed to copy",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outputDir := t.TempDir()
			if test.setup != nil {
				test.setup(t, outputDir)
			}
			err := cleanAndCopyLibrary(test.state, test.repo.GetDir(), test.libraryID, outputDir)
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				if !strings.Contains(err.Error(), test.errContains) {
					t.Errorf("want: %s, got %s", test.errContains, err.Error())
				}

				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCopyOneLibrary(t *testing.T) {
	t.Parallel()
	// Create files in src directory.
	setup := func(src string, files []string) {
		for _, relPath := range files {
			fullPath := filepath.Join(src, relPath)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				t.Error(err)
			}

			if _, err := os.Create(fullPath); err != nil {
				t.Error(err)
			}
		}
	}
	for _, test := range []struct {
		name          string
		dst           string
		src           string
		library       *config.LibraryState
		filesToCreate []string
		wantFiles     []string
		skipFiles     []string
		wantErr       bool
		wantErrMsg    string
	}{
		{
			name: "copied a library",
			dst:  filepath.Join(t.TempDir(), "dst"),
			src:  filepath.Join(t.TempDir(), "src"),
			library: &config.LibraryState{
				ID: "example-library",
				SourceRoots: []string{
					"a/path",
					"another/path",
				},
			},
			filesToCreate: []string{
				"a/path/example.txt",
				"another/path/example.txt",
				"skipped/path/example.txt",
			},
			wantFiles: []string{
				"a/path/example.txt",
				"another/path/example.txt",
			},
			skipFiles: []string{
				"skipped/path/example.txt",
			},
		},
		{
			name: "invalid src",
			dst:  os.TempDir(),
			src:  "/invalid-path",
			library: &config.LibraryState{
				ID: "example-library",
				SourceRoots: []string{
					"a-library/path",
				},
			},
			wantErr:    true,
			wantErrMsg: "failed to copy",
		},
		{
			name: "invalid dst",
			dst:  "/invalid-path",
			src:  os.TempDir(),
			library: &config.LibraryState{
				ID: "example-library",
				SourceRoots: []string{
					"a-library/path",
				},
			},
			wantErr:    true,
			wantErrMsg: "failed to copy",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !test.wantErr {
				setup(test.src, test.filesToCreate)
			}
			err := copyLibrary(test.dst, test.src, test.library)
			if test.wantErr {
				if err == nil {
					t.Errorf("copyOneLibrary() shoud fail")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %s, got: %s", test.wantErrMsg, err.Error())
				}

				return
			}
			if err != nil {
				t.Errorf("failed to run copyOneLibrary(): %s", err.Error())
			}

			for _, file := range test.wantFiles {
				fullPath := filepath.Join(test.dst, file)
				if _, err := os.Stat(fullPath); err != nil {
					t.Errorf("file %s is not copied to %s", file, test.dst)
				}
			}

			for _, file := range test.skipFiles {
				fullPath := filepath.Join(test.dst, file)
				if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
					t.Errorf("file %s should not be copied to %s", file, test.dst)
				}
			}
		})
	}
}

func TestCommitAndPush(t *testing.T) {
	for _, test := range []struct {
		name             string
		setupMockRepo    func(t *testing.T) gitrepo.Repository
		setupMockClient  func(t *testing.T) GitHubClient
		push             bool
		wantErr          bool
		expectedErrMsg   string
		validatePostTest func(t *testing.T, repo gitrepo.Repository)
	}{
		{
			name: "Push flag not specified",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				repoDir := newTestGitRepoWithCommit(t, "")
				repo, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{Dir: repoDir})
				if err != nil {
					t.Fatalf("Failed to create test repo: %v", err)
				}
				return repo
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return nil
			},
		},
		{
			name: "Happy Path",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				remote := git.NewRemote(memory.NewStorage(), &gogitConfig.RemoteConfig{
					Name: "origin",
					URLs: []string{"https://github.com/googleapis/librarian.git"},
				})
				return &MockRepository{
					Dir:          t.TempDir(),
					RemotesValue: []*git.Remote{remote},
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return &mockGitHubClient{
					createdPR: &github.PullRequestMetadata{Number: 123, Repo: &github.Repository{Owner: "test-owner", Name: "test-repo"}},
				}
			},
			push: true,
		},
		{
			name: "No GitHub Remote",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				return &MockRepository{
					Dir:          t.TempDir(),
					RemotesValue: []*git.Remote{}, // No remotes
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return nil
			},
			push:           true,
			wantErr:        true,
			expectedErrMsg: "could not find an 'origin' remote",
		},
		{
			name: "AddAll error",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				remote := git.NewRemote(memory.NewStorage(), &gogitConfig.RemoteConfig{
					Name: "origin",
					URLs: []string{"https://github.com/googleapis/librarian.git"},
				})
				return &MockRepository{
					Dir:          t.TempDir(),
					RemotesValue: []*git.Remote{remote},
					AddAllError:  errors.New("mock add all error"),
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return nil
			},
			push:           true,
			wantErr:        true,
			expectedErrMsg: "mock add all error",
		},
		{
			name: "Create branch error",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				remote := git.NewRemote(memory.NewStorage(), &gogitConfig.RemoteConfig{
					Name: "origin",
					URLs: []string{"https://github.com/googleapis/librarian.git"},
				})

				status := make(git.Status)
				status["file.txt"] = &git.FileStatus{Worktree: git.Modified}
				return &MockRepository{
					Dir:                          t.TempDir(),
					AddAllStatus:                 status,
					RemotesValue:                 []*git.Remote{remote},
					CreateBranchAndCheckoutError: errors.New("create branch error"),
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return nil
			},
			push:           true,
			wantErr:        true,
			expectedErrMsg: "create branch error",
		},
		{
			name: "Commit error",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				remote := git.NewRemote(memory.NewStorage(), &gogitConfig.RemoteConfig{
					Name: "origin",
					URLs: []string{"https://github.com/googleapis/librarian.git"},
				})

				status := make(git.Status)
				status["file.txt"] = &git.FileStatus{Worktree: git.Modified}
				return &MockRepository{
					Dir:          t.TempDir(),
					AddAllStatus: status,
					RemotesValue: []*git.Remote{remote},
					CommitError:  errors.New("commit error"),
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return nil
			},
			push:           true,
			wantErr:        true,
			expectedErrMsg: "commit error",
		},
		{
			name: "Push error",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				remote := git.NewRemote(memory.NewStorage(), &gogitConfig.RemoteConfig{
					Name: "origin",
					URLs: []string{"https://github.com/googleapis/librarian.git"},
				})

				status := make(git.Status)
				status["file.txt"] = &git.FileStatus{Worktree: git.Modified}
				return &MockRepository{
					Dir:          t.TempDir(),
					AddAllStatus: status,
					RemotesValue: []*git.Remote{remote},
					PushError:    errors.New("push error"),
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return nil
			},
			push:           true,
			wantErr:        true,
			expectedErrMsg: "push error",
		},
		{
			name: "Create pull request error",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				remote := git.NewRemote(memory.NewStorage(), &gogitConfig.RemoteConfig{
					Name: "origin",
					URLs: []string{"https://github.com/googleapis/librarian.git"},
				})

				status := make(git.Status)
				status["file.txt"] = &git.FileStatus{Worktree: git.Modified}
				return &MockRepository{
					Dir:          t.TempDir(),
					AddAllStatus: status,
					RemotesValue: []*git.Remote{remote},
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return &mockGitHubClient{
					createPullRequestErr: errors.New("create pull request error"),
				}
			},
			push:           true,
			wantErr:        true,
			expectedErrMsg: "failed to create pull request",
		},
		{
			name: "No changes to commit",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				remote := git.NewRemote(memory.NewStorage(), &gogitConfig.RemoteConfig{
					Name: "origin",
					URLs: []string{"https://github.com/googleapis/librarian.git"},
				})
				return &MockRepository{
					Dir:          t.TempDir(),
					AddAllStatus: git.Status{}, // Clean status
					RemotesValue: []*git.Remote{remote},
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return nil
			},
			push: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := test.setupMockRepo(t)
			client := test.setupMockClient(t)
			r := &generateRunner{
				cfg: &config.Config{
					Push: test.push,
				},
				repo:     repo,
				ghClient: client,
			}

			err := commitAndPush(context.Background(), r, "")

			if test.wantErr {
				if err == nil {
					t.Errorf("commitAndPush() expected error, got nil")
				} else if test.expectedErrMsg != "" && !strings.Contains(err.Error(), test.expectedErrMsg) {
					t.Errorf("commitAndPush() error = %v, expected to contain: %q", err, test.expectedErrMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("%s: commitAndPush() returned unexpected error: %v", test.name, err)
				return
			}

			if test.validatePostTest != nil {
				test.validatePostTest(t, repo)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		dst         string
		src         string
		wantSrcFile bool
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name:       "invalid src",
			src:        "/invalid-path/example.txt",
			wantErr:    true,
			wantErrMsg: "failed to open file",
		},
		{
			name:        "invalid dst path",
			src:         filepath.Join(os.TempDir(), "example.txt"),
			dst:         "/invalid-path/example.txt",
			wantSrcFile: true,
			wantErr:     true,
			wantErrMsg:  "failed to make directory",
		},
		{
			name:        "invalid dst file",
			src:         filepath.Join(os.TempDir(), "example.txt"),
			dst:         filepath.Join(os.TempDir(), "example\x00.txt"),
			wantSrcFile: true,
			wantErr:     true,
			wantErrMsg:  "failed to create file",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.wantSrcFile {
				if err := os.MkdirAll(filepath.Dir(test.src), 0755); err != nil {
					t.Error(err)
				}
				sourceFile, err := os.Create(test.src)
				if err != nil {
					t.Error(err)
				}
				if err := sourceFile.Close(); err != nil {
					t.Error(err)
				}
			}
			err := copyFile(test.dst, test.src)
			if test.wantErr {
				if err == nil {
					t.Errorf("copyFile() shoud fail")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %s, got: %s", test.wantErrMsg, err.Error())
				}

				return
			}

		})
	}
}
