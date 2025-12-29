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

func TestCreateLibrary(t *testing.T) {
	for _, test := range []struct {
		name            string
		libName         string
		output          string
		existingLibrary *config.Library
		wantOutput      string
	}{
		{
			name:       "create new library",
			libName:    "newlib",
			output:     "newlib-output",
			wantOutput: "newlib-output",
		},
		{
			name:    "regenerate existing library",
			libName: "existinglib",
			existingLibrary: &config.Library{
				Name:   "existinglib",
				Output: "existing-output",
			},
			wantOutput: "existing-output",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)

			// Create service config file for the library
			serviceConfigDir := filepath.Join(tmpDir, test.libName)
			if err := os.MkdirAll(serviceConfigDir, 0755); err != nil {
				t.Fatal(err)
			}
			serviceConfigPath := filepath.Join(serviceConfigDir, test.libName+".yaml")
			serviceConfigContent := "type: google.api.Service\nconfig_version: 3\n"
			if err := os.WriteFile(serviceConfigPath, []byte(serviceConfigContent), 0644); err != nil {
				t.Fatal(err)
			}

			cfg := &config.Config{
				Language: languageFake,
				Default: &config.Default{
					Output: "output",
				},
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Dir: tmpDir,
					},
				},
			}
			if test.existingLibrary != nil {
				cfg.Libraries = []*config.Library{test.existingLibrary}
			}
			if err := yaml.Write(librarianConfigPath, cfg); err != nil {
				t.Fatal(err)
			}

			if err := runCreate(t.Context(), test.libName, "", "", test.output, ""); err != nil {
				t.Fatal(err)
			}

			cfg, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				t.Fatal(err)
			}

			var found *config.Library
			for _, lib := range cfg.Libraries {
				if lib.Name == test.libName {
					found = lib
					break
				}
			}
			if found == nil {
				t.Fatal("library not found in config")
			}

			if found.Output != test.wantOutput {
				t.Errorf("output = %q, want %q", found.Output, test.wantOutput)
			}

			readmePath := filepath.Join(test.wantOutput, "README.md")
			if _, err := os.Stat(readmePath); err != nil {
				t.Errorf("expected README.md at %s: %v", readmePath, err)
			}

			versionPath := filepath.Join(test.wantOutput, "VERSION")
			content, err := os.ReadFile(versionPath)
			if err != nil {
				t.Fatal(err)
			}
			want := "0.0.0"
			if diff := cmp.Diff(want, string(content)); diff != "" {
				t.Errorf("VERSION mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateLibraryNoYaml(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	err := runCreate(t.Context(), "newlib", "", "", "output/newlib", "protobuf")
	if !errors.Is(err, errNoYaml) {
		t.Errorf("want error %v, got %v", errNoYaml, err)
	}
}

func TestCreateCommand(t *testing.T) {
	for _, test := range []struct {
		name    string
		args    []string
		wantErr error
	}{
		{
			name:    "no args",
			args:    []string{"librarian", "create"},
			wantErr: errMissingLibraryName,
		},
	} {
		t.Run(test.name, func(t *testing.T) {

			err := Run(t.Context(), test.args...)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("want error %v, got %v", test.wantErr, err)
				}
				return
			}
		})
	}
}

func TestDeriveSpecificationSource(t *testing.T) {
	for _, test := range []struct {
		name               string
		serviceConfig      string
		specSource         string
		expectedSpecSource string
		language           string
	}{
		{
			name:               "rust missing service-config",
			language:           "rust",
			specSource:         "google/cloud/storage/v1",
			expectedSpecSource: "google/cloud/storage/v1",
		},
		{
			name:               "rust missing specification-source",
			language:           "rust",
			serviceConfig:      "google/cloud/storage/v1/storage_v1.yaml",
			expectedSpecSource: "google/cloud/storage/v1",
		},
		{
			name:               "rust missing specification-source and service-config",
			language:           "rust",
			expectedSpecSource: "",
		},
		{
			name:               "non-rust language",
			language:           "other-lang",
			expectedSpecSource: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := deriveSpecSource(test.specSource, test.serviceConfig, test.language)
			if got != test.expectedSpecSource {
				t.Errorf("want specification source %q, got %q", test.expectedSpecSource, got)
			}
		})
	}
}

