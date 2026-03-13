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
			Name:          "google-cloud-configdelivery-v1",
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
		library.Output = outDir
	}
	if err := Generate(t.Context(), libraries, googleapisDir); err != nil {
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

func TestGenerate_Error(t *testing.T) {
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name      string
		libraries []*config.Library
		wantErr   error
	}{
		{
			name: "non existent api path",
			libraries: []*config.Library{
				{
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
			},
			wantErr: syscall.ENOENT,
		},
		{
			name: "no go api",
			libraries: []*config.Library{
				{
					Name:          "secretmanager",
					APIs:          []*config.API{{Path: "google/cloud/secretmanager/v1"}},
					Output:        t.TempDir(),
					Version:       "0.1.0",
					ReleaseLevel:  "preview",
					CopyrightYear: "2025",
				},
			},
			wantErr: errGoAPINotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outdir := t.TempDir()
			for _, library := range test.libraries {
				library.Output = outdir
			}

			gotErr := Generate(t.Context(), test.libraries, googleapisDir)
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
				DeleteGenerationOutputPaths: []string{"secretmanager/apiv1/secret_manager_client.go"},
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
						ImportPath:    "customdir/apiv1",
						Path:          "google/cloud/secretmanager/v1",
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
			outdir := t.TempDir()
			library := &config.Library{
				Name:         test.libraryName,
				Version:      "1.0.0",
				Output:       outdir,
				APIs:         test.apis,
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

func TestUpdateSnippetMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	library := &config.Library{
		Name:    "accessapproval",
		Output:  filepath.Join(tmpDir, "accessapproval"),
		Version: "1.2.3",
		APIs:    []*config.API{{Path: "google/cloud/accessapproval/v1"}},
		Go: &config.GoModule{
			GoAPIs: []*config.GoAPI{
				{
					ImportPath: "accessapproval/apiv1",
					Path:       "google/cloud/accessapproval/v1",
				},
			},
		},
	}

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
	if err := updateSnippetMetadata(library); err != nil {
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
		name      string
		protoOnly bool
		path      string
		fileName  string
		setup     func(base, path, data, fileName string)
	}{
		{
			name:     "skip non import path",
			path:     filepath.Join("internal", "generated", "snippets", "bigquery", "v2", "apiv2"),
			fileName: "snippet_metadata.google.cloud.bigquery.v2.json",
			setup: func(base, path, data, fileName string) {
				// We need to create this directory because snippets directory should exist before updating.
				if err := os.MkdirAll(filepath.Join(base, "internal/generated/snippets/bigquery/storage/apiv1"), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Join(base, path), 0755); err != nil {
					t.Fatal(err)
				}
				metadataFile := filepath.Join(base, path, fileName)
				if err := os.WriteFile(metadataFile, []byte(data), 0755); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:     "skip non-snippets file",
			path:     filepath.Join("internal", "generated", "snippets", "bigquery", "storage", "apiv1"),
			fileName: "non_metadata.google.cloud.bigquery.v1.json",
			setup: func(base, path, data, fileName string) {
				if err := os.MkdirAll(filepath.Join(base, path), 0755); err != nil {
					t.Fatal(err)
				}
				metadataFile := filepath.Join(base, path, fileName)
				if err := os.WriteFile(metadataFile, []byte(data), 0755); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:      "skip proto-only clients",
			protoOnly: true,
			// Do not create snippet directory to verify the function returns before
			// checking the existence of the directory.
		},
		{
			name: "snippet directory does not exist",
			// Do not create snippet directory to verify the function doesn't
			// return error in such ase.
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			library := &config.Library{
				Name:    "bigquery",
				Output:  filepath.Join(tmpDir, "bigquery"),
				Version: "1.2.3",
				APIs:    []*config.API{{Path: "google/cloud/bigquery/storage/v1"}},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ImportPath: "bigquery/storage/apiv1",
							Path:       "google/cloud/bigquery/storage/v1",
							ProtoOnly:  test.protoOnly,
						},
					},
				},
			}
			data := `{ 
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
`
			if test.setup != nil {
				test.setup(tmpDir, test.path, data, test.fileName)
			}
			if err := updateSnippetMetadata(library); err != nil {
				t.Fatal(err)
			}
			if test.setup == nil {
				// No need to check the content if the metadata file is not
				// created during setup function.
				return
			}
			metadataFile := filepath.Join(tmpDir, test.path, test.fileName)
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

func TestUpdateSnippetMetadata_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		setup   func(dir string)
		wantErr error
	}{
		{
			name: "no go api",
			library: &config.Library{
				Name:    "bigquery",
				Version: "1.2.3",
				APIs:    []*config.API{{Path: "google/cloud/bigquery/storage/v1"}},
			},
			wantErr: errGoAPINotFound,
		},
		{
			name: "no permission to read snippet directory",
			library: &config.Library{
				Name:    "bigquery",
				Version: "1.2.3",
				APIs:    []*config.API{{Path: "google/cloud/bigquery/storage/v1"}},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ImportPath: "bigquery/storage/apiv1",
							Path:       "google/cloud/bigquery/storage/v1",
						},
					},
				},
			},
			setup: func(dir string) {
				snippetDir := filepath.Join(dir, "internal", "generated", "snippets", "bigquery", "storage", "apiv1")
				if err := os.MkdirAll(snippetDir, 0755); err != nil {
					t.Fatal(err)
				}
				snippetFile := filepath.Join(snippetDir, "snippet_metadata.json")
				// Do not have the read permission.
				if err := os.WriteFile(snippetFile, []byte("{}"), 0333); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: syscall.EACCES,
		},
		{
			name: "no permission to update snippet directory",
			library: &config.Library{
				Name:    "bigquery",
				Version: "1.2.3",
				APIs:    []*config.API{{Path: "google/cloud/bigquery/storage/v1"}},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ImportPath: "bigquery/storage/apiv1",
							Path:       "google/cloud/bigquery/storage/v1",
						},
					},
				},
			},
			setup: func(dir string) {
				snippetDir := filepath.Join(dir, "internal", "generated", "snippets", "bigquery", "storage", "apiv1")
				if err := os.MkdirAll(snippetDir, 0755); err != nil {
					t.Fatal(err)
				}
				snippetFile := filepath.Join(snippetDir, "snippet_metadata.json")
				// Do not have the write permission.
				if err := os.WriteFile(snippetFile, []byte("{}"), 0555); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: syscall.EACCES,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			test.library.Output = filepath.Join(tmpDir, test.library.Name)
			if test.setup != nil {
				test.setup(tmpDir)
			}
			err := updateSnippetMetadata(test.library)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("updateSnippetMetadata() error = %v, wantErr %v", err, test.wantErr)
			}
		})
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
