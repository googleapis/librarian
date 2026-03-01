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

package golang

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestBump(t *testing.T) {
	for _, test := range []struct {
		name         string
		initialFiles map[string]string
		library      *config.Library
		version      string
		wantFiles    map[string]string
	}{
		{
			name: "bump internal version",
			initialFiles: map[string]string{
				"test-lib/internal/version.go": "package internal\n\nconst Version = \"0.1.0\"\n",
			},
			library: &config.Library{Name: "test-lib"},
			version: "0.2.0",
			wantFiles: map[string]string{
				"test-lib/internal/version.go": "package internal\n\nconst Version = \"0.2.0\"\n",
			},
		},
		{
			name: "ignore other files",
			initialFiles: map[string]string{
				"test-lib/version.go":                "package testlib\n\nconst Version = \"0.1.0\"\n",
				"test-lib/internal/other.go":         "package internal\n\nconst Version = \"0.1.0\"\n",
				"test-lib/other/internal/version.go": "package internal\n\nconst Version = \"0.1.0\"\n",
			},
			library: &config.Library{Name: "test-lib"},
			version: "0.2.0",
			wantFiles: map[string]string{
				"test-lib/version.go":                "package testlib\n\nconst Version = \"0.1.0\"\n",
				"test-lib/internal/other.go":         "package internal\n\nconst Version = \"0.1.0\"\n",
				"test-lib/other/internal/version.go": "package internal\n\nconst Version = \"0.2.0\"\n",
			},
		},
		{
			name: "ignore nested module",
			initialFiles: map[string]string{
				"test-lib/internal/version.go":               "package internal\n\nconst Version = \"0.1.0\"\n",
				"test-lib/nested-module/internal/version.go": "package internal\n\nconst Version = \"0.1.0\"\n",
			},
			library: &config.Library{
				Name: "test-lib",
				Go: &config.GoModule{
					NestedModule: "nested-module",
				},
			},
			version: "0.2.0",
			wantFiles: map[string]string{
				"test-lib/internal/version.go":               "package internal\n\nconst Version = \"0.2.0\"\n",
				"test-lib/nested-module/internal/version.go": "package internal\n\nconst Version = \"0.1.0\"\n",
			},
		},
		{
			name: "bump snippet metadata",
			initialFiles: map[string]string{
				"internal/generated/snippets/test-lib/nested-1/nested-2/snippet_metadata_foo.json":   "{\n  \"clientLibrary\": {\n    \"version\": \"0.1.0\"\n  }\n}\n",
				"internal/generated/snippets/test-lib/nested-mod/nested-2/snippet_metadata_foo.json": "{\n  \"clientLibrary\": {\n    \"version\": \"0.1.0\"\n  }\n}\n",
			},
			library: &config.Library{
				Name: "test-lib",
				Go: &config.GoModule{
					NestedModule: "nested-mod",
				},
			},
			version: "0.2.0",
			wantFiles: map[string]string{
				"internal/generated/snippets/test-lib/nested-1/nested-2/snippet_metadata_foo.json":   "{\n  \"clientLibrary\": {\n    \"version\": \"0.2.0\"\n  }\n}",
				"internal/generated/snippets/test-lib/nested-mod/nested-2/snippet_metadata_foo.json": "{\n  \"clientLibrary\": {\n    \"version\": \"0.1.0\"\n  }\n}\n",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			output := t.TempDir()
			libraryDir := filepath.Join(output, test.library.Name)
			if err := os.MkdirAll(libraryDir, 0755); err != nil {
				t.Fatal(err)
			}
			snippetsDir := filepath.Join(output, "internal", "generated", "snippets", test.library.Name)
			if err := os.MkdirAll(snippetsDir, 0755); err != nil {
				t.Fatal(err)
			}

			for path, content := range test.initialFiles {
				fullPath := filepath.Join(output, path)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}
			err := Bump(test.library, output, test.version)
			if err != nil {
				t.Fatal(err)
			}
			for path, wantContent := range test.wantFiles {
				fullPath := filepath.Join(output, path)
				content, err := os.ReadFile(fullPath)
				if err != nil {
					t.Error(err)
					continue
				}
				got := string(content)
				if diff := cmp.Diff(wantContent, got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestBump_Error(t *testing.T) {
	for _, test := range []struct {
		name         string
		initialFiles map[string]string
		library      *config.Library
		version      string
		setup        func(t *testing.T, dir string)
		wantErr      error
	}{
		{
			name: "internal version file is read-only",
			initialFiles: map[string]string{
				"test-lib/internal/version.go": "package internal\n\nconst Version = \"0.1.0\"\n",
			},
			library: &config.Library{Name: "test-lib"},
			version: "0.2.0",
			setup: func(t *testing.T, dir string) {
				if err := os.Chmod(filepath.Join(dir, "test-lib", "internal", "version.go"), 0444); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: os.ErrPermission,
		},
		{
			name: "snippet metadata is read-only",
			initialFiles: map[string]string{
				"internal/generated/snippets/test-lib/snippet_metadata_foo.json": "{\n  \"clientLibrary\": {\n    \"version\": \"0.1.0\"\n  }\n}\n",
			},
			library: &config.Library{Name: "test-lib"},
			version: "0.2.0",
			setup: func(t *testing.T, dir string) {
				if err := os.Chmod(filepath.Join(dir, "internal", "generated", "snippets", "test-lib", "snippet_metadata_foo.json"), 0444); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: os.ErrPermission,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			output := t.TempDir()
			libraryDir := filepath.Join(output, test.library.Name)
			if err := os.MkdirAll(libraryDir, 0755); err != nil {
				t.Fatal(err)
			}
			snippetsDir := filepath.Join(output, "internal", "generated", "snippets", test.library.Name)
			if err := os.MkdirAll(snippetsDir, 0755); err != nil {
				t.Fatal(err)
			}

			for path, content := range test.initialFiles {
				fullPath := filepath.Join(output, path)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			if test.setup != nil {
				test.setup(t, output)
			}
			err := Bump(test.library, output, test.version)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("Bump() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
