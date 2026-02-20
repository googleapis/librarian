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
		Keep:         []string{"CHANGES.md"},
		Output:       "src/generated/",
		ReleaseLevel: "stable",
		Transport:    "grpc+rest",
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
				Keep:         []string{"CHANGES.md"},
				Output:       "src/generated/",
				ReleaseLevel: "stable",
				Transport:    "grpc+rest",
			},
		},
		{
			name:     "preserves existing values",
			defaults: defaults,
			lib: &config.Library{
				Output:       "custom/output/",
				ReleaseLevel: "preview",
				Transport:    "grpc+rest",
			},
			want: &config.Library{
				Keep:         []string{"CHANGES.md"},
				Output:       "custom/output/",
				ReleaseLevel: "preview",
				Transport:    "grpc+rest",
			},
		},
		{
			name:     "partial fill",
			defaults: defaults,
			lib:      &config.Library{Output: "custom/output/"},
			want: &config.Library{
				Keep:         []string{"CHANGES.md"},
				Output:       "custom/output/",
				ReleaseLevel: "stable",
				Transport:    "grpc+rest",
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
			name: "java defaults",
			defaults: &config.Default{
				Java: &config.JavaDefault{
					FormatterJar: "default-formatter.jar",
					GeneratorJar: "default-generator.jar",
					GRPCPlugin:   "default-grpc-plugin",
				},
			},
			lib: &config.Library{Name: "test-lib"},
			want: &config.Library{
				Name: "test-lib",
				Java: &config.JavaPackage{
					JavaDefault: config.JavaDefault{
						FormatterJar: "default-formatter.jar",
						GeneratorJar: "default-generator.jar",
						GRPCPlugin:   "default-grpc-plugin",
					},
				},
			},
		},
		{
			name: "java defaults do not override library params",
			defaults: &config.Default{
				Java: &config.JavaDefault{
					FormatterJar: "default-formatter.jar",
					GeneratorJar: "default-generator.jar",
					GRPCPlugin:   "default-grpc-plugin",
				},
			},
			lib: &config.Library{
				Name: "test-lib",
				Java: &config.JavaPackage{
					JavaDefault: config.JavaDefault{
						FormatterJar: "custom-formatter.jar",
					},
				},
			},
			want: &config.Library{
				Name: "test-lib",
				Java: &config.JavaPackage{
					JavaDefault: config.JavaDefault{
						FormatterJar: "custom-formatter.jar",
						GeneratorJar: "default-generator.jar",
						GRPCPlugin:   "default-grpc-plugin",
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
		veneer      bool
		apis        []*config.API
		wantOutput  string
		wantErr     bool
		wantAPIPath string
	}{
		{
			name:       "empty output derives path from api",
			language:   "rust",
			apis:       []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			wantOutput: "src/generated/cloud/secretmanager/v1",
		},
		{
			name:       "explicit output keeps explicit path",
			language:   "rust",
			output:     "custom/output",
			apis:       []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			wantOutput: "custom/output",
		},
		{
			name:       "empty output uses default for non-rust",
			language:   "go",
			apis:       []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			wantOutput: "src/generated",
		},
		{
			name:        "rust with no apis creates default and derives path",
			language:    "rust",
			apis:        nil,
			wantOutput:  "src/generated/cloud/secretmanager/v1",
			wantAPIPath: "google/cloud/secretmanager/v1",
		},
		{
			name:        "veneer rust with no apis does not derive path",
			language:    "rust",
			output:      "src/storage/test/v1",
			veneer:      true,
			apis:        nil,
			wantOutput:  "src/storage/test/v1",
			wantAPIPath: "",
		},
		{
			name:    "veneer without output returns error",
			veneer:  true,
			wantErr: true,
		},
		{
			name:       "veneer with explicit output succeeds",
			veneer:     true,
			output:     "src/storage",
			wantOutput: "src/storage",
		},
		{
			name:        "rust lib without service config",
			language:    "rust",
			apis:        []*config.API{{Path: "google/cloud/orgpolicy/v1"}},
			wantOutput:  "src/generated/cloud/orgpolicy/v1",
			wantAPIPath: "google/cloud/orgpolicy/v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			lib := &config.Library{
				Name:   "google-cloud-secretmanager-v1",
				Output: test.output,
				Veneer: test.veneer,
				APIs:   test.apis,
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
