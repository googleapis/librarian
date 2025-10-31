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
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
	"gopkg.in/yaml.v3"

	"github.com/google/go-cmp/cmp"
)

func TestRun(t *testing.T) {
	if err := Run(t.Context(), []string{"version"}...); err != nil {
		log.Fatal(err)
	}
}

func TestVerboseFlag(t *testing.T) {
	for _, test := range []struct {
		name              string
		args              []string
		expectDebugLog    bool
		expectDebugSubstr string
	}{
		{
			name:              "generate with -v flag",
			args:              []string{"generate", "-v"},
			expectDebugLog:    true,
			expectDebugSubstr: "generate command verbose logging",
		},
		{
			name:           "generate without -v flag",
			args:           []string{"generate"},
			expectDebugLog: false,
		},
		{
			name:              "release init with -v flag",
			args:              []string{"release", "init", "-v"},
			expectDebugLog:    true,
			expectDebugSubstr: "init command verbose logging",
		},
		{
			name:           "release init without -v flag",
			args:           []string{"release", "init"},
			expectDebugLog: false,
		},
		{
			name:              "release tag with -v flag",
			args:              []string{"release", "tag", "-v"},
			expectDebugLog:    true,
			expectDebugSubstr: "tag command verbose logging",
		},
		{
			name:           "release tag without -v flag",
			args:           []string{"release", "tag"},
			expectDebugLog: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Redirect stderr to capture logs.
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w
			// Reset logger to default for test isolation.
			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

			_ = Run(context.Background(), test.args...)

			// Restore stderr and read the output.
			w.Close()
			os.Stderr = oldStderr
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			hasDebugLog := strings.Contains(output, "level=DEBUG")
			if test.expectDebugLog {
				if !hasDebugLog {
					t.Errorf("expected debug log to be present, but it wasn't. Output:\n%s", output)
				}
				if !strings.Contains(output, test.expectDebugSubstr) {
					t.Errorf("expected debug log to contain %q, but it didn't. Output:\n%s", test.expectDebugSubstr, output)
				}
			} else {
				if hasDebugLog {
					t.Errorf("expected debug log to be absent, but it was present. Output:\n%s", output)
				}
			}
		})
	}
}

func TestGenerate_DefaultBehavior(t *testing.T) {
	ctx := context.Background()

	// 1. Set up a mock repository with a state file
	repo := newTestGitRepo(t)
	repoDir := repo.GetDir()

	// Set up a dummy API Source repo to prevent cloning googleapis/googleapis
	apiSourceDir := t.TempDir()
	runGit(t, apiSourceDir, "init")
	runGit(t, apiSourceDir, "config", "user.email", "test@example.com")
	runGit(t, apiSourceDir, "config", "user.name", "Test User")
	runGit(t, apiSourceDir, "commit", "--allow-empty", "-m", "initial commit")

	t.Chdir(repoDir)

	// 2. Override dependency creation to use mocks
	mockContainer := &mockContainerClient{
		wantLibraryGen: true,
	}
	mockGH := &mockGitHubClient{}

	// 3. Call librarian.Run
	cfg := config.New("generate")
	cfg.WorkRoot = repoDir
	cfg.Repo = repoDir
	cfg.APISource = apiSourceDir
	runner, err := newGenerateRunner(cfg)
	if err != nil {
		t.Fatalf("newGenerateRunner() failed: %v", err)
	}

	runner.ghClient = mockGH
	runner.containerClient = mockContainer
	if err := runner.run(ctx); err != nil {
		t.Fatalf("runner.run() failed: %v", err)
	}

	// 4. Assertions
	expectedGenerateCalls := 1
	if mockContainer.generateCalls != expectedGenerateCalls {
		t.Errorf("Run(ctx, \"generate\"): got %d generate calls, want %d", mockContainer.generateCalls, expectedGenerateCalls)
	}
}

