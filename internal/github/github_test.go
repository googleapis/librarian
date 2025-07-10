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
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v69/github"
)

func newTestServerAndClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	client, err := NewClient("test-token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	c := github.NewClient(server.Client())
	c.BaseURL, _ = url.Parse(server.URL + "/")
	client.Client = c
	return client, server
}

func TestCreatePullRequest(t *testing.T) {
	repo := &Repository{Owner: "test-owner", Name: "test-repo"}
	remoteBranch := "test-branch"
	title := "Test PR"
	cases := []struct {
		name         string
		body         string
		mockResponse *github.PullRequest
		statusCode   int
		wantErr      bool
	}{
		{
			name: "Successful PR creation",
			body: "This is a test PR.",
			mockResponse: &github.PullRequest{
				Number:  github.Ptr(1),
				HTMLURL: github.Ptr("https://github.com/test-owner/test-repo/pull/1"),
			},
			statusCode: http.StatusCreated,
			wantErr:    false,
		},
		{
			name: "Successful PR creation with empty body",
			body: "",
			mockResponse: &github.PullRequest{
				Number:  github.Ptr(1),
				HTMLURL: github.Ptr("https://github.com/test-owner/test-repo/pull/1"),
			},
			statusCode: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:       "GitHub API error",
			body:       "This is a test PR.",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				if r.URL.Path != fmt.Sprintf("/repos/%s/%s/pulls", repo.Owner, repo.Name) {
					t.Errorf("expected request to /repos/%s/%s/pulls, got %s", repo.Owner, repo.Name, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
				if tc.mockResponse != nil {
					if err := json.NewEncoder(w).Encode(tc.mockResponse); err != nil {
						t.Fatalf("failed to write mock response: %v", err)
					}
				}
			})
			client, server := newTestServerAndClient(t, handler)
			defer server.Close()

			pr, err := client.CreatePullRequest(context.Background(), repo, remoteBranch, title, tc.body)

			if (err != nil) != tc.wantErr {
				t.Errorf("CreatePullRequest() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr {
				expectedPR := &PullRequestMetadata{
					Repo:   repo,
					Number: tc.mockResponse.GetNumber(),
				}
				if diff := cmp.Diff(expectedPR, pr); diff != "" {
					t.Errorf("CreatePullRequest() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestGetCommit(t *testing.T) {
	repo := &Repository{Owner: "test-owner", Name: "test-repo"}
	sha := "abcdef123"
	cases := []struct {
		name         string
		mockResponse *github.RepositoryCommit
		statusCode   int
		wantErr      bool
	}{
		{
			name: "Successful get",
			mockResponse: &github.RepositoryCommit{
				SHA: github.Ptr("abcdef123"),
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "GitHub API error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/repos/%s/%s/commits/%s", repo.Owner, repo.Name, sha)
				if r.URL.Path != expectedPath {
					t.Errorf("expected request to %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
				if tc.mockResponse != nil {
					if err := json.NewEncoder(w).Encode(tc.mockResponse); err != nil {
						t.Fatalf("failed to write mock response: %v", err)
					}
				}
			})
			client, server := newTestServerAndClient(t, handler)
			defer server.Close()

			commit, err := client.GetCommit(context.Background(), repo, sha)

			if (err != nil) != tc.wantErr {
				t.Errorf("GetCommit() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr {
				if diff := cmp.Diff(tc.mockResponse, commit); diff != "" {
					t.Errorf("GetCommit() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestGetRawContent(t *testing.T) {
	repo := &Repository{Owner: "test-owner", Name: "test-repo"}
	filePath := "test-path"
	ref := "test-ref"
	cases := []struct {
		name               string
		mockResponse       string
		statusCode         int
		downloadStatusCode int
		wantErr            bool
	}{
		{
			name:         "Successful get",
			mockResponse: "test content",
			statusCode:   http.StatusOK,
			wantErr:      false,
		},
		{
			name:       "GitHub API error on GetContents",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:               "Download returns non-OK status",
			statusCode:         http.StatusOK,
			downloadStatusCode: http.StatusNoContent,
			wantErr:            true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var serverURL string
			downloadStatusCode := tc.statusCode
			if tc.downloadStatusCode != 0 {
				downloadStatusCode = tc.downloadStatusCode
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "download") {
					// This is the download request
					w.WriteHeader(downloadStatusCode)
					if tc.mockResponse != "" && downloadStatusCode == http.StatusOK {
						_, err := w.Write([]byte(tc.mockResponse))
						if err != nil {
							t.Fatalf("failed to write mock response: %v", err)
						}
					}
					return
				}

				// This is the GetContents request
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				if tc.statusCode == http.StatusOK {
					response := []*github.RepositoryContent{
						{
							Name:        github.Ptr("test-path"),
							DownloadURL: github.Ptr(serverURL + "/download"),
						},
					}
					if err := json.NewEncoder(w).Encode(response); err != nil {
						t.Fatalf("failed to write mock response: %v", err)
					}
				}
			})
			client, server := newTestServerAndClient(t, handler)
			serverURL = server.URL
			defer server.Close()

			content, err := client.GetRawContent(context.Background(), repo, filePath, ref)

			if (err != nil) != tc.wantErr {
				t.Errorf("GetRawContent() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr {
				if diff := cmp.Diff(tc.mockResponse, string(content)); diff != "" {
					t.Errorf("GetRawContent() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestCreateGitHubRepoFromRepository(t *testing.T) {
	repo := &github.Repository{
		Owner: &github.User{
			Login: github.Ptr("test-owner"),
		},
		Name: github.Ptr("test-repo"),
	}
	want := &Repository{Owner: "test-owner", Name: "test-repo"}

	got := CreateGitHubRepoFromRepository(repo)

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("CreateGitHubRepoFromRepository() mismatch (-want +got):\n%s", diff)
	}
}

func TestParseUrl(t *testing.T) {
	cases := []struct {
		name      string
		remoteUrl string
		want      *Repository
		wantErr   bool
	}{
		{
			name:      "Valid HTTPS URL",
			remoteUrl: "https://github.com/test-owner/test-repo.git",
			want:      &Repository{Owner: "test-owner", Name: "test-repo"},
			wantErr:   false,
		},
		{
			name:      "Valid HTTPS URL without .git",
			remoteUrl: "https://github.com/test-owner/test-repo",
			want:      &Repository{Owner: "test-owner", Name: "test-repo"},
			wantErr:   false,
		},
		{
			name:      "Invalid URL scheme",
			remoteUrl: "http://github.com/test-owner/test-repo",
			wantErr:   true,
		},
		{
			name:      "Invalid URL format",
			remoteUrl: "https://github.com/test-owner",
			wantErr:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, err := ParseUrl(tc.remoteUrl)

			if (err != nil) != tc.wantErr {
				t.Errorf("ParseUrl() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr {
				if diff := cmp.Diff(tc.want, repo); diff != "" {
					t.Errorf("ParseUrl() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestGetPullRequestReviews(t *testing.T) {
	prMetadata := &PullRequestMetadata{
		Repo:   &Repository{Owner: "test-owner", Name: "test-repo"},
		Number: 1,
	}
	cases := []struct {
		name         string
		mockResponse []*github.PullRequestReview
		statusCode   int
		wantErr      bool
	}{
		{
			name: "Successful get",
			mockResponse: []*github.PullRequestReview{
				{
					ID:   github.Ptr(int64(1)),
					Body: github.Ptr("LGTM"),
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "GitHub API error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", prMetadata.Repo.Owner, prMetadata.Repo.Name, prMetadata.Number)
				if r.URL.Path != expectedPath {
					t.Errorf("expected request to %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
				if tc.mockResponse != nil {
					if err := json.NewEncoder(w).Encode(tc.mockResponse); err != nil {
						t.Fatalf("failed to write mock response: %v", err)
					}
				}
			})
			client, server := newTestServerAndClient(t, handler)
			defer server.Close()

			reviews, err := client.GetPullRequestReviews(context.Background(), prMetadata)

			if (err != nil) != tc.wantErr {
				t.Errorf("GetPullRequestReviews() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr {
				if diff := cmp.Diff(tc.mockResponse, reviews); diff != "" {
					t.Errorf("GetPullRequestReviews() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestGetPullRequestCheckRuns(t *testing.T) {
	pullRequest := &github.PullRequest{
		Head: &github.PullRequestBranch{
			Ref: github.Ptr("test-branch"),
			Repo: &github.Repository{
				Owner: &github.User{
					Login: github.Ptr("test-owner"),
				},
				Name: github.Ptr("test-repo"),
			},
			User: &github.User{
				Login: github.Ptr("test-owner"),
			},
		},
	}
	cases := []struct {
		name         string
		mockResponse *github.ListCheckRunsResults
		statusCode   int
		wantErr      bool
	}{
		{
			name: "Successful get",
			mockResponse: &github.ListCheckRunsResults{
				Total: github.Ptr(1),
				CheckRuns: []*github.CheckRun{
					{
						ID:     github.Ptr(int64(1)),
						Status: github.Ptr("completed"),
					},
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "GitHub API error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/repos/%s/%s/commits/%s/check-runs", *pullRequest.Head.Repo.Owner.Login, *pullRequest.Head.Repo.Name, *pullRequest.Head.Ref)
				if r.URL.Path != expectedPath {
					t.Errorf("expected request to %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
				if tc.mockResponse != nil {
					if err := json.NewEncoder(w).Encode(tc.mockResponse); err != nil {
						t.Fatalf("failed to write mock response: %v", err)
					}
				}
			})
			client, server := newTestServerAndClient(t, handler)
			defer server.Close()

			checkRuns, err := client.GetPullRequestCheckRuns(context.Background(), pullRequest)

			if (err != nil) != tc.wantErr {
				t.Errorf("GetPullRequestCheckRuns() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr {
				if diff := cmp.Diff(tc.mockResponse.CheckRuns, checkRuns); diff != "" {
					t.Errorf("GetPullRequestCheckRuns() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestGetPullRequest(t *testing.T) {
	repo := &Repository{Owner: "test-owner", Name: "test-repo"}
	prNumber := 1
	cases := []struct {
		name         string
		mockResponse *github.PullRequest
		statusCode   int
		wantErr      bool
	}{
		{
			name: "Successful get",
			mockResponse: &github.PullRequest{
				Number: github.Ptr(1),
				Title:  github.Ptr("Test PR"),
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "GitHub API error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/repos/%s/%s/pulls/%d", repo.Owner, repo.Name, prNumber)
				if r.URL.Path != expectedPath {
					t.Errorf("expected request to %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
				if tc.mockResponse != nil {
					if err := json.NewEncoder(w).Encode(tc.mockResponse); err != nil {
						t.Fatalf("failed to write mock response: %v", err)
					}
				}
			})
			client, server := newTestServerAndClient(t, handler)
			defer server.Close()

			pr, err := client.GetPullRequest(context.Background(), repo, prNumber)

			if (err != nil) != tc.wantErr {
				t.Errorf("GetPullRequest() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr {
				if diff := cmp.Diff(tc.mockResponse, pr); diff != "" {
					t.Errorf("GetPullRequest() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestMergePullRequest(t *testing.T) {
	repo := &Repository{Owner: "test-owner", Name: "test-repo"}
	prNumber := 1
	method := MergeMethodRebase
	cases := []struct {
		name         string
		mockResponse *github.PullRequestMergeResult
		statusCode   int
		wantErr      bool
	}{
		{
			name: "Successful merge",
			mockResponse: &github.PullRequestMergeResult{
				SHA:     github.Ptr("abcdef123"),
				Merged:  github.Ptr(true),
				Message: github.Ptr("PR merged"),
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "GitHub API error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Errorf("expected PUT request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/repos/%s/%s/pulls/%d/merge", repo.Owner, repo.Name, prNumber)
				if r.URL.Path != expectedPath {
					t.Errorf("expected request to %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
				if tc.mockResponse != nil {
					if err := json.NewEncoder(w).Encode(tc.mockResponse); err != nil {
						t.Fatalf("failed to write mock response: %v", err)
					}
				}
			})
			client, server := newTestServerAndClient(t, handler)
			defer server.Close()

			result, err := client.MergePullRequest(context.Background(), repo, prNumber, method)

			if (err != nil) != tc.wantErr {
				t.Errorf("MergePullRequest() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr {
				if diff := cmp.Diff(tc.mockResponse, result); diff != "" {
					t.Errorf("MergePullRequest() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestAddCommentToPullRequest(t *testing.T) {
	repo := &Repository{Owner: "test-owner", Name: "test-repo"}
	prNumber := 1
	comment := "test comment"
	cases := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "Successful comment addition",
			statusCode: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:       "GitHub API error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", repo.Owner, repo.Name, prNumber)
				if r.URL.Path != expectedPath {
					t.Errorf("expected request to %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
			})
			client, server := newTestServerAndClient(t, handler)
			defer server.Close()

			err := client.AddCommentToPullRequest(context.Background(), repo, prNumber, comment)

			if (err != nil) != tc.wantErr {
				t.Errorf("AddCommentToPullRequest() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestRemoveLabelFromPullRequest(t *testing.T) {
	repo := &Repository{Owner: "test-owner", Name: "test-repo"}
	prNumber := 1
	label := "test-label"
	cases := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "Successful label removal",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "GitHub API error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("expected DELETE request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/repos/%s/%s/issues/%d/labels/%s", repo.Owner, repo.Name, prNumber, label)
				if r.URL.Path != expectedPath {
					t.Errorf("expected request to %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
			})
			client, server := newTestServerAndClient(t, handler)
			defer server.Close()

			err := client.RemoveLabelFromPullRequest(context.Background(), repo, prNumber, label)

			if (err != nil) != tc.wantErr {
				t.Errorf("RemoveLabelFromPullRequest() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestAddLabelToPullRequest(t *testing.T) {
	prMetadata := &PullRequestMetadata{
		Repo:   &Repository{Owner: "test-owner", Name: "test-repo"},
		Number: 1,
	}
	label := "test-label"
	cases := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "Successful label addition",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "GitHub API error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/repos/%s/%s/issues/%d/labels", prMetadata.Repo.Owner, prMetadata.Repo.Name, prMetadata.Number)
				if r.URL.Path != expectedPath {
					t.Errorf("expected request to %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
			})
			client, server := newTestServerAndClient(t, handler)
			defer server.Close()

			err := client.AddLabelToPullRequest(context.Background(), prMetadata, label)

			if (err != nil) != tc.wantErr {
				t.Errorf("AddLabelToPullRequest() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestGetDiffCommits(t *testing.T) {
	repo := &Repository{Owner: "test-owner", Name: "test-repo"}
	source := "main"
	target := "test-branch"
	cases := []struct {
		name         string
		mockResponse *github.CommitsComparison
		statusCode   int
		wantErr      bool
	}{
		{
			name: "Successful get",
			mockResponse: &github.CommitsComparison{
				Commits: []*github.RepositoryCommit{
					{
						SHA: github.Ptr("abcdef123"),
					},
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "GitHub API error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/repos/%s/%s/compare/%s...%s", repo.Owner, repo.Name, source, target)
				if r.URL.Path != expectedPath {
					t.Errorf("expected request to %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
				if tc.mockResponse != nil {
					if err := json.NewEncoder(w).Encode(tc.mockResponse); err != nil {
						t.Fatalf("failed to write mock response: %v", err)
					}
				}
			})
			client, server := newTestServerAndClient(t, handler)
			defer server.Close()

			commits, err := client.GetDiffCommits(context.Background(), repo, source, target)

			if (err != nil) != tc.wantErr {
				t.Errorf("GetDiffCommits() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr {
				if diff := cmp.Diff(tc.mockResponse.Commits, commits); diff != "" {
					t.Errorf("GetDiffCommits() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestCreateRelease(t *testing.T) {
	repo := &Repository{Owner: "test-owner", Name: "test-repo"}
	tag := "v1.0.0"
	commit := "abcdef123"
	title := "Release v1.0.0"
	description := "This is a test release."
	prerelease := false
	cases := []struct {
		name         string
		mockResponse *github.RepositoryRelease
		statusCode   int
		wantErr      bool
	}{
		{
			name: "Successful release creation",
			mockResponse: &github.RepositoryRelease{
				ID:      github.Ptr(int64(1)),
				TagName: github.Ptr("v1.0.0"),
			},
			statusCode: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:       "GitHub API error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				if r.URL.Path != fmt.Sprintf("/repos/%s/%s/releases", repo.Owner, repo.Name) {
					t.Errorf("expected request to /repos/%s/%s/releases, got %s", repo.Owner, repo.Name, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
				if tc.mockResponse != nil {
					if err := json.NewEncoder(w).Encode(tc.mockResponse); err != nil {
						t.Fatalf("failed to write mock response: %v", err)
					}
				}
			})
			client, server := newTestServerAndClient(t, handler)
			defer server.Close()

			release, err := client.CreateRelease(context.Background(), repo, tag, commit, title, description, prerelease)

			if (err != nil) != tc.wantErr {
				t.Errorf("CreateRelease() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr {
				if diff := cmp.Diff(tc.mockResponse, release); diff != "" {
					t.Errorf("CreateRelease() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
