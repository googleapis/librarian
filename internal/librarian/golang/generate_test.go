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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/testhelper"
)

const googleapisDir = "../../testdata/googleapis"

// TestGenerateLibraries performs simple testing that multiple libraries can
// be generated. Only the presence of a single expected file per library is
// performed; TestGenerate is responsible for more detailed testing of
// per-library generation.
func TestGenerateLibraries(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "protoc-gen-go")
	testhelper.RequireCommand(t, "protoc-gen-go-grpc")
	testhelper.RequireCommand(t, "protoc-gen-go_gapic")
	outDir := t.TempDir()
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}

	libraries := []*config.Library{
		{
			Name:          "google-cloud-secretmanager-v1",
			Version:       "0.1.0",
			ReleaseLevel:  "preview",
			CopyrightYear: "2025",
			Transport:     "grpc",
			APIs: []*config.API{
				{
					Path: "google/cloud/secretmanager/v1",
				},
			},
		},
		{
			Name:          "google-cloud-configdelivery-v1",
			Version:       "0.1.0",
			ReleaseLevel:  "preview",
			CopyrightYear: "2025",
			APIs: []*config.API{
				{
					Path: "google/cloud/configdelivery/v1",
				},
			},
		},
	}
	for _, library := range libraries {
		library.Output = outDir
	}
	if err := GenerateLibraries(t.Context(), libraries, googleapisDir); err != nil {
		t.Fatal(err)
	}
	// Just check that a README.md file has been created for each library.
	for _, library := range libraries {
		expectedReadme := filepath.Join(library.Output, library.Name, "README.md")
		_, err := os.Stat(expectedReadme)
		if err != nil {
			t.Errorf("Stat(%s) returned error: %v", expectedReadme, err)
		}
	}
}

func TestGenerateLibraries_Error(t *testing.T) {
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}

	libraries := []*config.Library{
		{
			Name:          "non-existent-api",
			APIs:          []*config.API{{Path: "google/cloud/non-existent/v1"}},
			Output:        t.TempDir(),
			Version:       "0.1.0",
			ReleaseLevel:  "preview",
			CopyrightYear: "2025",
			Transport:     "grpc",
		},
	}
	gotErr := GenerateLibraries(t.Context(), libraries, googleapisDir)
	if !errors.Is(gotErr, syscall.ENOENT) {
		t.Fatalf("expected %v, got %v", syscall.ENOENT, gotErr)
	}
}

