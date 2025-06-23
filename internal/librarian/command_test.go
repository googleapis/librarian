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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
)

func TestCommandUsage(t *testing.T) {
	for _, c := range librarianCommands {
		t.Run(c.Name(), func(t *testing.T) {
			parts := strings.Fields(c.Usage)
			// The first word should always be "librarian".
			if parts[0] != "librarian" {
				t.Errorf("invalid usage text: %q (the first word should be `librarian`)", c.Usage)
			}
			// The second word should always be the command name.
			if parts[1] != c.Name() {
				t.Errorf("invalid usage text: %q (second word should be command name %q)", c.Usage, c.Name())
			}
		})
	}
}

func TestDeriveImage(t *testing.T) {
	tests := []struct {
		name              string
		language          string
		imageOverride     string
		defaultRepository string
		state             *statepb.PipelineState
		want              string
	}{
		{
			name:          "with image override",
			language:      "go",
			imageOverride: "my/custom-image:v1",
			want:          "my/custom-image:v1",
		},
		{
			name:     "no override, no repo, no state",
			language: "go",
			want:     "google-cloud-go-generator:latest",
		},
		{
			name:     "no override, no repo, with state",
			language: "go",
			state:    &statepb.PipelineState{ImageTag: "v1.2.3"},
			want:     "google-cloud-go-generator:v1.2.3",
		},
		{
			name:              "no override, with repo, no state",
			language:          "go",
			defaultRepository: "path/to/repo",
			want:              "path/to/repo/google-cloud-go-generator:latest",
		},
		{
			name:              "no override, with repo, with state",
			language:          "go",
			defaultRepository: "path/to/repo",
			state:             &statepb.PipelineState{ImageTag: "v1.2.3"},
			want:              "path/to/repo/google-cloud-go-generator:v1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveImage(tt.language, tt.imageOverride, tt.defaultRepository, tt.state)
			if got != tt.want {
				t.Errorf("deriveImage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreateWorkRoot(t *testing.T) {
	now := time.Now()
	tests := []struct {
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
				return expectedPath, func() { os.RemoveAll(expectedPath) }
			},
		},
		{
			name: "without override, dir exists",
			setup: func(t *testing.T) (string, func()) {
				expectedPath := filepath.Join(os.TempDir(), fmt.Sprintf("librarian-%s", formatTimestamp(now)))
				if err := os.Mkdir(expectedPath, 0755); err != nil {
					t.Fatalf("failed to create test dir: %v", err)
				}
				return expectedPath, func() { os.RemoveAll(expectedPath) }
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, cleanup := tt.setup(t)
			defer cleanup()

			got, err := createWorkRoot(now, tt.workRootOverride)
			if err != nil && !tt.wantErr {
				t.Errorf("createWorkRoot() got unexpected error: %v", err)
				return
			}
			if err == nil && tt.wantErr {
				t.Errorf("createWorkRoot() expected an error but got nil")
				return
			}
			if !tt.wantErr {
				if got != want {
					t.Errorf("createWorkRoot() = %v, want %v", got, want)
				}
				if tt.workRootOverride == "" {
					if _, err := os.Stat(got); os.IsNotExist(err) {
						t.Errorf("createWorkRoot() did not create directory %v", got)
					}
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

	tests := []struct {
		name     string
		repoRoot string
		repoURL  string
		language string
		wantErr  bool
		check    func(t *testing.T, repo *gitrepo.Repository)
		setup    func(t *testing.T, workRoot string) func()
	}{
		{
			name:     "repoRoot and repoURL both set",
			repoRoot: "a",
			repoURL:  "b",
			wantErr:  true,
		},
		{
			name:     "with clean repoRoot",
			repoRoot: cleanRepoPath,
			check: func(t *testing.T, repo *gitrepo.Repository) {
				absWantDir, _ := filepath.Abs(cleanRepoPath)
				if repo.Dir != absWantDir {
					t.Errorf("repo.Dir got %q, want %q", repo.Dir, absWantDir)
				}
			},
		},
		{
			name:     "no repoRoot or repoURL, default to open language monorepo",
			language: "go",
			// Setup for this specific test case: create the expected default repo.
			// This avoids actual network cloning.
			setup: func(t *testing.T, wr string) func() {
				repoPath := filepath.Join(wr, "google-cloud-go")
				newTestGitRepoWithCommit(t, repoPath)
				return func() { os.RemoveAll(repoPath) }
			},
			check: func(t *testing.T, repo *gitrepo.Repository) {
				wantDir := filepath.Join(workRoot, "google-cloud-go")
				if repo.Dir != wantDir {
					t.Errorf("repo.Dir got %q, want %q", repo.Dir, wantDir)
				}
			},
		},
		{
			name:     "with dirty repoRoot",
			repoRoot: dirtyRepoPath,
			wantErr:  true,
		},
		{
			name:     "with repoRoot that is not a repo",
			repoRoot: notARepoPath,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cleanup func()
			if tt.setup != nil {
				cleanup = tt.setup(t, workRoot)
			}
			defer func() {
				if cleanup != nil {
					cleanup()
				}
			}()
			repo, err := cloneOrOpenLanguageRepo(workRoot, tt.repoRoot, tt.repoURL, tt.language)
			if err != nil && !tt.wantErr {
				t.Errorf("cloneOrOpenLanguageRepo() got unexpected error: %v", err)
				return
			}
			if err == nil && tt.wantErr {
				t.Errorf("cloneOrOpenLanguageRepo() expected an error but got nil")
				return
			}
			if tt.check != nil {
				if repo == nil {
					t.Fatal("repo is nil")
				}
				tt.check(t, repo)
			}
		})
	}
}
