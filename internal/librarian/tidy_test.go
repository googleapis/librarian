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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestValidateLibraries(t *testing.T) {
	for _, test := range []struct {
		name      string
		libraries []*config.Library
		wantErr   error
	}{
		{
			name: "valid libraries",
			libraries: []*config.Library{
				{Name: "google-cloud-secretmanager-v1"},
				{Name: "google-cloud-storage-v1"},
			},
		},
		{
			name: "duplicate library names",
			libraries: []*config.Library{
				{Name: "google-cloud-secretmanager-v1"},
				{Name: "google-cloud-secretmanager-v1"},
			},
			wantErr: errDuplicateLibraryName,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{Libraries: test.libraries}
			err := validateLibraries(cfg)
			if test.wantErr == nil {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected %v, got nil", test.wantErr)
			}
			if !errors.Is(err, test.wantErr) {
				t.Errorf("expected %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestFormatConfig(t *testing.T) {
	cfg := formatConfig(&config.Config{
		Default: &config.Default{
			Rust: &config.RustDefault{
				PackageDependencies: []*config.RustPackageDependency{
					{Name: "z"},
					{Name: "a"},
				},
			},
		},
		Libraries: []*config.Library{
			{
				Name:    "google-cloud-storage-v1",
				Version: "1.0.0",
				APIs: []*config.API{
					{Path: "c"},
					{Path: "a"},
				},
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
							{Name: "y"},
							{Name: "b"},
						},
					},
				},
			},
			{Name: "google-cloud-bigquery-v1", Version: "2.0.0"},
			{Name: "google-cloud-secretmanager-v1", Version: "3.0.0"},
		},
	})
	t.Run("sorts libraries by name", func(t *testing.T) {
		want := []string{
			"google-cloud-bigquery-v1",
			"google-cloud-secretmanager-v1",
			"google-cloud-storage-v1",
		}
		var got []string
		for _, lib := range cfg.Libraries {
			got = append(got, lib.Name)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	var storageLib *config.Library
	for _, lib := range cfg.Libraries {
		if lib.Name == "google-cloud-storage-v1" {
			storageLib = lib
			break
		}
	}
	if storageLib == nil {
		t.Fatal("library google-cloud-storage-v1 not found after sorting")
	}

	t.Run("sorts apis by path", func(t *testing.T) {
		want := []string{"a", "c"}
		var got []string
		for _, ch := range storageLib.APIs {
			got = append(got, ch.Path)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("sorts default rust dependencies by name", func(t *testing.T) {
		want := []string{"a", "z"}
		var got []string
		for _, dep := range cfg.Default.Rust.PackageDependencies {
			got = append(got, dep.Name)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("sorts library rust dependencies by name", func(t *testing.T) {
		want := []string{"b", "y"}
		var got []string
		for _, dep := range storageLib.Rust.PackageDependencies {
			got = append(got, dep.Name)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestTidyCommand(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	configPath := filepath.Join(tempDir, librarianConfigPath)
	configContent := fmt.Sprintf(`language: rust
version: %s
sources:
  googleapis:
    commit: 94ccedca05acb0bb60780789e93371c9e4100ddc
    sha256: fff40946e897d96bbdccd566cb993048a87029b7e08eacee3fe99eac792721ba
libraries:
  - name: google-cloud-storage-v1
    version: "1.0.0"
  - name: google-cloud-bigquery-v1
    version: "2.0.0"
`, sample.LibrarianVersion)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Run(t.Context(), "librarian", "tidy"); err != nil {
		t.Fatal(err)
	}

	cfg, err := yaml.Read[config.Config](configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Version != sample.LibrarianVersion {
		t.Errorf("version = %q, want %q", cfg.Version, sample.LibrarianVersion)
	}

	var got []string
	for _, lib := range cfg.Libraries {
		got = append(got, lib.Name)
	}
	want := []string{
		"google-cloud-bigquery-v1",
		"google-cloud-storage-v1",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestTidy_DerivableFields(t *testing.T) {
	googleapisSource := &config.Sources{
		Googleapis: &config.Source{
			Commit: "94ccedca05acb0bb60780789e93371c9e4100ddc",
			SHA256: "fff40946e897d96bbdccd566cb993048a87029b7e08eacee3fe99eac792721ba",
		},
	}
	for _, test := range []struct {
		name                    string
		config                  *config.Config
		wantPath                string
		wantNumLibs             int
		wantNumChnls            int
		wantSpecificationFormat string
	}{
		{
			name: "derivable fields removed",
			config: &config.Config{
				Sources: googleapisSource,
				Libraries: []*config.Library{
					{
						Name:                "google-cloud-accessapproval-v1",
						SpecificationFormat: "protobuf",
						APIs: []*config.API{
							{
								Path: "google/cloud/accessapproval/v1",
							},
						},
					},
				},
			},
			wantPath:                "",
			wantNumLibs:             1,
			wantNumChnls:            0,
			wantSpecificationFormat: "",
		},
		{
			name: "non-derivable path not removed",
			config: &config.Config{
				Sources: googleapisSource,
				Libraries: []*config.Library{
					{
						Name: "google-cloud-aiplatform-v1-schema-predict-instance",
						APIs: []*config.API{
							{
								Path: "src/generated/cloud/aiplatform/schema/predict/instance",
							},
						},
					},
				},
			},
			wantPath:     "src/generated/cloud/aiplatform/schema/predict/instance",
			wantNumLibs:  1,
			wantNumChnls: 1,
		},
		{
			name: "api removed if only derivable path",
			config: &config.Config{
				Sources: googleapisSource,
				Libraries: []*config.Library{
					{
						Name: "google-cloud-orgpolicy-v1",
						APIs: []*config.API{
							{
								Path: "google/cloud/orgpolicy/v1",
							},
						},
					},
				},
			},
			wantPath:     "",
			wantNumLibs:  1,
			wantNumChnls: 0,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)

			RunTidyOnConfig(t.Context(), test.config)

			cfg, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				t.Fatal(err)
			}

			if len(cfg.Libraries) != test.wantNumLibs {
				t.Fatalf("wrong number of libraries")
			}
			lib := cfg.Libraries[0]
			if len(lib.APIs) != test.wantNumChnls {
				t.Fatalf("wrong number of apis")
			}
			if test.wantNumChnls > 0 {
				ch := lib.APIs[0]
				if ch.Path != test.wantPath {
					t.Errorf("path should be %s, got %q", test.wantPath, ch.Path)
				}
			}
			if lib.SpecificationFormat != test.wantSpecificationFormat {
				t.Errorf("specification_format = %q, want %q", lib.SpecificationFormat, test.wantSpecificationFormat)
			}
		})
	}
}

func TestTidyDuplicateError(t *testing.T) {
	cfg := &config.Config{
		Language: "rust",
		Sources: &config.Sources{
			Googleapis: &config.Source{
				Commit: "94ccedca05acb0bb60780789e93371c9e4100ddc",
				SHA256: "fff40946e897d96bbdccd566cb993048a87029b7e08eacee3fe99eac792721ba",
			},
		},
		Libraries: []*config.Library{
			{
				Name:    "google-cloud-storage-v1",
				Version: "1.0.0",
			},
			{
				Name:    "google-cloud-storage-v1",
				Version: "2.0.0",
			},
		},
	}

	err := RunTidyOnConfig(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for duplicate library")
	}
	if !errors.Is(err, errDuplicateLibraryName) {
		t.Errorf("expected %v, got %v", errDuplicateLibraryName, err)
	}
}

func TestTidy_DerivableOutput(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	cfg := &config.Config{
		Language: "rust",
		Default: &config.Default{
			Output: "generated/",
		},
		Sources: &config.Sources{
			Googleapis: &config.Source{
				Commit: "94ccedca05acb0bb60780789e93371c9e4100ddc",
				SHA256: "fff40946e897d96bbdccd566cb993048a87029b7e08eacee3fe99eac792721ba",
			},
		},
		Libraries: []*config.Library{
			{
				Name:   "google-cloud-secretmanager-v1",
				Output: "generated/cloud/secretmanager/v1",
				Roots:  []string{"googleapis"},
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
					},
				},
			},
		},
	}
	if err := RunTidyOnConfig(t.Context(), cfg); err != nil {
		t.Fatal(err)
	}
	got, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Libraries) != 1 {
		t.Fatalf("expected 1 library, got %d", len(got.Libraries))
	}
	if got.Libraries[0].Output != "" {
		t.Errorf("expected output to be empty, got %q", got.Libraries[0].Output)
	}
}

func TestTidy_DerivableAPIPath(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	cfg := &config.Config{
		Language: "dart",
		Default: &config.Default{
			Output: "generated/",
		},
		Sources: &config.Sources{
			Googleapis: &config.Source{
				Commit: "94ccedca05acb0bb60780789e93371c9e4100ddc",
				SHA256: "fff40946e897d96bbdccd566cb993048a87029b7e08eacee3fe99eac792721ba",
			},
		},
		Libraries: []*config.Library{
			{
				Name:  "google_cloud_secretmanager_v1",
				Roots: []string{"googleapis"},
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
					},
				},
			},
		},
	}
	if err := RunTidyOnConfig(t.Context(), cfg); err != nil {
		t.Fatal(err)
	}
	got, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Libraries) != 1 {
		t.Fatalf("expected 1 library, got %d", len(got.Libraries))
	}
	if got.Libraries[0].APIs != nil {
		t.Fatalf("expected 0 API, got %d", len(got.Libraries[0].APIs))
	}
}

func TestTidy_DerivableRoots(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	cfg := &config.Config{
		Language: "rust",
		Default: &config.Default{
			Output: "generated/",
		},
		Sources: &config.Sources{
			Googleapis: &config.Source{
				Commit: "94ccedca05acb0bb60780789e93371c9e4100ddc",
				SHA256: "fff40946e897d96bbdccd566cb993048a87029b7e08eacee3fe99eac792721ba",
			},
		},
		Libraries: []*config.Library{
			{
				Name:  "google-cloud-secretmanager-v1",
				Roots: []string{"googleapis"},
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
					},
				},
			},
		},
	}
	if err := RunTidyOnConfig(t.Context(), cfg); err != nil {
		t.Fatal(err)
	}
	got, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Libraries) != 1 {
		t.Fatalf("expected 1 library, got %d", len(got.Libraries))
	}
	if got.Libraries[0].Roots != nil {
		t.Errorf("expected roots to be nil, got %q", got.Libraries[0].Roots)
	}
}

func TestTidyLanguageConfig_Rust(t *testing.T) {
	for _, test := range []struct {
		name        string
		cfg         *config.Config
		wantNumLibs int
		wantNumMods int
	}{
		{
			name: "empty_module_removed",
			cfg: &config.Config{
				Language: "rust",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "94ccedca05acb0bb60780789e93371c9e4100ddc",
						SHA256: "fff40946e897d96bbdccd566cb993048a87029b7e08eacee3fe99eac792721ba",
					},
				},
				Default: &config.Default{
					Output: "generated/",
				},
				Libraries: []*config.Library{
					{
						Name:   "google-cloud-storage",
						Output: "src/storage",
						Rust: &config.RustCrate{
							Modules: []*config.RustModule{
								{
									Output:   "src/storage/src/generated/protos/storage",
									Source:   "google/storage/v2",
									Template: "prost",
								},
								{
									Output:   "src/storage/control",
									Source:   "none",
									Template: "",
								},
							},
						},
					},
				},
			},
			wantNumLibs: 1,
			wantNumMods: 1, // Modules should be removed
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)

			RunTidyOnConfig(t.Context(), test.cfg)

			cfg, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				t.Fatal(err)
			}

			if len(cfg.Libraries) != test.wantNumLibs {
				t.Fatalf("wrong number of libraries")
			}
			lib := cfg.Libraries[0]
			if len(lib.Rust.Modules) != test.wantNumMods {
				t.Fatalf("wrong number of modules")
			}
		})
	}
}

