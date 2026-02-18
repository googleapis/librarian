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
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			},
			want: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:       "google/cloud/secretmanager/v1",
							ImportPath: "secretmanager",
						},
					},
				},
			},
		},
		{
			name: "fill default import path and client directory",
			library: &config.Library{
				Name: "bigquery",
				APIs: []*config.API{
					{
						Path: "google/cloud/bigquery/analyticshub/v1",
					},
					{
						Path: "google/cloud/bigquery/biglake/v1",
					},
				},
			},
			want: &config.Library{
				Name: "bigquery",
				APIs: []*config.API{
					{
						Path: "google/cloud/bigquery/analyticshub/v1",
					},
					{
						Path: "google/cloud/bigquery/biglake/v1",
					},
				},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:            "google/cloud/bigquery/analyticshub/v1",
							ClientDirectory: "analyticshub",
							ImportPath:      "bigquery/analyticshub",
						},
						{
							Path:            "google/cloud/bigquery/biglake/v1",
							ClientDirectory: "biglake",
							ImportPath:      "bigquery/biglake",
						},
					},
				},
			},
		},
		{
			name: "skip non cloud api with nil Go module",
			library: &config.Library{
				Name: "ai",
				APIs: []*config.API{{Path: "google/ai/generativelanguage/v1"}},
			},
			want: &config.Library{
				Name: "ai",
				APIs: []*config.API{{Path: "google/ai/generativelanguage/v1"}},
				Go:   &config.GoModule{},
			},
		},
		{
			name: "skip non cloud api with Go module",
			library: &config.Library{
				Name: "ai",
				APIs: []*config.API{{Path: "google/ai/generativelanguage/v1"}},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:            "google/ai/generativelanguage/v1",
							ClientDirectory: "generativelanguage",
						},
					},
				},
			},
			want: &config.Library{
				Name: "ai",
				APIs: []*config.API{{Path: "google/ai/generativelanguage/v1"}},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:            "google/ai/generativelanguage/v1",
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
							NoRESTNumericEnums: true, // this value will be kept.
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
