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
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sources"
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
	libraries := []*config.Library{
		{
			Name:          "secretmanager",
			Version:       "0.1.0",
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
			Name:          "secretmanager-multi",
			Version:       "0.1.0",
			CopyrightYear: "2025",
			APIs: []*config.API{
				{Path: "google/cloud/secretmanager/v1"},
				{Path: "google/cloud/secretmanager/v1beta2"}, // Different path to avoid file collision
			},
			Go: &config.GoModule{
				GoAPIs: []*config.GoAPI{
					{
						ClientPackage: "secretmanager",
						ImportPath:    "secretmanager-multi/apiv1",
						Path:          "google/cloud/secretmanager/v1",
					},
					{
						ClientPackage: "secretmanager",
						ImportPath:    "secretmanager-multi/apiv1beta2",
						Path:          "google/cloud/secretmanager/v1beta2",
					},
				},
			},
		},
		{
			Name:          "secretmanager-delete",
			Version:       "0.1.0",
			CopyrightYear: "2025",
			APIs: []*config.API{
				{Path: "google/cloud/secretmanager/v1"},
			},
			Go: &config.GoModule{
				DeleteGenerationOutputPaths: []string{"apiv1/secret_manager_client.go"},
				GoAPIs: []*config.GoAPI{
					{
						ClientPackage: "secretmanager",
						ImportPath:    "secretmanager-delete/apiv1",
						Path:          "google/cloud/secretmanager/v1",
					},
				},
			},
		},
		{
			Name:          "secretmanager",
			Version:       "0.1.0-preview.1",
			CopyrightYear: "2025",
			Output:        "preview/internal",
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
		library.Output = filepath.Join(repoRoot, library.Output, library.Name)
	}
	for _, library := range libraries {
		if err := Generate(t.Context(), library, &sources.Sources{Googleapis: googleapisDir}); err != nil {
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
				CopyrightYear: "2025",
			},
			wantErr: errGoAPINotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outdir := t.TempDir()
			test.library.Output = outdir

			gotErr := Generate(t.Context(), test.library, &sources.Sources{Googleapis: googleapisDir})
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Generate error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

// TestGenerate_MkdirAllError tests that Generate returns a wrapped error
// with the expected context when os.MkdirAll fails because the path is a file.
func TestGenerate_MkdirAllError(t *testing.T) {
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file_blocking_dir")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Name:          "secretmanager",
		Version:       "0.1.0",
		CopyrightYear: "2025",
		Output:        filePath,
		APIs: []*config.API{
			{
				Path: "google/cloud/secretmanager/v1",
			},
		},
	}

	gotErr := Generate(t.Context(), library, &sources.Sources{Googleapis: googleapisDir})
	if gotErr == nil {
		t.Fatal("expected error, got nil")
	}

}
func TestGenerateLibrary(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "protoc-gen-go")
	testhelper.RequireCommand(t, "protoc-gen-go-grpc")
	testhelper.RequireCommand(t, "protoc-gen-go_gapic")
	t.Parallel()
	for _, test := range []struct {
		name    string
		library *config.Library
		want    []string
		removed []string
	}{
		{
			name: "basic",
			library: &config.Library{
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
			name: "v2 module",
			library: &config.Library{
				Name: "dataproc",
				APIs: []*config.API{{Path: "google/cloud/dataproc/v1"}},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ClientPackage: "dataproc",
							ImportPath:    "dataproc/v2/apiv1",
							Path:          "google/cloud/dataproc/v1",
						},
					},
					ModulePathVersion: "v2",
				},
			},
			want: []string{
				"dataproc/apiv1/batch_controller_client.go",
			},
		},
		{
			name: "delete paths after generation",
			library: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
				Go: &config.GoModule{
					DeleteGenerationOutputPaths: []string{"apiv1/secret_manager_client.go"},
					GoAPIs: []*config.GoAPI{
						{
							ClientPackage: "secretmanager",
							ImportPath:    "secretmanager/apiv1",
							Path:          "google/cloud/secretmanager/v1",
						},
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
			name: "custom client directory",
			library: &config.Library{
				Name: "cloudtasks",
				APIs: []*config.API{{Path: "google/cloud/tasks/v2"}},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ClientPackage: "cloudtasks",
							ImportPath:    "cloudtasks/apiv2",
							Path:          "google/cloud/tasks/v2",
						},
					},
				},
			},
			want: []string{
				"cloudtasks/apiv2/cloud_tasks_client.go",
			},
		},
		{
			name: "proto only",
			library: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ClientPackage: "secretmanager",
							ProtoOnly:     true,
							ImportPath:    "secretmanager/apiv1",
							Path:          "google/cloud/secretmanager/v1",
						},
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
			name: "nested protos",
			library: &config.Library{
				Name: "containeranalysis",
				APIs: []*config.API{{Path: "google/devtools/containeranalysis/v1beta1"}},
				Keep: []string{"apiv1beta1/grafeas/grafeaspb/grafeas.pb.go"},
				Go: &config.GoModule{
					DeleteGenerationOutputPaths: []string{"google.golang.org"},
					GoAPIs: []*config.GoAPI{
						{
							ClientPackage: "containeranalysis",
							ImportPath:    "containeranalysis/apiv1beta1",
							NestedProtos:  []string{"grafeas/grafeas.proto"},
							Path:          "google/devtools/containeranalysis/v1beta1",
						},
					},
				},
			},
			want: []string{
				"containeranalysis/apiv1beta1/container_analysis_v1_beta1_client.go",
				"containeranalysis/apiv1beta1/grafeas/grafeaspb/grafeas.pb.go",
			},
		},
		{
			// This test verifies that a library with nested import paths can be
			// generated correctly.
			// In this case, the import path, firestore/apiv1/admin, is nested in
			// the other import path, firestore/apiv1.
			name: "nested import paths",
			library: &config.Library{
				Name: "firestore",
				APIs: []*config.API{
					{Path: "google/firestore/v1"},
					{Path: "google/firestore/admin/v1"},
				},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ClientPackage: "firestore",
							ImportPath:    "firestore/apiv1",
							Path:          "google/firestore/v1",
						},
						{
							ClientPackage: "apiv1",
							ImportPath:    "firestore/apiv1/admin",
							Path:          "google/firestore/admin/v1",
						},
					},
				},
			},
			want: []string{
				"firestore/apiv1/firestorepb/firestore.pb.go",
				"firestore/apiv1/admin/adminpb/firestore_admin.pb.go",
			},
		},
		{
			name: "no api",
			library: &config.Library{
				Name: "auth",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repoRoot := t.TempDir()
			if err := os.MkdirAll(filepath.Join(repoRoot, "internal"), 0777); err != nil {
				t.Fatal(err)
			}
			test.library.Output = filepath.Join(repoRoot, test.library.Name)
			for _, file := range test.library.Keep {
				src := filepath.Join("..", "..", "testdata/golang-generate", file)
				dst := filepath.Join(test.library.Output, file)
				if err := os.MkdirAll(filepath.Dir(dst), 0777); err != nil {
					t.Fatal(err)
				}
				if err := filesystem.CopyFile(src, dst); err != nil {
					t.Fatal(err)
				}
			}
			if err := Generate(t.Context(), test.library, &sources.Sources{Googleapis: googleapisDir}); err != nil {
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
	if _, err := os.Stat(filepath.Join(moduleRoot, "README.md")); !errors.Is(err, fs.ErrNotExist) {
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
			name:    "no snippets",
			apiPath: "google/cloud/gkehub/v1",
			goAPI: &config.GoAPI{
				ClientPackage: "gkehub",
				ImportPath:    "gkehub/apiv1",
				NoSnippets:    true,
				Path:          "google/cloud/gkehub/v1",
			},
			googleapisDir: googleapisDir,
			want: []string{
				"go-gapic-package=cloud.google.com/go/gkehub/apiv1;gkehub",
				"metadata",
				"omit-snippets",
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

func TestBuildGAPICOpts_Error(t *testing.T) {
	for _, test := range []struct {
		name          string
		apiPath       string
		goAPI         *config.GoAPI
		googleapisDir string
	}{
		{
			name:    "api not in allowlist",
			apiPath: "nonexistent/api/v1",
			goAPI: &config.GoAPI{
				ClientPackage: "nonexistent",
				ImportPath:    "nonexistent/apiv1",
				Path:          "nonexistent/api/v1",
			},
			googleapisDir: googleapisDir,
		},
		{
			name:    "api not allowed for go",
			apiPath: "google/cloud/asset/v1p1beta1",
			goAPI: &config.GoAPI{
				ClientPackage: "asset",
				ImportPath:    "asset/apiv1p1beta1",
				Path:          "google/cloud/asset/v1p1beta1",
			},
			googleapisDir: googleapisDir,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := buildGAPICOpts(test.apiPath, test.goAPI, test.googleapisDir)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestMoveGeneratedFiles(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(t *testing.T, tmpDir string) (outDir, apiDir, snippetDir string, lib *config.Library)
	}{
		{
			name: "moves files successfully",
			setup: func(t *testing.T, tmpDir string) (string, string, string, *config.Library) {
				repoRoot := filepath.Join(tmpDir, "repo")
				outDir := filepath.Join(repoRoot, "lib")
				srcDir := filepath.Join(outDir, "cloud.google.com", "go", "lib", "apiv1")
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
				return outDir, filepath.Join(outDir, "apiv1"), filepath.Join(repoRoot, snippetDirSuffix), lib
			},
		},
		{
			name: "nested major version",
			setup: func(t *testing.T, tmpDir string) (string, string, string, *config.Library) {
				repoRoot := filepath.Join(tmpDir, "repo")
				outDir := filepath.Join(repoRoot, "lib", "v2")
				srcDir := filepath.Join(outDir, "cloud.google.com", "go", "lib", "v2", "apiv2")
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
				return outDir, filepath.Join(outDir, "apiv2"), filepath.Join(repoRoot, snippetDirSuffix), lib
			},
		},
		{
			name: "library configured with a versioned module path",
			setup: func(t *testing.T, tmpDir string) (string, string, string, *config.Library) {
				repoRoot := filepath.Join(tmpDir, "repo")
				outDir := filepath.Join(repoRoot, "lib")
				srcDir := filepath.Join(outDir, "cloud.google.com", "go", "lib", "v2", "apiv1")
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
				return outDir, filepath.Join(outDir, "apiv1"), filepath.Join(repoRoot, "internal", "generated", "snippets", "lib", "apiv1"), lib
			},
		},
		{
			name: "no snippets",
			setup: func(t *testing.T, tmpDir string) (string, string, string, *config.Library) {
				repoRoot := filepath.Join(tmpDir, "repo")
				outDir := filepath.Join(repoRoot, "lib")
				srcDir := filepath.Join(outDir, "cloud.google.com", "go", "lib", "apiv1")
				if err := os.MkdirAll(srcDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package foo"), 0644); err != nil {
					t.Fatal(err)
				}
				// Even if snippet source exists in cloud.google.com/go, it should not be moved.
				snippetDirSuffix := filepath.Join("internal", "generated", "snippets", "lib", "apiv1")
				snippetSrcDir := filepath.Join(outDir, "cloud.google.com", "go", snippetDirSuffix)
				if err := os.MkdirAll(snippetSrcDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(snippetSrcDir, "snippet.go"), []byte("package internal"), 0644); err != nil {
					t.Fatal(err)
				}
				lib := &config.Library{
					Name: "lib",
					APIs: []*config.API{{Path: "lib/v1"}},
					Go: &config.GoModule{
						GoAPIs: []*config.GoAPI{{Path: "lib/v1", ImportPath: "lib/apiv1", NoSnippets: true}},
					},
				}
				return outDir, filepath.Join(outDir, "apiv1"), filepath.Join(repoRoot, snippetDirSuffix), lib
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			outDir, apiDir, snippetDir, lib := test.setup(t, tmpDir)
			err := moveGeneratedFiles(lib, lib.Go.GoAPIs[0], outDir, outDir)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(apiDir, "main.go")); err != nil {
				t.Errorf("expected main.go to exist, got err: %v", err)
			}
			if lib.Go.GoAPIs[0].NoSnippets {
				if _, err := os.Stat(filepath.Join(snippetDir, "snippet.go")); !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("expected snippet.go to not exist, got err: %v", err)
				}
			} else {
				if _, err := os.Stat(filepath.Join(snippetDir, "snippet.go")); err != nil {
					t.Errorf("expected snippet.go to exist, got err: %v", err)
				}
			}
		})
	}
}
