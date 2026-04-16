// Copyright 2026 Google LLC
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

package librarian

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestGetConfigValue(t *testing.T) {
	currentConfig := &config.Config{
		Version: "v1.0.0",
		Sources: &config.Sources{
			Googleapis: &config.Source{
				Commit: "abcd123",
			},
		},
	}

	for _, test := range []struct {
		path string
		want string
	}{
		{
			path: "version",
			want: "v1.0.0",
		},
		{
			path: "sources.googleapis.commit",
			want: "abcd123",
		},
	} {
		t.Run(test.path, func(t *testing.T) {
			got, err := getConfigValue(currentConfig, test.path)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("getConfigValue(%q) = %q, want %q", test.path, got, test.want)
			}
		})
	}
}

func TestGetConfigValue_Error(t *testing.T) {
	currentConfig := &config.Config{
		Version: "v1.0.0",
	}
	for _, test := range []struct {
		name    string
		path    string
		wantErr error
	}{
		{
			name:    "invalid path",
			path:    "invalid.path",
			wantErr: errUnsupportedPath,
		},
		{
			name:    "missing sources",
			path:    "sources.googleapis.commit",
			wantErr: errSourceNotConfigured,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := getConfigValue(currentConfig, test.path)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("getConfigValue(%q) error = %v, wantErr %v", test.path, err, test.wantErr)
			}
		})
	}
}

func TestSetConfigValue(t *testing.T) {
	// Start a mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/googleapis/googleapis/commits/xyz789":
			w.Write([]byte("xyz789"))
		case "/googleapis/googleapis/archive/xyz789.tar.gz":
			w.Write([]byte("dummy content"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	// Override endpoints
	oldAPI := githubAPI
	oldDownload := githubDownload
	githubAPI = ts.URL
	githubDownload = ts.URL
	defer func() {
		githubAPI = oldAPI
		githubDownload = oldDownload
	}()

	dummySHA := fmt.Sprintf("%x", sha256.Sum256([]byte("dummy content")))

	for _, test := range []struct {
		path  string
		value string
		want  *config.Config
	}{
		{
			path:  "version",
			value: "v1.0.1",
			want: &config.Config{
				Version: "v1.0.1",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "abcd123",
					},
				},
			},
		},
		{
			path:  "sources.googleapis.commit",
			value: "xyz789",
			want: &config.Config{
				Version: "v1.0.0",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "xyz789",
						SHA256: dummySHA,
					},
				},
			},
		},
	} {
		t.Run(test.path, func(t *testing.T) {
			currentConfig := &config.Config{
				Version: "v1.0.0",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "abcd123",
					},
				},
			}
			err := setConfigValue(currentConfig, test.path, test.value)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, currentConfig); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSetConfigValue_Error(t *testing.T) {
	currentConfig := &config.Config{
		Version: "v1.0.0",
	}

	for _, test := range []struct {
		name    string
		path    string
		wantErr error
	}{
		{
			name:    "invalid path (misspelled)",
			path:    "surces.googleapis.commit",
			wantErr: errUnsupportedPath,
		},
		{
			name:    "unsupported path",
			path:    "unknown.field",
			wantErr: errUnsupportedPath,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := setConfigValue(currentConfig, test.path, "some-value")
			if !errors.Is(err, test.wantErr) {
				t.Errorf("setConfigValue(%q) error = %v, wantErr %v", test.path, err, test.wantErr)
			}
		})
	}
}
