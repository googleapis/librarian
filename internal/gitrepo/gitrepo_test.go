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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	gogitConfig "github.com/go-git/go-git/v5/config"
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
			name: "clone maybe",
			opts: &RepositoryOptions{
				Dir:        filepath.Join(tmpDir, "clone-maybe"),
				MaybeClone: true,
				RemoteURL:  remoteDir,
			},
			wantDir: filepath.Join(tmpDir, "clone-maybe"),
		},
		{
			name: "clone maybe no remote url",
			opts: &RepositoryOptions{
				Dir:        filepath.Join(tmpDir, "clone-maybe-no-remote"),
				MaybeClone: true,
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
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
			dir := t.TempDir()
			repo, err := git.PlainInit(dir, false)
			if err != nil {
				t.Fatalf("failed to init repo: %v", err)
			}
			w, err := repo.Worktree()
			if err != nil {
				t.Fatalf("failed to get worktree: %v", err)
			}

			r := &Repository{
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
		name        string
		setup       func(t *testing.T, dir string) string
		expectedNum int
		wantErr     bool
	}{
		{
			name: "add all files",
			setup: func(t *testing.T, dir string) string {
				filePath := filepath.Join(dir, "new_file.txt")
				if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
				return filePath
			},
			expectedNum: 1,
		},
		{
			name: "add all files with error",
			setup: func(t *testing.T, dir string) string {
				// Create a file that cannot be read to simulate an error
				filePath := filepath.Join(dir, "unreadable_file.txt")
				if err := os.WriteFile(filePath, []byte("test content"), 0200); err != nil { // Write-only permissions
					t.Fatalf("failed to write file: %v", err)
				}
				return filePath
			},
			expectedNum: 0,
			wantErr:     true,
		},
		{
			name: "no files to add",
			setup: func(t *testing.T, dir string) string {
				return ""
			},
			expectedNum: 0,
			wantErr:     true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			repo, err := git.PlainInit(dir, false)
			if err != nil {
				t.Fatalf("failed to init repo: %v", err)
			}
			r := &Repository{
				Dir:  dir,
				repo: repo,
			}
			if file := test.setup(t, dir); file != "" {
				_, err = r.AddAll()
				if (err != nil) != test.wantErr {
					t.Errorf("AddAll() returned an error: %v", err)
				}
			}
		})
	}

}

func TestCommit(t *testing.T) {
	t.Parallel()

	// setupRepo is a helper to create a repository with an initial commit.
	setupRepo := func(t *testing.T) *Repository {
		t.Helper()
		dir := t.TempDir()
		gogitRepo, err := git.PlainInit(dir, false)
		if err != nil {
			t.Fatalf("git.PlainInit failed: %v", err)
		}
		w, err := gogitRepo.Worktree()
		if err != nil {
			t.Fatalf("Worktree() failed: %v", err)
		}
		if _, err := w.Commit("initial commit", &git.CommitOptions{
			AllowEmptyCommits: true,
			Author:            &object.Signature{Name: "Test", Email: "test@example.com"},
		}); err != nil {
			t.Fatalf("initial commit failed: %v", err)
		}
		return &Repository{Dir: dir, repo: gogitRepo}
	}

	t.Run("successful commit", func(t *testing.T) {
		t.Parallel()
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

		commitMsg := "feat: add new file"
		if err := repo.Commit(commitMsg, "tester", "tester@example.com"); err != nil {
			t.Fatalf("Commit() unexpected error = %v", err)
		}

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
	})

	t.Run("clean repository", func(t *testing.T) {
		t.Parallel()
		repo := setupRepo(t)

		err := repo.Commit("no-op", "tester", "tester@example.com")
		if err == nil {
			t.Fatal("Commit() expected error, got nil")
		}
		wantErrMsg := "no modifications to commit"
		if !strings.Contains(err.Error(), wantErrMsg) {
			t.Errorf("Commit() error = %q, want to contain %q", err.Error(), wantErrMsg)
		}
	})
}

func TestRemotes(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			gogitRepo, err := git.PlainInit(dir, false)
			if err != nil {
				t.Fatalf("git.PlainInit failed: %v", err)
			}

			for name, urls := range tt.setupRemotes {
				if _, err := gogitRepo.CreateRemote(&gogitConfig.RemoteConfig{
					Name: name,
					URLs: urls,
				}); err != nil {
					t.Fatalf("CreateRemote failed: %v", err)
				}
			}

			repo := &Repository{Dir: dir, repo: gogitRepo}
			got, err := repo.Remotes()
			if (err != nil) != tt.wantErr {
				t.Errorf("Remotes() error = %v, wantErr %v", err, tt.wantErr)
			}

			gotRemotes := make(map[string][]string)
			for _, r := range got {
				gotRemotes[r.Config().Name] = r.Config().URLs
			}
			if diff := cmp.Diff(tt.setupRemotes, gotRemotes); diff != "" {
				t.Errorf("Remotes() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
