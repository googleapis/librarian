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
	"strings"
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
					Keep:         []string{"CHANGES.md", "go.mod", "go.sum"},
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
					Python:       &config.PythonDefault{CommonGAPICPaths: pythonDefaultCommonGAPICPaths},
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
					Python:       &config.PythonDefault{CommonGAPICPaths: pythonDefaultCommonGAPICPaths},
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
					Keep:         []string{"CHANGES.md", "go.mod", "go.sum"},
					Output:       ".",
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
			name: "parse BUILD.bazel with no GAPIC rule",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "example-library",
							APIs: []*legacyconfig.API{
								{
									Path: "google/cloud/no-gapic",
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
					Name: "example-library",
					APIs: []*config.API{{Path: "google/cloud/no-gapic"}},
				},
			},
		},
		{
			name: "parse BUILD.bazel with no Go API",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "bigquery",
							APIs: []*legacyconfig.API{
								{
									Path: "google/cloud/bigquery/analyticshub/v1",
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
					APIs: []*config.API{{Path: "google/cloud/bigquery/analyticshub/v1"}},
					Go: &config.GoModule{
						GoAPIs: []*config.GoAPI{
							{
								Path:               "google/cloud/bigquery/analyticshub/v1",
								NoRESTNumericEnums: true,
							},
						},
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
					Go: &config.GoModule{
						GoAPIs: []*config.GoAPI{
							{
								NoRESTNumericEnums: true,
								Path:               "google/cloud/bigquery/analyticshub/v1",
							},
						},
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
								NoRESTNumericEnums:       true,
								Path:                     "google/cloud/bigquery/v2",
							},
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
				NoRESTNumericEnums: true,
			},
		},
		{
			name:          "custom import path",
			googleapisDir: "testdata/parse-bazel/custom-import-path",
			buildPath:     "google/longrunning",
			want: &goGAPICInfo{
				ClientPackageName:  "longrunning",
				ImportPath:         "longrunning/autogen",
				NoRESTNumericEnums: true,
			},
		},
		{
			name:          "no GAPIC rules",
			googleapisDir: "testdata/parse-bazel/no-gapic-rule",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseBazel(test.googleapisDir, test.buildPath)
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
			_, err := parseBazel("", test.dir)
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
