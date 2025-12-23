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

package rust

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
)

func TestToSidekickConfig(t *testing.T) {
	for _, tt := range []struct {
		name           string
		library        *config.Library
		channel        *config.Channel
		googleapisDir  string
		discoveryDir   string
		protobufDir    string
		conformanceDir string
		showcaseDir    string
		protobufSubDir string
		want           *sidekickconfig.Config
	}{
		{
			name: "minimal config",
			library: &config.Library{
				Name: "google-cloud-storage",
			},
			channel: &config.Channel{
				Path:          "google/cloud/storage/v1",
				ServiceConfig: "google/cloud/storage/v1/storage_v1.yaml",
			},
			googleapisDir: "/tmp/googleapis",
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					ServiceConfig:       "google/cloud/storage/v1/storage_v1.yaml",
					SpecificationSource: "google/cloud/storage/v1",
				},
				Source: map[string]string{
					"googleapis-root": "/tmp/googleapis",
					"roots":           "googleapis",
				},
				Codec: map[string]string{
					"package-name-override": "google-cloud-storage",
				},
			},
		},
		{
			name: "with version and release level",
			library: &config.Library{
				Name:         "google-cloud-storage",
				Version:      "0.1.0",
				ReleaseLevel: "preview",
				Roots:        []string{"googleapis"},
			},
			channel: &config.Channel{
				Path:          "google/cloud/storage/v1",
				ServiceConfig: "google/cloud/storage/v1/storage_v1.yaml",
			},
			googleapisDir: "/tmp/googleapis",
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					ServiceConfig:       "google/cloud/storage/v1/storage_v1.yaml",
					SpecificationSource: "google/cloud/storage/v1",
				},
				Source: map[string]string{
					"googleapis-root": "/tmp/googleapis",
					"roots":           "googleapis",
				},
				Codec: map[string]string{
					"version":               "0.1.0",
					"release-level":         "preview",
					"package-name-override": "google-cloud-storage",
				},
			},
		},
		{
			name: "with copyright year",
			library: &config.Library{
				Name:          "google-cloud-storage",
				CopyrightYear: "2024",
				Roots:         []string{"googleapis"},
			},
			channel: &config.Channel{
				Path:          "google/cloud/storage/v1",
				ServiceConfig: "google/cloud/storage/v1/storage_v1.yaml",
			},
			googleapisDir: "/tmp/googleapis",
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					ServiceConfig:       "google/cloud/storage/v1/storage_v1.yaml",
					SpecificationSource: "google/cloud/storage/v1",
				},
				Source: map[string]string{
					"googleapis-root": "/tmp/googleapis",
					"roots":           "googleapis",
				},
				Codec: map[string]string{
					"copyright-year":        "2024",
					"package-name-override": "google-cloud-storage",
				},
			},
		},
		{
			name: "with rust config",
			library: &config.Library{
				Name: "google-cloud-storage",
				Keep: []string{"src/extra-module.rs"},
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
						GenerateSetterSamples:   "true",
					},
					ModulePath:                "gcs",
					PerServiceFeatures:        true,
					IncludeGrpcOnlyMethods:    true,
					DetailedTracingAttributes: true,
					HasVeneer:                 true,
					RoutingRequired:           true,
					GenerateRpcSamples:        true,
					DisabledClippyWarnings:    []string{"too_many_arguments"},
					DefaultFeatures:           []string{"default-feature"},
					TemplateOverride:          "custom-template",
				},
			},
			channel: &config.Channel{
				Path:          "google/cloud/storage/v1",
				ServiceConfig: "google/cloud/storage/v1/storage_v1.yaml",
			},
			googleapisDir: "/tmp/googleapis",
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					ServiceConfig:       "google/cloud/storage/v1/storage_v1.yaml",
					SpecificationSource: "google/cloud/storage/v1",
				},
				Source: map[string]string{
					"googleapis-root": "/tmp/googleapis",
					"roots":           "googleapis",
				},
				Codec: map[string]string{
					"module-path":                 "gcs",
					"per-service-features":        "true",
					"include-grpc-only-methods":   "true",
					"detailed-tracing-attributes": "true",
					"has-veneer":                  "true",
					"routing-required":            "true",
					"generate-setter-samples":     "true",
					"generate-rpc-samples":        "true",
					"disabled-rustdoc-warnings":   "broken_intra_doc_links",
					"disabled-clippy-warnings":    "too_many_arguments",
					"default-features":            "default-feature",
					"extra-modules":               "extra-module",
					"template-override":           "custom-template",
					"package-name-override":       "google-cloud-storage",
				},
			},
		},
		{
			name: "with skip publish (not for publication)",
			library: &config.Library{
				Name:        "google-cloud-storage",
				SkipPublish: true,
				Roots:       []string{"googleapis"},
				Rust:        &config.RustCrate{},
			},
			channel: &config.Channel{
				Path:          "google/cloud/storage/v1",
				ServiceConfig: "google/cloud/storage/v1/storage_v1.yaml",
			},
			googleapisDir: "/tmp/googleapis",
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					ServiceConfig:       "google/cloud/storage/v1/storage_v1.yaml",
					SpecificationSource: "google/cloud/storage/v1",
				},
				Source: map[string]string{
					"googleapis-root": "/tmp/googleapis",
					"roots":           "googleapis",
				},
				Codec: map[string]string{
					"not-for-publication":   "true",
					"package-name-override": "google-cloud-storage",
				},
			},
		},
		{
			name: "with package dependencies",
			library: &config.Library{
				Name: "google-cloud-storage",
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
							{
								Name:      "tokio",
								Package:   "tokio",
								Source:    "1.0",
								ForceUsed: true,
								UsedIf:    "feature = \"async\"",
								Feature:   "async",
							},
						},
					},
				},
			},
			channel: &config.Channel{
				Path:          "google/cloud/storage/v1",
				ServiceConfig: "google/cloud/storage/v1/storage_v1.yaml",
			},
			googleapisDir: "/tmp/googleapis",
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					ServiceConfig:       "google/cloud/storage/v1/storage_v1.yaml",
					SpecificationSource: "google/cloud/storage/v1",
				},
				Source: map[string]string{
					"googleapis-root": "/tmp/googleapis",
					"roots":           "googleapis",
				},
				Codec: map[string]string{
					"package:tokio":         "package=tokio,source=1.0,force-used=true,used-if=feature = \"async\",feature=async",
					"package-name-override": "google-cloud-storage",
				},
			},
		},
		{
			name: "with documentation overrides",
			library: &config.Library{
				Name: "google-cloud-storage",
				Rust: &config.RustCrate{
					DocumentationOverrides: []config.RustDocumentationOverride{
						{
							ID:      ".google.cloud.storage.v1.Bucket.name",
							Match:   "bucket name",
							Replace: "the name of the bucket",
						},
					},
				},
			},
			channel: &config.Channel{
				Path:          "google/cloud/storage/v1",
				ServiceConfig: "google/cloud/storage/v1/storage_v1.yaml",
			},
			googleapisDir:  "/tmp/googleapis",
			discoveryDir:   "",
			protobufDir:    "",
			conformanceDir: "",
			showcaseDir:    "",
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					ServiceConfig:       "google/cloud/storage/v1/storage_v1.yaml",
					SpecificationSource: "google/cloud/storage/v1",
				},
				Source: map[string]string{
					"googleapis-root": "/tmp/googleapis",
					"roots":           "googleapis",
				},
				Codec: map[string]string{
					"package-name-override": "google-cloud-storage",
				},
				CommentOverrides: []sidekickconfig.DocumentationOverride{
					{
						ID:      ".google.cloud.storage.v1.Bucket.name",
						Match:   "bucket name",
						Replace: "the name of the bucket",
					},
				},
			},
		},
		{
			name: "with veneer documentation overrides",
			library: &config.Library{
				Name: "google-cloud-storage",
				Rust: &config.RustCrate{
					Modules: []*config.RustModule{
						{
							DocumentationOverrides: []config.RustDocumentationOverride{
								{
									ID:      ".google.cloud.storage.v1.Bucket.name",
									Match:   "bucket name",
									Replace: "the name of the bucket",
								},
							},
						},
						{
							DocumentationOverrides: []config.RustDocumentationOverride{
								{
									ID:      ".google.cloud.storage.v1.Bucket.id",
									Match:   "bucket id",
									Replace: "the id of the bucket",
								},
							},
						},
					},
				},
			},
			channel: &config.Channel{
				Path:          "google/cloud/storage/v1",
				ServiceConfig: "google/cloud/storage/v1/storage_v1.yaml",
			},
			googleapisDir:  "/tmp/googleapis",
			discoveryDir:   "",
			protobufDir:    "",
			conformanceDir: "",
			showcaseDir:    "",
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					ServiceConfig:       "google/cloud/storage/v1/storage_v1.yaml",
					SpecificationSource: "google/cloud/storage/v1",
				},
				Source: map[string]string{
					"googleapis-root": "/tmp/googleapis",
					"roots":           "googleapis",
				},
				Codec: map[string]string{
					"package-name-override": "google-cloud-storage",
				},
				CommentOverrides: []sidekickconfig.DocumentationOverride{
					{
						ID:      ".google.cloud.storage.v1.Bucket.name",
						Match:   "bucket name",
						Replace: "the name of the bucket",
					},
					{
						ID:      ".google.cloud.storage.v1.Bucket.id",
						Match:   "bucket id",
						Replace: "the id of the bucket",
					},
				},
			},
		},
		{
			name: "with pagination overrides",
			library: &config.Library{
				Name: "google-cloud-storage",
				Rust: &config.RustCrate{
					PaginationOverrides: []config.RustPaginationOverride{
						{
							ID:        ".google.cloud.storage.v1.Storage.ListBuckets",
							ItemField: "buckets",
						},
					},
				},
			},
			channel: &config.Channel{
				Path:          "google/cloud/storage/v1",
				ServiceConfig: "google/cloud/storage/v1/storage_v1.yaml",
			},
			googleapisDir:  "/tmp/googleapis",
			discoveryDir:   "",
			protobufDir:    "",
			conformanceDir: "",
			showcaseDir:    "",
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					ServiceConfig:       "google/cloud/storage/v1/storage_v1.yaml",
					SpecificationSource: "google/cloud/storage/v1",
				},
				Source: map[string]string{
					"googleapis-root": "/tmp/googleapis",
					"roots":           "googleapis",
				},
				Codec: map[string]string{
					"package-name-override": "google-cloud-storage",
				},
				PaginationOverrides: []sidekickconfig.PaginationOverride{
					{
						ID:        ".google.cloud.storage.v1.Storage.ListBuckets",
						ItemField: "buckets",
					},
				},
			},
		},
		{
			name: "with discovery format",
			library: &config.Library{
				Name:                "google-cloud-compute-v1",
				SpecificationFormat: "discovery",
				Roots:               []string{"googleapis", "discovery"},
			},
			channel: &config.Channel{
				Path:          "discoveries/compute.v1.json",
				ServiceConfig: "google/cloud/compute/v1/compute_v1.yaml",
			},
			googleapisDir: "/tmp/googleapis",
			discoveryDir:  "/tmp/discovery-artifact-manager",
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "disco",
					ServiceConfig:       "google/cloud/compute/v1/compute_v1.yaml",
					SpecificationSource: "discoveries/compute.v1.json",
				},
				Source: map[string]string{
					"googleapis-root": "/tmp/googleapis",
					"discovery-root":  "/tmp/discovery-artifact-manager",
					"roots":           "googleapis,discovery",
				},
				Codec: map[string]string{
					"package-name-override": "google-cloud-compute-v1",
				},
			},
		},
		{
			name: "with multiple formats",
			library: &config.Library{
				Name:                "google-cloud-compute-v1",
				SpecificationFormat: "discovery",
				Roots:               []string{"googleapis", "discovery", "showcase"},
			},
			channel: &config.Channel{
				Path:          "discoveries/compute.v1.json",
				ServiceConfig: "google/cloud/compute/v1/compute_v1.yaml",
			},
			googleapisDir: "/tmp/googleapis",
			discoveryDir:  "/tmp/discovery-artifact-manager",
			showcaseDir:   "/tmp/showcase",
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "disco",
					ServiceConfig:       "google/cloud/compute/v1/compute_v1.yaml",
					SpecificationSource: "discoveries/compute.v1.json",
				},
				Source: map[string]string{
					"googleapis-root": "/tmp/googleapis",
					"discovery-root":  "/tmp/discovery-artifact-manager",
					"showcase-root":   "/tmp/showcase",
					"roots":           "googleapis,discovery,showcase",
				},
				Codec: map[string]string{
					"package-name-override": "google-cloud-compute-v1",
				},
			},
		},
		{
			name: "with title override",
			library: &config.Library{
				Name: "google-cloud-apps-script-type-gmail",
				Rust: &config.RustCrate{
					TitleOverride: "Google Apps Script Types",
				},
			},
			channel: &config.Channel{
				Path: "google/apps/script/type/gmail",
			},
			googleapisDir: "/tmp/googleapis",
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					SpecificationSource: "google/apps/script/type/gmail",
				},
				Source: map[string]string{
					"googleapis-root": "/tmp/googleapis",
					"title-override":  "Google Apps Script Types",
					"roots":           "googleapis",
				},
				Codec: map[string]string{
					"package-name-override": "google-cloud-apps-script-type-gmail",
				},
			},
		},
		{
			name: "with description override",
			library: &config.Library{
				Name:                "google-cloud-longrunning",
				DescriptionOverride: "Defines types and an abstract service to handle long-running operations.",
			},
			channel: &config.Channel{
				Path:          "google/longrunning",
				ServiceConfig: "google/longrunning/longrunning.yaml",
			},
			googleapisDir: "/tmp/googleapis",
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					ServiceConfig:       "google/longrunning/longrunning.yaml",
					SpecificationSource: "google/longrunning",
				},
				Source: map[string]string{
					"googleapis-root":      "/tmp/googleapis",
					"description-override": "Defines types and an abstract service to handle long-running operations.",
					"roots":                "googleapis",
				},
				Codec: map[string]string{
					"package-name-override": "google-cloud-longrunning",
				},
			},
		},
		{
			name: "with skipped ids",
			library: &config.Library{
				Name: "google-cloud-spanner-admin-database-v1",
				Rust: &config.RustCrate{
					SkippedIds: []string{
						".google.spanner.admin.database.v1.DatabaseAdmin.InternalUpdateGraphOperation",
						".google.spanner.admin.database.v1.InternalUpdateGraphOperationRequest",
						".google.spanner.admin.database.v1.InternalUpdateGraphOperationResponse",
					},
				},
			},
			channel: &config.Channel{
				Path:          "google/spanner/admin/database/v1",
				ServiceConfig: "google/spanner/admin/database/v1/spanner.yaml",
			},
			googleapisDir: "/tmp/googleapis",
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					ServiceConfig:       "google/spanner/admin/database/v1/spanner.yaml",
					SpecificationSource: "google/spanner/admin/database/v1",
				},
				Source: map[string]string{
					"googleapis-root": "/tmp/googleapis",
					"skipped-ids":     ".google.spanner.admin.database.v1.DatabaseAdmin.InternalUpdateGraphOperation,.google.spanner.admin.database.v1.InternalUpdateGraphOperationRequest,.google.spanner.admin.database.v1.InternalUpdateGraphOperationResponse",
					"roots":           "googleapis",
				},
				Codec: map[string]string{
					"package-name-override": "google-cloud-spanner-admin-database-v1",
				},
			},
		},
		{
			name: "with name overrides",
			library: &config.Library{
				Name: "google-cloud-storageinsights-v1",
				Rust: &config.RustCrate{
					NameOverrides: ".google.cloud.storageinsights.v1.DatasetConfig.cloud_storage_buckets=CloudStorageBucketsOneOf,.google.cloud.storageinsights.v1.DatasetConfig.cloud_storage_locations=CloudStorageLocationsOneOf",
				},
			},
			channel: &config.Channel{
				Path:          "google/cloud/storageinsights/v1",
				ServiceConfig: "google/cloud/storageinsights/v1/storageinsights_v1.yaml",
			},
			googleapisDir: "/tmp/googleapis",
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					ServiceConfig:       "google/cloud/storageinsights/v1/storageinsights_v1.yaml",
					SpecificationSource: "google/cloud/storageinsights/v1",
				},
				Source: map[string]string{
					"googleapis-root": "/tmp/googleapis",
					"roots":           "googleapis",
				},
				Codec: map[string]string{
					"package-name-override": "google-cloud-storageinsights-v1",
					"name-overrides":        ".google.cloud.storageinsights.v1.DatasetConfig.cloud_storage_buckets=CloudStorageBucketsOneOf,.google.cloud.storageinsights.v1.DatasetConfig.cloud_storage_locations=CloudStorageLocationsOneOf",
				},
			},
		},
		{
			name: "with discovery LRO polling config",
			library: &config.Library{
				Name:                "google-cloud-compute-v1",
				SpecificationFormat: "discovery",
				Roots:               []string{"googleapis", "discovery"},
				Rust: &config.RustCrate{
					Discovery: &config.RustDiscovery{
						OperationID: ".google.cloud.compute.v1.Operation",
						Pollers: []config.RustPoller{
							{
								Prefix:   "compute/v1/projects/{project}/zones/{zone}",
								MethodID: ".google.cloud.compute.v1.zoneOperations.get",
							},
							{
								Prefix:   "compute/v1/projects/{project}/regions/{region}",
								MethodID: ".google.cloud.compute.v1.regionOperations.get",
							},
							{
								Prefix:   "compute/v1/projects/{project}",
								MethodID: ".google.cloud.compute.v1.globalOperations.get",
							},
						},
					},
				},
			},
			channel: &config.Channel{
				Path:          "discoveries/compute.v1.json",
				ServiceConfig: "google/cloud/compute/v1/compute_v1.yaml",
			},
			googleapisDir: "/tmp/googleapis",
			discoveryDir:  "/tmp/discovery-artifact-manager",
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "disco",
					ServiceConfig:       "google/cloud/compute/v1/compute_v1.yaml",
					SpecificationSource: "discoveries/compute.v1.json",
				},
				Source: map[string]string{
					"googleapis-root": "/tmp/googleapis",
					"discovery-root":  "/tmp/discovery-artifact-manager",
					"roots":           "googleapis,discovery",
				},
				Codec: map[string]string{
					"package-name-override": "google-cloud-compute-v1",
				},
				Discovery: &sidekickconfig.Discovery{
					OperationID: ".google.cloud.compute.v1.Operation",
					Pollers: []*sidekickconfig.Poller{
						{
							Prefix:   "compute/v1/projects/{project}/zones/{zone}",
							MethodID: ".google.cloud.compute.v1.zoneOperations.get",
						},
						{
							Prefix:   "compute/v1/projects/{project}/regions/{region}",
							MethodID: ".google.cloud.compute.v1.regionOperations.get",
						},
						{
							Prefix:   "compute/v1/projects/{project}",
							MethodID: ".google.cloud.compute.v1.globalOperations.get",
						},
					},
				},
			},
		},
		{
			name: "with protobuf and conformance",
			library: &config.Library{
				Name:  "google-cloud-vision-v1",
				Roots: []string{"googleapis", "protobuf-src", "conformance"},
			},
			channel: &config.Channel{
				Path:          "google/cloud/vision/v1",
				ServiceConfig: "google/cloud/vision/v1/vision_v1.yaml",
			},
			googleapisDir:  "/tmp/googleapis",
			protobufDir:    "/tmp/protobuf",
			conformanceDir: "/tmp/conformance",
			protobufSubDir: "src",
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					ServiceConfig:       "google/cloud/vision/v1/vision_v1.yaml",
					SpecificationSource: "google/cloud/vision/v1",
				},
				Source: map[string]string{
					"googleapis-root":   "/tmp/googleapis",
					"protobuf-src-root": "/tmp/protobuf/src",
					"conformance-root":  "/tmp/conformance",
					"roots":             "googleapis,protobuf-src,conformance",
				},
				Codec: map[string]string{
					"package-name-override": "google-cloud-vision-v1",
				},
			},
		},
		{
			name: "with showcase as source",
			library: &config.Library{
				Name:  "google-cloud-showcase",
				Roots: []string{"showcase"},
			},
			channel: &config.Channel{
				Path:          "google/showcase/v1beta1",
				ServiceConfig: "google/showcase/v1beta1/showcase_v1beta1.yaml",
			},
			showcaseDir: "/tmp/gapic-showcase",
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					ServiceConfig:       "google/showcase/v1beta1/showcase_v1beta1.yaml",
					SpecificationSource: "google/showcase/v1beta1",
				},
				Source: map[string]string{
					"showcase-root": "/tmp/gapic-showcase",
					"roots":         "showcase",
				},
				Codec: map[string]string{
					"package-name-override": "google-cloud-showcase",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.library.Rust != nil && tt.library.Rust.Modules != nil {
				var commentOverrides []sidekickconfig.DocumentationOverride
				for _, module := range tt.library.Rust.Modules {
					got := moduleToSidekickConfig(tt.library, module, tt.googleapisDir, tt.protobufDir)
					commentOverrides = append(commentOverrides, got.CommentOverrides...)
				}
				if diff := cmp.Diff(tt.want.CommentOverrides, commentOverrides); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			} else {
				got := toSidekickConfig(tt.library, tt.channel, tt.googleapisDir, tt.discoveryDir, tt.protobufDir, tt.protobufSubDir, tt.conformanceDir, tt.showcaseDir)
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestExtraModulesFromKeep(t *testing.T) {
	for _, test := range []struct {
		name string
		keep []string
		want []string
	}{
		{
			name: "empty keep list",
			keep: nil,
			want: nil,
		},
		{
			name: "single module",
			keep: []string{"src/errors.rs"},
			want: []string{"errors"},
		},
		{
			name: "multiple modules",
			keep: []string{"src/errors.rs", "src/operation.rs"},
			want: []string{"errors", "operation"},
		},
		{
			name: "ignores non-src files",
			keep: []string{"Cargo.toml", "README.md"},
			want: nil,
		},
		{
			name: "ignores non-rs files in src",
			keep: []string{"src/lib.rs.bak"},
			want: nil,
		},
		{
			name: "mixed files",
			keep: []string{"Cargo.toml", "src/errors.rs", "README.md", "src/operation.rs"},
			want: []string{"errors", "operation"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := extraModulesFromKeep(test.keep)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatPackageDependency(t *testing.T) {
	for _, test := range []struct {
		name string
		dep  config.RustPackageDependency
		want string
	}{
		{
			name: "minimal dependency",
			dep: config.RustPackageDependency{
				Name:    "tokio",
				Package: "tokio",
			},
			want: "package=tokio",
		},
		{
			name: "with source",
			dep: config.RustPackageDependency{
				Name:    "tokio",
				Package: "tokio",
				Source:  "1.0",
			},
			want: "package=tokio,source=1.0",
		},
		{
			name: "with force used",
			dep: config.RustPackageDependency{
				Name:      "tokio",
				Package:   "tokio",
				ForceUsed: true,
			},
			want: "package=tokio,force-used=true",
		},
		{
			name: "with used if",
			dep: config.RustPackageDependency{
				Name:    "tokio",
				Package: "tokio",
				UsedIf:  "feature = \"async\"",
			},
			want: "package=tokio,used-if=feature = \"async\"",
		},
		{
			name: "with feature",
			dep: config.RustPackageDependency{
				Name:    "tokio",
				Package: "tokio",
				Feature: "async",
			},
			want: "package=tokio,feature=async",
		},
		{
			name: "all fields",
			dep: config.RustPackageDependency{
				Name:      "tokio",
				Package:   "tokio",
				Source:    "1.0",
				ForceUsed: true,
				UsedIf:    "feature = \"async\"",
				Feature:   "async",
				Ignore:    true,
			},
			want: "package=tokio,source=1.0,force-used=true,used-if=feature = \"async\",feature=async,ignore=true",
		},
		{
			name: "with ignore for self-referencing package",
			dep: config.RustPackageDependency{
				Name:   "longrunning",
				Ignore: true,
			},
			want: "ignore=true",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatPackageDependency(&test.dep)
			if got != test.want {
				t.Errorf("formatPackageDependency() = %q, want %q", got, test.want)
			}
		})
	}
}
