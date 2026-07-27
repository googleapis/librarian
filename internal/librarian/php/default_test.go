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

package php

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestFillStagingSubdir(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want *config.Library
	}{
		{
			name: "preserve existing staging subdir",
			lib: &config.Library{
				Name: "google-cloud-secretmanager",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						PHP: &config.PHPAPI{
							StagingSubdir: "custom-dir",
						},
					},
				},
			},
			want: &config.Library{
				Name: "google-cloud-secretmanager",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						PHP: &config.PHPAPI{
							StagingSubdir: "custom-dir",
						},
					},
				},
			},
		},
		{
			name: "fill empty staging subdir",
			lib: &config.Library{
				Name: "google-cloud-secretmanager",
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
						PHP: &config.PHPAPI{
							StagingSubdir: "v1",
						},
					},
				},
			},
		},
		{
			name: "multiple apis mixed",
			lib: &config.Library{
				Name: "google-cloud-example",
				APIs: []*config.API{
					{
						Path: "google/cloud/example/v1",
					},
					{
						Path: "google/cloud/example/v2",
						PHP: &config.PHPAPI{
							StagingSubdir: "custom-v2",
						},
					},
					{
						Path: "google/cloud/example/v3beta1",
					},
				},
			},
			want: &config.Library{
				Name: "google-cloud-example",
				APIs: []*config.API{
					{
						Path: "google/cloud/example/v1",
						PHP: &config.PHPAPI{
							StagingSubdir: "v1",
						},
					},
					{
						Path: "google/cloud/example/v2",
						PHP: &config.PHPAPI{
							StagingSubdir: "custom-v2",
						},
					},
					{
						Path: "google/cloud/example/v3beta1",
						PHP: &config.PHPAPI{
							StagingSubdir: "v3beta1",
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := fillStagingSubdir(test.lib)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