func TestGenerate(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "protoc-gen-go")
	testhelper.RequireCommand(t, "protoc-gen-go-grpc")
	testhelper.RequireCommand(t, "protoc-gen-go_gapic")
	for _, test := range []struct {
		name         string
		libraryName  string
		apis         []*config.API
		transport    string
		releaseLevel string
		goModule     *config.GoModule
		want         []string
		removed      []string
	}{
		{
			name:        "basic",
			libraryName: "secretmanager",
			apis:        []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			goModule: &config.GoModule{
				GoAPIs: []*config.GoAPI{
					{
						ClientDirectory: "secretmanager",
						ImportPath:      "secretmanager/apiv1",
						Path:            "google/cloud/secretmanager/v1",
					},
				},
			},
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
			name:        "v2 module",
			libraryName: "secretmanager",
			apis:        []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			goModule: &config.GoModule{
				GoAPIs: []*config.GoAPI{
					{
						ClientDirectory: "secretmanager",
						ImportPath:      "secretmanager/v2/apiv1",
						Path:            "google/cloud/secretmanager/v1",
					},
				},
			},
			want: []string{
				"secretmanager/v2/apiv1/secret_manager_client.go",
			},
		},
		{
			name:        "delete paths",
			libraryName: "secretmanager",
			apis:        []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			goModule: &config.GoModule{
				DeleteGenerationOutputPaths: []string{"secretmanager/apiv1/secretmanagerpb"},
				GoAPIs: []*config.GoAPI{
					{
						ClientDirectory: "secretmanager",
						ImportPath:      "secretmanager/apiv1",
						Path:            "google/cloud/secretmanager/v1",
					},
				},
			},
			want: []string{
				"secretmanager/apiv1/secret_manager_client.go",
			},
			removed: []string{
				"secretmanager/apiv1/secretmanagerpb",
			},
		},
		{
			name:        "with transport and release level",
			libraryName: "secretmanager",
			apis:        []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			goModule: &config.GoModule{
				GoAPIs: []*config.GoAPI{
					{
						ClientDirectory: "secretmanager",
						ImportPath:      "secretmanager/apiv1",
						Path:            "google/cloud/secretmanager/v1",
					},
				},
			},
			transport:    "grpc+rest",
			releaseLevel: "ga",
			want: []string{
				"secretmanager/apiv1/secret_manager_client.go",
			},
		},
		{
			name:        "custom client directory",
			libraryName: "secretmanager",
			apis:        []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			goModule: &config.GoModule{
				GoAPIs: []*config.GoAPI{
					{
						ClientDirectory: "secretmanager",
						ImportPath:      "customdir/apiv1",
						Path:            "google/cloud/secretmanager/v1",
					},
				},
			},
			want: []string{
				"customdir/apiv1/secret_manager_client.go",
			},
		},
		{
			name:        "disable gapic",
			libraryName: "secretmanager",
			apis:        []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			goModule: &config.GoModule{
				GoAPIs: []*config.GoAPI{
					{
						ClientDirectory: "secretmanager",
						DisableGAPIC:    true,
						ImportPath:      "secretmanager/apiv1",
						Path:            "google/cloud/secretmanager/v1",
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
			apis:        []*config.API{{Path: "google/cloud/gkehub/v1"}},
			goModule: &config.GoModule{
				GoAPIs: []*config.GoAPI{
					{
						ClientDirectory: "gkehub",
						ImportPath:      "gkehub/apiv1",
						NestedProtos: []string{
							"configmanagement/configmanagement.proto",
							"multiclusteringress/multiclusteringress.proto",
						},
						Path: "google/cloud/gkehub/v1",
					},
				},
			},
			want: []string{
				"gkehub/apiv1/gke_hub_client.go",
				"gkehub/configmanagement/apiv1/configmanagementpb/configmanagement.pb.go",
				"gkehub/multiclusteringress/apiv1/multiclusteringresspb/multiclusteringress.pb.go",
			},
		},
		{
			name:        "no api",
			libraryName: "auth",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outdir := t.TempDir()
			library := &config.Library{
				Name:         test.libraryName,
				Version:      "1.0.0",
				Output:       outdir,
				APIs:         test.apis,
				Transport:    test.transport,
				ReleaseLevel: test.releaseLevel,
				Go:           test.goModule,
			}

			if err := generate(t.Context(), library, googleapisDir); err != nil {
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

	api, err := serviceconfig.Find(googleapisDir, library.APIs[0].Path, serviceconfig.LangGo)
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

func TestGenerateREADME_Skipped(t *testing.T) {
	dir := t.TempDir()
	moduleRoot := filepath.Join(dir, "secretmanager")
	if err := os.MkdirAll(moduleRoot, 0755); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Name:   "secretmanager",
		Output: dir,
		APIs:   []*config.API{{Path: "google/cloud/secretmanager/v1"}},
		Keep:   []string{"README.md"},
	}

	api, err := serviceconfig.Find(googleapisDir, library.APIs[0].Path, serviceconfig.LangGo)
	if err != nil {
		t.Fatal(err)
	}
	if err := generateREADME(library, api, moduleRoot); err != nil {
		t.Fatal(err)
	}
	// README doesn't exist because the generation is skipped.
	if _, err := os.Stat(filepath.Join(moduleRoot, "README.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("want README.md to not exist, got: %v", err)
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

func TestUpdateSnippetMetadata_Skipped(t *testing.T) {
	for _, test := range []struct {
		name     string
		library  *config.Library
		dir      string
		fileName string
		data     string
	}{
		{
			name: "skipped due to nested module",
			library: &config.Library{
				Name:    "bigquery",
				Version: "1.2.3",
				Go: &config.GoModule{
					NestedModule: "v2",
				},
			},
			dir:      filepath.Join("internal", "generated", "snippets", "bigquery", "v2", "apiv2"),
			fileName: "snippet_metadata.google.cloud.bigquery.v2.json",
			data: `{ 
 "clientLibrary": {
    "name": "cloud.google.com/go/bigquery/v2/apiv2",
    "version": "$VERSION",
    "language": "GO",
    "apis": [
      {
        "id": "google.cloud.bigquery.v2",
        "version": "v2"
      }
    ]
 }
}
`,
		},
		{
			name: "skipped due to non-snippets file name",
			library: &config.Library{
				Name:    "bigquery",
				Version: "1.2.3",
			},
			dir:      filepath.Join("internal", "generated", "snippets", "bigquery", "apiv1"),
			fileName: "non_metadata.google.cloud.bigquery.v1.json",
			data: `{ 
 "clientLibrary": {
    "name": "cloud.google.com/go/bigquery/apiv1",
    "version": "$VERSION",
    "language": "GO",
    "apis": [
      {
        "id": "google.cloud.bigquery.v1",
        "version": "v1"
      }
    ]
 }
}
`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			err := os.MkdirAll(filepath.Join(tmpDir, test.dir), 0755)
			if err != nil {
				t.Fatal(err)
			}
			metadataFile := filepath.Join(tmpDir, test.dir, test.fileName)
			if err := os.WriteFile(metadataFile, []byte(test.data), 0755); err != nil {
				t.Fatal(err)
			}
			if err := updateSnippetMetadata(test.library, tmpDir); err != nil {
				t.Fatal(err)
			}

			content, err := os.ReadFile(metadataFile)
			if err != nil {
				t.Fatal(err)
			}
			s := string(content)
			if !strings.Contains(s, "$VERSION") {
				t.Errorf("want version in snippet metadata, got:\n%s", s)
			}
		})
	}
}

func TestBuildGAPICImportPath(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		goAPI   *config.GoAPI
		want    string
	}{
		{
			name: "no override",
			library: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			},
			goAPI: &config.GoAPI{
				ClientDirectory: "secretmanager",
				ImportPath:      "secretmanager/apiv1",
				Path:            "google/cloud/secretmanager/v1",
			},
			want: "cloud.google.com/go/secretmanager/apiv1;secretmanager",
		},
		{
			name: "customize package override",
			library: &config.Library{
				Name: "storage",
			},
			goAPI: &config.GoAPI{
				ClientDirectory:       "internal",
				ClientPackageOverride: "storage",
				ImportPath:            "storage/internal/apiv2",
				Path:                  "google/storage/v2",
			},
			want: "cloud.google.com/go/storage/internal/apiv2;storage",
		},
		{
			name: "version suffix",
			library: &config.Library{
				Name: "monitoring",
			},
			goAPI: &config.GoAPI{
				ClientDirectory: "monitoring",
				ImportPath:      "monitoring/apiv3",
				Path:            "google/monitoring/v3",
				VersionSuffix:   "v2",
			},
			want: "cloud.google.com/go/monitoring/apiv3/v2;monitoring",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildGAPICImportPath(test.library, test.goAPI)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReleaseLevel_Success(t *testing.T) {
	for _, test := range []struct {
		name    string
		apiPath string
		version string
		want    string
	}{
		{
			name:    "ga",
			apiPath: "google/cloud/secretmanager/v1",
			version: "1.0.0",
			want:    "ga",
		},
		{
			name:    "stable with pre-GA version",
			apiPath: "google/cloud/secretmanager/v1",
			version: "0.11.0",
			want:    "beta",
		},
		{
			name:    "alpha",
			apiPath: "google/cloud/secretmanager/v1alpha1",
			want:    "alpha",
		},
		{
			name:    "beta",
			apiPath: "google/cloud/secretmanager/v1beta2",
			want:    "beta",
		},
		{
			name:    "alpha in api path",
			apiPath: "google/cloud/alphabet/v1",
			version: "1.0.0",
			want:    "ga",
		},
		{
			name:    "empty version",
			apiPath: "google/cloud/alphabet/v1",
			version: "",
			want:    "alpha",
		},
		{
			name:    "empty version with beta path",
			apiPath: "google/cloud/alphabet/v1beta1",
			version: "",
			want:    "beta",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := releaseLevel(test.apiPath, test.version)
			if err != nil {
				t.Error(err)
				return
			}
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
			got := transport(test.sc)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildGAPICOpts(t *testing.T) {
	for _, test := range []struct {
		name          string
		apiPath       string
		library       *config.Library
		goAPI         *config.GoAPI
		googleapisDir string
		want          []string
	}{
		{
			name:    "basic case with service and grpc configs",
			apiPath: "google/cloud/secretmanager/v1",
			library: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.3",
			},
			goAPI: &config.GoAPI{
				ClientDirectory: "secretmanager",
				ImportPath:      "secretmanager/apiv1",
				Path:            "google/cloud/secretmanager/v1",
			},
			googleapisDir: googleapisDir,
			want: []string{
				"go-gapic-package=cloud.google.com/go/secretmanager/apiv1;secretmanager",
				"metadata",
				"rest-numeric-enums",
				"api-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"transport=grpc+rest",
				"release-level=ga",
			},
		},
		{
			name:    "no rest numeric enums",
			apiPath: "google/cloud/secretmanager/v1",
			library: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.3",
			},
			goAPI: &config.GoAPI{
				ClientDirectory:    "secretmanager",
				ImportPath:         "secretmanager/apiv1",
				NoRESTNumericEnums: true,
				Path:               "google/cloud/secretmanager/v1",
			},
			googleapisDir: googleapisDir,
			want: []string{
				"go-gapic-package=cloud.google.com/go/secretmanager/apiv1;secretmanager",
				"metadata",
				"api-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"transport=grpc+rest",
				"release-level=ga",
			},
		},
		{
			name:    "beta release level from version",
			apiPath: "google/cloud/secretmanager/v1",
			library: &config.Library{
				Name:    "secretmanager",
				Version: "0.2.3",
			},
			goAPI: &config.GoAPI{
				ClientDirectory: "secretmanager",
				ImportPath:      "secretmanager/apiv1",
				Path:            "google/cloud/secretmanager/v1",
			},
			googleapisDir: googleapisDir,
			want: []string{
				"go-gapic-package=cloud.google.com/go/secretmanager/apiv1;secretmanager",
				"metadata",
				"rest-numeric-enums",
				"api-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"transport=grpc+rest",
				"release-level=beta",
			},
		},
		{
			name:    "transport override",
			apiPath: "google/cloud/gkehub/v1",
			library: &config.Library{
				Name:    "gkehub",
				Version: "1.2.3",
			},
			googleapisDir: googleapisDir,
			want: []string{
				"go-gapic-package=cloud.google.com/go/gkehub/apiv1;gkehub",
				"metadata",
				"rest-numeric-enums",
				"api-service-config=" + filepath.Join(googleapisDir, "google/cloud/gkehub/v1/gkehub_v1.yaml"),
				"transport=grpc+rest",
				"release-level=ga",
			},
		},
		{
			name:    "no metadata",
			apiPath: "google/cloud/gkehub/v1",
			library: &config.Library{
				Name:    "gkehub",
				Version: "1.2.3",
			},
			goAPI: &config.GoAPI{
				NoMetadata: true,
			},
			googleapisDir: googleapisDir,
			want: []string{
				"go-gapic-package=cloud.google.com/go/gkehub/apiv1;gkehub",
				"rest-numeric-enums",
				"api-service-config=" + filepath.Join(googleapisDir, "google/cloud/gkehub/v1/gkehub_v1.yaml"),
				"transport=grpc+rest",
				"release-level=ga",
			},
		},
		{
			name:    "generator features",
			apiPath: "google/cloud/bigquery/v2",
			library: &config.Library{
				Name:    "bigquery/v2",
				Version: "1.2.3",
				APIs:    []*config.API{{Path: "google/cloud/bigquery/v2"}},
			},
			goAPI: &config.GoAPI{
				ClientDirectory:          "bigquery",
				EnabledGeneratorFeatures: []string{"F_wrapper_types_for_page_size"},
				ImportPath:               "bigquery/v2/apiv2",
				Path:                     "google/cloud/bigquery/v2",
			},
			googleapisDir: googleapisDir,
			want: []string{
				"go-gapic-package=cloud.google.com/go/bigquery/v2/apiv2;bigquery",
				"metadata",
				"rest-numeric-enums",
				"F_wrapper_types_for_page_size",
				"api-service-config=" + filepath.Join(googleapisDir, "google/cloud/bigquery/v2/bigquery_v2.yaml"),
				"transport=grpc+rest",
				"release-level=ga",
			},
		},
		{
			name:    "no transport",
			apiPath: "google/cloud/apigeeconnect/v1",
			library: &config.Library{
				Name:    "apigeeconnect",
				Version: "1.2.3",
				APIs:    []*config.API{{Path: "google/cloud/apigeeconnect/v1"}},
			},
			goAPI: &config.GoAPI{
				ClientDirectory: "apigeeconnect",
				ImportPath:      "apigeeconnect/apiv1",
				Path:            "google/cloud/apigeeconnect/v1",
			},
			googleapisDir: googleapisDir,
			want: []string{
				"go-gapic-package=cloud.google.com/go/apigeeconnect/apiv1;apigeeconnect",
				"metadata",
				"rest-numeric-enums",
				"api-service-config=" + filepath.Join(googleapisDir, "google/cloud/apigeeconnect/v1/apigeeconnect_1.yaml"),
				"release-level=ga",
			},
		},
		{
			name:    "diregapic",
			apiPath: "google/cloud/compute/v1",
			library: &config.Library{
				Name:    "compute",
				Version: "1.2.3",
				APIs:    []*config.API{{Path: "google/cloud/compute/v1"}},
			},
			goAPI: &config.GoAPI{
				HasDiregapic:       true,
				NoRESTNumericEnums: true,
				Path:               "google/cloud/compute/v1",
			},
			googleapisDir: googleapisDir,
			want: []string{
				"go-gapic-package=cloud.google.com/go/compute/apiv1;compute",
				"metadata",
				"diregapic",
				"api-service-config=" + filepath.Join(googleapisDir, "google/cloud/compute/v1/compute_v1.yaml"),
				"transport=rest",
				"release-level=ga",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := buildGAPICOpts(test.apiPath, test.library, test.goAPI, test.googleapisDir)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
