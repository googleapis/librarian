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

package ruby

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestTidy(t *testing.T) {
	for _, test := range []struct {
		name string
		in   *config.Library
		want *config.Library
	}{
		{
			name: "nil configurations",
			in: &config.Library{
				Name: "google-cloud-secretmanager-v1",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
					},
				},
			},
			want: &config.Library{
				Name: "google-cloud-secretmanager-v1",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
					},
				},
			},
		},
		{
			name: "nilifies empty api ruby config",
			in: &config.Library{
				Name: "google-cloud-secretmanager-v1",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Ruby: &config.RubyAPI{},
					},
				},
			},
			want: &config.Library{
				Name: "google-cloud-secretmanager-v1",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
					},
				},
			},
		},
		{
			name: "nilifies empty library ruby config",
			in: &config.Library{
				Name: "google-cloud-secretmanager",
				Ruby: &config.RubyPackage{},
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
					},
				},
			},
			want: &config.Library{
				Name: "google-cloud-secretmanager",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
					},
				},
			},
		},
		{
			name: "nilifies empty cloud opts",
			in: &config.Library{
				Name: "google-cloud-secretmanager",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Ruby: &config.RubyAPI{
							RubyCloudOpts: &config.RubyCloudOpts{},
						},
					},
				},
			},
			want: &config.Library{
				Name: "google-cloud-secretmanager",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
					},
				},
			},
		},
		{
			name: "sorts and deduplicates additional protos",
			in: &config.Library{
				Name: "google-cloud-secretmanager-v1",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Ruby: &config.RubyAPI{
							AdditionalProtos: []string{"b.proto", "a.proto", "a.proto"},
						},
					},
				},
			},
			want: &config.Library{
				Name: "google-cloud-secretmanager-v1",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Ruby: &config.RubyAPI{
							AdditionalProtos: []string{"a.proto", "b.proto"},
						},
					},
				},
			},
		},
		{
			name: "retains non-empty ruby configurations",
			in: &config.Library{
				Name: "google-cloud-secretmanager",
				Ruby: &config.RubyPackage{
					WrapperOf: []string{"google-cloud-secretmanager-v1"},
				},
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Ruby: &config.RubyAPI{
							RubyCloudOpts: &config.RubyCloudOpts{
								EnvPrefix: "SECRET_MANAGER",
							},
						},
					},
				},
			},
			want: &config.Library{
				Name: "google-cloud-secretmanager",
				Ruby: &config.RubyPackage{
					WrapperOf: []string{"google-cloud-secretmanager-v1"},
				},
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Ruby: &config.RubyAPI{
							RubyCloudOpts: &config.RubyCloudOpts{
								EnvPrefix: "SECRET_MANAGER",
							},
						},
					},
				},
			},
		},
		{
			name: "retains delete generation output paths in library ruby config",
			in: &config.Library{
				Name: "google-cloud-compute-v1",
				Ruby: &config.RubyPackage{
					DeleteGenerationOutputPaths: []string{"snippets"},
				},
				APIs: []*config.API{
					{
						Path: "google/cloud/compute/v1",
					},
				},
			},
			want: &config.Library{
				Name: "google-cloud-compute-v1",
				Ruby: &config.RubyPackage{
					DeleteGenerationOutputPaths: []string{"snippets"},
				},
				APIs: []*config.API{
					{
						Path: "google/cloud/compute/v1",
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Tidy(test.in)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