func TestIsURL(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "Valid HTTPS URL",
			input: "https://github.com/googleapis/librarian-go",
			want:  true,
		},
		{
			name:  "Valid HTTP URL",
			input: "http://example.com/path?query=value",
			want:  true,
		},
		{
			name:  "Valid FTP URL",
			input: "ftp://user:password@host/path",
			want:  true,
		},
		{
			name:  "URL without scheme",
			input: "google.com",
			want:  false,
		},
		{
			name:  "URL with scheme only",
			input: "https://",
			want:  false,
		},
		{
			name:  "Absolute Unix file path",
			input: "/home/user/file",
			want:  false,
		},
		{
			name:  "Relative file path",
			input: "home/user/file",
			want:  false,
		},
		{
			name:  "Empty string",
			input: "",
			want:  false,
		},
		{
			name:  "Plain string",
			input: "just-a-string",
			want:  false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isURL(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("isURL() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// newTestGitRepo creates a new git repository in a temporary directory.
func newTestGitRepo(t *testing.T) gitrepo.Repository {
	t.Helper()
	defaultState := &config.LibrarianState{
		Image: "some/image:v1.2.3",
		Libraries: []*config.LibraryState{
			{
				ID: "some-library",
				APIs: []*config.API{
					{
						Path:          "some/api",
						ServiceConfig: "api_config.yaml",
						Status:        config.StatusExisting,
					},
				},
				SourceRoots: []string{"src/a"},
			},
		},
	}
	return newTestGitRepoWithState(t, defaultState)
}

// newTestGitRepo creates a new git repository in a temporary directory.
func newTestGitRepoWithState(t *testing.T, state *config.LibrarianState) gitrepo.Repository {
	t.Helper()
	dir := t.TempDir()
	remoteURL := "https://github.com/googleapis/librarian.git"
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	// Create a state.yaml and config.yaml file in .librarian dir.
	librarianDir := filepath.Join(dir, config.LibrarianDir)
	if err := os.MkdirAll(librarianDir, 0755); err != nil {
		t.Fatalf("os.MkdirAll: %v", err)
	}

	// Setup each source root directory to be non-empty (one `random_file.txt`)
	// that can be used to test preserve or remove regex patterns
	for _, library := range state.Libraries {
		for _, sourceRoot := range library.SourceRoots {
			fullPath := filepath.Join(dir, sourceRoot, "random_file.txt")
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Create(fullPath); err != nil {
				t.Fatal(err)
			}
		}
	}

	bytes, err := yaml.Marshal(state)
	if err != nil {
		t.Fatalf("yaml.Marshal() = %v", err)
	}
	stateFile := filepath.Join(librarianDir, "state.yaml")
	if err := os.WriteFile(stateFile, bytes, 0644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	configFile := filepath.Join(librarianDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte{}, 0644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial commit")
	runGit(t, dir, "remote", "add", "origin", remoteURL)
	repo, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{Dir: dir})
	if err != nil {
		t.Fatalf("gitrepo.Open(%q) = %v", dir, err)
	}
	return repo
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}

// setupRepoForGetCommits creates an empty gitrepo and creates some commits and
// tags.
//
// Each commit has a file path and a commit message.
// Note that pathAndMessages should at least have one element. All tags are created
// after the first commit.
func setupRepoForGetCommits(t *testing.T, pathAndMessages []pathAndMessage, tags []string) *gitrepo.LocalRepository {
	t.Helper()
	dir := t.TempDir()
	gitRepo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("git.PlainInit failed: %v", err)
	}

	createAndCommit := func(path, msg string) {
		w, err := gitRepo.Worktree()
		if err != nil {
			t.Fatalf("Worktree() failed: %v", err)
		}
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("os.MkdirAll failed: %v", err)
		}
		content := fmt.Sprintf("content-%d", rand.Intn(10000))
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("os.WriteFile failed: %v", err)
		}
		if _, err := w.Add(path); err != nil {
			t.Fatalf("w.Add failed: %v", err)
		}
		_, err = w.Commit(msg, &git.CommitOptions{
			Author: &object.Signature{Name: "Test", Email: "test@example.com"},
		})
		if err != nil {
			t.Fatalf("w.Commit failed: %v", err)
		}
	}

	createAndCommit(pathAndMessages[0].path, pathAndMessages[0].message)
	head, err := gitRepo.Head()
	if err != nil {
		t.Fatalf("repo.Head() failed: %v", err)
	}
	for _, tag := range tags {
		if _, err := gitRepo.CreateTag(tag, head.Hash(), nil); err != nil {
			t.Fatalf("CreateTag failed: %v", err)
		}
	}

	for _, pam := range pathAndMessages[1:] {
		createAndCommit(pam.path, pam.message)
	}

	r, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{Dir: dir})
	if err != nil {
		t.Fatalf("gitrepo.NewRepository failed: %v", err)
	}
	return r
}

type pathAndMessage struct {
	path    string
	message string
}
