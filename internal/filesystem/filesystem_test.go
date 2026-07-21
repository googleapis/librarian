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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestMoveAndMerge_Success(t *testing.T) {
	t.Parallel()
	src, dst := t.TempDir(), t.TempDir()
	// Setup: src contains files and dirs, dst has no collisions.
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
		t.Helper()
		got := make(map[string]string)
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				t.Fatalf("failed to get relative path for %s: %v", path, err)
			}
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read file %s: %v", path, err)
			}
			got[rel] = string(b)
			return nil
		})
		if err != nil {
			t.Fatalf("failed to walk directory %s: %v", dir, err)
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

func TestMoveAndMerge_FileCollisionError(t *testing.T) {
	t.Parallel()
	src, dst := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "file.txt"), []byte("src"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "file.txt"), []byte("dst"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := MoveAndMerge(src, dst); err == nil {
		t.Error("MoveAndMerge() expected error for file collision, got nil")
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
	// Create a directory with the same name as src file in destination
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

func TestMoveAndMerge_SameSourceAndTarget(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "file.txt"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := MoveAndMerge(dir, dir); err == nil {
		t.Error("MoveAndMerge() expected error when source and destination are the same, got nil")
	}
}

func TestMoveAndMergeWithKeep(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name         string
		srcFiles     map[string]string
		dstFiles     map[string]string
		keepFunc     func(string) bool
		wantDstFiles map[string]string
	}{
		{
			name:     "overwrite when keepFunc nil",
			srcFiles: map[string]string{"file.txt": "new content", "sub/new.txt": "sub content"},
			dstFiles: map[string]string{"file.txt": "old content"},
			keepFunc: nil,
			wantDstFiles: map[string]string{
				"file.txt":    "new content",
				"sub/new.txt": "sub content",
			},
		},
		{
			name:     "preserve when keepFunc true",
			srcFiles: map[string]string{"README.md": "new readme", "code.go": "new code"},
			dstFiles: map[string]string{"README.md": "custom readme", "old.go": "old code"},
			keepFunc: func(rel string) bool { return rel == "README.md" },
			wantDstFiles: map[string]string{
				"README.md": "custom readme",
				"code.go":   "new code",
				"old.go":    "old code",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			src, dst := filepath.Join(root, "src"), filepath.Join(root, "dst")
			writeTestFiles(t, src, test.srcFiles)
			writeTestFiles(t, dst, test.dstFiles)

			if err := MoveAndMergeWithKeep(src, dst, dst, test.keepFunc); err != nil {
				t.Fatal(err)
			}
			checkDir(t, dst, test.wantDstFiles)
			checkDir(t, src, map[string]string{})
		})
	}
}

func TestMoveAndMergeWithKeep_Error(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		setupFunc func(t *testing.T, root string) (src, dst string)
		wantErr   error
	}{
		{
			name: "non-existent source directory",
			setupFunc: func(t *testing.T, root string) (string, string) {
				return "/non/existent/path", filepath.Join(root, "dst")
			},
			wantErr: fs.ErrNotExist,
		},
		{
			name: "read-only destination directory",
			setupFunc: func(t *testing.T, root string) (string, string) {
				src, dst := filepath.Join(root, "src"), filepath.Join(root, "dst")
				writeTestFiles(t, src, map[string]string{"file.txt": "content"})
				if err := os.MkdirAll(dst, 0444); err != nil {
					t.Fatal(err)
				}
				return src, dst
			},
			wantErr: fs.ErrPermission,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			src, dst := test.setupFunc(t, root)
			if err := MoveAndMergeWithKeep(src, dst, dst, nil); !errors.Is(err, test.wantErr) {
				t.Errorf("MoveAndMergeWithKeep() error = %v, want %v", err, test.wantErr)
			}
		})
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

// writeTestFiles creates directories and writes files to dir.
func writeTestFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for path, content := range files {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

// checkDir verifies that the files inside dir exactly match want using go-cmp.
func checkDir(t *testing.T, dir string, want map[string]string) {
	t.Helper()
	got := make(map[string]string)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			t.Fatalf("failed to get relative path for %s: %v", path, err)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file %s: %v", path, err)
		}
		got[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk directory %s: %v", dir, err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in %s (-want +got):\n%s", dir, diff)
	}
}

func TestRemoveEmptyDirs(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		setupDirs  []string
		setupFiles map[string]string
		keepFunc   func(string) bool
		want       []string
	}{
		{
			name:      "empty dirs are removed",
			setupDirs: []string{"dir1/dir2/dir3"},
			want:      nil,
		},
		{
			name:      "non-empty dirs are preserved",
			setupDirs: []string{"dir1/dir2"},
			setupFiles: map[string]string{
				"dir1/dir2/file.txt": "content",
			},
			want: []string{
				"dir1",
				"dir1/dir2",
				"dir1/dir2/file.txt",
			},
		},
		{
			name:      "partially empty dirs are cleaned",
			setupDirs: []string{"dir1/dir2", "dir1/dir3"},
			setupFiles: map[string]string{
				"dir1/dir3/file.txt": "content",
			},
			want: []string{
				"dir1",
				"dir1/dir3",
				"dir1/dir3/file.txt",
			},
		},
		{
			name:      "kept dirs are preserved",
			setupDirs: []string{"dir1/dir2/dir3"},
			keepFunc: func(rel string) bool {
				return rel == "dir1/dir2"
			},
			want: []string{
				"dir1",
				"dir1/dir2",
				"dir1/dir2/dir3",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for _, d := range test.setupDirs {
				if err := os.MkdirAll(filepath.Join(root, d), 0755); err != nil {
					t.Fatal(err)
				}
			}
			for f, content := range test.setupFiles {
				p := filepath.Join(root, f)
				if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := RemoveEmptyDirs(root, root, test.keepFunc); err != nil {
				t.Fatal(err)
			}
			var got []string
			if _, err := os.Stat(root); err == nil {
				err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if path == root {
						return nil
					}
					rel, err := filepath.Rel(root, path)
					if err != nil {
						return err
					}
					got = append(got, filepath.ToSlash(rel))
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			slices.Sort(got)
			slices.Sort(test.want)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
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
