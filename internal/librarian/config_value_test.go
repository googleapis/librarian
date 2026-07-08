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

func setupCfgUtilityTestServer(t *testing.T) {
	t.Helper()
	originalAPI := githubAPI
	originalDownload := githubDownload
	t.Cleanup(func() {
		githubAPI = originalAPI
		githubDownload = originalDownload
	})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/googleapis/googleapis/commits/main-branch":
			w.Write([]byte("googleapis123"))
		case "/googleapis/googleapis/archive/googleapis123.tar.gz":
			w.Write([]byte("googleapis-archive"))
		case "/repos/protocolbuffers/protobuf/commits/proto-branch":
			w.Write([]byte("protobuf123"))
		case "/protocolbuffers/protobuf/archive/protobuf123.tar.gz":
			w.Write([]byte("protobuf-archive"))
		default:
			http.NotFound(w, r)
		}
	}))
	githubAPI = ts.URL
	githubDownload = ts.URL
}

func TestGetConfigValue(t *testing.T) {
	currentConfig := &config.Config{
		Version: "v1.0.0",
		Sources: &config.Sources{
			Googleapis: &config.Source{
				Commit: "googleapis123",
				SHA256: "googleapis-sha",
				Dir:    "googleapis-dir",
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
			want: "googleapis123",
		},
		{
			path: "sources.googleapis.sha256",
			want: "googleapis-sha",
		},
		{
			path: "sources.googleapis.dir",
			want: "googleapis-dir",
		},
		{
			path: "sources.googleapis.subpath",
			want: "",
		},
	} {
		t.Run(test.path, func(t *testing.T) {
			got, err := getConfigValue(currentConfig, test.path)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
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
			name:    "unsupported path",
			path:    "invalid.path",
			wantErr: errUnsupportedPath,
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
	setupCfgUtilityTestServer(t)
	expectedGoogleapisSHA := fmt.Sprintf("%x", sha256.Sum256([]byte("googleapis-archive")))
	expectedProtobufSHA := fmt.Sprintf("%x", sha256.Sum256([]byte("protobuf-archive")))
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
			},
		},
		{
			path:  "sources.googleapis.commit",
			value: "main-branch",
			want: &config.Config{
				Version: "v1.0.0",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "googleapis123",
						SHA256: expectedGoogleapisSHA,
					},
				},
			},
		},
		{
			path:  "sources.protobuf.commit",
			value: "proto-branch",
			want: &config.Config{
				Version: "v1.0.0",
				Sources: &config.Sources{
					ProtobufSrc: &config.Source{
						Commit: "protobuf123",
						SHA256: expectedProtobufSHA,
					},
				},
			},
		},
		{
			path:  "sources.googleapis.dir",
			value: "some-dir",
			want: &config.Config{
				Version: "v1.0.0",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Dir: "some-dir",
					},
				},
			},
		},
		{
			path:  "sources.conformance.subpath",
			value: "some-subpath",
			want: &config.Config{
				Version: "v1.0.0",
				Sources: &config.Sources{
					Conformance: &config.Source{
						Subpath: "some-subpath",
					},
				},
			},
		},
	} {
		t.Run(test.path, func(t *testing.T) {
			cfg := &config.Config{
				Version: "v1.0.0",
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
	setupCfgUtilityTestServer(t)
	for _, test := range []struct {
		name    string
		path    string
		value   string
		wantErr error
	}{
		{
			name:    "unsupported path",
			path:    "unknown.field",
			value:   "some-value-not-used",
			wantErr: errUnsupportedPath,
		},
		{
			name:  "failed fetch commit",
			path:  "sources.googleapis.commit",
			value: "non-existent-branch",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				Version: "v1.0.0",
			}
			_, err := setConfigValue(cfg, test.path, test.value)
			if err == nil {
				t.Errorf("setConfigValue(%q, %q) got nil err, want error", test.path, test.value)
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Errorf("setConfigValue(%q, %q) error = %v, wantErr %v", test.path, test.value, err, test.wantErr)
			}
		})
	}
}
