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

// TestGenerate performs simple testing that multiple libraries can be
// generated. Only the presence of a single expected file per library is
// performed; TestGenerateLibrary is responsible for more detailed testing of
// per-library generation.
func TestGenerate(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "protoc-gen-go")
	testhelper.RequireCommand(t, "protoc-gen-go-grpc")
	testhelper.RequireCommand(t, "protoc-gen-go_gapic")
	repoRoot := t.TempDir()
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	// We need to create snippet directory root before running generation.
	if err := os.MkdirAll(filepath.Join(repoRoot, "internal"), 0777); err != nil {
		t.Fatal(err)
	}

	libraries := []*config.Library{
		{
			Name:          "secretmanager",
			Version:       "0.1.0",
			ReleaseLevel:  "preview",
			CopyrightYear: "2025",
			APIs: []*config.API{
				{
					Path: "google/cloud/secretmanager/v1",
				},
			},
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
		{
			Name:          "configdelivery",
			Version:       "0.1.0",
			ReleaseLevel:  "preview",
			CopyrightYear: "2025",
			APIs: []*config.API{
				{
					Path: "google/cloud/configdelivery/v1",
				},
			},
			Go: &config.GoModule{
				GoAPIs: []*config.GoAPI{
					{
						ClientPackage: "configdelivery",
						ImportPath:    "configdelivery/apiv1",
						Path:          "google/cloud/configdelivery/v1",
					},
				},
			},
		},
	}
	for _, library := range libraries {
		library.Output = filepath.Join(repoRoot, library.Name)
	}
	for _, library := range libraries {
		if err := Generate(t.Context(), library, googleapisDir); err != nil {
			t.Fatal(err)
		}
	}
	// Just check that a README.md file has been created for each library.
	for _, library := range libraries {
		expectedReadme := filepath.Join(library.Output, "README.md")
		_, err := os.Stat(expectedReadme)
		if err != nil {
			t.Errorf("Stat(%s) returned error: %v", expectedReadme, err)
		}
	}
}

