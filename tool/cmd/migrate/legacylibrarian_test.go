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
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
)

func TestRunMigrateLibrarian(t *testing.T) {
	fetchSource = func(ctx context.Context) (*config.Source, error) {
		return &config.Source{
			Commit: "abcd123",
			SHA256: "sha123",
			Dir:    "testdata/googleapis",
		}, nil
	}
	for _, test := range []struct {
		name     string
		repoPath string
		wantErr  error
	}{
		{
			name:     "success",
			repoPath: "testdata/run/success-python",
		},
		{
			name:     "tidy_failed",
			repoPath: "testdata/run/tidy-fails-python",
			wantErr:  errTidyFailed,
		},
		{
			name:     "no_repo_path",
			repoPath: "",
			wantErr:  errRepoNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outputPath := "librarian.yaml"
			t.Cleanup(func() {
				if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
					t.Fatalf("cleanup: remove %s: %v", outputPath, err)
				}
			})

			err := errRepoNotFound
			if test.repoPath != "" {
				err = runLibrarianMigration(t.Context(), "python", test.repoPath)
			}
			if err != nil {
				if test.wantErr == nil {
					t.Fatal(err)
				}
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("expected error containing %q, got: %v", test.wantErr, err)
				}
			} else if test.wantErr != nil {
				t.Fatalf("expected error containing %q, got nil", test.wantErr)
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
			lang:        "go",
			repoPath:    "testdata/google-cloud-go",
			state:       &legacyconfig.LibrarianState{},
			cfg:         &legacyconfig.LibrarianConfig{},
			fetchSource: defaultFetchSource,
			want: &config.Config{
				Language: "go",
				Repo:     "googleapis/google-cloud-go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "abcd123",
						SHA256: "sha123",
					},
				},
				Default: &config.Default{
					Keep:         []string{"CHANGES.md"},
					Output:       ".",
					ReleaseLevel: "ga",
					TagFormat:    defaultTagFormat,
				},
			},
		},
		{
			name:        "python_monorepo_defaults",
			lang:        "python",
			state:       &legacyconfig.LibrarianState{},
			cfg:         &legacyconfig.LibrarianConfig{},
			fetchSource: defaultFetchSource,
			want: &config.Config{
				Language: "python",
				Repo:     "googleapis/google-cloud-python",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "abcd123",
						SHA256: "sha123",
					},
				},
				Default: &config.Default{
					Output:       "packages",
					ReleaseLevel: "stable",
					TagFormat:    defaultTagFormat,
					Transport:    "grpc+rest",
				},
			},
		},
		{
			name: "no_librarian_config",
			lang: "python",
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
				Language: "python",
				Repo:     "googleapis/google-cloud-python",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "abcd123",
						SHA256: "sha123",
					},
				},
				Default: &config.Default{
					Output:       "packages",
					ReleaseLevel: "stable",
					TagFormat:    defaultTagFormat,
					Transport:    "grpc+rest",
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
				Language: "go",
				Repo:     "googleapis/google-cloud-go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "abcd123",
						SHA256: "sha123",
					},
				},
				Default: &config.Default{
					Keep:         []string{"CHANGES.md"},
					Output:       ".",
					TagFormat:    defaultTagFormat,
					ReleaseLevel: "ga",
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
									ClientDirectory: "proto_package: maps.fleetengine.v1",
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
								Path:            "google/maps/fleetengine/v1",
								ClientDirectory: "proto_package: maps.fleetengine.v1",
								DisableGAPIC:    true,
								NestedProtos:    []string{"grafeas/grafeas.proto"},
								ProtoPackage:    "google.cloud.translation.v3",
							},
						},
						ModulePathVersion: "v2",
					},
				},
			},
		},
		{
			name: "additional go module",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "ai",
							APIs: []*legacyconfig.API{
								{Path: "google/ai/generativelanguage/v1"},
								{Path: "google/ai/generativelanguage/v1alpha"},
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
				repoConfig:      nil,
				repoPath:        "testdata/google-cloud-go",
			},
			want: []*config.Library{
				{
					Name: "ai",
					APIs: []*config.API{
						{Path: "google/ai/generativelanguage/v1"},
						{Path: "google/ai/generativelanguage/v1alpha"},
					},
					ReleaseLevel: "beta",
					Go: &config.GoModule{
						GoAPIs: []*config.GoAPI{
							{
								Path:            "google/ai/generativelanguage/v1",
								ClientDirectory: "generativelanguage",
								ImportPath:      "ai/generativelanguage",
							},
							{
								Path:            "google/ai/generativelanguage/v1alpha",
								ClientDirectory: "generativelanguage",
								ImportPath:      "ai/generativelanguage",
							},
							{
								Path:            "google/ai/generativelanguage/v1beta",
								ClientDirectory: "generativelanguage",
								ImportPath:      "ai/generativelanguage",
							},
							{
								Path:            "google/ai/generativelanguage/v1beta2",
								ClientDirectory: "generativelanguage",
								ImportPath:      "ai/generativelanguage",
							},
						},
					},
				},
			},
		},
		{
			name: "fill keep list",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "accessapproval",
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
				repoConfig:      nil,
				repoPath:        "testdata/google-cloud-go",
			},
			want: []*config.Library{
				{
					Name: "accessapproval",
					Keep: []string{"aliasshim/aliasshim.go"},
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
			got, err := readRepoConfig(test.repoPath)
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

func TestLibraryWithAliasshim_Success(t *testing.T) {
	for _, test := range []struct {
		name     string
		repoPath string
		want     map[string]bool
	}{
		{
			name:     "success",
			repoPath: "testdata/library-with-aliasshim/success",
			want: map[string]bool{
				"accessapproval": true,
				"beyondcorp":     true,
				"clouddms":       true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := libraryWithAliasshim(test.repoPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLibraryWithAliasshim_Error(t *testing.T) {
	_, err := libraryWithAliasshim("testdata/non-existent-repo")
	if err == nil {
		t.Fatal(err)
	}
}
