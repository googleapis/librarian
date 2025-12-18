// Copyright 2025 Google LLC
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

package legacylibrarian

import (
	"errors"
	"strings"
	"testing"

	"cloud.google.com/go/librarian/internal/legacylibrarian/legacyconfig"
	"cloud.google.com/go/librarian/internal/legacylibrarian/legacygithub"
	"github.com/google/go-cmp/cmp"
	gh "github.com/google/go-github/v69/github"
)

func TestNewTagRunner(t *testing.T) {
	testcases := []struct {
		name    string
		cfg     *legacyconfig.Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &legacyconfig.Config{
				GitHubToken: "some-token",
				Repo:        "https://github.com/googleapis/some-test-repo",
				WorkRoot:    t.TempDir(),
				CommandName: tagCmdName,
			},
			wantErr: false,
		},
		{
			name: "missing github token",
			cfg: &legacyconfig.Config{
				CommandName: tagCmdName,
			},
			wantErr: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := newTagRunner(tc.cfg)
			if (err != nil) != tc.wantErr {
				t.Errorf("newTagRunner() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr && r == nil {
				t.Errorf("newTagRunner() got nil runner, want non-nil")
			}
		})
	}
}

func TestDeterminePullRequestsToProcess(t *testing.T) {
	pr123 := &legacygithub.PullRequest{}
	for _, test := range []struct {
		name       string
		cfg        *legacyconfig.Config
		ghClient   GitHubClient
		want       []*legacygithub.PullRequest
		wantErrMsg string
	}{
		{
			name: "with pull request config",
			cfg: &legacyconfig.Config{
				PullRequest: "https://github.com/googleapis/librarian/pulls/123",
			},
			ghClient: &mockGitHubClient{
				getPullRequestCalls: 1,
				pullRequest:         pr123,
			},
			want: []*legacygithub.PullRequest{pr123},
		},
		{
			name: "invalid pull request format",
			cfg: &legacyconfig.Config{
				PullRequest: "invalid",
			},
			ghClient:   &mockGitHubClient{},
			wantErrMsg: "invalid pull request format",
		},
		{
			name: "invalid pull request number",
			cfg: &legacyconfig.Config{
				PullRequest: "https://github.com/googleapis/librarian/pulls/abc",
			},
			ghClient:   &mockGitHubClient{},
			wantErrMsg: "invalid pull request number",
		},
		{
			name: "get pull request error",
			cfg: &legacyconfig.Config{
				PullRequest: "https://github.com/googleapis/librarian/pulls/123",
			},
			ghClient: &mockGitHubClient{
				getPullRequestCalls: 1,
				getPullRequestErr:   errors.New("get pr error"),
			},
			wantErrMsg: "failed to get pull request",
		},
		{
			name: "search pull requests",
			cfg:  &legacyconfig.Config{},
			ghClient: &mockGitHubClient{
				searchPullRequestsCalls: 1,
				pullRequests:            []*legacygithub.PullRequest{pr123},
			},
			want: []*legacygithub.PullRequest{pr123},
		},
		{
			name: "search pull requests error",
			cfg:  &legacyconfig.Config{},
			ghClient: &mockGitHubClient{
				searchPullRequestsCalls: 1,
				searchPullRequestsErr:   errors.New("search pr error"),
			},
			wantErrMsg: "failed to search pull requests",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := &tagRunner{
				pullRequest: test.cfg.PullRequest,
				ghClient:    test.ghClient,
			}
			got, err := r.determinePullRequestsToProcess(t.Context())
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

func Test_tagRunner_run(t *testing.T) {
	pr123 := &legacygithub.PullRequest{}
	pr456 := &legacygithub.PullRequest{}

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
				pullRequests: []*legacygithub.PullRequest{pr123},
			},
			wantSearchPullRequestsCalls: 1,
		},
		{
			name: "multiple pull requests to process",
			ghClient: &mockGitHubClient{
				pullRequests: []*legacygithub.PullRequest{pr123, pr456},
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
			r := &tagRunner{
				ghClient: test.ghClient,
			}
			err := r.run(t.Context())
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
	for _, test := range []struct {
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

* some feature ([12345678](https://github.com/googleapis/repo/commit/12345678))

</details>

<details><summary>library-two: 2.3.4</summary>

[2.3.4](https://github.com/googleapis/repo/compare/library-two-v2.3.3...library-two-v2.3.4) (2025-08-15)

### Bug Fixes

* some bug fix ([abcdefg](https://github.com/googleapis/repo/commit/abcdefg))

</details>`,
			want: []libraryRelease{
				{
					Version: "1.0.0",
					Library: "library-one",
					Body: `[1.0.0](https://github.com/googleapis/repo/compare/library-one-v0.9.0...library-one-v1.0.0) (2025-08-15)

### Features

* some feature ([12345678](https://github.com/googleapis/repo/commit/12345678))`,
				},
				{
					Version: "2.3.4",
					Library: "library-two",
					Body: `[2.3.4](https://github.com/googleapis/repo/compare/library-two-v2.3.3...library-two-v2.3.4) (2025-08-15)

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
		{
			name: "with bulk changes",
			body: `
<details><summary>google-cloud-storage: v1.2.3</summary>

[v1.2.3](https://github.com/googleapis/google-cloud-go/compare/google-cloud-storage-v1.2.2...google-cloud-storage-v1.2.3) (2025-08-15)

</details>


<details><summary>Bulk Changes</summary>

some content

</details>`,
			want: []libraryRelease{
				{
					Version: "v1.2.3",
					Library: "google-cloud-storage",
					Body:    "[v1.2.3](https://github.com/googleapis/google-cloud-go/compare/google-cloud-storage-v1.2.2...google-cloud-storage-v1.2.3) (2025-08-15)",
				},
			},
		},
		{
			name: "bulk_changes_appears_in_github_release",
			body: `
<details><summary>google-cloud-storage: v1.2.3</summary>

[v1.2.3](https://github.com/googleapis/google-cloud-go/compare/google-cloud-storage-v1.2.2...google-cloud-storage-v1.2.3) (2025-08-15)

### Features

* add SaveToGcsFindingsOutput ([1234567](https://github.com/googleapis/google-cloud-go/commit/1234567))

* another feature ([9876543](https://github.com/googleapis/google-cloud-go/commit/9876543))

### Documentation

* minor doc revision ([abcdefgh](https://github.com/googleapis/google-cloud-go/commit/abcdefgh))

</details>


<details><summary>Bulk Changes</summary>

* feat: this is a bulk change ([abcdefgh](https://github.com/googleapis/google-cloud-go/commit/abcdefgh))
Libraries: a,b,google-cloud-storage

* fix: this is another bulk change
Libraries: a,b,c

</details>`,
			want: []libraryRelease{
				{
					Version: "",
					Library: "a",
					Body: `### Features

* this is a bulk change ([abcdefgh](https://github.com/googleapis/google-cloud-go/commit/abcdefgh))

### Bug Fixes

* this is another bulk change`,
				},
				{
					Version: "",
					Library: "b",
					Body: `### Features

* this is a bulk change ([abcdefgh](https://github.com/googleapis/google-cloud-go/commit/abcdefgh))

### Bug Fixes

* this is another bulk change`,
				},
				{
					Version: "",
					Library: "c",
					Body: `### Bug Fixes

* this is another bulk change`,
				},
				{
					Version: "v1.2.3",
					Library: "google-cloud-storage",
					Body: `[v1.2.3](https://github.com/googleapis/google-cloud-go/compare/google-cloud-storage-v1.2.2...google-cloud-storage-v1.2.3) (2025-08-15)

### Features

* add SaveToGcsFindingsOutput ([1234567](https://github.com/googleapis/google-cloud-go/commit/1234567))

* another feature ([9876543](https://github.com/googleapis/google-cloud-go/commit/9876543))

* this is a bulk change ([abcdefgh](https://github.com/googleapis/google-cloud-go/commit/abcdefgh))

### Documentation

* minor doc revision ([abcdefgh](https://github.com/googleapis/google-cloud-go/commit/abcdefgh))`,
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := parsePullRequestBody(test.body)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("ParsePullRequestBody() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProcessPullRequest(t *testing.T) {
	prBody := `<details><summary>google-cloud-storage: v1.2.3</summary>release notes</details>`
	prNumber := 123
	mergeCommitSHA := "abcdef"
	branch := "main"
	prWithRelease := &legacygithub.PullRequest{
		Body:           &prBody,
		Number:         &prNumber,
		MergeCommitSHA: &mergeCommitSHA,
		Labels:         []*gh.Label{{Name: gh.Ptr(releasePendingLabel)}},
		Base: &gh.PullRequestBranch{
			Ref: &branch,
		},
	}
	body := "no release details"
	prWithoutRelease := &legacygithub.PullRequest{
		Body:           &body,
		Number:         &prNumber,
		MergeCommitSHA: &mergeCommitSHA,
		Labels:         []*gh.Label{{Name: gh.Ptr(releasePendingLabel)}},
		Base: &gh.PullRequestBranch{
			Ref: &branch,
		},
	}
	state := &legacyconfig.LibrarianState{
		Image: "gcr.io/some-project-id/some-test-image:latest",
		Libraries: []*legacyconfig.LibraryState{
			{
				ID:          "google-cloud-storage",
				SourceRoots: []string{"some/path"},
				TagFormat:   "v{version}",
			},
		},
	}

	for _, test := range []struct {
		name                   string
		pr                     *legacygithub.PullRequest
		ghClient               *mockGitHubClient
		wantErrMsg             string
		wantCreateReleaseCalls int
		wantReplaceLabelsCalls int
		wantCreateTagCalls     int
	}{
		{
			name: "happy path",
			pr:   prWithRelease,
			ghClient: &mockGitHubClient{
				librarianState: state,
			},
			wantCreateReleaseCalls: 1,
			wantReplaceLabelsCalls: 1,
			wantCreateTagCalls:     1,
		},
		{
			name: "no release details",
			pr:   prWithoutRelease,
			ghClient: &mockGitHubClient{
				librarianState: state,
			}},
		{
			name: "library not found",
			pr:   prWithRelease,
			ghClient: &mockGitHubClient{
				librarianState: &legacyconfig.LibrarianState{
					Image: "gcr.io/some-project-id/some-test-image:latest",
					Libraries: []*legacyconfig.LibraryState{
						{
							ID:          "other-library",
							SourceRoots: []string{"some/path"},
							TagFormat:   "v{version}",
						},
					},
				},
			},
			wantErrMsg: "library google-cloud-storage not found",
		},
		{
			name: "default tag format",
			pr:   prWithRelease,
			ghClient: &mockGitHubClient{
				librarianState: &legacyconfig.LibrarianState{
					Image: "gcr.io/some-project-id/some-test-image:latest",
					Libraries: []*legacyconfig.LibraryState{
						{
							ID:          "google-cloud-storage",
							SourceRoots: []string{"some/path"},
						},
					},
				},
			},
			wantCreateReleaseCalls: 1,
			wantReplaceLabelsCalls: 1,
			wantCreateTagCalls:     1,
		},
		{
			name: "skip_a_library_release",
			pr:   prWithRelease,
			ghClient: &mockGitHubClient{
				librarianState: &legacyconfig.LibrarianState{
					Image: "gcr.io/some-project-id/some-test-image:latest",
					Libraries: []*legacyconfig.LibraryState{
						{
							ID:          "google-cloud-storage",
							SourceRoots: []string{"some/path"},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{
					Libraries: []*legacyconfig.LibraryConfig{
						{
							LibraryID:                 "google-cloud-storage",
							SkipGitHubReleaseCreation: true,
						},
					},
				},
			},
			wantCreateReleaseCalls: 0,
			wantReplaceLabelsCalls: 1,
			wantCreateTagCalls:     1,
		},
		{
			name: "create release fails",
			pr:   prWithRelease,
			ghClient: &mockGitHubClient{
				createReleaseErr: errors.New("create release error"),
				librarianState:   state,
			},
			wantErrMsg:             "failed to create release",
			wantCreateReleaseCalls: 1,
			wantCreateTagCalls:     1,
		},
		{
			name: "replace labels fails",
			pr:   prWithRelease,
			ghClient: &mockGitHubClient{
				replaceLabelsErr: errors.New("replace labels error"),
				librarianState:   state,
			},
			wantErrMsg:             "failed to replace labels",
			wantCreateReleaseCalls: 1,
			wantReplaceLabelsCalls: 1,
			wantCreateTagCalls:     1,
		},
		{
			name: "create tag fails",
			pr:   prWithRelease,
			ghClient: &mockGitHubClient{
				createTagErr:   errors.New("create tag error"),
				librarianState: state,
			},
			wantErrMsg:         "failed to create tag",
			wantCreateTagCalls: 1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := &tagRunner{
				ghClient: test.ghClient,
			}
			err := r.processPullRequest(t.Context(), test.pr)
			if err != nil {
				if test.wantErrMsg == "" {
					t.Fatalf("unexpected error: %v", err)
				}
				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Fatalf("got %q, want contains %q", err, test.wantErrMsg)
				}
			} else if test.wantErrMsg != "" {
				t.Fatalf("expected error containing %q, got nil", test.wantErrMsg)
			}

			if test.ghClient.createReleaseCalls != test.wantCreateReleaseCalls {
				t.Errorf("createReleaseCalls = %v, want %v", test.ghClient.createReleaseCalls, test.wantCreateReleaseCalls)
			}
			if test.ghClient.replaceLabelsCalls != test.wantReplaceLabelsCalls {
				t.Errorf("replaceLabelsCalls = %v, want %v", test.ghClient.replaceLabelsCalls, test.wantReplaceLabelsCalls)
			}
		})
	}
}

func TestReplacePendingLabel(t *testing.T) {
	prWithPending := &legacygithub.PullRequest{
		Number: gh.Ptr(123),
		Labels: []*gh.Label{{Name: gh.Ptr(releasePendingLabel)}, {Name: gh.Ptr("label1")}},
	}
	prWithoutPending := &legacygithub.PullRequest{
		Number: gh.Ptr(123),
		Labels: []*gh.Label{{Name: gh.Ptr("label1")}},
	}

	for _, test := range []struct {
		name       string
		pr         *legacygithub.PullRequest
		ghClient   *mockGitHubClient
		wantErrMsg string
	}{
		{
			name:     "with pending label",
			pr:       prWithPending,
			ghClient: &mockGitHubClient{},
		},
		{
			name:     "without pending label",
			pr:       prWithoutPending,
			ghClient: &mockGitHubClient{},
		},
		{
			name: "replace labels fails",
			pr:   prWithPending,
			ghClient: &mockGitHubClient{
				replaceLabelsErr: errors.New("replace labels error"),
			},
			wantErrMsg: "failed to replace labels",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := &tagRunner{
				ghClient: test.ghClient,
			}
			err := r.replacePendingLabel(t.Context(), test.pr)
			if err != nil {
				if test.wantErrMsg == "" {
					t.Fatalf("unexpected error: %v", err)
				}
				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Fatalf("got %q, want contains %q", err, test.wantErrMsg)
				}
				return
			} else if test.wantErrMsg != "" {
				t.Fatalf("expected error containing %q, got nil", test.wantErrMsg)
			}
		})
	}
}

func Test_tagRunner_run_processPullRequests(t *testing.T) {
	branch := "main"
	pr1 := &legacygithub.PullRequest{
		Body:           gh.Ptr(`<details><summary>google-cloud-storage: v1.2.3</summary>release notes</details>`),
		Number:         gh.Ptr(123),
		MergeCommitSHA: gh.Ptr("abc123"),
		Labels:         []*gh.Label{{Name: gh.Ptr(releasePendingLabel)}},
		Base: &gh.PullRequestBranch{
			Ref: &branch,
		},
	}
	// This one will fail because the library details are not parsable.
	pr2 := &legacygithub.PullRequest{
		Body:           gh.Ptr(`<details><summary>unknown-library: v1.0.0</summary>release notes</details>`),
		Number:         gh.Ptr(456),
		MergeCommitSHA: gh.Ptr("xyz456"),
		Labels:         []*gh.Label{{Name: gh.Ptr(releasePendingLabel)}},
		Base: &gh.PullRequestBranch{
			Ref: &branch,
		},
	}
	ghClient := &mockGitHubClient{
		pullRequests: []*legacygithub.PullRequest{pr1, pr2},
		librarianState: &legacyconfig.LibrarianState{
			Image: "gcr.io/some-project/some-image:latest",
			Libraries: []*legacyconfig.LibraryState{
				{
					ID:          "google-cloud-storage",
					SourceRoots: []string{"some/path"},
					TagFormat:   "v{version}",
				},
			},
		},
	}

	r := &tagRunner{
		ghClient: ghClient,
	}
	err := r.run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "failed to process some pull requests") {
		t.Fatalf("expected error 'failed to process some pull requests', got %v", err)
	}
	if ghClient.createReleaseCalls != 1 {
		t.Errorf("createReleaseCalls = %v, want 1", ghClient.createReleaseCalls)
	}
	if ghClient.replaceLabelsCalls != 1 {
		t.Errorf("replaceLabelsCalls = %v, want 1", ghClient.replaceLabelsCalls)
	}
}
