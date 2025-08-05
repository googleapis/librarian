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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
			// The second word should always be the command name.
			if parts[1] != c.Name() {
				t.Errorf("invalid usage text: %q (second word should be command name %q)", c.UsageLine, c.Name())
			}
		})
	}
}

func TestDeriveRepoPath(t *testing.T) {
	for _, test := range []struct {
		name         string
		repoPath     string
		setup        func(t *testing.T, dir string)
		wantErr      bool
		wantRepoPath string
	}{
		{
			name:         "repo path provided",
			repoPath:     "/some/path",
			wantRepoPath: "/some/path",
		},
		{
			name: "empty repo path, state file exists",
			setup: func(t *testing.T, dir string) {
				stateDir := filepath.Join(dir, config.LibrarianDir)
				if err := os.MkdirAll(stateDir, 0755); err != nil {
					t.Fatal(err)
				}
				stateFile := filepath.Join(stateDir, pipelineStateFile)
				if err := os.WriteFile(stateFile, []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:    "empty repo path, no state file",
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if test.setup != nil {
				test.setup(t, tmpDir)
			}
			t.Chdir(tmpDir)

			gotRepoPath, err := deriveRepoPath(test.repoPath)
			if (err != nil) != test.wantErr {
				t.Errorf("deriveRepoPath() error = %v, wantErr %v", err, test.wantErr)
				return
			}

			wantPath := test.wantRepoPath
			if wantPath == "" && !test.wantErr {
				wantPath = tmpDir
			}

			if diff := cmp.Diff(wantPath, gotRepoPath); diff != "" {
				t.Errorf("deriveRepoPath() mismatch (-want +got):\n%s", diff)
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

func TestCreateWorkRoot(t *testing.T) {
	now := time.Now()
	for _, test := range []struct {
		name             string
		workRootOverride string
		setup            func(t *testing.T) (string, func())
		wantErr          bool
	}{
		{
			name:             "with override",
			workRootOverride: "/fake/path",
			setup: func(t *testing.T) (string, func()) {
				return "/fake/path", func() {}
			},
		},
		{
			name: "without override, new dir",
			setup: func(t *testing.T) (string, func()) {
				expectedPath := filepath.Join(os.TempDir(), fmt.Sprintf("librarian-%s", formatTimestamp(now)))
				return expectedPath, func() {
					if err := os.RemoveAll(expectedPath); err != nil {
						t.Errorf("os.RemoveAll(%q) = %v; want nil", expectedPath, err)
					}
				}
			},
		},
		{
			name: "without override, dir exists",
			setup: func(t *testing.T) (string, func()) {
				expectedPath := filepath.Join(os.TempDir(), fmt.Sprintf("librarian-%s", formatTimestamp(now)))
				if err := os.Mkdir(expectedPath, 0755); err != nil {
					t.Fatalf("failed to create test dir: %v", err)
				}
				return expectedPath, func() {
					if err := os.RemoveAll(expectedPath); err != nil {
						t.Errorf("os.RemoveAll(%q) = %v; want nil", expectedPath, err)
					}
				}
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			want, cleanup := test.setup(t)
			defer cleanup()

			got, err := createWorkRoot(now, test.workRootOverride)
			if test.wantErr {
				if err == nil {
					t.Error("createWorkRoot() expected an error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("createWorkRoot() got unexpected error: %v", err)
				return
			}

			if got != want {
				t.Errorf("createWorkRoot() = %v, want %v", got, want)
			}
			if test.workRootOverride == "" {
				if _, err := os.Stat(got); os.IsNotExist(err) {
					t.Errorf("createWorkRoot() did not create directory %v", got)
				}
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

			repo, err := cloneOrOpenRepo(workRoot, test.repo, test.ci)
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
				repoDir := newTestGitRepoWithCommit(t, "")
				// Add remote so FetchGitHubRepoFromRemote succeeds.
				cmd := exec.Command("git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
				cmd.Dir = repoDir
				if err := cmd.Run(); err != nil {
					t.Fatalf("git remote add: %v", err)
				}
				// Add a file to make the repo dirty, so there's something to commit.
				if err := os.WriteFile(filepath.Join(repoDir, "new-file.txt"), []byte("new content"), 0644); err != nil {
					t.Fatalf("WriteFile: %v", err)
				}
				repo, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{Dir: repoDir})
				if err != nil {
					t.Fatalf("Failed to create test repo: %v", err)
				}
				return repo
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return &mockGitHubClient{
					createdPR: &github.PullRequestMetadata{Number: 123, Repo: &github.Repository{Owner: "test-owner", Name: "test-repo"}},
				}
			},
			push: true,
			validatePostTest: func(t *testing.T, repo gitrepo.Repository) {
				localRepo, ok := repo.(*gitrepo.LocalRepository)
				if !ok {
					t.Fatalf("Expected *gitrepo.LocalRepository, got %T", repo)
				}
				isClean, err := localRepo.IsClean()
				if err != nil {
					t.Fatalf("Failed to check repo status: %v", err)
				}
				if !isClean {
					t.Errorf("Expected repository to be clean after commit, but it's dirty")
				}
			},
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
