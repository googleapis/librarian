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
	"github.com/googleapis/librarian/internal/snippetmetadata"
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
							NoMetadata: true, // this value will be kept.
							Path:       "google/cloud/example/v1",
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
							ClientPackage: "example",
							ImportPath:    "example/apiv1",
							NoMetadata:    true,
							Path:          "google/cloud/example/v1",
						},
					},
				},
			},
		},
		{
			name: "proto only API",
			library: &config.Library{
				Name: "oslogin",
				APIs: []*config.API{{Path: "google/cloud/oslogin/common"}},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ImportPath: "oslogin/common",
							Path:       "google/cloud/oslogin/common",
							ProtoOnly:  true,
						},
					},
				},
			},
			want: &config.Library{
				Name: "oslogin",
				APIs: []*config.API{{Path: "google/cloud/oslogin/common"}},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ImportPath: "oslogin/common",
							Path:       "google/cloud/oslogin/common",
							ProtoOnly:  true,
						},
					},
				},
			},
		},
		{
			name: "no API",
			library: &config.Library{
				Name: "auth",
			},
			want: &config.Library{
				Name: "auth",
				Go:   &config.GoModule{},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Fill(test.library)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFill_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		wantErr error
	}{
		{
			name: "import path not set",
			library: &config.Library{
				Name: "oslogin",
				APIs: []*config.API{{Path: "google/cloud/oslogin/common"}},
			},
			wantErr: errImportPathNotFound,
		},
		{
			name: "client package not set",
			library: &config.Library{
				Name: "oslogin",
				APIs: []*config.API{{Path: "google/cloud/oslogin/common"}},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ImportPath: "oslogin/common",
							Path:       "google/cloud/oslogin/common",
						},
					},
				},
			},
			wantErr: errClientPackageNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := Fill(test.library)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("Fill() error = %v, wantErr %v", err, test.wantErr)
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

func TestClientPathFromRepoRoot(t *testing.T) {
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
			// This test case should not happen in production since
			// GoAPI is part of Go config.
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
			got := clientPathFromRepoRoot(test.library, test.goAPI)
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

func TestRepoRootPath(t *testing.T) {
	for _, test := range []struct {
		name        string
		libraryName string
		output      string
		want        string
	}{
		{
			name:        "no prefix on library output",
			libraryName: "secretmanager",
			output:      "secretmanager",
			want:        ".",
		},
		{
			name:        "prefix on library output",
			libraryName: "secretmanager",
			output:      "tmp/secretmanager",
			want:        "tmp",
		},
		{
			name:        "nested major version",
			libraryName: "bigquery/v2",
			output:      "bigquery/v2",
			want:        ".",
		},
		{
			name:        "prefix with nested major version",
			libraryName: "bigquery/v2",
			output:      "tmp/bigquery/v2",
			want:        "tmp",
		},
		{
			name:        "root module",
			libraryName: "root-module",
			output:      ".",
			want:        ".",
		},
		{
			name:        "root module has an absolute output path",
			libraryName: "root-module",
			output:      "/home/anyone/repo",
			want:        "/home/anyone/repo",
		},
		{
			name:        "library output has an absolute output path",
			libraryName: "library-name",
			output:      "/home/anyone/repo/lib",
			want:        "/home/anyone/repo",
		},
		{
			name:        "nested library output has an absolute output path",
			libraryName: "bigquery/v2",
			output:      "/home/anyone/repo/lib/v2",
			want:        "/home/anyone/repo",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := repoRootPath(test.output, test.libraryName)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDefaultOutput(t *testing.T) {
	for _, test := range []struct {
		name        string
		defaultOut  string
		libraryName string
		want        string
	}{
		{
			name:        "no prefix",
			defaultOut:  "",
			libraryName: "secretmanager",
			want:        "secretmanager",
		},
		{
			name:        "no prefix",
			defaultOut:  "prefix",
			libraryName: "secretmanager",
			want:        "prefix/secretmanager",
		},
		{
			name:        "library name with slashes",
			defaultOut:  "",
			libraryName: "bigquery/v2",
			want:        "bigquery/v2",
		},
		{
			name:        "prefix and library name with slashes",
			defaultOut:  "app/repo",
			libraryName: "bigquery/v2",
			want:        "app/repo/bigquery/v2",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DefaultOutput(test.libraryName, test.defaultOut)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
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

func TestDefaultLibraryName(t *testing.T) {
	for _, test := range []struct {
		name string
		api  string
		want string
	}{
		{
			name: "versioned api",
			api:  "google/cloud/secretmanager/v1",
			want: "secretmanager",
		},
		{
			name: "non versioned api",
			api:  "google/shopping/type",
			want: "type",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DefaultLibraryName(test.api)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUpdateSnippetDirectory(t *testing.T) {
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
	before := `{ 
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
	// json is formatted by writeMetadata function in snippetmetadata package.
	want := `{
  "clientLibrary": {
    "apis": [
      {
        "id": "google.cloud.accessapproval.v1",
        "version": "v1"
      }
    ],
    "language": "GO",
    "name": "cloud.google.com/go/accessapproval/apiv1",
    "version": "1.2.3"
  }
}`
	if err := os.WriteFile(metadataFile, []byte(before), 0755); err != nil {
		t.Fatal(err)
	}
	if err := updateSnippetDirectory(library, library.Output, library.Version); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(metadataFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(content)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUpdateSnippetDirectory_Skipped(t *testing.T) {
	for _, test := range []struct {
		name         string
		protoOnly    bool
		pathToDelete []string
		path         string
		fileName     string
		setup        func(base, path, data, fileName string)
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
			name:         "snippet directory does not exist",
			pathToDelete: []string{"../internal/generated/snippets/bigquery/storage/apiv1"},
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
					DeleteGenerationOutputPaths: test.pathToDelete,
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
			if err := updateSnippetDirectory(library, library.Output, library.Version); err != nil {
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

func TestUpdateSnippetDirectory_Error(t *testing.T) {
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
			name: "no client library field",
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
				if err := os.WriteFile(snippetFile, []byte("{}"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: snippetmetadata.ErrNoClientLibraryField,
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
				if err := os.WriteFile(snippetFile, []byte("{\"clientLibrary\": {\"language\": \"GO\"}}"), 0555); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: syscall.EACCES,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			output := filepath.Join(tmpDir, test.library.Name)
			if test.setup != nil {
				test.setup(tmpDir)
			}
			err := updateSnippetDirectory(test.library, output, test.library.Version)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("updateSnippetDirectory() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
