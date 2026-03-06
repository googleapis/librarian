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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestFill(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		want    *config.Library
	}{
		{
			name: "fill defaults for non-nested api",
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
							ClientPackage: "secretmanager",
							ImportPath:    "secretmanager/apiv1",
							Path:          "google/cloud/secretmanager/v1",
						},
					},
				},
			},
		},
		{
			name: "fill defaults for nested api",
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
							ClientPackage: "analyticshub",
							ImportPath:    "bigquery/analyticshub/apiv1",
							Path:          "google/cloud/bigquery/analyticshub/v1",
						},
						{
							ClientPackage: "biglake",
							ImportPath:    "bigquery/biglake/apiv1",
							Path:          "google/cloud/bigquery/biglake/v1",
						},
					},
				},
			},
		},
		{
			name: "do not override library configs",
			library: &config.Library{
				Name: "example",
				APIs: []*config.API{{Path: "google/cloud/example/v1"}},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ClientPackage: "custom", // This value will be kept.
							Path:          "google/cloud/example/v1",
						},
					},
				},
			},
			want: &config.Library{
				Name: "example",
				APIs: []*config.API{{Path: "google/cloud/example/v1"}},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ClientPackage: "custom",
							ImportPath:    "example/apiv1",
							Path:          "google/cloud/example/v1",
						},
					},
				},
			},
		},
		{
			name: "merge defaults",
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
							ClientPackage:      "example",
							ImportPath:         "example/apiv1",
							NoRESTNumericEnums: true,
							Path:               "google/cloud/example/v1",
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
							Path:          "google/cloud/secretmanager/v1",
							ClientPackage: "customDir",
						},
					},
				},
			},
			apiPath: "google/cloud/secretmanager/v1",
			want: &config.GoAPI{
				Path:          "google/cloud/secretmanager/v1",
				ClientPackage: "customDir",
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
							Path:          "google/cloud/secretmanager/v1",
							ClientPackage: "customDir",
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

func TestDefaultImportPathAndClientPkg(t *testing.T) {
	for _, test := range []struct {
		name              string
		apiPath           string
		wantImportPath    string
		wantClientPkgName string
	}{
		{
			name:              "secretmanager",
			apiPath:           "google/cloud/secretmanager/v1",
			wantImportPath:    "secretmanager/apiv1",
			wantClientPkgName: "secretmanager",
		},
		{
			name:              "shopping",
			apiPath:           "google/shopping/merchant/quota/v1",
			wantImportPath:    "shopping/merchant/quota/apiv1",
			wantClientPkgName: "quota",
		},
		{
			name:              "non-versioned api path",
			apiPath:           "google/shopping/type",
			wantImportPath:    "",
			wantClientPkgName: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotImportPath, gotPkg := defaultImportPathAndClientPkg(test.apiPath)
			if diff := cmp.Diff(test.wantImportPath, gotImportPath); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantClientPkgName, gotPkg); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestClientPathFromLibraryRoot(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		goAPI   *config.GoAPI
		want    string
	}{
		{
			name: "no module path version",
			library: &config.Library{
				Go: &config.GoModule{},
			},
			goAPI: &config.GoAPI{
				ImportPath: "secretmanager/apiv1",
			},
			want: "secretmanager/apiv1",
		},
		{
			name: "with module path version v2",
			library: &config.Library{
				Go: &config.GoModule{
					ModulePathVersion: "v2",
				},
			},
			goAPI: &config.GoAPI{
				ImportPath: "secretmanager/v2/apiv1",
			},
			want: "secretmanager/apiv1",
		},
		{
			name: "with module path version v2 and api version v2",
			library: &config.Library{
				Go: &config.GoModule{
					ModulePathVersion: "v2",
				},
			},
			goAPI: &config.GoAPI{
				ImportPath: "secretmanager/v2/apiv2",
			},
			want: "secretmanager/apiv2",
		},
		{
			name: "with module path version v3",
			library: &config.Library{
				Go: &config.GoModule{
					ModulePathVersion: "v3",
				},
			},
			goAPI: &config.GoAPI{
				ImportPath: "secretmanager/v3/apiv1",
			},
			want: "secretmanager/apiv1",
		},
		{
			name: "library.Go is nil",
			library: &config.Library{
				Go: nil,
			},
			goAPI: &config.GoAPI{
				ImportPath: "secretmanager/apiv1",
			},
			want: "secretmanager/apiv1",
		},
		{
			name: "module path version not in import path",
			library: &config.Library{
				Go: &config.GoModule{
					ModulePathVersion: "v2",
				},
			},
			goAPI: &config.GoAPI{
				ImportPath: "secretmanager/apiv1",
			},
			want: "secretmanager/apiv1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := clientPathFromLibraryRoot(test.library, test.goAPI)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSnippetDirectory(t *testing.T) {
	output := t.TempDir()
	importPath := "example/apiv1"
	got := snippetDirectory(output, importPath)
	want := filepath.Join(output, "internal", "generated", "snippets", importPath)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestModulePath(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		want    string
	}{
		{
			name: "no module path version",
			library: &config.Library{
				Name: "pubsub",
			},
			want: "cloud.google.com/go/pubsub",
		},
		{
			name: "with module path version v2",
			library: &config.Library{
				Name: "pubsub",
				Go: &config.GoModule{
					ModulePathVersion: "v2",
				},
			},
			want: "cloud.google.com/go/pubsub/v2",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := modulePath(test.library)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInitModule(t *testing.T) {
	testhelper.RequireCommand(t, "go")
	outDir := t.TempDir()
	// Write an import so go mod tidy can generate a go.sum file.
	content := []byte("package main\nimport _ \"golang.org/x/text\"\n")
	if err := os.WriteFile(filepath.Join(outDir, "main.go"), content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := initModule(t.Context(), outDir, "example.com/testmod"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "go.mod")); err != nil {
		t.Errorf("expected go.mod to exist, but Stat failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "go.sum")); err != nil {
		t.Errorf("expected go.sum to exist, but Stat failed: %v", err)
	}
}
