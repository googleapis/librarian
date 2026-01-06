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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
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
				Channels: []*config.Channel{
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

	t.Run("sorts channels by path", func(t *testing.T) {
		want := []string{"a", "c"}
		var got []string
		for _, ch := range storageLib.Channels {
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
	configContent := `language: rust
sources:
  googleapis:
    commit: 94ccedca05acb0bb60780789e93371c9e4100ddc
    sha256: fff40946e897d96bbdccd566cb993048a87029b7e08eacee3fe99eac792721ba
libraries:
  - name: google-cloud-storage-v1
    version: "1.0.0"
  - name: google-cloud-bigquery-v1
    version: "2.0.0"
`
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
	for _, test := range []struct {
		name          string
		configContent string
		wantPath      string
		wantSvcCfg    string
		wantNumLibs   int
		wantNumChnls  int
	}{
		{
			name: "derivable fields removed",
			configContent: `
sources:
  googleapis:
    commit: 94ccedca05acb0bb60780789e93371c9e4100ddc
    sha256: fff40946e897d96bbdccd566cb993048a87029b7e08eacee3fe99eac792721ba
libraries:
  - name: google-cloud-accessapproval-v1
    channels:
      - path: google/cloud/accessapproval/v1
        service_config: google/cloud/accessapproval/v1/accessapproval_v1.yaml
`,
			wantPath:     "",
			wantSvcCfg:   "",
			wantNumLibs:  1,
			wantNumChnls: 0, // Channels should be removed
		},
		{
			name: "non-derivable path not removed",
			configContent: `
sources:
  googleapis:
    commit: 94ccedca05acb0bb60780789e93371c9e4100ddc
    sha256: fff40946e897d96bbdccd566cb993048a87029b7e08eacee3fe99eac792721ba
libraries:
  - name: google-cloud-aiplatform-v1-schema-predict-instance
    channels:
      - path: src/generated/cloud/aiplatform/schema/predict/instance
        service_config: google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml
`,
			wantPath:     "src/generated/cloud/aiplatform/schema/predict/instance",
			wantSvcCfg:   "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
			wantNumLibs:  1,
			wantNumChnls: 1,
		},
		{
			name: "path needs to be resolved",
			configContent: `
sources:
  googleapis:
    commit: 94ccedca05acb0bb60780789e93371c9e4100ddc
    sha256: fff40946e897d96bbdccd566cb993048a87029b7e08eacee3fe99eac792721ba
libraries:
  - name: google-cloud-vision-v1
    channels:
      - service_config: google/some/other/domain/vision/v1/vision_v1.yaml
`,
			wantPath:     "",
			wantSvcCfg:   "google/some/other/domain/vision/v1/vision_v1.yaml",
			wantNumLibs:  1,
			wantNumChnls: 1,
		},
		{
			name: "service config not derivable (no version at end of path)",
			configContent: `
sources:
  googleapis:
    commit: 94ccedca05acb0bb60780789e93371c9e4100ddc
    sha256: fff40946e897d96bbdccd566cb993048a87029b7e08eacee3fe99eac792721ba
libraries:
  - name: google-cloud-speech
    channels:
      - path: google/cloud/speech
        service_config: google/cloud/speech/1/speech_1.yaml
`,
			wantPath:     "",
			wantSvcCfg:   "google/cloud/speech/1/speech_1.yaml",
			wantNumLibs:  1,
			wantNumChnls: 1,
		},
		{
			name: "channel removed if service config does not exist",
			configContent: `
sources:
  googleapis:
    commit: 94ccedca05acb0bb60780789e93371c9e4100ddc
    sha256: fff40946e897d96bbdccd566cb993048a87029b7e08eacee3fe99eac792721ba
libraries:
  - name: google-cloud-orgpolicy-v1
    channels:
      - path: google/cloud/orgpolicy/v1
`,
			wantPath:     "",
			wantSvcCfg:   "",
			wantNumLibs:  1,
			wantNumChnls: 0,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)
			configPath := filepath.Join(tempDir, librarianConfigPath)

			if err := os.WriteFile(configPath, []byte(test.configContent), 0644); err != nil {
				t.Fatal(err)
			}
			if err := Run(t.Context(), "librarian", "tidy"); err != nil {
				t.Fatal(err)
			}

			cfg, err := yaml.Read[config.Config](configPath)
			if err != nil {
				t.Fatal(err)
			}

			if len(cfg.Libraries) != test.wantNumLibs {
				t.Fatalf("wrong number of libraries")
			}
			lib := cfg.Libraries[0]
			if len(lib.Channels) != test.wantNumChnls {
				t.Fatalf("wrong number of channels")
			}
			if test.wantNumChnls > 0 {
				ch := lib.Channels[0]
				if ch.Path != test.wantPath {
					t.Errorf("path should be %s, got %q", test.wantPath, ch.Path)
				}
				if ch.ServiceConfig != test.wantSvcCfg {
					t.Errorf("service_config should be %s, got %q", test.wantSvcCfg, ch.ServiceConfig)
				}
			}
		})
	}
}

func TestTidyCommandDuplicateError(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	configPath := filepath.Join(tempDir, librarianConfigPath)
	configContent := `language: rust
sources:
  googleapis:
    commit: abc123
libraries:
  - name: google-cloud-storage-v1
  - name: google-cloud-storage-v1
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	err := Run(t.Context(), "librarian", "tidy")
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
	configPath := filepath.Join(tempDir, librarianConfigPath)
	configContent := `
language: rust
default:
  output: generated/
sources:
  googleapis:
    commit: 94ccedca05acb0bb60780789e93371c9e4100ddc
    sha256: fff40946e897d96bbdccd566cb993048a87029b7e08eacee3fe99eac792721ba
libraries:
  - name: google-cloud-secretmanager-v1
    output: generated/cloud/secretmanager/v1
    channels:
      - path: google/cloud/secretmanager/v1
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := RunTidy(t.Context()); err != nil {
		t.Fatal(err)
	}
	cfg, err := yaml.Read[config.Config](configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Libraries) != 1 {
		t.Fatalf("expected 1 library, got %d", len(cfg.Libraries))
	}
	if cfg.Libraries[0].Output != "" {
		t.Errorf("expected output to be empty, got %q", cfg.Libraries[0].Output)
	}
}

func TestTidyLanguageConfig_Rust(t *testing.T) {
	for _, test := range []struct {
		name          string
		configContent string
		wantNumLibs   int
		wantNumMods   int
	}{
		{
			name: "empty_module_removed",
			configContent: `
language: rust
sources:
  googleapis:
    commit: 94ccedca05acb0bb60780789e93371c9e4100ddc
    sha256: fff40946e897d96bbdccd566cb993048a87029b7e08eacee3fe99eac792721ba
default:
  output: generated/
libraries:
  - name: google-cloud-storage
    output: src/storage
    rust:
      modules:
      - output: src/storage/src/generated/protos/storage
        source: google/storage/v2
        template: prost
      - output: src/storage/control
        source: none
        template: ""`,
			wantNumLibs: 1,
			wantNumMods: 1, // Modules should be removed
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)
			configPath := filepath.Join(tempDir, librarianConfigPath)

			if err := os.WriteFile(configPath, []byte(test.configContent), 0644); err != nil {
				t.Fatal(err)
			}
			if err := Run(t.Context(), "librarian", "tidy"); err != nil {
				t.Fatal(err)
			}

			cfg, err := yaml.Read[config.Config](configPath)
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

func TestTidyCommandMissingGoogleApisSource(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	configPath := filepath.Join(tempDir, librarianConfigPath)
	configContent := `language: rust
libraries:
  - name: google-cloud-storage-v1
    version: "1.0.0"
  - name: google-cloud-bigquery-v1
    version: "2.0.0"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	err := Run(t.Context(), "librarian", "tidy")
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
	configPath := filepath.Join(tempDir, librarianConfigPath)
	configContent := `
language: rust
sources:
  googleapis:
    commit: 94ccedca05acb0bb60780789e93371c9e4100ddc
    sha256: fff40946e897d96bbdccd566cb993048a87029b7e08eacee3fe99eac792721ba
libraries:
  - name: google-cloud-storage
    veneer: true
    skip_generate: true
    output: src/storage
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := RunTidy(t.Context()); err != nil {
		t.Fatal(err)
	}
	cfg, err := yaml.Read[config.Config](configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Libraries) != 1 {
		t.Fatalf("expected 1 library, got %d", len(cfg.Libraries))
	}
	if !cfg.Libraries[0].SkipGenerate {
		t.Errorf("expected skip_generate to be true for veneer library, got false")
	}
}
