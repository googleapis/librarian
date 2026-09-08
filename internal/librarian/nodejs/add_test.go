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

package nodejs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestAdd(t *testing.T) {
	in := &config.Library{
		Name: "google-cloud-secretmanager",
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1"},
		},
	}
	want := &config.Library{
		Name:    "google-cloud-secretmanager",
		Version: "0.0.0",
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1"},
		},
	}
	got := Add(nil, in)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAdd_CustomScopes(t *testing.T) {
	cfg := &config.Config{
		Default: &config.Default{
			Nodejs: &config.NodejsDefault{
				CustomScopes: map[string]string{
					"google/shopping/merchant": "@google-shopping",
					"google/shopping":          "@google-shopping",
					"google/maps":              "googlemaps",
					"google/chat":              "@google-apps/chat",
					"google/apps":              "@google-apps",
				},
			},
		},
	}

	for _, test := range []struct {
		name string
		in   *config.Library
		want *config.Library
	}{
		{
			name: "cloud API leaves package name empty",
			in: &config.Library{
				Name: "google-cloud-secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			want: &config.Library{
				Name:    "google-cloud-secretmanager",
				Version: "0.0.0",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
		},
		{
			name: "nested custom organization strips prefix and derives package name",
			in: &config.Library{
				Name: "google-shopping-merchant-loyaltycustomers",
				APIs: []*config.API{
					{Path: "google/shopping/merchant/loyaltycustomers/v1"},
				},
			},
			want: &config.Library{
				Name:    "google-shopping-merchant-loyaltycustomers",
				Version: "0.0.0",
				APIs: []*config.API{
					{Path: "google/shopping/merchant/loyaltycustomers/v1"},
				},
				Nodejs: &config.NodejsPackage{
					PackageName: "@google-shopping/loyaltycustomers",
				},
			},
		},
		{
			name: "prefix without leading @ is normalized",
			in: &config.Library{
				Name: "google-maps-routing",
				APIs: []*config.API{
					{Path: "google/maps/routing/v2"},
				},
			},
			want: &config.Library{
				Name:    "google-maps-routing",
				Version: "0.0.0",
				APIs: []*config.API{
					{Path: "google/maps/routing/v2"},
				},
				Nodejs: &config.NodejsPackage{
					PackageName: "@googlemaps/routing",
				},
			},
		},
		{
			name: "org containing slash returns exact package name when remainder empty",
			in: &config.Library{
				Name: "google-chat",
				APIs: []*config.API{
					{Path: "google/chat/v1"},
				},
			},
			want: &config.Library{
				Name:    "google-chat",
				Version: "0.0.0",
				APIs: []*config.API{
					{Path: "google/chat/v1"},
				},
				Nodejs: &config.NodejsPackage{
					PackageName: "@google-apps/chat",
				},
			},
		},
		{
			name: "general prefix derives package name from remainder",
			in: &config.Library{
				Name: "google-apps-meet",
				APIs: []*config.API{
					{Path: "google/apps/meet/v2"},
				},
			},
			want: &config.Library{
				Name:    "google-apps-meet",
				Version: "0.0.0",
				APIs: []*config.API{
					{Path: "google/apps/meet/v2"},
				},
				Nodejs: &config.NodejsPackage{
					PackageName: "@google-apps/meet",
				},
			},
		},
		{
			name: "unrecognized non-cloud API leaves package name empty",
			in: &config.Library{
				Name: "google-unrecognized-api",
				APIs: []*config.API{
					{Path: "google/unrecognized/api/v1"},
				},
			},
			want: &config.Library{
				Name:    "google-unrecognized-api",
				Version: "0.0.0",
				APIs: []*config.API{
					{Path: "google/unrecognized/api/v1"},
				},
			},
		},
		{
			name: "preserves explicitly configured package name",
			in: &config.Library{
				Name: "google-shopping-merchant-loyaltycustomers",
				APIs: []*config.API{
					{Path: "google/shopping/merchant/loyaltycustomers/v1"},
				},
				Nodejs: &config.NodejsPackage{
					PackageName: "@custom/override",
				},
			},
			want: &config.Library{
				Name:    "google-shopping-merchant-loyaltycustomers",
				Version: "0.0.0",
				APIs: []*config.API{
					{Path: "google/shopping/merchant/loyaltycustomers/v1"},
				},
				Nodejs: &config.NodejsPackage{
					PackageName: "@custom/override",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Add(cfg, test.in)
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
		{"google/cloud/secretmanager/v1", "google-cloud-secretmanager"},
		{"google/cloud/secretmanager/v1beta2", "google-cloud-secretmanager"},
		{"google/cloud/storage/v2alpha", "google-cloud-storage"},
		{"google/maps/addressvalidation/v1", "google-maps-addressvalidation"},
		{"google/api/v1", "google-api"},
		{"google/cloud/vision", "google-cloud-vision"},
	} {
		t.Run(test.api, func(t *testing.T) {
			got := DefaultLibraryName(test.api)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindExistingLibraryForNewAPI(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		libraries []*config.Library
		apiPath   string
		// The name of the library that should be returned, or empty if nill
		// should be returned.
		wantName string
	}{
		{
			name:      "no libraries",
			libraries: []*config.Library{},
			apiPath:   "google/cloud/test/v2",
		},
		{
			name: "exact match",
			libraries: []*config.Library{
				{
					Name: "google-cloud-other",
					APIs: []*config.API{{Path: "google/cloud/other"}},
				},
				{
					Name: "google-cloud-test",
					APIs: []*config.API{{Path: "google/cloud/test/v1"}},
				},
			},
			apiPath:  "google/cloud/test/v2",
			wantName: "google-cloud-test",
		},
		{
			name: "new API is prefix of existing after stripping versions",
			libraries: []*config.Library{
				{
					Name: "google-cloud-other",
					APIs: []*config.API{{Path: "google/cloud/other"}},
				},
				{
					Name: "google-cloud-test",
					APIs: []*config.API{{Path: "google/cloud/test/admin/v1"}},
				},
			},
			apiPath: "google/cloud/test/v2",
		},
		{
			name: "existing API is prefix of new one after stripping versions",
			libraries: []*config.Library{
				{
					Name: "google-cloud-other",
					APIs: []*config.API{{Path: "google/cloud/other"}},
				},
				{
					Name: "google-cloud-test",
					APIs: []*config.API{{Path: "google/cloud/test/v1"}},
				},
			},
			apiPath: "google/cloud/test/admin/v2",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := FindExistingLibraryForNewAPI(test.libraries, test.apiPath)
			gotName := ""
			if got != nil {
				gotName = got.Name
			}
			if diff := cmp.Diff(gotName, test.wantName); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
