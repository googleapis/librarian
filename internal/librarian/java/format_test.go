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
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestFormat(t *testing.T) {
	testhelper.RequireCommand(t, "google-java-format")
	for _, test := range []struct {
		name  string
		setup func(t *testing.T, root string)
	}{
		{
			name: "successful format",
			setup: func(t *testing.T, root string) {
				if err := os.WriteFile(filepath.Join(root, "SomeClass.java"), []byte("public class SomeClass {}"), 0o644); err != nil {
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
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "Nested.java"), []byte("public class Nested {}"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "files in excluded samples path are ignored",
			setup: func(t *testing.T, root string) {
				dir := filepath.Join(root, "samples", "snippets", "generated")
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				// This file should NOT be passed to the formatter.
				if err := os.WriteFile(filepath.Join(dir, "Ignored.java"), []byte("public class Ignored {}"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			test.setup(t, tmpDir)
			if err := FormatLibraries(t.Context(), []*config.Library{{Output: tmpDir}}); err != nil {
				t.Errorf("FormatLibraries() error = %v, want nil", err)
			}
		})
	}
}

func TestFormat_LookPathError(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "SomeClass.java"), []byte("public class SomeClass {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "")
	err := FormatLibraries(t.Context(), []*config.Library{{Output: tmpDir}})
	if err == nil {
		t.Fatal("expected error, got nil")
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
		"samples/snippets/src/Ignored.java",
	}
	for _, f := range filesToCreate {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
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
		t.Fatal(err)
	}
	sort.Strings(got)
	sort.Strings(want)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFormatLibraries_MultipleLibraries(t *testing.T) {
	testhelper.RequireCommand(t, "google-java-format")
	t.Parallel()
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()
	// Create unformatted files in both directories.
	// They have extra blank lines which should be collapsed.
	content1 := []byte("package test;\n\n\npublic class One {}")
	content2 := []byte("package test;\n\n\npublic class Two {}")
	file1 := filepath.Join(tmpDir1, "One.java")
	file2 := filepath.Join(tmpDir2, "Two.java")
	if err := os.WriteFile(file1, content1, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, content2, 0o644); err != nil {
		t.Fatal(err)
	}
	libraries := []*config.Library{
		{Output: tmpDir1},
		{Output: tmpDir2},
	}
	if err := FormatLibraries(t.Context(), libraries); err != nil {
		t.Fatalf("FormatLibraries() error = %v, want nil", err)
	}
	// Verify both files were formatted (empty lines collapsed).
	wantContent1 := []byte("package test;\n\npublic class One {}\n")
	wantContent2 := []byte("package test;\n\npublic class Two {}\n")
	gotContent1, err := os.ReadFile(file1)
	if err != nil {
		t.Fatal(err)
	}
	gotContent2, err := os.ReadFile(file2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotContent1, wantContent1) {
		t.Errorf("file1 content mismatch\ngot:  %q\nwant: %q", gotContent1, wantContent1)
	}
	if !bytes.Equal(gotContent2, wantContent2) {
		t.Errorf("file2 content mismatch\ngot:  %q\nwant: %q", gotContent2, wantContent2)
	}
}
