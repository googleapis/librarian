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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v69/github"
)

func TestCreatePullRequest(t *testing.T) {
	cases := []struct {
		name         string
		repo         *Repository
		remoteBranch string
		title        string
		body         string
		mockResponse *github.PullRequest
		statusCode   int
		wantErr      bool
	}{
		{
			name:         "Successful PR creation",
			repo:         &Repository{Owner: "test-owner", Name: "test-repo"},
			remoteBranch: "test-branch",
			title:        "Test PR",
			body:         "This is a test PR.",
			mockResponse: &github.PullRequest{
				Number:  github.Ptr(1),
				HTMLURL: github.Ptr("https://github.com/test-owner/test-repo/pull/1"),
			},
			statusCode: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:         "Successful PR creation with empty body",
			repo:         &Repository{Owner: "test-owner", Name: "test-repo"},
			remoteBranch: "test-branch",
			title:        "Test PR",
			body:         "",
			mockResponse: &github.PullRequest{
				Number:  github.Ptr(1),
				HTMLURL: github.Ptr("https://github.com/test-owner/test-repo/pull/1"),
			},
			statusCode: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:         "GitHub API error",
			repo:         &Repository{Owner: "test-owner", Name: "test-repo"},
			remoteBranch: "test-branch",
			title:        "Test PR",
			body:         "This is a test PR.",
			statusCode:   http.StatusInternalServerError,
			wantErr:      true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				if r.URL.Path != fmt.Sprintf("/repos/%s/%s/pulls", tc.repo.Owner, tc.repo.Name) {
					t.Errorf("expected request to /repos/%s/%s/pulls, got %s", tc.repo.Owner, tc.repo.Name, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
				if tc.mockResponse != nil {
					if err := json.NewEncoder(w).Encode(tc.mockResponse); err != nil {
						t.Fatalf("failed to write mock response: %v", err)
					}
				}
			}))
			defer server.Close()

			client, err := NewClient("test-token")
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			c := github.NewClient(server.Client())
			c.BaseURL, _ = url.Parse(server.URL + "/")
			client.Client = c

			pr, err := client.CreatePullRequest(context.Background(), tc.repo, tc.remoteBranch, tc.title, tc.body)

			if (err != nil) != tc.wantErr {
				t.Errorf("CreatePullRequest() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr {
				expectedPR := &PullRequestMetadata{
					Repo:   tc.repo,
					Number: tc.mockResponse.GetNumber(),
				}
				if diff := cmp.Diff(expectedPR, pr); diff != "" {
					t.Errorf("CreatePullRequest() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestGetRawContent(t *testing.T) {
	cases := []struct {
		name         string
		repo         *Repository
		path         string
		ref          string
		mockResponse string
		statusCode   int
		wantErr      bool
	}{
		{
			name:         "Successful get",
			repo:         &Repository{Owner: "test-owner", Name: "test-repo"},
			path:         "test-path",
			ref:          "test-ref",
			mockResponse: "test content",
			statusCode:   http.StatusOK,
			wantErr:      false,
		},
		{
			name:       "GitHub API error",
			repo:       &Repository{Owner: "test-owner", Name: "test-repo"},
			path:       "test-path",
			ref:        "test-ref",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/repos/%s/%s/contents/%s", tc.repo.Owner, tc.repo.Name, tc.path)
				if r.URL.Path != expectedPath {
					t.Errorf("expected request to %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
				if tc.mockResponse != "" {
					_, err := w.Write([]byte(tc.mockResponse))
					if err != nil {
						t.Fatalf("failed to write mock response: %v", err)
					}
				}
			}))
			defer server.Close()

			client, err := NewClient("test-token")
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			c := github.NewClient(server.Client())
			c.BaseURL, _ = url.Parse(server.URL + "/")
			client.Client = c

			content, err := client.GetRawContent(context.Background(), tc.repo, tc.path, tc.ref)

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
	cases := []struct {
		name         string
		prMetadata   *PullRequestMetadata
		mockResponse []*github.PullRequestReview
		statusCode   int
		wantErr      bool
	}{
		{
			name: "Successful get",
			prMetadata: &PullRequestMetadata{
				Repo:   &Repository{Owner: "test-owner", Name: "test-repo"},
				Number: 1,
			},
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
			name: "GitHub API error",
			prMetadata: &PullRequestMetadata{
				Repo:   &Repository{Owner: "test-owner", Name: "test-repo"},
				Number: 1,
			},
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", tc.prMetadata.Repo.Owner, tc.prMetadata.Repo.Name, tc.prMetadata.Number)
				if r.URL.Path != expectedPath {
					t.Errorf("expected request to %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
				if tc.mockResponse != nil {
					if err := json.NewEncoder(w).Encode(tc.mockResponse); err != nil {
						t.Fatalf("failed to write mock response: %v", err)
					}
				}
			}))
			defer server.Close()

			client, err := NewClient("test-token")
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			c := github.NewClient(server.Client())
			c.BaseURL, _ = url.Parse(server.URL + "/")
			client.Client = c

			reviews, err := client.GetPullRequestReviews(context.Background(), tc.prMetadata)

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
	cases := []struct {
		name         string
		pullRequest  *github.PullRequest
		mockResponse *github.ListCheckRunsResults
		statusCode   int
		wantErr      bool
	}{
		{
			name: "Successful get",
			pullRequest: &github.PullRequest{
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
			},
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
			name: "GitHub API error",
			pullRequest: &github.PullRequest{
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
			},
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/repos/%s/%s/commits/%s/check-runs", *tc.pullRequest.Head.Repo.Owner.Login, *tc.pullRequest.Head.Repo.Name, *tc.pullRequest.Head.Ref)
				if r.URL.Path != expectedPath {
					t.Errorf("expected request to %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
				if tc.mockResponse != nil {
					if err := json.NewEncoder(w).Encode(tc.mockResponse); err != nil {
						t.Fatalf("failed to write mock response: %v", err)
					}
				}
			}))
			defer server.Close()

			client, err := NewClient("test-token")
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			c := github.NewClient(server.Client())
			c.BaseURL, _ = url.Parse(server.URL + "/")
			client.Client = c

			checkRuns, err := client.GetPullRequestCheckRuns(context.Background(), tc.pullRequest)

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
	cases := []struct {
		name         string
		repo         *Repository
		prNumber     int
		mockResponse *github.PullRequest
		statusCode   int
		wantErr      bool
	}{
		{
			name:     "Successful get",
			repo:     &Repository{Owner: "test-owner", Name: "test-repo"},
			prNumber: 1,
			mockResponse: &github.PullRequest{
				Number: github.Ptr(1),
				Title:  github.Ptr("Test PR"),
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "GitHub API error",
			repo:       &Repository{Owner: "test-owner", Name: "test-repo"},
			prNumber:   1,
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/repos/%s/%s/pulls/%d", tc.repo.Owner, tc.repo.Name, tc.prNumber)
				if r.URL.Path != expectedPath {
					t.Errorf("expected request to %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
				if tc.mockResponse != nil {
					if err := json.NewEncoder(w).Encode(tc.mockResponse); err != nil {
						t.Fatalf("failed to write mock response: %v", err)
					}
				}
			}))
			defer server.Close()

			client, err := NewClient("test-token")
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			c := github.NewClient(server.Client())
			c.BaseURL, _ = url.Parse(server.URL + "/")
			client.Client = c

			pr, err := client.GetPullRequest(context.Background(), tc.repo, tc.prNumber)

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
	cases := []struct {
		name         string
		repo         *Repository
		prNumber     int
		method       github.MergeMethod
		mockResponse *github.PullRequestMergeResult
		statusCode   int
		wantErr      bool
	}{
		{
			name:     "Successful merge",
			repo:     &Repository{Owner: "test-owner", Name: "test-repo"},
			prNumber: 1,
			method:   MergeMethodRebase,
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
			repo:       &Repository{Owner: "test-owner", Name: "test-repo"},
			prNumber:   1,
			method:     MergeMethodRebase,
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Errorf("expected PUT request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/repos/%s/%s/pulls/%d/merge", tc.repo.Owner, tc.repo.Name, tc.prNumber)
				if r.URL.Path != expectedPath {
					t.Errorf("expected request to %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
				if tc.mockResponse != nil {
					if err := json.NewEncoder(w).Encode(tc.mockResponse); err != nil {
						t.Fatalf("failed to write mock response: %v", err)
					}
				}
			}))
			defer server.Close()

			client, err := NewClient("test-token")
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			c := github.NewClient(server.Client())
			c.BaseURL, _ = url.Parse(server.URL + "/")
			client.Client = c

			result, err := client.MergePullRequest(context.Background(), tc.repo, tc.prNumber, tc.method)

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
	cases := []struct {
		name       string
		repo       *Repository
		prNumber   int
		comment    string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "Successful comment addition",
			repo:       &Repository{Owner: "test-owner", Name: "test-repo"},
			prNumber:   1,
			comment:    "test comment",
			statusCode: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:       "GitHub API error",
			repo:       &Repository{Owner: "test-owner", Name: "test-repo"},
			prNumber:   1,
			comment:    "test comment",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", tc.repo.Owner, tc.repo.Name, tc.prNumber)
				if r.URL.Path != expectedPath {
					t.Errorf("expected request to %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			client, err := NewClient("test-token")
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			c := github.NewClient(server.Client())
			c.BaseURL, _ = url.Parse(server.URL + "/")
			client.Client = c

			err = client.AddCommentToPullRequest(context.Background(), tc.repo, tc.prNumber, tc.comment)

			if (err != nil) != tc.wantErr {
				t.Errorf("AddCommentToPullRequest() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestRemoveLabelFromPullRequest(t *testing.T) {
	cases := []struct {
		name       string
		repo       *Repository
		prNumber   int
		label      string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "Successful label removal",
			repo:       &Repository{Owner: "test-owner", Name: "test-repo"},
			prNumber:   1,
			label:      "test-label",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "GitHub API error",
			repo:       &Repository{Owner: "test-owner", Name: "test-repo"},
			prNumber:   1,
			label:      "test-label",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("expected DELETE request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/repos/%s/%s/issues/%d/labels/%s", tc.repo.Owner, tc.repo.Name, tc.prNumber, tc.label)
				if r.URL.Path != expectedPath {
					t.Errorf("expected request to %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			client, err := NewClient("test-token")
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			c := github.NewClient(server.Client())
			c.BaseURL, _ = url.Parse(server.URL + "/")
			client.Client = c

			err = client.RemoveLabelFromPullRequest(context.Background(), tc.repo, tc.prNumber, tc.label)

			if (err != nil) != tc.wantErr {
				t.Errorf("RemoveLabelFromPullRequest() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestAddLabelToPullRequest(t *testing.T) {
	cases := []struct {
		name       string
		prMetadata *PullRequestMetadata
		label      string
		statusCode int
		wantErr    bool
	}{
		{
			name: "Successful label addition",
			prMetadata: &PullRequestMetadata{
				Repo:   &Repository{Owner: "test-owner", Name: "test-repo"},
				Number: 1,
			},
			label:      "test-label",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "GitHub API error",
			prMetadata: &PullRequestMetadata{
				Repo:   &Repository{Owner: "test-owner", Name: "test-repo"},
				Number: 1,
			},
			label:      "test-label",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/repos/%s/%s/issues/%d/labels", tc.prMetadata.Repo.Owner, tc.prMetadata.Repo.Name, tc.prMetadata.Number)
				if r.URL.Path != expectedPath {
					t.Errorf("expected request to %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			client, err := NewClient("test-token")
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			c := github.NewClient(server.Client())
			c.BaseURL, _ = url.Parse(server.URL + "/")
			client.Client = c

			err = client.AddLabelToPullRequest(context.Background(), tc.prMetadata, tc.label)

			if (err != nil) != tc.wantErr {
				t.Errorf("AddLabelToPullRequest() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestCreateRelease(t *testing.T) {
	cases := []struct {
		name         string
		repo         *Repository
		tag          string
		commit       string
		title        string
		description  string
		prerelease   bool
		mockResponse *github.RepositoryRelease
		statusCode   int
		wantErr      bool
	}{
		{
			name:        "Successful release creation",
			repo:        &Repository{Owner: "test-owner", Name: "test-repo"},
			tag:         "v1.0.0",
			commit:      "abcdef123",
			title:       "Release v1.0.0",
			description: "This is a test release.",
			prerelease:  false,
			mockResponse: &github.RepositoryRelease{
				ID:      github.Ptr(int64(1)),
				TagName: github.Ptr("v1.0.0"),
			},
			statusCode: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:        "GitHub API error",
			repo:        &Repository{Owner: "test-owner", Name: "test-repo"},
			tag:         "v1.0.0",
			commit:      "abcdef123",
			title:       "Release v1.0.0",
			description: "This is a test release.",
			prerelease:  false,
			statusCode:  http.StatusInternalServerError,
			wantErr:     true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				if r.URL.Path != fmt.Sprintf("/repos/%s/%s/releases", tc.repo.Owner, tc.repo.Name) {
					t.Errorf("expected request to /repos/%s/%s/releases, got %s", tc.repo.Owner, tc.repo.Name, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
				if tc.mockResponse != nil {
					if err := json.NewEncoder(w).Encode(tc.mockResponse); err != nil {
						t.Fatalf("failed to write mock response: %v", err)
					}
				}
			}))
			defer server.Close()

			client, err := NewClient("test-token")
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			c := github.NewClient(server.Client())
			c.BaseURL, _ = url.Parse(server.URL + "/")
			client.Client = c

			release, err := client.CreateRelease(context.Background(), tc.repo, tc.tag, tc.commit, tc.title, tc.description, tc.prerelease)

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
