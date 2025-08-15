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
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
)

func TestNewTagAndReleaseRunner(t *testing.T) {
	testcases := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &config.Config{
				GitHubToken: "some-token",
				Repo:        newTestGitRepo(t).GetDir(),
				WorkRoot:    t.TempDir(),
				CommandName: tagAndReleaseCmdName,
			},
			wantErr: false,
		},
		{
			name: "missing github token",
			cfg: &config.Config{
				CommandName: tagAndReleaseCmdName,
			},
			wantErr: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := newTagAndReleaseRunner(tc.cfg)
			if (err != nil) != tc.wantErr {
				t.Errorf("newTagAndReleaseRunner() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr && r == nil {
				t.Errorf("newTagAndReleaseRunner() got nil runner, want non-nil")
			}
		})
	}
}

func TestDeterminePullRequestsToProcess(t *testing.T) {
	pr123 := &github.PullRequest{}
	for _, test := range []struct {
		name       string
		cfg        *config.Config
		ghClient   GitHubClient
		want       []*github.PullRequest
		wantErrMsg string
	}{
		{
			name: "with pull request config",
			cfg: &config.Config{
				PullRequest: "github.com/googleapis/librarian/pulls/123",
			},
			ghClient: &mockGitHubClient{
				getPullRequestCalls: 1,
				pullRequest:         pr123,
			},
			want: []*github.PullRequest{pr123},
		},
		{
			name: "invalid pull request format",
			cfg: &config.Config{
				PullRequest: "invalid",
			},
			ghClient:   &mockGitHubClient{},
			wantErrMsg: "invalid pull request format",
		},
		{
			name: "invalid pull request number",
			cfg: &config.Config{
				PullRequest: "github.com/googleapis/librarian/pulls/abc",
			},
			ghClient:   &mockGitHubClient{},
			wantErrMsg: "invalid pull request number",
		},
		{
			name: "get pull request error",
			cfg: &config.Config{
				PullRequest: "github.com/googleapis/librarian/pulls/123",
			},
			ghClient: &mockGitHubClient{
				getPullRequestCalls: 1,
				getPullRequestErr:   errors.New("get pr error"),
			},
			wantErrMsg: "failed to get pull request",
		},
		{
			name: "search pull requests",
			cfg:  &config.Config{},
			ghClient: &mockGitHubClient{
				searchPullRequestsCalls: 1,
				pullRequests:            []*github.PullRequest{pr123},
			},
			want: []*github.PullRequest{pr123},
		},
		{
			name: "search pull requests error",
			cfg:  &config.Config{},
			ghClient: &mockGitHubClient{
				searchPullRequestsCalls: 1,
				searchPullRequestsErr:   errors.New("search pr error"),
			},
			wantErrMsg: "failed to search pull requests",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := &tagAndReleaseRunner{
				cfg:      test.cfg,
				ghClient: test.ghClient,
			}
			got, err := r.determinePullRequestsToProcess(context.Background())
			if err != nil {
				if test.wantErrMsg == "" {
					t.Fatalf("unexpected error: %v", err)
				}
				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Fatalf("got %q, want contains %q", err, test.wantErrMsg)
				}
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("determinePullRequestsToProcess() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_tagAndReleaseRunner_run(t *testing.T) {
	pr123 := &github.PullRequest{}
	pr456 := &github.PullRequest{}

	for _, test := range []struct {
		name                        string
		ghClient                    *mockGitHubClient
		wantErrMsg                  string
		wantSearchPullRequestsCalls int
		wantGetPullRequestCalls     int
	}{
		{
			name:                        "no pull requests to process",
			ghClient:                    &mockGitHubClient{},
			wantSearchPullRequestsCalls: 1,
		},
		{
			name: "one pull request to process",
			ghClient: &mockGitHubClient{
				pullRequests: []*github.PullRequest{pr123},
			},
			wantSearchPullRequestsCalls: 1,
		},
		{
			name: "multiple pull requests to process",
			ghClient: &mockGitHubClient{
				pullRequests: []*github.PullRequest{pr123, pr456},
			},
			wantSearchPullRequestsCalls: 1,
		},
		{
			name: "error determining pull requests",
			ghClient: &mockGitHubClient{
				searchPullRequestsErr: errors.New("search pr error"),
			},
			wantSearchPullRequestsCalls: 1,
			wantErrMsg:                  "failed to search pull requests",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := &tagAndReleaseRunner{
				cfg:      &config.Config{}, // empty config so it searches
				ghClient: test.ghClient,
			}
			err := r.run(context.Background())
			if err != nil {
				if test.wantErrMsg == "" {
					t.Fatalf("unexpected error: %v", err)
				}
				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Fatalf("got %q, want contains %q", err, test.wantErrMsg)
				}
				return
			}
			if test.ghClient.searchPullRequestsCalls != test.wantSearchPullRequestsCalls {
				t.Errorf("searchPullRequestsCalls = %v, want %v", test.ghClient.searchPullRequestsCalls, test.wantSearchPullRequestsCalls)
			}
			if test.ghClient.getPullRequestCalls != test.wantGetPullRequestCalls {
				t.Errorf("getPullRequestCalls = %v, want %v", test.ghClient.getPullRequestCalls, test.wantGetPullRequestCalls)
			}
		})
	}
}
