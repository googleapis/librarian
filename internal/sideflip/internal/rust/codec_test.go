// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"github.com/googleapis/librarian/internal/sideflip/config"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
)

func TestToSidekickConfig(t *testing.T) {
	for _, test := range []struct {
		name            string
		library         *config.Library
		googleapisDir   string
		serviceConfig   string
		want            *sidekickconfig.Config
		wantCodecSubset map[string]string
	}{
		{
			name: "minimal config",
			library: &config.Library{
				Channel: "google/cloud/storage/v1",
				Name:    "google-cloud-storage",
			},
			googleapisDir: "/tmp/googleapis",
			serviceConfig: "google/cloud/storage/v1/storage_v1.yaml",
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
			},
			wantCodecSubset: map[string]string{
				"package-name-override": "google-cloud-storage",
			},
		},
		{
			name: "with version and release level",
			library: &config.Library{
				Channel:      "google/cloud/storage/v1",
				Name:         "google-cloud-storage",
				Version:      "0.1.0",
				ReleaseLevel: "preview",
			},
			googleapisDir: "/tmp/googleapis",
			serviceConfig: "google/cloud/storage/v1/storage_v1.yaml",
			wantCodecSubset: map[string]string{
				"version":               "0.1.0",
				"release-level":         "preview",
				"package-name-override": "google-cloud-storage",
			},
		},
		{
			name: "with copyright year",
			library: &config.Library{
				Channel:       "google/cloud/storage/v1",
				Name:          "google-cloud-storage",
				CopyrightYear: "2024",
			},
			googleapisDir: "/tmp/googleapis",
			serviceConfig: "google/cloud/storage/v1/storage_v1.yaml",
			wantCodecSubset: map[string]string{
				"copyright-year":        "2024",
				"package-name-override": "google-cloud-storage",
			},
		},
		{
			name: "with rust config",
			library: &config.Library{
				Channel: "google/cloud/storage/v1",
				Name:    "google-cloud-storage",
				Rust: &config.RustCrate{
					ModulePath:                "gcs",
					PerServiceFeatures:        true,
					IncludeGrpcOnlyMethods:    true,
					DetailedTracingAttributes: true,
					HasVeneer:                 true,
					RoutingRequired:           true,
					GenerateSetterSamples:     true,
					DisabledRustdocWarnings:   []string{"broken_intra_doc_links"},
					DisabledClippyWarnings:    []string{"too_many_arguments"},
					DefaultFeatures:           []string{"default-feature"},
					ExtraModules:              []string{"extra-module"},
					TemplateOverride:          "custom-template",
				},
			},
			googleapisDir: "/tmp/googleapis",
			serviceConfig: "google/cloud/storage/v1/storage_v1.yaml",
			wantCodecSubset: map[string]string{
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
		{
			name: "with publish disabled",
			library: &config.Library{
				Channel: "google/cloud/storage/v1",
				Name:    "google-cloud-storage",
				Publish: &config.LibraryPublish{
					Disabled: true,
				},
				Rust: &config.RustCrate{},
			},
			googleapisDir: "/tmp/googleapis",
			serviceConfig: "google/cloud/storage/v1/storage_v1.yaml",
			wantCodecSubset: map[string]string{
				"not-for-publication":   "true",
				"package-name-override": "google-cloud-storage",
			},
		},
		{
			name: "with package dependencies",
			library: &config.Library{
				Channel: "google/cloud/storage/v1",
				Name:    "google-cloud-storage",
				Rust: &config.RustCrate{
					PackageDependencies: []config.RustPackageDependency{
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
			googleapisDir: "/tmp/googleapis",
			serviceConfig: "google/cloud/storage/v1/storage_v1.yaml",
			wantCodecSubset: map[string]string{
				"package:tokio":         "package=tokio,source=1.0,force-used=true,used-if=feature = \"async\",feature=async",
				"package-name-override": "google-cloud-storage",
			},
		},
		{
			name: "with documentation overrides",
			library: &config.Library{
				Channel: "google/cloud/storage/v1",
				Name:    "google-cloud-storage",
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
			googleapisDir: "/tmp/googleapis",
			serviceConfig: "google/cloud/storage/v1/storage_v1.yaml",
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
				CommentOverrides: []sidekickconfig.DocumentationOverride{
					{
						ID:      ".google.cloud.storage.v1.Bucket.name",
						Match:   "bucket name",
						Replace: "the name of the bucket",
					},
				},
			},
			wantCodecSubset: map[string]string{
				"package-name-override": "google-cloud-storage",
			},
		},
		{
			name: "with pagination overrides",
			library: &config.Library{
				Channel: "google/cloud/storage/v1",
				Name:    "google-cloud-storage",
				Rust: &config.RustCrate{
					PaginationOverrides: []config.RustPaginationOverride{
						{
							ID:        ".google.cloud.storage.v1.Storage.ListBuckets",
							ItemField: "buckets",
						},
					},
				},
			},
			googleapisDir: "/tmp/googleapis",
			serviceConfig: "google/cloud/storage/v1/storage_v1.yaml",
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
				PaginationOverrides: []sidekickconfig.PaginationOverride{
					{
						ID:        ".google.cloud.storage.v1.Storage.ListBuckets",
						ItemField: "buckets",
					},
				},
			},
			wantCodecSubset: map[string]string{
				"package-name-override": "google-cloud-storage",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := toSidekickConfig(test.library, test.googleapisDir, test.serviceConfig)
			if err != nil {
				t.Fatal(err)
			}

			if test.want != nil {
				if diff := cmp.Diff(test.want.General, got.General); diff != "" {
					t.Errorf("General mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(test.want.Source, got.Source); diff != "" {
					t.Errorf("Source mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(test.want.CommentOverrides, got.CommentOverrides); diff != "" {
					t.Errorf("CommentOverrides mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(test.want.PaginationOverrides, got.PaginationOverrides); diff != "" {
					t.Errorf("PaginationOverrides mismatch (-want +got):\n%s", diff)
				}
			}

			if test.wantCodecSubset != nil {
				for key, wantValue := range test.wantCodecSubset {
					if gotValue, ok := got.Codec[key]; !ok {
						t.Errorf("Codec missing key %q", key)
					} else if gotValue != wantValue {
						t.Errorf("Codec[%q] = %q, want %q", key, gotValue, wantValue)
					}
				}
			}
		})
	}
}

func TestFormatPackageDependency(t *testing.T) {
	for _, test := range []struct {
		name string
		dep  *config.RustPackageDependency
		want string
	}{
		{
			name: "minimal dependency",
			dep: &config.RustPackageDependency{
				Name:    "tokio",
				Package: "tokio",
			},
			want: "package=tokio",
		},
		{
			name: "with source",
			dep: &config.RustPackageDependency{
				Name:    "tokio",
				Package: "tokio",
				Source:  "1.0",
			},
			want: "package=tokio,source=1.0",
		},
		{
			name: "with force used",
			dep: &config.RustPackageDependency{
				Name:      "tokio",
				Package:   "tokio",
				ForceUsed: true,
			},
			want: "package=tokio,force-used=true",
		},
		{
			name: "with used if",
			dep: &config.RustPackageDependency{
				Name:    "tokio",
				Package: "tokio",
				UsedIf:  "feature = \"async\"",
			},
			want: "package=tokio,used-if=feature = \"async\"",
		},
		{
			name: "with feature",
			dep: &config.RustPackageDependency{
				Name:    "tokio",
				Package: "tokio",
				Feature: "async",
			},
			want: "package=tokio,feature=async",
		},
		{
			name: "all fields",
			dep: &config.RustPackageDependency{
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
			got := formatPackageDependency(test.dep)
			if got != test.want {
				t.Errorf("formatPackageDependency() = %q, want %q", got, test.want)
			}
		})
	}
}
