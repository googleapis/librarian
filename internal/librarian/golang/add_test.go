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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestAdd(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		goPackage string
		lib       *config.Library
		want      *config.Library
	}{
		{
			name:      "versioned api matching default import path",
			goPackage: "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb;secretmanagerpb",
			lib: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			},
			want: &config.Library{
				Name:    "secretmanager",
				Version: defaultVersion,
				APIs:    []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			},
		},
		{
			name:      "versioned api differing from default import path",
			goPackage: "cloud.google.com/go/developerknowledge/apiv1/developerknowledgepb",
			lib: &config.Library{
				Name: "developerknowledge",
				APIs: []*config.API{{Path: "google/developers/knowledge/v1"}},
			},
			want: &config.Library{
				Name:    "developerknowledge",
				Version: defaultVersion,
				APIs: []*config.API{{
					Path: "google/developers/knowledge/v1",
					Go: &config.GoAPI{
						ImportPath: "developerknowledge/apiv1",
					},
				}},
			},
		},
		{
			name:      "versioned nested api",
			goPackage: "cloud.google.com/go/maps/addressvalidation/apiv1/addressvalidationpb;addressvalidationpb",
			lib: &config.Library{
				Name: "maps",
				APIs: []*config.API{{Path: "google/maps/addressvalidation/v1"}},
			},
			want: &config.Library{
				Name:    "maps",
				Version: defaultVersion,
				APIs: []*config.API{{
					Path: "google/maps/addressvalidation/v1",
				}},
			},
		},
		{
			name:      "versionless api",
			goPackage: "cloud.google.com/go/shopping/type/typepb;typepb",
			lib: &config.Library{
				Name: "shopping",
				APIs: []*config.API{{Path: "google/shopping/type"}},
			},
			want: &config.Library{
				Name:    "shopping",
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
		{
			name:      "versionless single-segment api",
			goPackage: "cloud.google.com/go/type/typepb;typepb",
			lib: &config.Library{
				Name: "type",
				APIs: []*config.API{{Path: "google/type"}},
			},
			want: &config.Library{
				Name:    "type",
				Version: defaultVersion,
				APIs: []*config.API{{
					Path: "google/type",
					Go: &config.GoAPI{
						ImportPath: "type/typepb",
						ProtoOnly:  true,
					},
				}},
			},
		},
		{
			name:      "preserves existing version",
			goPackage: "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb;secretmanagerpb",
			lib: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.0",
				APIs:    []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			},
			want: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.0",
				APIs:    []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			},
		},
		{
			name:      "preserves existing go api config",
			goPackage: "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb;secretmanagerpb",
			lib: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{{
					Path: "google/cloud/secretmanager/v1",
					Go: &config.GoAPI{
						ImportPath: "custom/import/path",
					},
				}},
			},
			want: &config.Library{
				Name:    "secretmanager",
				Version: defaultVersion,
				APIs: []*config.API{{
					Path: "google/cloud/secretmanager/v1",
					Go: &config.GoAPI{
						ImportPath: "custom/import/path",
					},
				}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			googleapisDir := t.TempDir()
			for _, api := range test.lib.APIs {
				protoDir := filepath.Join(googleapisDir, api.Path)
				if err := os.MkdirAll(protoDir, 0o755); err != nil {
					t.Fatal(err)
				}
				content := fmt.Sprintf("option go_package = %q;", test.goPackage)
				if err := os.WriteFile(filepath.Join(protoDir, "service.proto"), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, err := Add(test.lib, googleapisDir)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAdd_Error(t *testing.T) {
	t.Parallel()
	lib := &config.Library{
		Name: "secretmanager",
		APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
	}
	if _, err := Add(lib, t.TempDir()); err == nil {
		t.Fatal("expected error when proto directory is missing, got nil")
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
