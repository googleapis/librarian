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

func TestPrepareLibrary(t *testing.T) {
	for _, test := range []struct {
		name            string
		language        string
		output          string
		veneer          bool
		channels        []*config.Channel
		wantOutput      string
		wantErr         bool
		wantChannelPath string
	}{
		{
			name:       "empty output derives path from channel",
			language:   "rust",
			channels:   []*config.Channel{{Path: "google/cloud/secretmanager/v1"}},
			wantOutput: "src/generated/cloud/secretmanager/v1",
		},
		{
			name:       "explicit output keeps explicit path",
			language:   "rust",
			output:     "custom/output",
			channels:   []*config.Channel{{Path: "google/cloud/secretmanager/v1"}},
			wantOutput: "custom/output",
		},
		{
			name:       "empty output uses default for non-rust",
			language:   "go",
			channels:   []*config.Channel{{Path: "google/cloud/secretmanager/v1"}},
			wantOutput: "src/generated",
		},
		{
			name:            "rust with no channels creates default and derives path",
			language:        "rust",
			channels:        nil,
			wantOutput:      "src/generated/cloud/secretmanager/v1",
			wantChannelPath: "google/cloud/secretmanager/v1",
		},
		{
			name:            "veneer rust with no channels does not derive path",
			language:        "rust",
			output:          "src/storage/test/v1",
			veneer:          true,
			channels:        nil,
			wantOutput:      "src/storage/test/v1",
			wantChannelPath: "",
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
			name:            "rust lib without service config",
			language:        "rust",
			channels:        []*config.Channel{{Path: "google/cloud/orgpolicy/v1"}},
			wantOutput:      "src/generated/cloud/orgpolicy/v1",
			wantChannelPath: "google/cloud/orgpolicy/v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			lib := &config.Library{
				Name:     "google-cloud-secretmanager-v1",
				Output:   test.output,
				Veneer:   test.veneer,
				Channels: test.channels,
			}
			defaults := &config.Default{
				Output: "src/generated",
			}
			got, err := prepareLibrary(test.language, lib, defaults, true)
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
			if len(got.Channels) > 0 {
				ch := got.Channels[0]
				if test.wantChannelPath != "" && ch.Path != test.wantChannelPath {
					t.Errorf("got channel path %q, want %q", ch.Path, test.wantChannelPath)
				}
			}
		})
	}
}
