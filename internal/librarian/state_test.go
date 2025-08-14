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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"gopkg.in/yaml.v3"
)

func TestParseLibrarianState(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		source  string
		want    *config.LibrarianState
		wantErr bool
	}{
		{
			name: "valid state",
			content: `image: gcr.io/test/image:v1.2.3
libraries:
  - id: a/b
    source_roots:
      - src/a
      - src/b
    apis:
      - path: a/b/v1
        service_config: a/b/v1/service.yaml
`,
			source: "",
			want: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:          "a/b",
						SourceRoots: []string{"src/a", "src/b"},
						APIs: []*config.API{
							{
								Path:          "a/b/v1",
								ServiceConfig: "a/b/v1/service.yaml",
							},
						},
					},
				},
			},
		},
		{
			name: "invalid source",
			content: `image: gcr.io/test/image:v1.2.3
libraries:
  - id: a/b
    source_roots:
      - src/a
      - src/b
    apis:
      - path: a/b/v1
        service_config: 
`,
			source:  "/non-existed-path",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid yaml",
			content: "image: gcr.io/test/image:v1.2.3\n  libraries: []\n",
			source:  "",
			wantErr: true,
		},
		{
			name:    "validation error",
			content: "image: gcr.io/test/image:v1.2.3\nlibraries: []\n",
			source:  "",
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "state.yaml")
			if err := os.WriteFile(path, []byte(test.content), 0644); err != nil {
				t.Fatalf("os.WriteFile() failed: %v", err)
			}
			got, err := parseLibrarianState(path, test.source)
			if (err != nil) != test.wantErr {
				t.Errorf("parseLibrarianState() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("parseLibrarianState() mismatch (-want +got): %s", diff)
			}
		})
	}
}

func TestFindServiceConfigIn(t *testing.T) {
	for _, test := range []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "find a service config",
			path: filepath.Join("..", "..", "testdata", "find_a_service_config"),
			want: "service_config.yaml",
		},
		{
			name:    "non existed source path",
			path:    filepath.Join("..", "..", "testdata", "non-existed-path"),
			want:    "",
			wantErr: true,
		},
		{
			name:    "non service config in a source path",
			path:    filepath.Join("..", "..", "testdata", "no_service_config"),
			want:    "",
			wantErr: true,
		},
		{
			name:    "simulated load error",
			path:    filepath.Join("..", "..", "testdata", "no_service_config"),
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid yaml",
			path:    filepath.Join("..", "..", "testdata", "invalid_yaml"),
			want:    "",
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := findServiceConfigIn(test.path)
			if test.wantErr {
				if err == nil {
					t.Errorf("findServiceConfigIn() should return error")
				}

				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("fetchRemoteLibrarianState() mismatch (-want +got): %s", diff)
			}
		})
	}
}

func TestPopulateServiceConfig(t *testing.T) {
	for _, test := range []struct {
		name    string
		state   *config.LibrarianState
		path    string
		want    *config.LibrarianState
		wantErr bool
	}{
		{
			name: "populate service config",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "example-id",
						APIs: []*config.API{
							{
								Path: "example/api",
							},
							{
								Path:          "another/example/api",
								ServiceConfig: "another_config.yaml",
							},
						},
					},
				},
			},
			path: filepath.Join("..", "..", "testdata", "populate_service_config"),
			want: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "example-id",
						APIs: []*config.API{
							{
								Path:          "example/api",
								ServiceConfig: "example_api_config.yaml",
							},
							{
								Path:          "another/example/api",
								ServiceConfig: "another_config.yaml",
							},
						},
					},
				},
			},
		},
		{
			name: "non valid api path",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "example-id",
						APIs: []*config.API{
							{
								Path: "non-existed/example/api",
							},
						},
					},
				},
			},
			path:    "/non-existed-source-path",
			want:    nil,
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := populateServiceConfigIfEmpty(test.state, test.path)
			if test.wantErr {
				if err == nil {
					t.Errorf("findServiceConfigIn() should return error")
				}

				return
			}
			if diff := cmp.Diff(test.want, test.state); diff != "" {
				t.Errorf("fetchRemoteLibrarianState() mismatch (-want +got): %s", diff)
			}
		})
	}
}

