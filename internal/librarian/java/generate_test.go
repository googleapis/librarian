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

package java

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestGenerateLibraries(t *testing.T) {
	libraries := []*config.Library{
		{
			Name: "test-lib",
			APIs: []*config.API{
				{Path: "google/cloud/test/v1"},
			},
		},
	}
	googleapisDir := "/tmp/googleapis"

	if err := GenerateLibraries(t.Context(), libraries, googleapisDir); err != nil {
		t.Errorf("GenerateLibraries() error = %v, want nil", err)
	}
}

func TestGenerateLibraries_Error(t *testing.T) {
	for _, test := range []struct {
		name      string
		libraries []*config.Library
	}{
		{
			name: "no apis",
			libraries: []*config.Library{
				{
					Name: "test-lib",
					APIs: nil,
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := GenerateLibraries(t.Context(), test.libraries, "/tmp"); err == nil {
				t.Error("GenerateLibraries() error = nil, want error")
			}
		})
	}
}

func TestFormat_Success(t *testing.T) {
	t.Parallel()
	testhelper.RequireCommand(t, "google-java-format")
	for _, test := range []struct {
		name  string
		setup func(t *testing.T, root string)
	}{
		{
			name: "successful format",
			setup: func(t *testing.T, root string) {
				if err := os.WriteFile(filepath.Join(root, "SomeClass.java"), []byte("public class SomeClass {}"), 0644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:  "no files found",
			setup: func(t *testing.T, root string) {},
		},
		{
			name: "nested files in subdirectories",
			setup: func(t *testing.T, root string) {
				dir := filepath.Join(root, "sub", "dir")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "Nested.java"), []byte("public class Nested {}"), 0644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "files in excluded samples path are ignored",
			setup: func(t *testing.T, root string) {
				dir := filepath.Join(root, "samples", "snippets", "generated")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				// This file should NOT be passed to the formatter.
				if err := os.WriteFile(filepath.Join(dir, "Ignored.java"), []byte("public class Ignored {}"), 0644); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			test.setup(t, tmpDir)
			if err := Format(t.Context(), &config.Library{Output: tmpDir}); err != nil {
				t.Errorf("Format() error = %v, want nil", err)
			}
		})
	}
}

func TestFormat_LookPathError(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "SomeClass.java"), []byte("public class SomeClass {}"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "")
	err := Format(t.Context(), &config.Library{Output: tmpDir})
	if err == nil {
		t.Fatal("Format() error = nil, want error")
	}
}

func TestCollectJavaFiles(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	// Create a mix of files
	filesToCreate := []string{
		"Root.java",
		"subdir/Nested.java",
		"subdir/NotJava.txt",
		"samples/snippets/generated/Ignored.java",
		"another/dir/More.java",
	}
	for _, f := range filesToCreate {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{
		filepath.Join(tmpDir, "Root.java"),
		filepath.Join(tmpDir, "subdir", "Nested.java"),
		filepath.Join(tmpDir, "another", "dir", "More.java"),
	}
	got, err := collectJavaFiles(tmpDir)
	if err != nil {
		t.Fatalf("collectJavaFiles() error = %v", err)
	}
	sort.Strings(got)
	sort.Strings(want)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("collectJavaFiles() mismatch (-want +got):\n%s", diff)
	}
}
