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

func TestParseUrl(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
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
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repo, err := ParseUrl(test.remoteUrl)

			if test.wantErr {
				if err == nil {
					t.Errorf("ParseUrl() err = nil, want error containing %q", test.wantErrSubstr)
				} else if !strings.Contains(err.Error(), test.wantErrSubstr) {
					t.Errorf("ParseUrl() err = %v, want error containing %q", err, test.wantErrSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("ParseUrl() err = %v, want nil", err)
				}
				if diff := cmp.Diff(test.wantRepo, repo); diff != "" {
					t.Errorf("ParseUrl() repo mismatch (-want +got): %s", diff)
				}
			}
		})
	}
}
