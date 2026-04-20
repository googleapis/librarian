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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestGetConfigValue(t *testing.T) {
	currentConfig := &config.Config{
		Version: "v1.0.0",
		Sources: &config.Sources{
			Googleapis:  &config.Source{Commit: "abcd123"},
			Conformance: &config.Source{Commit: "conf123"},
			Discovery:   &config.Source{Commit: "disc123"},
			ProtobufSrc: &config.Source{Commit: "proto123"},
			Showcase:    &config.Source{Commit: "show123"},
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
		{
			path: "sources.conformance.commit",
			want: "conf123",
		},
		{
			path: "sources.discovery.commit",
			want: "disc123",
		},
		{
			path: "sources.protobuf.commit",
			want: "proto123",
		},
		{
			path: "sources.showcase.commit",
			want: "show123",
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
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/repos/") {
			w.Write([]byte("xyz789"))
			return
		}
		if strings.HasSuffix(r.URL.Path, ".tar.gz") {
			w.Write([]byte("dummy content"))
			return
		}
		http.NotFound(w, r)
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
		{
			path:  "sources.conformance.commit",
			value: "xyz789",
			want: &config.Config{
				Version: "v1.0.0",
				Sources: &config.Sources{
					Googleapis: &config.Source{Commit: "abcd123"},
					Conformance: &config.Source{
						Commit: "xyz789",
						SHA256: dummySHA,
					},
				},
			},
		},
		{
			path:  "sources.discovery.commit",
			value: "xyz789",
			want: &config.Config{
				Version: "v1.0.0",
				Sources: &config.Sources{
					Googleapis: &config.Source{Commit: "abcd123"},
					Discovery: &config.Source{
						Commit: "xyz789",
						SHA256: dummySHA,
					},
				},
			},
		},
		{
			path:  "sources.protobuf.commit",
			value: "xyz789",
			want: &config.Config{
				Version: "v1.0.0",
				Sources: &config.Sources{
					Googleapis: &config.Source{Commit: "abcd123"},
					ProtobufSrc: &config.Source{
						Commit: "xyz789",
						SHA256: dummySHA,
					},
				},
			},
		},
		{
			path:  "sources.showcase.commit",
			value: "xyz789",
			want: &config.Config{
				Version: "v1.0.0",
				Sources: &config.Sources{
					Googleapis: &config.Source{Commit: "abcd123"},
					Showcase: &config.Source{
						Commit: "xyz789",
						SHA256: dummySHA,
					},
				},
			},
		},
		{
			path:  "sources.protobuf.subpath",
			value: "src",
			want: &config.Config{
				Version: "v1.0.0",
				Sources: &config.Sources{
					Googleapis: &config.Source{Commit: "abcd123"},
					ProtobufSrc: &config.Source{
						Subpath: "src",
					},
				},
			},
		},
	} {
		t.Run(test.path, func(t *testing.T) {
			cfg := &config.Config{
				Version: "v1.0.0",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "abcd123",
					},
				},
			}
			got, err := setConfigValue(cfg, test.path, test.value)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSetConfigValue_Error(t *testing.T) {
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
			cfg := &config.Config{
				Version: "v1.0.0",
			}
			got, err := setConfigValue(cfg, test.path, "some-value")
			if !errors.Is(err, test.wantErr) {
				t.Errorf("setConfigValue(%q) error = %v, wantErr %v", test.path, err, test.wantErr)
			}
			if got != nil {
				t.Errorf("setConfigValue(%q) got = %v, want nil on error", test.path, got)
			}
		})
	}
}
func TestSetConfigValue_FetchFailure_NoMutation(t *testing.T) {
	oldAPI := githubAPI
	oldDownload := githubDownload
	githubAPI = "://invalid-url"
	githubDownload = "://invalid-url"
	t.Cleanup(func() {
		githubAPI = oldAPI
		githubDownload = oldDownload
	})

	cfg := &config.Config{
		Version: "v1.0.0",
		Sources: &config.Sources{
			Googleapis: &config.Source{
				Commit: "abcd123",
			},
		},
	}
	originalConfig := &config.Config{
		Version: "v1.0.0",
		Sources: &config.Sources{
			Googleapis: &config.Source{
				Commit: "abcd123",
			},
		},
	}

	got, err := setConfigValue(cfg, "sources.googleapis.commit", "xyz789")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got != nil {
		t.Errorf("expected nil config on failure, got %v", got)
	}

	if diff := cmp.Diff(originalConfig, cfg); diff != "" {
		t.Errorf("cfg was mutated on failure (-want +got):\n%s", diff)
	}
}

func TestSetConfigValue_FetchFailure_NilSourcesRemainsNil(t *testing.T) {
	oldAPI := githubAPI
	oldDownload := githubDownload
	githubAPI = "://invalid-url"
	githubDownload = "://invalid-url"
	t.Cleanup(func() {
		githubAPI = oldAPI
		githubDownload = oldDownload
	})

	cfg := &config.Config{
		Version: "v1.0.0",
	}
	originalConfig := &config.Config{
		Version: "v1.0.0",
	}

	got, err := setConfigValue(cfg, "sources.googleapis.commit", "xyz789")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got != nil {
		t.Errorf("expected nil config on failure, got %v", got)
	}

	if diff := cmp.Diff(originalConfig, cfg); diff != "" {
		t.Errorf("cfg was mutated on failure (-want +got):\n%s", diff)
	}
}

func TestLibrarianSourcesYAML(t *testing.T) {
	filePath := "testdata/librarian_sources.yaml"
	cfg, err := yaml.Read[config.Config](filePath)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path string
		want string
	}{
		{"sources.conformance.commit", "b407e8416e3893036aee5af9a12bd9b6a0e2b2e6"},
		{"sources.discovery.commit", "a9217d2c6b2ce003fde50c0729c2bb8434434b5b"},
		{"sources.googleapis.commit", "f5cb7afc40b63d52f43bc306cb9b64a87b681aea"},
		{"sources.protobuf.commit", "b407e8416e3893036aee5af9a12bd9b6a0e2b2e6"},
		{"sources.protobuf.subpath", "src"},
		{"sources.showcase.commit", "3fd9cb2f682d5f8263d913eaba8b78e045acc4d2"},
	}

	for _, test := range tests {
		got, err := getConfigValue(cfg, test.path)
		if err != nil {
			t.Errorf("getConfigValue(%q) error = %v", test.path, err)
			continue
		}
		if got != test.want {
			t.Errorf("getConfigValue(%q) = %q, want %q", test.path, got, test.want)
		}
	}

	updatedCfg, err := setConfigValue(cfg, "sources.conformance.subpath", "new-subpath")
	if err != nil {
		t.Fatal(err)
	}

	got, err := getConfigValue(updatedCfg, "sources.conformance.subpath")
	if err != nil {
		t.Fatal(err)
	}
	if got != "new-subpath" {
		t.Errorf("Expected conformance subpath to be 'new-subpath', got %q", got)
	}
}
