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
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestClean(t *testing.T) {
	libraryName := "example"
	for _, test := range []struct {
		name         string
		outputFiles  []string
		snippetFiles []string
		keep         []string
		nestedModule string
		setup        func(dir string)
		wantOutput   []string
		wantSnippets []string
	}{
		{
			name: "removes all generated files",
			outputFiles: []string{
				"apiv1/.repo-metadata.json",
				"apiv1/auxiliary.go",
				"apiv1/auxiliary_go123.go",
				"apiv1/doc.go",
				"apiv1/operations.go",
				"apiv1/library_client.go",
				"apiv1/library_client_example_go123_test.go",
				"apiv1/library_client_example_test.go",
				"apiv1/gapic_metadata.json",
				"apiv1/helpers.go",
				"apiv1/librarypb/content.pb.go",
				"apiv1/non-generated.go",
				"internal/version.go",
				"README.md",
			},
			snippetFiles: []string{
				"apiv1/main.go",
			},
			wantOutput: []string{
				"apiv1/non-generated.go",
			},
		},
		{
			name: "remove all generated files except keep list",
			outputFiles: []string{
				"apiv1/auxiliary.go",
				"apiv1/auxiliary_go123.go",
				"README.md",
			},
			snippetFiles: []string{"apiv1/snippet1.go"},
			keep:         []string{"README.md"},
			wantOutput:   []string{"README.md"},
		},
		{
			name: "client files in nested module are not cleaned",
			outputFiles: []string{
				"nested/apiv1/auxiliary.go",
				"nested/apiv1/auxiliary_go123.go",
			},
			snippetFiles: []string{"nested/apiv1/snippet1.go"},
			// They are not cleaned because these files are not within
			// import path of the GoAPI.
			wantOutput: []string{
				"nested/apiv1/auxiliary.go",
				"nested/apiv1/auxiliary_go123.go",
			},
			wantSnippets: []string{
				"internal/generated/snippets/example/nested/apiv1/snippet1.go",
			},
		},
		{
			name:        "no snippets",
			outputFiles: []string{"apiv1/auxiliary.go"},
		},
		{
			name: "no client files",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			outputPath := filepath.Join(root, libraryName)
			snippetPath := filepath.Join(root, "internal", "generated", "snippets", libraryName)
			lib := &config.Library{
				Name: libraryName,
				APIs: []*config.API{
					{
						Path: "google/example/v1",
					},
				},
				Output: root,
				Keep:   test.keep,
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ClientPackage: libraryName,
							ImportPath:    "example/apiv1",
							Path:          "google/example/v1",
						},
					},
					NestedModule: test.nestedModule,
				},
			}
			if test.outputFiles != nil {
				createFiles(t, outputPath, test.outputFiles)
			}
			if test.snippetFiles != nil {
				createFiles(t, snippetPath, test.snippetFiles)
			}

			err := Clean(lib)
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
		library      *config.Library
		outputFiles  []string
		snippetFiles []string
		wantErr      error
	}{
		{
			name: "keep file not found in output directory",
			library: &config.Library{
				Name: "testlib",
				Keep: []string{"file1.go"},
			},
			outputFiles:  []string{"file2.go"},
			snippetFiles: []string{"snippet1.go"},
			wantErr:      fs.ErrNotExist,
		},
		{
			name: "no go api",
			library: &config.Library{
				Name: "testlib",
				APIs: []*config.API{
					{
						Path: "google/example/v1",
					},
				},
			},
			outputFiles:  []string{"testlib/apiv1/file2.go"},
			snippetFiles: []string{"internal/generated/snippets/testlib/apiv1/snippet1.go"},
			wantErr:      errGoAPINotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			outputPath := filepath.Join(root, libraryName)
			snippetPath := filepath.Join(root, "internal", "generated", "snippets", libraryName)
			test.library.Output = root
			if test.outputFiles != nil {
				createFiles(t, outputPath, test.outputFiles)
			}
			if test.snippetFiles != nil {
				createFiles(t, snippetPath, test.snippetFiles)
			}

			err := Clean(test.library)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, test.wantErr) {
				t.Errorf("CleanLibrary error = %v, wantErr %v", err, test.wantErr)
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
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil
	}
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
