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

package filesystem

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMoveAndMerge(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	dst := t.TempDir()

	// Setup source:
	// src/file1.txt
	// src/dir1/file2.txt
	// src/dir2/file3.txt
	if err := os.WriteFile(filepath.Join(src, "file1.txt"), []byte("file1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(src, "dir1"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "dir1", "file2.txt"), []byte("file2"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(src, "dir2"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "dir2", "file3.txt"), []byte("file3"), 0644); err != nil {
		t.Fatal(err)
	}

	// Setup destination with a collision:
	// dst/dir1/existing.txt
	if err := os.Mkdir(filepath.Join(dst, "dir1"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "dir1", "existing.txt"), []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := MoveAndMerge(src, dst); err != nil {
		t.Fatalf("MoveAndMerge() error = %v", err)
	}

	// Verify results:
	// dst/file1.txt
	// dst/dir1/file2.txt
	// dst/dir1/existing.txt
	// dst/dir2/file3.txt
	tests := []struct {
		path string
		want string
	}{
		{filepath.Join(dst, "file1.txt"), "file1"},
		{filepath.Join(dst, "dir1", "file2.txt"), "file2"},
		{filepath.Join(dst, "dir1", "existing.txt"), "existing"},
		{filepath.Join(dst, "dir2", "file3.txt"), "file3"},
	}

	for _, tt := range tests {
		got, err := os.ReadFile(tt.path)
		if err != nil {
			t.Errorf("failed to read %s: %v", tt.path, err)
			continue
		}
		if diff := cmp.Diff(tt.want, string(got)); diff != "" {
			t.Errorf("content mismatch at %s (-want +got):\n%s", tt.path, diff)
		}
	}

	// Verify source entries:
	// file1.txt (file) should be gone.
	// dir1 (merged directory) should still exist (but be empty).
	// dir2 (renamed directory) should be gone.
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == "file1.txt" || entry.Name() == "dir2" {
			t.Errorf("expected %s to be gone from source", entry.Name())
		}
		if entry.Name() == "dir1" {
			subEntries, err := os.ReadDir(filepath.Join(src, "dir1"))
			if err != nil {
				t.Fatalf("ReadDir failed: %v", err)
			}
			if len(subEntries) != 0 {
				t.Errorf("expected merged directory dir1 to be empty, but it has %d entries", len(subEntries))
			}
		}
	}
}

func TestCopyFile(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "dst.txt")
	content := "hello world"

	if err := os.WriteFile(src, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile() error = %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Errorf("CopyFile() got = %q, want %q", string(got), content)
	}
}

func TestUnzip(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name    string
		files   map[string]string
		wantErr bool
	}{
		{
			name: "basic extraction",
			files: map[string]string{
				"file1.txt":     "content1",
				"sub/file2.txt": "content2",
			},
		},
		{
			name: "zip slip protection",
			files: map[string]string{
				"../../outside.txt": "danger",
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmp := t.TempDir()
			zipPath := filepath.Join(tmp, "test.zip")
			destDir := filepath.Join(tmp, "dest")

			// Create a zip file
			f, err := os.Create(zipPath)
			if err != nil {
				t.Fatal(err)
			}
			zw := zip.NewWriter(f)

			for name, content := range test.files {
				// Use CreateHeader to allow testing invalid names like "../../"
				h := &zip.FileHeader{Name: name}
				w, err := zw.CreateHeader(h)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := w.Write([]byte(content)); err != nil {
					t.Fatal(err)
				}
			}
			zw.Close()
			f.Close()

			err = Unzip(zipPath, destDir)
			if (err != nil) != test.wantErr {
				t.Fatalf("Unzip() error = %v, wantErr %v", err, test.wantErr)
			}

			if !test.wantErr {
				for name, want := range test.files {
					got, err := os.ReadFile(filepath.Join(destDir, name))
					if err != nil {
						t.Errorf("failed to read %s: %v", name, err)
						continue
					}
					if string(got) != want {
						t.Errorf("content mismatch for %s: got %q, want %q", name, string(got), want)
					}
				}
			}
		})
	}
}

