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

package postprocessing

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.txt")
	dstPath := filepath.Join(dir, "dst.txt")
	content := "hello copy"
	if err := os.WriteFile(srcPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatal(err)
	}
	gotBytes, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(gotBytes)
	if diff := cmp.Diff(content, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestCopyFile_Error(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "nonexistent.txt")
	dstPath := filepath.Join(dir, "dst.txt")
	err := CopyFile(srcPath, dstPath)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("CopyFile() returned unexpected error: got %v, want %v", err, fs.ErrNotExist)
	}
}

func TestRemoveFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveFile(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("RemoveFile() expected file to be removed, but it still exists")
	}
}

func TestRemoveFile_NonExistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.txt")
	err := RemoveFile(path)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("RemoveFile() returned unexpected error: got %v, want %v", err, fs.ErrNotExist)
	}
}

func TestRemoveFile_Error(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target")
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "sub.txt"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	err := RemoveFile(path)
	if !errors.Is(err, syscall.ENOTEMPTY) {
		t.Errorf("RemoveFile() error = %v, wantErr %v", err, syscall.ENOTEMPTY)
	}
}
