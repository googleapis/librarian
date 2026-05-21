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
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
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
					GroupID:                  "com.google.shopping",
					DistributionNameOverride: "com.google.shopping:google-shopping-css",
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
					GroupID:                  "com.google.maps",
					DistributionNameOverride: "com.google.maps:google-maps-routing",
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
					GroupID:                  "please-configure-java-group-id",
					DistributionNameOverride: "please-configure-java-group-id:google-foo-bar",
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
					GroupID:                  "com.google.api-ads",
					DistributionNameOverride: "com.google.api-ads:google-ads-admanager",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Add(test.lib)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDefaultLibraryName(t *testing.T) {
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	srcs := &sources.Sources{
		Googleapis: googleapisDir,
	}
	for _, test := range []struct {
		api  string
		want string
	}{
		{"google/cloud/secretmanager/v1", "secretmanager"},
		{"google/cloud/apigeeconnect/v1", "apigeeconnect"},
		{"google/cloud/tasks/v2", "cloudtasks"},
		{"google/cloud/workflows/v1", "workflows"},
		{"google/maps/places/v1", "places"},
	} {
		t.Run(test.api, func(t *testing.T) {
			got, err := DefaultLibraryName(srcs, test.api)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDefaultLibraryName_Error(t *testing.T) {
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	srcs := &sources.Sources{
		Googleapis: googleapisDir,
	}
	for _, test := range []struct {
		name    string
		api     string
		wantErr error
	}{
		{
			name:    "missing configuration directory",
			api:     "google/cloud/nonexistent/v1",
			wantErr: ErrServiceNameNotFound,
		},
		{
			name:    "unallowed non-cloud API",
			api:     "google/unallowed/v1",
			wantErr: ErrAPIValidation,
		},
		{
			name:    "language-restricted API",
			api:     "google/ai/generativelanguage/v1",
			wantErr: ErrAPIValidation,
		},
		{
			name:    "non-standard service name (does not end in .googleapis.com)",
			api:     "google/cloud/nonstandardname/v1",
			wantErr: ErrAPIValidation,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := DefaultLibraryName(srcs, test.api)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("DefaultLibraryName(%q) error = %v, wantErr %v", test.api, err, test.wantErr)
			}
		})
	}
}
