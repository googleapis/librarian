// Copyright 2025 Google LLC
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

package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
)

func TestBuildPythonLibraries(t *testing.T) {
	for _, test := range []struct {
		name  string
		input *MigrationInput
		want  []*config.Library
	}{
		{
			name: "basic",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID:      "example-library",
							Version: "1.2.3",
							APIs: []*legacyconfig.API{
								{
									Path:          "google/example/api/v1",
									ServiceConfig: "path/to/config.yaml",
								},
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
			},
			want: []*config.Library{
				{
					Name:    "example-library",
					Version: "1.2.3",
					APIs: []*config.API{
						{
							Path: "google/example/api/v1",
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildPythonLibraries(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
