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
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

// TestRunConfigGet tests that the config get command successfully reads a value from librarian.yaml.
func TestRunConfigGet(t *testing.T) {
	for _, test := range []struct {
		name       string
		path       string
		value      string
		configYAML string
		want       string
	}{
		{
			name:       "get string value",
			path:       "version",
			configYAML: "version: 1.2.3\n",
			want:       "1.2.3\n",
		},
		{
			name:  "get library (existing)",
			path:  "libraries",
			value: "google/cloud/secretmanager/v1",
			configYAML: `language: python
libraries:
- name: mysecretmanager
  apis:
  - path: google/cloud/secretmanager/v1
`,
			want: "mysecretmanager\n",
		},
		{
			name:  "get library with custom name (ruby)",
			path:  "libraries",
			value: "google/cloud/secretmanager/v1",
			configYAML: `language: ruby
libraries:
- name: google-cloud-secret_manager-v1
  apis:
  - path: google/cloud/secretmanager/v1
`,
			want: "google-cloud-secret_manager-v1\n",
		},
		{
			name:  "get library with subpackage API path (go)",
			path:  "libraries",
			value: "google/cloud/oslogin/common",
			configYAML: `language: go
libraries:
- name: oslogin
  apis:
  - path: google/cloud/oslogin/v1
  - path: google/cloud/oslogin/common
`,
			want: "oslogin\n",
		},
		{
			name:  "get library with API path derived from name (rust)",
			path:  "libraries",
			value: "google/cloud/secretmanager/v1",
			configYAML: `language: rust
libraries:
- name: google-cloud-secretmanager-v1
`,
			want: "google-cloud-secretmanager-v1\n",
		},
		{
			name:  "get versioned library, not wrapper (ruby)",
			path:  "libraries",
			value: "google/cloud/secretmanager/v1",
			configYAML: `language: ruby
libraries:
- name: google-cloud-secret_manager
  apis:
  - path: google/cloud/secretmanager/v1
  ruby:
    wrapper_of:
    - v1:0.0
- name: google-cloud-secret_manager-v1
  apis:
  - path: google/cloud/secretmanager/v1
`,
			want: "google-cloud-secret_manager-v1\n",
		},
		{
			name:  "get wrapper library for unversioned path (ruby)",
			path:  "libraries",
			value: "google/cloud/secretmanager",
			configYAML: `language: ruby
libraries:
- name: google-cloud-secret_manager
  apis:
  - path: google/cloud/secretmanager/v1
  ruby:
    wrapper_of:
    - v1:0.0
- name: google-cloud-secret_manager-v1
  apis:
  - path: google/cloud/secretmanager/v1
`,
			want: "google-cloud-secret_manager\n",
		},
		{
			name:  "get mixed library via rust module API path",
			path:  "libraries",
			value: "google/storage/v2",
			configYAML: `language: rust
libraries:
- name: google-cloud-storage
  rust:
    modules:
    - api_path: google/storage/v2
    - api_path: google/iam/v1
`,
			want: "google-cloud-storage\n",
		},
		{
			name:  "get mixed library via swift module API path",
			path:  "libraries",
			value: "google/storage/v2",
			configYAML: `language: swift
libraries:
- name: google-cloud-storage
  swift:
    modules:
    - api_path: google/storage/v2
`,
			want: "google-cloud-storage\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)
			cfg, err := yaml.Unmarshal[config.Config]([]byte(test.configYAML))
			if err != nil {
				t.Fatal(err)
			}
			if err := yaml.Write("librarian.yaml", cfg); err != nil {
				t.Fatal(err)
			}
			var buf bytes.Buffer
			err = runConfigGet(&buf, test.path, test.value)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, buf.String()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestRunConfigGet_Error tests that the config get command returns an error when the path is missing or the key is not found.
func TestRunConfigGet_Error(t *testing.T) {
	googleapisDir, err := filepath.Abs("../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		path       string
		value      string
		configYAML string
		wantErr    error
	}{
		{
			name:       "missing path",
			path:       "",
			configYAML: "version: 1.2.3\n",
			wantErr:    errPathRequired,
		},
		{
			name:       "key not found",
			path:       "foo",
			configYAML: "version: 1.2.3\n",
			wantErr:    errUnsupportedPath,
		},
		{
			name:       "missing value for library path",
			path:       "libraries",
			value:      "",
			configYAML: "language: go\n",
			wantErr:    errValueRequired,
		},
		{
			name:       "library not found for API",
			path:       "libraries",
			value:      "google/cloud/secretmanager/v1",
			configYAML: "language: go\n",
			wantErr:    ErrLibraryNotFound,
		},
		{
			name:  "unconfigured API path in configured service (go)",
			path:  "libraries",
			value: "google/cloud/secretmanager/logging/v1",
			configYAML: `language: go
libraries:
- name: secretmanager
  apis:
  - path: google/cloud/secretmanager/v1
`,
			wantErr: ErrLibraryNotFound,
		},
		{
			name:  "API path not in googleapis (go)",
			path:  "libraries",
			value: "google/cloud/secretmanager/v1beta3",
			configYAML: fmt.Sprintf(`language: go
sources:
  googleapis:
    dir: %s
libraries:
- name: secretmanager
  apis:
  - path: google/cloud/secretmanager/v1
`, googleapisDir),
			wantErr: ErrLibraryNotFound,
		},
		{
			name:  "unconfigured version of configured service (python)",
			path:  "libraries",
			value: "google/cloud/secretmanager/v1beta2",
			configYAML: `language: python
libraries:
- name: google-cloud-secret-manager
  apis:
  - path: google/cloud/secretmanager/v1
`,
			wantErr: ErrLibraryNotFound,
		},
		{
			name:  "unconfigured service (ruby)",
			path:  "libraries",
			value: "google/cloud/secretmanager/v1",
			configYAML: `language: ruby
libraries:
- name: google-cloud-tasks-v2
  apis:
  - path: google/cloud/tasks/v2
`,
			wantErr: ErrLibraryNotFound,
		},
		{
			name:  "embedded mixin module API path is not matched (rust)",
			path:  "libraries",
			value: "google/iam/v1",
			configYAML: `language: rust
libraries:
- name: google-cloud-storage
  rust:
    modules:
    - api_path: google/storage/v2
    - api_path: google/iam/v1
`,
			wantErr: ErrLibraryNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)
			if err := os.WriteFile("librarian.yaml", []byte(test.configYAML), 0o644); err != nil {
				t.Fatal(err)
			}
			var buf bytes.Buffer
			err := runConfigGet(&buf, test.path, test.value)
			if err == nil {
				t.Fatal("expected error; got nil")
			}
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("got error %v, want %v", err, test.wantErr)
			}
		})
	}
}

