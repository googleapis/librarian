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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/testhelper"
)

const googleapisDir = "../../testdata/googleapis"

func TestGenerate(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "protoc-gen-go")
	testhelper.RequireCommand(t, "protoc-gen-go-grpc")
	testhelper.RequireCommand(t, "protoc-gen-go_gapic")
	for _, test := range []struct {
		name         string
		libraryName  string
		apiPath      string
		transport    string
		releaseLevel string
		goModule     *config.GoModule
		want         []string
		removed      []string
	}{
		{
			name: "basic",
			want: []string{
				"secretmanager/apiv1/secret_manager_client.go",
				"secretmanager/apiv1/secretmanagerpb/service.pb.go",
				"secretmanager/apiv1/version.go",
				"secretmanager/internal/version.go",
				"secretmanager/README.md",
			},
			removed: []string{
				"cloud.google.com",
			},
		},
		{
			name:     "v2 module",
			goModule: &config.GoModule{ModulePathVersion: "v2"},
			want: []string{
				"secretmanager/apiv1/secret_manager_client.go",
			},
			removed: []string{
				"secretmanager/v2",
			},
		},
		{
			name: "delete paths",
			goModule: &config.GoModule{
				DeleteGenerationOutputPaths: []string{"secretmanager/apiv1/secretmanagerpb"},
			},
			want: []string{
				"secretmanager/apiv1/secret_manager_client.go",
			},
			removed: []string{
				"secretmanager/apiv1/secretmanagerpb",
			},
		},
		{
			name:         "with transport and release level",
			transport:    "grpc+rest",
			releaseLevel: "ga",
			want: []string{
				"secretmanager/apiv1/secret_manager_client.go",
			},
		},
		{
			name: "client directory",
			goModule: &config.GoModule{
				GoAPIs: []*config.GoAPI{
					{
						Path:            "google/cloud/secretmanager/v1",
						ClientDirectory: "customdir",
					},
				},
			},
			want: []string{
				"customdir/apiv1/secret_manager_client.go",
			},
		},
		{
			name: "disable gapic",
			goModule: &config.GoModule{
				GoAPIs: []*config.GoAPI{
					{
						Path:         "google/cloud/secretmanager/v1",
						DisableGAPIC: true,
					},
				},
			},
			want: []string{
				"secretmanager/apiv1/secretmanagerpb/service.pb.go",
			},
			removed: []string{
				"secretmanager/apiv1/secret_manager_client.go",
			},
		},
		{
			name:        "nested protos",
			libraryName: "gkehub",
			apiPath:     "google/cloud/gkehub/v1",
			goModule: &config.GoModule{
				GoAPIs: []*config.GoAPI{
					{
						Path: "google/cloud/gkehub/v1",
						NestedProtos: []string{
							"configmanagement/configmanagement.proto",
							"multiclusteringress/multiclusteringress.proto",
						},
					},
				},
			},
			want: []string{
				"gkehub/apiv1/gke_hub_client.go",
				"gkehub/configmanagement/apiv1/configmanagementpb/configmanagement.pb.go",
				"gkehub/multiclusteringress/apiv1/multiclusteringresspb/multiclusteringress.pb.go",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outdir := t.TempDir()
			libraryName := test.libraryName
			if libraryName == "" {
				libraryName = "secretmanager"
			}
			apiPath := test.apiPath
			if apiPath == "" {
				apiPath = "google/cloud/secretmanager/v1"
			}
			library := &config.Library{
				Name:         libraryName,
				Output:       outdir,
				APIs:         []*config.API{{Path: apiPath}},
				Transport:    test.transport,
				ReleaseLevel: test.releaseLevel,
				Go:           test.goModule,
			}

			if err := Generate(t.Context(), library, googleapisDir); err != nil {
				t.Fatal(err)
			}
			if err := Format(t.Context(), library); err != nil {
				t.Fatal(err)
			}

			for _, path := range test.want {
				if _, err := os.Stat(filepath.Join(outdir, path)); err != nil {
					t.Errorf("missing %s", path)
				}
			}
			for _, path := range test.removed {
				if _, err := os.Stat(filepath.Join(outdir, path)); err == nil {
					t.Errorf("%s should not exist", path)
				}
			}
		})
	}
}

