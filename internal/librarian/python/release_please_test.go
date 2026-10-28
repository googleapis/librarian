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

package python

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestReleasePleaseExtraFiles(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		lib  *config.Library
		want []any
	}{
		{
			name: "single versioned API",
			lib: &config.Library{
				APIs: []*config.API{
					{Path: "google/cloud/foo/v1"},
				},
			},
			want: []any{
				"google/cloud/foo/gapic_version.py",
				"google/cloud/foo_v1/gapic_version.py",
				map[string]any{
					"jsonpath": "$.clientLibrary.version",
					"path":     "samples/generated_samples/snippet_metadata_google.cloud.foo.v1.json",
					"type":     "json",
				},
			},
		},
		{
			name: "versionless API",
			lib: &config.Library{
				APIs: []*config.API{
					{Path: "google/cloud/foo/type"},
				},
			},
			want: []any{
				"google/cloud/foo_type/gapic_version.py",
				map[string]any{
					"jsonpath": "$.clientLibrary.version",
					"path":     "samples/generated_samples/snippet_metadata_google.cloud.foo.type.json",
					"type":     "json",
				},
			},
		},
		{
			name: "multiple APIs sharing versionless path",
			lib: &config.Library{
				APIs: []*config.API{
					{Path: "google/cloud/foo/v1"},
					{Path: "google/cloud/foo/v2"},
				},
			},
			want: []any{
				"google/cloud/foo/gapic_version.py",
				"google/cloud/foo_v1/gapic_version.py",
				map[string]any{
					"jsonpath": "$.clientLibrary.version",
					"path":     "samples/generated_samples/snippet_metadata_google.cloud.foo.v1.json",
					"type":     "json",
				},
				"google/cloud/foo_v2/gapic_version.py",
				map[string]any{
					"jsonpath": "$.clientLibrary.version",
					"path":     "samples/generated_samples/snippet_metadata_google.cloud.foo.v2.json",
					"type":     "json",
				},
			},
		},
		{
			name: "nested API with version",
			lib: &config.Library{
				APIs: []*config.API{
					{Path: "google/shopping/merchant/loyaltycustomers/v1"},
				},
			},
			want: []any{
				"google/shopping/merchant_loyaltycustomers/gapic_version.py",
				"google/shopping/merchant_loyaltycustomers_v1/gapic_version.py",
				map[string]any{
					"jsonpath": "$.clientLibrary.version",
					"path":     "samples/generated_samples/snippet_metadata_google.shopping.merchant.loyaltycustomers.v1.json",
					"type":     "json",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := ReleasePleaseExtraFiles(test.lib)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFlattenNestedPath(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		lib  *config.Library
		want string
	}{
		{
			name: "single path segment after namespace",
			path: "google/cloud/secretmanager",
			lib:  &config.Library{},
			want: "google/cloud/secretmanager",
		},
		{
			name: "nested path segments under cloud namespace",
			path: "google/cloud/datacatalog/lineage",
			lib:  &config.Library{},
			want: "google/cloud/datacatalog_lineage",
		},
		{
			name: "deeply nested path segments under non-cloud namespace",
			path: "google/shopping/merchant/loyaltycustomer",
			lib:  &config.Library{},
			want: "google/shopping/merchant_loyaltycustomer",
		},
		{
			name: "single segment under non-cloud namespace",
			path: "google/shopping/type",
			lib:  &config.Library{},
			want: "google/shopping/type",
		},
		{
			name: "options override namespace and name",
			path: "google/shopping/merchant/loyaltycustomer",
			lib: &config.Library{
				Python: &config.PythonPackage{
					OptArgsByAPI: map[string][]string{
						"google/shopping/merchant/loyaltycustomer": {
							"python-gapic-namespace=custom.ns",
							"python-gapic-name=custom_name",
						},
					},
				},
			},
			want: "custom/ns/custom_name",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := flattenNestedPath(test.path, test.lib)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
