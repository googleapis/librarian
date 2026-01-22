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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"cloud.google.com/go/sdk/internal/config"
	"cloud.google.com/go/sdk/internal/yaml"
	"github.com/google/go-cmp/cmp"
)

func TestReadRootSidekick(t *testing.T) {
	for _, test := range []struct {
		name    string
		path    string
		want    *config.Config
		wantErr error
	}{
		{
			name: "success",
			path: "testdata/root-sidekick/success",
			want: &config.Config{
				Language: "rust",
				Sources: &config.Sources{
					Discovery: &config.Source{
						Commit: "0bb1100f52bf0bae06f4b4d76742e7eba5c59793",
						SHA256: "67c8d3792f0ebf5f0582dce675c379d0f486604eb0143814c79e788954aa1212",
					},
					Googleapis: &config.Source{
						Commit: "fe58211356a91f4140ed51893703910db05ade91",
						SHA256: "839e897c39cada559b97d64f90378715a4a43fbc972d8cf93296db4156662085",
					},
					Showcase: &config.Source{
						Commit: "69bdd62035d793f3d23a0c960dee547023c1c5ac",
						SHA256: "96491310ba1b5c0c71738d3d80327a95196c1b6ac16f033e3fa440870efbbf5c",
					},
					ProtobufSrc: &config.Source{
						Commit:  "b407e8416e3893036aee5af9a12bd9b6a0e2b2e6",
						SHA256:  "55912546338433f465a552e9ef09930c63b9eb697053937416890cff83a8622d",
						Subpath: "src",
					},
					Conformance: &config.Source{
						Commit: "b407e8416e3893036aee5af9a12bd9b6a0e2b2e6",
						SHA256: "55912546338433f465a552e9ef09930c63b9eb697053937416890cff83a8622d",
					},
				},
				Default: &config.Default{
					Output:       "src/generated/",
					ReleaseLevel: "stable",
					Rust: &config.RustDefault{
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
						GenerateSetterSamples: "true",
					},
				},
				Release: &config.Release{
					Remote:         "upstream",
					Branch:         "main",
					IgnoredChanges: []string{".repo-metadata.json", ".sidekick.toml"},
				},
			},
		},
		{
			name:    "no_sidekick_file",
			path:    "testdata/root-sidekick/no_sidekick_file",
			wantErr: errSidekickNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
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
			wantErr: os.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
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

func TestBuildGAPIC(t *testing.T) {
	for _, test := range []struct {
		name     string
		files    []string
		repoName string
		want     map[string]*config.Library
		wantErr  error
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
					APIs: []*config.API{
						{
							Path: "google/cloud/security/publicca/v1",
						},
					},
					Version:       "1.1.0",
					CopyrightYear: "2025",
					Keep: []string{
						"src/errors.rs",
						"src/operation.rs",
					},
					DescriptionOverride: "Description override",
					SpecificationFormat: "discovery",
					Output:              "testdata/read-sidekick-files/success-read/nested",
					Rust: &config.RustCrate{
						RustDefault: config.RustDefault{
							DisabledRustdocWarnings: []string{"bare_urls", "broken_intra_doc_links", "redundant_explicit_links"},
							GenerateSetterSamples:   "true",
							GenerateRpcSamples:      "true",
						},
						PerServiceFeatures:        true,
						ModulePath:                "crate",
						TemplateOverride:          "templates/mod",
						PackageNameOverride:       "google-cloud-security-publicca-v1",
						RootName:                  "conformance-root",
						Roots:                     []string{"discovery", "googleapis"},
						DefaultFeatures:           []string{"instances", "projects"},
						IncludeList:               []string{"api.proto", "source_context.proto", "type.proto", "descriptor.proto"},
						IncludedIds:               []string{".google.iam.v2.Resource"},
						SkippedIds:                []string{".google.iam.v1.ResourcePolicyMember"},
						DisabledClippyWarnings:    []string{"doc_lazy_continuation"},
						HasVeneer:                 true,
						RoutingRequired:           true,
						IncludeGrpcOnlyMethods:    true,
						PostProcessProtos:         "example post processing",
						DetailedTracingAttributes: true,
						NameOverrides:             ".google.cloud.security/publicca.v1.Storage=StorageControl",
					},
				},
				"google-cloud-sql-v1": {
					Name: "google-cloud-sql-v1",
					APIs: []*config.API{
						{
							Path: "google/cloud/sql/v1",
						},
					},
					SkipPublish:         true,
					Version:             "1.2.0",
					CopyrightYear:       "2025",
					SpecificationFormat: "openapi",
					Output:              "testdata/read-sidekick-files/success-read",
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
									Ignore:    true,
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
			name: "unable_to_calculate_output_path",
			files: []string{
				"testdata/read-sidekick-files/success-read/.sidekick.toml",
			},
			repoName: "/invalid/repo/path",
			wantErr:  errUnableToCalculateOutputPath,
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
		{
			name: "with_discovery",
			files: []string{
				"testdata/read-sidekick-files/discovery/.sidekick.toml",
			},
			want: map[string]*config.Library{
				"google-cloud-compute-v1": {
					Name: "google-cloud-compute-v1",
					APIs: []*config.API{
						{
							Path: "google/cloud/compute/v1",
						},
					},
					Version:             "0.1.0",
					SpecificationFormat: "discovery",
					Output:              "testdata/read-sidekick-files/discovery",
					Rust: &config.RustCrate{
						Discovery: &config.RustDiscovery{
							OperationID: ".google.cloud.compute.v1.Operation",
							Pollers: []config.RustPoller{
								{
									Prefix:   "compute/v1/projects/{project}/global/operations",
									MethodID: ".google.cloud.compute.v1.globalOperations.get",
								},
								{
									Prefix:   "compute/v1/projects/{project}/regions/{region}/operations",
									MethodID: ".google.cloud.compute.v1.regionOperations.get",
								},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildGAPIC(test.files, test.repoName)
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
			got := deriveLibraryName(test.api)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindCargos(t *testing.T) {
	for _, test := range []struct {
		name    string
		path    string
		want    []string
		wantErr error
	}{
		{
			name: "success",
			path: "testdata/find-cargos/success",
			want: []string{
				"testdata/find-cargos/success/Cargo.toml",
				"testdata/find-cargos/success/dir-1/Cargo.toml",
				"testdata/find-cargos/success/dir-2/dirdir-2/Cargo.toml",
			},
		},
		{
			name:    "invalid_path",
			path:    "testdata/find-cargos/non-existent-path",
			wantErr: os.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := findCargos(test.path)
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

func TestBuildVeneer(t *testing.T) {
	for _, test := range []struct {
		name     string
		files    []string
		repoPath string
		want     map[string]*config.Library
		wantErr  error
	}{
		{
			name: "success",
			files: []string{
				"testdata/build-veneer/success/lib-1/Cargo.toml",
				"testdata/build-veneer/success/lib-2/Cargo.toml",
			},
			repoPath: "testdata/build-veneer/success",
			want: map[string]*config.Library{
				"google-cloud-storage": {
					Name:          "google-cloud-storage",
					Veneer:        true,
					Output:        "lib-1",
					Version:       "1.5.0",
					CopyrightYear: "2025",
					Rust: &config.RustCrate{
						Modules: []*config.RustModule{
							{
								DisabledRustdocWarnings: []string{},
								ModuleRoots:             nil,
								HasVeneer:               true,
								IncludedIds: []string{
									".google.storage.v2.Storage.DeleteBucket",
									".google.storage.v2.Storage.GetBucket",
									".google.storage.v2.Storage.CreateBucket",
									".google.storage.v2.Storage.ListBuckets",
								},
								IncludeGrpcOnlyMethods: true,
								NameOverrides:          ".google.storage.v2.Storage=StorageControl",
								Output:                 "lib-1/dir-1",
								RoutingRequired:        true,
								ServiceConfig:          "google/storage/v2/storage_v2.yaml",
								SkippedIds:             []string{".google.iam.v1.ResourcePolicyMember"},
								Source:                 "google/storage/v2",
								Template:               "grpc-client",
							},
							{
								GenerateSetterSamples: "false",
								ModulePath:            "crate::generated::gapic_control::model",
								ModuleRoots: map[string]string{
									"project-root": ".",
								},
								NameOverrides: ".google.storage.control.v2.IntelligenceConfig.Filter.cloud_storage_buckets=CloudStorageBucketsOneOf",
								Output:        "lib-1/dir-2/dirdir-2",
								Source:        "google/storage/control/v2",
								Template:      "convert-prost",
							},
						},
					},
				},
				"google-cloud-spanner": {
					Name:          "google-cloud-spanner",
					Veneer:        true,
					Output:        "lib-2",
					CopyrightYear: "2025",
					SkipPublish:   true,
				},
			},
		},
		{
			name: "with_overrides",
			files: []string{
				"testdata/build-veneer/with-overrides/lib-1/Cargo.toml",
			},
			repoPath: "testdata/build-veneer/with-overrides",
			want: map[string]*config.Library{
				"google-cloud-storage-overridden": {
					Name:          "google-cloud-storage-overridden",
					Veneer:        true,
					Output:        "lib-1",
					Version:       "1.5.0",
					CopyrightYear: "2025",
					Rust: &config.RustCrate{
						Modules: []*config.RustModule{
							{
								DocumentationOverrides: []config.RustDocumentationOverride{
									{
										ID:      ".google.storage.v2.Storage",
										Match:   "The service helps to manage cloud storage.",
										Replace: "The service helps to manage cloud storage resources.",
									},
								},
								HasVeneer:              true,
								IncludeGrpcOnlyMethods: true,
								NameOverrides:          ".google.storage.v2.Storage=StorageControl",
								Output:                 "lib-1/dir-1",
								RoutingRequired:        true,
								ServiceConfig:          "google/storage/v2/storage_v2.yaml",
								SkippedIds:             []string{".google.iam.v1.ResourcePolicyMember"},
								Source:                 "google/storage/v2",
								ModuleRoots: map[string]string{
									"discovery-root":  "",
									"googleapis-root": "",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "no_rust_modules",
			files: []string{
				"testdata/build-veneer/success/lib-2/Cargo.toml",
			},
			repoPath: "testdata/build-veneer/success",
			want: map[string]*config.Library{
				"google-cloud-spanner": {
					Name:          "google-cloud-spanner",
					Veneer:        true,
					Output:        "lib-2",
					CopyrightYear: "2025",
					SkipPublish:   true,
				},
			},
		},
		{
			name: "google_cloud-wkt",
			files: []string{
				"testdata/build-veneer/wkt/Cargo.toml",
				"testdata/build-veneer/wkt/tests/common/Cargo.toml",
			},
			repoPath: "testdata/build-veneer/wkt",
			want: map[string]*config.Library{
				"common": {
					Name:          "common",
					Veneer:        true,
					Output:        "tests/common",
					CopyrightYear: "2025",
					SkipPublish:   true,
					Rust: &config.RustCrate{
						Modules: []*config.RustModule{
							{
								DisabledRustdocWarnings: []string{},
								ModulePath:              "crate::generated",
								ModuleRoots: map[string]string{
									"project-root": ".",
								},
								Output:                "tests/common/src/generated",
								Source:                "src/wkt/tests/protos",
								Template:              "mod",
								GenerateSetterSamples: "false",
							},
						},
					},
				},
				"google-cloud-wkt": {
					Name:          "google-cloud-wkt",
					Veneer:        true,
					Output:        ".",
					Version:       "1.2.0",
					CopyrightYear: "2025",
					Rust: &config.RustCrate{
						Modules: []*config.RustModule{
							{
								IncludeList: "api.proto,source_context.proto,type.proto,descriptor.proto",
								ModulePath:  "crate",
								Output:      "src/generated",
								Source:      "google/protobuf",
								Template:    "mod",
							},
						},
					},
				},
			},
		},
		{
			name: "excluded_library",
			files: []string{
				"testdata/build-veneer/success/lib-1/Cargo.toml",
				"testdata/build-veneer/success/echo-server/Cargo.toml",
			},
			repoPath: "testdata/build-veneer/success",
			want: map[string]*config.Library{
				"google-cloud-storage": {
					Name:          "google-cloud-storage",
					Veneer:        true,
					Output:        "lib-1",
					Version:       "1.5.0",
					CopyrightYear: "2025",
					Rust: &config.RustCrate{
						Modules: []*config.RustModule{
							{
								DisabledRustdocWarnings: []string{},
								ModuleRoots:             nil,
								HasVeneer:               true,
								IncludedIds: []string{
									".google.storage.v2.Storage.DeleteBucket",
									".google.storage.v2.Storage.GetBucket",
									".google.storage.v2.Storage.CreateBucket",
									".google.storage.v2.Storage.ListBuckets",
								},
								IncludeGrpcOnlyMethods: true,
								NameOverrides:          ".google.storage.v2.Storage=StorageControl",
								Output:                 "lib-1/dir-1",
								RoutingRequired:        true,
								ServiceConfig:          "google/storage/v2/storage_v2.yaml",
								SkippedIds:             []string{".google.iam.v1.ResourcePolicyMember"},
								Source:                 "google/storage/v2",
								Template:               "grpc-client",
							},
							{
								GenerateSetterSamples: "false",
								ModulePath:            "crate::generated::gapic_control::model",
								ModuleRoots: map[string]string{
									"project-root": ".",
								},
								NameOverrides: ".google.storage.control.v2.IntelligenceConfig.Filter.cloud_storage_buckets=CloudStorageBucketsOneOf",
								Output:        "lib-1/dir-2/dirdir-2",
								Source:        "google/storage/control/v2",
								Template:      "convert-prost",
							},
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildVeneer(test.files, test.repoPath)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("got error %v, want %v", err, test.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("got error %v, want nil", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildConfig(t *testing.T) {
	for _, test := range []struct {
		name      string
		libraries map[string]*config.Library
		defaults  *config.Config
		want      *config.Config
		wantErr   error
	}{
		{
			name: "rust_defaults",
			defaults: &config.Config{
				Default: &config.Default{
					Output: "src/generated/",
					Rust: &config.RustDefault{
						DisabledRustdocWarnings: []string{"bare_urls", "broken_intra_doc_links", "redundant_explicit_links"},
					},
				},
			},
			want: &config.Config{
				Default: &config.Default{
					Output: "src/generated/",
					Rust: &config.RustDefault{
						DisabledRustdocWarnings: []string{"bare_urls", "broken_intra_doc_links", "redundant_explicit_links"},
					},
				},
			},
		},
		{
			name:     "copy_libraries",
			defaults: &config.Config{},
			libraries: map[string]*config.Library{
				"google-cloud-security-publicca-v1": {
					Name: "google-cloud-security-publicca-v1",
					APIs: []*config.API{
						{
							Path: "google/cloud/security/publicca/v1",
						},
					},
					Version:       "1.1.0",
					CopyrightYear: "2025",
					Rust: &config.RustCrate{
						RustDefault: config.RustDefault{
							DisabledRustdocWarnings: []string{"bare_urls", "broken_intra_doc_links", "redundant_explicit_links"},
							GenerateSetterSamples:   "true",
							GenerateRpcSamples:      "true",
						},
						PerServiceFeatures: true,
						NameOverrides:      ".google.cloud.security/publicca.v1.Storage=StorageControl",
					},
				},
				"skipped": {
					Name: "google-cloud-sql-v1",
					APIs: []*config.API{
						{
							Path: "google/cloud/sql/v1",
						},
					},
					SkipPublish: true,
					Version:     "1.2.0",
				},
			},
			want: &config.Config{
				Libraries: []*config.Library{
					{
						Name: "google-cloud-security-publicca-v1",
						APIs: []*config.API{
							{
								Path: "google/cloud/security/publicca/v1",
							},
						},
						Version:       "1.1.0",
						CopyrightYear: "2025",
						Rust: &config.RustCrate{
							RustDefault: config.RustDefault{
								DisabledRustdocWarnings: []string{"bare_urls", "broken_intra_doc_links", "redundant_explicit_links"},
								GenerateSetterSamples:   "true",
								GenerateRpcSamples:      "true",
							},
							PerServiceFeatures: true,
							NameOverrides:      ".google.cloud.security/publicca.v1.Storage=StorageControl",
						},
					},
				},
			},
		},
		{
			name:     "service does not exist",
			defaults: &config.Config{},
			libraries: map[string]*config.Library{
				"google-cloud-orgpolicy-v1": {
					Name: "google-cloud-orgpolicy-v1",
					APIs: []*config.API{
						{
							Path: "google/cloud/orgpolicy/v1",
						},
					},
					Version:       "1.1.0",
					CopyrightYear: "2025",
					Rust: &config.RustCrate{
						RustDefault: config.RustDefault{
							DisabledRustdocWarnings: []string{"bare_urls", "broken_intra_doc_links", "redundant_explicit_links"},
							GenerateSetterSamples:   "true",
							GenerateRpcSamples:      "true",
						},
						PerServiceFeatures: true,
						NameOverrides:      ".google.cloud.orgpolicy.v1.OrgPolicy=OrgPolicyControl",
					},
				},
			},
			want: &config.Config{
				Libraries: []*config.Library{
					{
						Name: "google-cloud-orgpolicy-v1",
						APIs: []*config.API{
							{
								Path: "google/cloud/orgpolicy/v1",
							},
						},
						Version:       "1.1.0",
						CopyrightYear: "2025",
						Rust: &config.RustCrate{
							RustDefault: config.RustDefault{
								DisabledRustdocWarnings: []string{"bare_urls", "broken_intra_doc_links", "redundant_explicit_links"},
								GenerateSetterSamples:   "true",
								GenerateRpcSamples:      "true",
							},
							PerServiceFeatures: true,
							NameOverrides:      ".google.cloud.orgpolicy.v1.OrgPolicy=OrgPolicyControl",
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildConfig(test.libraries, test.defaults)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunMigrateCommand(t *testing.T) {
	for _, test := range []struct {
		name                         string
		path                         string
		wantErr                      error
		checkDocumentOverrideReplace []string
		checkDocumentOverrideMatch   []string
	}{
		{
			name:                         "success",
			path:                         "testdata/run/success",
			checkDocumentOverrideMatch:   []string{"example match", "Ancestry subtrees must be in one of the following formats:"},
			checkDocumentOverrideReplace: []string{"example replace", " \nAncestry subtrees must be in one of the following formats:"},
		},
		{
			name:    "tidy_command_fails",
			path:    "testdata/run/tidy-fails",
			wantErr: errTidyFailed,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// ensure librarian.yaml generated is removed after the test,
			// even if the test fails
			outputPath := "librarian.yaml"
			t.Cleanup(func() {
				if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
					t.Logf("cleanup: remove %s: %v", outputPath, err)
				}
			})
			abs, err := filepath.Abs(test.path)
			if err != nil {
				t.Fatal(err)
			}
			if err := runSidekickMigration(t.Context(), abs); err != nil {
				if test.wantErr == nil {
					t.Fatal(err)
				}
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("expected error containing %q, got: %v", test.wantErr, err)
				}
			} else if test.wantErr != nil {
				t.Fatalf("expected error containing %q, got nil", test.wantErr)
			} else {
				data, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("librarian file does not exist")
				}

				librarianConfig, err := yaml.Unmarshal[config.Config](data)
				if err != nil {
					t.Fatalf("unable to parse librarian file")
				}
				if len(librarianConfig.Libraries) != 1 {
					t.Fatalf("librarian yaml does not contain library")
				}
				if len(test.checkDocumentOverrideReplace) > 0 {
					for index, expected := range test.checkDocumentOverrideReplace {
						got := librarianConfig.Libraries[0].Rust.DocumentationOverrides[index].Replace
						if got != expected {
							t.Fatalf("expected checkDocumentOverrideValue: %s got: %s", expected, got)
						}
						gotMatch := librarianConfig.Libraries[0].Rust.DocumentationOverrides[index].Match
						if expected := test.checkDocumentOverrideMatch[index]; gotMatch != expected {
							t.Fatalf("expected checkDocumentOverrideMatch: %s got: %s", expected, gotMatch)
						}

					}
				}
			}

		})
	}
}
