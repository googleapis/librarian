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

	"github.com/googleapis/librarian/internal/config"
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

func TestDeriveImage(t *testing.T) {
	for _, test := range []struct {
		name          string
		imageOverride string
		state         *config.PipelineState
		want          string
		wantErr       bool
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
			state:         &config.PipelineState{ImageTag: "v1.2.3"},
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
			state:         &config.PipelineState{ImageTag: "v1.2.3"},
			want:          "v1.2.3",
		},
		{
			name:          "no override, with state, empty tag",
			imageOverride: "",
			state:         &config.PipelineState{ImageTag: ""},
			wantErr:       true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := deriveImage(test.imageOverride, test.state)

			if test.wantErr {
				if err == nil {
					t.Error("deriveImage() expected an error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("deriveImage() got unexpected error: %v", err)
				return
			}
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
		check   func(t *testing.T, repo *gitrepo.Repository)
		setup   func(t *testing.T, workRoot string) func()
	}{
		{
			name: "with clean repoRoot",
			repo: cleanRepoPath,
			check: func(t *testing.T, repo *gitrepo.Repository) {
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
			check: func(t *testing.T, repo *gitrepo.Repository) {
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

			repo, err := cloneOrOpenLanguageRepo(workRoot, test.repo, test.ci)
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

func TestCommitAll(t *testing.T) {
	for _, test := range []struct {
		name       string
		setup      func(t *testing.T, repoDir string)
		wantCommit bool
	}{
		{
			name: "clean repo, no commit",
			setup: func(t *testing.T, repoDir string) {
			},
			wantCommit: false,
		},
		{
			name: "dirty repo, with commit",
			setup: func(t *testing.T, repoDir string) {
				filePath := filepath.Join(repoDir, "new-file.txt")
				if err := os.WriteFile(filePath, []byte("some content"), 0644); err != nil {
					t.Fatalf("WriteFile: %v", err)
				}
			},
			wantCommit: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoDir := newTestGitRepoWithCommit(t, "")
			repo, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{Dir: repoDir})
			if err != nil {
				t.Fatalf("NewRepository() error = %v", err)
			}

			initialHead, err := repo.HeadHash()
			if err != nil {
				t.Fatalf("repo.HeadHash() error = %v", err)
			}

			test.setup(t, repoDir)

			if err := commitAll(repo, "test commit", "tester@example.com,tester"); err != nil {
				t.Errorf("commitAll() error = %v, wantErr nil", err)
			}

			finalHead, err := repo.HeadHash()
			if err != nil {
				t.Fatalf("repo.HeadHash() error = %v", err)
			}

			hasCommitted := initialHead != finalHead
			if hasCommitted != test.wantCommit {
				t.Errorf("commitAll() commit status = %v, want %v", hasCommitted, test.wantCommit)
			}
		})
	}
}

func TestParsePushConfig(t *testing.T) {
	for _, test := range []struct {
		name       string
		pushConfig string
		wantEmail  string
		wantName   string
		wantErr    bool
	}{
		{
			name:       "valid input",
			pushConfig: "tester@example.com,tester",
			wantEmail:  "tester@example.com",
			wantName:   "tester",
		},
		{
			name:       "invalid input, too few parts",
			pushConfig: "tester@example.com",
			wantErr:    true,
		},
		{
			name:       "invalid input, too many parts",
			pushConfig: "tester@example.com,tester,extra",
			wantErr:    true,
		},
		{
			name:       "empty input",
			pushConfig: "",
			wantErr:    true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotEmail, gotName, err := parsePushConfig(test.pushConfig)

			if test.wantErr {
				if err == nil {
					t.Error("parsePushConfig() expected an error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("parsePushConfig() got unexpected error: %v", err)
				return
			}
			if gotEmail != test.wantEmail {
				t.Errorf("parsePushConfig() email = %q, want %q", gotEmail, test.wantEmail)
			}
			if gotName != test.wantName {
				t.Errorf("parsePushConfig() name = %q, want %q", gotName, test.wantName)
			}
		})
	}
}
