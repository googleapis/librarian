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

package config

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestRead(t *testing.T) {
	got, err := yaml.Read[Config]("testdata/rust/librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := &Config{
		Language: "rust",
		Sources: &Sources{
			Discovery: &Source{
				Commit: "b27c80574e918a7e2a36eb21864d1d2e45b8c032",
				SHA256: "67c8d3792f0ebf5f0582dce675c379d0f486604eb0143814c79e788954aa1212",
			},
			Googleapis: &Source{
				Commit: "9fcfbea0aa5b50fa22e190faceb073d74504172b",
				SHA256: "81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98",
			},
		},
		Default: &Default{
			Output:       "src/generated/",
			ReleaseLevel: "stable",
			TagFormat:    "{name}/v{version}",
			Rust: &RustDefault{
				DisabledRustdocWarnings: []string{
					"redundant_explicit_links",
					"broken_intra_doc_links",
				},
				PackageDependencies: []*RustPackageDependency{
					{Name: "bytes", Package: "bytes", ForceUsed: true},
					{Name: "serde", Package: "serde", ForceUsed: true},
				},
			},
		},
		Libraries: []*Library{
			{
				Name:    "google-cloud-secretmanager-v1",
				Version: "1.2.3",
				Channels: []*Channel{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			{
				Name:    "google-cloud-storage-v2",
				Version: "2.3.4",
				Channels: []*Channel{
					{Path: "google/cloud/storage/v2"},
				},
				Rust: &RustCrate{
					RustDefault: RustDefault{
						DisabledRustdocWarnings: []string{"rustdoc::bare_urls"},
					},
				},
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestWrite(t *testing.T) {
	want, err := yaml.Read[Config]("testdata/rust/librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}
	data, err := yaml.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := yaml.Unmarshal[Config](data)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFill(t *testing.T) {
	defaults := &Default{
		Output:       "src/generated/",
		ReleaseLevel: "stable",
		Transport:    "default_transport",
	}
	for _, test := range []struct {
		name     string
		defaults *Default
		lib      *Library
		want     *Library
	}{
		{
			name:     "fills empty fields",
			defaults: defaults,
			lib:      &Library{},
			want: &Library{
				Output:       "src/generated/",
				ReleaseLevel: "stable",
				Transport:    "default_transport",
			},
		},
		{
			name:     "preserves existing values",
			defaults: defaults,
			lib: &Library{
				Output:       "custom/output/",
				ReleaseLevel: "preview",
				Transport:    "custom_transport",
			},
			want: &Library{
				Output:       "custom/output/",
				ReleaseLevel: "preview",
				Transport:    "custom_transport",
			},
		},
		{
			name:     "partial fill",
			defaults: defaults,
			lib:      &Library{Output: "custom/output/"},
			want: &Library{
				Output:       "custom/output/",
				ReleaseLevel: "stable",
				Transport:    "default_transport",
			},
		},
		{
			name:     "nil defaults",
			defaults: nil,
			lib:      &Library{Output: "foo/"},
			want:     &Library{Output: "foo/"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.lib.Fill(test.defaults)
			if diff := cmp.Diff(test.want, test.lib); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFill_Rust(t *testing.T) {
	defaults := &Default{
		Rust: &RustDefault{
			PackageDependencies: []*RustPackageDependency{
				{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
				{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
			},
			DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
		},
	}
	for _, test := range []struct {
		name string
		lib  *Library
		want *Library
	}{
		{
			name: "fills rust defaults",
			lib:  &Library{},
			want: &Library{
				Rust: &RustCrate{
					RustDefault: RustDefault{
						PackageDependencies: []*RustPackageDependency{
							{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
							{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
						},
						DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
					},
				},
			},
		},
		{
			name: "merges package dependencies",
			lib: &Library{
				Rust: &RustCrate{
					RustDefault: RustDefault{
						PackageDependencies: []*RustPackageDependency{
							{Name: "custom", Package: "custom-pkg"},
						},
					},
				},
			},
			want: &Library{
				Rust: &RustCrate{
					RustDefault: RustDefault{
						PackageDependencies: []*RustPackageDependency{
							{Name: "custom", Package: "custom-pkg"},
							{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
							{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
						},
						DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
					},
				},
			},
		},
		{
			name: "library overrides default",
			lib: &Library{
				Rust: &RustCrate{
					RustDefault: RustDefault{
						PackageDependencies: []*RustPackageDependency{
							{Name: "wkt", Package: "custom-wkt"},
						},
					},
				},
			},
			want: &Library{
				Rust: &RustCrate{
					RustDefault: RustDefault{
						PackageDependencies: []*RustPackageDependency{
							{Name: "wkt", Package: "custom-wkt"},
							{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
						},
						DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
					},
				},
			},
		},
		{
			name: "preserves existing warnings",
			lib: &Library{
				Rust: &RustCrate{
					RustDefault: RustDefault{
						DisabledRustdocWarnings: []string{"custom_warning"},
					},
				},
			},
			want: &Library{
				Rust: &RustCrate{
					RustDefault: RustDefault{
						PackageDependencies: []*RustPackageDependency{
							{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
							{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
						},
						DisabledRustdocWarnings: []string{"custom_warning"},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.lib.Fill(defaults)
			if diff := cmp.Diff(test.want, test.lib); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLibraryByName(t *testing.T) {
	for _, test := range []struct {
		name        string
		libraryName string
		config      *Config
		want        *Library
		wantErr     error
	}{
		{
			name:        "find_a_library",
			libraryName: "example-library",
			config: &Config{
				Libraries: []*Library{
					{
						Name: "example-library",
					},
					{
						Name: "another-library",
					},
				},
			},
			want: &Library{
				Name: "example-library",
			},
		},
		{
			name:        "no_library_in_config",
			libraryName: "example-library",
			config:      &Config{},
			wantErr:     errLibraryNotFound,
		},
		{
			name:        "does_not_find_a_library",
			libraryName: "non-existent-library",
			config: &Config{
				Libraries: []*Library{
					{
						Name: "example-library",
					},
					{
						Name: "another-library",
					},
				},
			},
			wantErr: errLibraryNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.config.LibraryByName(test.libraryName)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("got error %v, want %v", err, test.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("LibraryByName(%q): %v", test.libraryName, err)
				return
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
