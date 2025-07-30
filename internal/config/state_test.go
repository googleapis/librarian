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

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLibrarianState_Validate(t *testing.T) {
	for _, test := range []struct {
		name    string
		state   *LibrarianState
		wantErr bool
	}{
		{
			name: "valid state",
			state: &LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*LibraryState{
					{
						ID:          "a/b",
						SourcePaths: []string{"src/a", "src/b"},
						APIs: []*API{
							{
								Path: "a/b/v1",
							},
						},
					},
				},
			},
		},
		{
			name: "missing image",
			state: &LibrarianState{
				Libraries: []*LibraryState{
					{
						ID:          "a/b",
						SourcePaths: []string{"src/a", "src/b"},
						APIs: []*API{
							{
								Path: "a/b/v1",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing libraries",
			state: &LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.state.Validate(); (err != nil) != test.wantErr {
				t.Errorf("LibrarianState.Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestLibrary_Validate(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *LibraryState
		wantErr bool
	}{
		{
			name: "valid library",
			library: &LibraryState{
				ID:          "a/b",
				SourcePaths: []string{"src/a", "src/b"},
				APIs: []*API{
					{
						Path: "a/b/v1",
					},
				},
			},
		},
		{
			name: "missing id",
			library: &LibraryState{
				SourcePaths: []string{"src/a", "src/b"},
				APIs: []*API{
					{
						Path: "a/b/v1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "id is dot",
			library: &LibraryState{
				ID:          ".",
				SourcePaths: []string{"src/a", "src/b"},
				APIs: []*API{
					{
						Path: "a/b/v1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "id is double dot",
			library: &LibraryState{
				ID:          "..",
				SourcePaths: []string{"src/a", "src/b"},
				APIs: []*API{
					{
						Path: "a/b/v1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing source paths",
			library: &LibraryState{
				ID: "a/b",
				APIs: []*API{
					{
						Path: "a/b/v1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing apis",
			library: &LibraryState{
				ID:          "a/b",
				SourcePaths: []string{"src/a", "src/b"},
			},
			wantErr: true,
		},
		{
			name: "valid version without v prefix",
			library: &LibraryState{
				ID:          "a/b",
				Version:     "1.2.3",
				SourcePaths: []string{"src/a", "src/b"},
				APIs: []*API{
					{
						Path: "a/b/v1",
					},
				},
			},
		},
		{
			name: "invalid id characters",
			library: &LibraryState{
				ID:          "a/b!",
				SourcePaths: []string{"src/a", "src/b"},
				APIs: []*API{
					{
						Path: "a/b/v1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid last generated commit non-hex",
			library: &LibraryState{
				ID:                  "a/b",
				LastGeneratedCommit: "not-a-hex-string",
				SourcePaths:         []string{"src/a", "src/b"},
				APIs: []*API{
					{
						Path: "a/b/v1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid last generated commit wrong length",
			library: &LibraryState{
				ID:                  "a/b",
				LastGeneratedCommit: "deadbeef",
				SourcePaths:         []string{"src/a", "src/b"},
				APIs: []*API{
					{
						Path: "a/b/v1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid preserve_regex",
			library: &LibraryState{
				ID:            "a/b",
				SourcePaths:   []string{"src/a"},
				APIs:          []*API{{Path: "a/b/v1"}},
				PreserveRegex: []string{".*\\.txt"},
			},
		},
		{
			name: "invalid preserve_regex",
			library: &LibraryState{
				ID:            "a/b",
				SourcePaths:   []string{"src/a"},
				APIs:          []*API{{Path: "a/b/v1"}},
				PreserveRegex: []string{"["},
			},
			wantErr: true,
		},
		{
			name: "valid remove_regex",
			library: &LibraryState{
				ID:          "a/b",
				SourcePaths: []string{"src/a"},
				APIs:        []*API{{Path: "a/b/v1"}},
				RemoveRegex: []string{".*\\.log"},
			},
		},
		{
			name: "invalid remove_regex",
			library: &LibraryState{
				ID:          "a/b",
				SourcePaths: []string{"src/a"},
				APIs:        []*API{{Path: "a/b/v1"}},
				RemoveRegex: []string{"("},
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.library.Validate(); (err != nil) != test.wantErr {
				t.Errorf("Library.Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestAPI_Validate(t *testing.T) {
	for _, test := range []struct {
		name    string
		api     *API
		wantErr bool
	}{
		{
			name: "valid api",
			api: &API{
				Path: "a/b/v1",
			},
		},
		{
			name:    "missing path",
			api:     &API{},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.api.Validate(); (err != nil) != test.wantErr {
				t.Errorf("API.Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestIsValidDirPath(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		want bool
	}{
		{"valid", "a/b/c", true},
		{"valid with dots", "a/./b/../c", true},
		{"empty", "", false},
		{"absolute", "/a/b", false},
		{"up traversal", "../a", false},
		{"double dot", "..", false},
		{"single dot", ".", false},
		{"invalid chars", "a/b<c", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := isValidDirPath(test.path); got != test.want {
				t.Errorf("isValidDirPath(%q) = %v, want %v", test.path, got, test.want)
			}
		})
	}
}

func TestIsValidImage(t *testing.T) {
	for _, test := range []struct {
		name  string
		image string
		want  bool
	}{
		{"valid with tag", "gcr.io/google/go-container:v1", true},
		{"valid with latest tag", "ubuntu:latest", true},
		{"valid with port and tag", "my-registry:5000/my/image:v1", true},
		{"invalid no tag", "gcr.io/google/go-container", false},
		{"invalid with port no tag", "my-registry:5000/my/image", false},
		{"invalid with spaces", "gcr.io/google/go-container with spaces", false},
		{"invalid no repo", ":v1", false},
		{"invalid empty tag", "my-image:", false},
		{"invalid empty", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := isValidImage(test.image); got != test.want {
				t.Errorf("isValidImage(%q) = %v, want %v", test.image, got, test.want)
			}
		})
	}
}

func TestReadResponseJSON(t *testing.T) {
	t.Parallel()
	contentLoader := func(data []byte, state *LibraryState) error {
		return json.Unmarshal(data, state)
	}
	for _, test := range []struct {
		name         string
		jsonFilePath string
		wantState    *LibraryState
	}{
		{
			name:         "successful-unmarshal",
			jsonFilePath: "../../testdata/successful-unmarshal-libraryState.json",
			wantState: &LibraryState{
				ID:                  "google-cloud-go",
				Version:             "1.0.0",
				LastGeneratedCommit: "abcd123",
				APIs: []*API{
					{
						Path:          "google/cloud/compute/v1",
						ServiceConfig: "example_service_config.yaml",
					},
				},
				SourcePaths:   []string{"src/example/path"},
				PreserveRegex: []string{"example-preserve-regex"},
				RemoveRegex:   []string{"example-remove-regex"},
			},
		},
		{
			name:         "empty libraryState",
			jsonFilePath: "../../testdata/empty-libraryState.json",
			wantState:    &LibraryState{},
		},
		{
			name:      "invalid_file_name",
			wantState: nil,
		},
		{
			name:         "invalid content loader",
			jsonFilePath: "../../testdata/invalid-contentLoader.json",
			wantState:    nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			if test.name == "invalid_file_name" {
				filePath := filepath.Join(tempDir, "my\x00file.json")
				_, err := ReadResponse(contentLoader, filePath)
				if err == nil {
					t.Error("readResponse() expected an error but got nil")
				}

				if g, w := err.Error(), "failed to read response file"; !strings.Contains(g, w) {
					t.Errorf("got %q, wanted it to contain %q", g, w)
				}

				return
			}

			if test.name == "invalid content loader" {
				invalidContentLoader := func(data []byte, state *LibraryState) error {
					return errors.New("simulated Unmarshal error")
				}
				dst := fmt.Sprintf("%s/copy.json", os.TempDir())
				if err := copyFile(dst, test.jsonFilePath); err != nil {
					t.Error(err)
				}
				_, err := ReadResponse(invalidContentLoader, dst)
				if err == nil {
					t.Errorf("readResponse() expected an error but got nil")
				}

				if g, w := err.Error(), "failed to load file"; !strings.Contains(g, w) {
					t.Errorf("got %q, wanted it to contain %q", g, w)
				}
				return
			}

			// The response file is removed by the readResponse() function,
			// so we create a copy and read from it.
			dstFilePath := fmt.Sprintf("%s/copy.json", os.TempDir())
			if err := copyFile(dstFilePath, test.jsonFilePath); err != nil {
				t.Error(err)
			}

			gotState, err := ReadResponse(contentLoader, dstFilePath)

			if err != nil {
				t.Fatalf("readResponse() unexpected error: %v", err)
			}

			if diff := cmp.Diff(test.wantState, gotState); diff != "" {
				t.Errorf("Response library state mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWriteLibrarianState(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		state *LibrarianState
	}{
		{
			name: "successful-marshaling-librarianState-yaml",
			state: &LibrarianState{
				Image: "v1.0.0",
				Libraries: []*LibraryState{
					{
						ID:                  "google-cloud-go",
						Version:             "1.0.0",
						LastGeneratedCommit: "abcd123",
						APIs: []*API{
							{
								Path:          "google/cloud/compute/v1",
								ServiceConfig: "example_service_config.yaml",
							},
						},
						SourcePaths: []string{
							"src/example/path",
						},
						PreserveRegex: []string{
							"example-preserve-regex",
						},
						RemoveRegex: []string{
							"example-remove-regex",
						},
					},
					{
						ID:      "google-cloud-storage",
						Version: "1.2.3",
						APIs: []*API{
							{
								Path:          "google/storage/v1",
								ServiceConfig: "storage_service_config.yaml",
							},
						},
					},
				},
			},
		},
		{
			name:  "empty-librarianState-yaml",
			state: &LibrarianState{},
		},
		{
			name:  "invalid_file_name",
			state: &LibrarianState{},
		},
		{
			name:  "invalid content parser",
			state: &LibrarianState{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			contentParser := func(state *LibrarianState) ([]byte, error) {
				return yaml.Marshal(state)
			}
			if test.name == "invalid_file_name" {
				filePath := filepath.Join(tempDir, "my\x00file.yaml")
				err := WriteLibrarianState(contentParser, test.state, filePath)
				if err == nil {
					t.Errorf("writeLibrarianState() expected an error but got nil")
				}

				if g, w := err.Error(), "failed to create librarian state file"; !strings.Contains(g, w) {
					t.Errorf("got %q, wanted it to contain %q", g, w)
				}
				return
			}

			if test.name == "invalid content parser" {
				filePath := filepath.Join(tempDir, "state.yaml")
				invalidContentParser := func(state *LibrarianState) ([]byte, error) {
					return nil, errors.New("simulated parsing error")
				}
				err := WriteLibrarianState(invalidContentParser, test.state, filePath)
				if err == nil {
					t.Errorf("writeLibrarianState() expected an error but got nil")
				}

				if g, w := err.Error(), "failed to convert state to bytes"; !strings.Contains(g, w) {
					t.Errorf("got %q, wanted it to contain %q", g, w)
				}
				return
			}

			filePath := filepath.Join(tempDir, "state.yaml")
			err := WriteLibrarianState(contentParser, test.state, filePath)

			if err != nil {
				t.Fatalf("writeLibrarianState() unexpected error: %v", err)
			}

			// Verify the file content
			gotBytes, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read generated file: %v", err)
			}

			fileName := fmt.Sprintf("%s.yaml", test.name)
			wantBytes, readErr := os.ReadFile(filepath.Join("..", "..", "testdata", fileName))
			if readErr != nil {
				t.Fatalf("Failed to read expected state for comparison: %v", readErr)
			}

			if diff := cmp.Diff(string(wantBytes), string(gotBytes)); diff != "" {
				t.Errorf("Generated YAML mismatch (-want +got):\n%s", diff)
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
