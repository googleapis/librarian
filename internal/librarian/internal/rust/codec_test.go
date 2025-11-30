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
	for _, test := range []struct {
		name          string
		library       *config.Library
		channel       *config.Channel
		googleapisDir string
		discoveryDir  string
		want          *sidekickconfig.Config
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
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
					},
					ModulePath:                "gcs",
					PerServiceFeatures:        true,
					IncludeGrpcOnlyMethods:    true,
					DetailedTracingAttributes: true,
					HasVeneer:                 true,
					RoutingRequired:           true,
					GenerateSetterSamples:     true,
					DisabledClippyWarnings:    []string{"too_many_arguments"},
					DefaultFeatures:           []string{"default-feature"},
					ExtraModules:              []string{"extra-module"},
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
				},
				Codec: map[string]string{
					"module-path":                 "gcs",
					"per-service-features":        "true",
					"include-grpc-only-methods":   "true",
					"detailed-tracing-attributes": "true",
					"has-veneer":                  "true",
					"routing-required":            "true",
					"generate-setter-samples":     "true",
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
					"roots":           "discovery,googleapis",
				},
				Codec: map[string]string{
					"package-name-override": "google-cloud-compute-v1",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := toSidekickConfig(test.library, test.channel, test.googleapisDir, test.discoveryDir)
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
			},
			want: "package=tokio,source=1.0,force-used=true,used-if=feature = \"async\",feature=async",
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
