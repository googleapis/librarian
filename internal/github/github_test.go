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
	"time"

	"github.com/go-git/go-git/v5"
	gogitConfig "github.com/go-git/go-git/v5/config"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v69/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

func TestToken(t *testing.T) {
	t.Parallel()
	want := "fake-token"
	repo := &Repository{Owner: "owner", Name: "repo"}
	client, err := NewClient(want, repo)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if got := client.Token(); got != want {
		t.Errorf("Token() = %q, want %q", got, want)
	}
}

func TestGetRawContent(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name           string
		handler        http.HandlerFunc
		wantContent    []byte
		wantErr        bool
		wantErrSubstr  string
		wantHTTPMethod string
		wantURLPath    string
	}{
		{
			name: "Success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "contents/path/to") {
					fmt.Fprintf(w, `[{"content":"file content", "name":"file", "download_url": "http://%s/download"}]`, r.Host)
				} else if strings.HasSuffix(r.URL.Path, "download") {
					fmt.Fprint(w, "file content")
				} else {
					t.Error("unexpected URL path")
					w.WriteHeader(http.StatusNotFound)
				}
			},
			wantContent:    []byte("file content"),
			wantErr:        false,
			wantHTTPMethod: http.MethodGet,
			wantURLPath:    "/repos/owner/repo/contents/path/to/file",
		},
		{
			name: "Not Found",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr:        true,
			wantErrSubstr:  "404",
			wantHTTPMethod: http.MethodGet,
			wantURLPath:    "/repos/owner/repo/contents/path/to/file",
		},
		{
			name: "Internal Server Error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr:        true,
			wantErrSubstr:  "500",
			wantHTTPMethod: http.MethodGet,
			wantURLPath:    "/repos/owner/repo/contents/path/to/file",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				test.handler(w, r)
			}))
			defer server.Close()

			repo := &Repository{Owner: "owner", Name: "repo"}
			client, err := newClientWithHTTP("fake-token", repo, server.Client())
			if err != nil {
				t.Fatalf("newClientWithHTTP() error = %v", err)
			}

			client.BaseURL, _ = url.Parse(server.URL + "/")
			content, err := client.GetRawContent(context.Background(), "path/to/file", "main")

			if test.wantErr {
				if err == nil {
					t.Errorf("GetRawContent() err = nil, want error containing %q", test.wantErrSubstr)
				} else if !strings.Contains(err.Error(), test.wantErrSubstr) {
					t.Errorf("GetRawContent() err = %v, want error containing %q", err, test.wantErrSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("GetRawContent() err = %v, want nil", err)
				}
				if diff := cmp.Diff(test.wantContent, content); diff != "" {
					t.Errorf("GetRawContent() content mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// newTestGitRepo creates a new git repository in a temporary directory with the given remotes.
func newTestGitRepo(t *testing.T, remotes map[string][]string) *gitrepo.LocalRepository {
	t.Helper()
	dir := t.TempDir()

	r, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("git.PlainInit failed: %v", err)
	}

	for name, urls := range remotes {
		_, err := r.CreateRemote(&gogitConfig.RemoteConfig{
			Name: name,
			URLs: urls,
		})
		if err != nil {
			t.Fatalf("CreateRemote failed: %v", err)
		}
	}

	repo, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{Dir: dir})
	if err != nil {
		t.Fatalf("gitrepo.NewRepository failed: %v", err)
	}
	return repo
}

func TestFetchGitHubRepoFromRemote(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		remotes       map[string][]string
		wantRepo      *Repository
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name: "origin is a GitHub remote",
			remotes: map[string][]string{
				"origin": {"https://github.com/owner/repo.git"},
			},
			wantRepo: &Repository{Owner: "owner", Name: "repo"},
		},
		{
			name:          "No remotes",
			remotes:       map[string][]string{},
			wantErr:       true,
			wantErrSubstr: "could not find an 'origin' remote",
		},
		{
			name: "origin is not a GitHub remote",
			remotes: map[string][]string{
				"origin": {"https://gitlab.com/owner/repo.git"},
			},
			wantErr:       true,
			wantErrSubstr: "is not a GitHub remote",
		},
		{
			name: "upstream is GitHub, but no origin",
			remotes: map[string][]string{
				"gitlab":   {"https://gitlab.com/owner/repo.git"},
				"upstream": {"https://github.com/gh-owner/gh-repo.git"},
			},
			wantErr:       true,
			wantErrSubstr: "could not find an 'origin' remote",
		},
		{
			name: "origin and upstream are GitHub remotes, should use origin",
			remotes: map[string][]string{
				"origin":   {"https://github.com/owner/repo.git"},
				"upstream": {"https://github.com/owner2/repo2.git"},
			},
			wantRepo: &Repository{Owner: "owner", Name: "repo"},
		},
		{
			name: "origin is not GitHub, but upstream is",
			remotes: map[string][]string{
				"origin":   {"https://gitlab.com/owner/repo.git"},
				"upstream": {"https://github.com/gh-owner/gh-repo.git"},
			},
			wantErr:       true,
			wantErrSubstr: "is not a GitHub remote",
		},
		{
			name: "origin has multiple URLs, first is GitHub",
			remotes: map[string][]string{
				"origin": {"https://github.com/owner/repo.git", "https://gitlab.com/owner/repo.git"},
			},
			wantRepo: &Repository{Owner: "owner", Name: "repo"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repo := newTestGitRepo(t, test.remotes)

			got, err := FetchGitHubRepoFromRemote(repo)

			if test.wantErr {
				if err == nil {
					t.Errorf("FetchGitHubRepoFromRemote() err = nil, want error containing %q", test.wantErrSubstr)
				} else if !strings.Contains(err.Error(), test.wantErrSubstr) {
					t.Errorf("FetchGitHubRepoFromRemote() err = %v, want error containing %q", err, test.wantErrSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("FetchGitHubRepoFromRemote() err = %v, want nil", err)
				}
				if diff := cmp.Diff(test.wantRepo, got); diff != "" {
					t.Errorf("FetchGitHubRepoFromRemote() repo mismatch (-want +got): %s", diff)
				}
			}
		})
	}
}

func TestParseURL(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		remoteURL     string
		wantRepo      *Repository
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:      "Valid HTTPS URL",
			remoteURL: "https://github.com/owner/repo.git",
			wantRepo:  &Repository{Owner: "owner", Name: "repo"},
			wantErr:   false,
		},
		{
			name:      "Valid HTTPS URL without .git",
			remoteURL: "https://github.com/owner/repo",
			wantRepo:  &Repository{Owner: "owner", Name: "repo"},
			wantErr:   false,
		},
		{
			name:          "Invalid URL scheme",
			remoteURL:     "http://github.com/owner/repo.git",
			wantErr:       true,
			wantErrSubstr: "not a GitHub remote",
		},
		{
			name:      "URL with extra path components",
			remoteURL: "https://github.com/owner/repo/pulls",
			wantRepo:  &Repository{Owner: "owner", Name: "repo"},
			wantErr:   false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repo, err := ParseRemote(test.remoteURL)

			if test.wantErr {
				if err == nil {
					t.Errorf("ParseURL() err = nil, want error containing %q", test.wantErrSubstr)
				} else if !strings.Contains(err.Error(), test.wantErrSubstr) {
					t.Errorf("ParseURL() err = %v, want error containing %q", err, test.wantErrSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("ParseURL() err = %v, want nil", err)
				}
				if diff := cmp.Diff(test.wantRepo, repo); diff != "" {
					t.Errorf("ParseURL() repo mismatch (-want +got): %s", diff)
				}
			}
		})
	}
}