func TestMoveAndMerge_ReadDirError(t *testing.T) {
	t.Parallel()
	if err := MoveAndMerge("/non/existent/path", t.TempDir()); err == nil {
		t.Error("MoveAndMerge() expected error for non-existent source, got nil")
	}
}

func TestMoveAndMerge_RenameError(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	dst := t.TempDir()

	if err := os.WriteFile(filepath.Join(src, "file"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create a directory with the same name in destination
	if err := os.Mkdir(filepath.Join(dst, "file"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := MoveAndMerge(src, dst); err == nil {
		t.Error("MoveAndMerge() expected error when renaming file to directory, got nil")
	}
}

func TestMoveAndMerge_RecursiveError(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	dst := t.TempDir()

	// Create src/dir/file
	if err := os.MkdirAll(filepath.Join(src, "dir"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "dir", "file"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create dst/dir
	if err := os.MkdirAll(filepath.Join(dst, "dir"), 0755); err != nil {
		t.Fatal(err)
	}

	// Make src/dir unreadable to cause ReadDir failure inside MoveAndMerge
	if err := os.Chmod(filepath.Join(src, "dir"), 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(filepath.Join(src, "dir"), 0755) // cleanup for TempDir

	if err := MoveAndMerge(src, dst); err == nil {
		t.Error("MoveAndMerge() expected error for recursive failure, got nil")
	}
}

func TestCopyFile_Error(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	if err := CopyFile("/non/existent/src", filepath.Join(tmp, "dst")); err == nil {
		t.Error("CopyFile() expected error for non-existent source, got nil")
	}

	src := filepath.Join(tmp, "src")
	if err := os.WriteFile(src, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	// Try to create file in a non-existent directory
	if err := CopyFile(src, "/non/existent/dir/dst"); err == nil {
		t.Error("CopyFile() expected error for invalid destination, got nil")
	}
}

func TestUnzip_Error(t *testing.T) {
	t.Parallel()
	if err := Unzip("/non/existent/zip", t.TempDir()); err == nil {
		t.Error("Unzip() expected error for non-existent zip, got nil")
	}
}

func TestUnzip_MkdirAllError(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "test.zip")
	destDir := filepath.Join(tmp, "dest")

	// Create a file where a directory should be
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "file"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a zip with a file that would need to create a directory where "file" is
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("file/inner")
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("content"))
	zw.Close()
	f.Close()

	if err := Unzip(zipPath, destDir); err == nil {
		t.Error("Unzip() expected error for MkdirAll failure, got nil")
	}
}

func TestUnzip_OpenFileError(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "test.zip")
	destDir := filepath.Join(tmp, "dest")

	// Create a directory where a file should be
	if err := os.MkdirAll(filepath.Join(destDir, "file"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create a zip with a file named "file"
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("file")
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("content"))
	zw.Close()
	f.Close()

	if err := Unzip(zipPath, destDir); err == nil {
		t.Error("Unzip() expected error for OpenFile failure, got nil")
	}
}

func TestUnzip_Permissions(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "perm.zip")
	destDir := filepath.Join(tmp, "dest")

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)

	// Create a directory and an executable file in the zip
	hDir := &zip.FileHeader{Name: "dir/"}
	hDir.SetMode(0755)
	if _, err := zw.CreateHeader(hDir); err != nil {
		t.Fatal(err)
	}

	h := &zip.FileHeader{Name: "exec.sh"}
	h.SetMode(0755)
	w, err := zw.CreateHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("#!/bin/sh\necho hi"))
	zw.Close()
	f.Close()

	if err := Unzip(zipPath, destDir); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(destDir, "dir"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Errorf("expected directory to be traversable, got %v", info.Mode().Perm())
	}

	info, err = os.Stat(filepath.Join(destDir, "exec.sh"))
	if err != nil {
		t.Fatal(err)
	}
	// Check if the executable bit is set (0111)
	if info.Mode()&0111 == 0 {
		t.Errorf("expected file to be executable, got mode %v", info.Mode())
	}
}