func TestTidyMissingGoogleApisSource(t *testing.T) {
	cfg := &config.Config{
		Language: "rust",
		Libraries: []*config.Library{
			{
				Name:    "google-cloud-storage-v1",
				Version: "1.0.0",
			},
			{
				Name:    "google-cloud-bigquery-v1",
				Version: "2.0.0",
			},
		},
	}
	err := RunTidyOnConfig(t.Context(), cfg)
	if err == nil {
		t.Fatalf("expected error, got %v", nil)
	}
	if !errors.Is(err, errNoGoogleapiSourceInfo) {
		t.Errorf("mismatch error want %v got %v", errNoGoogleapiSourceInfo, err)
	}
}

func TestTidy_VeneerSkipGenerate(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	cfg := &config.Config{
		Language: "rust",
		Sources: &config.Sources{
			Googleapis: &config.Source{
				Commit: "94ccedca05acb0bb60780789e93371c9e4100ddc",
				SHA256: "fff40946e897d96bbdccd566cb993048a87029b7e08eacee3fe99eac792721ba",
			},
		},
		Libraries: []*config.Library{
			{
				Name:         "google-cloud-storage",
				Veneer:       true,
				SkipGenerate: true,
				Output:       "src/storage",
			},
		},
	}
	if err := RunTidyOnConfig(t.Context(), cfg); err != nil {
		t.Fatal(err)
	}
	cfg, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Libraries) != 1 {
		t.Fatalf("expected 1 library, got %d", len(cfg.Libraries))
	}
	if cfg.Libraries[0].SkipGenerate {
		t.Errorf("expected skip_generate to be false for veneer library, got true")
	}
}