func TestParseSSHRemote(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		remote        string
		wantRepo      *Repository
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:     "Valid SSH URL with .git",
			remote:   "git@github.com:owner/repo.git",
			wantRepo: &Repository{Owner: "owner", Name: "repo"},
		},
		{
			name:     "Valid SSH URL without .git",
			remote:   "git@github.com:owner/repo",
			wantRepo: &Repository{Owner: "owner", Name: "repo"},
		},
		{
			name:          "Invalid remote, no git@ prefix",
			remote:        "https://github.com/owner/repo.git",
			wantErr:       true,
			wantErrSubstr: "not a GitHub remote",
		},
		{
			name:          "Invalid remote, no colon",
			remote:        "git@github.com-owner/repo.git",
			wantErr:       true,
			wantErrSubstr: "not a GitHub remote",
		},
		{
			name:          "Invalid remote, no slash",
			remote:        "git@github.com:owner-repo.git",
			wantErr:       true,
			wantErrSubstr: "not a GitHub remote",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repo, err := parseSSHRemote(test.remote)
			if test.wantErr {
				if err == nil {
					t.Errorf("ParseSSHRemote() err = nil, want error containing %q", test.wantErrSubstr)
				} else if !strings.Contains(err.Error(), test.wantErrSubstr) {
					t.Errorf("ParseSSHRemote() err = %v, want error containing %q", err, test.wantErrSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("ParseSSHRemote() err = %v, want nil", err)
				}
				if diff := cmp.Diff(test.wantRepo, repo); diff != "" {
					t.Errorf("ParseSSHRemote() repo mismatch (-want +got): %s", diff)
				}
			}
		})
	}
}

