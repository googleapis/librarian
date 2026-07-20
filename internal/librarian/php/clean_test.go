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

package php

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestClean(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		keep        []string
		setupFiles  []string
		contentMap  map[string]string
		wantDeleted []string
	}{
		{
			name: "gapic metadata is deleted",
			setupFiles: []string{
				"src/V1/gapic_metadata.json",
				"src/gapic_metadata.json",
				"gapic_metadata.json", // in root, won't be walked, so kept
			},
			wantDeleted: []string{
				"src/V1/gapic_metadata.json",
				"src/gapic_metadata.json",
			},
		},
		{
			name: "php files are deleted only if they have markers",
			setupFiles: []string{
				"src/V1/Client/ServiceClient.php",
				"src/V1/ServiceClientTest.php",
				"metadata/V1/Service.php",
				"src/V1/Handwritten.php",
				"tests/Unit/HandwrittenTest.php",
			},
			contentMap: map[string]string{
				"src/V1/Client/ServiceClient.php": "<?php\n// " + string(gapicMarker) + "\nclass ServiceClient {}",
				"src/V1/ServiceClientTest.php":    "<?php\n// " + string(gapicMarker) + "\nclass ServiceClientTest {}",
				"metadata/V1/Service.php":         "<?php\n// " + string(protobufMarker) + "\nclass Service {}",
				"src/V1/Handwritten.php":          "<?php\nclass Handwritten {}",
				"tests/Unit/HandwrittenTest.php":  "<?php\nclass HandwrittenTest {}",
			},
			wantDeleted: []string{
				"src/V1/Client/ServiceClient.php",
				"src/V1/ServiceClientTest.php",
				"metadata/V1/Service.php",
			},
		},
		{
			name: "obey keep list",
			setupFiles: []string{
				"src/V1/Client/ServiceClient.php",
				"src/V1/gapic_metadata.json",
				"VERSION",
			},
			contentMap: map[string]string{
				"src/V1/Client/ServiceClient.php": "<?php\n// " + string(gapicMarker) + "\nclass ServiceClient {}",
			},
			keep: []string{
				"src/V1/Client/ServiceClient.php",
				"src/V1/gapic_metadata.json",
				"VERSION",
			},
			wantDeleted: nil, // everything in keep is preserved
		},
		{
			name: "other directories are not cleaned",
			setupFiles: []string{
				"other/V1/ServiceClient.php",
			},
			contentMap: map[string]string{
				"other/V1/ServiceClient.php": "<?php\n// " + string(gapicMarker) + "\nclass ServiceClient {}",
			},
			wantDeleted: nil, // 'other' is not in directoriesToClean
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			lib := &config.Library{
				Name:   "test",
				Output: filepath.Join(repoRoot, "test"),
				Keep:   test.keep,
			}
			for _, file := range test.setupFiles {
				createFileAndDirectories(t, filepath.Join(lib.Output, file), test.contentMap[file])
			}
			if err := Clean(lib); err != nil {
				t.Fatal(err)
			}
			verifyFileDeletions(t, lib.Output, test.setupFiles, test.wantDeleted)
		})
	}
}

func verifyFileDeletions(t *testing.T, dir string, setupFiles, wantDeleted []string) {
	t.Helper()
	for _, file := range setupFiles {
		fullPath := filepath.Join(dir, file)
		_, err := os.Stat(fullPath)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			t.Fatal(err)
		}
		got := err != nil
		want := slices.Contains(wantDeleted, file)
		if got != want {
			t.Errorf("file %s deleted: got %t, want %t", file, got, want)
		}
	}
}

func createFileAndDirectories(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestClean_StatError(t *testing.T) {
	t.Parallel()
	repoRoot := t.TempDir()
	lib := &config.Library{
		Name:   "test",
		Output: filepath.Join(repoRoot, "test"),
	}
	dir := filepath.Join(lib.Output, "src")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(lib.Output, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(lib.Output, 0755)
	})
	err := Clean(lib)
	if err == nil {
		t.Error("Clean() expected error, got nil")
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Errorf("Clean() error = %v, want os.ErrPermission", err)
	}
}

func TestClean_ReadFileError(t *testing.T) {
	t.Parallel()
	repoRoot := t.TempDir()
	lib := &config.Library{
		Name:   "test",
		Output: filepath.Join(repoRoot, "test"),
	}
	filePath := filepath.Join(lib.Output, "src/V1/Service.php")
	createFileAndDirectories(t, filePath, "<?php // "+string(gapicMarker))
	if err := os.Chmod(filePath, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filePath, 0644)
	})
	err := Clean(lib)
	if err == nil {
		t.Error("Clean() expected error, got nil")
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Errorf("Clean() error = %v, want os.ErrPermission", err)
	}
}

func TestClean_RemoveError(t *testing.T) {
	t.Parallel()
	repoRoot := t.TempDir()
	lib := &config.Library{
		Name:   "test",
		Output: filepath.Join(repoRoot, "test"),
	}
	dirPath := filepath.Join(lib.Output, "src/V1")
	filePath := filepath.Join(dirPath, gapicMetadataFile)
	createFileAndDirectories(t, filePath, "{}")
	if err := os.Chmod(dirPath, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(dirPath, 0755)
	})
	err := Clean(lib)
	if err == nil {
		t.Error("Clean() expected error, got nil")
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Errorf("Clean() error = %v, want os.ErrPermission", err)
	}
}

func TestClean_WalkDirError(t *testing.T) {
	t.Parallel()
	repoRoot := t.TempDir()
	lib := &config.Library{
		Name:   "test",
		Output: filepath.Join(repoRoot, "test"),
	}
	dir := filepath.Join(lib.Output, "src")
	subdir := filepath.Join(dir, "V1")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(subdir, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(subdir, 0755)
	})
	err := Clean(lib)
	if err == nil {
		t.Error("Clean() expected error, got nil")
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Errorf("Clean() error = %v, want os.ErrPermission", err)
	}
}
