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

package golang

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestClean_Success(t *testing.T) {
	t.Parallel()
	libraryName := "testlib"
	for _, test := range []struct {
		name         string
		outputFiles  []string
		snippetFiles []string
		keep         []string
		nestedModule string
		wantOutput   []string
		wantSnippets []string
	}{
		{
			name: "removes all files",
			outputFiles: []string{
				".repo-metadata.json",
				"auxiliary.go",
				"auxiliary_go123.go",
				"doc.go",
				"library_client.go",
				"library_client_example_go123_test.go",
				"library_client_example_test.go",
				"gapic_metadata.json",
				"helpers.go",
				"nested/content.pb.go",
				"non-generated.go",
			},
			snippetFiles: []string{"snippet1.go", "snippet2.go", "README.md"},
			keep:         []string{},
			wantOutput:   []string{"non-generated.go"},
		},
		{
			name: "remove all files except keep list",
			outputFiles: []string{
				"auxiliary.go",
				"auxiliary_go123.go",
				"doc.go",
			},
			snippetFiles: []string{"snippet1.go"},
			keep:         []string{"doc.go"},
			wantOutput:   []string{"doc.go"},
		},
		{
			name:         "nested module",
			outputFiles:  []string{"nested/doc.go.go"},
			snippetFiles: []string{"nested/snippet.go"},
			keep:         []string{},
			nestedModule: "nested",
			wantOutput:   []string{"nested/doc.go.go"},
			wantSnippets: []string{"internal/generated/snippets/testlib/nested/snippet.go"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			outputPath := filepath.Join(root, libraryName)
			snippetPath := filepath.Join(root, "internal", "generated", "snippets", libraryName)
			lib := &config.Library{
				Name:   libraryName,
				Output: filepath.Join(root),
				Keep:   test.keep,
				Go: &config.GoModule{
					NestedModule: test.nestedModule,
				},
			}
			if test.outputFiles != nil {
				createFiles(t, outputPath, test.outputFiles)
			}
			if test.snippetFiles != nil {
				createFiles(t, snippetPath, test.snippetFiles)
			}

			_, err := Clean(lib)
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
			if !slices.Equal(gotSnippetFiles, test.wantSnippets) {
				t.Errorf("snippet directory: got %v, want %v", gotSnippetFiles, test.wantSnippets)
			}
		})
	}
}

func TestClean_Error(t *testing.T) {
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
			root := t.TempDir()
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

			_, err := Clean(lib)
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