func TestCreatePullRequest(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		remoteBranch  string
		remoteBase    string
		title         string
		body          string
		handler       http.HandlerFunc
		wantMetadata  *PullRequestMetadata
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:         "Success with provided body",
			remoteBranch: "feature-branch",
			remoteBase:   "base-branch",
			title:        "New Feature",
			body:         "This is a new feature.",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("unexpected method: got %s, want %s", r.Method, http.MethodPost)
				}
				if r.URL.Path != "/repos/owner/repo/pulls" {
					t.Errorf("unexpected path: got %s, want %s", r.URL.Path, "/repos/owner/repo/pulls")
				}
				var newPR github.NewPullRequest
				if err := json.NewDecoder(r.Body).Decode(&newPR); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}
				if *newPR.Title != "New Feature" {
					t.Errorf("unexpected title: got %q, want %q", *newPR.Title, "New Feature")
				}
				if *newPR.Body != "This is a new feature." {
					t.Errorf("unexpected body: got %q, want %q", *newPR.Body, "This is a new feature.")
				}
				if *newPR.Head != "feature-branch" {
					t.Errorf("unexpected head: got %q, want %q", *newPR.Head, "feature-branch")
				}
				if *newPR.Base != "base-branch" {
					t.Errorf("unexpected base: got %q, want %q", *newPR.Base, "base-branch")
				}
				fmt.Fprint(w, `{"number": 1, "html_url": "https://github.com/owner/repo/pull/1"}`)
			},
			wantMetadata: &PullRequestMetadata{Repo: &Repository{Owner: "owner", Name: "repo"}, Number: 1},
		},
		{
			name:         "Success with empty body",
			remoteBranch: "another-branch",
			remoteBase:   "main",
			title:        "Another PR",
			body:         "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				var newPR github.NewPullRequest
				if err := json.NewDecoder(r.Body).Decode(&newPR); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}
				expectedBody := "Regenerated all changed APIs. See individual commits for details."
				if *newPR.Body != expectedBody {
					t.Errorf("unexpected body: got %q, want %q", *newPR.Body, expectedBody)
				}
				fmt.Fprint(w, `{"number": 1, "html_url": "https://github.com/owner/repo/pull/1"}`)
			},
			wantMetadata: &PullRequestMetadata{Repo: &Repository{Owner: "owner", Name: "repo"}, Number: 1},
		},
		{
			name:          "GitHub API error",
			remoteBranch:  "error-branch",
			remoteBase:    "main",
			title:         "Error PR",
			body:          "This will fail.",
			handler:       func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
			wantErr:       true,
			wantErrSubstr: "500",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(test.handler)
			defer server.Close()

			repo := &Repository{Owner: "owner", Name: "repo"}
			client, err := newClientWithHTTP("fake-token", repo, server.Client())
			if err != nil {
				t.Fatalf("newClientWithHTTP() error = %v", err)
			}
			client.BaseURL, _ = url.Parse(server.URL + "/")

			metadata, err := client.CreatePullRequest(context.Background(), repo, test.remoteBranch, test.remoteBase, test.title, test.body)

			if test.wantErr {
				if err == nil {
					t.Errorf("CreatePullRequest() err = nil, want error containing %q", test.wantErrSubstr)
				} else if !strings.Contains(err.Error(), test.wantErrSubstr) {
					t.Errorf("CreatePullRequest() err = %v, want error containing %q", err, test.wantErrSubstr)
				}
			} else if err != nil {
				t.Errorf("CreatePullRequest() err = %v, want nil", err)
			}

			if diff := cmp.Diff(test.wantMetadata, metadata); diff != "" {
				t.Errorf("CreatePullRequest() metadata mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAddLabelsToIssue(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		handler       http.HandlerFunc
		issueNum      int
		labels        []string
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name: "add labels to an issue",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("unexpected method: got %s, want %s", r.Method, http.MethodPost)
				}
				wantPath := "/repos/owner/repo/issues/7/labels"
				if r.URL.Path != wantPath {
					t.Errorf("unexpected path: got %s, want %s", r.URL.Path, wantPath)
				}
				var labels []string
				if err := json.NewDecoder(r.Body).Decode(&labels); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}
				expectedBody := []string{"new-label", "another-label"}
				if strings.Join(labels, ",") != strings.Join(expectedBody, ",") {
					t.Errorf("unexpected body: got %q, want %q", labels, expectedBody)
				}
			},
			issueNum: 7,
			labels:   []string{"new-label", "another-label"},
		},
		{
			name:          "GitHub API error",
			handler:       func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
			wantErr:       true,
			wantErrSubstr: "500",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(test.handler)
			defer server.Close()

			repo := &Repository{Owner: "owner", Name: "repo"}
			client, err := newClientWithHTTP("fake-token", repo, server.Client())
			if err != nil {
				t.Fatalf("newClientWithHTTP() error = %v", err)
			}
			client.BaseURL, _ = url.Parse(server.URL + "/")

			err = client.AddLabelsToIssue(context.Background(), repo, test.issueNum, test.labels)

			if test.wantErr {
				if err == nil {
					t.Errorf("AddLabelsToIssue() should return an error")
				}
				if !strings.Contains(err.Error(), test.wantErrSubstr) {
					t.Errorf("AddLabelsToIssue() err = %v, want error containing %q", err, test.wantErrSubstr)
				}

				return
			}

			if err != nil {
				t.Errorf("AddLabelsToIssue() err = %v, want nil", err)
			}
		})
	}
}

