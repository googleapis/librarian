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
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

// mockGitHubClient is a mock implementation of the GitHubClient interface for testing.
type mockGitHubClient struct {
	GitHubClient
	rawContent []byte
	rawErr     error
}

func (m *mockGitHubClient) GetRawContent(ctx context.Context, path, ref string) ([]byte, error) {
	return m.rawContent, m.rawErr
}

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

func TestFetchRemoteLibrarianState(t *testing.T) {
	for _, test := range []struct {
		name    string
		client  GitHubClient
		want    *config.LibrarianState
		wantErr bool
	}{
		{
			name: "success",
			client: &mockGitHubClient{
				rawContent: []byte(`image: gcr.io/test/image:v1.2.3
libraries:
  - id: a/b
    source_paths:
      - src/a
      - src/b
    apis:
      - path: a/b/v1
        service_config: a/b/v1/service.yaml
`),
			},
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
			name: "GetRawContent error",
			client: &mockGitHubClient{
				rawErr: errors.New("GetRawContent error"),
			},
			wantErr: true,
		},
		{
			name: "invalid yaml",
			client: &mockGitHubClient{
				rawContent: []byte("invalid yaml"),
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := fetchRemoteLibrarianState(context.Background(), test.client, "main", "")
			if (err != nil) != test.wantErr {
				t.Errorf("fetchRemoteLibrarianState() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("fetchRemoteLibrarianState() mismatch (-want +got): %s", diff)
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
