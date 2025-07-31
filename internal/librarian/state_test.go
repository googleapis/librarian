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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
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
    source_paths:
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
						SourcePaths: []string{"src/a", "src/b"},
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
    source_paths:
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
			contentLoader := func(path string) ([]byte, error) {
				return []byte(path), nil
			}
			got, err := parseLibrarianState(contentLoader, test.content, test.source)
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
		name          string
		contentLoader func(file string) ([]byte, error)
		path          string
		want          string
		wantErr       bool
	}{
		{
			name: "find a service config",
			contentLoader: func(file string) ([]byte, error) {
				return os.ReadFile(file)
			},
			path: filepath.Join("..", "..", "testdata", "find_a_service_config"),
			want: "service_config.yaml",
		},
		{
			name: "non existed source path",
			contentLoader: func(file string) ([]byte, error) {
				return os.ReadFile(file)
			},
			path:    filepath.Join("..", "..", "testdata", "non-existed-path"),
			want:    "",
			wantErr: true,
		},
		{
			name: "non service config in a source path",
			contentLoader: func(file string) ([]byte, error) {
				return os.ReadFile(file)
			},
			path:    filepath.Join("..", "..", "testdata", "no_service_config"),
			want:    "",
			wantErr: true,
		},
		{
			name: "simulated load error",
			contentLoader: func(file string) ([]byte, error) {
				return nil, errors.New("simulate loading error for testing")
			},
			path:    filepath.Join("..", "..", "testdata", "no_service_config"),
			want:    "",
			wantErr: true,
		},
		{
			name: "invalid yaml",
			contentLoader: func(file string) ([]byte, error) {
				return os.ReadFile(file)
			},
			path:    filepath.Join("..", "..", "testdata", "invalid_yaml"),
			want:    "",
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := findServiceConfigIn(test.contentLoader, test.path)
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
			contentLoader := func(file string) ([]byte, error) {
				return os.ReadFile(file)
			}
			err := populateServiceConfigIfEmpty(test.state, contentLoader, test.path)
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

func TestReadConfigureResponseJSON(t *testing.T) {
	t.Parallel()
	contentLoader := func(data []byte, state *config.LibraryState) error {
		return json.Unmarshal(data, state)
	}
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
				SourcePaths:   []string{"src/example/path"},
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
			name:      "invalid file name",
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
			if test.name == "invalid file name" {
				filePath := filepath.Join(tempDir, "my\x00file.json")
				_, err := readConfigureResponse(contentLoader, filePath)
				if err == nil {
					t.Error("readConfigureResponse() expected an error but got nil")
				}

				if g, w := err.Error(), "failed to read response file"; !strings.Contains(g, w) {
					t.Errorf("got %q, wanted it to contain %q", g, w)
				}

				return
			}

			if test.name == "invalid content loader" {
				invalidContentLoader := func(data []byte, state *config.LibraryState) error {
					return errors.New("simulated Unmarshal error")
				}
				dst := fmt.Sprintf("%s/copy.json", os.TempDir())
				if err := copyFile(dst, test.jsonFilePath); err != nil {
					t.Error(err)
				}
				_, err := readConfigureResponse(invalidContentLoader, dst)
				if err == nil {
					t.Errorf("readConfigureResponse() expected an error but got nil")
				}

				if g, w := err.Error(), "failed to load file"; !strings.Contains(g, w) {
					t.Errorf("got %q, wanted it to contain %q", g, w)
				}
				return
			}

			// The response file is removed by the readConfigureResponse() function,
			// so we create a copy and read from it.
			dstFilePath := fmt.Sprintf("%s/copy.json", os.TempDir())
			if err := copyFile(dstFilePath, test.jsonFilePath); err != nil {
				t.Error(err)
			}

			gotState, err := readConfigureResponse(contentLoader, dstFilePath)

			if err != nil {
				t.Fatalf("readConfigureResponse() unexpected error: %v", err)
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
		name     string
		filename string
		state    *config.LibrarianState
	}{
		{
			name:     "successful parsing librarianState to yaml",
			filename: "successful-parsing-librarianState-yaml",
			state: &config.LibrarianState{
				Image: "v1.0.0",
				Libraries: []*config.LibraryState{
					{
						ID:                  "google-cloud-go",
						Version:             "1.0.0",
						LastGeneratedCommit: "abcd123",
						APIs: []*config.API{
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
						APIs: []*config.API{
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
			name:     "empty librarianState to yaml",
			filename: "empty-librarianState-yaml",
			state:    &config.LibrarianState{},
		},
		{
			name:  "invalid file name",
			state: &config.LibrarianState{},
		},
		{
			name:  "invalid content parser",
			state: &config.LibrarianState{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			contentParser := func(state *config.LibrarianState) ([]byte, error) {
				data := &bytes.Buffer{}
				encoder := yaml.NewEncoder(data)
				encoder.SetIndent(2)
				if err := encoder.Encode(state); err != nil {
					return nil, err
				}

				if err := encoder.Close(); err != nil {
					return nil, err
				}
				return data.Bytes(), nil
			}
			if test.name == "invalid file name" {
				filePath := filepath.Join(tempDir, "my\x00file.yaml")
				err := writeLibrarianState(contentParser, test.state, filePath)
				if err == nil {
					t.Errorf("writeLibrarianState() expected an error but got nil")
				}

				if g, w := err.Error(), "failed to create librarian state file"; !strings.Contains(g, w) {
					t.Errorf("got %q, wanted it to contain %q", g, w)
				}
				return
			}

			if test.name == "invalid content parser" {
				invalidContentParser := func(state *config.LibrarianState) ([]byte, error) {
					return nil, errors.New("simulated parsing error")
				}
				if err := os.MkdirAll(filepath.Join(tempDir, ".librarian"), 0755); err != nil {
					t.Errorf("MkdirAll() failed to make directory")
				}
				err := writeLibrarianState(invalidContentParser, test.state, tempDir)
				if err == nil {
					t.Errorf("writeLibrarianState() expected an error but got nil")
				}

				if g, w := err.Error(), "failed to convert state to bytes"; !strings.Contains(g, w) {
					t.Errorf("got %q, wanted it to contain %q", g, w)
				}
				return
			}

			if err := os.MkdirAll(filepath.Join(tempDir, ".librarian"), 0755); err != nil {
				t.Errorf("MkdirAll() failed to make directory")
			}

			err := writeLibrarianState(contentParser, test.state, tempDir)

			if err != nil {
				t.Fatalf("writeLibrarianState() unexpected error: %v", err)
			}

			// Verify the file content
			gotBytes, err := os.ReadFile(filepath.Join(tempDir, ".librarian", pipelineStateFile))
			if err != nil {
				t.Fatalf("Failed to read generated file: %v", err)
			}

			fileName := fmt.Sprintf("%s.yaml", test.filename)
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
