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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestParseLibrarianState(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		want    *config.LibrarianState
		wantErr bool
	}{{
		name:    "valid state",
		content: "image: gcr.io/test/image:v1.2.3\nlibraries:\n  - id: a/b\n    source_paths:\n      - src/a\n      - src/b\n    apis:\n      - path: a/b/v1\n        service_config: a/b/v1/service.yaml\n",
		want: &config.LibrarianState{
			Image: "gcr.io/test/image:v1.2.3",
			Libraries: []config.Library{
				{
					Id:          "a/b",
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