func TestGetLabels(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		handler       http.HandlerFunc
		issueNum      int
		wantLabels    []string
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name: "get labels from an issue",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("unexpected method: got %s, want %s", r.Method, http.MethodGet)
				}
				wantPath := "/repos/owner/repo/issues/7/labels"
				if r.URL.Path != wantPath {
					t.Errorf("unexpected path: got %s, want %s", r.URL.Path, wantPath)
				}
				fmt.Fprint(w, `[{"name": "label1"}, {"name": "label2"}]`)
			},
			issueNum:   7,
			wantLabels: []string{"label1", "label2"},
		},
		{
			name: "get labels with pagination",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("page") == "2" {
					fmt.Fprint(w, `[{"name": "label3"}]`)
					return
				}
				w.Header().Set("Link", `<http://`+r.Host+`/repos/owner/repo/issues/7/labels?page=2>; rel="next"`)
				fmt.Fprint(w, `[{"name": "label1"}, {"name": "label2"}]`)
			},
			issueNum:   7,
			wantLabels: []string{"label1", "label2", "label3"},
		},
		{
			name:          "GitHub API error",
			handler:       func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
			issueNum:      7,
			wantErr:       true,
			wantErrSubstr: "500",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(test.handler)
			defer server.Close()

			repo := &Repository{Owner: "owner", Name: "repo"}
			client, err := newClientWithHTTP("fake-token", repo, server.Client())
			if err != nil {
				t.Fatalf("newClientWithHTTP() error = %v", err)
			}
			client.BaseURL, _ = url.Parse(server.URL + "/")

			gotLabels, err := client.GetLabels(context.Background(), test.issueNum)

			if test.wantErr {
				if err == nil {
					t.Errorf("GetLabels() should return an error")
				}
				if !strings.Contains(err.Error(), test.wantErrSubstr) {
					t.Errorf("GetLabels() err = %v, want error containing %q", err, test.wantErrSubstr)
				}
				return
			}

			if err != nil {
				t.Errorf("GetLabels() err = %v, want nil", err)
			}
			if diff := cmp.Diff(test.wantLabels, gotLabels); diff != "" {
				t.Errorf("GetLabels() labels mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReplaceLabels(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		handler       http.HandlerFunc
		issueNum      int
		labels        []string
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name: "replace labels for an issue",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Errorf("unexpected method: got %s, want %s", r.Method, http.MethodPut)
				}
				wantPath := "/repos/owner/repo/issues/7/labels"
				if r.URL.Path != wantPath {
					t.Errorf("unexpected path: got %s, want %s", r.URL.Path, wantPath)
				}
				var labels []string
				if err := json.NewDecoder(r.Body).Decode(&labels); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}
				expectedBody := []string{"new-label", "another-label"}
				if diff := cmp.Diff(expectedBody, labels); diff != "" {
					t.Errorf("ReplaceLabels() request body mismatch (-want +got):\n%s", diff)
				}
				fmt.Fprint(w, `[]`)
			},
			issueNum: 7,
			labels:   []string{"new-label", "another-label"},
		},
		{
			name:          "GitHub API error",
			handler:       func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
			issueNum:      7,
			labels:        []string{"some-label"},
			wantErr:       true,
			wantErrSubstr: "500",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(test.handler)
			defer server.Close()

			repo := &Repository{Owner: "owner", Name: "repo"}
			client, err := newClientWithHTTP("fake-token", repo, server.Client())
			if err != nil {
				t.Fatalf("newClientWithHTTP() error = %v", err)
			}
			client.BaseURL, _ = url.Parse(server.URL + "/")

			err = client.ReplaceLabels(context.Background(), test.issueNum, test.labels)

			if test.wantErr {
				if err == nil {
					t.Errorf("ReplaceLabels() should return an error")
				}
				if !strings.Contains(err.Error(), test.wantErrSubstr) {
					t.Errorf("ReplaceLabels() err = %v, want error containing %q", err, test.wantErrSubstr)
				}
				return
			}

			if err != nil {
				t.Errorf("ReplaceLabels() err = %v, want nil", err)
			}
		})
	}
}

