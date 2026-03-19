// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package librarianops

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func testRunCommand(wantResults map[string]error, defaultErr error) (func(context.Context, string, ...string) error, *[][]string) {
	var mu sync.Mutex
	var gotCommands [][]string
	run := func(ctx context.Context, name string, args ...string) error {
		mu.Lock()
		defer mu.Unlock()
		cmd := append([]string{name}, args...)
		gotCommands = append(gotCommands, cmd)
		key := formatCmd(name, args...)
		if err, ok := wantResults[key]; ok {
			return err
		}
		return defaultErr
	}
	return run, &gotCommands
}

func formatCmd(name string, args ...string) string {
	return fmt.Sprintf("%s %s", name, strings.Join(args, " "))
}

func TestUploadToGithub(t *testing.T) {

	for _, test := range []struct {
		name     string
		details  GithubDetails
		wantCmds [][]string
	}{
		{
			name: "standard success",
			details: GithubDetails{
				PrTitle:    "feat: support new api",
				PrBody:     "This PR adds support for the new API.",
				BranchName: "autoupgrade-2026-01-02",
			},
			wantCmds: [][]string{
				{"git", "checkout", "-b", "autoupgrade-2026-01-02"},
				{"git", "add", "."},
				{"git", "commit", "-m", "feat: support new api"},
				{"git", "push", "-u", "origin", "HEAD"},
				{"gh", "pr", "create", "--title", "feat: support new api", "--body", "This PR adds support for the new API."},
			},
		},
		{
			name: "new feature",
			details: GithubDetails{
				PrTitle:    "feat: new feature",
				PrBody:     "pr body",
				BranchName: "autoupgrade-2026-01-02",
			},
			wantCmds: [][]string{
				{"git", "checkout", "-b", "autoupgrade-2026-01-02"},
				{"git", "add", "."},
				{"git", "commit", "-m", "feat: new feature"},
				{"git", "push", "-u", "origin", "HEAD"},
				{"gh", "pr", "create", "--title", "feat: new feature", "--body", "pr body"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			wantResults := map[string]error{
				formatCmd("git", "checkout", "-b", test.details.BranchName):                                     nil,
				formatCmd("git", "add", "."):                                                                    nil,
				formatCmd("git", "commit", "-m", test.details.PrTitle):                                          nil,
				formatCmd("git", "push", "-u", "origin", "HEAD"):                                                nil,
				formatCmd("gh", "pr", "create", "--title", test.details.PrTitle, "--body", test.details.PrBody): nil,
			}
			testRun, gotCommands := testRunCommand(wantResults, nil)
			runCommand = testRun

			if err := uploadToGithub(context.Background(), test.details); err != nil {
				t.Fatalf("got unexpected error: %v", err)
			}

			if diff := cmp.Diff(test.wantCmds, *gotCommands); diff != "" {
				t.Errorf("commands mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUploadToGithub_Error(t *testing.T) {
	const (
		branch = "fix-branch"
		title  = "fix: bug"
	)

	for _, test := range []struct {
		name        string
		wantResults map[string]error
	}{
		{
			name: "branch creation fails",
			wantResults: map[string]error{
				formatCmd("git", "checkout", "-b", branch): fmt.Errorf("branch error"),
			},
		},
		{
			name: "commit fails",
			wantResults: map[string]error{
				formatCmd("git", "checkout", "-b", branch): nil,
				formatCmd("git", "add", "."):               fmt.Errorf("git add failed"),
			},
		},
		{
			name: "push fails",
			wantResults: map[string]error{
				formatCmd("git", "checkout", "-b", branch):       nil,
				formatCmd("git", "add", "."):                     nil,
				formatCmd("git", "commit", "-m", title):          nil,
				formatCmd("git", "push", "-u", "origin", "HEAD"): fmt.Errorf("network error"),
			},
		},
		{
			name: "pr creation fails",
			wantResults: map[string]error{
				formatCmd("git", "checkout", "-b", branch):                      nil,
				formatCmd("git", "add", "."):                                    nil,
				formatCmd("git", "commit", "-m", title):                         nil,
				formatCmd("git", "push", "-u", "origin", "HEAD"):                nil,
				formatCmd("gh", "pr", "create", "--title", title, "--body", ""): fmt.Errorf("api error"),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testRun, _ := testRunCommand(test.wantResults, nil)
			runCommand = testRun

			err := uploadToGithub(context.Background(), GithubDetails{PrTitle: title, BranchName: branch})

			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestCloneRepoInDir(t *testing.T) {
	const (
		repoName = "librarian"
		repoDir  = "/tmp/repo"
	)
	testRun, gotCommands := testRunCommand(nil, nil)
	runCommand = testRun
	if err := cloneRepoInDir(context.Background(), repoName, repoDir); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	want := [][]string{{"gh", "repo", "clone", "googleapis/librarian", "/tmp/repo"}}
	if diff := cmp.Diff(want, *gotCommands); diff != "" {
		t.Errorf("command mismatch (-want +got):\n%s", diff)
	}
}

func TestCloneRepoInDir_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		wantErr error
	}{
		{name: "auth error", wantErr: fmt.Errorf("not logged in")},
		{name: "not found", wantErr: fmt.Errorf("repository not found")},
	} {
		t.Run(test.name, func(t *testing.T) {
			testRun, _ := testRunCommand(nil, test.wantErr)
			runCommand = testRun

			if err := cloneRepoInDir(context.Background(), "repo", "dir"); err == nil {
				t.Fatalf("got nil, want error %v", test.wantErr)
			}
		})
	}
}

func TestCreateBranch(t *testing.T) {
	for _, test := range []struct {
		name   string
		branch string
	}{
		{name: "standard branch", branch: "feat-1"},
		{name: "branch with slash", branch: "fix/bug-123"},
	} {
		t.Run(test.name, func(t *testing.T) {
			testRun, gotCommands := testRunCommand(nil, nil)
			runCommand = testRun

			if err := createBranch(context.Background(), test.branch); err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			want := [][]string{{"git", "checkout", "-b", test.branch}}
			if diff := cmp.Diff(want, *gotCommands); diff != "" {
				t.Errorf("command mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateBranch_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		wantErr error
	}{
		{name: "generic git error", wantErr: fmt.Errorf("git error")},
		{name: "branch already exists", wantErr: fmt.Errorf("already exists")},
	} {
		t.Run(test.name, func(t *testing.T) {
			testRun, _ := testRunCommand(nil, test.wantErr)
			runCommand = testRun

			if err := createBranch(context.Background(), "feat-branch"); err == nil {
				t.Fatalf("got nil, want error %v", test.wantErr)
			}
		})
	}
}

func TestCommitChanges(t *testing.T) {
	for _, test := range []struct {
		name  string
		title string
	}{
		{name: "feat commit", title: "feat: something"},
		{name: "fix commit", title: "fix: something"},
	} {
		t.Run(test.name, func(t *testing.T) {
			testRun, gotCommands := testRunCommand(nil, nil)
			runCommand = testRun

			if err := commitChanges(context.Background(), test.title); err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			want := [][]string{
				{"git", "add", "."},
				{"git", "commit", "-m", test.title},
			}
			if diff := cmp.Diff(want, *gotCommands); diff != "" {
				t.Errorf("commands mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCommitChanges_Error(t *testing.T) {
	for _, test := range []struct {
		name        string
		wantResults map[string]error
		wantErr     error
	}{
		{
			name: "add fails",
			wantResults: map[string]error{
				formatCmd("git", "add", "."): fmt.Errorf("add fail"),
			},
			wantErr: fmt.Errorf("add fail"),
		},
		{
			name: "commit fails",
			wantResults: map[string]error{
				formatCmd("git", "add", "."):            nil,
				formatCmd("git", "commit", "-m", "msg"): fmt.Errorf("commit fail"),
			},
			wantErr: fmt.Errorf("commit fail"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testRun, _ := testRunCommand(test.wantResults, nil)
			runCommand = testRun
			err := commitChanges(context.Background(), "msg")
			if err == nil || !strings.Contains(err.Error(), test.wantErr.Error()) {
				t.Errorf("got error %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestPushChanges(t *testing.T) {
	testRun, gotCommands := testRunCommand(nil, nil)
	runCommand = testRun

	if err := pushChanges(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	want := [][]string{{"git", "push", "-u", "origin", "HEAD"}}
	if diff := cmp.Diff(want, *gotCommands); diff != "" {
		t.Errorf("command mismatch (-want +got):\n%s", diff)
	}
}

func TestPushChanges_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		wantErr error
	}{
		{name: "remote rejected", wantErr: fmt.Errorf("permission denied")},
		{name: "network failure", wantErr: fmt.Errorf("could not resolve host")},
	} {
		t.Run(test.name, func(t *testing.T) {
			testRun, _ := testRunCommand(nil, test.wantErr)
			runCommand = testRun

			if err := pushChanges(context.Background()); err == nil {
				t.Fatalf("got nil, want error %v", test.wantErr)
			}
		})
	}
}

func TestCreatePR(t *testing.T) {
	testRun, gotCommands := testRunCommand(nil, nil)
	runCommand = testRun
	details := GithubDetails{PrTitle: "T", PrBody: "B"}
	if err := createPR(context.Background(), details); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	want := [][]string{{"gh", "pr", "create", "--title", details.PrTitle, "--body", details.PrBody}}
	if diff := cmp.Diff(want, *gotCommands); diff != "" {
		t.Errorf("command mismatch (-want +got):\n%s", diff)
	}
}

func TestCreatePR_Error(t *testing.T) {
	wantErr := fmt.Errorf("could not create pr")
	testRun, _ := testRunCommand(nil, wantErr)
	runCommand = testRun
	if err := createPR(context.Background(), GithubDetails{}); err == nil {
		t.Fatalf("got nil, want error %v", wantErr)
	}
}

func TestGenerateBranchName(t *testing.T) {
	got := generateBranchName("pref-", time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC))
	want := "pref-20260102T000000Z"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
