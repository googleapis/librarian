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
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetRawContent(t *testing.T) {
	t.Parallel()
	path := "path/to/file"
	ref := "main"

	testCases := []struct {
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tc.handler(w, r)
			}))
			defer server.Close()

			repo := &Repository{Owner: "owner", Name: "repo"}
			client, err := newClientWithHTTP("fake-token", repo, server.Client())
			if err != nil {
				t.Fatalf("newClientWithHTTP() error = %v", err)
			}

			client.BaseURL, _ = url.Parse(server.URL + "/")
			content, err := client.GetRawContent(context.Background(), path, ref)

			if tc.wantErr {
				if err == nil {
					t.Errorf("GetRawContent() err = nil, want error containing %q", tc.wantErrSubstr)
				} else if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("GetRawContent() err = %v, want error containing %q", err, tc.wantErrSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("GetRawContent() err = %v, want nil", err)
				}
				if diff := cmp.Diff(tc.wantContent, content); diff != "" {
					t.Errorf("GetRawContent() content mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestParseUrl(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name          string
		remoteUrl     string
		wantRepo      *Repository
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:      "Valid HTTPS URL",
			remoteUrl: "https://github.com/owner/repo.git",
			wantRepo:  &Repository{Owner: "owner", Name: "repo"},
			wantErr:   false,
		},
		{
			name:      "Valid HTTPS URL without .git",
			remoteUrl: "https://github.com/owner/repo",
			wantRepo:  &Repository{Owner: "owner", Name: "repo"},
			wantErr:   false,
		},
		{
			name:          "Invalid URL scheme",
			remoteUrl:     "http://github.com/owner/repo.git",
			wantErr:       true,
			wantErrSubstr: "not a GitHub remote",
		},
		{
			name:      "URL with extra path components",
			remoteUrl: "https://github.com/owner/repo/pulls",
			wantRepo:  &Repository{Owner: "owner", Name: "repo"},
			wantErr:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo, err := ParseUrl(tc.remoteUrl)

			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseUrl() err = nil, want error containing %q", tc.wantErrSubstr)
				} else if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("ParseUrl() err = %v, want error containing %q", err, tc.wantErrSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("ParseUrl() err = %v, want nil", err)
				}
				if diff := cmp.Diff(tc.wantRepo, repo); diff != "" {
					t.Errorf("ParseUrl() repo mismatch (-want +got): %s", diff)
				}
			}
		})
	}
}
