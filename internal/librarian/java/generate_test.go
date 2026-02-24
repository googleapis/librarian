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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"
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

func TestClean(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	libraryName := "google-cloud-secretmanager"
	version := "v1"
	// Create directories to clean
	dirs := []string{
		filepath.Join(tmpDir, libraryName, "src"),
		filepath.Join(tmpDir, fmt.Sprintf("proto-%s-%s", libraryName, version), "src"),
		filepath.Join(tmpDir, fmt.Sprintf("grpc-%s-%s", libraryName, version), "src"),
		filepath.Join(tmpDir, "samples", "snippets", "generated"),
		filepath.Join(tmpDir, "kept-dir"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	// Create files
	files := []string{
		filepath.Join(tmpDir, libraryName, "src", "Main.java"),
		filepath.Join(tmpDir, libraryName, "src", "test", "java", "com", "google", "cloud", "secretmanager", "v1", "it", "ITSecretManagerTest.java"),
		filepath.Join(tmpDir, "kept-file.txt"),
		filepath.Join(tmpDir, "kept-dir", "file.txt"),
	}
	for _, file := range files {
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	lib := &config.Library{
		Name:   "secretmanager",
		Output: tmpDir,
		Keep:   []string{"kept-file.txt", "kept-dir"},
	}
	if err := Clean(lib); err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	// Verify cleaned paths
	cleanedPaths := []string{
		filepath.Join(tmpDir, libraryName, "src", "Main.java"),
		filepath.Join(tmpDir, fmt.Sprintf("proto-%s-%s", libraryName, version)),
		filepath.Join(tmpDir, fmt.Sprintf("grpc-%s-%s", libraryName, version)),
		filepath.Join(tmpDir, "samples", "snippets", "generated"),
	}
	for _, p := range cleanedPaths {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("expected path %s to be removed, but it still exists", p)
		}
	}
	// Verify kept paths
	keptPaths := []string{
		filepath.Join(tmpDir, "kept-file.txt"),
		filepath.Join(tmpDir, "kept-dir", "file.txt"),
		filepath.Join(tmpDir, libraryName, "src", "test", "java", "com", "google", "cloud", "secretmanager", "v1", "it", "ITSecretManagerTest.java"),
	}
	for _, p := range keptPaths {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected path %s to be kept, but it was removed: %v", p, err)
		}
	}
}

func TestIsDirNotEmpty(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "generic error",
			err:  errors.New("generic error"),
			want: false,
		},
		{
			name: "ENOTEMPTY",
			err:  &os.PathError{Op: "remove", Path: "/tmp", Err: syscall.ENOTEMPTY},
			want: true,
		},
		{
			name: "EEXIST",
			err:  &os.PathError{Op: "remove", Path: "/tmp", Err: syscall.EEXIST},
			want: true,
		},
		{
			name: "EACCES",
			err:  &os.PathError{Op: "remove", Path: "/tmp", Err: syscall.EACCES},
			want: false,
		},
		{
			name: "wrapped ENOTEMPTY",
			err:  fmt.Errorf("failed: %w", &os.PathError{Op: "remove", Path: "/tmp", Err: syscall.ENOTEMPTY}),
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isDirNotEmpty(test.err)
			if got != test.want {
				t.Errorf("isDirNotEmpty(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
}
