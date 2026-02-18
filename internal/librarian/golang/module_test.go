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

package golang

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestFill(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		want    *config.Library
	}{
		{
			name: "fill default import path",
			library: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{{Path: "google/cloud/secretmanager"}},
			},
			want: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{{Path: "google/cloud/secretmanager"}},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:       "google/cloud/secretmanager",
							ImportPath: "secretmanager",
						},
					},
				},
			},
		},
		{
			name: "fill default import client directory",
			library: &config.Library{
				Name: "ai",
				APIs: []*config.API{{Path: "google/cloud/ai/generativelanguage/v1"}},
			},
			want: &config.Library{
				Name: "ai",
				APIs: []*config.API{{Path: "google/cloud/ai/generativelanguage/v1"}},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:            "google/cloud/ai/generativelanguage/v1",
							ImportPath:      "ai/generativelanguage",
							ClientDirectory: "generativelanguage",
						},
					},
				},
			},
		},
		{
			name: "defaults do not override library config",
			library: &config.Library{
				Name: "example",
				APIs: []*config.API{{Path: "google/cloud/example/v1"}},
				Go: &config.GoModule{
					DeleteGenerationOutputPaths: []string{"example"},
					GoAPIs: []*config.GoAPI{
						{
							Path:               "google/cloud/example/v1",
							ImportPath:         "example",
							ClientDirectory:    "example",
							NoRESTNumericEnums: true,
						},
					},
				},
			},
			want: &config.Library{
				Name: "example",
				APIs: []*config.API{{Path: "google/cloud/example/v1"}},
				Go: &config.GoModule{
					DeleteGenerationOutputPaths: []string{"example"},
					GoAPIs: []*config.GoAPI{
						{
							Path:               "google/cloud/example/v1",
							ImportPath:         "example",
							ClientDirectory:    "example",
							NoRESTNumericEnums: true,
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Fill(test.library)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindGoAPI(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		apiPath string
		want    *config.GoAPI
	}{
		{
			name: "find an api",
			library: &config.Library{
				Name: "secretmanager",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:            "google/cloud/secretmanager/v1",
							ClientDirectory: "customDir",
						},
					},
				},
			},
			apiPath: "google/cloud/secretmanager/v1",
			want: &config.GoAPI{
				Path:            "google/cloud/secretmanager/v1",
				ClientDirectory: "customDir",
			},
		},
		{
			name: "do not have a go module",
			library: &config.Library{
				Name: "secretmanager",
			},
			apiPath: "google/cloud/secretmanager/v1",
		},
		{
			name: "find an api",
			library: &config.Library{
				Name: "secretmanager",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:            "google/cloud/secretmanager/v1",
							ClientDirectory: "customDir",
						},
					},
				},
			},
			apiPath: "google/cloud/secretmanager/v1beta1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := findGoAPI(test.library, test.apiPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