func TestSaveLibrarianState(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, config.LibrarianDir), 0755); err != nil {
		t.Fatal(err)
	}
	state := &config.LibrarianState{
		Image: "gcr.io/test/image:v1.2.3",
		Libraries: []*config.LibraryState{
			{
				ID: "a/b",
				SourceRoots: []string{
					"src/a",
					"src/b",
				},
				APIs: []*config.API{
					{
						Path:          "a/b/v1",
						ServiceConfig: "a/b/v1/service.yaml",
					},
				},
				PreserveRegex: []string{},
				RemoveRegex:   []string{},
			},
		},
	}
	if err := saveLibrarianState(tmpDir, state); err != nil {
		t.Fatalf("saveLibrarianState() failed: %v", err)
	}

	path := filepath.Join(tmpDir, config.LibrarianDir, "state.yaml")
	gotBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() failed: %v", err)
	}
	gotState := &config.LibrarianState{}
	if err := yaml.Unmarshal(gotBytes, gotState); err != nil {
		t.Fatalf("yaml.Unmarshal() failed: %v", err)
	}

	if diff := cmp.Diff(state, gotState); diff != "" {
		t.Errorf("saveLibrarianState() mismatch (-want +got): %s", diff)
	}
}

func TestReadConfigureResponseJSON(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name         string
		jsonFilePath string
		wantState    *config.LibraryState
	}{
		{
			name:         "successful load content",
			jsonFilePath: "../../testdata/successful-unmarshal-libraryState.json",
			wantState: &config.LibraryState{
				ID:                  "google-cloud-go",
				Version:             "1.0.0",
				LastGeneratedCommit: "abcd123",
				APIs: []*config.API{
					{
						Path:          "google/cloud/compute/v1",
						ServiceConfig: "example_service_config.yaml",
					},
				},
				SourceRoots:   []string{"src/example/path"},
				PreserveRegex: []string{"example-preserve-regex"},
				RemoveRegex:   []string{"example-remove-regex"},
			},
		},
		{
			name:         "empty libraryState",
			jsonFilePath: "../../testdata/empty-libraryState.json",
			wantState:    &config.LibraryState{},
		},
		{
			name:         "load content with an error message",
			jsonFilePath: "../../testdata/unmarshal-libraryState-with-error-msg.json",
			wantState:    nil,
		},
		{
			name:      "invalid file name",
			wantState: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			if test.name == "invalid file name" {
				filePath := filepath.Join(tempDir, "my\x00file.json")
				_, err := readLibraryState(filePath)
				if err == nil {
					t.Error("readLibraryState() expected an error but got nil")
				}

				if g, w := err.Error(), "failed to read response file"; !strings.Contains(g, w) {
					t.Errorf("got %q, wanted it to contain %q", g, w)
				}

				return
			}

			if test.name == "invalid content loader" {
				dst := fmt.Sprintf("%s/copy.json", os.TempDir())
				if err := copyFile(dst, test.jsonFilePath); err != nil {
					t.Error(err)
				}
				_, err := readLibraryState(dst)
				if err == nil {
					t.Errorf("readLibraryState() expected an error but got nil")
				}

				if g, w := err.Error(), "failed to load file"; !strings.Contains(g, w) {
					t.Errorf("got %q, wanted it to contain %q", g, w)
				}
				return
			}

			// The response file is removed by the readLibraryState() function,
			// so we create a copy and read from it.
			dstFilePath := fmt.Sprintf("%s/copy.json", os.TempDir())
			if err := copyFile(dstFilePath, test.jsonFilePath); err != nil {
				t.Error(err)
			}

			gotState, err := readLibraryState(dstFilePath)

			if test.name == "load content with an error message" {
				if err == nil {
					t.Errorf("readLibraryState() expected an error but got nil")
				}

				if g, w := err.Error(), "failed with error message"; !strings.Contains(g, w) {
					t.Errorf("got %q, wanted it to contain %q", g, w)
				}

				return
			}

			if err != nil {
				t.Fatalf("readLibraryState() unexpected error: %v", err)
			}

			if diff := cmp.Diff(test.wantState, gotState); diff != "" {
				t.Errorf("Response library state mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func copyFile(dst, src string) (err error) {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}

	defer func() {
		if err = errors.Join(err, destinationFile.Close()); err != nil {
			err = fmt.Errorf("copyFile(%q, %q): %w", dst, src, err)
		}
	}()

	_, err = io.Copy(destinationFile, sourceFile)

	return err
}
