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
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestAddLibrary(t *testing.T) {
	copyrightYear := strconv.Itoa(time.Now().Year())
	for _, test := range []struct {
		name                   string
		libName                string
		initialLibraries       []*config.Library
		wantFinalLibraries     []*config.Library
		wantGeneratedOutputDir string
		wantError              error
	}{
		{
			name:                   "create new library",
			libName:                "google-cloud-secretmanager",
			initialLibraries:       []*config.Library{},
			wantGeneratedOutputDir: "newlib-output",
			wantFinalLibraries: []*config.Library{
				{
					Name:          "google-cloud-secretmanager",
					CopyrightYear: copyrightYear,
				},
			},
		},
		{
			name:    "fail create existing library",
			libName: "google-cloud-secretmanager",
			initialLibraries: []*config.Library{
				{
					Name: "google-cloud-secretmanager",
				},
			},
			wantGeneratedOutputDir: "existing-output",
			wantError:              errLibraryAlreadyExists,
		},
		{
			name:    "create new library and tidy existing",
			libName: "google-cloud-secretmanager",
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
					Name:          "google-cloud-secretmanager",
					CopyrightYear: copyrightYear,
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			googleapisDir, err := filepath.Abs("testdata/googleapis")
			if err != nil {
				t.Fatal(err)
			}
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)

			cfg := sample.Config()
			cfg.Default.Output = "output"
			cfg.Libraries = test.initialLibraries
			cfg.Sources.Googleapis.Dir = googleapisDir
			if err := yaml.Write(librarianConfigPath, cfg); err != nil {
				t.Fatal(err)
			}
			err = runAdd(t.Context(), test.libName)
			if test.wantError != nil {
				if !errors.Is(err, test.wantError) {
					t.Errorf("expected error %v, got %v", test.wantError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("runCreate() failed with unexpected error: %v", err)
			}

			gotCfg, err := yaml.Read[config.Config](librarianConfigPath)
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

func TestAddCommand(t *testing.T) {
	googleapisDir, err := filepath.Abs("testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}

	testName := "google-cloud-secret-manager"
	for _, test := range []struct {
		name     string
		args     []string
		wantErr  error
		wantAPIs []*config.API
	}{
		{
			name:    "no args",
			args:    []string{"librarian", "add"},
			wantErr: errMissingLibraryName,
		},
		{
			name: "library name only",
			args: []string{
				"librarian",
				"add",
				testName,
			},
		},
		{
			name: "library with single API",
			args: []string{
				"librarian",
				"add",
				testName,
				"google/cloud/secretmanager/v1",
			},
			wantAPIs: []*config.API{
				{
					Path: "google/cloud/secretmanager/v1",
				},
			},
		},
		{
			name: "library with multiple APIs",
			args: []string{
				"librarian",
				"add",
				testName,
				"google/cloud/secretmanager/v1",
				"google/cloud/secretmanager/v1beta2",
				"google/cloud/secrets/v1beta1",
			},
			wantAPIs: []*config.API{
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

			cfg := sample.Config()
			cfg.Default.Output = "output"
			cfg.Libraries = nil
			cfg.Sources.Googleapis.Dir = googleapisDir
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
			got, err := findLibrary(gotCfg, testName)
			if err != nil {
				t.Fatal(err)
			}
			if test.wantAPIs != nil {
				if diff := cmp.Diff(test.wantAPIs, got.APIs); diff != "" {
					t.Errorf("apis mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestAddLibraryToLibrarianYaml(t *testing.T) {
	for _, test := range []struct {
		name        string
		libraryName string
		apis        []string
		want        []*config.API
	}{
		{
			name:        "library with no specification-source",
			libraryName: "newlib",
		},
		{
			name:        "library with single API",
			libraryName: "newlib",
			apis:        []string{"google/cloud/storage/v1"},
			want: []*config.API{
				{
					Path: "google/cloud/storage/v1",
				},
			},
		},
		{
			name:        "library with multiple APIs",
			libraryName: "google-cloud-secret-manager",
			apis: []string{
				"google/cloud/secretmanager/v1",
				"google/cloud/secretmanager/v1beta2",
				"google/cloud/secrets/v1beta1",
			},
			want: []*config.API{
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

			cfg := sample.Config()
			cfg.Libraries = []*config.Library{
				{
					Name:   "existinglib",
					Output: "output/existinglib",
				},
			}
			if err := yaml.Write(librarianConfigPath, cfg); err != nil {
				t.Fatal(err)
			}
			cfg = addLibraryToLibrarianConfig(cfg, test.libraryName, test.apis...)
			if len(cfg.Libraries) != 2 {
				t.Errorf("libraries count = %d, want 2", len(cfg.Libraries))
			}

			found, err := findLibrary(cfg, test.libraryName)
			if err != nil {
				t.Fatal(err)
			}
			if found.Version != "" {
				t.Errorf("version = %q, want %q", found.Version, "")
			}
			if diff := cmp.Diff(test.want, found.APIs); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
