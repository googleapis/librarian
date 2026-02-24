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
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestMoveAndMerge_Success(t *testing.T) {
	t.Parallel()
	src, dst := t.TempDir(), t.TempDir()
	// Setup: src contains files and dirs, dst has one collision.
	writeFile := func(dir, path, content string) {
		p := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(src, "file1.txt", "file1")
	writeFile(src, "dir1/file2.txt", "file2")
	writeFile(src, "dir2/file3.txt", "file3")
	writeFile(dst, "dir1/existing.txt", "existing")

	if err := MoveAndMerge(src, dst); err != nil {
		t.Fatalf("MoveAndMerge() error = %v", err)
	}

	// Verify destination: all files moved or merged.
	checkDir := func(dir string, want map[string]string) {
		got := make(map[string]string)
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(dir, path)
			b, _ := os.ReadFile(path)
			got[rel] = string(b)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch in %s (-want +got):\n%s", dir, diff)
		}
	}
	checkDir(dst, map[string]string{
		"file1.txt":         "file1",
		"dir1/file2.txt":    "file2",
		"dir1/existing.txt": "existing",
		"dir2/file3.txt":    "file3",
	})
	// Verify source: all entries should be gone from the source directory.
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected source directory to be empty, but it has %d entries", len(entries))
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

func TestCopyFile_Success(t *testing.T) {
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

func TestUnzip_Success(t *testing.T) {
	t.Parallel()
	testhelper.RequireCommand(t, "unzip")
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "test.zip")
	destDir := filepath.Join(tmp, "dest")
	files := map[string]string{
		"file1.txt":     "content1",
		"sub/file2.txt": "content2",
	}
	createZip(t, zipPath, files, nil)
	if err := Unzip(t.Context(), zipPath, destDir); err != nil {
		t.Fatalf("Unzip() error = %v", err)
	}
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(destDir, name))
		if err != nil {
			t.Errorf("failed to read %s: %v", name, err)
			continue
		}
		if diff := cmp.Diff(want, string(got)); diff != "" {
			t.Errorf("content mismatch for %s (-want +got):\n%s", name, diff)
		}
	}
}

func TestUnzip_Permissions(t *testing.T) {
	t.Parallel()
	testhelper.RequireCommand(t, "unzip")
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "perm.zip")
	destDir := filepath.Join(tmp, "dest")
	files := map[string]string{
		"dir/":    "",
		"exec.sh": "#!/bin/sh\necho hi",
	}
	modes := map[string]os.FileMode{
		"dir/":    0755,
		"exec.sh": 0755,
	}
	createZip(t, zipPath, files, modes)

	if err := Unzip(t.Context(), zipPath, destDir); err != nil {
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

func createZip(t *testing.T, path string, files map[string]string, modes map[string]os.FileMode) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()

	for name, content := range files {
		h := &zip.FileHeader{Name: name}
		if m, ok := modes[name]; ok {
			h.SetMode(m)
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
}
