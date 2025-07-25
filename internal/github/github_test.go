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
func newTestGitRepo(t *testing.T, remotes map[string][]string) *gitrepo.Repository {
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
			name: "Single GitHub remote",
			remotes: map[string][]string{
				"origin": {"https://github.com/owner/repo.git"},
			},
			wantRepo: &Repository{Owner: "owner", Name: "repo"},
		},
		{
			name:          "No remotes",
			remotes:       map[string][]string{},
			wantErr:       true,
			wantErrSubstr: "no GitHub remotes found",
		},
		{
			name: "No GitHub remotes",
			remotes: map[string][]string{
				"origin": {"https://gitlab.com/owner/repo.git"},
			},
			wantErr:       true,
			wantErrSubstr: "no GitHub remotes found",
		},
		{
			name: "Multiple remotes, one is GitHub",
			remotes: map[string][]string{
				"gitlab":   {"https://gitlab.com/owner/repo.git"},
				"upstream": {"https://github.com/gh-owner/gh-repo.git"},
			},
			wantRepo: &Repository{Owner: "gh-owner", Name: "gh-repo"},
		},
		{
			name: "Multiple GitHub remotes",
			remotes: map[string][]string{
				"origin":   {"https://github.com/owner/repo.git"},
				"upstream": {"https://github.com/owner2/repo2.git"},
			},
			wantErr:       true,
			wantErrSubstr: "can only determine the GitHub repo with a single matching remote",
		},
		{
			name: "Remote with multiple URLs, first is not GitHub",
			remotes: map[string][]string{
				"origin": {"https://gitlab.com/owner/repo.git", "https://github.com/owner/repo.git"},
			},
			wantErr:       true,
			wantErrSubstr: "no GitHub remotes found",
		},
		{
			name: "Remote with multiple URLs, first is GitHub",
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
			repo, err := ParseURL(test.remoteURL)

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

func TestCreatePullRequest(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		remoteBranch  string
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
				if *newPR.Base != "main" {
					t.Errorf("unexpected base: got %q, want %q", *newPR.Base, "main")
				}
				fmt.Fprint(w, `{"number": 1, "html_url": "https://github.com/owner/repo/pull/1"}`)
			},
			wantMetadata: &PullRequestMetadata{Repo: &Repository{Owner: "owner", Name: "repo"}, Number: 1},
		},
		{
			name:         "Success with empty body",
			remoteBranch: "another-branch",
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

			metadata, err := client.CreatePullRequest(context.Background(), repo, test.remoteBranch, test.title, test.body)

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
