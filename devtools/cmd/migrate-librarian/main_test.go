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

func TestBuildConfig(t *testing.T) {
	for _, test := range []struct {
		name  string
		lang  string
		state *legacyconfig.LibrarianState
		cfg   *legacyconfig.LibrarianConfig
		want  *config.Config
	}{
		{
			name:  "go_monorepo_defaults",
			lang:  "go",
			state: &legacyconfig.LibrarianState{},
			cfg:   &legacyconfig.LibrarianConfig{},
			want: &config.Config{
				Language: "go",
				Repo:     "googleapis/google-cloud-go",
				Default: &config.Default{
					TagFormat: defaultTagFormat,
				},
			},
		},
		{
			name:  "python_monorepo_defaults",
			lang:  "python",
			state: &legacyconfig.LibrarianState{},
			cfg:   &legacyconfig.LibrarianConfig{},
			want: &config.Config{
				Language: "python",
				Repo:     "googleapis/google-cloud-python",
				Default: &config.Default{
					TagFormat: defaultTagFormat,
				},
			},
		},
		{
			name: "no_librarian_config",
			lang: "python",
			state: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:      "example-library",
						Version: "1.0.0",
						APIs: []*legacyconfig.API{
							{
								Path:          "google/example/api/v1",
								ServiceConfig: "google/example/api/v1/service.yaml",
							},
						},
						PreserveRegex: []string{
							"example-preserve-1",
							"example-preserve-2",
						},
					},
				},
			},
			cfg: &legacyconfig.LibrarianConfig{},
			want: &config.Config{
				Language: "python",
				Repo:     "googleapis/google-cloud-python",
				Default: &config.Default{
					TagFormat: defaultTagFormat,
				},
				Libraries: []*config.Library{
					{
						Name:    "example-library",
						Version: "1.0.0",
						Channels: []*config.Channel{
							{
								Path:          "google/example/api/v1",
								ServiceConfig: "google/example/api/v1/service.yaml",
							},
						},
						Keep: []string{
							"example-preserve-1",
							"example-preserve-2",
						},
						SpecificationFormat: "protobuf",
					},
				},
			},
		},
		{
			name: "has_a_librarian_config",
			lang: "python",
			state: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:      "example-library",
						Version: "1.0.0",
					},
					{
						ID:      "another-library",
						Version: "2.0.0",
					},
				},
			},
			cfg: &legacyconfig.LibrarianConfig{
				Libraries: []*legacyconfig.LibraryConfig{
					{
						LibraryID:       "example-library",
						GenerateBlocked: true,
						ReleaseBlocked:  true,
					},
				},
			},
			want: &config.Config{
				Language: "python",
				Repo:     "googleapis/google-cloud-python",
				Default: &config.Default{
					TagFormat: defaultTagFormat,
				},
				Libraries: []*config.Library{
					{
						Name:                "another-library",
						Version:             "2.0.0",
						SpecificationFormat: "protobuf",
					},
					{
						Name:                "example-library",
						Version:             "1.0.0",
						SpecificationFormat: "protobuf",
						SkipGenerate:        true,
						SkipRelease:         true,
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildConfig(test.state, test.cfg, test.lang)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
