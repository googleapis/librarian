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

package librarian

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractZip(t *testing.T) {
	mockZip, err := createMockZip(t)
	if err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "mock.zip")
	if err := os.WriteFile(zipPath, mockZip, 0644); err != nil {
		t.Fatal(err)
	}
	destDir := t.TempDir()
	if err := extractZip(zipPath, destDir); err != nil {
		t.Fatal(err)
	}
	expectedFiles := []string{
		filepath.Join(destDir, "bin", "protoc"),
		filepath.Join(destDir, "include", "google", "protobuf", "any.proto"),
	}
	for _, expected := range expectedFiles {
		if _, err := os.Stat(expected); err != nil {
			t.Errorf("expected file %q was not extracted: %v", expected, err)
		}
	}

	unexpectedFiles := []string{
		filepath.Join(destDir, "some_other_file.txt"),
	}
	for _, unexpected := range unexpectedFiles {
		if _, err := os.Stat(unexpected); err == nil {
			t.Errorf("unexpected file %q exists in destination directory", unexpected)
		}
	}
}

func createMockZip(t *testing.T) ([]byte, error) {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	files := []struct {
		Name, Body string
	}{
		{"bin/protoc", "mock protoc binary"},
		{"include/google/protobuf/any.proto", "mock any proto"},
		{"some_other_file.txt", "should be ignored"},
	}
	for _, file := range files {
		f, err := w.Create(file.Name)
		if err != nil {
			return nil, err
		}
		_, err = f.Write([]byte(file.Body))
		if err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
