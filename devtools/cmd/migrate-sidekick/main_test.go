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

package migrate_sidekick

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestRun(t *testing.T) {
	if err := run(); err == nil || !errors.Is(err, errRepoNotFound) {
		t.Error(err)
	}
}

func TestReadRootSidekick(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		path    string
		want    *RootDefaults
		wantErr error
	}{
		{
			name: "success",
			path: "testdata/root-sidekick/success",
			want: &RootDefaults{
				DisabledRustdocWarnings: []string{
					"redundant_explicit_links",
					"broken_intra_doc_links",
				},
				PackageDependencies: []*config.RustPackageDependency{
					{
						Feature: "_internal-http-client",
						Name:    "gaxi",
						Package: "google-cloud-gax-internal",
						Source:  "internal",
						UsedIf:  "services",
					},
					{
						Name:      "lazy_static",
						Package:   "lazy_static",
						UsedIf:    "services",
						ForceUsed: true,
					},
				},
				Remote: "upstream",
				Branch: "main",
			},
		},
		{
			name:    "no_sidekick_file",
			path:    "testdata/root-sidekick/no_sidekick_file",
			wantErr: errSidekickNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := readRootSidekick(test.path)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("got error %v, want %v", err, test.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("got error %v, want nil", err)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindSidekickFiles(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		path    string
		want    []string
		wantErr error
	}{
		{
			name: "found_sidekick_files",
			path: "testdata/find-sidekick-files/success",
			want: []string{
				"testdata/find-sidekick-files/success/src/generated/sub-1/.sidekick.toml",
				"testdata/find-sidekick-files/success/src/generated/sub-1/subsub-1/.sidekick.toml",
			},
		},
		{
			name:    "no_src_directory",
			path:    "testdata/find-sidekick-files/no-src",
			wantErr: errSrcNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := findSidekickFiles(test.path)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("got error %v, want %v", err, test.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("got error %v, want nil", err)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReadSidekickFiles(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		files   []string
		want    map[string]*config.Library
		wantErr error
	}{
		{
			name: "read_sidekick_files",
			files: []string{
				"testdata/read-sidekick-files/success-read/.sidekick.toml",
				"testdata/read-sidekick-files/success-read/nested/.sidekick.toml",
			},
			want: map[string]*config.Library{
				"google-cloud-security-publicca-v1": {
					Name: "google-cloud-security-publicca-v1",
					Channels: []*config.Channel{
						{
							Path:          "google/cloud/security/publicca/v1",
							ServiceConfig: "google/cloud/security/publicca/v1/publicca_v1.yaml",
						},
					},
					Version:       "1.1.0",
					CopyrightYear: "2025",
					Rust: &config.RustCrate{
						RustDefault: config.RustDefault{
							DisabledRustdocWarnings: []string{"bare_urls", "broken_intra_doc_links", "redundant_explicit_links"},
						},
						PerServiceFeatures:        true,
						ModulePath:                "crate",
						TemplateOverride:          "templates/mod",
						TitleOverride:             "Google Apps Script Types",
						DescriptionOverride:       "Description override",
						PackageNameOverride:       "google-cloud-security-publicca-v1",
						RootName:                  "conformance-root",
						Roots:                     []string{"discovery", "googleapis"},
						DefaultFeatures:           []string{"instances", "projects"},
						ExtraModules:              []string{"errors", "operation"},
						IncludeList:               []string{"api.proto", "source_context.proto", "type.proto", "descriptor.proto"},
						IncludedIds:               []string{".google.iam.v2.Resource"},
						SkippedIds:                []string{".google.iam.v1.ResourcePolicyMember"},
						DisabledClippyWarnings:    []string{"doc_lazy_continuation"},
						HasVeneer:                 true,
						RoutingRequired:           true,
						IncludeGrpcOnlyMethods:    true,
						GenerateSetterSamples:     true,
						PostProcessProtos:         "example post processing",
						DetailedTracingAttributes: true,
						NameOverrides:             ".google.cloud.security/publicca.v1.Storage=StorageControl",
					},
				},
				"google-cloud-sql-v1": {
					Name: "google-cloud-sql-v1",
					Channels: []*config.Channel{
						{
							Path:          "google/cloud/sql/v1",
							ServiceConfig: "google/cloud/sql/v1/sqladmin_v1.yaml",
						},
					},
					SkipPublish:   true,
					Version:       "1.2.0",
					CopyrightYear: "2025",
					Rust: &config.RustCrate{
						RustDefault: config.RustDefault{
							PackageDependencies: []*config.RustPackageDependency{
								{
									Feature: "_internal-http-client",
									Name:    "gaxi",
									Package: "google-cloud-gax-internal",
									Source:  "internal",
									UsedIf:  "services",
								},
								{
									ForceUsed: true,
									Name:      "lazy_static",
									Package:   "lazy_static",
									UsedIf:    "services",
								},
							},
						},
						DocumentationOverrides: []config.RustDocumentationOverride{
							{
								ID:      ".google.api.ProjectProperties",
								Match:   "example match",
								Replace: "example replace",
							},
						},
						PaginationOverrides: []config.RustPaginationOverride{
							{
								ID:        ".google.cloud.sql.v1.SqlInstancesService.List",
								ItemField: "items",
							},
						},
					},
				},
			},
		},
		{
			name: "no_api_path",
			files: []string{
				"testdata/read-sidekick-files/no-api-path/.sidekick.toml",
			},
			want: map[string]*config.Library{},
		},
		{
			name: "no_package_name",
			files: []string{
				"testdata/read-sidekick-files/no-package-name/.sidekick.toml",
			},
			want: map[string]*config.Library{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := readSidekickFiles(test.files)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("got error %v, want %v", err, test.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("got error %v, want nil", err)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeriveLibraryName(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		api  string
		want string
	}{
		{
			name: "strip_google_prefix",
			api:  "google/cloud/secretmanager/v1",
			want: "google-cloud-secretmanager-v1",
		},
		{
			name: "strip_devtools_prefix",
			api:  "google/devtools/artifactregistry/v1",
			want: "google-cloud-artifactregistry-v1",
		},
		{
			name: "strip_api_prefix",
			api:  "google/api/apikeys/v1",
			want: "google-cloud-apikeys-v1",
		},
		{
			name: "do_not_strip_api_prefix",
			api:  "google/api/servicecontrol/v1",
			want: "google-cloud-api-servicecontrol-v1",
		},
		{
			name: "no_google_prefix",
			api:  "grafeas/v1",
			want: "google-cloud-grafeas-v1",
		},
		{
			name: "no_cloud_prefix",
			api:  "spanner/admin/instances/v1",
			want: "google-cloud-spanner-admin-instances-v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := deriveLibraryName(test.api)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildConfig(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name         string
		libraries    map[string]*config.Library
		googleapis   string
		rootDefaults *RootDefaults
		want         *config.Config
		wantErr      error
	}{
		{
			name: "rust_defaults",
			rootDefaults: &RootDefaults{
				DisabledRustdocWarnings: []string{"bare_urls", "broken_intra_doc_links", "redundant_explicit_links"},
				PackageDependencies: []*config.RustPackageDependency{
					{
						Feature: "_internal-http-client",
						Name:    "gaxi",
						Package: "google-cloud-gax-internal",
						Source:  "internal",
						UsedIf:  "services",
					},
					{
						ForceUsed: true,
						Name:      "lazy_static",
						Package:   "lazy_static",
						UsedIf:    "services",
					},
				},
			},
			want: &config.Config{
				Language: "rust",
				Default: &config.Default{
					Output: "src/generated/",
					Rust: &config.RustDefault{
						DisabledRustdocWarnings: []string{"bare_urls", "broken_intra_doc_links", "redundant_explicit_links"},
						PackageDependencies: []*config.RustPackageDependency{
							{
								Feature: "_internal-http-client",
								Name:    "gaxi",
								Package: "google-cloud-gax-internal",
								Source:  "internal",
								UsedIf:  "services",
							},
							{
								ForceUsed: true,
								Name:      "lazy_static",
								Package:   "lazy_static",
								UsedIf:    "services",
							},
						},
					},
				},
			},
		},
		{
			name:         "copy_libraries",
			rootDefaults: &RootDefaults{},
			libraries: map[string]*config.Library{
				"google-cloud-security-publicca-v1": {
					Name: "google-cloud-security-publicca-v1",
					Channels: []*config.Channel{
						{
							Path:          "google/cloud/security/publicca/v1",
							ServiceConfig: "google/cloud/security/publicca/v1/publicca_v1.yaml",
						},
					},
					Version:       "1.1.0",
					CopyrightYear: "2025",
					Rust: &config.RustCrate{
						RustDefault: config.RustDefault{
							DisabledRustdocWarnings: []string{"bare_urls", "broken_intra_doc_links", "redundant_explicit_links"},
						},
						PerServiceFeatures:    true,
						GenerateSetterSamples: true,
						NameOverrides:         ".google.cloud.security/publicca.v1.Storage=StorageControl",
					},
				},
				"skipped": {
					Name: "google-cloud-sql-v1",
					Channels: []*config.Channel{
						{
							Path:          "google/cloud/sql/v1",
							ServiceConfig: "google/cloud/sql/v1/sqladmin_v1.yaml",
						},
					},
					SkipPublish: true,
					Version:     "1.2.0",
				},
			},
			want: &config.Config{
				Language: "rust",
				Default:  &config.Default{Output: "src/generated/", Rust: &config.RustDefault{}},
				Libraries: []*config.Library{
					{
						Name: "google-cloud-security-publicca-v1",
						Channels: []*config.Channel{
							{
								Path:          "google/cloud/security/publicca/v1",
								ServiceConfig: "google/cloud/security/publicca/v1/publicca_v1.yaml",
							},
						},
						Version:       "1.1.0",
						CopyrightYear: "2025",
						Rust: &config.RustCrate{
							RustDefault: config.RustDefault{
								DisabledRustdocWarnings: []string{"bare_urls", "broken_intra_doc_links", "redundant_explicit_links"},
							},
							PerServiceFeatures:    true,
							GenerateSetterSamples: true,
							NameOverrides:         ".google.cloud.security/publicca.v1.Storage=StorageControl",
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := buildConfig(test.libraries, test.googleapis, test.rootDefaults)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetGitCommit(t *testing.T) {
	t.Parallel()
	gitDir = "test-git"
	for _, test := range []struct {
		name    string
		path    string
		want    string
		wantErr error
	}{
		{
			name: "direct_commit_sha",
			path: "testdata/get-git-commit/direct-commit-sha",
			want: "1234567abcdefg",
		},
		{
			name: "branch_sha",
			path: "testdata/get-git-commit/branch",
			want: "95bdc62f7448ffb183aada62de1be5a704e54a8c",
		},
		{
			name:    "no_head",
			path:    "testdata/get-git-commit/no-head",
			wantErr: errHeadNotFound,
		},
		{
			name:    "branch_not_found",
			path:    "testdata/get-git-commit/branch-not-found",
			wantErr: errBranchNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := getGitCommit(test.path)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("got error %v, want %v", err, test.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("got error %v, want nil", err)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