func TestSearchPullRequests(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		query         string
		handler       http.HandlerFunc
		wantPRs       []*PullRequest
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:  "Success with single page",
			query: "is:pr is:open author:app/dependabot",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/search/issues") {
					if r.URL.Query().Get("q") != "is:pr is:open author:app/dependabot" {
						t.Errorf("unexpected query: got %q", r.URL.Query().Get("q"))
					}
					fmt.Fprint(w, `{"items": [{"number": 1, "pull_request": {}}]}`)
				} else if strings.HasPrefix(r.URL.Path, "/repos/owner/repo/pulls/1") {
					fmt.Fprint(w, `{"number": 1, "title": "PR 1"}`)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			},
			wantPRs: []*PullRequest{
				{Number: github.Ptr(1), Title: github.Ptr("PR 1")},
			},
		},
		{
			name:  "Success with multiple pages",
			query: "is:pr is:open",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/search/issues") {
					if r.URL.Query().Get("page") == "2" {
						fmt.Fprint(w, `{"items": [{"number": 2, "pull_request": {}}]}`)
					} else {
						w.Header().Set("Link", `<http://`+r.Host+`/search/issues?page=2>; rel="next"`)
						fmt.Fprint(w, `{"items": [{"number": 1, "pull_request": {}}]}`)
					}
				} else if strings.HasPrefix(r.URL.Path, "/repos/owner/repo/pulls/1") {
					fmt.Fprint(w, `{"number": 1, "title": "PR 1"}`)
				} else if strings.HasPrefix(r.URL.Path, "/repos/owner/repo/pulls/2") {
					fmt.Fprint(w, `{"number": 2, "title": "PR 2"}`)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			},
			wantPRs: []*PullRequest{
				{Number: github.Ptr(1), Title: github.Ptr("PR 1")},
				{Number: github.Ptr(2), Title: github.Ptr("PR 2")},
			},
		},
		{
			name:  "Search API error",
			query: "is:pr",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/search/issues") {
					w.WriteHeader(http.StatusInternalServerError)
				}
			},
			wantErr:       true,
			wantErrSubstr: "500",
		},
		{
			name:  "Get PR API error",
			query: "is:pr",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/search/issues") {
					fmt.Fprint(w, `{"items": [{"number": 1, "pull_request": {}}]}`)
				} else if strings.HasPrefix(r.URL.Path, "/repos/owner/repo/pulls/1") {
					w.WriteHeader(http.StatusInternalServerError)
				}
			},
			wantErr:       true,
			wantErrSubstr: "500",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(test.handler)
			defer server.Close()

			repo := &Repository{Owner: "owner", Name: "repo"}
			client, err := newClientWithHTTP("fake-token", repo, server.Client())
			if err != nil {
				t.Fatalf("newClientWithHTTP() error = %v", err)
			}
			client.BaseURL, _ = url.Parse(server.URL + "/")

			prs, err := client.SearchPullRequests(context.Background(), test.query)

			if test.wantErr {
				if err == nil {
					t.Errorf("SearchPullRequests() err = nil, want error containing %q", test.wantErrSubstr)
				} else if !strings.Contains(err.Error(), test.wantErrSubstr) {
					t.Errorf("SearchPullRequests() err = %v, want error containing %q", err, test.wantErrSubstr)
				}
			} else if err != nil {
				t.Errorf("SearchPullRequests() err = %v, want nil", err)
			}

			if diff := cmp.Diff(test.wantPRs, prs); diff != "" {
				t.Errorf("SearchPullRequests() prs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetPullRequest(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		number        int
		handler       http.HandlerFunc
		wantPR        *PullRequest
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:   "Success",
			number: 42,
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("unexpected method: got %s, want %s", r.Method, http.MethodGet)
				}
				wantPath := "/repos/owner/repo/pulls/42"
				if r.URL.Path != wantPath {
					t.Errorf("unexpected path: got %s, want %s", r.URL.Path, wantPath)
				}
				fmt.Fprint(w, `{"number": 42, "title": "The Answer"}`)
			},
			wantPR: &PullRequest{Number: github.Ptr(42), Title: github.Ptr("The Answer")},
		},
		{
			name:   "Not Found",
			number: 43,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr:       true,
			wantErrSubstr: "404",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(test.handler)
			defer server.Close()

			repo := &Repository{Owner: "owner", Name: "repo"}
			client, err := newClientWithHTTP("fake-token", repo, server.Client())
			if err != nil {
				t.Fatalf("newClientWithHTTP() error = %v", err)
			}
			client.BaseURL, _ = url.Parse(server.URL + "/")

			pr, err := client.GetPullRequest(context.Background(), test.number)

			if test.wantErr {
				if err == nil {
					t.Errorf("GetPullRequest() err = nil, want error containing %q", test.wantErrSubstr)
				} else if !strings.Contains(err.Error(), test.wantErrSubstr) {
					t.Errorf("GetPullRequest() err = %v, want error containing %q", err, test.wantErrSubstr)
				}
			} else if err != nil {
				t.Errorf("GetPullRequest() err = %v, want nil", err)
			}

			if diff := cmp.Diff(test.wantPR, pr); diff != "" {
				t.Errorf("GetPullRequest() pr mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateRelease(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		tagName       string
		releaseName   string
		body          string
		commitish     string
		handler       http.HandlerFunc
		wantRelease   *github.RepositoryRelease
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:        "Success",
			tagName:     "v1.0.0",
			releaseName: "Version 1.0.0",
			body:        "Initial release",
			commitish:   "main",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("unexpected method: got %s, want %s", r.Method, http.MethodPost)
				}
				wantPath := "/repos/owner/repo/releases"
				if r.URL.Path != wantPath {
					t.Errorf("unexpected path: got %s, want %s", r.URL.Path, wantPath)
				}
				var newRelease github.RepositoryRelease
				if err := json.NewDecoder(r.Body).Decode(&newRelease); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}
				if *newRelease.TagName != "v1.0.0" {
					t.Errorf("unexpected tag name: got %q, want %q", *newRelease.TagName, "v1.0.0")
				}
				fmt.Fprint(w, `{"tag_name": "v1.0.0", "name": "Version 1.0.0"}`)
			},
			wantRelease: &github.RepositoryRelease{TagName: github.Ptr("v1.0.0"), Name: github.Ptr("Version 1.0.0")},
		},
		{
			name:          "API Error",
			tagName:       "v1.0.0",
			releaseName:   "Version 1.0.0",
			body:          "Initial release",
			commitish:     "main",
			handler:       func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
			wantErr:       true,
			wantErrSubstr: "500",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(test.handler)
			defer server.Close()

			repo := &Repository{Owner: "owner", Name: "repo"}
			client, err := newClientWithHTTP("fake-token", repo, server.Client())
			if err != nil {
				t.Fatalf("newClientWithHTTP() error = %v", err)
			}
			client.BaseURL, _ = url.Parse(server.URL + "/")

			release, err := client.CreateRelease(context.Background(), test.tagName, test.releaseName, test.body, test.commitish)

			if test.wantErr {
				if err == nil {
					t.Errorf("CreateRelease() err = nil, want error containing %q", test.wantErrSubstr)
				} else if !strings.Contains(err.Error(), test.wantErrSubstr) {
					t.Errorf("CreateRelease() err = %v, want error containing %q", err, test.wantErrSubstr)
				}
			} else if err != nil {
				t.Errorf("CreateRelease() err = %v, want nil", err)
			}

			if diff := cmp.Diff(test.wantRelease, release); diff != "" {
				t.Errorf("CreateRelease() release mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateIssueComment(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		number        int
		body          string
		handler       http.HandlerFunc
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:   "Success",
			number: 123,
			body:   "This is a comment.",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("unexpected method: got %s, want %s", r.Method, http.MethodPost)
				}
				wantPath := "/repos/owner/repo/issues/123/comments"
				if r.URL.Path != wantPath {
					t.Errorf("unexpected path: got %s, want %s", r.URL.Path, wantPath)
				}
				var comment github.IssueComment
				if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}
				if *comment.Body != "This is a comment." {
					t.Errorf("unexpected body: got %q, want %q", *comment.Body, "This is a comment.")
				}
				w.WriteHeader(http.StatusCreated)
			},
		},
		{
			name:          "API Error",
			number:        123,
			body:          "This is a comment.",
			handler:       func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
			wantErr:       true,
			wantErrSubstr: "500",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(test.handler)
			defer server.Close()

			repo := &Repository{Owner: "owner", Name: "repo"}
			client, err := newClientWithHTTP("fake-token", repo, server.Client())
			if err != nil {
				t.Fatalf("newClientWithHTTP() error = %v", err)
			}
			client.BaseURL, _ = url.Parse(server.URL + "/")

			err = client.CreateIssueComment(context.Background(), test.number, test.body)

			if test.wantErr {
				if err == nil {
					t.Errorf("CreateComment() err = nil, want error containing %q", test.wantErrSubstr)
				} else if !strings.Contains(err.Error(), test.wantErrSubstr) {
					t.Errorf("CreateComment() err = %v, want error containing %q", err, test.wantErrSubstr)
				}
			} else if err != nil {
				t.Errorf("CreateComment() err = %v, want nil", err)
			}
		})
	}
}

