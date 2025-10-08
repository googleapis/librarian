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

package gitrepo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/go-git/go-git/v5"
	goGitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-cmp/cmp"
)

func TestNewRepository(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	remoteDir := filepath.Join(tmpDir, "remote")
	if err := os.Mkdir(remoteDir, 0755); err != nil {
		t.Fatal(err)
	}
	remoteRepo, err := git.PlainInit(remoteDir, false)
	if err != nil {
		t.Fatal(err)
	}
	w, err := remoteRepo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(remoteDir, "README.md"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Add("README.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com"},
	}); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name    string
		opts    *RepositoryOptions
		wantDir string
		wantErr bool
		initGit bool
		setup   func(t *testing.T) (cleanup func())
	}{
		{
			name:    "no dir",
			opts:    &RepositoryOptions{},
			wantErr: true,
		},
		{
			name: "open existing",
			opts: &RepositoryOptions{
				Dir: tmpDir,
			},
			wantDir: tmpDir,
			initGit: true,
		},
		{
			name: "open existing not valid git dir",
			opts: &RepositoryOptions{
				Dir: filepath.Join(tmpDir, "non-git-dir"),
			},
			wantErr: true,
			setup: func(t *testing.T) func() {
				if err := os.Mkdir(filepath.Join(tmpDir, "non-git-dir"), 0755); err != nil {
					t.Fatalf("failed to create test dir: %v", err)
				}
				return func() {}
			},
		},
		{
			name: "clone maybe",
			opts: &RepositoryOptions{
				Dir:          filepath.Join(tmpDir, "clone-maybe"),
				MaybeClone:   true,
				RemoteURL:    remoteDir,
				RemoteBranch: "master",
			},
			wantDir: filepath.Join(tmpDir, "clone-maybe"),
		},
		{
			name: "maybe clone with existing repo",
			opts: &RepositoryOptions{
				Dir:        filepath.Join(tmpDir, "existing-repo"),
				MaybeClone: true,
			},
			wantDir: filepath.Join(tmpDir, "existing-repo"),
			initGit: true,
		},
		{
			name: "clone maybe no remote url",
			opts: &RepositoryOptions{
				Dir:          filepath.Join(tmpDir, "clone-maybe-no-remote"),
				MaybeClone:   true,
				RemoteBranch: "main",
			},
			wantErr: true,
		},
		{
			name: "clone maybe no remote branch",
			opts: &RepositoryOptions{
				Dir:        filepath.Join(tmpDir, "clone-maybe-no-remote"),
				MaybeClone: true,
				RemoteURL:  remoteDir,
			},
			wantErr: true,
		},
		{
			name: "stat error",
			opts: &RepositoryOptions{
				Dir:        filepath.Join(tmpDir, "unreadable/repo"),
				MaybeClone: true,
			},
			wantErr: true,
			setup: func(t *testing.T) func() {
				unreadableDir := filepath.Join(tmpDir, "unreadable")
				if err := os.Mkdir(unreadableDir, 0000); err != nil {
					t.Fatalf("os.Mkdir() failed: %v", err)
				}
				return func() {
					if err := os.Chmod(unreadableDir, 0755); err != nil {
						t.Logf("failed to restore permissions on %s: %v", unreadableDir, err)
					}
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if test.setup != nil {
				cleanup := test.setup(t)
				defer cleanup()
			}
			if test.initGit {
				if _, err := git.PlainInit(test.opts.Dir, false); err != nil {
					t.Fatal(err)
				}
			}
			got, err := NewRepository(test.opts)
			if (err != nil) != test.wantErr {
				t.Errorf("NewRepository() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if err != nil {
				return
			}
			if got.Dir != test.wantDir {
				t.Errorf("NewRepository() got = %v, want %v", got.Dir, test.wantDir)
			}
		})
	}
}

func TestIsClean(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		setup     func(t *testing.T, dir string, w *git.Worktree)
		wantClean bool
	}{
		{
			name:      "initial state is clean",
			setup:     func(t *testing.T, dir string, w *git.Worktree) {},
			wantClean: true,
		},
		{
			name: "untracked file is not clean",
			setup: func(t *testing.T, dir string, w *git.Worktree) {
				filePath := filepath.Join(dir, "untracked.txt")
				if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
			},
			wantClean: false,
		},
		{
			name: "added file is not clean",
			setup: func(t *testing.T, dir string, w *git.Worktree) {
				filePath := filepath.Join(dir, "added.txt")
				if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
				if _, err := w.Add("added.txt"); err != nil {
					t.Fatalf("failed to add file: %v", err)
				}
			},
			wantClean: false,
		},
		{
			name: "committed file is clean",
			setup: func(t *testing.T, dir string, w *git.Worktree) {
				filePath := filepath.Join(dir, "committed.txt")
				if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
				if _, err := w.Add("committed.txt"); err != nil {
					t.Fatalf("failed to add file: %v", err)
				}
				_, err := w.Commit("commit", &git.CommitOptions{
					Author: &object.Signature{Name: "Test", Email: "test@example.com"},
				})
				if err != nil {
					t.Fatalf("failed to commit: %v", err)
				}
			},
			wantClean: true,
		},
		{
			name: "modified file is not clean",
			setup: func(t *testing.T, dir string, w *git.Worktree) {
				// First, commit a file.
				filePath := filepath.Join(dir, "modified.txt")
				if err := os.WriteFile(filePath, []byte("initial"), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
				if _, err := w.Add("modified.txt"); err != nil {
					t.Fatalf("failed to add file: %v", err)
				}
				_, err := w.Commit("commit", &git.CommitOptions{
					Author: &object.Signature{Name: "Test", Email: "test@example.com"},
				})
				if err != nil {
					t.Fatalf("failed to commit: %v", err)
				}

				// Now modify it.
				if err := os.WriteFile(filePath, []byte("modified"), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
			},
			wantClean: false,
		},
		{
			name: "deleted file is not clean",
			setup: func(t *testing.T, dir string, w *git.Worktree) {
				// First, commit a file.
				filePath := filepath.Join(dir, "deleted.txt")
				if err := os.WriteFile(filePath, []byte("initial"), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
				if _, err := w.Add("deleted.txt"); err != nil {
					t.Fatalf("failed to add file: %v", err)
				}
				_, err := w.Commit("commit", &git.CommitOptions{
					Author: &object.Signature{Name: "Test", Email: "test@example.com"},
				})
				if err != nil {
					t.Fatalf("failed to commit: %v", err)
				}

				// Now delete it.
				if err := os.Remove(filePath); err != nil {
					t.Fatalf("failed to remove file: %v", err)
				}
			},
			wantClean: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repo, dir := initTestRepo(t)
			w, err := repo.Worktree()
			if err != nil {
				t.Fatalf("failed to get worktree: %v", err)
			}

			r := &LocalRepository{
				Dir:  dir,
				repo: repo,
			}

			test.setup(t, dir, w)
			gotClean, err := r.IsClean()
			if err != nil {
				t.Fatalf("IsClean() returned an error: %v", err)
			}

			if gotClean != test.wantClean {
				t.Errorf("IsClean() = %v, want %v", gotClean, test.wantClean)
			}
		})
	}
}

func TestAddAll(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name              string
		setup             func(t *testing.T, dir string)
		wantStatusIsClean bool
		wantErr           bool
	}{
		{
			name: "add a new file",
			setup: func(t *testing.T, dir string) {
				filePath := filepath.Join(dir, "new_file.txt")
				if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
			},
			wantStatusIsClean: false,
		},
		{
			name: "no files to add",
			setup: func(t *testing.T, dir string) {
				// Do nothing, repo is clean.
			},
			wantStatusIsClean: true,
		},
		{
			name: "add unreadable file",
			setup: func(t *testing.T, dir string) {
				filePath := filepath.Join(dir, "unreadable_file.txt")
				if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
				// Make file unreadable to cause an error during `git add`.
				if err := os.Chmod(filePath, 0222); err != nil {
					t.Fatalf("failed to chmod file: %v", err)
				}
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			gogitRepo, dir := initTestRepo(t)
			r := &LocalRepository{
				Dir:  dir,
				repo: gogitRepo,
			}

			test.setup(t, dir)

			err := r.AddAll()
			if (err != nil) != test.wantErr {
				t.Errorf("AddAll() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if err != nil {
				return
			}
			isClean, err := r.IsClean()
			if err != nil {
				t.Fatalf("IsClean() returned an error: %v", err)
			}
			if isClean != test.wantStatusIsClean {
				t.Errorf("AddAll() status.IsClean() = %v, want %v", isClean, test.wantStatusIsClean)
			}
		})
	}

}

func TestCommit(t *testing.T) {
	t.Parallel()
	name, email := "tester", "tester@example.com"
	// setupRepo is a helper to create a repository with an initial commit.
	setupRepo := func(t *testing.T) *LocalRepository {
		t.Helper()
		goGitRepo, dir := initTestRepo(t)

		author := struct {
			Name  string
			Email string
		}{
			Name:  name,
			Email: email,
		}
		config, err := goGitRepo.Config()
		if err != nil {
			t.Fatalf("gitRepo.Config failed: %v", err)
		}
		config.User = author
		if err := goGitRepo.SetConfig(config); err != nil {
			t.Fatalf("gitRepo.SetConfig failed: %v", err)
		}

		w, err := goGitRepo.Worktree()
		if err != nil {
			t.Fatalf("Worktree() failed: %v", err)
		}
		if _, err := w.Commit("initial commit", &git.CommitOptions{
			AllowEmptyCommits: true,
			Author:            &object.Signature{Name: "Test", Email: "test@example.com"},
		}); err != nil {
			t.Fatalf("initial commit failed: %v", err)
		}
		return &LocalRepository{Dir: dir, repo: goGitRepo}
	}

	for _, test := range []struct {
		name       string
		setup      func(t *testing.T) *LocalRepository
		commitMsg  string
		wantErr    bool
		wantErrMsg string
		check      func(t *testing.T, repo *LocalRepository, commitMsg string)
	}{
		{
			name: "successful commit",
			setup: func(t *testing.T) *LocalRepository {
				repo := setupRepo(t)
				// Add a file to be committed.
				filePath := filepath.Join(repo.Dir, "new.txt")
				if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
					t.Fatalf("os.WriteFile failed: %v", err)
				}
				w, err := repo.repo.Worktree()
				if err != nil {
					t.Fatalf("Worktree() failed: %v", err)
				}
				if _, err := w.Add("new.txt"); err != nil {
					t.Fatalf("w.Add failed: %v", err)
				}
				return repo
			},
			commitMsg: "feat: add new file",
			check: func(t *testing.T, repo *LocalRepository, commitMsg string) {
				head, err := repo.repo.Head()
				if err != nil {
					t.Fatalf("repo.repo.Head() failed: %v", err)
				}
				commit, err := repo.repo.CommitObject(head.Hash())
				if err != nil {
					t.Fatalf("repo.repo.CommitObject() failed: %v", err)
				}
				if commit.Message != commitMsg {
					t.Errorf("Commit() message = %q, want %q", commit.Message, commitMsg)
				}
				author := commit.Author
				if author.Name != "tester" {
					t.Errorf("Commit() author name = %q, want %q", author.Name, "tester")
				}
				if author.Email != "tester@example.com" {
					t.Errorf("Commit() author email = %q, want %q", author.Email, "tester@example.com")
				}
			},
		},
		{
			name: "clean repository",
			setup: func(t *testing.T) *LocalRepository {
				return setupRepo(t)
			},
			commitMsg:  "no-op",
			wantErr:    true,
			wantErrMsg: "no modifications to commit",
		},
		{
			name: "worktree error",
			setup: func(t *testing.T) *LocalRepository {
				dir := t.TempDir()
				// Create a bare repository which has no worktree.
				goGitRepo, err := git.PlainInit(dir, true)
				if err != nil {
					t.Fatalf("git.PlainInit failed: %v", err)
				}
				return &LocalRepository{Dir: dir, repo: goGitRepo}
			},
			commitMsg:  "any message",
			wantErr:    true,
			wantErrMsg: "worktree not available",
		},
		{
			name: "status error",
			setup: func(t *testing.T) *LocalRepository {
				repo := setupRepo(t)
				// Add a file to make the worktree dirty.
				filePath := filepath.Join(repo.Dir, "new.txt")
				if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
					t.Fatalf("os.WriteFile failed: %v", err)
				}
				w, err := repo.repo.Worktree()
				if err != nil {
					t.Fatalf("Worktree() failed: %v", err)
				}
				if _, err := w.Add("new.txt"); err != nil {
					t.Fatalf("w.Add failed: %v", err)
				}

				// Make the worktree unreadable to cause worktree.Status() to fail.
				if err := os.Chmod(repo.Dir, 0000); err != nil {
					t.Fatalf("os.Chmod failed: %v", err)
				}
				t.Cleanup(func() {
					if err := os.Chmod(repo.Dir, 0755); err != nil {
						t.Logf("failed to restore permissions: %v", err)
					}
				})
				return repo
			},
			commitMsg:  "any message",
			wantErr:    true,
			wantErrMsg: "permission denied",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repo := test.setup(t)

			err := repo.Commit(test.commitMsg)

			if test.wantErr {
				if err == nil {
					t.Fatalf("Commit() expected error, got nil")
				}
				if test.wantErrMsg != "" && !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("Commit() error = %q, want to contain %q", err.Error(), test.wantErrMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("Commit() unexpected error = %v", err)
			}

			if test.check != nil {
				test.check(t, repo, test.commitMsg)
			}
		})
	}
}

