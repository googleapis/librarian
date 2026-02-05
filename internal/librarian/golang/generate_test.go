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
	testhelper.RequireCommand(t, "gofmt")
	outDir := t.TempDir()
	goFile := filepath.Join(outDir, "test.go")
	if err := os.WriteFile(goFile, []byte("package main\n\n\n\nfunc main() { \n\nfmt.Print(\"hello world\") \n\n}"), 0644); err != nil {
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

func main() {

	fmt.Print("hello world")

}
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
