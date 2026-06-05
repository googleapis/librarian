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

package java

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestAdd(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want *config.Library
	}{
		{
			name: "standard cloud API",
			lib: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			want: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
				Version:       defaultVersion,
				CopyrightYear: "",
				Java: &config.JavaModule{
					ReleasedVersion: defaultReleasedVersion,
				},
			},
		},
		{
			name: "shopping API",
			lib: &config.Library{
				Name: "shopping-css",
				APIs: []*config.API{
					{Path: "google/shopping/css/v1"},
				},
			},
			want: &config.Library{
				Name: "shopping-css",
				APIs: []*config.API{
					{Path: "google/shopping/css/v1"},
				},
				Version:       defaultVersion,
				CopyrightYear: "",
				Java: &config.JavaModule{
					ArtifactID:      "google-shopping-css",
					GroupID:         "com.google.shopping",
					ReleasedVersion: defaultReleasedVersion,
				},
			},
		},
		{
			name: "maps API",
			lib: &config.Library{
				Name: "maps-routing",
				APIs: []*config.API{
					{Path: "google/maps/routing/v1"},
				},
			},
			want: &config.Library{
				Name: "maps-routing",
				APIs: []*config.API{
					{Path: "google/maps/routing/v1"},
				},
				Version:       defaultVersion,
				CopyrightYear: "",
				Java: &config.JavaModule{
					ArtifactID:      "google-maps-routing",
					GroupID:         "com.google.maps",
					ReleasedVersion: defaultReleasedVersion,
				},
			},
		},
		{
			name: "unrecognized non-cloud API",
			lib: &config.Library{
				Name: "foo-bar",
				APIs: []*config.API{
					{Path: "google/foo/bar/v1"},
				},
			},
			want: &config.Library{
				Name: "foo-bar",
				APIs: []*config.API{
					{Path: "google/foo/bar/v1"},
				},
				Version:       defaultVersion,
				CopyrightYear: "",
				Java: &config.JavaModule{
					ArtifactID:      "google-foo-bar",
					GroupID:         "please-configure-java-group-id",
					ReleasedVersion: defaultReleasedVersion,
				},
			},
		},
		{
			name: "ads API",
			lib: &config.Library{
				Name: "ads-admanager",
				APIs: []*config.API{
					{Path: "google/ads/admanager/v1"},
				},
			},
			want: &config.Library{
				Name: "ads-admanager",
				APIs: []*config.API{
					{Path: "google/ads/admanager/v1"},
				},
				Version:       defaultVersion,
				CopyrightYear: "",
				Java: &config.JavaModule{
					ArtifactID:      "google-ads-admanager",
					GroupID:         "com.google.api-ads",
					ReleasedVersion: defaultReleasedVersion,
				},
			},
		},
		{
			name: "cloud API outside google/cloud/",
			lib: &config.Library{
				Name: "iam",
				APIs: []*config.API{
					{Path: "google/iam/v3"},
				},
			},
			want: &config.Library{
				Name: "iam",
				APIs: []*config.API{
					{Path: "google/iam/v3"},
				},
				Version:       defaultVersion,
				CopyrightYear: "",
				Java: &config.JavaModule{
					ReleasedVersion: defaultReleasedVersion,
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Add(test.lib, googleapisDir)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDefaultLibraryName(t *testing.T) {
	for _, test := range []struct {
		api  string
		want string
	}{
		{"google/cloud/secretmanager/v1", "secretmanager"},
		{"google/api/serviceusage/v1", "serviceusage"},
		{"google/devtools/cloudbuild/v1", "cloudbuild"},
		{"google/pubsub/v1", "pubsub"},
		{"other/api/v1", "other-api"},
		{"google/cloud/datacatalog/lineage/v1", "datacatalog-lineage"},
		{"google/cloud/aiplatform/v1beta1", "aiplatform"},
		{"google/shopping/merchant/datasources/v1", "shopping-merchant-datasources"},
	} {
		t.Run(test.api, func(t *testing.T) {
			got := DefaultLibraryName(test.api)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
