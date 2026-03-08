// Copyright 2026 Google LLC
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
)

func TestBuildDotnetConfig(t *testing.T) {
	for _, test := range []struct {
		name string
		apis *ApisJSON
		want *config.Config
	}{
		{
			name: "basic generated library",
			apis: &ApisJSON{
				APIs: []APIEntry{
					{
						ID:        "Google.Cloud.SecretManager.V1",
						Version:   "2.7.0",
						Generator: "micro",
						ProtoPath: "google/cloud/secretmanager/v1",
						Transport: "grpc+rest",
					},
				},
			},
			want: &config.Config{
				Language: "dotnet",
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "testgoogleapis"},
				},
				Default: &config.Default{
					Output:    "apis",
					TagFormat: "{name}-{version}",
				},
				Libraries: []*config.Library{
					{
						Name:    "Google.Cloud.SecretManager.V1",
						Version: "2.7.0",
						APIs: []*config.API{
							{Path: "google/cloud/secretmanager/v1"},
						},
					},
				},
			},
		},
		{
			name: "proto-only library",
			apis: &ApisJSON{
				APIs: []APIEntry{
					{
						ID:        "Google.Cloud.Iam.V1",
						Version:   "1.0.0",
						Generator: "proto",
						ProtoPath: "google/iam/v1",
						Transport: "grpc+rest",
					},
				},
			},
			want: &config.Config{
				Language: "dotnet",
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "testgoogleapis"},
				},
				Default: &config.Default{
					Output:    "apis",
					TagFormat: "{name}-{version}",
				},
				Libraries: []*config.Library{
					{
						Name:    "Google.Cloud.Iam.V1",
						Version: "1.0.0",
						APIs: []*config.API{
							{Path: "google/iam/v1"},
						},
						Dotnet: &config.DotnetPackage{
							Generator: "proto",
						},
					},
				},
			},
		},
		{
			name: "handwritten library",
			apis: &ApisJSON{
				APIs: []APIEntry{
					{
						ID:        "Google.Cloud.Storage.V2",
						Version:   "4.0.0",
						Generator: "None",
						Transport: "grpc+rest",
					},
				},
			},
			want: &config.Config{
				Language: "dotnet",
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "testgoogleapis"},
				},
				Default: &config.Default{
					Output:    "apis",
					TagFormat: "{name}-{version}",
				},
				Libraries: []*config.Library{
					{
						Name:    "Google.Cloud.Storage.V2",
						Version: "4.0.0",
						Veneer:  true,
					},
				},
			},
		},
		{
			name: "preview version",
			apis: &ApisJSON{
				APIs: []APIEntry{
					{
						ID:        "Google.Cloud.Foo.V1Beta1",
						Version:   "1.0.0-beta05",
						Generator: "micro",
						ProtoPath: "google/cloud/foo/v1beta1",
						Transport: "grpc+rest",
					},
				},
			},
			want: &config.Config{
				Language: "dotnet",
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "testgoogleapis"},
				},
				Default: &config.Default{
					Output:    "apis",
					TagFormat: "{name}-{version}",
				},
				Libraries: []*config.Library{
					{
						Name:         "Google.Cloud.Foo.V1Beta1",
						Version:      "1.0.0-beta05",
						ReleaseLevel: "preview",
						APIs: []*config.API{
							{Path: "google/cloud/foo/v1beta1"},
						},
					},
				},
			},
		},
		{
			name: "non-default transport",
			apis: &ApisJSON{
				APIs: []APIEntry{
					{
						ID:        "Google.Cloud.Compute.V1",
						Version:   "3.0.0",
						Generator: "micro",
						ProtoPath: "google/cloud/compute/v1",
						Transport: "rest",
					},
				},
			},
			want: &config.Config{
				Language: "dotnet",
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "testgoogleapis"},
				},
				Default: &config.Default{
					Output:    "apis",
					TagFormat: "{name}-{version}",
				},
				Libraries: []*config.Library{
					{
						Name:      "Google.Cloud.Compute.V1",
						Version:   "3.0.0",
						Transport: "rest",
						APIs: []*config.API{
							{Path: "google/cloud/compute/v1"},
						},
					},
				},
			},
		},
		{
			name: "dependencies filtered",
			apis: &ApisJSON{
				APIs: []APIEntry{
					{
						ID:        "Google.Cloud.SecretManager.V1",
						Version:   "2.7.0",
						Generator: "micro",
						ProtoPath: "google/cloud/secretmanager/v1",
						Transport: "grpc+rest",
						Dependencies: map[string]string{
							"Google.Api.Gax.Grpc":               "default",
							"Google.Cloud.Iam.V1":               "project",
							"Google.Cloud.SecretManager.V1Beta": "1.0.0",
						},
					},
				},
			},
			want: &config.Config{
				Language: "dotnet",
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "testgoogleapis"},
				},
				Default: &config.Default{
					Output:    "apis",
					TagFormat: "{name}-{version}",
				},
				Libraries: []*config.Library{
					{
						Name:    "Google.Cloud.SecretManager.V1",
						Version: "2.7.0",
						APIs: []*config.API{
							{Path: "google/cloud/secretmanager/v1"},
						},
						Dotnet: &config.DotnetPackage{
							Dependencies: map[string]string{
								"Google.Cloud.SecretManager.V1Beta": "1.0.0",
							},
						},
					},
				},
			},
		},
		{
			name: "package group",
			apis: &ApisJSON{
				APIs: []APIEntry{
					{
						ID:        "Google.Cloud.Datastore.V1",
						Version:   "4.0.0",
						Generator: "micro",
						ProtoPath: "google/datastore/v1",
						Transport: "grpc+rest",
					},
					{
						ID:        "Google.Cloud.Datastore.Admin.V1",
						Version:   "3.0.0",
						Generator: "micro",
						ProtoPath: "google/datastore/admin/v1",
						Transport: "grpc+rest",
					},
				},
				PackageGroups: []PackageGroup{
					{
						ID: "Google.Cloud.Datastore",
						PackageIDs: []string{
							"Google.Cloud.Datastore.V1",
							"Google.Cloud.Datastore.Admin.V1",
						},
					},
				},
			},
			want: &config.Config{
				Language: "dotnet",
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "testgoogleapis"},
				},
				Default: &config.Default{
					Output:    "apis",
					TagFormat: "{name}-{version}",
				},
				Libraries: []*config.Library{
					{
						Name:    "Google.Cloud.Datastore.Admin.V1",
						Version: "3.0.0",
						APIs: []*config.API{
							{Path: "google/datastore/admin/v1"},
						},
					},
					{
						Name:    "Google.Cloud.Datastore.V1",
						Version: "4.0.0",
						APIs: []*config.API{
							{Path: "google/datastore/v1"},
						},
						Dotnet: &config.DotnetPackage{
							PackageGroup: []string{
								"Google.Cloud.Datastore.V1",
								"Google.Cloud.Datastore.Admin.V1",
							},
						},
					},
				},
			},
		},
		{
			name: "block release",
			apis: &ApisJSON{
				APIs: []APIEntry{
					{
						ID:           "Google.Cloud.Blocked.V1",
						Version:      "1.0.0",
						Generator:    "micro",
						ProtoPath:    "google/cloud/blocked/v1",
						Transport:    "grpc+rest",
						BlockRelease: "Blocked for testing",
					},
				},
			},
			want: &config.Config{
				Language: "dotnet",
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "testgoogleapis"},
				},
				Default: &config.Default{
					Output:    "apis",
					TagFormat: "{name}-{version}",
				},
				Libraries: []*config.Library{
					{
						Name:        "Google.Cloud.Blocked.V1",
						Version:     "1.0.0",
						SkipRelease: true,
						APIs: []*config.API{
							{Path: "google/cloud/blocked/v1"},
						},
					},
				},
			},
		},
		{
			name: "sorted by name",
			apis: &ApisJSON{
				APIs: []APIEntry{
					{
						ID:        "Google.Cloud.Zzz.V1",
						Version:   "1.0.0",
						Generator: "micro",
						ProtoPath: "google/cloud/zzz/v1",
						Transport: "grpc+rest",
					},
					{
						ID:        "Google.Cloud.Aaa.V1",
						Version:   "2.0.0",
						Generator: "micro",
						ProtoPath: "google/cloud/aaa/v1",
						Transport: "grpc+rest",
					},
				},
			},
			want: &config.Config{
				Language: "dotnet",
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "testgoogleapis"},
				},
				Default: &config.Default{
					Output:    "apis",
					TagFormat: "{name}-{version}",
				},
				Libraries: []*config.Library{
					{
						Name:    "Google.Cloud.Aaa.V1",
						Version: "2.0.0",
						APIs: []*config.API{
							{Path: "google/cloud/aaa/v1"},
						},
					},
					{
						Name:    "Google.Cloud.Zzz.V1",
						Version: "1.0.0",
						APIs: []*config.API{
							{Path: "google/cloud/zzz/v1"},
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildDotnetConfig(test.apis, &config.Source{Dir: "testgoogleapis"})
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunDotnetMigration(t *testing.T) {
	fetchSource = func(ctx context.Context) (*config.Source, error) {
		return &config.Source{
			Commit: "abcd123",
			SHA256: "sha123",
			Dir:    "../../internal/testdata/googleapis",
		}, nil
	}
	for _, test := range []struct {
		name     string
		repoPath string
		wantErr  error
	}{
		{
			name:     "success",
			repoPath: "testdata/run/success-dotnet",
		},
		{
			name:     "tidy_failed",
			repoPath: "testdata/run/tidy-fails-dotnet",
			wantErr:  errTidyFailed,
		},
		{
			name:     "missing_file",
			repoPath: "testdata/run/non-existent",
			wantErr:  os.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outputPath := "librarian.yaml"
			t.Cleanup(func() {
				if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
					t.Fatalf("cleanup: %v", err)
				}
			})
			err := runDotnetMigration(t.Context(), test.repoPath)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("expected error %v, got %v", test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestReadApisJSON(t *testing.T) {
	for _, test := range []struct {
		name     string
		repoPath string
		wantErr  bool
	}{
		{
			name:     "valid file",
			repoPath: "testdata/run/success-dotnet",
		},
		{
			name:     "missing file",
			repoPath: "testdata/run/non-existent",
			wantErr:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := readApisJSON(test.repoPath)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(got.APIs) == 0 {
				t.Error("expected at least one API entry")
			}
		})
	}
}