func TestRemotes(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name         string
		setupRemotes map[string][]string
		wantErr      bool
	}{
		{
			name:         "no remotes",
			setupRemotes: map[string][]string{},
		},
		{
			name: "single remote",
			setupRemotes: map[string][]string{
				"origin": {"https://github.com/test/repo.git"},
			},
		},
		{
			name: "multiple remotes with multiple URLs",
			setupRemotes: map[string][]string{
				"origin":   {"https://github.com/test/origin.git"},
				"upstream": {"https://github.com/test/upstream.git", "git@github.com:test/upstream.git"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			gogitRepo, dir := initTestRepo(t)

			for name, urls := range test.setupRemotes {
				if _, err := gogitRepo.CreateRemote(&goGitConfig.RemoteConfig{
					Name: name,
					URLs: urls,
				}); err != nil {
					t.Fatalf("CreateRemote failed: %v", err)
				}
			}

			repo := &LocalRepository{Dir: dir, repo: gogitRepo}
			got, err := repo.Remotes()
			if (err != nil) != test.wantErr {
				t.Errorf("Remotes() error = %v, wantErr %v", err, test.wantErr)
			}

			gotRemotes := make(map[string][]string)
			for _, r := range got {
				gotRemotes[r.Name] = r.URLs
			}
			if diff := cmp.Diff(test.setupRemotes, gotRemotes); diff != "" {
				t.Errorf("Remotes() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetCommit(t *testing.T) {
	t.Parallel()
	setup := func(t *testing.T, dir string) string {
		gitRepo, err := git.PlainInit(dir, false)
		if err != nil {
			t.Fatalf("git.PlainInit failed: %v", err)
		}
		w, err := gitRepo.Worktree()
		if err != nil {
			t.Fatalf("Worktree() failed: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Add("README.md"); err != nil {
			t.Fatal(err)
		}
		commitHash, err := w.Commit("initial commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Test",
				Email: "test@example.com",
				When:  time.Now(),
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		return commitHash.String()
	}

	for _, test := range []struct {
		name       string
		commitHash string
		want       *Commit
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "get a commit",
			want: &Commit{
				Message: "initial commit",
			},
		},
		{
			name:       "failed to get a commit",
			commitHash: "wrong-sha",
			wantErr:    true,
			wantErrMsg: "object not found",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			commitHash := setup(t, dir)
			if test.commitHash != "" {
				commitHash = test.commitHash
			}

			repo, err := NewRepository(&RepositoryOptions{Dir: dir})
			if err != nil {
				t.Error(err)
			}

			got, err := repo.GetCommit(commitHash)
			if test.wantErr {
				if err == nil {
					t.Fatal("GetCommit() should fail")
				}
				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message %s, got %s", test.wantErrMsg, err.Error())
				}

				return
			}
			if err != nil {
				t.Fatalf("GetCommit() failed: %v", err)
			}

			test.want.Hash = plumbing.NewHash(commitHash)
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(Commit{}, "When")); diff != "" {
				t.Errorf("GetDir() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHeadHash(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		setup   func(t *testing.T, dir string)
		wantErr bool
	}{
		{
			name: "success",
			setup: func(t *testing.T, dir string) {
				gitRepo, err := git.PlainInit(dir, false)
				if err != nil {
					t.Fatalf("git.PlainInit failed: %v", err)
				}
				w, err := gitRepo.Worktree()
				if err != nil {
					t.Fatalf("Worktree() failed: %v", err)
				}
				if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
				if _, err := w.Add("README.md"); err != nil {
					t.Fatal(err)
				}
				if _, err := w.Commit("initial commit", &git.CommitOptions{
					Author: &object.Signature{Name: "Test", Email: "test@example.com"},
				}); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "error",
			setup: func(t *testing.T, dir string) {
				if _, err := git.PlainInit(dir, false); err != nil {
					t.Fatalf("git.PlainInit failed: %v", err)
				}
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			test.setup(t, dir)
			repo, err := NewRepository(&RepositoryOptions{Dir: dir})
			if err != nil {
				t.Fatalf("NewRepository() failed: %v", err)
			}
			_, err = repo.HeadHash()
			if (err != nil) != test.wantErr {
				t.Errorf("HeadHash() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
func TestGetDir(t *testing.T) {
	t.Parallel()
	want := "/test/dir"
	repo := &LocalRepository{
		Dir: want,
	}

	got := repo.GetDir()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GetDir() mismatch (-want +got):\n%s", diff)
	}
}

func TestGetHashForPathOrEmpty(t *testing.T) {
	t.Parallel()

	setupInitialRepo := func(t *testing.T) (*git.Repository, *object.Commit) {
		t.Helper()
		repo, _ := initTestRepo(t)
		commit := createAndCommit(t, repo, "initial.txt", []byte("initial content"), "initial commit")
		return repo, commit
	}

	for _, test := range []struct {
		name     string
		setup    func(t *testing.T) (commit *object.Commit, path string)
		wantHash func(commit *object.Commit, path string) string
		wantErr  bool
	}{
		{
			name: "existing file",
			setup: func(t *testing.T) (*object.Commit, string) {
				_, commit := setupInitialRepo(t)
				return commit, "initial.txt"
			},
			wantHash: func(commit *object.Commit, path string) string {
				tree, err := commit.Tree()
				if err != nil {
					t.Fatalf("failed to get tree: %v", err)
				}
				entry, err := tree.FindEntry(path)
				if err != nil {
					t.Fatalf("failed to find entry for path %q: %v", path, err)
				}
				return entry.Hash.String()
			},
		},
		{
			name: "existing directory",
			setup: func(t *testing.T) (*object.Commit, string) {
				repo, _ := setupInitialRepo(t)
				// Create a directory and a file inside it to ensure the directory gets a hash
				_ = createAndCommit(t, repo, "my_dir/file_in_dir.txt", []byte("content of file in dir"), "add dir and file")
				head, err := repo.Head()
				if err != nil {
					t.Fatalf("repo.Head failed: %v", err)
				}
				commit, err := repo.CommitObject(head.Hash())
				if err != nil {
					t.Fatalf("repo.CommitObject failed: %v", err)
				}
				return commit, "my_dir"
			},
			wantHash: func(commit *object.Commit, path string) string {
				tree, err := commit.Tree()
				if err != nil {
					t.Fatalf("failed to get tree: %v", err)
				}
				entry, err := tree.FindEntry(path)
				if err != nil {
					t.Fatalf("failed to find entry for path %q: %v", path, err)
				}
				return entry.Hash.String()
			},
		},
		{
			name: "non-existent file",
			setup: func(t *testing.T) (*object.Commit, string) {
				_, commit := setupInitialRepo(t)
				return commit, "non_existent_file.txt"
			},
			wantHash: func(commit *object.Commit, path string) string {
				return ""
			},
		},
		{
			name: "non-existent directory",
			setup: func(t *testing.T) (*object.Commit, string) {
				_, commit := setupInitialRepo(t)
				return commit, "non_existent_dir"
			},
			wantHash: func(commit *object.Commit, path string) string {
				return ""
			},
		},
		{
			name: "file in subdirectory",
			setup: func(t *testing.T) (*object.Commit, string) {
				repo, _ := setupInitialRepo(t)
				_ = createAndCommit(t, repo, "another_dir/sub_dir/nested_file.txt", []byte("nested content"), "add nested file")
				head, err := repo.Head()
				if err != nil {
					t.Fatalf("repo.Head failed: %v", err)
				}
				commit, err := repo.CommitObject(head.Hash())
				if err != nil {
					t.Fatalf("repo.CommitObject failed: %v", err)
				}
				return commit, "another_dir/sub_dir/nested_file.txt"
			},
			wantHash: func(commit *object.Commit, path string) string {
				tree, err := commit.Tree()
				if err != nil {
					t.Fatalf("failed to get tree: %v", err)
				}
				entry, err := tree.FindEntry(path)
				if err != nil {
					t.Fatalf("failed to find entry for path %q: %v", path, err)
				}
				return entry.Hash.String()
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			commit, path := test.setup(t)

			got, err := getHashForPathOrEmpty(commit, path)
			if (err != nil) != test.wantErr {
				t.Errorf("getHashForPathOrEmpty() error = %v, wantErr %v", err, test.wantErr)
				return
			}

			wantHash := test.wantHash(commit, path)
			if diff := cmp.Diff(wantHash, got); diff != "" {
				t.Errorf("getHashForPathOrEmpty() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestChangedFilesInCommit(t *testing.T) {
	t.Parallel()
	r, commitHashes := setupRepoForChangedFilesTest(t)

	for _, test := range []struct {
		name       string
		commitHash string
		wantFiles  []string
		wantErr    bool
	}{
		{
			name:       "commit 1",
			commitHash: commitHashes["commit 1"],
			wantFiles:  []string{"file1.txt"},
		},
		{
			name:       "commit 2",
			commitHash: commitHashes["commit 2"],
			wantFiles:  []string{"file1.txt"},
		},
		{
			name:       "commit 3",
			commitHash: commitHashes["commit 3"],
			wantFiles:  []string{"file2.txt"},
		},
		{
			name:       "invalid commit hash",
			commitHash: "invalid",
			wantErr:    true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			gotFiles, err := r.ChangedFilesInCommit(test.commitHash)
			if (err != nil) != test.wantErr {
				t.Errorf("ChangedFilesInCommit() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.wantFiles, gotFiles); diff != "" {
				t.Errorf("ChangedFilesInCommit() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetCommitsForPathsSinceCommit(t *testing.T) {
	t.Parallel()

	repo, commits := setupRepoForGetCommitsTest(t)

	for _, test := range []struct {
		name          string
		paths         []string
		tagName       string
		sinceCommit   string
		wantCommits   []string
		wantErr       bool
		wantErrPhrase string
	}{
		{
			name:        "one path, one commit",
			paths:       []string{"file2.txt"},
			sinceCommit: commits["commit1"],
			wantCommits: []string{"feat: commit 2"},
		},
		{
			name:        "all paths, all commits",
			paths:       []string{"file1.txt", "file2.txt", "file3.txt"},
			sinceCommit: "",
			// The current implementation skips the initial commit.
			wantCommits: []string{"feat: commit 3", "feat: commit 2"},
		},
		{
			name:        "no matching commits",
			paths:       []string{"non-existent-file.txt"},
			sinceCommit: "",
			wantCommits: []string{},
		},
		{
			name:          "no paths specified",
			paths:         []string{},
			tagName:       "v1.0.0",
			wantCommits:   []string{},
			wantErr:       true,
			wantErrPhrase: "no paths to check for commits",
		},
		{
			name:          "since commit not found",
			paths:         []string{"file1.txt"},
			sinceCommit:   "nonexistenthash",
			wantCommits:   []string{},
			wantErr:       true,
			wantErrPhrase: "did not find commit",
		},
		{
			name:        "root path matches all commits",
			paths:       []string{"."},
			sinceCommit: "",
			// The current implementation skips the initial commit.
			wantCommits: []string{"feat: commit 3", "feat: commit 2"},
		},
	} {

		t.Run(test.name, func(t *testing.T) {
			var (
				gotCommits []*Commit
				err        error
			)

			gotCommits, err = repo.GetCommitsForPathsSinceCommit(test.paths, test.sinceCommit)

			if (err != nil) != test.wantErr {
				t.Errorf("GetCommitsForPathsSinceCommit() error = %v, wantErr %v", err, test.wantErr)
				return
			}

			if test.wantErr {
				if err == nil {
					t.Fatalf("%s should return error", test.name)
				}
				if !strings.Contains(err.Error(), test.wantErrPhrase) {
					t.Errorf("GetCommitsForPathsSinceCommit() returned error %q, want to contain %q", err.Error(), test.wantErrPhrase)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			gotCommitMessages := []string{}
			for _, c := range gotCommits {
				gotCommitMessages = append(gotCommitMessages, strings.Split(c.Message, "\n")[0])
			}

			if diff := cmp.Diff(test.wantCommits, gotCommitMessages); diff != "" {
				t.Errorf("GetCommitsForPathsSinceCommit() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetCommitsForPathsSinceTag(t *testing.T) {
	t.Parallel()

	repo, _ := setupRepoForGetCommitsTest(t)

	for _, test := range []struct {
		name          string
		paths         []string
		tagName       string
		sinceCommit   string
		wantCommits   []string
		wantErr       bool
		wantErrPhrase string
	}{
		{
			name:        "all paths, multiple commits",
			paths:       []string{"file2.txt", "file3.txt"},
			tagName:     "v1.0.0",
			wantCommits: []string{"feat: commit 3", "feat: commit 2"},
		},
		{
			name:          "invalid tag",
			paths:         []string{"file2.txt"},
			tagName:       "non-existent-tag",
			wantCommits:   []string{},
			wantErr:       true,
			wantErrPhrase: "failed to find tag",
		},
	} {

		t.Run(test.name, func(t *testing.T) {
			var (
				gotCommits []*Commit
				err        error
			)
			gotCommits, err = repo.GetCommitsForPathsSinceTag(test.paths, test.tagName)

			if test.wantErr {
				if err == nil {
					t.Fatalf("%s should return error", test.name)
				}
				if !strings.Contains(err.Error(), test.wantErrPhrase) {
					t.Errorf("GetCommitsForPathsSinceTag() returned error %q, want to contain %q", err.Error(), test.wantErrPhrase)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			gotCommitMessages := []string{}
			for _, c := range gotCommits {
				gotCommitMessages = append(gotCommitMessages, strings.Split(c.Message, "\n")[0])
			}

			if diff := cmp.Diff(test.wantCommits, gotCommitMessages); diff != "" {
				t.Errorf("GetCommitsForPathsSinceTag() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateBranchAndCheckout(t *testing.T) {
	for _, test := range []struct {
		name          string
		branchName    string
		wantErr       bool
		wantErrPhrase string
	}{
		{
			name:       "works",
			branchName: "test-branch",
		},
		{
			name:          "invalid branch name",
			branchName:    "invalid branch name",
			wantErr:       true,
			wantErrPhrase: "invalid",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, _ := setupRepoForGetCommitsTest(t)
			err := repo.CreateBranchAndCheckout(test.branchName)
			if test.wantErr {
				if err == nil {
					t.Fatalf("%s should return error", test.name)
				}
				if !strings.Contains(err.Error(), test.wantErrPhrase) {
					t.Errorf("CreateBranchAndCheckout() returned error %q, want to contain %q", err.Error(), test.wantErrPhrase)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			head, _ := repo.repo.Head()
			if diff := cmp.Diff(test.branchName, head.Name().Short()); diff != "" {
				t.Errorf("CreateBranchAndCheckout() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRestore(t *testing.T) {
	for _, test := range []struct {
		name          string
		paths         []string
		wantErr       bool
		wantErrPhrase string
	}{
		{
			name: "restore files in paths",
			paths: []string{
				"first/path",
				"second/path",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, dir := initTestRepo(t)
			localRepo := &LocalRepository{
				Dir:  dir,
				repo: repo,
			}
			// Create files in test.paths and commit the change.
			for _, path := range test.paths {
				file := filepath.Join(path, "example.txt")
				createAndCommit(t, repo, file, []byte("old content"), fmt.Sprintf("commit path, %s", path))
			}

			// Change file contents.
			for _, path := range test.paths {
				file := filepath.Join(dir, path, "example.txt")
				if err := os.WriteFile(file, []byte("new content"), 0755); err != nil {
					t.Fatal(err)
				}
				// Create untracked files.
				untrackedFile := filepath.Join(dir, path, "untracked.txt")
				if err := os.WriteFile(untrackedFile, []byte("new content"), 0755); err != nil {
					t.Fatal(err)
				}
			}

			err := localRepo.Restore(test.paths)
			if test.wantErr {
				if err == nil {
					t.Fatalf("%s should return error", test.name)
				}
				if !strings.Contains(err.Error(), test.wantErrPhrase) {
					t.Errorf("Restore() returned error %q, want to contain %q", err.Error(), test.wantErrPhrase)
				}
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			for _, path := range test.paths {
				got, err := os.ReadFile(filepath.Join(dir, path, "example.txt"))
				if err != nil {
					t.Fatal(err)
				}
				// Verify file contents are restored.
				if diff := cmp.Diff("old content", string(got)); diff != "" {
					t.Errorf("Restore() mismatch (-want +got):\n%s", diff)
				}
				// Verify the untracked files are untouched.
				untrackedFile := filepath.Join(dir, path, "untracked.txt")
				if _, err := os.Stat(untrackedFile); err != nil {
					t.Errorf("untracked file, %s should not be removed", untrackedFile)
				}
			}
		})
	}
}

func TestCleanUntracked(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		paths []string
	}{
		{
			name: "remove_untracked_files_in_paths",
			paths: []string{
				"first/path",
				"second/path",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repo, dir := initTestRepo(t)
			localRepo := &LocalRepository{
				Dir:  dir,
				repo: repo,
			}

			// Create tracked files.
			for _, path := range test.paths {
				relPath := filepath.Join(dir, path)
				if err := os.MkdirAll(relPath, 0755); err != nil {
					t.Fatal(err)
				}
				trackedFile := filepath.Join(relPath, "tracked.txt")
				if err := os.WriteFile(trackedFile, []byte("new content"), 0755); err != nil {
					t.Fatal(err)
				}
			}

			if err := localRepo.AddAll(); err != nil {
				t.Fatal(err)
			}

			// Create untracked files.
			for _, path := range test.paths {
				relPath := filepath.Join(dir, path)
				if err := os.MkdirAll(relPath, 0755); err != nil {
					t.Fatal(err)
				}
				untrackedFile := filepath.Join(relPath, "untracked.txt")
				if err := os.WriteFile(untrackedFile, []byte("new content"), 0755); err != nil {
					t.Fatal(err)
				}
			}

			if err := localRepo.CleanUntracked(test.paths); err != nil {
				t.Error(err)

				return
			}

			for _, path := range test.paths {
				// Verify the untracked files are removed.
				untrackedFile := filepath.Join(dir, path, "untracked.txt")
				if _, err := os.Stat(untrackedFile); !os.IsNotExist(err) {
					t.Errorf("untracked file, %s should be removed", untrackedFile)
				}
				// Verify the tracked files are untouched.
				trackedFile := filepath.Join(dir, path, "tracked.txt")
				if _, err := os.Stat(trackedFile); err != nil {
					t.Errorf("tracked file, %s should not be touched: %q", trackedFile, err)
				}
			}
		})
	}
}

func TestGetLatestCommit(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		path        string
		setup       func(t *testing.T, repo *git.Repository, path string)
		want        *Commit
		wantErr     bool
		wantErrType error
	}{
		{
			name: "get_latest_commit_of_a_path",
			path: "a/path/example.txt",
			setup: func(t *testing.T, repo *git.Repository, path string) {
				createAndCommit(t, repo, path, []byte("1st content"), "first commit")
				createAndCommit(t, repo, path, []byte("2nd content"), "second commit")
			},
			want: &Commit{
				Message: "second commit",
			},
		},
		{
			name: "no_commit_of_a_path",
			path: "a/path/example.txt",
			setup: func(t *testing.T, repo *git.Repository, path string) {
				// Do nothing.
			},
			wantErr:     true,
			wantErrType: plumbing.ErrReferenceNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repo, dir := initTestRepo(t)
			test.setup(t, repo, test.path)
			localRepo := &LocalRepository{Dir: dir, repo: repo}
			got, err := localRepo.GetLatestCommit(test.path)
			if test.wantErr {
				if err == nil {
					t.Error("GetLatestCommit should return an error")
					return
				}
				if !errors.Is(err, test.wantErrType) {
					t.Errorf("unexpected error type: got %v, want %v", err, test.wantErrType)
				}

				return
			}
			if err != nil {
				t.Error(err)
				return
			}
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(Commit{}, "When", "Hash")); diff != "" {
				t.Errorf("GetLatestCommit() mismatch in %s (-want +got):\n%s", test.name, diff)
			}
		})
	}
}

// initTestRepo creates a new git repository in a temporary directory.
func initTestRepo(t *testing.T) (*git.Repository, string) {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("git.PlainInit failed: %v", err)
	}
	return repo, dir
}

// createAndCommit creates and commits a file or directory.
func createAndCommit(t *testing.T, repo *git.Repository, path string, content []byte, commitMsg string) *object.Commit {
	t.Helper()
	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree() failed: %v", err)
	}

	fullPath := filepath.Join(w.Filesystem.Root(), path)
	if content != nil { // It's a file
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("os.MkdirAll failed: %v", err)
		}
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			t.Fatalf("os.WriteFile failed: %v", err)
		}
	} else { // It's a directory
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			t.Fatalf("os.MkdirAll failed: %v", err)
		}
	}

	if _, err := w.Add(path); err != nil {
		t.Fatalf("w.Add failed: %v", err)
	}
	hash, err := w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com"},
	})
	if err != nil {
		t.Fatalf("w.Commit failed: %v", err)
	}
	commit, err := repo.CommitObject(hash)
	if err != nil {
		t.Fatalf("repo.CommitObject failed: %v", err)
	}
	return commit
}

// setupRepoForChangedFilesTest sets up a repository with a series of commits for testing.
// It returns the repository and a map of commit names to their hashes.
func setupRepoForChangedFilesTest(t *testing.T) (*LocalRepository, map[string]string) {
	t.Helper()
	repo, dir := initTestRepo(t)

	commitHashes := make(map[string]string)

	// Commit 1
	commit1 := createAndCommit(t, repo, "file1.txt", []byte("content1"), "commit 1")
	commitHashes["commit 1"] = commit1.Hash.String()

	// Commit 2 (modify file1.txt)
	commit2 := createAndCommit(t, repo, "file1.txt", []byte("content2"), "commit 2")
	commitHashes["commit 2"] = commit2.Hash.String()

	// Commit 3 (add file2.txt)
	commit3 := createAndCommit(t, repo, "file2.txt", []byte("content3"), "commit 3")
	commitHashes["commit 3"] = commit3.Hash.String()

	return &LocalRepository{Dir: dir, repo: repo}, commitHashes
}

// setupRepoForGetCommitsTest creates a repository with a few commits and tags.
func setupRepoForGetCommitsTest(t *testing.T) (*LocalRepository, map[string]string) {
	t.Helper()
	repo, dir := initTestRepo(t)
	commits := make(map[string]string)

	// Commit 1
	commit1 := createAndCommit(t, repo, "file1.txt", []byte("content1"), "feat: commit 1")
	commits["commit1"] = commit1.Hash.String()

	// Tag for commit 1
	if _, err := repo.CreateTag("v1.0.0", commit1.Hash, nil); err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	// Commit 2
	commit2 := createAndCommit(t, repo, "file2.txt", []byte("content2"), "feat: commit 2")
	commits["commit2"] = commit2.Hash.String()

	// Commit 3
	commit3 := createAndCommit(t, repo, "file3.txt", []byte("content3"), "feat: commit 3")
	commits["commit3"] = commit3.Hash.String()

	return &LocalRepository{Dir: dir, repo: repo}, commits
}

func TestCanUseSSH(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		remoteURI string
		want      bool
	}{
		{
			name:      "remote_https_uri",
			remoteURI: "https://github.com/googleapis/librarian.git",
			want:      false,
		},
		{
			name:      "remote_ssh_uri",
			remoteURI: "git@github.com:googleapis/librarian.git",
			want:      true,
		},
		{
			name:      "remote_ssh_uri_with_scheme",
			remoteURI: "ssh://git@github.com/googleapis/librarian.git",
			want:      true,
		},
		{
			name:      "invalid_remote_uri",
			remoteURI: "nonsense-uri",
			want:      false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := canUseSSH(test.remoteURI)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("canUseSSH() mismatch in %s (-want +got):\n%s", test.name, diff)
			}
		})
	}
}
