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
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/source"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestResolveDependencies(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	sources := &source.Sources{
		Googleapis: googleapisDir,
	}

	for _, test := range []struct {
		name    string
		lib     *config.Library
		cfg     *config.Config
		want    []*config.RustPackageDependency
		wantErr bool
	}{
		{
			name: "resolve from other libraries",
			lib: &config.Library{
				Name: "google-cloud-gkehub-v1",
				APIs: []*config.API{
					{Path: "google/cloud/gkehub/v1"},
				},
			},
			cfg: &config.Config{
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: googleapisDir},
				},
				Language: "rust",
				Default: &config.Default{
					Rust: &config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
							{Name: "google-cloud-rpc", Package: "google-cloud-rpc", Source: "google.rpc"},
						},
					},
				},
				Libraries: []*config.Library{
					{
						Name: "google-cloud-gkehub-configmanagement-v1",
						APIs: []*config.API{
							{Path: "google/cloud/gkehub/v1/configmanagement"},
						},
					},
					{
						Name: "no-apis-library",
					},
				},
			},
			want: []*config.RustPackageDependency{
				{Name: "google-cloud-gkehub-configmanagement-v1", Package: "google-cloud-gkehub-configmanagement-v1", Source: "google.cloud.gkehub.configmanagement.v1"},
			},
		},
		{
			name: "respect default dependencies",
			lib: &config.Library{
				Name: "google-cloud-secretmanager-v1",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			cfg: &config.Config{
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: googleapisDir},
				},
				Language: "rust",
				Default: &config.Default{
					Rust: &config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
							{Name: "google-cloud-iam-v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
							{Name: "google-cloud-location", Package: "google-cloud-location", Source: "google.cloud.location"},
							{Name: "google-cloud-type", Package: "google-cloud-type", Source: "google.type"},
							{Name: "google-cloud-wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "existing dependencies preserved",
			lib: &config.Library{
				Name: "google-cloud-secretmanager-v1",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
							{Name: "custom", Package: "custom-crate", Source: "custom.proto"},
						},
					},
				},
			},
			cfg: &config.Config{
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: googleapisDir},
				},
				Language: "rust",
				Default: &config.Default{
					Rust: &config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
							{Name: "google-cloud-iam-v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
							{Name: "google-cloud-location", Package: "google-cloud-location", Source: "google.cloud.location"},
							{Name: "google-cloud-type", Package: "google-cloud-type", Source: "google.type"},
						},
					},
				},
			},
			want: []*config.RustPackageDependency{
				{Name: "custom", Package: "custom-crate", Source: "custom.proto"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.cfg.Libraries = append(test.cfg.Libraries, test.lib)
			_, err := ResolveDependencies(t.Context(), test.cfg, test.lib, sources)
			if (err != nil) != test.wantErr {
				t.Errorf("ResolveDependencies() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if err == nil {
				gotLib := test.lib
				var got []*config.RustPackageDependency
				if gotLib.Rust != nil {
					got = gotLib.Rust.PackageDependencies
				}
				sort.Slice(got, func(i, j int) bool {
					return got[i].Source < got[j].Source
				})
				sort.Slice(test.want, func(i, j int) bool {
					return test.want[i].Source < test.want[j].Source
				})
				if diff := cmp.Diff(test.want, got, cmpopts.IgnoreUnexported(config.RustPackageDependency{})); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