// TestRunConfigSet tests that the config set command successfully updates a value in librarian.yaml.
func TestRunConfigSet(t *testing.T) {
	for _, test := range []struct {
		name       string
		path       string
		value      string
		configYAML string
		wantYAML   string
	}{
		{
			name:       "set string value",
			path:       "version",
			value:      "1.2.4",
			configYAML: "version: 1.2.3\n",
			wantYAML:   "version: 1.2.4\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)
			if err := os.WriteFile("librarian.yaml", []byte(test.configYAML), 0o644); err != nil {
				t.Fatal(err)
			}
			err := runConfigSet(test.path, test.value)
			if err != nil {
				t.Fatal(err)
			}
			gotYAML, err := os.ReadFile("librarian.yaml")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasSuffix(string(gotYAML), test.wantYAML) {
				t.Errorf("got YAML =\n%s\nwant ending with %q", string(gotYAML), test.wantYAML)
			}
		})
	}
}

// TestRunConfigSet_Error tests that the config set command returns an error when the path or value is missing.
func TestRunConfigSet_Error(t *testing.T) {
	for _, test := range []struct {
		name       string
		path       string
		value      string
		configYAML string
		wantErr    error
	}{
		{
			name:       "missing path",
			path:       "",
			value:      "",
			configYAML: "version: 1.2.3\n",
			wantErr:    errPathRequired,
		},
		{
			name:       "missing value",
			path:       "version",
			value:      "",
			configYAML: "version: 1.2.3\n",
			wantErr:    errValueRequired,
		},
		{
			name:       "unsupported path",
			path:       "foo",
			value:      "bar",
			configYAML: "version: 1.2.3\n",
			wantErr:    errUnsupportedPath,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)
			if err := os.WriteFile("librarian.yaml", []byte(test.configYAML), 0o644); err != nil {
				t.Fatal(err)
			}
			err := runConfigSet(test.path, test.value)
			if err == nil {
				t.Fatal("expected error; got nil")
			}
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("got error %v, want %v", err, test.wantErr)
			}
		})
	}
}

// TestRunConfigSet_FileNotFound tests that the config set command returns an error when the file doesn't exist.
func TestRunConfigSet_FileNotFound(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	err := runConfigSet("version", "1.2.4")
	if err == nil {
		t.Fatal("expected error; got nil")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("got error %v, want %v", err, fs.ErrNotExist)
	}
}
