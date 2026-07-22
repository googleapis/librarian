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

package ruby

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestClean(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		files     []string
		keep      []string
		wantFiles []string
	}{
		{
			name:      "directory does not exist",
			files:     nil,
			keep:      nil,
			wantFiles: nil,
		},
		{
			name:      "removes generated root files and directories, leaving CHANGELOG.md and custom root files",
			files:     []string{"CHANGELOG.md", "CHANGES.md", "README.md", "AUTHENTICATION.md", "google-cloud-secret_manager-v1.gemspec", "lib/foo.rb", "lib/bar.rb", "proto_docs/doc.rb", "snippets/s1.rb", "test/test_foo.rb"},
			keep:      []string{"README.md"},
			wantFiles: []string{"CHANGELOG.md", "CHANGES.md", "README.md"},
		},
		{
			name:      "removes generated root files except keep list",
			files:     []string{"CHANGELOG.md", "README.md", "AUTHENTICATION.md", "lib/foo.rb"},
			keep:      []string{"README.md"},
			wantFiles: []string{"CHANGELOG.md", "README.md"},
		},
		{
			name:      "keep is nil",
			files:     []string{"CHANGELOG.md", "README.md", "AUTHENTICATION.md", "lib/foo.rb"},
			keep:      nil,
			wantFiles: []string{"CHANGELOG.md"},
		},
		{
			name:      "keep file does not exist",
			files:     []string{"CHANGELOG.md", "README.md", "AUTHENTICATION.md", "lib/foo.rb"},
			keep:      []string{"missing.rb"},
			wantFiles: []string{"CHANGELOG.md"},
		},
		{
			name:      "keep directory preserves nested files",
			files:     []string{"CHANGELOG.md", "snippets/s1.rb", "lib/gen.rb"},
			keep:      []string{"snippets"},
			wantFiles: []string{"CHANGELOG.md", "snippets/s1.rb"},
		},
		{
			name:      "removes empty directories in subdirectories",
			files:     []string{"lib/google/cloud/v1/gen.rb", "CHANGELOG.md"},
			keep:      nil,
			wantFiles: []string{"CHANGELOG.md"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			targetDir := filepath.Join(dir, "lib_out")
			if test.files != nil {
				for _, file := range test.files {
					path := filepath.Join(targetDir, file)
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}
			lib := &config.Library{
				Name:   "google-cloud-test",
				Output: targetDir,
				Keep:   test.keep,
			}
			if err := Clean(lib); err != nil {
				t.Fatal(err)
			}
			var gotFiles []string
			if test.files != nil {
				err := filepath.WalkDir(targetDir, func(path string, entry fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if entry.IsDir() {
						return nil
					}
					rel, err := filepath.Rel(targetDir, path)
					if err != nil {
						return err
					}
					gotFiles = append(gotFiles, filepath.ToSlash(rel))
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			slices.Sort(gotFiles)
			slices.Sort(test.wantFiles)
			if diff := cmp.Diff(test.wantFiles, gotFiles); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestClean_Error(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		lib     *config.Library
		setup   func(t *testing.T, targetDir string)
		wantErr error
	}{
		{
			name: "output is a file",
			lib: &config.Library{
				Name: "test",
			},
			setup: func(t *testing.T, targetDir string) {
				if err := os.WriteFile(targetDir, []byte("file"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: errNotADirectory,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			targetDir := filepath.Join(dir, "lib_out")
			if test.setup != nil {
				test.setup(t, targetDir)
			}
			test.lib.Output = targetDir
			gotErr := Clean(test.lib)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Clean() error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestCleanSubdirectory(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		subDir    string
		files     []string
		keep      []string
		wantFiles []string
	}{
		{
			name:      "subdirectory does not exist",
			subDir:    "nonexistent",
			files:     nil,
			keep:      nil,
			wantFiles: nil,
		},
		{
			name:      "removes unkept files in subdirectory",
			subDir:    "lib",
			files:     []string{"lib/gen1.rb", "lib/gen2.rb", "lib/custom.rb"},
			keep:      []string{"lib/custom.rb"},
			wantFiles: []string{"lib/custom.rb"},
		},
		{
			name:      "removes empty nested directories bottom up",
			subDir:    "lib",
			files:     []string{"lib/google/cloud/v1/gen.rb"},
			keep:      nil,
			wantFiles: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			targetSubDir := filepath.Join(dir, test.subDir)
			if test.files != nil {
				for _, file := range test.files {
					path := filepath.Join(dir, file)
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}
			keepSet := make(map[string]bool)
			for _, keepPath := range test.keep {
				keepSet[keepPath] = true
			}
			if err := cleanSubdirectory(dir, targetSubDir, keepSet); err != nil {
				t.Fatal(err)
			}
			var gotFiles []string
			if test.files != nil {
				err := filepath.WalkDir(targetSubDir, func(path string, entry fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if entry.IsDir() {
						return nil
					}
					rel, err := filepath.Rel(dir, path)
					if err != nil {
						return err
					}
					gotFiles = append(gotFiles, filepath.ToSlash(rel))
					return nil
				})
				if err != nil && !errors.Is(err, fs.ErrNotExist) {
					t.Fatal(err)
				}
			}
			slices.Sort(gotFiles)
			slices.Sort(test.wantFiles)
			if diff := cmp.Diff(test.wantFiles, gotFiles); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsKept(t *testing.T) {
	keepSet := map[string]bool{
		"README.md":          true,
		"snippets":           true,
		"lib/custom/file.rb": true,
	}
	for _, test := range []struct {
		name     string
		relSlash string
		want     bool
	}{
		{
			name:     "exact file match",
			relSlash: "README.md",
			want:     true,
		},
		{
			name:     "file in kept directory",
			relSlash: "snippets/v1/sample.rb",
			want:     true,
		},
		{
			name:     "nested file match",
			relSlash: "lib/custom/file.rb",
			want:     true,
		},
		{
			name:     "unkept file",
			relSlash: "lib/google/cloud/client.rb",
			want:     false,
		},
		{
			name:     "unkept root file",
			relSlash: "AUTHENTICATION.md",
			want:     false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isKept(test.relSlash, keepSet)
			if got != test.want {
				t.Errorf("isKept(%q) = %v, want %v", test.relSlash, got, test.want)
			}
		})
	}
}
