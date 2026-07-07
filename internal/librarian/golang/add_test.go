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

func TestAdd(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want *config.Library
	}{
		{
			name: "versioned api",
			lib: &config.Library{
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			},
			want: &config.Library{
				Version: defaultVersion,
				APIs:    []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			},
		},
		{
			name: "versionless api",
			lib: &config.Library{
				APIs: []*config.API{{Path: "google/shopping/type"}},
			},
			want: &config.Library{
				Version: defaultVersion,
				APIs: []*config.API{{
					Path: "google/shopping/type",
					Go: &config.GoAPI{
						ImportPath: "shopping/type/typepb",
						ProtoOnly:  true,
					},
				}},
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

func TestReleasePleaseExtraFiles(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want []any
	}{
		{
			name: "proto-only is skipped",
			lib: &config.Library{
				Name: "oslogin",
				APIs: []*config.API{
					{
						Path: "google/cloud/oslogin/common",
						Go: &config.GoAPI{
							ProtoOnly: true,
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "no snippets is skipped",
			lib: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Go: &config.GoAPI{
							NoSnippets: true,
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "derived import path and proto package",
			lib: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			want: []any{
				map[string]any{
					"jsonpath": "$.clientLibrary.version",
					"path":     "examples/apiv1/snippet_metadata.google.cloud.secretmanager.v1.json",
					"type":     "json",
				},
			},
		},
		{
			name: "explicit import path and proto package",
			lib: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Go: &config.GoAPI{
							ImportPath:   "secretmanager/custom/path",
							ProtoPackage: "google.cloud.secretmanager.custom.v1",
						},
					},
				},
			},
			want: []any{
				map[string]any{
					"jsonpath": "$.clientLibrary.version",
					"path":     "examples/custom/path/snippet_metadata.google.cloud.secretmanager.custom.v1.json",
					"type":     "json",
				},
			},
		},
		{
			name: "strips module path version",
			lib: &config.Library{
				Name: "pubsub",
				Go: &config.GoModule{
					ModulePathVersion: "v2",
				},
				APIs: []*config.API{
					{
						Path: "google/cloud/pubsub/v1",
						Go: &config.GoAPI{
							ImportPath: "pubsub/v2/apiv1",
						},
					},
				},
			},
			want: []any{
				map[string]any{
					"jsonpath": "$.clientLibrary.version",
					"path":     "examples/apiv1/snippet_metadata.google.cloud.pubsub.v1.json",
					"type":     "json",
				},
			},
		},
		{
			name: "deleted generation path is skipped",
			lib: &config.Library{
				Name: "secretmanager",
				Go: &config.GoModule{
					DeleteGenerationOutputPaths: []string{"examples/apiv1"},
				},
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			want: nil,
		},
		{
			name: "domain prefix in import path is handled",
			lib: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Go: &config.GoAPI{
							ImportPath: "cloud.google.com/go/secretmanager/apiv1",
						},
					},
				},
			},
			want: []any{
				map[string]any{
					"jsonpath": "$.clientLibrary.version",
					"path":     "examples/apiv1/snippet_metadata.google.cloud.secretmanager.v1.json",
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
