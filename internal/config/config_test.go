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

package config

import (
	"bytes"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestRead(t *testing.T) {
	got, err := Read("testdata/rust/librarian.yaml")
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
				Commit: "ded7ed1e4cce7c165c56a417572cebea9bc1d82c",
				SHA256: "839e897c39cada559b97d64f90378715a4a43fbc972d8cf93296db4156662085",
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
				Version: "0.1.0",
				Channels: []*Channel{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			{
				Name:    "google-cloud-storage-v2",
				Version: "0.2.0",
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
	cfg, err := Read("testdata/rust/librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	enc := yaml.NewEncoder(&got)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		t.Fatal(err)
	}

	wantCfg, err := Read("testdata/rust/librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var gotCfg Config
	if err := yaml.Unmarshal(got.Bytes(), &gotCfg); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(wantCfg, &gotCfg); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
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
