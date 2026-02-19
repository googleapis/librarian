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
	"github.com/googleapis/librarian/internal/sidekick/api"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sidekick/source"
)

const (
	googleapisRoot  = "../../../internal/testdata/googleapis"
	discoveryRoot   = "fake/path/to/testdata/discovery"
	protobufSrcRoot = "fake/path/to/testdata/protobuf-src"
	conformanceRoot = "fake/path/to/testdata/conformance"
	showcaseRoot    = "../../../internal/testdata/gapic-showcase"
)

func absPath(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestLibraryToModelConfig(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		api     *config.API
		want    *parser.ModelConfig
	}{
		{
			name: "minimal config",
			library: &config.Library{
				Name:   "google-cloud-secretmanager",
				Rust:   &config.RustCrate{},
				Veneer: true,
			},
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
			},
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "google/cloud/secretmanager/v1",
				ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
				},
				Override: api.ModelOverride{
					Title: "Secret Manager API",
				},
			},
		},
		{
			name: "with version and release level",
			library: &config.Library{
				Name:         "google-cloud-secretmanager",
				Version:      "0.1.0",
				ReleaseLevel: "preview",
				Roots:        []string{"googleapis"},
			},
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
			},
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "google/cloud/secretmanager/v1",
				ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
				},
				Override: api.ModelOverride{
					Title: "Secret Manager API",
				},
			},
		},
		{
			name: "with copyright year",
			library: &config.Library{
				Name:          "google-cloud-secretmanager",
				CopyrightYear: "2024",
				Roots:         []string{"googleapis"},
			},
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
			},
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "google/cloud/secretmanager/v1",
				ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
				},
				Override: api.ModelOverride{
					Title: "Secret Manager API",
				},
			},
		},
		{
			name: "with rust config",
			library: &config.Library{
				Name: "google-cloud-secretmanager",
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
				Path: "google/cloud/secretmanager/v1",
			},
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "google/cloud/secretmanager/v1",
				ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
				},
				Override: api.ModelOverride{
					Title: "Secret Manager API",
				},
			},
		},
		{
			name: "with skip publish (not for publication)",
			library: &config.Library{
				Name:        "google-cloud-secretmanager",
				Roots:       []string{"googleapis"},
				SkipRelease: true,
			},
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
			},
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "google/cloud/secretmanager/v1",
				ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
				},
				Override: api.ModelOverride{
					Title: "Secret Manager API",
				},
			},
		},
		{
			name: "with package dependencies",
			library: &config.Library{
				Name: "google-cloud-secretmanager",
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
				Path: "google/cloud/secretmanager/v1",
			},
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "google/cloud/secretmanager/v1",
				ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
				},
				Override: api.ModelOverride{
					Title: "Secret Manager API",
				},
			},
		},
		{
			name: "with documentation overrides",
			library: &config.Library{
				Name: "google-cloud-secretmanager",
				Rust: &config.RustCrate{
					DocumentationOverrides: []config.RustDocumentationOverride{
						{
							ID:      ".google.cloud.secretmanager.v1.Secret.name",
							Match:   "secret name",
							Replace: "the name of the Secret",
						},
					},
				},
			},
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
			},
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "google/cloud/secretmanager/v1",
				ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
				},
				Override: api.ModelOverride{
					Title: "Secret Manager API",
				},
				CommentOverrides: []api.DocumentationOverride{
					{
						ID:      ".google.cloud.secretmanager.v1.Secret.name",
						Match:   "secret name",
						Replace: "the name of the Secret",
					},
				},
			},
		},
		{
			name: "with pagination secretmanager",
			library: &config.Library{
				Name: "google-cloud-secretmanager",
				Rust: &config.RustCrate{
					PaginationOverrides: []config.RustPaginationOverride{
						{
							ID:        ".google.cloud.secretmanager.v1.Secret.ListSecrets",
							ItemField: "secrets",
						},
					},
				},
			},
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
			},
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "google/cloud/secretmanager/v1",
				ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
				},
				Override: api.ModelOverride{
					Title: "Secret Manager API",
				},
				PaginationOverrides: []api.PaginationOverride{
					{
						ID:        ".google.cloud.secretmanager.v1.Secret.ListSecrets",
						ItemField: "secrets",
					},
				},
			},
		},
		{
			name: "with discovery format",
			library: &config.Library{
				Name:                "google-cloud-compute-v1",
				Roots:               []string{"googleapis", "discovery"},
				SpecificationFormat: config.SpecDiscovery,
			},
			api: &config.API{
				Path: "discoveries/compute.v1.json",
			},
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecDiscovery,
				SpecificationSource: "discoveries/compute.v1.json",
				ServiceConfig:       "google/cloud/compute/v1/compute_v1.yaml",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"discovery-root":  absPath(t, discoveryRoot),
					"roots":           "googleapis,discovery",
				},
				Override: api.ModelOverride{
					Title: "Google Compute Engine API",
				},
			},
		},
		{
			name: "with openapi format",
			library: &config.Library{
				Name:                "secretmanager-openapi-v1",
				Roots:               []string{"googleapis"},
				SpecificationFormat: config.SpecOpenAPI,
			},
			api: &config.API{
				Path: "testdata/secretmanager_openapi_v1.json",
			},
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecOpenAPI,
				SpecificationSource: "testdata/secretmanager_openapi_v1.json",
				ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
				},
				Override: api.ModelOverride{
					Title: "Secret Manager API",
				},
			},
		},
		{
			name: "with multiple formats",
			library: &config.Library{
				Name:                "google-cloud-compute-v1",
				Roots:               []string{"googleapis", "discovery", "showcase"},
				SpecificationFormat: config.SpecDiscovery,
			},
			api: &config.API{
				Path: "discoveries/compute.v1.json",
			},
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecDiscovery,
				SpecificationSource: "discoveries/compute.v1.json",
				ServiceConfig:       "google/cloud/compute/v1/compute_v1.yaml",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"discovery-root":  absPath(t, discoveryRoot),
					"showcase-root":   absPath(t, showcaseRoot),
					"roots":           "googleapis,discovery,showcase",
				},
				Override: api.ModelOverride{
					Title: "Google Compute Engine API",
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
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "google/apps/script/type/gmail",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
				},
				Override: api.ModelOverride{
					Title: "Google Apps Script Types",
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
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "google/longrunning",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
				},
				Override: api.ModelOverride{
					Description: "Defines types and an abstract service to handle long-running operations.",
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
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "google/spanner/admin/database/v1",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
				},
				Override: api.ModelOverride{
					SkippedIDs: []string{
						".google.spanner.admin.database.v1.DatabaseAdmin.InternalUpdateGraphOperation",
						".google.spanner.admin.database.v1.InternalUpdateGraphOperationRequest",
						".google.spanner.admin.database.v1.InternalUpdateGraphOperationResponse",
					},
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
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "google/cloud/storageinsights/v1",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
				},
			},
		},
		{
			name: "with discovery LRO polling config",
			library: &config.Library{
				Name:                "google-cloud-compute-v1",
				Roots:               []string{"googleapis", "discovery"},
				SpecificationFormat: config.SpecDiscovery,
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
			api: &config.API{
				Path: "discoveries/compute.v1.json",
			},
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecDiscovery,
				SpecificationSource: "discoveries/compute.v1.json",
				ServiceConfig:       "google/cloud/compute/v1/compute_v1.yaml",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"discovery-root":  absPath(t, discoveryRoot),
					"roots":           "googleapis,discovery",
				},
				Override: api.ModelOverride{
					Title: "Google Compute Engine API",
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
			api: &config.API{
				Path: "google/cloud/vision/v1",
			},
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "google/cloud/vision/v1",
				Source: map[string]string{
					"googleapis-root":   absPath(t, googleapisRoot),
					"protobuf-src-root": absPath(t, protobufSrcRoot),
					"conformance-root":  absPath(t, conformanceRoot),
					"roots":             "googleapis,protobuf-src,conformance",
				},
			},
		},
		{
			name: "with showcase as source",
			library: &config.Library{
				Name:  "google-cloud-showcase",
				Roots: []string{"showcase", "googleapis"},
			},
			api: &config.API{
				Path: "schema/google/showcase/v1beta1",
			},
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "schema/google/showcase/v1beta1",
				ServiceConfig:       "schema/google/showcase/v1beta1/showcase_v1beta1.yaml",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"showcase-root":   absPath(t, showcaseRoot),
					"roots":           "showcase,googleapis",
				},
				Override: api.ModelOverride{
					Title: "Client Libraries Showcase API",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			sources := &source.Sources{
				Conformance: absPath(t, conformanceRoot),
				Discovery:   absPath(t, discoveryRoot),
				Googleapis:  absPath(t, googleapisRoot),
				ProtobufSrc: absPath(t, protobufSrcRoot),
				Showcase:    absPath(t, showcaseRoot),
			}

			got, err := libraryToModelConfig(test.library, test.api, sources)
			if err != nil {
				t.Fatal(err)
			}
			if test.want.Codec == nil {
				test.want.Codec = buildCodec(test.library)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestModuleToModelConfig(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		want    *parser.ModelConfig
	}{
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
			want: &parser.ModelConfig{
				Language: "rust",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
				},
				CommentOverrides: []api.DocumentationOverride{
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
			want: &parser.ModelConfig{
				Language:            "rust_storage",
				SpecificationFormat: config.SpecProtobuf,
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
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
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: "none",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
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
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecProtobuf,
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
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
							APIPath:  "google/logging/type",
						},
					},
				},
			},
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "google/logging/type",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
				},
				Override: api.ModelOverride{
					Title: "Logging types",
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
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecProtobuf,
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"include-list":    "example-list",
					"roots":           "googleapis",
				},
				Override: api.ModelOverride{
					IncludedIDs: []string{"id1", "id2"},
					SkippedIDs:  []string{"id3", "id4"},
				},
			},
		},
		{
			name: "with conformance as module source",
			library: &config.Library{
				Name: "google-cloud-example",
				Rust: &config.RustCrate{
					Modules: []*config.RustModule{
						{
							APIPath: "conformance",
						},
					},
				},
				Roots: []string{"conformance"},
			},
			want: &parser.ModelConfig{
				Language:            "rust",
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "conformance",
				Source: map[string]string{
					"conformance-root": absPath(t, conformanceRoot),
					"roots":            "conformance",
				},
			},
		},
		{
			name: "with pagination overrides in rust module",
			library: &config.Library{
				Name: "google-cloud-example",
				Rust: &config.RustCrate{
					PaginationOverrides: []config.RustPaginationOverride{
						{
							ID:        ".google.cloud.example.v1.Example.ListExamples",
							ItemField: "examples",
						},
					},
					Modules: []*config.RustModule{
						{
							Template: "prost",
						},
					},
				},
			},
			want: &parser.ModelConfig{
				Language: "rust",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
				},
				PaginationOverrides: []api.PaginationOverride{
					{
						ID:        ".google.cloud.example.v1.Example.ListExamples",
						ItemField: "examples",
					},
				},
			},
		},
		{
			name: "with pagination overrides in multiple rust modules",
			library: &config.Library{
				Name: "google-cloud-example",
				Rust: &config.RustCrate{
					PaginationOverrides: []config.RustPaginationOverride{
						{
							ID:        ".google.cloud.example.v1.Example.ListExamples",
							ItemField: "examples",
						},
					},
					Modules: []*config.RustModule{
						{
							Template: "prost",
						},
						{
							Template: "prost",
						},
					},
				},
			},
			want: &parser.ModelConfig{
				Language: "rust",
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
				},
				PaginationOverrides: []api.PaginationOverride{
					{
						ID:        ".google.cloud.example.v1.Example.ListExamples",
						ItemField: "examples",
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			sources := &source.Sources{
				Conformance: absPath(t, conformanceRoot),
				Discovery:   absPath(t, discoveryRoot),
				Googleapis:  absPath(t, googleapisRoot),
				ProtobufSrc: absPath(t, protobufSrcRoot),
				Showcase:    absPath(t, showcaseRoot),
			}

			var commentOverrides []api.DocumentationOverride
			for _, module := range test.library.Rust.Modules {
				got, err := moduleToModelConfig(test.library, module, sources)
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(test.want.Source, got.Source); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
				commentOverrides = append(commentOverrides, got.CommentOverrides...)
				if diff := cmp.Diff(test.want.PaginationOverrides, got.PaginationOverrides); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
			if diff := cmp.Diff(test.want.CommentOverrides, commentOverrides); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
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

func TestBuildCodec(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		want    map[string]string
	}{
		{
			name: "minimal config",
			library: &config.Library{
				Name: "google-cloud-secretmanager",
			},
			want: map[string]string{
				"package-name-override": "google-cloud-secretmanager",
			},
		},
		{
			name: "with general config",
			library: &config.Library{
				Name:          "google-cloud-secretmanager",
				Version:       "0.1.0",
				ReleaseLevel:  "preview",
				CopyrightYear: "2024",
				SkipRelease:   true,
				Keep:          []string{"src/mod1.rs", "src/mod2.rs", "other.txt"},
			},
			want: map[string]string{
				"package-name-override": "google-cloud-secretmanager",
				"version":               "0.1.0",
				"release-level":         "preview",
				"copyright-year":        "2024",
				"not-for-publication":   "true",
				"extra-modules":         "mod1,mod2",
			},
		},
		{
			name: "with rust config",
			library: &config.Library{
				Name: "google-cloud-secretmanager",
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						GenerateSetterSamples:   "true",
						GenerateRpcSamples:      "true",
						DisabledRustdocWarnings: []string{"warning1", "warning2"},
						PackageDependencies: []*config.RustPackageDependency{
							{
								Name:    "dep1",
								Package: "pkg1",
								Source:  "1.0",
							},
							{
								Name:    "dep2",
								Feature: "feat2",
								Ignore:  true,
							},
						},
					},
					ModulePath:                "gcs",
					TemplateOverride:          "custom-template",
					IncludeGrpcOnlyMethods:    true,
					PerServiceFeatures:        true,
					DetailedTracingAttributes: true,
					HasVeneer:                 true,
					RoutingRequired:           true,
					NameOverrides:             "foo=bar",
					DefaultFeatures:           []string{"feature1", "feature2"},
					DisabledClippyWarnings:    []string{"clippy1", "clippy2"},
				},
			},
			want: map[string]string{
				"package-name-override":       "google-cloud-secretmanager",
				"module-path":                 "gcs",
				"template-override":           "custom-template",
				"include-grpc-only-methods":   "true",
				"per-service-features":        "true",
				"detailed-tracing-attributes": "true",
				"has-veneer":                  "true",
				"routing-required":            "true",
				"generate-setter-samples":     "true",
				"generate-rpc-samples":        "true",
				"name-overrides":              "foo=bar",
				"default-features":            "feature1,feature2",
				"disabled-rustdoc-warnings":   "warning1,warning2",
				"disabled-clippy-warnings":    "clippy1,clippy2",
				"package:dep1":                "package=pkg1,source=1.0",
				"package:dep2":                "feature=feat2,ignore=true",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildCodec(test.library)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
