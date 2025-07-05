// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package librarian

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func setupTestFile(t *testing.T, content []byte, createDir bool) string {
	t.Helper()
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")
	if !createDir {
		filePath = filepath.Join(tempDir, "non-existent-dir", "test.txt")
	}
	if content != nil {
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}
	return filePath
}

func TestReadAllBytesFromFile(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		content []byte
		wantErr bool
	}{
		{"success", []byte("hello world"), false},
		{"empty file", []byte(""), false},
		{"non-existent file", nil, true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			filePath := setupTestFile(t, tc.content, true)

			got, err := readAllBytesFromFile(filePath)
			if (err != nil) != tc.wantErr {
				t.Fatalf("readAllBytesFromFile() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !tc.wantErr {
				if diff := cmp.Diff(tc.content, got); diff != "" {
					t.Errorf("readAllBytesFromFile() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestAppendToFile(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name            string
		initialContent  []byte
		appendedContent string
		expectedContent string
		createDir       bool
		wantErr         bool
	}{
		{"success", []byte("hello"), " world", "hello world", true, false},
		{"empty initial content", []byte(""), "hello", "hello", true, false},
		{"empty appended content", []byte("hello"), "", "hello", true, false},
		{"append to non-existent file", nil, "hello", "hello", true, false},
		{"non-existent dir", nil, "hello", "", false, true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			filePath := setupTestFile(t, tc.initialContent, tc.createDir)

			err := appendToFile(filePath, tc.appendedContent)
			if (err != nil) != tc.wantErr {
				t.Fatalf("appendToFile() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !tc.wantErr {
				got, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatalf("ReadFile() error = %v", err)
				}
				if string(got) != tc.expectedContent {
					t.Errorf("appendToFile() got %q, want %q", string(got), tc.expectedContent)
				}
			}
		})
	}
}

func TestCreateAndWriteToFile(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name      string
		content   string
		createDir bool
		wantErr   bool
	}{
		{"success", "hello world", true, false},
		{"empty content", "", true, false},
		{"non-existent dir", "hello", false, true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			filePath := setupTestFile(t, nil, tc.createDir)

			err := createAndWriteToFile(filePath, tc.content)
			if (err != nil) != tc.wantErr {
				t.Fatalf("createAndWriteToFile() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !tc.wantErr {
				got, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatalf("ReadFile() error = %v", err)
				}
				if string(got) != tc.content {
					t.Errorf("createAndWriteToFile() got %q, want %q", string(got), tc.content)
				}
			}
		})
	}
}

func TestCreateAndWriteBytesToFile(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name      string
		content   []byte
		createDir bool
		wantErr   bool
	}{
		{"success", []byte("hello world"), true, false},
		{"empty content", []byte(""), true, false},
		{"non-existent dir", []byte("hello"), false, true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			filePath := setupTestFile(t, nil, tc.createDir)

			err := createAndWriteBytesToFile(filePath, tc.content)
			if (err != nil) != tc.wantErr {
				t.Fatalf("createAndWriteBytesToFile() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !tc.wantErr {
				got, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatalf("ReadFile() error = %v", err)
				}
				if diff := cmp.Diff(tc.content, got); diff != "" {
					t.Errorf("createAndWriteBytesToFile() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
