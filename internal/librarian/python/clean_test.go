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

package python

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestCleanLibrary(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		lib         *config.Library
		setupFiles  []string
		wantDeleted []string
	}{
		{
			name: "output directory doesn't exist",
			lib: &config.Library{
				Name: "test",
			},
		},
		{
			name: "no APIs",
			lib: &config.Library{
				Name: "test",
			},
			setupFiles: []string{"README.md"},
		},
		{
			name: "proto-only API",
			lib: &config.Library{
				Name: "test",
				APIs: []*config.API{
					{Path: "google/type"},
				},
				Python: &config.PythonPackage{
					ProtoOnlyAPIs: []string{"google/type"},
				},
				Keep: []string{"google/type/keep.proto"},
			},
			setupFiles: []string{
				"README.md",
				"google/type/date.proto",
				"google/type/keep.proto",
				"google/type/date_pb2.py",
				"google/type/date_pb2.pyi",
				"google/type/README.txt",
			},
			wantDeleted: []string{
				"google/type/date.proto",
				"google/type/date_pb2.py",
				"google/type/date_pb2.pyi",
			},
		},
		{
			name: "GAPIC API",
			lib: &config.Library{
				Name: "test",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
				Python: &config.PythonPackage{
					PythonDefault: config.PythonDefault{
						CommonGAPICPaths: []string{
							"{neutral-source}/delete-me.txt",
							// This is in the keep list as well, so should be kept
							"{neutral-source}/keep-me.txt",
							"docs/delete-me.txt",
						},
					},
				},
				Keep: []string{
					"google/cloud/secretmanager/keep-me.txt",
					"google/cloud/secretmanager_v1/keep-me.txt",
				},
			},
			setupFiles: []string{
				"README.md",
				"google/cloud/secretmanager/delete-me.txt",
				"google/cloud/secretmanager/leave-me.txt",
				"google/cloud/secretmanager/keep-me.txt",
				"google/cloud/secretmanager/delete-me.txt",
				"google/cloud/secretmanager_v1/delete-me.txt",
				"google/cloud/secretmanager_v1/keep-me.txt",
				"docs/delete-me.txt",
			},
			wantDeleted: []string{
				"google/cloud/secretmanager/delete-me.txt",
				"google/cloud/secretmanager_v1/delete-me.txt",
				"docs/delete-me.txt",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			// Note: deliberately not creating the subdirectory to start with,
			// so that if we have no files to create, the directory isn't
			// created either.
			test.lib.Output = filepath.Join(dir, test.lib.Name)
			for _, file := range test.setupFiles {
				fullPath := filepath.Join(test.lib.Output, file)
				createFileAndDirectories(t, fullPath)
			}

			if err := CleanLibrary(test.lib); err != nil {
				t.Fatal(err)
			}

			for _, file := range test.setupFiles {
				fullPath := filepath.Join(test.lib.Output, file)
				_, err := os.Stat(fullPath)
				if err != nil && !os.IsNotExist(err) {
					t.Fatal(err)
				}
				gotDeleted := err != nil
				wantDeleted := slices.Contains(test.wantDeleted, file)
				if gotDeleted != wantDeleted {
					t.Errorf("file %s: wantDeleted=%t, gotDeleted=%t", file, wantDeleted, gotDeleted)
				}
			}
		})
	}
}

func TestCleanLibrary_Error(t *testing.T) {
}

func createFileAndDirectories(t *testing.T, path string) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
}
