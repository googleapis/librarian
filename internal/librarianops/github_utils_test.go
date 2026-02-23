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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
)

func TestUploadToGithub(t *testing.T) {
	t.Parallel()
	const (
		wantTitle  = "feat: support new api"
		wantBody   = "This PR adds support for the new API."
		wantBranch = "autoupgrade-2026-01-02"
	)
	for _, tc := range []struct {
		name     string
		details  GithubDetails
		wantCmds [][]string
	}{
		{
			name: "standard success",
			details: GithubDetails{
				PrTitle:    wantTitle,
				PrBody:     wantBody,
				BranchName: wantBranch,
			},
			wantCmds: [][]string{
				{"git", "checkout", "-b", wantBranch},
				{"git", "add", "."},
				{"git", "commit", "-m", wantTitle},
				{"git", "push", "-u", "origin", "HEAD"},
				{"gh", "pr", "create", "--title", wantTitle, "--body", wantBody},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mocker := &command.MockCommander{
				MockResults: map[string]command.MockResult{
					command.FormatCmd("git", "checkout", "-b", tc.details.BranchName):                                   {},
					command.FormatCmd("git", "add", "."):                                                                {},
					command.FormatCmd("git", "commit", "-m", tc.details.PrTitle):                                        {},
					command.FormatCmd("git", "push", "-u", "origin", "HEAD"):                                            {},
					command.FormatCmd("gh", "pr", "create", "--title", tc.details.PrTitle, "--body", tc.details.PrBody): {},
				},
			}
			ctx := mocker.InjectContext(t.Context())

			if err := uploadToGithub(ctx, tc.details); err != nil {
				t.Fatalf("got unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.wantCmds, mocker.GotCommands); diff != "" {
				t.Errorf("commands mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUploadToGithub_Error(t *testing.T) {
	t.Parallel()

	const (
		branch = "fix-branch"
		title  = "fix: bug"
	)

	for _, tc := range []struct {
		name         string
		mockResults  map[string]command.MockResult
		wantErr      string
		maxCmdsCount int
	}{
		{
			name: "branch creation fails",
			mockResults: map[string]command.MockResult{
				command.FormatCmd("git", "checkout", "-b", branch): {Error: fmt.Errorf("branch error")},
			},
			wantErr:      "branch error",
			maxCmdsCount: 1,
		},
		{
			name: "commit fails",
			mockResults: map[string]command.MockResult{
				command.FormatCmd("git", "checkout", "-b", branch): {},
				command.FormatCmd("git", "add", "."):               {Error: fmt.Errorf("git add failed")},
			},
			wantErr:      "git add failed",
			maxCmdsCount: 2,
		},
		{
			name: "push fails",
			mockResults: map[string]command.MockResult{
				command.FormatCmd("git", "checkout", "-b", branch):       {},
				command.FormatCmd("git", "add", "."):                     {},
				command.FormatCmd("git", "commit", "-m", title):          {},
				command.FormatCmd("git", "push", "-u", "origin", "HEAD"): {Error: fmt.Errorf("network error")},
			},
			wantErr:      "network error",
			maxCmdsCount: 4,
		},
		{
			name: "pr creation fails",
			mockResults: map[string]command.MockResult{
				command.FormatCmd("git", "checkout", "-b", branch):                      {},
				command.FormatCmd("git", "add", "."):                                    {},
				command.FormatCmd("git", "commit", "-m", title):                         {},
				command.FormatCmd("git", "push", "-u", "origin", "HEAD"):                {},
				command.FormatCmd("gh", "pr", "create", "--title", title, "--body", ""): {Error: fmt.Errorf("api error")},
			},
			wantErr:      "api error",
			maxCmdsCount: 5,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mocker := &command.MockCommander{MockResults: tc.mockResults}
			ctx := mocker.InjectContext(t.Context())

			err := uploadToGithub(ctx, GithubDetails{PrTitle: title, BranchName: branch})

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not contain expected string %q", err, tc.wantErr)
			}
			if len(mocker.GotCommands) > tc.maxCmdsCount {
				t.Errorf("executed %d commands, want at most %d", len(mocker.GotCommands), tc.maxCmdsCount)
			}
		})
	}
}

func TestCloneRepoInDir(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		repoName string
		repoDir  string
	}{
		{
			name:     "standard repo",
			repoName: "librarian",
			repoDir:  "/tmp/repo",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mocker := &command.MockCommander{}
			ctx := mocker.InjectContext(t.Context())

			if err := cloneRepoInDir(ctx, tc.repoName, tc.repoDir); err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			want := [][]string{{"gh", "repo", "clone", "googleapis/librarian", "/tmp/repo"}}
			if diff := cmp.Diff(want, mocker.GotCommands); diff != "" {
				t.Errorf("command mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCloneRepoInDir_Error(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		mockErr error
	}{
		{name: "auth error", mockErr: fmt.Errorf("not logged in")},
		{name: "not found", mockErr: fmt.Errorf("repository not found")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mocker := &command.MockCommander{Default: command.MockResult{Error: tc.mockErr}}
			ctx := mocker.InjectContext(t.Context())

			if err := cloneRepoInDir(ctx, "repo", "dir"); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestCreateBranch(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		branch string
	}{
		{name: "standard branch", branch: "feat-1"},
		{name: "branch with slash", branch: "fix/bug-123"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mocker := &command.MockCommander{}
			ctx := mocker.InjectContext(t.Context())

			if err := createBranch(ctx, tc.branch); err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			want := [][]string{{"git", "checkout", "-b", tc.branch}}
			if diff := cmp.Diff(want, mocker.GotCommands); diff != "" {
				t.Errorf("command mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateBranch_Error(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		mockErr error
	}{
		{name: "generic git error", mockErr: fmt.Errorf("git error")},
		{name: "branch already exists", mockErr: fmt.Errorf("already exists")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mocker := &command.MockCommander{Default: command.MockResult{Error: tc.mockErr}}
			ctx := mocker.InjectContext(t.Context())

			if err := createBranch(ctx, "feat-branch"); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestCommitChanges(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		title string
	}{
		{name: "feat commit", title: "feat: something"},
		{name: "fix commit", title: "fix: something"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mocker := &command.MockCommander{}
			ctx := mocker.InjectContext(t.Context())

			if err := commitChanges(ctx, tc.title); err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			want := [][]string{
				{"git", "add", "."},
				{"git", "commit", "-m", tc.title},
			}
			if diff := cmp.Diff(want, mocker.GotCommands); diff != "" {
				t.Errorf("commands mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCommitChanges_Error(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		mockResults map[string]command.MockResult
		wantErr     string
	}{
		{
			name: "add fails",
			mockResults: map[string]command.MockResult{
				command.FormatCmd("git", "add", "."): {Error: fmt.Errorf("add fail")},
			},
			wantErr: "add fail",
		},
		{
			name: "commit fails",
			mockResults: map[string]command.MockResult{
				command.FormatCmd("git", "add", "."):            {},
				command.FormatCmd("git", "commit", "-m", "msg"): {Error: fmt.Errorf("commit fail")},
			},
			wantErr: "commit fail",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mocker := &command.MockCommander{MockResults: tc.mockResults}
			ctx := mocker.InjectContext(t.Context())
			err := commitChanges(ctx, "msg")
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("got error %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestPushChanges(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
	}{
		{name: "standard push"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mocker := &command.MockCommander{}
			ctx := mocker.InjectContext(t.Context())

			if err := pushChanges(ctx); err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			want := [][]string{{"git", "push", "-u", "origin", "HEAD"}}
			if diff := cmp.Diff(want, mocker.GotCommands); diff != "" {
				t.Errorf("command mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPushChanges_Error(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		mockErr error
	}{
		{name: "remote rejected", mockErr: fmt.Errorf("permission denied")},
		{name: "network failure", mockErr: fmt.Errorf("could not resolve host")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mocker := &command.MockCommander{Default: command.MockResult{Error: tc.mockErr}}
			ctx := mocker.InjectContext(t.Context())

			if err := pushChanges(ctx); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestCreatePR(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		details GithubDetails
	}{
		{
			name:    "minimal pr",
			details: GithubDetails{PrTitle: "T", PrBody: "B"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mocker := &command.MockCommander{}
			ctx := mocker.InjectContext(t.Context())

			if err := createPR(ctx, tc.details); err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			want := [][]string{{"gh", "pr", "create", "--title", tc.details.PrTitle, "--body", tc.details.PrBody}}
			if diff := cmp.Diff(want, mocker.GotCommands); diff != "" {
				t.Errorf("command mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreatePR_Error(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		mockErr error
	}{
		{name: "gh cli error", mockErr: fmt.Errorf("could not create pr")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mocker := &command.MockCommander{Default: command.MockResult{Error: tc.mockErr}}
			ctx := mocker.InjectContext(t.Context())
			if err := createPR(ctx, GithubDetails{}); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestGenerateBranchName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		prefix string
		time   time.Time
		want   string
	}{
		{
			name:   "standard date",
			prefix: "pref-",
			time:   time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
			want:   "pref-2026-01-02",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := generateBranchName(tc.prefix, tc.time)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
