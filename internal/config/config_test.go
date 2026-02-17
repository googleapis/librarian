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
	"path/filepath"
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
			Showcase: &Source{
				Commit: "3f4e3f4f5e2f4c6e8b6f4e2f4c6e8b6f4e2f4c6e",
				SHA256: "d41d8cd98f00b204e9800998ecf8427e",
			},
			ProtobufSrc: &Source{
				Commit:  "4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b",
				SHA256:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				Subpath: "src",
			},
			Conformance: &Source{
				Commit: "1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b",
				SHA256: "f572d396fae9206628714fb2ce00f72e94f2258f",
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
				APIs: []*API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			{
				Name:    "google-cloud-storage-v2",
				Version: "2.3.4",
				APIs: []*API{
					{Path: "google/cloud/storage/v2"},
				},
				Roots: []string{"googleapis"},
				Rust: &RustCrate{
					RustDefault: RustDefault{
						DisabledRustdocWarnings: []string{"rustdoc::bare_urls"},
					},
				},
			},
			{
				Name:    "google-cloud-storage",
				Version: "1.4.0",
				Veneer:  true,
				Rust: &RustCrate{
					Modules: []*RustModule{
						{
							APIPath:         "google/storage/v2",
							ServiceConfig:   "google/storage/v2/storage_v2.yaml",
							Output:          "src/storage/src/generated/gapic",
							Template:        "grpc-client",
							HasVeneer:       true,
							RoutingRequired: true,
							IncludedIds: []string{
								".google.storage.v2.Storage.GetBucket",
								".google.storage.v2.Storage.ListBuckets",
							},
						},
						{
							APIPath:  "google/storage/v2",
							Output:   "src/storage/src/generated/protos/storage",
							Template: "prost",
						},
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

func TestConfigReadAndWrite(t *testing.T) {
	want, err := yaml.Read[Config]("testdata/librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}

	newFile := filepath.Join(t.TempDir(), "new_librarian.yaml")
	if err := yaml.Write(newFile, want); err != nil {
		t.Fatalf("yaml.Write() failed: %v", err)
	}

	got, err := yaml.Read[Config](newFile)
	if err != nil {
		t.Fatalf("yaml.Read() failed: %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
