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

package python

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestFill(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		library *config.Library
		want    *config.Library
	}{
		{
			name:    "empty library",
			library: &config.Library{},
			want:    &config.Library{},
		},
		{
			name: "preview library default output",
			library: &config.Library{
				Name:   "google-cloud-secret-manager",
				Output: "packages/google-cloud-secret-manager",
				Preview: &config.Library{
					Name: "google-cloud-secret-manager-preview",
				},
			},
			want: &config.Library{
				Name:   "google-cloud-secret-manager",
				Output: "packages/google-cloud-secret-manager",
				Preview: &config.Library{
					Name:   "google-cloud-secret-manager-preview",
					Output: "preview-packages/google-cloud-secret-manager",
				},
			},
		},
		{
			name: "preview library explicit output",
			library: &config.Library{
				Name:   "google-cloud-secret-manager",
				Output: "packages/google-cloud-secret-manager",
				Preview: &config.Library{
					Name:   "google-cloud-secret-manager-preview",
					Output: "some/other/path",
				},
			},
			want: &config.Library{
				Name:   "google-cloud-secret-manager",
				Output: "packages/google-cloud-secret-manager",
				Preview: &config.Library{
					Name:   "google-cloud-secret-manager-preview",
					Output: "some/other/path",
				},
			},
		},
		{
			name: "preview library filters OptArgsByAPI",
			library: &config.Library{
				Name:   "google-cloud-secret-manager",
				Output: "packages/google-cloud-secret-manager",
				Python: &config.PythonPackage{
					OptArgsByAPI: map[string][]string{
						"google/cloud/secretmanager/v1": {"opt1"},
						"google/cloud/secretmanager/v2": {"opt2"},
					},
				},
				Preview: &config.Library{
					Name: "google-cloud-secret-manager-preview",
					APIs: []*config.API{
						{Path: "google/cloud/secretmanager/v1"},
					},
				},
			},
			want: &config.Library{
				Name:   "google-cloud-secret-manager",
				Output: "packages/google-cloud-secret-manager",
				Python: &config.PythonPackage{
					OptArgsByAPI: map[string][]string{
						"google/cloud/secretmanager/v1": {"opt1"},
						"google/cloud/secretmanager/v2": {"opt2"},
					},
				},
				Preview: &config.Library{
					Name:   "google-cloud-secret-manager-preview",
					Output: "preview-packages/google-cloud-secret-manager",
					APIs: []*config.API{
						{Path: "google/cloud/secretmanager/v1"},
					},
					Python: &config.PythonPackage{
						OptArgsByAPI: map[string][]string{
							"google/cloud/secretmanager/v1": {"opt1"},
						},
					},
				},
			},
		},
		{
			name: "preview library merges Python config",
			library: &config.Library{
				Python: &config.PythonPackage{
					PythonDefault: config.PythonDefault{
						LibraryType: "GAPIC",
					},
					MetadataNameOverride: "secretmanager",
					OptArgsByAPI: map[string][]string{
						"google/cloud/secretmanager/v1": {"opt1"},
					},
				},
				Preview: &config.Library{
					APIs: []*config.API{
						{Path: "google/cloud/secretmanager/v1"},
					},
					Python: &config.PythonPackage{
						MetadataNameOverride: "secretmanager-preview",
					},
				},
			},
			want: &config.Library{
				Python: &config.PythonPackage{
					PythonDefault: config.PythonDefault{
						LibraryType: "GAPIC",
					},
					MetadataNameOverride: "secretmanager",
					OptArgsByAPI: map[string][]string{
						"google/cloud/secretmanager/v1": {"opt1"},
					},
				},
				Preview: &config.Library{
					Output: "",
					APIs: []*config.API{
						{Path: "google/cloud/secretmanager/v1"},
					},
					Python: &config.PythonPackage{
						MetadataNameOverride: "secretmanager-preview",
						OptArgsByAPI: map[string][]string{
							"google/cloud/secretmanager/v1": {"opt1"},
						},
					},
				},
			},
		},
		{
			name: "preview library OptArgsByAPI takes priority",
			library: &config.Library{
				Name:   "google-cloud-secret-manager",
				Output: "packages/google-cloud-secret-manager",
				Python: &config.PythonPackage{
					OptArgsByAPI: map[string][]string{
						"google/cloud/secretmanager/v1": {"stable-opt"},
					},
				},
				Preview: &config.Library{
					Name: "google-cloud-secret-manager-preview",
					APIs: []*config.API{
						{Path: "google/cloud/secretmanager/v1"},
					},
					Python: &config.PythonPackage{
						OptArgsByAPI: map[string][]string{
							"google/cloud/secretmanager/v1": {"preview-opt"},
						},
					},
				},
			},
			want: &config.Library{
				Name:   "google-cloud-secret-manager",
				Output: "packages/google-cloud-secret-manager",
				Python: &config.PythonPackage{
					OptArgsByAPI: map[string][]string{
						"google/cloud/secretmanager/v1": {"stable-opt"},
					},
				},
				Preview: &config.Library{
					Name:   "google-cloud-secret-manager-preview",
					Output: "preview-packages/google-cloud-secret-manager",
					APIs: []*config.API{
						{Path: "google/cloud/secretmanager/v1"},
					},
					Python: &config.PythonPackage{
						OptArgsByAPI: map[string][]string{
							"google/cloud/secretmanager/v1": {"preview-opt"},
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Fill(test.library)
			if err != nil {
				t.Fatalf("Fill() error = %v", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("Fill() returned diff (-want +got):\n%s", diff)
			}
		})
	}
}
