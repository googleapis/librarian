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

package proto

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGather(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		relPath   string
		files     []string
		wantFiles []string
	}{
		{
			name:    "recursive collection including subdirectories",
			relPath: "google/cloud/secretmanager/v1",
			files: []string{
				"service.proto",
				"resources.proto",
				"schema/schema.proto",
				"schema/nested/nested.proto",
				"README.md",
				"config.yaml",
			},
			wantFiles: []string{
				"resources.proto",
				"schema/nested/nested.proto",
				"schema/schema.proto",
				"service.proto",
			},
		},
		{
			name:    "non-recursive for google/api",
			relPath: "google/api",
			files: []string{
				"annotations.proto",
				"http.proto",
				"sub/nested.proto",
				"sub/more/deep.proto",
			},
			wantFiles: []string{
				"annotations.proto",
				"http.proto",
			},
		},
		{
			name:    "non-recursive for google/cloud",
			relPath: "google/cloud",
			files: []string{
				"common.proto",
				"secretmanager/v1/service.proto",
			},
			wantFiles: []string{
				"common.proto",
			},
		},
		{
			name:    "non-recursive with OS-native path separators",
			relPath: filepath.FromSlash("google/cloud"),
			files: []string{
				"common.proto",
				"nested/nested.proto",
			},
			wantFiles: []string{
				"common.proto",
			},
		},
		{
			name:    "no proto files in directory",
			relPath: "google/cloud/empty",
			files: []string{
				"README.md",
				"doc.txt",
			},
			wantFiles: nil,
		},
		{
			name:      "empty directory",
			relPath:   "google/cloud/empty",
			files:     nil,
			wantFiles: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for _, f := range test.files {
				full := filepath.Join(root, f)
				if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(full, []byte("// proto"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, err := Gather(root, test.relPath)
			if err != nil {
				t.Fatal(err)
			}
			var want []string
			for _, f := range test.wantFiles {
				want = append(want, filepath.Join(root, f))
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGather_Error(t *testing.T) {
	_, err := Gather("/non/existent/path", "google/cloud/foo")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Gather() error = %v, want %v", err, fs.ErrNotExist)
	}
}

func TestSearch(t *testing.T) {
	t.Parallel()
	phpNamespaceRe := regexp.MustCompile(`option\s+php_namespace\s*=\s*"([^"]+)";`)
	for _, test := range []struct {
		name      string
		regex     *regexp.Regexp
		files     map[string]string
		want      string
		wantFound bool
	}{
		{
			name:  "match found",
			regex: phpNamespaceRe,
			files: map[string]string{
				"service.proto": `option php_namespace = "Google\\Cloud\\SecretManager\\V1";`,
			},
			want:      `Google\\Cloud\\SecretManager\\V1`,
			wantFound: true,
		},
		{
			name:  "extra whitespace",
			regex: phpNamespaceRe,
			files: map[string]string{
				"service.proto": `  option   php_namespace   =   "Google\\Cloud\\Storage\\V1";`,
			},
			want:      `Google\\Cloud\\Storage\\V1`,
			wantFound: true,
		},
		{
			name:  "first match returned",
			regex: phpNamespaceRe,
			files: map[string]string{
				"service.proto": "option php_namespace = \"Google\\\\Cloud\\\\First\";\noption php_namespace = \"Google\\\\Cloud\\\\Second\";",
			},
			want:      `Google\\Cloud\\First`,
			wantFound: true,
		},
		{
			name:  "ignores comments",
			regex: phpNamespaceRe,
			files: map[string]string{
				"service.proto": "// option php_namespace = \"Google\\\\Cloud\\\\Ignored\";\noption php_namespace = \"Google\\\\Cloud\\\\SecretManager\\\\V1\";",
			},
			want:      `Google\\Cloud\\SecretManager\\V1`,
			wantFound: true,
		},
		{
			name:  "ignores non-proto files",
			regex: phpNamespaceRe,
			files: map[string]string{
				"README.md":     `option php_namespace = "Google\\Cloud\\Ignored";`,
				"service.yaml":  `type: google.api.Service`,
				"service.proto": `option php_namespace = "Google\\Cloud\\SecretManager\\V1";`,
			},
			want:      `Google\\Cloud\\SecretManager\\V1`,
			wantFound: true,
		},
		{
			name:  "no match in proto",
			regex: phpNamespaceRe,
			files: map[string]string{
				"service.proto": `syntax = "proto3";` + "\n" + `package google.cloud.secretmanager.v1;`,
			},
			want:      "",
			wantFound: false,
		},
		{
			name:  "only commented match",
			regex: phpNamespaceRe,
			files: map[string]string{
				"service.proto": `// option php_namespace = "Google\\Cloud\\SecretManager\\V1";`,
			},
			want:      "",
			wantFound: false,
		},
		{
			name:  "different option regex",
			regex: regexp.MustCompile(`option\s+go_package\s*=\s*"([^"]+)";`),
			files: map[string]string{
				"service.proto": `option go_package = "cloud.google.com/go/secretmanager/apiv1;secretmanager";`,
			},
			want:      "cloud.google.com/go/secretmanager/apiv1;secretmanager",
			wantFound: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			apiPath := "google/cloud/secretmanager/v1"
			dir := filepath.Join(tmpDir, apiPath)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			for name, content := range test.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, found, err := Search(tmpDir, apiPath, test.regex)
			if err != nil {
				t.Fatal(err)
			}
			if found != test.wantFound {
				t.Errorf("found = %v, want %v", found, test.wantFound)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSearch_Error(t *testing.T) {
	t.Parallel()
	phpNamespaceRe := regexp.MustCompile(`option\s+php_namespace\s*=\s*"([^"]+)";`)
	for _, test := range []struct {
		name    string
		setup   func(t *testing.T) (string, string)
		wantErr error
	}{
		{
			name: "nonexistent directory",
			setup: func(t *testing.T) (string, string) {
				return t.TempDir(), "google/cloud/nonexistent/v1"
			},
			wantErr: fs.ErrNotExist,
		},
		{
			name: "no proto files in directory",
			setup: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				apiPath := "google/cloud/test/v1"
				dir := filepath.Join(tmpDir, apiPath)
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# docs"), 0o644); err != nil {
					t.Fatal(err)
				}
				return tmpDir, apiPath
			},
			wantErr: fs.ErrNotExist,
		},
		{
			name: "unreadable proto file",
			setup: func(t *testing.T) (string, string) {
				if os.Geteuid() == 0 {
					t.Skip("skipping permission test when running as root")
				}
				tmpDir := t.TempDir()
				apiPath := "google/cloud/test/v1"
				dir := filepath.Join(tmpDir, apiPath)
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				file := filepath.Join(dir, "service.proto")
				if err := os.WriteFile(file, []byte("syntax = \"proto3\";"), 0o000); err != nil {
					t.Fatal(err)
				}
				return tmpDir, apiPath
			},
			wantErr: fs.ErrPermission,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			googleapisDir, apiPath := test.setup(t)
			got, found, err := Search(googleapisDir, apiPath, phpNamespaceRe)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("Search() error = %v, want %v", err, test.wantErr)
			}
			if found {
				t.Errorf("Search() found = %v, want false", found)
			}
			if got != "" {
				t.Errorf("Search() got = %q, want empty string", got)
			}
		})
	}
}
