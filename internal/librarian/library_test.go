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
		},
	}
	for _, test := range []struct {
		name string
		lib  *config.Library
		want *config.Library
	}{
		{
			name: "fills rust defaults",
			lib:  &config.Library{},
			want: &config.Library{
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
							{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
							{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
						},
						DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
						GenerateSetterSamples:   "true",
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

func TestMergePackageDependencies(t *testing.T) {
	defaults := []*config.RustPackageDependency{
		{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
		{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
	}
	for _, test := range []struct {
		name string
		lib  []*config.RustPackageDependency
		want []*config.RustPackageDependency
	}{
		{
			name: "no lib dependencies",
			lib:  nil,
			want: []*config.RustPackageDependency{
				{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
				{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
			},
		},
		{
			name: "lib with one dependency",
			lib: []*config.RustPackageDependency{
				{Name: "custom", Package: "custom-pkg"},
			},
			want: []*config.RustPackageDependency{
				{Name: "custom", Package: "custom-pkg"},
				{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
				{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
			},
		},
		{
			name: "lib overrides default",
			lib: []*config.RustPackageDependency{
				{Name: "wkt", Package: "custom-wkt"},
			},
			want: []*config.RustPackageDependency{
				{Name: "wkt", Package: "custom-wkt"},
				{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := mergePackageDependencies(defaults, test.lib)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}