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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
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

func TestFormat(t *testing.T) {
	library := &config.Library{Name: "test-lib"}

	if err := Format(t.Context(), library); err != nil {
		t.Errorf("Format() error = %v, want nil", err)
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
