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

package snippetmetadata

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestUpdateLibraryVersion(t *testing.T) {
	// Copy the input file, as it's modified in place.
	path := copyInputFileToTemp(t, "testdata/version-before.json")
	if err := updateLibraryVersion(path, "1.20.0"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/version-after.json")
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUpdateLibraryVersion_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		setup   func(*testing.T, string)
		wantErr error
	}{
		{
			name:    "no such file",
			setup:   func(t *testing.T, path string) {},
			wantErr: os.ErrNotExist,
		},
		{
			name: "invalid json",
			setup: func(t *testing.T, path string) {
				if err := os.WriteFile(path, []byte("this isn't valid JSON"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			// The json module makes this hard to test more precisely.
			wantErr: nil,
		},
		{
			name: "no clientLibrary field",
			setup: func(t *testing.T, path string) {
				if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: errNoClientLibraryField,
		},
		{
			name: "readonly file",
			setup: func(t *testing.T, path string) {
				if err := os.WriteFile(path, []byte("{\"clientLibrary\":{}}"), 0444); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: os.ErrPermission,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "snippet_metadata.json")
			test.setup(t, path)
			gotErr := updateLibraryVersion(path, "1.2.3")
			if gotErr == nil {
				t.Fatal("expected error, got nil")
			}
			if test.wantErr != nil && !errors.Is(gotErr, test.wantErr) {
				t.Errorf("updateLibraryVersion error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestUpdateAllLibraryVersions(t *testing.T) {
	// Copy the input file, as it's modified in place.
	metadataPath := copyInputFileToTemp(t, "testdata/version-before.json")
	dir := filepath.Dir(metadataPath)
	readmePath := filepath.Join(dir, "README.txt")
	readmeContent := "This is a README file"
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := UpdateAllLibraryVersions(dir, "1.20.0"); err != nil {
		t.Fatal(err)
	}

	// The metadata file should have been updated
	got, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/version-after.json")
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// The readme file shouldn't have changed
	got, err = os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(readmeContent), string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUpdateAllLibraryVersions_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		setup   func(*testing.T, string)
		wantErr error
	}{
		{
			name: "unexpected directory",
			setup: func(t *testing.T, dir string) {
				if err := os.Mkdir(filepath.Join(dir, "snippet_metadata.json"), 0755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: errSnippetMetadataDirectory,
		},
		{
			name: "readonly file",
			setup: func(t *testing.T, dir string) {
				content := "{\"clientLibrary\":{}}"
				if err := os.WriteFile(filepath.Join(dir, "snippet_metadata.json"), []byte(content), 0444); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: os.ErrPermission,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			test.setup(t, dir)
			gotErr := UpdateAllLibraryVersions(dir, "1.20.0")
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("UpdateAllLibraryVersions error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestReformat(t *testing.T) {
	path := copyInputFileToTemp(t, "testdata/unformatted.json")
	if err := reformat(path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/formatted.json")
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestReformat_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		setup   func(*testing.T, string)
		wantErr error
	}{
		{
			name:    "no such file",
			setup:   func(t *testing.T, path string) {},
			wantErr: os.ErrNotExist,
		},
		{
			name: "invalid json",
			setup: func(t *testing.T, path string) {
				if err := os.WriteFile(path, []byte("This isn't valid JSON"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			// The json module makes this hard to test more precisely.
			wantErr: nil,
		},
		{
			name: "readonly file",
			setup: func(t *testing.T, path string) {
				if err := os.WriteFile(path, []byte("{}"), 0444); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: os.ErrPermission,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "snippet_metadata.json")
			test.setup(t, path)
			gotErr := reformat(path)
			if gotErr == nil {
				t.Fatal("expected error, got nil")
			}
			if test.wantErr != nil && !errors.Is(gotErr, test.wantErr) {
				t.Errorf("reformat error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestReformatAll(t *testing.T) {
	// We'll have two copies of unformatted.json - one with a filename
	// indicating it's snippet metadata, and the other just "unformatted.json".
	// The unformatted.json file shouldn't be touched.
	metadataPath := copyInputFileToTemp(t, "testdata/unformatted.json")
	dir := filepath.Dir(metadataPath)
	unformattedPath := filepath.Join(dir, "unformatted.json")
	unformattedContent, err := os.ReadFile("testdata/unformatted.json")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(unformattedPath, []byte(unformattedContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ReformatAll(dir); err != nil {
		t.Fatal(err)
	}

	// The metadata file should have been formatted
	got, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/formatted.json")
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// The unformatted.json file shouldn't have changed
	got, err = os.ReadFile(unformattedPath)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(unformattedContent), string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestReformatAll_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		setup   func(*testing.T, string)
		wantErr error
	}{
		{
			name: "unexpected directory",
			setup: func(t *testing.T, dir string) {
				if err := os.Mkdir(filepath.Join(dir, "snippet_metadata.json"), 0755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: errSnippetMetadataDirectory,
		},
		{
			name: "readonly file",
			setup: func(t *testing.T, dir string) {
				if err := os.WriteFile(filepath.Join(dir, "snippet_metadata.json"), []byte("{}"), 0444); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: os.ErrPermission,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			test.setup(t, dir)
			gotErr := ReformatAll(dir)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("ReformatAll error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestFindAll(t *testing.T) {
	dir := t.TempDir()
	snippetMetadataFiles := []string{
		"snippet_metadata-x.json",
		"a/snippet_metadata-x.json",
		"a/b/snippet_metadata-y.json",
	}
	nonSnippetMetadataFiles := []string{
		"snippets-x.json",
		"snippet_metadata-x.txt",
		"README.md",
		"a/README.md",
	}
	allFiles := append(snippetMetadataFiles, nonSnippetMetadataFiles...)
	for _, file := range allFiles {
		path := filepath.Join(dir, file)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		// The content doesn't matter for this test; empty files are fine.
		if err := os.WriteFile(path, nil, 0644); err != nil {
			t.Fatal(err)
		}
	}
	wantFiles := []string{}
	for _, file := range snippetMetadataFiles {
		wantFiles = append(wantFiles, filepath.Join(dir, file))
	}
	gotFiles, err := findAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(wantFiles)
	slices.Sort(gotFiles)
	if diff := cmp.Diff(wantFiles, gotFiles); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFindAll_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		setup   func(*testing.T, string)
		wantErr error
	}{
		{
			name: "match is a directory",
			setup: func(t *testing.T, dir string) {
				if err := os.Mkdir(filepath.Join(dir, "snippet_metadata.json"), 0755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: errSnippetMetadataDirectory,
		},
		{
			name: "match is a symlink",
			setup: func(t *testing.T, dir string) {
				normalPath := filepath.Join(dir, "test.txt")
				matchPath := filepath.Join(dir, "snippet_metadata.json")
				if err := os.WriteFile(normalPath, nil, 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(normalPath, matchPath); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: errSnippetMetadataLink,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "snippet_metadata.json")
			test.setup(t, dir)
			_, gotErr := findAll(path)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("findAll() error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

// copyInputFileToTemp creates a temporary directory, copies the given file
// into it (as snippet_metadata.json), and returns the path to the new file.
func copyInputFileToTemp(t *testing.T, inputFile string) string {
	dir := t.TempDir()
	content, err := os.ReadFile(inputFile)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "snippet_metadata.json")
	err = os.WriteFile(path, content, 0644)
	if err != nil {
		t.Fatal(err)
	}
	return path
}