func TestFindMergedPullRequestsWithPendingReleaseLabel(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		handler       http.HandlerFunc
		wantPRs       []*PullRequest
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name: "Success with single page",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("state") != "closed" {
					t.Errorf("unexpected state: got %q", r.URL.Query().Get("state"))
				}
				pr0 := github.PullRequest{Number: github.Ptr(0), HTMLURL: github.Ptr("https://github.com/owner/repo/pull/0"), MergedAt: &github.Timestamp{Time: time.Date(2025, time.September, 5, 20, 44, 59, 0, time.UTC)}, Labels: []*github.Label{{Name: github.Ptr("release:pending")}}}
				pr1 := github.PullRequest{Number: github.Ptr(1), Labels: []*github.Label{{Name: github.Ptr("release:pending")}}}
				pr2 := github.PullRequest{Number: github.Ptr(2), Labels: []*github.Label{{Name: github.Ptr("other-label")}}}
				pr3 := github.PullRequest{Number: github.Ptr(3), HTMLURL: github.Ptr("https://github.com/owner/repo/pull/3"), ClosedAt: &github.Timestamp{Time: time.Date(2025, time.September, 5, 20, 44, 59, 0, time.UTC)}, Labels: []*github.Label{{Name: github.Ptr("release:pending")}}}
				prs := []*github.PullRequest{&pr0, &pr1, &pr2, &pr3}
				b, err := json.Marshal(prs)
				if err != nil {
					t.Fatalf("json.Marshal() failed: %v", err)
				}
				fmt.Fprint(w, string(b))
			},
			wantPRs: []*PullRequest{
				{Number: github.Ptr(0), HTMLURL: github.Ptr("https://github.com/owner/repo/pull/0"), MergedAt: &github.Timestamp{Time: time.Date(2025, time.September, 5, 20, 44, 59, 0, time.UTC)}, Labels: []*github.Label{{Name: github.Ptr("release:pending")}}},
			},
		},
		{
			name: "API error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr:       true,
			wantErrSubstr: "500",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(test.handler)
			defer server.Close()

			repo := &Repository{Owner: "owner", Name: "repo"}
			client, err := newClientWithHTTP("fake-token", repo, server.Client())
			if err != nil {
				t.Fatalf("newClientWithHTTP() error = %v", err)
			}
			client.BaseURL, _ = url.Parse(server.URL + "/")

			prs, err := client.FindMergedPullRequestsWithPendingReleaseLabel(context.Background(), "owner", "repo")

			if test.wantErr {
				if err == nil {
					t.Errorf("FindMergedPullRequestsWithPendingReleaseLabel() err = nil, want error containing %q", test.wantErrSubstr)
				} else if !strings.Contains(err.Error(), test.wantErrSubstr) {
					t.Errorf("FindMergedPullRequestsWithPendingReleaseLabel() err = %v, want error containing %q", err, test.wantErrSubstr)
				}
			} else if err != nil {
				t.Errorf("FindMergedPullRequestsWithPendingReleaseLabel() err = %v, want nil", err)
			}

			if diff := cmp.Diff(test.wantPRs, prs); diff != "" {
				t.Errorf("FindMergedPullRequestsWithPendingReleaseLabel() prs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