func TestFormat(t *testing.T) {
	testhelper.RequireCommand(t, "goimports")
	outDir := t.TempDir()
	goFile := filepath.Join(outDir, "test.go")
	unformatted := `package main

import (
"fmt"
"os"
)

func main() {
fmt.Println("Hello World")
}
`
	if err := os.WriteFile(goFile, []byte(unformatted), 0644); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Output: outDir,
	}
	if err := Format(t.Context(), library); err != nil {
		t.Fatal(err)
	}

	gotBytes, err := os.ReadFile(goFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(gotBytes)
	want := `package main

import (
	"fmt"
)

func main() {
	fmt.Println("Hello World")
}
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateREADME(t *testing.T) {
	dir := t.TempDir()
	moduleRoot := filepath.Join(dir, "secretmanager")
	if err := os.MkdirAll(moduleRoot, 0755); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Name:   "secretmanager",
		Output: dir,
		APIs:   []*config.API{{Path: "google/cloud/secretmanager/v1"}},
	}

	api, err := serviceconfig.Find(googleapisDir, library.APIs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if err := generateREADME(library, api, moduleRoot); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(moduleRoot, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)
	if !strings.Contains(s, "Secret Manager API") {
		t.Errorf("want title in README, got:\n%s", s)
	}
	if !strings.Contains(s, "cloud.google.com/go/secretmanager") {
		t.Errorf("want module path in README, got:\n%s", s)
	}
}

func TestUpdateSnippetMetadata(t *testing.T) {
	library := &config.Library{
		Name:    "accessapproval",
		Version: "1.2.3",
	}

	tmpDir := t.TempDir()
	metadataDir := filepath.Join(tmpDir, "internal", "generated", "snippets", "accessapproval", "apiv1")
	err := os.MkdirAll(metadataDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	metadataFile := filepath.Join(metadataDir, "snippet_metadata.google.cloud.accessapproval.v1.json")
	data := `{ 
 "clientLibrary": {
    "name": "cloud.google.com/go/accessapproval/apiv1",
    "version": "$VERSION",
    "language": "GO",
    "apis": [
      {
        "id": "google.cloud.accessapproval.v1",
        "version": "v1"
      }
    ]
 }
}
`
	if err := os.WriteFile(metadataFile, []byte(data), 0755); err != nil {
		return
	}
	if err := updateSnippetMetadata(library, tmpDir); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(metadataFile)
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)
	if !strings.Contains(s, "1.2.3") {
		t.Errorf("want version in snippet metadata, got:\n%s", s)
	}
}

func TestBuildGAPICImportPath(t *testing.T) {
	for _, test := range []struct {
		name    string
		apiPath string
		library *config.Library
		goAPI   *config.GoAPI
		want    string
	}{
		{
			name:    "single version without customize client directory",
			apiPath: "google/cloud/accessapproval/v1",
			library: &config.Library{
				Name: "accessapproval",
			},
			want: "cloud.google.com/go/accessapproval/apiv1;accessapproval",
		},
		{
			name:    "customize client directory",
			apiPath: "google/ai/generativelanguage/v1",
			library: &config.Library{
				Name: "ai",
			},
			goAPI: &config.GoAPI{
				ClientDirectory: "generativelanguage",
			},
			want: "cloud.google.com/go/generativelanguage/apiv1;generativelanguage",
		},
		{
			name:    "customize import path and client directory",
			apiPath: "google/ai/generativelanguage/v1",
			library: &config.Library{
				Name: "ai",
			},
			goAPI: &config.GoAPI{
				ClientDirectory: "generativelanguage",
				ImportPath:      "ai/generativelanguage",
			},
			want: "cloud.google.com/go/ai/generativelanguage/apiv1;generativelanguage",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildGAPICImportPath(test.apiPath, test.library, test.goAPI)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetReleaseLevel(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		sc   *serviceconfig.API
		want string
	}{
		{
			name: "not set",
			lib:  &config.Library{},
			sc:   nil,
			want: "",
		},
		{
			name: "found in library",
			lib: &config.Library{
				ReleaseLevel: "ga",
			},
			sc:   nil,
			want: "ga",
		},
		{
			name: "found in api",
			sc: &serviceconfig.API{
				ReleaseLevels: map[string]string{"go": "beta"},
			},
			want: "beta",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := getReleaseLevel(test.lib, test.sc)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetTransport(t *testing.T) {
	for _, test := range []struct {
		name string
		sc   *serviceconfig.API
		want string
	}{
		{
			name: "nil serviceconfig",
			sc:   nil,
			want: "grpc+rest",
		},
		{
			name: "empty serviceconfig",
			sc:   &serviceconfig.API{},
			want: "grpc+rest",
		},
		{
			name: "go specific transport",
			sc: &serviceconfig.API{
				Transports: map[string]serviceconfig.Transport{
					"go": serviceconfig.GRPC,
				},
			},
			want: "grpc",
		},
		{
			name: "other language transport",
			sc: &serviceconfig.API{
				Transports: map[string]serviceconfig.Transport{
					"python": serviceconfig.GRPC,
				},
			},
			want: "grpc+rest",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := getTransport(test.sc)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
