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
			name:    "non-recursive for google/rpc",
			relPath: "google/rpc",
			files: []string{
				"status.proto",
				"context/attribute_context.proto",
			},
			wantFiles: []string{
				"status.proto",
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
