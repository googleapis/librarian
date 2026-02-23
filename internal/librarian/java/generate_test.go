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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
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
	t.Parallel()
	testhelper.RequireCommand(t, "google-java-format")

	for _, test := range []struct {
		name    string
		setup   func(t *testing.T, root string)
		wantErr bool
	}{
		{
			name: "successful format",
			setup: func(t *testing.T, root string) {
				if err := os.WriteFile(filepath.Join(root, "SomeClass.java"), []byte("public class SomeClass {}"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: false,
		},
		{
			name:    "no files found",
			setup:   func(t *testing.T, root string) {},
			wantErr: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			test.setup(t, tmpDir)
			err := Format(t.Context(), &config.Library{Output: tmpDir})
			if (err != nil) != test.wantErr {
				t.Errorf("Format() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestCollectJavaFiles(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create a mix of files
	filesToCreate := []string{
		"Root.java",
		"subdir/Nested.java",
		"subdir/NotJava.txt",
		"samples/snippets/generated/Ignored.java",
		"another/dir/More.java",
	}

	for _, f := range filesToCreate {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	want := []string{
		filepath.Join(tmpDir, "Root.java"),
		filepath.Join(tmpDir, "subdir", "Nested.java"),
		filepath.Join(tmpDir, "another", "dir", "More.java"),
	}

	got, err := collectJavaFiles(tmpDir)
	if err != nil {
		t.Fatalf("collectJavaFiles() error = %v", err)
	}

	// Sort both for comparison
	sort := func(s []string) {
		for i := 0; i < len(s); i++ {
			for j := i + 1; j < len(s); j++ {
				if s[i] > s[j] {
					s[i], s[j] = s[j], s[i]
				}
			}
		}
	}
	sort(got)
	sort(want)

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("collectJavaFiles() mismatch (-want +got):\n%s", diff)
	}
}

func TestClean(t *testing.T) {
	library := &config.Library{Name: "test-lib"}

	if err := Clean(library); err != nil {
		t.Errorf("Clean() error = %v, want nil", err)
	}
}
