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
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestCleanOutput(t *testing.T) {
	for _, test := range []struct {
		name    string
		files   []string
		keep    []string
		want    []string
		wantErr bool
	}{
		{
			name:  "removes all except keep list",
			files: []string{"Cargo.toml", "README.md", "src/lib.rs"},
			keep:  []string{"Cargo.toml"},
			want:  []string{"Cargo.toml"},
		},
		{
			name:    "empty directory with keep list",
			files:   []string{},
			keep:    []string{"Cargo.toml"},
			wantErr: true,
		},
		{
			name:  "only kept file",
			files: []string{"Cargo.toml"},
			keep:  []string{"Cargo.toml"},
			want:  []string{"Cargo.toml"},
		},
		{
			name:    "keep file not found",
			files:   []string{"README.md", "src/lib.rs"},
			keep:    []string{"Cargo.toml"},
			wantErr: true,
		},
		{
			name:  "keep multiple files",
			files: []string{"Cargo.toml", "README.md", "src/lib.rs"},
			keep:  []string{"Cargo.toml", "README.md"},
			want:  []string{"Cargo.toml", "README.md"},
		},
		{
			name:  "empty keep list",
			files: []string{"Cargo.toml", "README.md"},
			keep:  []string{},
			want:  []string{},
		},
		{
			name:  "keep nested files",
			files: []string{"Cargo.toml", "README.md", "src/lib.rs", "src/operation.rs", "src/endpoint.rs"},
			keep:  []string{"src/operation.rs", "src/endpoint.rs"},
			want:  []string{"src/endpoint.rs", "src/operation.rs"},
		},
		{
			// While it would definitely be odd to use "./" here, the
			// most common case for canonicalizing is for Windows where
			// the directory separator is a backslash. This test ensures
			// the logic is tested even on Unix.
			name:  "keep entries are canonicalized",
			files: []string{"Cargo.toml", "README.md", "src/lib.rs"},
			keep:  []string{"./Cargo.toml"},
			want:  []string{"Cargo.toml"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range test.files {
				path := filepath.Join(dir, f)
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			err := checkAndClean(dir, test.keep)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, f := range test.files {
				if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
					got = append(got, f)
				}
			}
			slices.Sort(got)
			slices.Sort(test.want)
			if !slices.Equal(got, test.want) {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestCleanGo_Success(t *testing.T) {
	root := t.TempDir()
	libraryName := "testlib"
	for _, test := range []struct {
		name         string
		outputFiles  []string
		snippetFiles []string
		keep         []string
		wantOutput   []string
	}{
		{
			name:         "removes all except keep list",
			outputFiles:  []string{"file1.go", "file2.go", "go.mod"},
			snippetFiles: []string{"snippet1.go", "snippet2.go", "README.md"},
			keep:         []string{"file1.go"},
			wantOutput:   []string{"file1.go"},
		},
		{
			name:         "remove all files",
			outputFiles:  []string{"file1.go", "file2.go"},
			snippetFiles: []string{"snippet1.go"},
			keep:         []string{},
			wantOutput:   []string{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outputPath := filepath.Join(root, libraryName)
			snippetPath := filepath.Join(root, "internal", "generated", "snippets", libraryName)
			lib := &config.Library{
				Name:   libraryName,
				Output: filepath.Join(root),
				Keep:   test.keep,
			}
			if test.outputFiles != nil {
				createFiles(t, outputPath, test.outputFiles)
			}
			if test.snippetFiles != nil {
				createFiles(t, snippetPath, test.snippetFiles)
			}

			_, err := cleanGo(lib)
			if err != nil {
				t.Fatal(err)
			}

			gotOutputFiles := getFilesInDir(t, outputPath, filepath.Join(root, libraryName))
			slices.Sort(gotOutputFiles)
			slices.Sort(test.wantOutput)
			if !slices.Equal(gotOutputFiles, test.wantOutput) {
				t.Errorf("output directory: got %v, want %v", gotOutputFiles, test.wantOutput)
			}

			gotSnippetFiles := getFilesInDir(t, snippetPath, root)
			if !slices.Equal(gotSnippetFiles, []string{}) {
				t.Errorf("snippet directory: got %v, want %v", gotSnippetFiles, []string{})
			}
		})
	}
}

func TestCleanGo_Error(t *testing.T) {
	root := t.TempDir()
	libraryName := "testlib"
	for _, test := range []struct {
		name         string
		outputFiles  []string
		snippetFiles []string
		keep         []string
		wantErrMsg   string
	}{
		{
			name:         "keep file not found in output directory",
			outputFiles:  []string{"file2.go"},
			snippetFiles: []string{"snippet1.go"},
			keep:         []string{"file1.go"},
			wantErrMsg:   "keep file \"file1.go\" does not exist",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outputPath := filepath.Join(root, libraryName)
			snippetPath := filepath.Join(root, "internal", "generated", "snippets", libraryName)
			lib := &config.Library{
				Name:   libraryName,
				Output: filepath.Join(root),
				Keep:   test.keep,
			}
			if test.outputFiles != nil {
				createFiles(t, outputPath, test.outputFiles)
			}
			if test.snippetFiles != nil {
				createFiles(t, snippetPath, test.snippetFiles)
			}

			_, err := cleanGo(lib)
			if err == nil {
				t.Error(err)
				return
			}
			if diff := cmp.Diff(test.wantErrMsg, err.Error()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func createFiles(t *testing.T, base string, files []string) {
	t.Helper()
	if err := os.MkdirAll(base, 0755); err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		filePath := filepath.Join(base, file)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

// getFilesInDir is a test helper to get relative paths of files in a directory.
func getFilesInDir(t *testing.T, dirPath, basePath string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(basePath, path)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk directory %q: %v", dirPath, err)
	}
	return files
}
