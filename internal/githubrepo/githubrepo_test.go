// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package githubrepo

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGetPullRequestReviews(t *testing.T) {
	oldPageSize := pageSize
	pageSize = 1
	t.Cleanup(func() { pageSize = oldPageSize })

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			fmt.Fprintln(w, `[{"id": 2, "body": "LGTM"}]`)
		} else {
			w.Header().Set("Link", fmt.Sprintf("<%s/api/v3/repos/o/r/pulls/1/reviews?page=2>; rel=\"next\"", server.URL))
			fmt.Fprintln(w, `[{"id": 1, "body": "Needs changes"}]`)
		}
	}))
	defer server.Close()

	client, err := NewClient("token")
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}
	serverURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatalf("Failed to parse server URL: %v", err)
	}
	client.BaseURL = serverURL

	prMetadata := &PullRequestMetadata{
		Repo:   &Repository{Owner: "o", Name: "r"},
		Number: 1,
	}
	reviews, err := client.GetPullRequestReviews(context.Background(), prMetadata)
	if err != nil {
		t.Fatalf("GetPullRequestReviews() failed: %v", err)
	}

	if len(reviews) != 2 {
		t.Errorf("Expected 2 reviews, got %d", len(reviews))
	}
}

func TestGetDiffCommits(t *testing.T) {
	oldPageSize := pageSize
	pageSize = 1
	t.Cleanup(func() { pageSize = oldPageSize })

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			fmt.Fprintln(w, `{"commits": [{"sha": "def"}]}`)
		} else {
			w.Header().Set("Link", fmt.Sprintf("<%s/api/v3/repos/o/r/compare/a...b?page=2>; rel=\"next\"", server.URL))
			fmt.Fprintln(w, `{"commits": [{"sha": "abc"}]}`)
		}
	}))
	defer server.Close()

	client, err := NewClient("token")
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}
	serverURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatalf("Failed to parse server URL: %v", err)
	}
	client.BaseURL = serverURL

	repo := &Repository{Owner: "o", Name: "r"}
	commits, err := client.GetDiffCommits(context.Background(), repo, "a", "b")
	if err != nil {
		t.Fatalf("GetDiffCommits() failed: %v", err)
	}

	if len(commits) != 2 {
		t.Errorf("Expected 2 commits, got %d", len(commits))
	}
}

func TestGetCommit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"sha": "abc"}`)
	}))
	defer server.Close()

	client, err := NewClient("token")
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}
	serverURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatalf("Failed to parse server URL: %v", err)
	}
	client.BaseURL = serverURL

	repo := &Repository{Owner: "o", Name: "r"}
	commit, err := client.GetCommit(context.Background(), repo, "abc")
	if err != nil {
		t.Fatalf("GetCommit() failed: %v", err)
	}

	if commit.GetSHA() != "abc" {
		t.Errorf("Expected commit SHA 'abc', got '%s'", commit.GetSHA())
	}
}
