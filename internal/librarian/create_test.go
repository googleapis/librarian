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
		wantError       error
	}{
		{
			name:       "create new library",
			libName:    "newlib",
			output:     "newlib-output",
			wantOutput: "newlib-output",
		},
		{
			name:    "fail create existing library",
			libName: "existinglib",
			existingLibrary: &config.Library{
				Name: "existinglib",
			},
			wantError: errLibraryAlreadyExists,
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
			err := runCreate(t.Context(), test.libName, test.output)
			if test.wantError != nil {
				if !errors.Is(err, test.wantError) {
					t.Errorf("expected error %v, got %v", test.wantError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("runCreate() failed with unexpected error: %v", err)
			}

			cfg, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				t.Fatal(err)
			}

			found := findLibrary(cfg, test.libName)
			if found == nil {
				t.Fatal("library not found in config")
			}
			if found.Output != test.wantOutput {
				t.Fatalf("output = %q, want %q", found.Output, test.wantOutput)
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

	err := runCreate(t.Context(), "newlib", "output/newlib")
	if !errors.Is(err, errNoYaml) {
		t.Errorf("want error %v, got %v", errNoYaml, err)
	}
}

func TestCreateCommand(t *testing.T) {
	googleapisDir, err := filepath.Abs("testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}

	testName := "google-cloud-secret-manager"
	for _, test := range []struct {
		name         string
		args         []string
		wantErr      error
		wantChannels []*config.Channel
		wantOutput   string
	}{
		{
			name:    "no args",
			args:    []string{"librarian", "create"},
			wantErr: errMissingLibraryName,
		},
		{
			name: "library name only",
			args: []string{
				"librarian",
				"create",
				"google-cloud-secretmanager-v1",
			},
		},
		{
			name: "library with single API",
			args: []string{
				"librarian",
				"create",
				testName,
				"google/cloud/secretmanager/v1",
			},
			wantChannels: []*config.Channel{
				{
					Path: "google/cloud/secretmanager/v1",
				},
			},
		},
		{
			name: "library with multiple APIs",
			args: []string{
				"librarian",
				"create",
				testName,
				"google/cloud/secretmanager/v1",
				"google/cloud/secretmanager/v1beta2",
				"google/cloud/secrets/v1beta1",
			},
			wantChannels: []*config.Channel{
				{
					Path: "google/cloud/secretmanager/v1",
				},
				{
					Path: "google/cloud/secretmanager/v1beta2",
				},
				{
					Path: "google/cloud/secrets/v1beta1",
				},
			},
		},
		{
			name: "library with multiple APIs and output flag",
			args: []string{
				"librarian",
				"create",
				testName,
				"google/cloud/secretmanager/v1",
				"google/cloud/secretmanager/v1beta2",
				"google/cloud/secrets/v1beta1",
				"--output",
				"packages/google-cloud-secret-manager",
			},
			wantOutput: "packages/google-cloud-secret-manager",
			wantChannels: []*config.Channel{
				{
					Path: "google/cloud/secretmanager/v1",
				},
				{
					Path: "google/cloud/secretmanager/v1beta2",
				},
				{
					Path: "google/cloud/secrets/v1beta1",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)

			cfg := &config.Config{
				Language: languageFake,
				Default: &config.Default{
					Output: "output",
				},
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Dir: googleapisDir,
					},
				},
			}
			if err := yaml.Write(librarianConfigPath, cfg); err != nil {
				t.Fatal(err)
			}
			err := Run(t.Context(), test.args...)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("want error %v, got %v", test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			gotCfg, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				t.Fatal(err)
			}
			got := findLibrary(gotCfg, testName)
			if test.wantOutput != "" && got.Output != test.wantOutput {
				t.Errorf("output = %q, want %q", got.Output, test.wantOutput)
			}
			if test.wantChannels != nil {
				if diff := cmp.Diff(test.wantChannels, got.Channels); diff != "" {
					t.Errorf("channels mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestAddLibraryToLibrarianYaml(t *testing.T) {
	for _, test := range []struct {
		name        string
		libraryName string
		output      string
		channels    []string
		want        []*config.Channel
	}{
		{
			name:        "library with no specification-source",
			libraryName: "newlib",
			output:      "output/newlib",
		},
		{
			name:        "library with single API",
			libraryName: "newlib",
			output:      "output/newlib",
			channels:    []string{"google/cloud/storage/v1"},
			want: []*config.Channel{
				{
					Path: "google/cloud/storage/v1",
				},
			},
		},
		{
			name:        "library with multiple APIs",
			libraryName: "google-cloud-secret-manager",
			output:      "output/google-cloud-secret-manager",
			channels: []string{
				"google/cloud/secretmanager/v1",
				"google/cloud/secretmanager/v1beta2",
				"google/cloud/secrets/v1beta1",
			},
			want: []*config.Channel{
				{
					Path: "google/cloud/secretmanager/v1",
				},
				{
					Path: "google/cloud/secretmanager/v1beta2",
				},
				{
					Path: "google/cloud/secrets/v1beta1",
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
			if err := addLibraryToLibrarianConfig(cfg, test.libraryName, test.output, test.channels...); err != nil {
				t.Fatal(err)
			}

			cfg, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				t.Fatal(err)
			}

			if len(cfg.Libraries) != 2 {
				t.Errorf("libraries count = %d, want 2", len(cfg.Libraries))
			}

			found := findLibrary(cfg, test.libraryName)
			if found == nil {
				t.Fatalf("library %q not found in config", test.libraryName)
			}
			if found.Output != test.output {
				t.Errorf("output = %q, want %q", found.Output, test.output)
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

func findLibrary(cfg *config.Config, name string) *config.Library {
	for i := range cfg.Libraries {
		if cfg.Libraries[i].Name == name {
			return cfg.Libraries[i]
		}
	}
	return nil
}
