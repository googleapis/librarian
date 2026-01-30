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
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
)

const (
	googleapisRoot  = "../../../internal/testdata/googleapis"
	discoveryRoot   = "../../../internal/testdata/discovery"
	protobufSrcRoot = "../../../internal/testdata/protobuf-src"
	conformanceRoot = "../../../internal/testdata/conformance"
	showcaseRoot    = "../../../internal/testdata/gapic-showcase"
)

func absPath(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		panic(err)
	}
	return a
}

func TestToSidekickConfig(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		api     *config.API
		want    *sidekickconfig.Config
	}{
		{
			name: "minimal config",
			library: &config.Library{
				Name: "google-cloud-storage",
				Rust: &config.RustCrate{},
			},
			api: &config.API{
				Path: "google/cloud/storage/v1",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					SpecificationSource: "google/cloud/storage/v1",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
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
				Rust: &config.RustCrate{
					Roots: []string{"googleapis"},
				},
			},
			api: &config.API{
				Path: "google/cloud/storage/v1",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					SpecificationSource: "google/cloud/storage/v1",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
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
				Rust: &config.RustCrate{
					Roots: []string{"googleapis"},
				},
			},
			api: &config.API{
				Path: "google/cloud/storage/v1",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					SpecificationSource: "google/cloud/storage/v1",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
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
						GenerateRpcSamples:      "true",
					},
					ModulePath:                "gcs",
					PerServiceFeatures:        true,
					IncludeGrpcOnlyMethods:    true,
					DetailedTracingAttributes: true,
					HasVeneer:                 true,
					RoutingRequired:           true,
					DisabledClippyWarnings:    []string{"too_many_arguments"},
					DefaultFeatures:           []string{"default-feature"},
					TemplateOverride:          "custom-template",
				},
			},
			api: &config.API{
				Path: "google/cloud/storage/v1",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					SpecificationSource: "google/cloud/storage/v1",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
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
				Rust: &config.RustCrate{
					Roots: []string{"googleapis"},
				},
			},
			api: &config.API{
				Path: "google/cloud/storage/v1",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					SpecificationSource: "google/cloud/storage/v1",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
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
			api: &config.API{
				Path: "google/cloud/storage/v1",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					SpecificationSource: "google/cloud/storage/v1",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
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
			api: &config.API{
				Path: "google/cloud/storage/v1",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					SpecificationSource: "google/cloud/storage/v1",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
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
			api: &config.API{
				Path: "google/cloud/storage/v1",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					SpecificationSource: "google/cloud/storage/v1",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
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
			api: &config.API{
				Path: "google/cloud/storage/v1",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					SpecificationSource: "google/cloud/storage/v1",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
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
				Rust: &config.RustCrate{
					Roots: []string{"googleapis", "discovery"},
				},
			},
			api: &config.API{
				Path: "discoveries/compute.v1.json",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "disco",
					SpecificationSource: "discoveries/compute.v1.json",
					ServiceConfig:       "google/cloud/compute/v1/compute_v1.yaml",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
					"discovery-root":  absPath(discoveryRoot),
					"roots":           "googleapis,discovery",
					"title-override":  "Google Compute Engine API",
				},
				Codec: map[string]string{
					"package-name-override": "google-cloud-compute-v1",
				},
			},
		},
		{
			name: "with openapi format",
			library: &config.Library{
				Name:                "secretmanager-openapi-v1",
				SpecificationFormat: "openapi",
				Rust: &config.RustCrate{
					Roots: []string{"googleapis"},
				},
			},
			api: &config.API{
				Path: "testdata/secretmanager_openapi_v1.json",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "openapi",
					SpecificationSource: "testdata/secretmanager_openapi_v1.json",
					ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
					"roots":           "googleapis",
					"title-override":  "Secret Manager API",
				},
				Codec: map[string]string{
					"package-name-override": "secretmanager-openapi-v1",
				},
			},
		},
		{
			name: "with multiple formats",
			library: &config.Library{
				Name:                "google-cloud-compute-v1",
				SpecificationFormat: "discovery",
				Rust: &config.RustCrate{
					Roots: []string{"googleapis", "discovery", "showcase"},
				},
			},
			api: &config.API{
				Path: "discoveries/compute.v1.json",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "disco",
					SpecificationSource: "discoveries/compute.v1.json",
					ServiceConfig:       "google/cloud/compute/v1/compute_v1.yaml",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
					"discovery-root":  absPath(discoveryRoot),
					"showcase-root":   absPath(showcaseRoot),
					"roots":           "googleapis,discovery,showcase",
					"title-override":  "Google Compute Engine API",
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
			},
			api: &config.API{
				Path: "google/apps/script/type/gmail",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					SpecificationSource: "google/apps/script/type/gmail",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
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
			api: &config.API{
				Path: "google/longrunning",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					SpecificationSource: "google/longrunning",
				},
				Source: map[string]string{
					"googleapis-root":      absPath(googleapisRoot),
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
			api: &config.API{
				Path: "google/spanner/admin/database/v1",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					SpecificationSource: "google/spanner/admin/database/v1",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
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
			api: &config.API{
				Path: "google/cloud/storageinsights/v1",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					SpecificationSource: "google/cloud/storageinsights/v1",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
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
					Roots: []string{"googleapis", "discovery"},
				},
			},
			api: &config.API{
				Path: "discoveries/compute.v1.json",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "disco",
					SpecificationSource: "discoveries/compute.v1.json",
					ServiceConfig:       "google/cloud/compute/v1/compute_v1.yaml",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
					"discovery-root":  absPath(discoveryRoot),
					"roots":           "googleapis,discovery",
					"title-override":  "Google Compute Engine API",
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
				Name: "google-cloud-vision-v1",
				Rust: &config.RustCrate{
					Roots: []string{"googleapis", "protobuf-src", "conformance"},
				},
			},
			api: &config.API{
				Path: "google/cloud/vision/v1",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					SpecificationSource: "google/cloud/vision/v1",
				},
				Source: map[string]string{
					"googleapis-root":   absPath(googleapisRoot),
					"protobuf-src-root": absPath(protobufSrcRoot),
					"conformance-root":  absPath(conformanceRoot),
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
				Name: "google-cloud-showcase",
				Rust: &config.RustCrate{
					Roots: []string{"showcase", "googleapis"},
				},
			},
			api: &config.API{
				Path: "schema/google/showcase/v1beta1",
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "protobuf",
					SpecificationSource: "schema/google/showcase/v1beta1",
					ServiceConfig:       "schema/google/showcase/v1beta1/showcase_v1beta1.yaml",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
					"showcase-root":   absPath(showcaseRoot),
					"roots":           "showcase,googleapis",
					"title-override":  "Client Libraries Showcase API",
				},
				Codec: map[string]string{
					"package-name-override": "google-cloud-showcase",
				},
			},
		},
		{
			name: "with custom module language",
			library: &config.Library{
				Name: "google-cloud-showcase",
				Rust: &config.RustCrate{
					Modules: []*config.RustModule{
						{
							Language: "rust_storage",
						},
					},
				},
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust_storage",
					SpecificationFormat: "protobuf",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
					"roots":           "googleapis",
				},
			},
		},
		{
			name: "with custom module specification format",
			library: &config.Library{
				Name: "google-cloud-showcase",
				Rust: &config.RustCrate{
					Modules: []*config.RustModule{
						{
							SpecificationFormat: "none",
						},
					},
				},
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust",
					SpecificationFormat: "none",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
					"roots":           "googleapis",
				},
			},
		},
		{
			name: "with prost as module template",
			library: &config.Library{
				Name: "google-cloud-showcase",
				Rust: &config.RustCrate{
					Modules: []*config.RustModule{
						{
							Template: "prost",
						},
					},
				},
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language: "rust+prost",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
					"roots":           "googleapis",
				},
			},
		},
		{
			name: "with api source and title",
			library: &config.Library{
				Name: "google-cloud-logging",
				Rust: &config.RustCrate{
					Modules: []*config.RustModule{
						{
							Template: "prost",
							Source:   "google/logging/type",
						},
					},
				},
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust+prost",
					SpecificationFormat: "protobuf",
					SpecificationSource: "google/logging/type",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
					"roots":           "googleapis",
					"title-override":  "Logging types",
				},
			},
		},
		{
			name: "with included ids in rust module",
			library: &config.Library{
				Name: "google-cloud-example",
				Rust: &config.RustCrate{
					Modules: []*config.RustModule{
						{
							Template:    "prost",
							IncludedIds: []string{"id1", "id2"},
							SkippedIds:  []string{"id3", "id4"},
							IncludeList: "example-list",
						},
					},
				},
			},
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust+prost",
					SpecificationFormat: "protobuf",
				},
				Source: map[string]string{
					"googleapis-root": absPath(googleapisRoot),
					"included-ids":    "id1,id2",
					"include-list":    "example-list",
					"roots":           "googleapis",
					"skipped-ids":     "id3,id4",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			sources := &Sources{
				Conformance: absPath(conformanceRoot),
				Discovery:   absPath(discoveryRoot),
				Googleapis:  absPath(googleapisRoot),
				ProtobufSrc: absPath(protobufSrcRoot),
				Showcase:    absPath(showcaseRoot),
			}

			if test.library.Rust != nil && test.library.Rust.Modules != nil {
				var commentOverrides []sidekickconfig.DocumentationOverride
				for _, module := range test.library.Rust.Modules {
					got, err := moduleToSidekickConfig(test.library, module, sources)
					if err != nil {
						t.Fatal(err)
					}
					if diff := cmp.Diff(test.want.Source, got.Source); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}
					commentOverrides = append(commentOverrides, got.CommentOverrides...)
				}
				if diff := cmp.Diff(test.want.CommentOverrides, commentOverrides); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			} else {
				got, err := toSidekickConfig(test.library, test.api, sources)
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(test.want, got); diff != "" {
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
