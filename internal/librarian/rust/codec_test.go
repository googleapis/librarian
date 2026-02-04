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

func TestModuleToSidekickConfig(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		want    *sidekickconfig.Config
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
			want: &sidekickconfig.Config{
				Source: map[string]string{
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
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
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					SpecificationFormat: "none",
				},
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
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "rust+prost",
					SpecificationFormat: "protobuf",
				},
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
					"googleapis-root": absPath(t, googleapisRoot),
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
					"googleapis-root": absPath(t, googleapisRoot),
					"roots":           "googleapis",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			sources := &Sources{
				Conformance: absPath(t, conformanceRoot),
				Discovery:   absPath(t, discoveryRoot),
				Googleapis:  absPath(t, googleapisRoot),
				ProtobufSrc: absPath(t, protobufSrcRoot),
				Showcase:    absPath(t, showcaseRoot),
			}

			for _, module := range test.library.Rust.Modules {
				got, err := moduleToSidekickConfig(test.library, module, sources)
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(test.want.Source, got.Source); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
				if test.want.General.Language != "" {
					if diff := cmp.Diff(test.want.General, got.General); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}
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
			name: "with version and release level",
			library: &config.Library{
				Name:         "google-cloud-secretmanager",
				Version:      "0.1.0",
				ReleaseLevel: "preview",
			},
			want: map[string]string{
				"package-name-override": "google-cloud-secretmanager",
				"version":               "0.1.0",
				"release-level":         "preview",
			},
		},
		{
			name: "with rust config",
			library: &config.Library{
				Name: "google-cloud-secretmanager",
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
					},
					ModulePath: "gcs",
				},
			},
			want: map[string]string{
				"package-name-override":     "google-cloud-secretmanager",
				"disabled-rustdoc-warnings": "broken_intra_doc_links",
				"module-path":               "gcs",
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
