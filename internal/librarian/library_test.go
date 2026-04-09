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

package librarian

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestFillDefaults(t *testing.T) {
	defaults := &config.Default{
		Keep:   []string{"CHANGES.md"},
		Output: "src/generated/",
	}
	for _, test := range []struct {
		name     string
		defaults *config.Default
		lib      *config.Library
		want     *config.Library
	}{
		{
			name:     "fills empty fields",
			defaults: defaults,
			lib:      &config.Library{},
			want: &config.Library{
				Keep:   []string{"CHANGES.md"},
				Output: "src/generated/",
			},
		},
		{
			name:     "preserves existing values",
			defaults: defaults,
			lib: &config.Library{
				Output: "custom/output/",
			},
			want: &config.Library{
				Keep:   []string{"CHANGES.md"},
				Output: "custom/output/",
			},
		},
		{
			name:     "partial fill",
			defaults: defaults,
			lib:      &config.Library{Output: "custom/output/"},
			want: &config.Library{
				Keep:   []string{"CHANGES.md"},
				Output: "custom/output/",
			},
		},
		{
			name:     "nil defaults",
			defaults: nil,
			lib:      &config.Library{Output: "foo/"},
			want:     &config.Library{Output: "foo/"},
		},
		{
			name: "dart defaults",
			defaults: &config.Default{
				Dart: &config.DartPackage{
					APIKeysEnvironmentVariables: "apiKey-1,apiKey-2",
					Dependencies:                "dep-1,dep-2",
					IssueTrackerURL:             "https://issue-tracker-example/dart",
					Packages: map[string]string{
						"package:one": "^1.2.3",
						"package:two": "^2.0.0",
					},
					Prefixes: map[string]string{
						"prefix:google.logging.type": "logging_type",
					},
					Protos: map[string]string{
						"proto:google.api":          "package:google_cloud_api/api.dart",
						"proto:google.cloud.common": "package:google_cloud_common/common.dart",
					},
					Version: "0.4.0",
				},
			},
			lib: &config.Library{Output: "foo/"},
			want: &config.Library{
				Output:  "foo/",
				Version: "0.4.0",
				Dart: &config.DartPackage{
					APIKeysEnvironmentVariables: "apiKey-1,apiKey-2",
					Dependencies:                "dep-1,dep-2",
					IssueTrackerURL:             "https://issue-tracker-example/dart",
					Packages:                    map[string]string{"package:one": "^1.2.3", "package:two": "^2.0.0"},
					Prefixes: map[string]string{
						"prefix:google.logging.type": "logging_type",
					},
					Protos: map[string]string{
						"proto:google.api":          "package:google_cloud_api/api.dart",
						"proto:google.cloud.common": "package:google_cloud_common/common.dart",
					},
				},
			},
		},
		{
			name: "dart defaults do not override library params",
			defaults: &config.Default{
				Dart: &config.DartPackage{
					APIKeysEnvironmentVariables: "apiKey-1,apiKey-2",
					Dependencies:                "dep-1,dep-2",
					IssueTrackerURL:             "https://issue-tracker-example/dart",
					Packages: map[string]string{
						"package:one": "^1.2.3",
						"package:two": "^2.0.0",
					},
					Prefixes: map[string]string{
						"prefix:google.logging.type": "logging_type",
					},
					Protos: map[string]string{
						"proto:google.api":          "package:google_cloud_api/api.dart",
						"proto:google.cloud.common": "package:google_cloud_common/common.dart",
					},
					Version: "0.4.0",
				},
			},
			lib: &config.Library{
				Output:  "foo/",
				Version: "0.5.0",
				Dart: &config.DartPackage{
					APIKeysEnvironmentVariables: "apiKey-3,apiKey-4",
					Dependencies:                "dep-1,dep-3,dep-4",
					IssueTrackerURL:             "https://another-issue-tracker-example/dart",
					Packages: map[string]string{
						"package:three": "^1.0.0",
					},
					Prefixes: map[string]string{
						"prefix:google.logging.type": "logging_type_v2",
					},
					Protos: map[string]string{
						"proto:google.cloud.location": "package:google_cloud_location/location.dart",
					},
				},
			},
			want: &config.Library{
				Output:  "foo/",
				Version: "0.5.0",
				Dart: &config.DartPackage{
					APIKeysEnvironmentVariables: "apiKey-3,apiKey-4",
					Dependencies:                "dep-1,dep-3,dep-4,dep-2",
					IssueTrackerURL:             "https://another-issue-tracker-example/dart",
					Packages: map[string]string{
						"package:one":   "^1.2.3",
						"package:two":   "^2.0.0",
						"package:three": "^1.0.0",
					},
					Prefixes: map[string]string{
						"prefix:google.logging.type": "logging_type_v2",
					},
					Protos: map[string]string{
						"proto:google.cloud.location": "package:google_cloud_location/location.dart",
						"proto:google.api":            "package:google_cloud_api/api.dart",
						"proto:google.cloud.common":   "package:google_cloud_common/common.dart",
					},
				},
			},
		},
		{
			name: "swift defaults",
			defaults: &config.Default{
				Swift: &config.SwiftDefault{
					Dependencies: []config.SwiftDependency{
						{Name: "wkt", URL: "https://github.com/googleapis/swift-protobuf"},
					},
				},
			},
			lib: &config.Library{Output: "foo/"},
			want: &config.Library{
				Output: "foo/",
				Swift: &config.SwiftPackage{
					SwiftDefault: config.SwiftDefault{
						Dependencies: []config.SwiftDependency{
							{Name: "wkt", URL: "https://github.com/googleapis/swift-protobuf"},
						},
					},
				},
			},
		},
		{
			name: "swift defaults do not override library params",
			defaults: &config.Default{
				Swift: &config.SwiftDefault{
					Dependencies: []config.SwiftDependency{
						{Name: "wkt", URL: "https://github.com/googleapis/swift-protobuf"},
						{Name: "gax", URL: "https://github.com/googleapis/gax-swift"},
					},
				},
			},
			lib: &config.Library{
				Output: "foo/",
				Swift: &config.SwiftPackage{
					SwiftDefault: config.SwiftDefault{
						Dependencies: []config.SwiftDependency{
							{Name: "wkt", URL: "https://github.com/custom/swift-protobuf"},
						},
					},
				},
			},
			want: &config.Library{
				Output: "foo/",
				Swift: &config.SwiftPackage{
					SwiftDefault: config.SwiftDefault{
						Dependencies: []config.SwiftDependency{
							{Name: "wkt", URL: "https://github.com/custom/swift-protobuf"},
							{Name: "gax", URL: "https://github.com/googleapis/gax-swift"},
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := fillDefaults(test.lib, test.defaults)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFillDefaults_Rust(t *testing.T) {
	defaults := &config.Default{
		Rust: &config.RustDefault{
			PackageDependencies: []*config.RustPackageDependency{
				{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
				{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
			},
			DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
			GenerateSetterSamples:   "true",
			GenerateRpcSamples:      "true",
		},
	}
	for _, test := range []struct {
		name string
		lib  *config.Library
		want *config.Library
	}{
		{
			name: "fills rust defaults",
			lib: &config.Library{
				Rust: &config.RustCrate{
					Modules: []*config.RustModule{{}},
				},
			},
			want: &config.Library{
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
							{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
							{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
						},
						DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
						GenerateSetterSamples:   "true",
						GenerateRpcSamples:      "true",
					},
					Modules: []*config.RustModule{
						{
							GenerateSetterSamples: "true",
							GenerateRpcSamples:    "true",
						},
					},
				},
			},
		},
		{
			name: "merges package dependencies",
			lib: &config.Library{
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
							{Name: "custom", Package: "custom-pkg"},
						},
						GenerateSetterSamples: "true",
						GenerateRpcSamples:    "true",
					},
				},
			},
			want: &config.Library{
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
							{Name: "custom", Package: "custom-pkg"},
							{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
							{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
						},
						DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
						GenerateSetterSamples:   "true",
						GenerateRpcSamples:      "true",
					},
				},
			},
		},
		{
			name: "library overrides default",
			lib: &config.Library{
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
							{Name: "wkt", Package: "custom-wkt"},
						},
						GenerateSetterSamples: "false",
						GenerateRpcSamples:    "false",
					},
				},
			},
			want: &config.Library{
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
							{Name: "wkt", Package: "custom-wkt"},
							{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
						},
						DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
						GenerateSetterSamples:   "false",
						GenerateRpcSamples:      "false",
					},
				},
			},
		},
		{
			name: "preserves existing warnings",
			lib: &config.Library{
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						DisabledRustdocWarnings: []string{"custom_warning"},
					},
				},
			},
			want: &config.Library{
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
							{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
							{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
						},
						DisabledRustdocWarnings: []string{"custom_warning"},
						GenerateSetterSamples:   "true",
						GenerateRpcSamples:      "true",
					},
				},
			},
		},
		{
			name: "module overrides defaults",
			lib: &config.Library{
				Rust: &config.RustCrate{
					Modules: []*config.RustModule{
						{
							GenerateSetterSamples: "false",
							GenerateRpcSamples:    "false",
						},
					},
				},
			},
			want: &config.Library{
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
							{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
							{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
						},
						DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
						GenerateSetterSamples:   "true",
						GenerateRpcSamples:      "true",
					},
					Modules: []*config.RustModule{
						{
							GenerateSetterSamples: "false",
							GenerateRpcSamples:    "false",
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := fillDefaults(test.lib, defaults)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFillDefaults_Python(t *testing.T) {
	for _, test := range []struct {
		name     string
		lib      *config.Library
		defaults *config.PythonDefault
		want     *config.Library
	}{
		{
			name: "common_gapic_paths only in defaults",
			lib:  &config.Library{},
			defaults: &config.PythonDefault{
				CommonGAPICPaths: []string{"a", "b"},
			},
			want: &config.Library{
				Python: &config.PythonPackage{
					PythonDefault: config.PythonDefault{
						CommonGAPICPaths: []string{"a", "b"},
					},
				},
			},
		},
		{
			name: "common_gapic_paths only in package",
			lib: &config.Library{
				Python: &config.PythonPackage{
					PythonDefault: config.PythonDefault{
						CommonGAPICPaths: []string{"a", "b"},
					},
				},
			},
			defaults: &config.PythonDefault{},
			want: &config.Library{
				Python: &config.PythonPackage{
					PythonDefault: config.PythonDefault{
						CommonGAPICPaths: []string{"a", "b"},
					},
				},
			},
		},
		{
			name: "common_gapic_paths merged",
			lib: &config.Library{
				Python: &config.PythonPackage{
					PythonDefault: config.PythonDefault{
						CommonGAPICPaths: []string{"c", "d"},
					},
				},
			},
			defaults: &config.PythonDefault{
				CommonGAPICPaths: []string{"a", "b"},
			},
			want: &config.Library{
				Python: &config.PythonPackage{
					PythonDefault: config.PythonDefault{
						CommonGAPICPaths: []string{"a", "b", "c", "d"},
					},
				},
			},
		},
		{
			name: "library type defaults",
			lib:  &config.Library{},
			defaults: &config.PythonDefault{
				LibraryType: "GAPIC_AUTO",
			},
			want: &config.Library{
				Python: &config.PythonPackage{
					PythonDefault: config.PythonDefault{
						LibraryType: "GAPIC_AUTO",
					},
				},
			},
		},
		{
			name: "library type overridden",
			lib: &config.Library{
				Python: &config.PythonPackage{
					PythonDefault: config.PythonDefault{
						LibraryType: "CORE",
					},
				},
			},
			defaults: &config.PythonDefault{
				LibraryType: "GAPIC_AUTO",
			},
			want: &config.Library{
				Python: &config.PythonPackage{
					PythonDefault: config.PythonDefault{
						LibraryType: "CORE",
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			defaults := &config.Default{
				Python: test.defaults,
			}
			got := fillDefaults(test.lib, defaults)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPrepareLibrary(t *testing.T) {
	for _, test := range []struct {
		name        string
		language    string
		output      string
		rust        *config.RustCrate
		apis        []*config.API
		wantOutput  string
		wantErr     bool
		wantAPIPath string
	}{
		{
			name:       "empty output derives path from api",
			language:   config.LanguageRust,
			apis:       []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			wantOutput: "src/generated/cloud/secretmanager/v1",
		},
		{
			name:       "explicit output keeps explicit path",
			language:   config.LanguageRust,
			output:     "custom/output",
			apis:       []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			wantOutput: "custom/output",
		},
		{
			name:       "empty output uses default for non-rust",
			language:   config.LanguageGo,
			apis:       []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			wantOutput: "src/generated/google-cloud-secretmanager-v1",
		},
		{
			name:        "rust with no apis creates default and derives path",
			language:    config.LanguageRust,
			apis:        nil,
			wantOutput:  "src/generated/cloud/secretmanager/v1",
			wantAPIPath: "google/cloud/secretmanager/v1",
		},
		{
			name:       "veneer rust with no apis does not derive path",
			language:   config.LanguageRust,
			output:     "src/storage/test/v1",
			rust:       &config.RustCrate{Modules: []*config.RustModule{{APIPath: "google/storage/v2"}}},
			apis:       nil,
			wantOutput: "src/storage/test/v1",
		},
		{
			name:     "veneer without output returns error",
			language: config.LanguageRust,
			rust:     &config.RustCrate{Modules: []*config.RustModule{{APIPath: "google/storage/v2"}}},
			wantErr:  true,
		},
		{
			name:       "veneer with explicit output succeeds",
			language:   config.LanguageRust,
			rust:       &config.RustCrate{Modules: []*config.RustModule{{APIPath: "google/storage/v2"}}},
			output:     "src/storage",
			wantOutput: "src/storage",
		},
		{
			name:        "rust lib without service config",
			language:    config.LanguageRust,
			apis:        []*config.API{{Path: "google/cloud/orgpolicy/v1"}},
			wantOutput:  "src/generated/cloud/orgpolicy/v1",
			wantAPIPath: "google/cloud/orgpolicy/v1",
		},
		{
			name:       "Go lib without api path",
			language:   config.LanguageGo,
			wantOutput: "src/generated/google-cloud-secretmanager-v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			lib := &config.Library{
				Name:   "google-cloud-secretmanager-v1",
				Output: test.output,
				APIs:   test.apis,
				Rust:   test.rust,
			}
			defaults := &config.Default{
				Output: "src/generated",
			}
			got, err := applyDefaults(test.language, lib, defaults)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Output != test.wantOutput {
				t.Errorf("got output %q, want %q", got.Output, test.wantOutput)
			}
			if len(got.APIs) > 0 {
				ch := got.APIs[0]
				if test.wantAPIPath != "" && ch.Path != test.wantAPIPath {
					t.Errorf("got %q, want %q", ch.Path, test.wantAPIPath)
				}
			}
		})
	}
}

func TestCanDeriveAPIPath(t *testing.T) {
	for _, test := range []struct {
		name     string
		language string
		want     bool
	}{
		{
			name:     "dart",
			language: config.LanguageDart,
			want:     true,
		},
		{
			name:     "go",
			language: config.LanguageGo,
		},
		{
			name:     "python",
			language: config.LanguagePython,
		},
		{
			name:     "rust",
			language: config.LanguageRust,
			want:     true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := canDeriveAPIPath(test.language)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsVeneer(t *testing.T) {
	for _, test := range []struct {
		name     string
		language string
		lib      *config.Library
		want     bool
	}{
		{
			name:     "rust is veneer",
			language: config.LanguageRust,
			lib: &config.Library{
				Rust: &config.RustCrate{
					Modules: []*config.RustModule{{APIPath: "google/storage/v2"}},
				},
			},
			want: true,
		},
		{
			name:     "rust is not veneer",
			language: config.LanguageRust,
			lib:      &config.Library{},
			want:     false,
		},
		{
			name:     "nodejs handwritten tool is veneer",
			language: config.LanguageNodejs,
			lib: &config.Library{
				Output: "packages/typeless-sample-bot",
				APIs:   nil,
			},
			want: true,
		},
		{
			name:     "nodejs gapic lib is not veneer",
			language: config.LanguageNodejs,
			lib: &config.Library{
				Output: "packages/gapic-lib",
				APIs:   []*config.API{{Path: "google/example/v1"}},
			},
			want: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := isVeneer(test.language, test.lib); got != test.want {
				t.Errorf("isVeneer(%q, %+v) = %v, want %v", test.language, test.lib, got, test.want)
			}
		})
	}
}

func TestResolvePreview(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want *config.Library
	}{
		{
			name: "nil lib returns nil",
			lib:  nil,
			want: nil,
		},
		{
			name: "no preview returns nil",
			lib:  &config.Library{Name: "foo"},
			want: nil,
		},
		{
			name: "overrides all supported fields",
			lib: &config.Library{
				Name:                "base-name",
				Version:             "1.0.0",
				CopyrightYear:       "2024",
				DescriptionOverride: "base desc",
				Keep:                []string{"base-keep"},
				Output:              "base-out",
				Roots:               []string{"base-root"},
				SkipGenerate:        false,
				SkipRelease:         false,
				SpecificationFormat: "protobuf",
				Go: &config.GoModule{
					ModulePathVersion: "v1",
				},
				Preview: &config.Library{
					Name:                "preview-name",
					Version:             "1.1.0-alpha",
					APIs:                []*config.API{{Path: "preview/api"}},
					CopyrightYear:       "2025",
					DescriptionOverride: "preview desc",
					Keep:                []string{"preview-keep"},
					Output:              "preview-out",
					Roots:               []string{"preview-root"},
					SkipGenerate:        true,
					SkipRelease:         true,
					SpecificationFormat: "discovery",
					Go: &config.GoModule{
						NestedModule: "v2",
					},
				},
			},
			want: &config.Library{
				Name:                "preview-name",
				Version:             "1.1.0-alpha",
				APIs:                []*config.API{{Path: "preview/api"}},
				CopyrightYear:       "2025",
				DescriptionOverride: "preview desc",
				Keep:                []string{"preview-keep"},
				Output:              "preview-out",
				Roots:               []string{"preview-root"},
				SkipGenerate:        true,
				SkipRelease:         true,
				SpecificationFormat: "discovery",
				Go: &config.GoModule{
					ModulePathVersion: "v1",
					NestedModule:      "v2",
				},
				Preview: nil,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := ResolvePreview(test.lib)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolvePreview_NoMutation(t *testing.T) {
	lib := &config.Library{
		Name: "base",
		Keep: []string{"base-keep"},
		APIs: []*config.API{{Path: "base-api"}},
		Go: &config.GoModule{
			ModulePathVersion: "v1",
		},
		Preview: &config.Library{
			Keep: []string{"preview-keep"},
			APIs: []*config.API{{Path: "preview-api"}},
			Go: &config.GoModule{
				NestedModule: "v2",
			},
		},
	}

	want := *lib

	_ = ResolvePreview(lib)

	if diff := cmp.Diff(want, *lib); diff != "" {
		t.Errorf("ResolvePreview mutated the input library (-want +got):\n%s", diff)
	}
}

func TestMergeDotnet(t *testing.T) {
	for _, test := range []struct {
		name string
		dst  *config.DotnetPackage
		src  *config.DotnetPackage
		want *config.DotnetPackage
	}{
		{
			name: "nil src returns dst",
			dst:  &config.DotnetPackage{Generator: "foo"},
			src:  nil,
			want: &config.DotnetPackage{Generator: "foo"},
		},
		{
			name: "nil dst returns src",
			dst:  nil,
			src:  &config.DotnetPackage{Generator: "bar"},
			want: &config.DotnetPackage{Generator: "bar"},
		},
		{
			name: "merges all fields",
			dst: &config.DotnetPackage{
				Generator: "foo",
				Csproj: &config.DotnetCsproj{
					Snippets: &config.DotnetCsprojSnippets{
						EmbeddedResources: []string{"base-res"},
					},
					IntegrationTests: &config.DotnetCsprojSnippets{
						EmbeddedResources: []string{"base-test-res"},
					},
				},
			},
			src: &config.DotnetPackage{
				AdditionalServiceDescriptors: []string{"desc"},
				Csproj: &config.DotnetCsproj{
					Snippets: &config.DotnetCsprojSnippets{
						EmbeddedResources: []string{"new-res"},
					},
					IntegrationTests: &config.DotnetCsprojSnippets{
						EmbeddedResources: []string{"new-test-res"},
					},
				},
				Dependencies: map[string]string{"dep": "v1"},
				Generator:    "bar",
				PackageGroup: []string{"group"},
				Postgeneration: []*config.DotnetPostgeneration{
					{Run: "post"},
				},
				Pregeneration: []*config.DotnetPregeneration{
					{RenameMessage: &config.DotnetRenameMessage{From: "A", To: "B"}},
				},
			},
			want: &config.DotnetPackage{
				AdditionalServiceDescriptors: []string{"desc"},
				Csproj: &config.DotnetCsproj{
					Snippets: &config.DotnetCsprojSnippets{
						EmbeddedResources: []string{"new-res"},
					},
					IntegrationTests: &config.DotnetCsprojSnippets{
						EmbeddedResources: []string{"new-test-res"},
					},
				},
				Dependencies: map[string]string{"dep": "v1"},
				Generator:    "bar",
				PackageGroup: []string{"group"},
				Postgeneration: []*config.DotnetPostgeneration{
					{Run: "post"},
				},
				Pregeneration: []*config.DotnetPregeneration{
					{RenameMessage: &config.DotnetRenameMessage{From: "A", To: "B"}},
				},
			},
		},
		{
			name: "nested snippets nil branch coverage",
			dst: &config.DotnetPackage{
				Csproj: &config.DotnetCsproj{
					Snippets: &config.DotnetCsprojSnippets{EmbeddedResources: []string{"res"}},
				},
			},
			src: &config.DotnetPackage{
				Csproj: &config.DotnetCsproj{
					IntegrationTests: &config.DotnetCsprojSnippets{EmbeddedResources: []string{"test"}},
				},
			},
			want: &config.DotnetPackage{
				Csproj: &config.DotnetCsproj{
					Snippets:         &config.DotnetCsprojSnippets{EmbeddedResources: []string{"res"}},
					IntegrationTests: &config.DotnetCsprojSnippets{EmbeddedResources: []string{"test"}},
				},
			},
		},
		{
			name: "nested src snippets nil branch coverage",
			dst: &config.DotnetPackage{
				Csproj: &config.DotnetCsproj{
					IntegrationTests: &config.DotnetCsprojSnippets{EmbeddedResources: []string{"test"}},
				},
			},
			src: &config.DotnetPackage{
				Csproj: &config.DotnetCsproj{
					Snippets: &config.DotnetCsprojSnippets{EmbeddedResources: []string{"res"}},
				},
			},
			want: &config.DotnetPackage{
				Csproj: &config.DotnetCsproj{
					Snippets:         &config.DotnetCsprojSnippets{EmbeddedResources: []string{"res"}},
					IntegrationTests: &config.DotnetCsprojSnippets{EmbeddedResources: []string{"test"}},
				},
			},
		},
		{
			name: "embedded resources nil branch coverage",
			dst: &config.DotnetPackage{
				Csproj: &config.DotnetCsproj{
					Snippets: &config.DotnetCsprojSnippets{EmbeddedResources: []string{"res"}},
				},
			},
			src: &config.DotnetPackage{
				Csproj: &config.DotnetCsproj{
					Snippets: &config.DotnetCsprojSnippets{},
				},
			},
			want: &config.DotnetPackage{
				Csproj: &config.DotnetCsproj{
					Snippets: &config.DotnetCsprojSnippets{EmbeddedResources: []string{"res"}},
				},
			},
		},
		{
			name: "src.Csproj nil branch coverage",
			dst: &config.DotnetPackage{
				Csproj: &config.DotnetCsproj{},
			},
			src: &config.DotnetPackage{},
			want: &config.DotnetPackage{
				Csproj: &config.DotnetCsproj{},
			},
		},
		{
			name: "dst.Csproj nil branch coverage",
			dst:  &config.DotnetPackage{},
			src: &config.DotnetPackage{
				Csproj: &config.DotnetCsproj{},
			},
			want: &config.DotnetPackage{
				Csproj: &config.DotnetCsproj{},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := mergeDotnet(test.dst, test.src)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMergeDart(t *testing.T) {
	for _, test := range []struct {
		name string
		dst  *config.DartPackage
		src  *config.DartPackage
		want *config.DartPackage
	}{
		{
			name: "nil src returns dst",
			dst:  &config.DartPackage{Version: "1.0.0"},
			src:  nil,
			want: &config.DartPackage{Version: "1.0.0"},
		},
		{
			name: "nil dst returns src",
			dst:  nil,
			src:  &config.DartPackage{Version: "2.0.0"},
			want: &config.DartPackage{Version: "2.0.0"},
		},
		{
			name: "merges all fields",
			dst: &config.DartPackage{
				Version: "1.0.0",
			},
			src: &config.DartPackage{
				APIKeysEnvironmentVariables: "KEYS",
				Dependencies:                "deps",
				DevDependencies:             "dev-deps",
				ExtraImports:                "imports",
				IncludeList:                 []string{"proto"},
				IssueTrackerURL:             "url",
				LibraryPathOverride:         "path",
				NameOverride:                "name",
				Packages:                    map[string]string{"p": "v"},
				PartFile:                    "part",
				Prefixes:                    map[string]string{"pre": "val"},
				Protos:                      map[string]string{"pro": "path"},
				ReadmeAfterTitleText:        "after",
				ReadmeQuickstartText:        "quick",
				RepositoryURL:               "repo",
				TitleOverride:               "title",
				Version:                     "2.0.0",
			},
			want: &config.DartPackage{
				APIKeysEnvironmentVariables: "KEYS",
				Dependencies:                "deps",
				DevDependencies:             "dev-deps",
				ExtraImports:                "imports",
				IncludeList:                 []string{"proto"},
				IssueTrackerURL:             "url",
				LibraryPathOverride:         "path",
				NameOverride:                "name",
				Packages:                    map[string]string{"p": "v"},
				PartFile:                    "part",
				Prefixes:                    map[string]string{"pre": "val"},
				Protos:                      map[string]string{"pro": "path"},
				ReadmeAfterTitleText:        "after",
				ReadmeQuickstartText:        "quick",
				RepositoryURL:               "repo",
				TitleOverride:               "title",
				Version:                     "2.0.0",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := mergeDart(test.dst, test.src)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMergeGo(t *testing.T) {
	for _, test := range []struct {
		name string
		dst  *config.GoModule
		src  *config.GoModule
		want *config.GoModule
	}{
		{
			name: "nil src returns dst",
			dst:  &config.GoModule{ModulePathVersion: "v1"},
			src:  nil,
			want: &config.GoModule{ModulePathVersion: "v1"},
		},
		{
			name: "nil dst returns src",
			dst:  nil,
			src:  &config.GoModule{ModulePathVersion: "v2"},
			want: &config.GoModule{ModulePathVersion: "v2"},
		},
		{
			name: "merges all fields",
			dst:  &config.GoModule{ModulePathVersion: "v1"},
			src: &config.GoModule{
				DeleteGenerationOutputPaths: []string{"p"},
				GoAPIs:                      []*config.GoAPI{{Path: "foo"}},
				ModulePathVersion:           "v2",
				NestedModule:                "nested",
			},
			want: &config.GoModule{
				DeleteGenerationOutputPaths: []string{"p"},
				GoAPIs:                      []*config.GoAPI{{Path: "foo"}},
				ModulePathVersion:           "v2",
				NestedModule:                "nested",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := mergeGo(test.dst, test.src)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMergeJava(t *testing.T) {
	for _, test := range []struct {
		name string
		dst  *config.JavaModule
		src  *config.JavaModule
		want *config.JavaModule
	}{
		{
			name: "nil src returns dst",
			dst:  &config.JavaModule{GroupID: "com.google"},
			src:  nil,
			want: &config.JavaModule{GroupID: "com.google"},
		},
		{
			name: "nil dst returns src",
			dst:  nil,
			src:  &config.JavaModule{GroupID: "com.other"},
			want: &config.JavaModule{GroupID: "com.other"},
		},
		{
			name: "merges all fields",
			dst:  &config.JavaModule{GroupID: "com.google"},
			src: &config.JavaModule{
				APIIDOverride:                "id",
				APIReference:                 "ref",
				APIDescriptionOverride:       "desc",
				ClientDocumentationOverride:  "doc",
				NonCloudAPI:                  true,
				CodeownerTeam:                "team",
				DistributionNameOverride:     "dist",
				ExcludedDependencies:         "ex-dep",
				ExcludedPOMs:                 "ex-pom",
				ExtraVersionedModules:        "extra",
				GroupID:                      "com.new",
				IssueTrackerOverride:         "issue",
				LibrariesBOMVersion:          "bom",
				LibraryTypeOverride:          "type",
				MinJavaVersion:               11,
				NamePrettyOverride:           "pretty",
				JavaAPIs:                     []*config.JavaAPI{{Path: "p"}},
				ProductDocumentationOverride: "prod-doc",
				RecommendedPackage:           "rec",
				BillingNotRequired:           true,
				RestDocumentation:            "rest",
				RpcDocumentation:             "rpc",
			},
			want: &config.JavaModule{
				APIIDOverride:                "id",
				APIReference:                 "ref",
				APIDescriptionOverride:       "desc",
				ClientDocumentationOverride:  "doc",
				NonCloudAPI:                  true,
				CodeownerTeam:                "team",
				DistributionNameOverride:     "dist",
				ExcludedDependencies:         "ex-dep",
				ExcludedPOMs:                 "ex-pom",
				ExtraVersionedModules:        "extra",
				GroupID:                      "com.new",
				IssueTrackerOverride:         "issue",
				LibrariesBOMVersion:          "bom",
				LibraryTypeOverride:          "type",
				MinJavaVersion:               11,
				NamePrettyOverride:           "pretty",
				JavaAPIs:                     []*config.JavaAPI{{Path: "p"}},
				ProductDocumentationOverride: "prod-doc",
				RecommendedPackage:           "rec",
				BillingNotRequired:           true,
				RestDocumentation:            "rest",
				RpcDocumentation:             "rpc",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := mergeJava(test.dst, test.src)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMergeNodejs(t *testing.T) {
	for _, test := range []struct {
		name string
		dst  *config.NodejsPackage
		src  *config.NodejsPackage
		want *config.NodejsPackage
	}{
		{
			name: "nil src returns dst",
			dst:  &config.NodejsPackage{PackageName: "foo"},
			src:  nil,
			want: &config.NodejsPackage{PackageName: "foo"},
		},
		{
			name: "nil dst returns src",
			dst:  nil,
			src:  &config.NodejsPackage{PackageName: "bar"},
			want: &config.NodejsPackage{PackageName: "bar"},
		},
		{
			name: "merges all fields",
			dst:  &config.NodejsPackage{PackageName: "foo"},
			src: &config.NodejsPackage{
				BundleConfig:          "bundle",
				Dependencies:          map[string]string{"d": "v"},
				ExtraProtocParameters: []string{"p"},
				HandwrittenLayer:      true,
				MainService:           "service",
				Mixins:                "mixin",
				PackageName:           "bar",
			},
			want: &config.NodejsPackage{
				BundleConfig:          "bundle",
				Dependencies:          map[string]string{"d": "v"},
				ExtraProtocParameters: []string{"p"},
				HandwrittenLayer:      true,
				MainService:           "service",
				Mixins:                "mixin",
				PackageName:           "bar",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := mergeNodejs(test.dst, test.src)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMergePython(t *testing.T) {
	for _, test := range []struct {
		name string
		dst  *config.PythonPackage
		src  *config.PythonPackage
		want *config.PythonPackage
	}{
		{
			name: "nil src returns dst",
			dst:  &config.PythonPackage{PythonDefault: config.PythonDefault{LibraryType: "GAPIC"}},
			src:  nil,
			want: &config.PythonPackage{PythonDefault: config.PythonDefault{LibraryType: "GAPIC"}},
		},
		{
			name: "nil dst returns src",
			dst:  nil,
			src:  &config.PythonPackage{PythonDefault: config.PythonDefault{LibraryType: "GAPIC"}},
			want: &config.PythonPackage{PythonDefault: config.PythonDefault{LibraryType: "GAPIC"}},
		},
		{
			name: "merges all fields",
			dst:  &config.PythonPackage{PythonDefault: config.PythonDefault{LibraryType: "GAPIC"}},
			src: &config.PythonPackage{
				PythonDefault: config.PythonDefault{
					CommonGAPICPaths: []string{"p"},
					LibraryType:      "NEW",
				},
				OptArgsByAPI:                 map[string][]string{"a": {"o"}},
				ProtoOnlyAPIs:                []string{"proto"},
				NamePrettyOverride:           "pretty",
				ProductDocumentationOverride: "prod-doc",
				APIShortnameOverride:         "short",
				APIIDOverride:                "id",
				ClientDocumentationOverride:  "client-doc",
				IssueTrackerOverride:         "issue",
				MetadataNameOverride:         "meta",
				DefaultVersion:               "v1",
				SkipReadmeCopy:               true,
			},
			want: &config.PythonPackage{
				PythonDefault: config.PythonDefault{
					CommonGAPICPaths: []string{"p"},
					LibraryType:      "NEW",
				},
				OptArgsByAPI:                 map[string][]string{"a": {"o"}},
				ProtoOnlyAPIs:                []string{"proto"},
				NamePrettyOverride:           "pretty",
				ProductDocumentationOverride: "prod-doc",
				APIShortnameOverride:         "short",
				APIIDOverride:                "id",
				ClientDocumentationOverride:  "client-doc",
				IssueTrackerOverride:         "issue",
				MetadataNameOverride:         "meta",
				DefaultVersion:               "v1",
				SkipReadmeCopy:               true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := mergePython(test.dst, test.src)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMergeRust(t *testing.T) {
	detailedTracing := true
	resourceHeuristic := false
	for _, test := range []struct {
		name string
		dst  *config.RustCrate
		src  *config.RustCrate
		want *config.RustCrate
	}{
		{
			name: "nil src returns dst",
			dst:  &config.RustCrate{PackageNameOverride: "foo"},
			src:  nil,
			want: &config.RustCrate{PackageNameOverride: "foo"},
		},
		{
			name: "nil dst returns src",
			dst:  nil,
			src:  &config.RustCrate{PackageNameOverride: "bar"},
			want: &config.RustCrate{PackageNameOverride: "bar"},
		},
		{
			name: "merges all fields",
			dst: &config.RustCrate{
				PackageNameOverride: "foo",
				Discovery: &config.RustDiscovery{
					OperationID: "op-old",
				},
			},
			src: &config.RustCrate{
				RustDefault: config.RustDefault{
					PackageDependencies:       []*config.RustPackageDependency{{Name: "dep"}},
					DisabledRustdocWarnings:   []string{"w"},
					GenerateSetterSamples:     "true",
					GenerateRpcSamples:        "true",
					DetailedTracingAttributes: &detailedTracing,
					ResourceNameHeuristic:     &resourceHeuristic,
				},
				Modules:                   []*config.RustModule{{Output: "out"}},
				PerServiceFeatures:        true,
				ModulePath:                "path",
				TemplateOverride:          "temp",
				PackageNameOverride:       "bar",
				RootName:                  "root",
				DefaultFeatures:           []string{"f"},
				IncludeList:               []string{"inc"},
				IncludedIds:               []string{"iid"},
				SkippedIds:                []string{"sid"},
				DisabledClippyWarnings:    []string{"clip"},
				HasVeneer:                 true,
				RoutingRequired:           true,
				IncludeGrpcOnlyMethods:    true,
				PostProcessProtos:         "post",
				DocumentationOverrides:    []config.RustDocumentationOverride{{ID: "id"}},
				PaginationOverrides:       []config.RustPaginationOverride{{ID: "pid"}},
				NameOverrides:             "name",
				Discovery:                 &config.RustDiscovery{OperationID: "op-new", Pollers: []config.RustPoller{{Prefix: "pre"}}},
				QuickstartServiceOverride: "quick",
			},
			want: &config.RustCrate{
				RustDefault: config.RustDefault{
					PackageDependencies:       []*config.RustPackageDependency{{Name: "dep"}},
					DisabledRustdocWarnings:   []string{"w"},
					GenerateSetterSamples:     "true",
					GenerateRpcSamples:        "true",
					DetailedTracingAttributes: &detailedTracing,
					ResourceNameHeuristic:     &resourceHeuristic,
				},
				Modules:                   []*config.RustModule{{Output: "out"}},
				PerServiceFeatures:        true,
				ModulePath:                "path",
				TemplateOverride:          "temp",
				PackageNameOverride:       "bar",
				RootName:                  "root",
				DefaultFeatures:           []string{"f"},
				IncludeList:               []string{"inc"},
				IncludedIds:               []string{"iid"},
				SkippedIds:                []string{"sid"},
				DisabledClippyWarnings:    []string{"clip"},
				HasVeneer:                 true,
				RoutingRequired:           true,
				IncludeGrpcOnlyMethods:    true,
				PostProcessProtos:         "post",
				DocumentationOverrides:    []config.RustDocumentationOverride{{ID: "id"}},
				PaginationOverrides:       []config.RustPaginationOverride{{ID: "pid"}},
				NameOverrides:             "name",
				Discovery:                 &config.RustDiscovery{OperationID: "op-new", Pollers: []config.RustPoller{{Prefix: "pre"}}},
				QuickstartServiceOverride: "quick",
			},
		},
		{
			name: "src discovery fields nil branch coverage",
			dst: &config.RustCrate{
				Discovery: &config.RustDiscovery{
					OperationID: "op",
					Pollers:     []config.RustPoller{{Prefix: "p"}},
				},
			},
			src: &config.RustCrate{
				Discovery: &config.RustDiscovery{},
			},
			want: &config.RustCrate{
				Discovery: &config.RustDiscovery{
					OperationID: "op",
					Pollers:     []config.RustPoller{{Prefix: "p"}},
				},
			},
		},
		{
			name: "src.Discovery nil branch coverage",
			dst: &config.RustCrate{
				Discovery: &config.RustDiscovery{},
			},
			src: &config.RustCrate{},
			want: &config.RustCrate{
				Discovery: &config.RustDiscovery{},
			},
		},
		{
			name: "dst.Discovery nil branch coverage",
			dst:  &config.RustCrate{},
			src: &config.RustCrate{
				Discovery: &config.RustDiscovery{},
			},
			want: &config.RustCrate{
				Discovery: &config.RustDiscovery{},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := mergeRust(test.dst, test.src)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
