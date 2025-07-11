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
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
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