func TestGenerate_Error(t *testing.T) {
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		library *config.Library
		wantErr error
	}{
		{
			name: "non existent api path",
			library: &config.Library{
				Name:          "non-existent-api",
				APIs:          []*config.API{{Path: "google/cloud/non-existent/v1"}},
				Output:        t.TempDir(),
				Version:       "0.1.0",
				ReleaseLevel:  "preview",
				CopyrightYear: "2025",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ClientPackage: "non-existent",
							ImportPath:    "non-existent/apiv1",
							Path:          "google/cloud/non-existent/v1",
						},
					},
				},
			},
			wantErr: syscall.ENOENT,
		},
		{
			name: "no go api",
			library: &config.Library{
				Name:          "secretmanager",
				APIs:          []*config.API{{Path: "google/cloud/secretmanager/v1"}},
				Output:        t.TempDir(),
				Version:       "0.1.0",
				ReleaseLevel:  "preview",
				CopyrightYear: "2025",
			},
			wantErr: errGoAPINotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outdir := t.TempDir()
			test.library.Output = outdir

			gotErr := Generate(t.Context(), test.library, googleapisDir)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Generate error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestGenerateLibrary(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "protoc-gen-go")
	testhelper.RequireCommand(t, "protoc-gen-go-grpc")
	testhelper.RequireCommand(t, "protoc-gen-go_gapic")
	for _, test := range []struct {
		name         string
		libraryName  string
		apis         []*config.API
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
						ClientPackage: "secretmanager",
						ImportPath:    "secretmanager/apiv1",
						Path:          "google/cloud/secretmanager/v1",
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
						ClientPackage: "secretmanager",
						ImportPath:    "secretmanager/v2/apiv1",
						Path:          "google/cloud/secretmanager/v1",
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
				DeleteGenerationOutputPaths: []string{"apiv1/secret_manager_client.go"},
				GoAPIs: []*config.GoAPI{
					{
						ClientPackage: "secretmanager",
						ImportPath:    "secretmanager/apiv1",
						Path:          "google/cloud/secretmanager/v1",
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
			name:        "with release level",
			libraryName: "secretmanager",
			apis:        []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			goModule: &config.GoModule{
				GoAPIs: []*config.GoAPI{
					{
						ClientPackage: "secretmanager",
						ImportPath:    "secretmanager/apiv1",
						Path:          "google/cloud/secretmanager/v1",
					},
				},
			},
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
						ClientPackage: "secretmanager",
						ImportPath:    "secretmanager/customdir/apiv1",
						Path:          "google/cloud/secretmanager/v1",
					},
				},
			},
			want: []string{
				"secretmanager/customdir/apiv1/secret_manager_client.go",
			},
		},
		{
			name:        "disable gapic",
			libraryName: "secretmanager",
			apis:        []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			goModule: &config.GoModule{
				GoAPIs: []*config.GoAPI{
					{
						ClientPackage: "secretmanager",
						ProtoOnly:     true,
						ImportPath:    "secretmanager/apiv1",
						Path:          "google/cloud/secretmanager/v1",
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
						ClientPackage: "gkehub",
						ImportPath:    "gkehub/apiv1",
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
			repoRoot := t.TempDir()
			if err := os.MkdirAll(filepath.Join(repoRoot, "internal"), 0777); err != nil {
				t.Fatal(err)
			}
			library := &config.Library{
				Name:         test.libraryName,
				Version:      "1.0.0",
				Output:       filepath.Join(repoRoot, test.libraryName),
				APIs:         test.apis,
				ReleaseLevel: test.releaseLevel,
				Go:           test.goModule,
			}

			if err := Generate(t.Context(), library, googleapisDir); err != nil {
				t.Fatal(err)
			}

			for _, path := range test.want {
				if _, err := os.Stat(filepath.Join(repoRoot, path)); err != nil {
					t.Errorf("missing %s", path)
				}
			}
			for _, path := range test.removed {
				if _, err := os.Stat(filepath.Join(repoRoot, path)); err == nil {
					t.Errorf("%s should not exist", path)
				}
			}
		})
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

	api, err := serviceconfig.Find(googleapisDir, library.APIs[0].Path, config.LanguageGo)
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

	api, err := serviceconfig.Find(googleapisDir, library.APIs[0].Path, config.LanguageGo)
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

func TestBuildGAPICImportPath(t *testing.T) {
	for _, test := range []struct {
		name  string
		goAPI *config.GoAPI
		want  string
	}{
		{
			name: "no override",
			goAPI: &config.GoAPI{
				ClientPackage: "secretmanager",
				ImportPath:    "secretmanager/apiv1",
				Path:          "google/cloud/secretmanager/v1",
			},
			want: "cloud.google.com/go/secretmanager/apiv1;secretmanager",
		},
		{
			name: "customize package override",
			goAPI: &config.GoAPI{
				ClientPackage: "storage",
				ImportPath:    "storage/internal/apiv2",
				Path:          "google/storage/v2",
			},
			want: "cloud.google.com/go/storage/internal/apiv2;storage",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildGAPICImportPath(test.goAPI)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHasRESTNumericEnums(t *testing.T) {
	for _, test := range []struct {
		name string
		sc   *serviceconfig.API
		want bool
	}{
		{
			name: "all languages do not have REST enums",
			sc: &serviceconfig.API{
				NoRESTNumericEnums: map[string]bool{
					config.LanguageAll: true,
				},
			},
		},
		{
			name: "go language do not have REST enums",
			sc: &serviceconfig.API{
				NoRESTNumericEnums: map[string]bool{
					config.LanguageGo: true,
				},
			},
		},
		{
			name: "another language do not have REST enums",
			sc: &serviceconfig.API{
				NoRESTNumericEnums: map[string]bool{
					config.LanguagePython: true,
				},
			},
			want: true,
		},
		{
			name: "empty map",
			sc:   &serviceconfig.API{},
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := hasRESTNumericEnums(test.sc)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReleaseLevel(t *testing.T) {
	for _, test := range []struct {
		name string
		sc   *serviceconfig.API
		want string
	}{
		{
			name: "empty release levels",
			sc:   &serviceconfig.API{},
			want: "ga",
		},
		{
			name: "release levels do not have go",
			sc: &serviceconfig.API{
				ReleaseLevels: map[string]string{config.LanguagePython: "beta"},
			},
			want: "ga",
		},
		{
			name: "alpha",
			sc: &serviceconfig.API{
				ReleaseLevels: map[string]string{config.LanguageGo: "alpha"},
			},
			want: "alpha",
		},
		{
			name: "beta",
			sc: &serviceconfig.API{
				ReleaseLevels: map[string]string{config.LanguageGo: "beta"},
			},
			want: "beta",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := releaseLevel(test.sc)
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
		want serviceconfig.Transport
	}{
		{
			name: "nil serviceconfig",
			sc:   nil,
			want: serviceconfig.GRPCRest,
		},
		{
			name: "empty serviceconfig",
			sc:   &serviceconfig.API{},
			want: serviceconfig.GRPCRest,
		},
		{
			name: "go specific transport",
			sc: &serviceconfig.API{
				Transports: map[string]serviceconfig.Transport{
					config.LanguageGo: serviceconfig.GRPC,
				},
			},
			want: serviceconfig.GRPC,
		},
		{
			name: "other language transport",
			sc: &serviceconfig.API{
				Transports: map[string]serviceconfig.Transport{
					config.LanguagePython: serviceconfig.GRPC,
				},
			},
			want: serviceconfig.GRPCRest,
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
		goAPI         *config.GoAPI
		googleapisDir string
		want          []string
	}{
		{
			name:    "basic case with service and grpc configs",
			apiPath: "google/cloud/secretmanager/v1",
			goAPI: &config.GoAPI{
				ClientPackage: "secretmanager",
				ImportPath:    "secretmanager/apiv1",
				Path:          "google/cloud/secretmanager/v1",
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
			apiPath: "google/cloud/bigquery/v2",
			goAPI: &config.GoAPI{
				ClientPackage: "bigquery",
				ImportPath:    "bigquery/v2/apiv2",
				Path:          "google/cloud/bigquery/v2",
			},
			googleapisDir: googleapisDir,
			want: []string{
				"go-gapic-package=cloud.google.com/go/bigquery/v2/apiv2;bigquery",
				"metadata",
				"api-service-config=" + filepath.Join(googleapisDir, "google/cloud/bigquery/v2/bigquery_v2.yaml"),
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/bigquery/v2/bigquery_grpc_service_config.json"),
				"transport=grpc+rest",
				"release-level=alpha",
			},
		},
		{
			name:    "transport override",
			apiPath: "google/cloud/gkehub/v1",
			goAPI: &config.GoAPI{
				ClientPackage: "gkehub",
				ImportPath:    "gkehub/apiv1",
				Path:          "google/cloud/gkehub/v1",
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
			goAPI: &config.GoAPI{
				ClientPackage: "gkehub",
				ImportPath:    "gkehub/apiv1",
				NoMetadata:    true,
				Path:          "google/cloud/gkehub/v1",
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
			goAPI: &config.GoAPI{
				ClientPackage:            "bigquery",
				EnabledGeneratorFeatures: []string{"F_wrapper_types_for_page_size"},
				ImportPath:               "bigquery/v2/apiv2",
				Path:                     "google/cloud/bigquery/v2",
			},
			googleapisDir: googleapisDir,
			want: []string{
				"go-gapic-package=cloud.google.com/go/bigquery/v2/apiv2;bigquery",
				"metadata",
				"F_wrapper_types_for_page_size",
				"api-service-config=" + filepath.Join(googleapisDir, "google/cloud/bigquery/v2/bigquery_v2.yaml"),
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/bigquery/v2/bigquery_grpc_service_config.json"),
				"transport=grpc+rest",
				"release-level=alpha",
			},
		},
		{
			name:    "no transport",
			apiPath: "google/cloud/apigeeconnect/v1",
			goAPI: &config.GoAPI{
				ClientPackage: "apigeeconnect",
				ImportPath:    "apigeeconnect/apiv1",
				Path:          "google/cloud/apigeeconnect/v1",
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
			goAPI: &config.GoAPI{
				ClientPackage: "compute",
				ImportPath:    "compute/apiv1",
				DIREGAPIC:     true,
				Path:          "google/cloud/compute/v1",
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
			got, err := buildGAPICOpts(test.apiPath, test.goAPI, test.googleapisDir)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMoveSnippetDirectory(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(t *testing.T, tmpDir string) (snippetDir string, lib *config.Library)
	}{
		{
			name: "snippet directory does not exist",
			setup: func(t *testing.T, tmpDir string) (string, *config.Library) {
				lib := &config.Library{
					Name:   "foo",
					Output: filepath.Join(tmpDir, "repo", "lib"),
				}
				return filepath.Join(tmpDir, "nonexistent"), lib
			},
		},
		{
			name: "snippet directory exists",
			setup: func(t *testing.T, tmpDir string) (string, *config.Library) {
				repoRoot := filepath.Join(tmpDir, "repo")
				snippetDir := filepath.Join(tmpDir, "snippets")
				if err := os.MkdirAll(snippetDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(snippetDir, "main.go"), []byte("package main"), 0644); err != nil {
					t.Fatal(err)
				}
				internalDir := filepath.Join(repoRoot, "internal")
				if err := os.MkdirAll(internalDir, 0755); err != nil {
					t.Fatal(err)
				}
				lib := &config.Library{
					Name:   "foo",
					Output: filepath.Join(repoRoot, "lib"),
				}
				return snippetDir, lib
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			snippetDir, lib := test.setup(t, tmpDir)
			err := moveSnippetDirectory(lib, snippetDir)
			if err != nil {
				t.Fatal(err)
			}

			if test.name == "snippet directory does not exist" {
				return
			}
			repoRoot := filepath.Join(tmpDir, "repo")
			movedFile := filepath.Join(repoRoot, "internal", "main.go")
			if _, err := os.Stat(movedFile); err != nil {
				t.Errorf("expected moved file %q to exist, got err: %v", movedFile, err)
			}
		})
	}
}

func TestMoveGeneratedFiles(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(t *testing.T, tmpDir string) (outDir, snippetDir string, lib *config.Library)
	}{
		{
			name: "moves files successfully",
			setup: func(t *testing.T, tmpDir string) (string, string, *config.Library) {
				repoRoot := filepath.Join(tmpDir, "repo")
				outDir := filepath.Join(repoRoot, "lib")
				srcDir := filepath.Join(outDir, "cloud.google.com", "go", "lib")
				if err := os.MkdirAll(srcDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package foo"), 0644); err != nil {
					t.Fatal(err)
				}
				snippetDirSuffix := filepath.Join("internal", "generated", "snippets", "lib", "apiv1")
				snippetDir := filepath.Join(outDir, "cloud.google.com", "go", snippetDirSuffix)
				if err := os.MkdirAll(snippetDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(snippetDir, "snippet.go"), []byte("package internal"), 0644); err != nil {
					t.Fatal(err)
				}
				lib := &config.Library{
					Name: "lib",
					APIs: []*config.API{{Path: "lib/v1"}},
					Go: &config.GoModule{
						GoAPIs: []*config.GoAPI{{Path: "lib/v1", ImportPath: "lib/apiv1"}},
					},
				}
				return outDir, filepath.Join(repoRoot, snippetDirSuffix), lib
			},
		},
		{
			name: "nested major version",
			setup: func(t *testing.T, tmpDir string) (string, string, *config.Library) {
				repoRoot := filepath.Join(tmpDir, "repo")
				outDir := filepath.Join(repoRoot, "lib", "v2")
				srcDir := filepath.Join(outDir, "cloud.google.com", "go", "lib", "v2")
				if err := os.MkdirAll(srcDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package foo"), 0644); err != nil {
					t.Fatal(err)
				}
				snippetDirSuffix := filepath.Join("internal", "generated", "snippets", "lib", "v2", "apiv2")
				snippetDir := filepath.Join(outDir, "cloud.google.com", "go", snippetDirSuffix)
				if err := os.MkdirAll(snippetDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(snippetDir, "snippet.go"), []byte("package internal"), 0644); err != nil {
					t.Fatal(err)
				}
				lib := &config.Library{
					Name: "lib/v2",
					APIs: []*config.API{{Path: "lib/v2"}},
					Go: &config.GoModule{
						GoAPIs: []*config.GoAPI{
							{Path: "lib/v2", ImportPath: "lib/v2/apiv2"},
						},
					},
				}
				return outDir, filepath.Join(repoRoot, snippetDirSuffix), lib
			},
		},
		{
			name: "library configured with a versioned module path",
			setup: func(t *testing.T, tmpDir string) (string, string, *config.Library) {
				repoRoot := filepath.Join(tmpDir, "repo")
				outDir := filepath.Join(repoRoot, "lib")
				srcDir := filepath.Join(outDir, "cloud.google.com", "go", "lib", "v2")
				if err := os.MkdirAll(srcDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package foo"), 0644); err != nil {
					t.Fatal(err)
				}
				snippetDirSuffix := filepath.Join("internal", "generated", "snippets", "lib", "v2", "apiv1")
				snippetDir := filepath.Join(outDir, "cloud.google.com", "go", snippetDirSuffix)
				if err := os.MkdirAll(snippetDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(snippetDir, "snippet.go"), []byte("package internal"), 0644); err != nil {
					t.Fatal(err)
				}
				lib := &config.Library{
					Name: "lib",
					APIs: []*config.API{{Path: "lib/v1"}},
					Go: &config.GoModule{
						GoAPIs: []*config.GoAPI{
							{Path: "lib/v1", ImportPath: "lib/v2/apiv1"},
						},
						ModulePathVersion: "v2",
					},
				}
				return outDir, filepath.Join(repoRoot, "internal", "generated", "snippets", "lib", "apiv1"), lib
			},
		},
		{
			name: "library configured paths to delete after generation",
			setup: func(t *testing.T, tmpDir string) (string, string, *config.Library) {
				repoRoot := filepath.Join(tmpDir, "repo")
				outDir := filepath.Join(repoRoot, "lib")

				srcDir := filepath.Join(outDir, "cloud.google.com", "go", "lib")
				if err := os.MkdirAll(srcDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package foo"), 0644); err != nil {
					t.Fatal(err)
				}

				snippetDir := filepath.Join(outDir, "cloud.google.com", "go", "internal", "generated", "snippets", "lib")
				if err := os.MkdirAll(snippetDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(snippetDir, "snippet.go"), []byte("package internal"), 0644); err != nil {
					t.Fatal(err)
				}
				pathToDelete := filepath.Join(outDir, "delete")
				if err := os.MkdirAll(pathToDelete, 0755); err != nil {
					t.Fatal(err)
				}

				internalDir := filepath.Join(repoRoot, "internal")
				if err := os.MkdirAll(internalDir, 0755); err != nil {
					t.Fatal(err)
				}

				lib := &config.Library{
					Name:   "lib",
					APIs:   []*config.API{{Path: "lib/v1"}},
					Output: outDir,
					Go: &config.GoModule{
						DeleteGenerationOutputPaths: []string{"delete"},
					},
				}
				return outDir, "", lib
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			outDir, snippetDir, lib := test.setup(t, tmpDir)
			err := moveGeneratedFiles(lib, outDir)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(outDir, "main.go")); err != nil {
				t.Errorf("expected main.go to exist, got err: %v", err)
			}
			if _, err := os.Stat(filepath.Join(snippetDir, "snippet.go")); err != nil {
				t.Errorf("expected snippet.go to exist, got err: %v", err)
			}
			if lib.Go == nil || len(lib.Go.DeleteGenerationOutputPaths) == 0 {
				return
			}
			if _, err := os.Stat(filepath.Join(outDir, lib.Go.DeleteGenerationOutputPaths[0])); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("expected %v to be deleted", lib.Go.DeleteGenerationOutputPaths[0])
			}
		})
	}
}
