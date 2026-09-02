// Copyright 2025 Google LLC
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

package librarian

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestAddLibraryCommand(t *testing.T) {
	copyrightYear := strconv.Itoa(time.Now().Year())
	for _, test := range []struct {
		name                   string
		apiPath                string
		initialLibraries       []*config.Library
		wantFinalLibraries     []*config.Library
		wantGeneratedOutputDir string
	}{
		{
			name:                   "create new library",
			apiPath:                "google/cloud/secretmanager/v1",
			initialLibraries:       []*config.Library{},
			wantGeneratedOutputDir: "newlib-output",
			wantFinalLibraries: []*config.Library{
				{
					Name:          "google-cloud-secretmanager-v1",
					CopyrightYear: copyrightYear,
					Version:       defaultVersion, // added by language-specific add
				},
			},
		},
		{
			name:    "create new library and tidy existing",
			apiPath: "google/cloud/orgpolicy/v1",
			initialLibraries: []*config.Library{
				{
					Name: "existinglib",
					APIs: []*config.API{
						{Path: "google/cloud/secretmanager/v1"},
					},
				},
			},
			wantGeneratedOutputDir: "newlib-output",
			wantFinalLibraries: []*config.Library{
				{
					Name: "existinglib",
					APIs: []*config.API{
						{Path: "google/cloud/secretmanager/v1"},
					},
				},
				{
					Name:          "google-cloud-orgpolicy-v1",
					CopyrightYear: copyrightYear,
					Version:       defaultVersion, // added by language-specific add
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			googleapisDir, err := filepath.Abs("../testdata/googleapis")
			if err != nil {
				t.Fatal(err)
			}
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)

			cfg := sample.Config()
			cfg.Default.Output = "output"
			cfg.Libraries = test.initialLibraries
			cfg.Sources.Googleapis.Dir = googleapisDir
			if err := yaml.Write(config.LibrarianYAML, cfg); err != nil {
				t.Fatal(err)
			}
			if err := runAdd(t.Context(), cfg, test.apiPath, ""); err != nil {
				t.Fatal(err)
			}

			gotCfg, err := yaml.Read[config.Config](config.LibrarianYAML)
			if err != nil {
				t.Fatal(err)
			}

			sort.Slice(gotCfg.Libraries, func(i, j int) bool {
				return gotCfg.Libraries[i].Name < gotCfg.Libraries[j].Name
			})

			if diff := cmp.Diff(test.wantFinalLibraries, gotCfg.Libraries); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAddLibraryCommand_Error(t *testing.T) {
	googleapisDir, err := filepath.Abs("../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	defaultCfg := func() *config.Config {
		cfg := sample.Config()
		cfg.Default.Output = "output"
		cfg.Sources.Googleapis.Dir = googleapisDir
		return cfg
	}

	for _, test := range []struct {
		name    string
		apiPath string
		cfg     *config.Config
		wantErr error
	}{
		{
			name:    "fail create existing library",
			apiPath: "google/cloud/secretmanager/v1",
			cfg: func() *config.Config {
				cfg := defaultCfg()
				cfg.Libraries = []*config.Library{
					{
						Name: "google-cloud-secretmanager-v1",
					},
				}
				return cfg
			}(),
			wantErr: errLibraryAlreadyExists,
		},
		{
			name:    "fail on non-existent api path",
			apiPath: "google/cloud/nonexistent/v1",
			cfg:     defaultCfg(),
			wantErr: errAPINotFound,
		},
		{
			name:    "fail on empty api path",
			apiPath: "",
			cfg:     defaultCfg(),
			wantErr: errAPINotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)

			if err := yaml.Write(config.LibrarianYAML, test.cfg); err != nil {
				t.Fatal(err)
			}
			err = runAdd(t.Context(), test.cfg, test.apiPath, "")
			if !errors.Is(err, test.wantErr) {
				t.Errorf("expected error %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestAddCommand(t *testing.T) {
	googleapisDir, err := filepath.Abs("../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name     string
		args     []string
		wantName string
	}{
		{
			name:     "single API",
			args:     []string{"google/cloud/secretmanager/v1"},
			wantName: "google-cloud-secretmanager-v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)
			if err := os.WriteFile(filepath.Join(tmpDir, "versions.txt"), nil, 0o644); err != nil {
				t.Fatal(err)
			}

			cfg := sample.Config()
			cfg.Default.Output = "output"
			cfg.Libraries = nil
			cfg.Sources.Googleapis.Dir = googleapisDir
			if err := yaml.Write(config.LibrarianYAML, cfg); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"librarian", "add"}, test.args...)
			if err := Run(t.Context(), args...); err != nil {
				t.Fatal(err)
			}

			gotCfg, err := yaml.Read[config.Config](config.LibrarianYAML)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := FindLibrary(gotCfg, test.wantName); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAddCommand_Error(t *testing.T) {
	googleapisDir, err := filepath.Abs("../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name    string
		args    []string
		wantErr error
	}{
		{
			name:    "no args",
			wantErr: errWrongAPICount,
		},
		{
			name: "multiple args",
			args: []string{
				"google/cloud/secretmanager/v1",
				"google/cloud/secretmanager/v1beta2",
				"google/cloud/secrets/v1beta1",
			},
			wantErr: errWrongAPICount,
		},
		{
			name:    "non-existent API",
			args:    []string{"google/cloud/nonexistent/v1"},
			wantErr: errAPINotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)
			if err := os.WriteFile(filepath.Join(tmpDir, "versions.txt"), nil, 0o644); err != nil {
				t.Fatal(err)
			}

			cfg := sample.Config()
			cfg.Default.Output = "output"
			cfg.Libraries = nil
			cfg.Sources.Googleapis.Dir = googleapisDir
			if err := yaml.Write(config.LibrarianYAML, cfg); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"librarian", "add"}, test.args...)
			err := Run(t.Context(), args...)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("want error %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestValidateAPIPathExistence(t *testing.T) {
	googleapisDir, err := filepath.Abs("../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateAPIPathExistence(googleapisDir, "google/cloud/secretmanager/v1"); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestValidateAPIPathExistence_Error(t *testing.T) {
	googleapisDir, err := filepath.Abs("../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		apiPath string
		wantErr error
	}{
		{
			name:    "empty API path",
			apiPath: "",
			wantErr: errAPINotFound,
		},
		{
			name:    "non-existent API",
			apiPath: "google/cloud/nonexistent/v1",
			wantErr: errAPINotFound,
		},
		{
			name:    "directory traversal API",
			apiPath: "../../etc",
			wantErr: errAPINotFound,
		},
		{
			name:    "file path instead of directory",
			apiPath: "google/cloud/secretmanager/v1/secretmanager.proto",
			wantErr: errAPINotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateAPIPathExistence(googleapisDir, test.apiPath)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("expected error %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestAddLibrary(t *testing.T) {
	for _, test := range []struct {
		name     string
		apiPath  string
		wantName string
		wantAPIs []*config.API
	}{
		{
			name:     "library with single API",
			apiPath:  "google/cloud/storage/v1",
			wantName: "google-cloud-storage-v1",
			wantAPIs: []*config.API{
				{
					Path: "google/cloud/storage/v1",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)
			if err := os.WriteFile(filepath.Join(tmpDir, "versions.txt"), nil, 0o644); err != nil {
				t.Fatal(err)
			}

			cfg := sample.Config()
			cfg.Libraries = []*config.Library{
				{
					Name:   "existinglib",
					Output: "output/existinglib",
				},
			}
			if err := yaml.Write(config.LibrarianYAML, cfg); err != nil {
				t.Fatal(err)
			}
			gotName, cfg, err := addLibrary(cfg, test.apiPath, "")
			if err != nil {
				t.Fatal(err)
			}
			if gotName != test.wantName {
				t.Errorf("gotName = %q, want %q", gotName, test.wantName)
			}
			if len(cfg.Libraries) != 2 {
				t.Errorf("libraries count = %d, want 2", len(cfg.Libraries))
			}

			found, err := FindLibrary(cfg, test.wantName)
			if err != nil {
				t.Fatal(err)
			}
			// [config.LanguageFake] has language-specific mutation in add.
			if found.Version != defaultVersion {
				t.Errorf("version = %q, want %q", found.Version, defaultVersion)
			}
			if diff := cmp.Diff(test.wantAPIs, found.APIs); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAddLibrary_ExistingLibrary(t *testing.T) {
	for _, test := range []struct {
		name     string
		apiPath  string
		cfg      *config.Config
		wantName string
		wantCfg  *config.Config
	}{
		{
			name:    "update existing library (go)",
			apiPath: "google/cloud/secretmanager/v1beta2",
			cfg: &config.Config{
				Language: config.LanguageGo,
				Libraries: []*config.Library{
					{
						Name:    "secretmanager",
						Version: "1.2.3",
						APIs: []*config.API{
							{Path: "google/cloud/secretmanager/v1"},
						},
					},
				},
			},
			wantName: "secretmanager",
			wantCfg: &config.Config{
				Language: config.LanguageGo,
				Libraries: []*config.Library{
					{
						Name:    "secretmanager",
						Version: "1.2.3",
						APIs: []*config.API{
							{Path: "google/cloud/secretmanager/v1"},
							{Path: "google/cloud/secretmanager/v1beta2"},
						},
					},
				},
			},
		},
		{
			name: "update existing library (python)",
			// The API path here deliberately doesn't match the library name,
			// to demonstrate that we're finding the right library based on
			// existing API paths.
			apiPath: "google/firestore/queries/v1beta2",
			cfg: &config.Config{
				Language: config.LanguagePython,
				Libraries: []*config.Library{
					{
						Name:    "google-cloud-firestore",
						Version: "1.2.3",
						APIs: []*config.API{
							{Path: "google/firestore/admin/v2"},
							{Path: "google/firestore/v1"},
						},
						Python: &config.PythonPackage{
							DefaultVersion: "v1",
						},
					},
				},
			},
			wantName: "google-cloud-firestore",
			wantCfg: &config.Config{
				Language: config.LanguagePython,
				Libraries: []*config.Library{
					{
						Name:    "google-cloud-firestore",
						Version: "1.2.3",
						APIs: []*config.API{
							{Path: "google/firestore/admin/v2"},
							{Path: "google/firestore/v1"},
							{Path: "google/firestore/queries/v1beta2"},
						},
						Python: &config.PythonPackage{
							DefaultVersion: "v1",
						},
					},
				},
			},
		},
		{
			name: "update existing library (nodejs)",
			// The API path here deliberately doesn't match the library name,
			// to demonstrate that we're finding the right library based on
			// existing API paths.
			apiPath: "google/firestore/v2",
			cfg: &config.Config{
				Language: config.LanguageNodejs,
				Libraries: []*config.Library{
					{
						Name:    "google-cloud-firestore",
						Version: "1.2.3",
						APIs: []*config.API{
							{Path: "google/firestore/v1"},
						},
					},
				},
			},
			wantName: "google-cloud-firestore",
			wantCfg: &config.Config{
				Language: config.LanguageNodejs,
				Libraries: []*config.Library{
					{
						Name:    "google-cloud-firestore",
						Version: "1.2.3",
						APIs: []*config.API{
							{Path: "google/firestore/v1"},
							{Path: "google/firestore/v2"},
						},
					},
				},
			},
		},
		{
			name:    "update existing library (java)",
			apiPath: "google/cloud/secretmanager/v1beta2",
			cfg: &config.Config{
				Language: config.LanguageJava,
				Libraries: []*config.Library{
					{
						Name:    "secretmanager",
						Version: "1.2.3",
						APIs: []*config.API{
							{Path: "google/cloud/secretmanager/v1"},
						},
					},
				},
			},
			wantName: "secretmanager",
			wantCfg: &config.Config{
				Language: config.LanguageJava,
				Libraries: []*config.Library{
					{
						Name:    "secretmanager",
						Version: "1.2.3",
						APIs: []*config.API{
							{Path: "google/cloud/secretmanager/v1"},
							{Path: "google/cloud/secretmanager/v1beta2"},
						},
						Java: &config.JavaModule{ReleasedVersion: "1.2.3"},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)
			if err := os.WriteFile(filepath.Join(tmpDir, "versions.txt"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := yaml.Write(config.LibrarianYAML, test.cfg); err != nil {
				t.Fatal(err)
			}
			gotName, gotCfg, err := addLibrary(test.cfg, test.apiPath, "")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantName, gotName); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantCfg, gotCfg); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAddLibrary_ExistingLibrary_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		apiPath string
		cfg     *config.Config
		wantErr error
	}{
		{
			name:    "fail if api already exists",
			apiPath: "google/cloud/secretmanager/v1beta2",
			cfg: &config.Config{
				Language: config.LanguageGo,
				Libraries: []*config.Library{
					{
						Name:    "secretmanager",
						Version: "1.2.3",
						APIs: []*config.API{
							{Path: "google/cloud/secretmanager/v1"},
							{Path: "google/cloud/secretmanager/v1beta2"},
						},
					},
				},
			},
			wantErr: errAPIAlreadyExists,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)
			if err := os.WriteFile(filepath.Join(tmpDir, "versions.txt"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := yaml.Write(config.LibrarianYAML, test.cfg); err != nil {
				t.Fatal(err)
			}
			_, _, err := addLibrary(test.cfg, test.apiPath, "")
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestAddLibrary_Preview(t *testing.T) {
	for _, test := range []struct {
		name             string
		apiPath          string
		initialLibraries []*config.Library
		wantPreview      *config.Library
	}{
		{
			name:    "add preview to existing library",
			apiPath: "preview/google/cloud/secretmanager/v1",
			initialLibraries: []*config.Library{
				{
					Name:    "secretmanager",
					Version: "1.0.0",
					APIs:    []*config.API{{Path: "google/cloud/secretmanager/v1"}},
				},
			},
			wantPreview: &config.Library{
				APIs:    []*config.API{{Path: "google/cloud/secretmanager/v1"}},
				Version: "1.1.0-preview.1",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				Language:  config.LanguageGo,
				Libraries: test.initialLibraries,
			}
			gotName, gotCfg, err := addLibrary(cfg, test.apiPath, "")
			if err != nil {
				t.Fatal(err)
			}

			got, err := FindLibrary(gotCfg, gotName)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantPreview, got.Preview); diff != "" {
				t.Errorf("preview mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAddLibrary_Preview_Error(t *testing.T) {
	for _, test := range []struct {
		name             string
		apiPath          string
		initialLibraries []*config.Library
		wantErr          error
	}{
		{
			name:    "fail if library doesn't exist",
			apiPath: "preview/google/cloud/secretmanager/v1",
			initialLibraries: []*config.Library{
				{
					Name: "otherlib",
					APIs: []*config.API{{Path: "google/cloud/other/v1"}},
				},
			},
			wantErr: errPreviewRequiresLibrary,
		},
		{
			name:    "fail preview already exists",
			apiPath: "preview/google/cloud/secretmanager/v1",
			initialLibraries: []*config.Library{
				{
					Name: "secretmanager",
					APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
					Preview: &config.Library{
						APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
					},
				},
			},
			wantErr: errPreviewAlreadyExists,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				Language:  config.LanguageGo,
				Libraries: test.initialLibraries,
			}
			_, _, err := addLibrary(cfg, test.apiPath, "")
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestDeriveLibraryName(t *testing.T) {
	for _, test := range []struct {
		language string
		apiPath  string
		want     string
	}{
		{config.LanguageDart, "google/cloud/secretmanager/v1", "google_cloud_secretmanager_v1"},
		{config.LanguagePython, "google/cloud/secretmanager/v1", "google-cloud-secretmanager"},
		{config.LanguagePython, "google/cloud/secretmanager/v1beta2", "google-cloud-secretmanager"},
		{config.LanguagePython, "google/cloud/storage/v2alpha", "google-cloud-storage"},
		{config.LanguagePython, "google/maps/addressvalidation/v1", "google-maps-addressvalidation"},
		{config.LanguagePython, "google/api/v1", "google-api"},
		{config.LanguageRust, "google/cloud/secretmanager/v1", "google-cloud-secretmanager-v1"},
		{config.LanguageRust, "google/cloud/secretmanager/v1beta2", "google-cloud-secretmanager-v1beta2"},
		{config.LanguageFake, "google/cloud/secretmanager/v1", "google-cloud-secretmanager-v1"},
		{config.LanguageGo, "google/cloud/secretmanager/v1", "secretmanager"},
		{config.LanguageJava, "google/cloud/secretmanager/v1", "secretmanager"},
		{config.LanguageJava, "google/api/serviceusage/v1", "serviceusage"},
		{config.LanguageJava, "google/devtools/cloudbuild/v1", "cloudbuild"},
		{config.LanguageJava, "google/pubsub/v1", "pubsub"},
		{config.LanguageJava, "other/api/v1", "other-api"},
		{config.LanguageJava, "google/cloud/datacatalog/lineage/v1", "datacatalog-lineage"},
		{config.LanguageNodejs, "google/cloud/secretmanager/v1", "google-cloud-secretmanager"},
		{config.LanguageNodejs, "google/cloud/secretmanager/v1beta2", "google-cloud-secretmanager"},
		{config.LanguageNodejs, "google/cloud/storage/v2alpha", "google-cloud-storage"},
		{config.LanguageNodejs, "google/maps/addressvalidation/v1", "google-maps-addressvalidation"},
		{config.LanguagePhp, "google/cloud/secretmanager/v1", "secretmanager"},
		{config.LanguagePhp, "google/cloud/security/privateca/v1", "security-privateca"},
		{config.LanguagePhp, "google/pubsub/v1", "pubsub"},
		{config.LanguageRuby, "google/cloud/secretmanager/v1", "google-cloud-secretmanager-v1"},
	} {
		t.Run(test.language+"/"+test.apiPath, func(t *testing.T) {
			got := deriveLibraryName(test.language, test.apiPath)
			if got != test.want {
				t.Errorf("deriveLibraryName(%q, %q) = %q, want %q", test.language, test.apiPath, got, test.want)
			}
		})
	}
}

func TestAddLibraryCommand_Java(t *testing.T) {
	googleapisDir, err := filepath.Abs("../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	if err := os.WriteFile(filepath.Join(tmpDir, "versions.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := sample.Config()
	cfg.Language = config.LanguageJava
	cfg.Default.Output = "output"
	cfg.Libraries = []*config.Library{{Name: "google-cloud-java", Version: "1.0.0"}, {Name: "google-cloud-pom-parent", Version: "1.0.0"}}
	cfg.Sources.Googleapis.Dir = googleapisDir
	if err := yaml.Write(config.LibrarianYAML, cfg); err != nil {
		t.Fatal(err)
	}
	// developerconnect has Locations mixin in its service.yaml
	err = runAdd(t.Context(), cfg, "google/cloud/developerconnect/v1", "")
	if err != nil {
		t.Fatal(err)
	}
	gotCfg, err := yaml.Read[config.Config](config.LibrarianYAML)
	if err != nil {
		t.Fatal(err)
	}
	wantLibraries := []*config.Library{
		{
			Name:          "developerconnect",
			CopyrightYear: "",
			Version:       "0.1.0-SNAPSHOT",
			APIs: []*config.API{
				{
					Path: "google/cloud/developerconnect/v1",
					Java: &config.JavaAPI{
						AdditionalProtos: []*config.AdditionalProto{
							{
								Path: "google/cloud/location/locations.proto",
							},
						},
					},
				},
			},
		},
		{Name: "google-cloud-java", Version: "1.0.0"},
		{Name: "google-cloud-pom-parent", Version: "1.0.0"},
	}
	if diff := cmp.Diff(wantLibraries, gotCfg.Libraries); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAddLibraryCommand_Php(t *testing.T) {
	googleapisDir, err := filepath.Abs("../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	cfg := sample.Config()
	cfg.Language = config.LanguagePhp
	commonResources := true
	cfg.Default.PHP = &config.PHPDefault{
		CommonResources: &commonResources,
	}
	cfg.Default.Output = "output"
	cfg.Libraries = []*config.Library{}
	cfg.Sources.Googleapis.Dir = googleapisDir
	if err := yaml.Write(config.LibrarianYAML, cfg); err != nil {
		t.Fatal(err)
	}
	// developerconnect has Locations mixin in its service.yaml
	err = runAdd(t.Context(), cfg, "google/cloud/developerconnect/v1", "")
	if err != nil {
		t.Fatal(err)
	}
	gotCfg, err := yaml.Read[config.Config](config.LibrarianYAML)
	if err != nil {
		t.Fatal(err)
	}
	wantLibraries := []*config.Library{
		{
			Name:          "developerconnect",
			Version:       "0.0.0",
			CopyrightYear: strconv.Itoa(time.Now().Year()),
			APIs: []*config.API{
				{
					Path: "google/cloud/developerconnect/v1",
					PHP: &config.PHPAPI{
						StagingSubdir: "v1",
						AdditionalProtos: []string{
							"google/cloud/location/locations.proto",
						},
					},
				},
			},
		},
	}
	if diff := cmp.Diff(wantLibraries, gotCfg.Libraries); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAddLibrary_Swift(t *testing.T) {
	copyrightYear := strconv.Itoa(time.Now().Year())
	for _, test := range []struct {
		name               string
		swiftDefault       *config.SwiftDefault
		wantFinalLibraries []*config.Library
	}{
		{
			name:         "new library stable",
			swiftDefault: &config.SwiftDefault{DefaultVersion: "1.0.0"},
			wantFinalLibraries: []*config.Library{
				{
					Name:          "google-cloud-secretmanager-v1",
					CopyrightYear: copyrightYear,
					Version:       "1.0.0",
				},
			},
		},
		{
			name:         "new library preview",
			swiftDefault: &config.SwiftDefault{DefaultVersion: "0.1.2-preview"},
			wantFinalLibraries: []*config.Library{
				{
					Name:          "google-cloud-secretmanager-v1",
					CopyrightYear: copyrightYear,
					Version:       "0.1.2-preview",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			googleapisDir, err := filepath.Abs("../testdata/googleapis")
			if err != nil {
				t.Fatal(err)
			}
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)

			cfg := &config.Config{
				Language: config.LanguageSwift,
				Default: &config.Default{
					Swift:  test.swiftDefault,
					Output: "output",
				},
				Libraries: []*config.Library{},
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Dir: googleapisDir,
					},
				},
			}
			if err := yaml.Write(config.LibrarianYAML, cfg); err != nil {
				t.Fatal(err)
			}
			err = runAdd(t.Context(), cfg, "google/cloud/secretmanager/v1", "")
			if err != nil {
				t.Fatal(err)
			}

			gotCfg, err := yaml.Read[config.Config](config.LibrarianYAML)
			if err != nil {
				t.Fatal(err)
			}

			less := func(a, b *config.Library) bool { return a.Name < b.Name }
			if diff := cmp.Diff(test.wantFinalLibraries, gotCfg.Libraries, cmpopts.SortSlices(less), cmpopts.IgnoreFields(config.Library{}, "APIs")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAddLibraryCommand_Ruby(t *testing.T) {
	googleapisDir, err := filepath.Abs("../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name               string
		initialLibraries   []*config.Library
		initialManifest    map[string]string
		initialPackages    map[string]any
		args               []string
		wantFinalLibraries []*config.Library
		wantManifest       map[string]string
		wantPackages       map[string]any
	}{
		{
			name: "explicit library name for versioned gem with release please",
			args: []string{"google/cloud/secretmanager/v1", "--name", "google-cloud-secret_manager-v1"},
			wantFinalLibraries: []*config.Library{
				{
					Name:          "google-cloud-secret_manager-v1",
					CopyrightYear: strconv.Itoa(time.Now().Year()),
					Version:       "0.0.1",
					APIs: []*config.API{
						{Path: "google/cloud/secretmanager/v1"},
					},
				},
			},
			wantManifest: map[string]string{
				"google-cloud-secret_manager-v1":        "0.0.1",
				"google-cloud-secret_manager-v1+FILLER": "0.0.0",
			},
			wantPackages: map[string]any{
				"google-cloud-secret_manager-v1": map[string]any{
					"component":    "google-cloud-secret_manager-v1",
					"version_file": "lib/google/cloud/secret_manager/v1/version.rb",
				},
			},
		},
		{
			name: "explicit library name for wrapper gem with release please",
			initialLibraries: []*config.Library{
				{
					Name:          "google-cloud-secret_manager-v1",
					CopyrightYear: strconv.Itoa(time.Now().Year()),
					Version:       "0.0.1",
					APIs: []*config.API{
						{Path: "google/cloud/secretmanager/v1"},
					},
				},
			},
			initialManifest: map[string]string{
				"google-cloud-secret_manager-v1":        "0.0.1",
				"google-cloud-secret_manager-v1+FILLER": "0.0.0",
			},
			initialPackages: map[string]any{
				"google-cloud-secret_manager-v1": map[string]any{
					"component":    "google-cloud-secret_manager-v1",
					"version_file": "lib/google/cloud/secret_manager/v1/version.rb",
				},
			},
			args: []string{"google/cloud/secretmanager", "--name", "google-cloud-secret_manager"},
			wantFinalLibraries: []*config.Library{
				{
					Name:          "google-cloud-secret_manager",
					CopyrightYear: strconv.Itoa(time.Now().Year()),
					Version:       "0.0.1",
					APIs: []*config.API{
						{Path: "google/cloud/secretmanager/v1"},
					},
					Ruby: &config.RubyPackage{
						WrapperOf: []string{"v1:0.0"},
					},
				},
				{
					Name:          "google-cloud-secret_manager-v1",
					CopyrightYear: strconv.Itoa(time.Now().Year()),
					Version:       "0.0.1",
					APIs: []*config.API{
						{Path: "google/cloud/secretmanager/v1"},
					},
				},
			},
			wantManifest: map[string]string{
				"google-cloud-secret_manager":           "0.0.1",
				"google-cloud-secret_manager+FILLER":    "0.0.0",
				"google-cloud-secret_manager-v1":        "0.0.1",
				"google-cloud-secret_manager-v1+FILLER": "0.0.0",
			},
			wantPackages: map[string]any{
				"google-cloud-secret_manager": map[string]any{
					"component":    "google-cloud-secret_manager",
					"version_file": "lib/google/cloud/secret_manager/version.rb",
				},
				"google-cloud-secret_manager-v1": map[string]any{
					"component":    "google-cloud-secret_manager-v1",
					"version_file": "lib/google/cloud/secret_manager/v1/version.rb",
				},
			},
		},
		{
			name: "default library name without flag",
			args: []string{"google/cloud/secretmanager/v1"},
			wantFinalLibraries: []*config.Library{
				{
					Name:          "google-cloud-secretmanager-v1",
					CopyrightYear: strconv.Itoa(time.Now().Year()),
					Version:       "0.0.1",
					APIs: []*config.API{
						{Path: "google/cloud/secretmanager/v1"},
					},
				},
			},
			wantManifest: map[string]string{
				"google-cloud-secretmanager-v1":        "0.0.1",
				"google-cloud-secretmanager-v1+FILLER": "0.0.0",
			},
			wantPackages: map[string]any{
				"google-cloud-secretmanager-v1": map[string]any{
					"component":    "google-cloud-secretmanager-v1",
					"version_file": "lib/google/cloud/secretmanager/v1/version.rb",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)
			manifest := test.initialManifest
			if manifest == nil {
				manifest = map[string]string{}
			}
			manifestBytes, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(tmpDir, ".release-please-manifest.json"), manifestBytes, 0o644); err != nil {
				t.Fatal(err)
			}
			packages := test.initialPackages
			if packages == nil {
				packages = map[string]any{}
			}
			rpConfigBytes, err := json.Marshal(map[string]any{"packages": packages})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(tmpDir, "release-please-config.json"), rpConfigBytes, 0o644); err != nil {
				t.Fatal(err)
			}
			cfg := sample.Config()
			cfg.Language = config.LanguageRuby
			cfg.Default.Output = "output"
			cfg.Libraries = test.initialLibraries
			cfg.Sources.Googleapis.Dir = googleapisDir
			if err := yaml.Write(config.LibrarianYAML, cfg); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"librarian", "add"}, test.args...)
			if err := Run(t.Context(), args...); err != nil {
				t.Fatal(err)
			}
			gotCfg, err := yaml.Read[config.Config](config.LibrarianYAML)
			if err != nil {
				t.Fatal(err)
			}
			sort.Slice(gotCfg.Libraries, func(i, j int) bool {
				return gotCfg.Libraries[i].Name < gotCfg.Libraries[j].Name
			})
			if diff := cmp.Diff(test.wantFinalLibraries, gotCfg.Libraries); diff != "" {
				t.Errorf("libraries mismatch (-want +got):\n%s", diff)
			}
			gotManifest, err := readJSONFile[map[string]string](filepath.Join(tmpDir, ".release-please-manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantManifest, gotManifest); diff != "" {
				t.Errorf("manifest mismatch (-want +got):\n%s", diff)
			}
			gotRPConfig, err := readJSONFile[map[string]any](filepath.Join(tmpDir, "release-please-config.json"))
			if err != nil {
				t.Fatal(err)
			}
			wantRPConfig := map[string]any{"packages": test.wantPackages}
			if diff := cmp.Diff(wantRPConfig, gotRPConfig); diff != "" {
				t.Errorf("release please config mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAddLibraryCommand_Ruby_Error(t *testing.T) {
	googleapisDir, err := filepath.Abs("../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name             string
		language         string
		initialLibraries []*config.Library
		args             []string
		wantErr          error
	}{
		{
			name:     "name flag specified for non-ruby language",
			language: config.LanguageGo,
			args:     []string{"google/cloud/secretmanager/v1", "--name", "custom-name"},
			wantErr:  errNameOnlyForRuby,
		},
		{
			name:     "duplicate API path",
			language: config.LanguageRuby,
			initialLibraries: []*config.Library{
				{
					Name: "google-cloud-secret_manager-v1",
					APIs: []*config.API{
						{Path: "google/cloud/secretmanager/v1"},
					},
				},
			},
			args:    []string{"google/cloud/secretmanager/v1", "--name", "google-cloud-secret_manager-v1"},
			wantErr: errAPIAlreadyExists,
		},
		{
			name:     "duplicate library name with different API",
			language: config.LanguageRuby,
			initialLibraries: []*config.Library{
				{
					Name: "google-cloud-secret_manager-v1",
					APIs: []*config.API{
						{Path: "google/cloud/secretmanager/v2"},
					},
				},
			},
			args:    []string{"google/cloud/secretmanager/v1", "--name", "google-cloud-secret_manager-v1"},
			wantErr: errLibraryAlreadyExists,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)
			cfg := sample.Config()
			cfg.Language = test.language
			cfg.Default.Output = "output"
			cfg.Libraries = test.initialLibraries
			cfg.Sources.Googleapis.Dir = googleapisDir
			if err := yaml.Write(config.LibrarianYAML, cfg); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"librarian", "add"}, test.args...)
			err := Run(t.Context(), args...)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestAddLibraryCommand_GoogleapisCache(t *testing.T) {
	googleapisDir, err := filepath.Abs("../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	cacheDir := t.TempDir()
	t.Setenv(cache.EnvLibrarianCache, cacheDir)

	const commit = "abcdef123456"
	cachedExtracted := filepath.Join(cacheDir, fmt.Sprintf("%s@%s", googleapisRepo, commit))
	if err := os.CopyFS(cachedExtracted, os.DirFS(googleapisDir)); err != nil {
		t.Fatal(err)
	}

	cfg := sample.Config()
	cfg.Default.Output = "output"
	cfg.Libraries = nil
	cfg.Sources = &config.Sources{
		Googleapis: &config.Source{
			Commit: commit,
			SHA256: "fake-sha",
		},
	}
	if err := yaml.Write(config.LibrarianYAML, cfg); err != nil {
		t.Fatal(err)
	}
	err = runAdd(t.Context(), cfg, "google/cloud/secretmanager/v1", "")
	if err != nil {
		t.Fatalf("expected runAdd to succeed with cached googleapis, got %v", err)
	}
}
