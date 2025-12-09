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
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
)

func TestRunMigrateLibrarian(t *testing.T) {
	fetchSource = func(ctx context.Context) (*config.Source, error) {
		return &config.Source{
			Commit: "abcd123",
			SHA256: "sha123",
			Dir:    "path/to/repo",
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
		{
			name:     "unsupported_language",
			repoPath: "unused-path",
			wantErr:  errLangNotSupported,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// ensure librarian.yaml generated is removed after the test,
			// even if the test fails
			outputPath := "librarian.yaml"
			t.Cleanup(func() {
				if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
					t.Fatalf("cleanup: remove %s: %v", outputPath, err)
				}
			})

			args := []string{"-output", outputPath}
			if test.repoPath != "" {
				args = append(args, test.repoPath)
			}

			if err := run(t.Context(), args); err != nil {
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

func TestDeriveLanguage(t *testing.T) {
	for _, test := range []struct {
		name     string
		repoPath string
		want     string
		wantErr  error
	}{
		{
			name:     "golang",
			repoPath: "path/to/google-cloud-go",
			want:     "go",
		},
		{
			name:     "python",
			repoPath: "path/to/google-cloud-python",
			want:     "python",
		},
		{
			name:     "unsupported_language",
			repoPath: "path/to/unsupported-language",
			wantErr:  errLangNotSupported,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := deriveLanguage(test.repoPath)
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

func TestBuildConfig(t *testing.T) {
	defaultFetchSource := func(ctx context.Context) (*config.Source, error) {
		return &config.Source{
			Commit: "abcd123",
			SHA256: "sha123",
			Dir:    "path/to/repo",
		}, nil
	}
	for _, test := range []struct {
		name        string
		lang        string
		state       *legacyconfig.LibrarianState
		cfg         *legacyconfig.LibrarianConfig
		fetchSource func(ctx context.Context) (*config.Source, error)
		want        *config.Config
		wantErr     error
	}{
		{
			name:        "go_monorepo_defaults",
			lang:        "go",
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
						Dir:    "path/to/repo",
					},
				},
				Default: &config.Default{
					TagFormat: defaultTagFormat,
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
						Dir:    "path/to/repo",
					},
				},
				Default: &config.Default{
					TagFormat: defaultTagFormat,
				},
			},
		},
		{
			name: "no_librarian_config",
			lang: "python",
			state: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:      "example-library",
						Version: "1.0.0",
						APIs: []*legacyconfig.API{
							{
								Path: "google/example/api/v1",
							},
						},
						PreserveRegex: []string{
							"example-preserve-1",
							"example-preserve-2",
						},
					},
					{
						ID:                  "another-library",
						LastGeneratedCommit: "abcd123",
					},
				},
			},
			cfg:         &legacyconfig.LibrarianConfig{},
			fetchSource: defaultFetchSource,
			want: &config.Config{
				Language: "python",
				Repo:     "googleapis/google-cloud-python",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "abcd123",
						SHA256: "sha123",
						Dir:    "path/to/repo",
					},
				},
				Default: &config.Default{
					TagFormat: defaultTagFormat,
				},
				Libraries: []*config.Library{
					{
						Name: "another-library",
					},
					{
						Name:    "example-library",
						Version: "1.0.0",
						Channels: []*config.Channel{
							{
								Path: "google/example/api/v1",
							},
						},
						Keep: []string{
							"example-preserve-1",
							"example-preserve-2",
						},
					},
				},
			},
		},
		{
			name: "has_a_librarian_config",
			lang: "python",
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
				Language: "python",
				Repo:     "googleapis/google-cloud-python",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "abcd123",
						SHA256: "sha123",
						Dir:    "path/to/repo",
					},
				},
				Default: &config.Default{
					TagFormat: defaultTagFormat,
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
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			fetchSource = test.fetchSource
			languageCfg := &Config{
				librarianState:  test.state,
				librarianConfig: test.cfg,
				lang:            test.lang,
			}
			got, err := buildConfig(ctx, languageCfg)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("expected error containing %q, got: %v", test.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Error(err)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildLibraries(t *testing.T) {
	for _, test := range []struct {
		name    string
		langCfg *Config
		want    []*config.Library
	}{
		{
			name: "sorted_libraries",
			langCfg: &Config{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "example-library",
							APIs: []*legacyconfig.API{
								{
									Path:          "google/example/api/v1",
									ServiceConfig: "path/to/config.yaml",
								},
							},
						},
						{
							ID: "another-library",
							APIs: []*legacyconfig.API{
								{
									Path:          "google/another/api/v1",
									ServiceConfig: "another/config.yaml",
								},
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
			},
			want: []*config.Library{
				{
					Name: "another-library",
					Channels: []*config.Channel{
						{
							Path: "google/another/api/v1",
						},
					},
				},
				{
					Name: "example-library",
					Channels: []*config.Channel{
						{
							Path: "google/example/api/v1",
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildLibraries(test.langCfg)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
