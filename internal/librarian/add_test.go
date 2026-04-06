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
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestAddLibraryCommand(t *testing.T) {
	copyrightYear := strconv.Itoa(time.Now().Year())
	for _, test := range []struct {
		name                   string
		apis                   []string
		initialLibraries       []*config.Library
		wantFinalLibraries     []*config.Library
		wantGeneratedOutputDir string
		wantError              error
	}{
		{
			name:                   "create new library",
			apis:                   []string{"google/cloud/secretmanager/v1"},
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
			name: "fail create existing library",
			apis: []string{"google/cloud/secretmanager/v1"},
			initialLibraries: []*config.Library{
				{
					Name: "google-cloud-secretmanager-v1",
				},
			},
			wantGeneratedOutputDir: "existing-output",
			wantError:              errLibraryAlreadyExists,
		},
		{
			name: "create new library and tidy existing",
			apis: []string{"google/cloud/orgpolicy/v1"},
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
			err = runAdd(t.Context(), cfg, test.apis...)
			if test.wantError != nil {
				if !errors.Is(err, test.wantError) {
					t.Errorf("expected error %v, got %v", test.wantError, err)
				}
				return
			}
			if err != nil {
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

func TestAddCommand(t *testing.T) {
	googleapisDir, err := filepath.Abs("../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name     string
		apis     []string
		wantName string
		wantAPIs []*config.API
		wantErr  error
	}{
		{
			name:    "no args",
			wantErr: errMissingAPI,
		},
		{
			name:     "single API",
			apis:     []string{"google/cloud/secretmanager/v1"},
			wantName: "google-cloud-secretmanager-v1",
		},
		{
			name: "multiple APIs",
			apis: []string{
				"google/cloud/secretmanager/v1",
				"google/cloud/secretmanager/v1beta2",
				"google/cloud/secrets/v1beta1",
			},
			wantName: "google-cloud-secretmanager-v1",
			wantAPIs: []*config.API{
				{Path: "google/cloud/secretmanager/v1"},
				{Path: "google/cloud/secretmanager/v1beta2"},
				{Path: "google/cloud/secrets/v1beta1"},
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
			if err := yaml.Write(config.LibrarianYAML, cfg); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"librarian", "add"}, test.apis...)
			err := Run(t.Context(), args...)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("want error %v, got %v", test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			gotCfg, err := yaml.Read[config.Config](config.LibrarianYAML)
			if err != nil {
				t.Fatal(err)
			}
			got, err := FindLibrary(gotCfg, test.wantName)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantAPIs, got.APIs); diff != "" {
				t.Errorf("apis mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAddLibrary(t *testing.T) {
	for _, test := range []struct {
		name     string
		apis     []string
		wantName string
		wantAPIs []*config.API
	}{
		{
			name:     "library with single API",
			apis:     []string{"google/cloud/storage/v1"},
			wantName: "google-cloud-storage-v1",
			wantAPIs: []*config.API{
				{
					Path: "google/cloud/storage/v1",
				},
			},
		},
		{
			name: "library with multiple APIs",
			apis: []string{
				"google/cloud/secretmanager/v1",
				"google/cloud/secretmanager/v1beta2",
				"google/cloud/secrets/v1beta1",
			},
			wantName: "google-cloud-secretmanager-v1",
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
			cfg.Libraries = []*config.Library{
				{
					Name:   "existinglib",
					Output: "output/existinglib",
				},
			}
			if err := yaml.Write(config.LibrarianYAML, cfg); err != nil {
				t.Fatal(err)
			}
			gotName, cfg, err := addLibrary(cfg, test.apis...)
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
		apis     []string
		cfg      *config.Config
		wantName string
		wantCfg  *config.Config
	}{
		{
			name: "update existing library",
			apis: []string{"google/cloud/secretmanager/v1beta2"},
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
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)
			if err := yaml.Write(config.LibrarianYAML, test.cfg); err != nil {
				t.Fatal(err)
			}
			gotName, gotCfg, err := addLibrary(test.cfg, test.apis...)
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
		apis    []string
		cfg     *config.Config
		wantErr error
	}{
		{
			name: "fail if api already exists",
			apis: []string{"google/cloud/secretmanager/v1beta2"},
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
		{
			name: "python doesn't support updating existing library",
			apis: []string{"google/cloud/secretmanager/v1beta2"},
			cfg: &config.Config{
				Language: config.LanguagePython,
				Libraries: []*config.Library{
					{
						Name:    "google-cloud-secretmanager",
						Version: "1.2.3",
						APIs: []*config.API{
							{Path: "google/cloud/secretmanager/v1"},
							{Path: "google/cloud/secretmanager/v1beta2"},
						},
					},
				},
			},
			wantErr: errLibraryAlreadyExists,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)
			if err := yaml.Write(config.LibrarianYAML, test.cfg); err != nil {
				t.Fatal(err)
			}
			_, _, err := addLibrary(test.cfg, test.apis...)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestAddLibrary_Preview(t *testing.T) {
	for _, test := range []struct {
		name             string
		apis             []string
		initialLibraries []*config.Library
		wantPreview      *config.Library
	}{
		{
			name: "add preview to existing library",
			apis: []string{"preview/google/cloud/secretmanager/v1"},
			initialLibraries: []*config.Library{
				{
					Name: "secretmanager",
					APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
				},
			},
			wantPreview: &config.Library{
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			},
		},
		{
			name: "add preview with multiple APIs",
			apis: []string{
				"preview/google/cloud/secretmanager/v1",
				"preview/google/cloud/secretmanager/v2",
			},
			initialLibraries: []*config.Library{
				{
					Name: "secretmanager",
					APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
				},
			},
			wantPreview: &config.Library{
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
					{Path: "google/cloud/secretmanager/v2"},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				Language:  config.LanguageGo,
				Libraries: test.initialLibraries,
			}
			gotName, gotCfg, err := addLibrary(cfg, test.apis...)
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
		apis             []string
		initialLibraries []*config.Library
		wantErr          error
	}{
		{
			name: "fail if library doesn't exist",
			apis: []string{"preview/google/cloud/secretmanager/v1"},
			initialLibraries: []*config.Library{
				{
					Name: "otherlib",
					APIs: []*config.API{{Path: "google/cloud/other/v1"}},
				},
			},
			wantErr: errPreviewRequiresLibrary,
		},
		{
			name: "fail if mixing preview and non-preview APIs",
			apis: []string{
				"preview/google/cloud/secretmanager/v1",
				"google/cloud/secretmanager/v1beta2",
			},
			wantErr: errMixedPreviewAndNonPreview,
		},
		{
			name: "fail preview already exists",
			apis: []string{
				"preview/google/cloud/secretmanager/v1",
			},
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
			_, _, err := addLibrary(cfg, test.apis...)
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
	} {
		t.Run(test.language+"/"+test.apiPath, func(t *testing.T) {
			got := deriveLibraryName(test.language, test.apiPath)
			if got != test.want {
				t.Errorf("deriveLibraryName(%q, %q) = %q, want %q", test.language, test.apiPath, got, test.want)
			}
		})
	}
}

func TestSyncToStateYAML(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name         string
		initialState *legacyconfig.LibrarianState
		cfg          *config.Config
		wantState    *legacyconfig.LibrarianState
	}{
		{
			name: "sync new library",
			initialState: &legacyconfig.LibrarianState{
				Image: "gcr.io/my-image:latest",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:            "existing",
						Version:       "1.2.3",
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
						APIs:          []*legacyconfig.API{{Path: "google/cloud/existing/v1"}},
						SourceRoots:   []string{"existing"},
					},
				},
			},
			cfg: &config.Config{
				Language: config.LanguageRust,
				Libraries: []*config.Library{
					{
						Name:    "existing",
						Version: "1.2.3",
						APIs: []*config.API{
							{Path: "google/cloud/existing/v1"},
						},
					},
					{
						Name:    "new",
						Version: "0.1.0",
						APIs: []*config.API{
							{Path: "google/cloud/new/v1"},
						},
					},
				},
			},
			wantState: &legacyconfig.LibrarianState{
				Image: "gcr.io/my-image:latest",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:            "existing",
						Version:       "1.2.3",
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
						APIs:          []*legacyconfig.API{{Path: "google/cloud/existing/v1"}},
						SourceRoots:   []string{"existing"},
					},
					{
						ID:            "new",
						Version:       "0.1.0",
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
						APIs:          []*legacyconfig.API{{Path: "google/cloud/new/v1"}},
						SourceRoots:   []string{"new"},
						TagFormat:     "{id}/v{version}",
					},
				},
			},
		},
		{
			name: "multiple new libraries",
			initialState: &legacyconfig.LibrarianState{
				Image:     "gcr.io/my-image:latest",
				Libraries: []*legacyconfig.LibraryState{},
			},
			cfg: &config.Config{
				Libraries: []*config.Library{
					{Name: "lib-b", Version: "1.0.0", APIs: []*config.API{{Path: "google/cloud/lib-b/v1"}}},
					{Name: "lib-a", Version: "2.0.0", APIs: []*config.API{{Path: "google/cloud/lib-a/v1"}}},
				},
			},
			wantState: &legacyconfig.LibrarianState{
				Image: "gcr.io/my-image:latest",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:            "lib-a",
						Version:       "2.0.0",
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
						APIs:          []*legacyconfig.API{{Path: "google/cloud/lib-a/v1"}},
						SourceRoots:   []string{"lib-a"},
						TagFormat:     "{id}/v{version}",
					},
					{
						ID:            "lib-b",
						Version:       "1.0.0",
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
						APIs:          []*legacyconfig.API{{Path: "google/cloud/lib-b/v1"}},
						SourceRoots:   []string{"lib-b"},
						TagFormat:     "{id}/v{version}",
					},
				},
			},
		},
		{
			name: "no new libraries",
			initialState: &legacyconfig.LibrarianState{
				Image: "gcr.io/my-image:latest",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:            "lib-a",
						Version:       "2.0.0",
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
						APIs:          []*legacyconfig.API{{Path: "google/cloud/lib-a/v1"}},
						SourceRoots:   []string{"lib-a"},
						TagFormat:     "{id}/v{version}",
					},
					{
						ID:            "lib-b",
						Version:       "1.0.0",
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
						APIs:          []*legacyconfig.API{{Path: "google/cloud/lib-b/v1"}},
						SourceRoots:   []string{"lib-b"},
						TagFormat:     "{id}/v{version}",
					},
				},
			},
			cfg: &config.Config{
				Libraries: []*config.Library{
					{Name: "lib-b", Version: "1.0.0", APIs: []*config.API{{Path: "google/cloud/lib-b/v1"}}},
					{Name: "lib-a", Version: "2.0.0", APIs: []*config.API{{Path: "google/cloud/lib-a/v1"}}},
				},
			},
			wantState: &legacyconfig.LibrarianState{
				Image: "gcr.io/my-image:latest",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:            "lib-a",
						Version:       "2.0.0",
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
						APIs:          []*legacyconfig.API{{Path: "google/cloud/lib-a/v1"}},
						SourceRoots:   []string{"lib-a"},
						TagFormat:     "{id}/v{version}",
					},
					{
						ID:            "lib-b",
						Version:       "1.0.0",
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
						APIs:          []*legacyconfig.API{{Path: "google/cloud/lib-b/v1"}},
						SourceRoots:   []string{"lib-b"},
						TagFormat:     "{id}/v{version}",
					},
				},
			},
		},
		{
			name: "add an api to an existing library",
			initialState: &legacyconfig.LibrarianState{
				Image: "gcr.io/my-image:latest",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:            "lib-a",
						Version:       "2.0.0",
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
						APIs:          []*legacyconfig.API{{Path: "google/cloud/lib-a/v1"}},
						SourceRoots:   []string{"lib-a"},
						TagFormat:     "{id}/v{version}",
					},
				},
			},
			cfg: &config.Config{
				Libraries: []*config.Library{
					{
						Name:    "lib-a",
						Version: "2.0.0",
						APIs:    []*config.API{{Path: "google/cloud/lib-a/v1"}, {Path: "google/cloud/lib-a/v2"}}},
				},
			},
			wantState: &legacyconfig.LibrarianState{
				Image: "gcr.io/my-image:latest",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:            "lib-a",
						Version:       "2.0.0",
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
						APIs: []*legacyconfig.API{
							{Path: "google/cloud/lib-a/v1"},
							{Path: "google/cloud/lib-a/v2"},
						},
						SourceRoots: []string{"lib-a"},
						TagFormat:   "{id}/v{version}",
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			stateFile := filepath.Join(tmpDir, legacyconfig.LibrarianDir, legacyconfig.LibrarianStateFile)
			if err := os.Mkdir(filepath.Join(tmpDir, legacyconfig.LibrarianDir), 0755); err != nil {
				t.Fatal(err)
			}
			if err := yaml.Write(stateFile, test.initialState); err != nil {
				t.Fatal(err)
			}
			if err := syncToStateYAML(tmpDir, test.cfg); err != nil {
				t.Fatal(err)
			}
			gotState, err := yaml.Read[legacyconfig.LibrarianState](stateFile)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantState, gotState); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSyncToStateYAML_Error(t *testing.T) {
	for _, test := range []struct {
		name         string
		initialState *legacyconfig.LibrarianState
		cfg          *config.Config
		wantError    error
	}{
		{
			name:      "state.yaml does not exist",
			wantError: fs.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := syncToStateYAML(t.TempDir(), test.cfg)
			if !errors.Is(err, test.wantError) {
				t.Errorf("syncToStateYAML(%s): got error %v, want %v", test.name, err, test.wantError)
			}
		})
	}
}
