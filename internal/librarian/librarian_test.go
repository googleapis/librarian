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
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
	"gopkg.in/yaml.v3"

	"github.com/google/go-cmp/cmp"
)

// TODO(https://github.com/googleapis/librarian/issues/202): add better tests
// for librarian.Run.
func TestRun(t *testing.T) {
	if err := Run(t.Context(), []string{"version"}...); err != nil {
		log.Fatal(err)
	}
}

func TestParentCommands(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		command    string
		wantErr    bool
		wantErrMsg string // Expected substring in the error
	}{
		{
			name:       "release no subcommand",
			command:    "release",
			wantErr:    true,
			wantErrMsg: `command "release" requires a subcommand`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := Run(ctx, test.command)

			if test.wantErr {
				if err == nil {
					t.Fatalf("Run(ctx, %q) got nil, want error containing %q", test.command, test.wantErrMsg)
				}
				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("Run(ctx, %q) got error %q, want error containing %q", test.command, err.Error(), test.wantErrMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("Run(ctx, %q) got error %v, want nil", test.command, err)
			}
		})
	}
}

func TestGenerate_DefaultBehavior(t *testing.T) {
	ctx := context.Background()

	// 1. Setup a mock repository with a state file
	repo := newTestGitRepoWithState(t, true)
	repoDir := repo.GetDir()

	// Setup a dummy API Source repo to prevent cloning googleapis/googleapis
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
	runner, err := newGenerateRunner(cfg, func(token string, repo *github.Repository) (GitHubClient, error) {
		return mockGH, nil
	}, func(workRoot, image, userUID, userGID string) (ContainerClient, error) {
		return mockContainer, nil
	})
	if err != nil {
		t.Fatalf("newGenerateRunner() failed: %v", err)
	}
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
	return newTestGitRepoWithState(t, true)
}

// newTestGitRepo creates a new git repository in a temporary directory.
func newTestGitRepoWithState(t *testing.T, writeState bool) gitrepo.Repository {
	t.Helper()
	dir := t.TempDir()
	remoteURL := "https://github.com/googleapis/librarian.git"
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	if writeState {
		// Create a state.yaml and config.yaml file in .librarian dir.
		librarianDir := filepath.Join(dir, config.LibrarianDir)
		if err := os.MkdirAll(librarianDir, 0755); err != nil {
			t.Fatalf("os.MkdirAll: %v", err)
		}

		state := &config.LibrarianState{
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

func TestLookupCommand(t *testing.T) {
	sub1sub1 := &cli.Command{
		Short:     "sub1sub1 does something",
		UsageLine: "sub1sub1",
		Long:      "sub1sub1 does something",
	}
	sub1 := &cli.Command{
		Short:     "sub1 does something",
		UsageLine: "sub1",
		Long:      "sub1 does something",
		Commands:  []*cli.Command{sub1sub1},
	}
	sub2 := &cli.Command{
		Short:     "sub2 does something",
		UsageLine: "sub2",
		Long:      "sub2 does something",
	}
	root := &cli.Command{
		Short:     "root does something",
		UsageLine: "root",
		Long:      "root does something",
		Commands: []*cli.Command{
			sub1,
			sub2,
		},
	}
	root.Init()
	sub1.Init()
	sub2.Init()
	sub1sub1.Init()

	testCases := []struct {
		name     string
		cmd      *cli.Command
		args     []string
		wantCmd  *cli.Command
		wantArgs []string
		wantErr  bool
	}{
		{
			name:    "no args",
			cmd:     root,
			args:    []string{},
			wantCmd: root,
		},
		{
			name:    "find sub1",
			cmd:     root,
			args:    []string{"sub1"},
			wantCmd: sub1,
		},
		{
			name:     "find sub2",
			cmd:      root,
			args:     []string{"sub2"},
			wantCmd:  sub2,
			wantArgs: []string{},
		},
		{
			name:     "find sub1sub1",
			cmd:      root,
			args:     []string{"sub1", "sub1sub1"},
			wantCmd:  sub1sub1,
			wantArgs: []string{},
		},
		{
			name:     "find sub1sub1 with args",
			cmd:      root,
			args:     []string{"sub1", "sub1sub1", "arg1"},
			wantCmd:  sub1sub1,
			wantArgs: []string{"arg1"},
		},
		{
			name:    "unknown command",
			cmd:     root,
			args:    []string{"unknown"},
			wantErr: true,
		},
		{
			name:    "unknown subcommand",
			cmd:     root,
			args:    []string{"sub1", "unknown"},
			wantErr: true,
		},
		{
			name:     "find sub1 with flag arguments",
			cmd:      root,
			args:     []string{"sub1", "-h"},
			wantCmd:  sub1,
			wantArgs: []string{"-h"},
		},
		{
			name:     "find sub1sub1 with flag arguments",
			cmd:      root,
			args:     []string{"sub1", "sub1sub1", "-h"},
			wantCmd:  sub1sub1,
			wantArgs: []string{"-h"},
		},
		{
			name:     "find sub1 with a flag argument in between subcommands",
			cmd:      root,
			args:     []string{"sub1", "-h", "sub1sub1"},
			wantCmd:  sub1,
			wantArgs: []string{"-h", "sub1sub1"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotCmd, gotArgs, err := lookupCommand(tc.cmd, tc.args)
			if (err != nil) != tc.wantErr {
				t.Errorf("lookupCommand() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if gotCmd != tc.wantCmd {
				var gotName, wantName string
				if gotCmd != nil {
					gotName = gotCmd.Name()
				}
				if tc.wantCmd != nil {
					wantName = tc.wantCmd.Name()
				}
				t.Errorf("lookupCommand() gotCmd.Name() = %q, want %q", gotName, wantName)
			}
			if diff := cmp.Diff(tc.wantArgs, gotArgs); diff != "" {
				t.Errorf("lookupCommand() args mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
