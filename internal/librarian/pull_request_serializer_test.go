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
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/github"
)

func TestNewGistOverflowHandler(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name               string
		client             *mockGitHubClient
		maxContentSize     int
		wantMaxContentSize int
	}{
		{
			name:               "default configs",
			client:             &mockGitHubClient{},
			wantMaxContentSize: maxPullRequestBodySize,
		},
		{
			name:               "custom length configs",
			client:             &mockGitHubClient{},
			maxContentSize:     1234,
			wantMaxContentSize: 1234,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := NewGistOverflowHandler(test.client, test.maxContentSize)
			if err != nil {
				t.Fatalf("unexpected error in NewGistOverflowHandler() %v", err)
			}

			if diff := cmp.Diff(test.wantMaxContentSize, got.maxContentSize); diff != "" {
				t.Errorf("%s: NewGistOverflowHandler() maxContentSize mismatch (-want +got):%s", test.name, diff)
			}
		})
	}
}

func TestSavePullRequestBody(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name                string
		client              *mockGitHubClient
		maxContentSize      int
		content             string
		want                string
		wantErr             bool
		wantErrMsg          string
		wantCreateGistCalls int
	}{
		{
			name:    "short content",
			client:  &mockGitHubClient{},
			content: "some-content",
			want:    "some-content",
		},
		{
			name: "content too long content",
			client: &mockGitHubClient{
				createdGist: &github.Gist{
					ID:  "abcd1234",
					Url: "https://gist.github.com/some-user/abcd1234",
				},
			},
			content:             "super long content",
			maxContentSize:      5, // force overflow handling
			want:                "See full release notes at: https://gist.github.com/some-user/abcd1234",
			wantCreateGistCalls: 1,
		},
		{
			name: "content too long content, error creating gist",
			client: &mockGitHubClient{
				createGistError: fmt.Errorf("some create gist error"),
				createdGist: &github.Gist{
					ID:  "abcd1234",
					Url: "https://gist.github.com/some-user/abcd1234",
				},
			},
			content:             "super long content",
			maxContentSize:      5, // force overflow handling
			wantErr:             true,
			wantErrMsg:          "some create gist error",
			wantCreateGistCalls: 1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler, err := NewGistOverflowHandler(test.client, test.maxContentSize)
			if err != nil {
				t.Fatalf("unexpected error in NewGistOverflowHandler() %v", err)
			}

			got, err := handler.SavePullRequestBody(t.Context(), test.content)
			if test.wantErr {
				if err == nil {
					t.Fatalf("SavePullRequestBody() error = %v, wantErr %v", err, test.wantErr)
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Fatalf("want error message: %s, got: %s", test.wantErrMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error in SavePullRequestBody() %v", err)
			}

			if diff := cmp.Diff(got, test.want); diff != "" {
				t.Errorf("%s: SavePullRequestBody() mismatch (-want +got):%s", test.name, diff)
			}

			if diff := cmp.Diff(test.wantCreateGistCalls, test.client.createGistCalls); diff != "" {
				t.Errorf("%s: SavePullRequestBody() createGistCalls mismatch (-want +got):%s", test.name, diff)
			}
		})
	}
}

func TestFetchPullRequestBody(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name                    string
		client                  *mockGitHubClient
		maxContentSize          int
		content                 string
		want                    string
		wantErr                 bool
		wantErrMsg              string
		wantGetGistContentCalls int
	}{
		{
			name:    "non overflow",
			client:  &mockGitHubClient{},
			content: "some release notes",
			want:    "some release notes",
		},
		{
			name: "with overflow",
			client: &mockGitHubClient{
				getGistContent: map[string]string{
					"release-notes.md": "some release notes",
				},
			},
			content:                 "See full release notes at: https://gist.github.com/some-user/abcd1234",
			want:                    "some release notes",
			wantGetGistContentCalls: 1,
		},
		{
			name: "with overflow, error fetching gist",
			client: &mockGitHubClient{
				getGistError: fmt.Errorf("some fetch gist error"),
			},
			content:                 "See full release notes at: https://gist.github.com/some-user/abcd1234",
			wantErr:                 true,
			wantErrMsg:              "some fetch gist error",
			wantGetGistContentCalls: 1,
		},
		{
			name: "with overflow, wrong file",
			client: &mockGitHubClient{
				getGistContent: map[string]string{
					"unexpected file": "some release notes",
				},
			},
			content:                 "See full release notes at: https://gist.github.com/some-user/abcd1234",
			wantErr:                 true,
			wantErrMsg:              "unable to find",
			wantGetGistContentCalls: 1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler, err := NewGistOverflowHandler(test.client, test.maxContentSize)
			if err != nil {
				t.Fatalf("unexpected error in NewGistOverflowHandler() %v", err)
			}

			got, err := handler.FetchPullRequestBody(t.Context(), test.content)

			if test.wantErr {
				if err == nil {
					t.Fatalf("FetchPullRequestBody() error = %v, wantErr %v", err, test.wantErr)
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Fatalf("want error message: %s, got: %s", test.wantErrMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error in FetchPullRequestBody() %v", err)
			}

			if diff := cmp.Diff(got, test.want); diff != "" {
				t.Errorf("%s: FetchPullRequestBody() mismatch (-want +got):%s", test.name, diff)
			}

			if diff := cmp.Diff(test.wantGetGistContentCalls, test.client.getGistContentCalls); diff != "" {
				t.Errorf("%s: FetchPullRequestBody() createGistCalls mismatch (-want +got):%s", test.name, diff)
			}
		})
	}
}