func TestDeriveOutput(t *testing.T) {
	for _, test := range []struct {
		name           string
		specSource     string
		output         string
		defaultOutput  string
		expectedOutput string
		libraryName    string
		language       string
		wantErr        error
	}{

		{
			name:           "default rust output directory used with spec source",
			language:       "rust",
			specSource:     "google/cloud/storage/v1",
			defaultOutput:  "default",
			expectedOutput: "default/cloud/storage/v1",
		},
		{
			name:           "default rust output directory used with default package",
			language:       "rust",
			defaultOutput:  "default",
			libraryName:    "google-cloud-storage-v1",
			expectedOutput: "default/cloud/storage/v1",
		},
		{
			name:           "rust override output directory",
			language:       "rust",
			output:         "override",
			expectedOutput: "override",
		},
		{
			name:        "rust no default output directory",
			language:    "rust",
			specSource:  "google/cloud/storage/v1",
			libraryName: "google-cloud-storage-v1",
			wantErr:     errOutputFlagRequired,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				Language: test.language,
			}
			if test.defaultOutput != "" {
				cfg.Default = &config.Default{Output: test.defaultOutput}
			}
			got, err := deriveOutput(test.output, cfg, test.libraryName, test.specSource, test.language)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("want error %v, got %v", test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.expectedOutput {
				t.Errorf("want output %q, got %q", test.expectedOutput, got)
			}
		})
	}
}

func TestAddLibraryToLibrarianYaml(t *testing.T) {
	for _, test := range []struct {
		name          string
		libraryName   string
		output        string
		specSource    string
		serviceConfig string
		specFormat    string
		want          []*config.Channel
	}{
		{
			name:        "library with no specification-source and service-config",
			libraryName: "newlib",
			output:      "output/newlib",
			specFormat:  "protobuf",
		},
		{
			name:          "library with specification-source and service-config",
			libraryName:   "newlib",
			output:        "output/newlib",
			specFormat:    "protobuf",
			specSource:    "google/cloud/storage/v1",
			serviceConfig: "google/cloud/storage/v1/storage_v1.yaml",
			want: []*config.Channel{
				{
					Path:          "google/cloud/storage/v1",
					ServiceConfig: "google/cloud/storage/v1/storage_v1.yaml",
				},
			},
		},
		{
			name:        "library with specification-source",
			libraryName: "newlib",
			output:      "output/newlib",
			specFormat:  "protobuf",
			specSource:  "google/cloud/storage/v1",
			want: []*config.Channel{
				{
					Path: "google/cloud/storage/v1",
				},
			},
		},
		{
			name:          "library with service-config",
			libraryName:   "newlib",
			output:        "output/newlib",
			specFormat:    "protobuf",
			serviceConfig: "google/cloud/storage/v1/storage_v1.yaml",
			want: []*config.Channel{
				{
					ServiceConfig: "google/cloud/storage/v1/storage_v1.yaml",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)

			cfg := &config.Config{
				Language: languageFake,
				Libraries: []*config.Library{
					{
						Name:   "existinglib",
						Output: "output/existinglib",
					},
				},
			}
			if err := yaml.Write(librarianConfigPath, cfg); err != nil {
				t.Fatal(err)
			}

			if err := addLibraryToLibrarianConfig(cfg, test.libraryName, test.output, test.specSource, test.serviceConfig, test.specFormat); err != nil {
				t.Fatal(err)
			}

			cfg, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				t.Fatal(err)
			}

			if len(cfg.Libraries) != 2 {
				t.Errorf("libraries count = %d, want 2", len(cfg.Libraries))
			}

			var found *config.Library
			for _, lib := range cfg.Libraries {
				if lib.Name == test.libraryName {
					found = lib
					break
				}
			}
			if found == nil {
				t.Fatalf("library %q not found in config", test.libraryName)
			}

			if found.Output != test.output {
				t.Errorf("output = %q, want %q", found.Output, test.output)
			}
			if found.SpecificationFormat != test.specFormat {
				t.Errorf("specification format = %q, want %q", found.SpecificationFormat, test.specFormat)
			}
			if found.Version != "0.1.0" {
				t.Errorf("version = %q, want %q", found.Version, "0.1.0")
			}
			if diff := cmp.Diff(test.want, found.Channels); diff != "" {
				t.Errorf("channels mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
