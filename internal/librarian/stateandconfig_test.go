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
		want    *config.LibrarianState
		wantErr bool
	}{{
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
		want: &config.LibrarianState{
			Image: "gcr.io/test/image:v1.2.3",
			Libraries: []*config.LibraryState{
				{
					ID:          "a/b",
					SourcePaths: []string{"src/a", "src/b"},
					APIs: []config.API{
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
			name:    "invalid yaml",
			content: "image: gcr.io/test/image:v1.2.3\n  libraries: []\n",
			wantErr: true,
		},
		{
			name:    "validation error",
			content: "image: gcr.io/test/image:v1.2.3\nlibraries: []\n",
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			contentLoader := func() ([]byte, error) {
				return []byte(test.content), nil
			}
			got, err := parseLibrarianState(contentLoader)
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
						APIs: []config.API{
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
			got, err := fetchRemoteLibrarianState(context.Background(), test.client, "main")
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
