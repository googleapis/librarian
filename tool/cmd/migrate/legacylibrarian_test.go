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

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestRunMigrateLibrarian(t *testing.T) {
	absGoogleapis, err := filepath.Abs("testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}

	fetchSource = func(ctx context.Context) (*config.Source, error) {
		return &config.Source{
			Commit: "abcd123",
			SHA256: "sha123",
			Dir:    absGoogleapis,
		}, nil
	}
	for _, test := range []struct {
		name               string
		repoPath           string
		librariesToMigrate []string
		wantLibraries      []string
	}{
		{
			name:          "success",
			repoPath:      "testdata/run/success-python",
			wantLibraries: []string{"google-ads-admanager"},
		},
		{
			name:               "selective migration",
			repoPath:           "testdata/run/selective-migration",
			librariesToMigrate: []string{"google-cloud-audit-log"},
			wantLibraries:      []string{"google-cloud-audit-log"},
		},
		{
			name:               "incremental migration without overlap",
			repoPath:           "testdata/run/incremental-migration",
			librariesToMigrate: []string{"google-cloud-audit-log"},
			// Initial librarian.yaml contains google-ads-admanager and google-cloud-functions
			wantLibraries: []string{"google-ads-admanager", "google-cloud-audit-log", "google-cloud-functions"},
		},
		{
			name:               "incremental migration with overlap",
			repoPath:           "testdata/run/incremental-migration",
			librariesToMigrate: []string{"google-ads-admanager", "google-cloud-audit-log"},
			// Initial librarian.yaml contains google-ads-admanager and google-cloud-functions
			wantLibraries: []string{"google-ads-admanager", "google-cloud-audit-log", "google-cloud-functions"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.CopyFS(dir, os.DirFS(test.repoPath)); err != nil {
				t.Fatal(err)
			}
			if err := runLibrarianMigration(t.Context(), "python", dir, test.librariesToMigrate); err != nil {
				t.Fatal(err)
			}
			gotConfig, err := yaml.Read[config.Config](filepath.Join(dir, "librarian.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			var gotLibraries []string
			for _, lib := range gotConfig.Libraries {
				gotLibraries = append(gotLibraries, lib.Name)
			}
			if diff := cmp.Diff(test.wantLibraries, gotLibraries); diff != "" {
				t.Errorf("mismatch in resulting libraries (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunMigrateLibrarian_Error(t *testing.T) {
	absGoogleapis, err := filepath.Abs("testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	fetchSource = func(ctx context.Context) (*config.Source, error) {
		return &config.Source{
			Commit: "abcd123",
			SHA256: "sha123",
			Dir:    absGoogleapis,
		}, nil
	}
	for _, test := range []struct {
		name               string
		repoPath           string
		librariesToMigrate []string
		wantErr            error
	}{
		{
			name:     "tidy_failed",
			repoPath: "testdata/run/tidy-fails-python",
			wantErr:  errTidyFailed,
		},
		{
			name:               "specified library doesn't exist",
			repoPath:           "testdata/run/selective-migration",
			librariesToMigrate: []string{"google-cloud-functions"},
			wantErr:            librarian.ErrLibraryNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.CopyFS(dir, os.DirFS(test.repoPath)); err != nil {
				t.Fatal(err)
			}
			err := runLibrarianMigration(t.Context(), "python", dir, test.librariesToMigrate)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", test.wantErr, err)
			}
		})
	}
}

func TestBuildConfigFromLibrarian(t *testing.T) {
	defaultFetchSource := func(ctx context.Context) (*config.Source, error) {
		return &config.Source{
			Commit: "abcd123",
			SHA256: "sha123",
			Dir:    "testdata/googleapis",
		}, nil
	}
	for _, test := range []struct {
		name        string
		lang        string
		repoPath    string
		state       *legacyconfig.LibrarianState
		cfg         *legacyconfig.LibrarianConfig
		fetchSource func(ctx context.Context) (*config.Source, error)
		want        *config.Config
		wantErr     error
	}{
		{
			name:        "python_monorepo_defaults",
			lang:        config.LanguagePython,
			state:       &legacyconfig.LibrarianState{},
			cfg:         &legacyconfig.LibrarianConfig{},
			fetchSource: defaultFetchSource,
			want: &config.Config{
				Language: config.LanguagePython,
				Repo:     "googleapis/google-cloud-python",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "abcd123",
						SHA256: "sha123",
					},
				},
				Release: &config.Release{
					IgnoredChanges: pythonIgnoredChanges,
				},
				Default: &config.Default{
					Output:    "packages",
					TagFormat: pythonTagFormat,
					Python: &config.PythonDefault{
						CommonGAPICPaths: pythonDefaultCommonGAPICPaths,
						LibraryType:      pythonDefaultLibraryType,
					},
				},
			},
		},
		{
			name: "no_librarian_config",
			lang: config.LanguagePython,
			state: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:      "google-cloud-secret-manager",
						Version: "1.0.0",
						APIs: []*legacyconfig.API{
							{
								Path: "google/cloud/secretmanager/v1",
							},
						},
					},
				},
			},
			cfg:         &legacyconfig.LibrarianConfig{},
			fetchSource: defaultFetchSource,
			repoPath:    "testdata/google-cloud-python",
			want: &config.Config{
				Language: config.LanguagePython,
				Repo:     "googleapis/google-cloud-python",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "abcd123",
						SHA256: "sha123",
					},
				},
				Release: &config.Release{
					IgnoredChanges: pythonIgnoredChanges,
				},
				Default: &config.Default{
					Output:    "packages",
					TagFormat: pythonTagFormat,
					Python: &config.PythonDefault{
						CommonGAPICPaths: pythonDefaultCommonGAPICPaths,
						LibraryType:      pythonDefaultLibraryType,
					},
				},
				Libraries: []*config.Library{
					{
						Name:                "google-cloud-secret-manager",
						Version:             "1.0.0",
						DescriptionOverride: "Stores, manages, and secures access to application secrets.",
						APIs: []*config.API{
							{
								Path: "google/cloud/secretmanager/v1",
							},
						},
						Python: &config.PythonPackage{
							DefaultVersion:               "v1",
							MetadataNameOverride:         "secretmanager",
							ProductDocumentationOverride: "https://cloud.google.com/secret-manager/",
							NamePrettyOverride:           "Secret Manager",
							APIShortnameOverride:         "secretmanager",
							APIIDOverride:                "secretmanager.googleapis.com",
							OptArgsByAPI: map[string][]string{
								"google/cloud/secretmanager/v1": {"warehouse-package-name=google-cloud-secret-manager"},
							},
						},
					},
				},
			},
		},
		{
			name: "fetch_source_fails",
			fetchSource: func(ctx context.Context) (*config.Source, error) {
				return nil, errors.New("fetch_source_fails")
			},
			wantErr: errFetchSource,
		},
		{
			// This is just one example of how python migration can fail,
			// used to check that when it does, the error is propagated.
			name: "python migration fails - source root doesn't exist",
			lang: "python",
			state: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:            "example-library",
						Version:       "1.0.0",
						SourceRoots:   []string{"packages/non-existent"},
						PreserveRegex: []string{"docs/CHANGELOG.md"},
					},
				},
			},
			cfg:         &legacyconfig.LibrarianConfig{},
			fetchSource: defaultFetchSource,
			wantErr:     os.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fetchSource = test.fetchSource
			input := &MigrationInput{
				librarianState:  test.state,
				librarianConfig: test.cfg,
				lang:            test.lang,
				repoPath:        test.repoPath,
			}
			got, err := buildConfigFromLibrarian(t.Context(), input)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("expected error containing %q, got: %v", test.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatal(err)
			}
			// The tests don't specify a version; ensure there is one, but
			// then clear the field for further comparisons.
			if got.Version == "" {
				t.Errorf("expected non-empty version; was empty")
			}
			got.Version = ""
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestToAPIs(t *testing.T) {
	legacyAPIs := []*legacyconfig.API{
		{Path: "google/cloud/functions/v1"},
		{Path: "google/cloud/functions/v2"},
	}
	want := []*config.API{
		{Path: "google/cloud/functions/v2"},
		{Path: "google/cloud/functions/v1"},
	}
	got := toAPIs(legacyAPIs)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestBlockLegacyGeneration(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(tempDir, librarianDir), 0755); err != nil {
		t.Fatal(err)
	}
	originalConfig := &legacyconfig.LibrarianConfig{
		TagFormat: "xyz",
		Libraries: []*legacyconfig.LibraryConfig{
			{
				LibraryID: "not-migrated",
			},
			{
				LibraryID:       "not-migrated-already-generate-blocked",
				GenerateBlocked: true,
			},
			{
				LibraryID:       "migrated-already-generate-blocked",
				GenerateBlocked: true,
			},
			{
				LibraryID:   "migrated",
				NextVersion: "1.2.3",
			},
		},
	}
	configFile := filepath.Join(tempDir, librarianDir, librarianConfigFile)
	if err := yaml.Write(configFile, originalConfig); err != nil {
		t.Fatal(err)
	}
	migratedConfig := &config.Config{
		Libraries: []*config.Library{
			{Name: "migrated-already-generate-blocked"},
			{Name: "not-previously-in-config"},
			{Name: "migrated"},
		},
	}
	if err := blockLegacyGeneration(tempDir, migratedConfig); err != nil {
		t.Fatal(err)
	}
	wantConfig := &legacyconfig.LibrarianConfig{
		TagFormat: "xyz",
		Libraries: []*legacyconfig.LibraryConfig{
			{
				LibraryID: "not-migrated",
			},
			{
				LibraryID:       "not-migrated-already-generate-blocked",
				GenerateBlocked: true,
			},
			{
				LibraryID:       "migrated-already-generate-blocked",
				GenerateBlocked: true,
			},
			{
				LibraryID:       "migrated",
				NextVersion:     "1.2.3",
				GenerateBlocked: true,
			},
			{
				LibraryID:       "not-previously-in-config",
				GenerateBlocked: true,
			},
		},
	}
	gotConfig, err := readLegacyConfig(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(wantConfig, gotConfig); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestBlockLegacyGeneration_Error(t *testing.T) {
	tempDir := t.TempDir()
	migratedConfig := &config.Config{}
	gotErr := blockLegacyGeneration(tempDir, migratedConfig)
	wantErr := os.ErrNotExist
	if !errors.Is(gotErr, wantErr) {
		t.Errorf("blockLegacyGeneration error = %v, wantErr %v", gotErr, wantErr)
	}
}

func TestFetchGoogleapisWithCommit(t *testing.T) {
	const (
		wantCommit = "abcd123"
		wantSHA    = "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8" // sha256 of "password"
	)
	// Mock GitHub server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/commits/") {
			w.Write([]byte(wantCommit))
			return
		}
		if strings.Contains(r.URL.Path, ".tar.gz") {
			w.Write([]byte("password"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	endpoints := &fetch.Endpoints{
		API:      ts.URL,
		Download: ts.URL,
	}
	// Mock cache
	tmp := t.TempDir()
	t.Setenv("LIBRARIAN_CACHE", tmp)
	// Pre-populate cache to avoid RepoDir downloading (which ignores our mock download URL)
	cachePath := filepath.Join(tmp, fmt.Sprintf("%s@%s", googleapisRepo, wantCommit))
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cachePath, "dummy"), []byte("dummy"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := fetchGoogleapisWithCommit(t.Context(), endpoints, "master")
	if err != nil {
		t.Fatal(err)
	}

	want := &config.Source{
		Commit: wantCommit,
		SHA256: wantSHA,
		Dir:    cachePath,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAugmentLegacyReleaseExcludePaths(t *testing.T) {
	for _, test := range []struct {
		name         string
		cfg          *config.Config
		initialState *legacyconfig.LibrarianState
		wantState    *legacyconfig.LibrarianState
		wantErr      error
	}{
		{
			name: "all",
			cfg: &config.Config{
				Libraries: []*config.Library{
					{Name: "existing-exclude"},
					{Name: "initially-empty-exclude"},
					{Name: "librarian-extra"},
				},
				Default: &config.Default{
					Output: "packages",
				},
				Release: &config.Release{
					IgnoredChanges: []string{"metadata", "docs/readme"},
				},
			},
			initialState: &legacyconfig.LibrarianState{
				Image: "test-image",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:                  "existing-exclude",
						Version:             "1.2.3",
						ReleaseExcludePaths: []string{"packages/existing-exclude/other"},
					},
					{
						ID:      "initially-empty-exclude",
						Version: "2.3.4",
					},
					{
						ID:      "legacylibrarian-extra",
						Version: "3.4.5",
					},
				},
			},
			wantState: &legacyconfig.LibrarianState{
				Image: "test-image",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:      "existing-exclude",
						Version: "1.2.3",
						ReleaseExcludePaths: []string{
							"packages/existing-exclude/other",
							"packages/existing-exclude/metadata",
							"packages/existing-exclude/docs/readme",
						},
						APIs:          []*legacyconfig.API{},
						SourceRoots:   []string{},
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
					},
					{
						ID:      "initially-empty-exclude",
						Version: "2.3.4",
						ReleaseExcludePaths: []string{
							"packages/initially-empty-exclude/metadata",
							"packages/initially-empty-exclude/docs/readme",
						},
						APIs:          []*legacyconfig.API{},
						SourceRoots:   []string{},
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
					},
					{
						ID:            "legacylibrarian-extra",
						Version:       "3.4.5",
						APIs:          []*legacyconfig.API{},
						SourceRoots:   []string{},
						PreserveRegex: []string{},
						RemoveRegex:   []string{},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoDir := t.TempDir()
			stateFile := filepath.Join(repoDir, librarianDir, librarianStateFile)
			if err := os.MkdirAll(filepath.Dir(stateFile), 0755); err != nil {
				t.Fatal(err)
			}
			if err := yaml.Write(stateFile, test.initialState); err != nil {
				t.Fatal(err)
			}
			if err := augmentLegacyReleaseExcludePaths(repoDir, test.cfg); err != nil {
				t.Fatal(err)
			}
			gotState, err := readState(repoDir)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantState, gotState); diff != "" {
				t.Errorf("mismatch in resulting libraries (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAugmentLegacyReleaseExcludePaths_Error(t *testing.T) {
	// Deliberately don't create a state file.
	repoDir := t.TempDir()
	cfg := &config.Config{
		Language: config.LanguagePython,
		Repo:     "google-cloud-python",
		Libraries: []*config.Library{
			{Name: "irrelevant"},
		},
		Default: &config.Default{
			Output: "packages",
		},
		Release: &config.Release{
			IgnoredChanges: []string{"irrelevant"},
		},
	}
	gotErr := augmentLegacyReleaseExcludePaths(repoDir, cfg)
	wantErr := os.ErrNotExist
	if !errors.Is(gotErr, wantErr) {
		t.Errorf("error = %v, wantErr %v", gotErr, wantErr)
	}
}
