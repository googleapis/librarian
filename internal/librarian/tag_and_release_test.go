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

func TestParsePullRequestBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []libraryRelease
	}{
		{
			name: "single library",
			body: `
Librarian Version: v0.2.0
Language Image: image

<details><summary>google-cloud-storage: 1.2.3</summary>

[1.2.3](https://github.com/googleapis/google-cloud-go/compare/google-cloud-storage-v1.2.2...google-cloud-storage-v1.2.3) (2025-08-15)

### Features

* Add new feature ([abcdef1](https://github.com/googleapis/google-cloud-go/commit/abcdef1))

</details>`,
			want: []libraryRelease{
				{
					Version: "1.2.3",
					Library: "google-cloud-storage",
					Body: `[1.2.3](https://github.com/googleapis/google-cloud-go/compare/google-cloud-storage-v1.2.2...google-cloud-storage-v1.2.3) (2025-08-15)

### Features

* Add new feature ([abcdef1](https://github.com/googleapis/google-cloud-go/commit/abcdef1))`,
				},
			},
		},
		{
			name: "multiple libraries",
			body: `
Librarian Version: 1.2.3
Language Image: gcr.io/test/image:latest

<details><summary>library-one: 1.0.0</summary>

[1.0.0](https://github.com/googleapis/repo/compare/library-one-v0.9.0...library-one-v1.0.0) (2025-08-15)

### Features

* some feature ([1234567](https://github.com/googleapis/repo/commit/1234567))

</details>

<details><summary>another-library-name: 2.3.4</summary>

[2.3.4](https://github.com/googleapis/repo/compare/another-library-name-v2.3.3...another-library-name-v2.3.4) (2025-08-15)

### Bug Fixes

* some bug fix ([abcdefg](https://github.com/googleapis/repo/commit/abcdefg))

</details>`,
			want: []libraryRelease{
				{
					Version: "1.0.0",
					Library: "library-one",
					Body: `[1.0.0](https://github.com/googleapis/repo/compare/library-one-v0.9.0...library-one-v1.0.0) (2025-08-15)

### Features

* some feature ([1234567](https://github.com/googleapis/repo/commit/1234567))`,
				},
				{
					Version: "2.3.4",
					Library: "another-library-name",
					Body: `[2.3.4](https://github.com/googleapis/repo/compare/another-library-name-v2.3.3...another-library-name-v2.3.4) (2025-08-15)

### Bug Fixes

* some bug fix ([abcdefg](https://github.com/googleapis/repo/commit/abcdefg))`,
				},
			},
		},
		{
			name: "empty body",
			body: "",
			want: nil,
		},
		{
			name: "malformed summary",
			body: `
Librarian Version: 1.2.3
Language Image: gcr.io/test/image:latest

<details><summary>no-version-here</summary>

some content

</details>`,
			want: nil,
		},
		{
			name: "v prefix in version",
			body: `
<details><summary>google-cloud-storage: v1.2.3</summary>

[v1.2.3](https://github.com/googleapis/google-cloud-go/compare/google-cloud-storage-v1.2.2...google-cloud-storage-v1.2.3) (2025-08-15)

</details>`,
			want: []libraryRelease{
				{
					Version: "v1.2.3",
					Library: "google-cloud-storage",
					Body:    "[v1.2.3](https://github.com/googleapis/google-cloud-go/compare/google-cloud-storage-v1.2.2...google-cloud-storage-v1.2.3) (2025-08-15)",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePullRequestBody(tt.body)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ParsePullRequestBody() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
