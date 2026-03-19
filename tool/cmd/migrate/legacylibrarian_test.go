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
	"github.com/google/go-cmp/cmp/cmpopts"
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
			name:        "go_monorepo_defaults",
			lang:        config.LanguageGo,
			repoPath:    "testdata/google-cloud-go",
			state:       &legacyconfig.LibrarianState{},
			cfg:         &legacyconfig.LibrarianConfig{},
			fetchSource: defaultFetchSource,
			want: &config.Config{
				Language: config.LanguageGo,
				Repo:     "googleapis/google-cloud-go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "abcd123",
						SHA256: "sha123",
					},
				},
				Release: &config.Release{
					Branch: "main",
				},
				Default: &config.Default{
					ReleaseLevel: "ga",
					TagFormat:    defaultTagFormat,
				},
			},
		},
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
				Release: &config.Release{Branch: "main"},
				Default: &config.Default{
					Output:       "packages",
					ReleaseLevel: "stable",
					TagFormat:    pythonTagFormat,
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
				Release: &config.Release{Branch: "main"},
				Default: &config.Default{
					Output:       "packages",
					ReleaseLevel: "stable",
					TagFormat:    pythonTagFormat,
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
			name:     "has_a_librarian_config",
			lang:     "go",
			repoPath: "testdata/google-cloud-go",
			state: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:      "example-library",
						Version: "1.0.0",
					},
					{
						ID:      "another-library",
						Version: "2.0.0",
					},
				},
			},
			cfg: &legacyconfig.LibrarianConfig{
				Libraries: []*legacyconfig.LibraryConfig{
					{
						LibraryID:       "example-library",
						GenerateBlocked: true,
						ReleaseBlocked:  true,
					},
				},
			},
			fetchSource: defaultFetchSource,
			want: &config.Config{
				Language: config.LanguageGo,
				Repo:     "googleapis/google-cloud-go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "abcd123",
						SHA256: "sha123",
					},
				},
				Release: &config.Release{
					Branch: "main",
				},
				Default: &config.Default{
					ReleaseLevel: "ga",
					TagFormat:    defaultTagFormat,
				},
				Libraries: []*config.Library{
					{
						Name:    "another-library",
						Version: "2.0.0",
					},
					{
						Name:         "example-library",
						Version:      "1.0.0",
						SkipGenerate: true,
						SkipRelease:  true,
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
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildGoLibraries(t *testing.T) {
	for _, test := range []struct {
		name  string
		input *MigrationInput
		want  []*config.Library
	}{
		{
			name: "go_libraries",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "another-library",
							APIs: []*legacyconfig.API{
								{
									Path:          "google/another/api/v1",
									ServiceConfig: "another/config.yaml",
								},
							},
						},
						{
							ID: "example-library",
							APIs: []*legacyconfig.API{
								{
									Path:          "google/example/api/v1",
									ServiceConfig: "path/to/config.yaml",
								},
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
				repoConfig: &RepoConfig{
					Modules: []*RepoConfigModule{
						{
							Name: "example-library",
							DeleteGenerationOutputPaths: []string{
								"internal/generated/snippets/storage/internal",
							},
							APIs: []*RepoConfigAPI{
								{
									Path:            "google/maps/fleetengine/v1",
									ClientDirectory: "fleetengine",
									DisableGAPIC:    true,
									NestedProtos:    []string{"grafeas/grafeas.proto"},
									ProtoPackage:    "google.cloud.translation.v3",
								},
							},
							ModulePathVersion: "v2",
						},
					},
				},
				repoPath: "testdata/google-cloud-go",
			},
			want: []*config.Library{
				{
					Name: "another-library",
					APIs: []*config.API{
						{
							Path: "google/another/api/v1",
						},
					},
				},
				{
					Name: "example-library",
					APIs: []*config.API{
						{
							Path: "google/example/api/v1",
						},
					},
					Go: &config.GoModule{
						DeleteGenerationOutputPaths: []string{
							"internal/generated/snippets/storage/internal",
						},
						GoAPIs: []*config.GoAPI{
							{
								Path:          "google/maps/fleetengine/v1",
								ClientPackage: "fleetengine",
								ProtoOnly:     true,
								NestedProtos:  []string{"grafeas/grafeas.proto"},
								ProtoPackage:  "google.cloud.translation.v3",
							},
						},
						ModulePathVersion: "v2",
					},
				},
			},
		},
		{
			name: "parse BUILD.bazel with empty Go API",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "bigquery",
							APIs: []*legacyconfig.API{
								{
									Path: "google/cloud/bigquery/biglake/v1",
								},
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
				repoConfig:      nil,
				repoPath:        "testdata/google-cloud-go",
				googleapisDir:   "testdata/googleapis",
			},
			want: []*config.Library{
				{
					Name: "bigquery",
					APIs: []*config.API{{Path: "google/cloud/bigquery/biglake/v1"}},
					Keep: []string{"README.md"},
					Go:   &config.GoModule{NestedModule: "v2"},
				},
			},
		},
		{
			name: "parse BUILD.bazel with Go API merge",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID:   "bigquery",
							APIs: []*legacyconfig.API{{Path: "google/cloud/bigquery/analyticshub/v1"}},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
				repoConfig: &RepoConfig{
					Modules: []*RepoConfigModule{
						{
							Name: "bigquery",
							APIs: []*RepoConfigAPI{{Path: "google/cloud/bigquery/analyticshub/v1"}},
						},
					},
				},
				repoPath:      "testdata/google-cloud-go",
				googleapisDir: "testdata/googleapis",
			},
			want: []*config.Library{
				{
					Name: "bigquery",
					APIs: []*config.API{{Path: "google/cloud/bigquery/analyticshub/v1"}},
					Keep: []string{"README.md"},
					Go: &config.GoModule{
						GoAPIs: []*config.GoAPI{
							{
								NoMetadata: true,
								Path:       "google/cloud/bigquery/analyticshub/v1",
							},
						},
						NestedModule: "v2",
					},
				},
			},
		},
		{
			name: "parse enable generator features from api level",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "secretmanager",
							APIs: []*legacyconfig.API{
								{
									Path: "google/cloud/secretmanager/v1",
								},
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
				repoConfig: &RepoConfig{
					Modules: []*RepoConfigModule{
						{
							Name: "secretmanager",
							EnabledGeneratorFeatures: []string{
								"feature-1",
								"feature-2",
							},
							APIs: []*RepoConfigAPI{
								{
									EnabledGeneratorFeatures: []string{
										"feature-3",
										"feature-1",
									},
									Path: "google/cloud/secretmanager/v1",
								},
							},
						},
					},
				},
				repoPath:      "testdata/google-cloud-go",
				googleapisDir: "testdata/googleapis",
			},
			want: []*config.Library{
				{
					Name: "secretmanager",
					APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
					Go: &config.GoModule{
						GoAPIs: []*config.GoAPI{
							{
								EnabledGeneratorFeatures: []string{
									"feature-1",
									"feature-2",
									"feature-3",
								},
								Path: "google/cloud/secretmanager/v1",
							},
						},
					},
				},
			},
		},
		{
			name: "parse enable generator features from library level",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "secretmanager",
							APIs: []*legacyconfig.API{
								{
									Path: "google/cloud/secretmanager/v1",
								},
								{
									Path: "google/cloud/secretmanager/v1beta1",
								},
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
				repoConfig: &RepoConfig{
					Modules: []*RepoConfigModule{
						{
							Name: "secretmanager",
							EnabledGeneratorFeatures: []string{
								"feature-1",
								"feature-2",
							},
							APIs: []*RepoConfigAPI{
								{
									Path: "google/cloud/secretmanager/v1beta1",
								},
							},
						},
					},
				},
				repoPath:      "testdata/google-cloud-go",
				googleapisDir: "testdata/googleapis",
			},
			want: []*config.Library{
				{
					Name: "secretmanager",
					APIs: []*config.API{
						{Path: "google/cloud/secretmanager/v1"},
						{Path: "google/cloud/secretmanager/v1beta1"},
					},
					Go: &config.GoModule{
						GoAPIs: []*config.GoAPI{
							{
								// This API is created because the enabled
								// generator features are not empty.
								EnabledGeneratorFeatures: []string{
									"feature-1",
									"feature-2",
								},
								Path: "google/cloud/secretmanager/v1",
							},
							{
								// Enabled generator features merge into
								// this API.
								EnabledGeneratorFeatures: []string{
									"feature-1",
									"feature-2",
								},
								Path: "google/cloud/secretmanager/v1beta1",
							},
						},
					},
				},
			},
		},
		{
			name: "nested major version",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "bigquery/v2",
							APIs: []*legacyconfig.API{
								{
									Path: "google/cloud/bigquery/v2",
								},
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
				repoConfig: &RepoConfig{
					Modules: []*RepoConfigModule{
						{
							Name: "bigquery/v2",
							APIs: []*RepoConfigAPI{
								{
									Path: "google/cloud/bigquery/v2",
									EnabledGeneratorFeatures: []string{
										"F_wrapper_types_for_page_size",
									},
								},
							},
						},
					},
				},
				repoPath:      "testdata/google-cloud-go",
				googleapisDir: "testdata/googleapis",
			},
			want: []*config.Library{
				{
					Name: "bigquery/v2",
					APIs: []*config.API{
						{Path: "google/cloud/bigquery/v2"},
					},
					Go: &config.GoModule{
						GoAPIs: []*config.GoAPI{
							{
								EnabledGeneratorFeatures: []string{"F_wrapper_types_for_page_size"},
								ImportPath:               "bigquery/v2/apiv2",
								Path:                     "google/cloud/bigquery/v2",
							},
						},
					},
				},
			},
		},
		{
			name: "parse metadata from BUILD.bazel",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "asset",
							APIs: []*legacyconfig.API{
								{
									Path: "google/cloud/asset/v1",
								},
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
				repoPath:        "testdata/google-cloud-go",
				googleapisDir:   "testdata/googleapis",
			},
			want: []*config.Library{
				{
					Name: "asset",
					APIs: []*config.API{
						{Path: "google/cloud/asset/v1"},
					},
					Go: &config.GoModule{
						GoAPIs: []*config.GoAPI{
							{
								NoMetadata: true,
								Path:       "google/cloud/asset/v1",
							},
						},
					},
				},
			},
		},
		{
			name: "parse disable GAPIC BUILD.bazel",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "asset",
							APIs: []*legacyconfig.API{
								{
									Path: "google/cloud/no-gapic",
								},
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
				repoPath:        "testdata/google-cloud-go",
				googleapisDir:   "testdata/googleapis",
			},
			want: []*config.Library{
				{
					Name: "asset",
					APIs: []*config.API{
						{Path: "google/cloud/no-gapic"},
					},
					Go: &config.GoModule{
						GoAPIs: []*config.GoAPI{
							{
								ProtoOnly: true,
								Path:      "google/cloud/no-gapic",
							},
						},
					},
				},
			},
		},
		{
			name: "parse diregapic from BUILD.bazel",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "compute",
							APIs: []*legacyconfig.API{
								{
									Path: "google/cloud/compute/v1",
								},
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
				repoPath:        "testdata/google-cloud-go",
				googleapisDir:   "testdata/googleapis",
			},
			want: []*config.Library{
				{
					Name: "compute",
					APIs: []*config.API{
						{Path: "google/cloud/compute/v1"},
					},
					Go: &config.GoModule{
						GoAPIs: []*config.GoAPI{
							{
								DIREGAPIC: true,
								Path:      "google/cloud/compute/v1",
							},
						},
						NestedModule: "metadata",
					},
				},
			},
		},
		{
			name: "add nested module to a library",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "pubsub",
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
				repoPath:        "testdata/google-cloud-go",
				googleapisDir:   "testdata/googleapis",
			},
			want: []*config.Library{
				{
					Name: "pubsub",
					Keep: []string{"README.md", "internal/version.go"},
					Go: &config.GoModule{
						NestedModule: "v2",
					},
				},
			},
		},
		{
			name: "add keep to library",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "vmmigration",
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
				repoPath:        "testdata/google-cloud-go",
				googleapisDir:   "testdata/googleapis",
			},
			want: []*config.Library{
				{
					Name: "vmmigration",
					Keep: []string{"apiv1/iam_policy_client.go"},
				},
			},
		},
		{
			name: "add output",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "root-module",
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
				repoPath:        "testdata/google-cloud-go",
				googleapisDir:   "testdata/googleapis",
			},
			want: []*config.Library{
				{
					Name:   "root-module",
					Output: ".",
					Keep:   []string{"README.md", "internal/version.go"},
				},
			},
		},
		{
			name: "bigtable proto_only is not override",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "bigtable",
							APIs: []*legacyconfig.API{
								{Path: "google/bigtable/admin/v2"},
								{Path: "google/bigtable/v2"},
							},
						},
					},
				},
				repoConfig: &RepoConfig{
					Modules: []*RepoConfigModule{
						{
							Name: "bigtable",
							APIs: []*RepoConfigAPI{
								{Path: "google/bigtable/admin/v2", DisableGAPIC: true},
								{Path: "google/bigtable/v2", DisableGAPIC: true},
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
				repoPath:        "testdata/google-cloud-go",
				googleapisDir:   "testdata/googleapis",
			},
			want: []*config.Library{
				{
					Name: "bigtable",
					APIs: []*config.API{
						{Path: "google/bigtable/admin/v2"},
						{Path: "google/bigtable/v2"},
					},
					Go: &config.GoModule{
						GoAPIs: []*config.GoAPI{
							{Path: "google/bigtable/admin/v2", NoMetadata: true, ProtoOnly: true},
							{Path: "google/bigtable/v2", NoMetadata: true, ProtoOnly: true},
						},
					},
				},
			},
		},
		{
			name: "shopping type import path is not override",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "shopping",
							APIs: []*legacyconfig.API{
								{Path: "google/shopping/type"},
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
				repoPath:        "testdata/google-cloud-go",
				googleapisDir:   "testdata/googleapis",
			},
			want: []*config.Library{
				{
					Name: "shopping",
					APIs: []*config.API{
						{Path: "google/shopping/type"},
					},
					Go: &config.GoModule{
						GoAPIs: []*config.GoAPI{
							{Path: "google/shopping/type", ImportPath: "shopping/type", ProtoOnly: true},
						},
					},
				},
			},
		},
		{
			name: "delete output after generation",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "storage",
							APIs: []*legacyconfig.API{
								{Path: "google/storage/v2"},
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
				repoPath:        "testdata/google-cloud-go",
				googleapisDir:   "testdata/googleapis",
			},
			want: []*config.Library{
				{
					Name: "storage",
					APIs: []*config.API{
						{Path: "google/storage/v2"},
					},
					Keep: []string{"README.md"},
					Go: &config.GoModule{
						DeleteGenerationOutputPaths: []string{"../internal/generated/snippets/storage/internal"},
						GoAPIs: []*config.GoAPI{
							{Path: "google/storage/v2", ImportPath: "storage/internal/apiv2"},
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildGoLibraries(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got, cmpopts.SortSlices(func(a, b *config.Library) bool {
				return a.Name < b.Name
			})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReadRepoConfig(t *testing.T) {
	for _, test := range []struct {
		name     string
		repoPath string
		want     *RepoConfig
		wantErr  error
	}{
		{
			name:     "success",
			repoPath: "testdata/read-repo-config/success",
			want: &RepoConfig{
				Modules: []*RepoConfigModule{
					{
						Name: "bigquery/v2",
						APIs: []*RepoConfigAPI{
							{
								Path:            "google/cloud/bigquery/v2",
								ClientDirectory: "v2/apiv2"},
						},
					},
					{
						Name: "bigtable",
						APIs: []*RepoConfigAPI{
							{Path: "google/bigtable/v2", DisableGAPIC: true},
							{Path: "google/bigtable/admin/v2", DisableGAPIC: true},
						},
					},
				},
			},
		},
		{
			name:     "no_file",
			repoPath: "testdata/read-repo-config/no_file",
			want:     nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := readLegacyGoRepoConfig(test.repoPath)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("expected error containing %q, got: %v", test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseBazel(t *testing.T) {
	for _, test := range []struct {
		name          string
		googleapisDir string
		buildPath     string
		want          *goGAPICInfo
	}{
		{
			name:          "success",
			googleapisDir: "testdata/parse-bazel/success",
			buildPath:     "google/cloud/bigquery/analyticshub/v1",
			want: &goGAPICInfo{
				NoMetadata: true,
			},
		},
		{
			name:          "custom import path",
			googleapisDir: "testdata/parse-bazel/custom-import-path",
			buildPath:     "google/longrunning",
			want: &goGAPICInfo{
				ClientPackageName: "longrunning",
				ImportPath:        "longrunning/autogen",
			},
		},
		{
			name:          "no GAPIC rules",
			googleapisDir: "testdata/parse-bazel/no-gapic-rule",
			want: &goGAPICInfo{
				DisableGAPIC: true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseGoBazel(test.googleapisDir, test.buildPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseBazel_Error(t *testing.T) {
	for _, test := range []struct {
		name       string
		dir        string
		wantErrMsg string
	}{
		{
			name:       "multiple GAPIC rules",
			dir:        "testdata/parse-bazel/error",
			wantErrMsg: "multiple go_gapic_library rules",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseGoBazel("", test.dir)
			if err == nil {
				t.Fatalf("parseBazel(%q): expected error", test.dir)
			}
			if !strings.Contains(err.Error(), test.wantErrMsg) {
				t.Errorf("mismatch (-want +got):\n%s\n%s", test.wantErrMsg, err.Error())
			}
		})
	}
}

func TestToAPIs(t *testing.T) {
	legacyAPIs := []*legacyconfig.API{
		{Path: "google/cloud/functions/v2"},
		{Path: "google/cloud/functions/v1"},
	}
	want := []*config.API{
		{Path: "google/cloud/functions/v1"},
		{Path: "google/cloud/functions/v2"},
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
