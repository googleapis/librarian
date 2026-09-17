// Copyright 2026 Google LLC
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
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

const testGoogleapisDir = "../testdata/googleapis"

func TestGenerateDocIndex_Rust(t *testing.T) {
	cfg := &config.Config{
		Language: config.LanguageRust,
		Libraries: []*config.Library{
			{
				Name: "google-cloud-secretmanager-v1",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			{
				Name: "google-cloud-compute-v1",
				APIs: []*config.API{
					{Path: "google/cloud/compute/v1"},
				},
			},
			{
				Name:   "google-cloud-location",
				Output: "src/location",
				Rust: &config.RustCrate{
					Modules: []*config.RustModule{
						{APIPath: "google/iam/v1"},
						{APIPath: "google/cloud/location"},
					},
				},
			},
			// Libraries without configured APIs or service configs are skipped.
			{
				Name:   "google-cloud-auth",
				Output: "src/auth",
			},
			{
				Name:        "google-cloud-internal-mock",
				SkipRelease: true,
				APIs: []*config.API{
					{Path: "google/cloud/compute/v1"},
				},
			},
		},
	}

	gotBytes, err := GenerateDocIndex(cfg, testGoogleapisDir)
	if err != nil {
		t.Fatalf("GenerateDocIndex() error = %v", err)
	}

	want := `{
  "google-cloud-compute-v1": [
    {
      "PkgName": "google-cloud-compute-v1",
      "Language": "Rust",
      "DocsURL": "https://docs.rs/google-cloud-compute-v1/latest",
      "APIShortname": "compute",
      "Product": "Google Compute Engine API"
    }
  ],
  "google-cloud-location": [
    {
      "PkgName": "google-cloud-location",
      "Language": "Rust",
      "DocsURL": "https://docs.rs/google-cloud-location/latest",
      "APIShortname": "cloud",
      "Product": "Cloud Metadata API"
    }
  ],
  "google-cloud-secretmanager-v1": [
    {
      "PkgName": "google-cloud-secretmanager-v1",
      "Language": "Rust",
      "DocsURL": "https://docs.rs/google-cloud-secretmanager-v1/latest",
      "APIShortname": "secretmanager",
      "Product": "Secret Manager API"
    }
  ]
}
`
	if diff := cmp.Diff(want, string(gotBytes)); diff != "" {
		t.Errorf("GenerateDocIndex() mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateDocIndex_Swift(t *testing.T) {
	cfg := &config.Config{
		Language: config.LanguageSwift,
		Libraries: []*config.Library{
			{
				Name:    "google-cloud-secretmanager-v1",
				Version: "0.1.0-preview",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			{
				Name:    "swift-google-cloud-compute-v1",
				Version: "1.2.0",
				APIs: []*config.API{
					{Path: "google/cloud/compute/v1"},
				},
			},
			{
				Name:    "google-cloud-location",
				Version: "0.1.0-preview",
				Output:  "pkgs/swift-google-cloud-location/Sources/generated",
				Swift: &config.SwiftPackage{
					Modules: []*config.SwiftModule{
						{APIPath: "google/iam/v1"},
						{APIPath: "google/cloud/location"},
					},
				},
			},
			// Libraries without configured APIs or service configs are skipped.
			{
				Name:    "google-cloud-auth",
				Version: "0.0.0-preview",
				Output:  "pkgs/swift-google-auth/Sources/GoogleCloudAuth/generated",
			},
			{
				Name:    "google-cloud-gax",
				Version: "0.0.0-preview",
				Output:  "pkgs/swift-google-gax/Sources/GoogleCloudGax/generated",
			},
			{
				Name:        "google-cloud-internal-mock",
				SkipRelease: true,
				APIs: []*config.API{
					{Path: "google/cloud/compute/v1"},
				},
			},
		},
	}

	gotBytes, err := GenerateDocIndex(cfg, testGoogleapisDir)
	if err != nil {
		t.Fatalf("GenerateDocIndex() error = %v", err)
	}

	want := `{
  "swift-google-cloud-compute-v1": [
    {
      "PkgName": "swift-google-cloud-compute-v1",
      "Language": "Swift",
      "DocsURL": "https://swiftpackageindex.com/googleapis/swift-google-cloud-compute-v1/1.2.0/documentation",
      "APIShortname": "compute",
      "Product": "Google Compute Engine API"
    }
  ],
  "swift-google-cloud-location": [
    {
      "PkgName": "swift-google-cloud-location",
      "Language": "Swift",
      "DocsURL": "https://swiftpackageindex.com/googleapis/swift-google-cloud-location/0.1.0-preview/documentation",
      "APIShortname": "cloud",
      "Product": "Cloud Metadata API"
    }
  ],
  "swift-google-cloud-secretmanager-v1": [
    {
      "PkgName": "swift-google-cloud-secretmanager-v1",
      "Language": "Swift",
      "DocsURL": "https://swiftpackageindex.com/googleapis/swift-google-cloud-secretmanager-v1/0.1.0-preview/documentation",
      "APIShortname": "secretmanager",
      "Product": "Secret Manager API"
    }
  ]
}
`
	if diff := cmp.Diff(want, string(gotBytes)); diff != "" {
		t.Errorf("GenerateDocIndex() mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateDocIndex_Stability(t *testing.T) {
	cfg := &config.Config{
		Language: config.LanguageSwift,
		Libraries: []*config.Library{
			{
				Name:    "google-cloud-secretmanager-v1",
				Version: "0.1.0-preview",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			{
				Name:    "swift-google-cloud-compute-v1",
				Version: "1.2.0",
				APIs: []*config.API{
					{Path: "google/cloud/compute/v1"},
				},
			},
		},
	}

	first, err := GenerateDocIndex(cfg, testGoogleapisDir)
	if err != nil {
		t.Fatalf("GenerateDocIndex() error = %v", err)
	}

	for i := range 20 {
		repeat, err := GenerateDocIndex(cfg, testGoogleapisDir)
		if err != nil {
			t.Fatalf("GenerateDocIndex() iteration %d error = %v", i, err)
		}
		if !bytes.Equal(first, repeat) {
			t.Fatalf("GenerateDocIndex() output unstable across runs at iteration %d", i)
		}
	}
}

func TestGenerateDocIndex_ShowcaseAndNoAPIsSkipped(t *testing.T) {
	cfg := &config.Config{
		Language: config.LanguageRust,
		Libraries: []*config.Library{
			{
				Name: "google-cloud-secretmanager-v1",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			{
				Name: "gapic-showcase",
				APIs: []*config.API{
					{Path: "schema/google/showcase/v1beta1"},
				},
			},
			{
				Name: "empty-library",
				APIs: []*config.API{},
			},
		},
	}

	gotBytes, err := GenerateDocIndex(cfg, testGoogleapisDir)
	if err != nil {
		t.Fatalf("GenerateDocIndex() error = %v", err)
	}

	want := `{
  "google-cloud-secretmanager-v1": [
    {
      "PkgName": "google-cloud-secretmanager-v1",
      "Language": "Rust",
      "DocsURL": "https://docs.rs/google-cloud-secretmanager-v1/latest",
      "APIShortname": "secretmanager",
      "Product": "Secret Manager API"
    }
  ]
}
`
	if diff := cmp.Diff(want, string(gotBytes)); diff != "" {
		t.Errorf("GenerateDocIndex() mismatch (-want +got):\n%s", diff)
	}
}

func TestWriteDocIndex(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "generated")
	absGoogleapisDir, err := filepath.Abs(testGoogleapisDir)
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Language: config.LanguageRust,
		Default: &config.Default{
			Output: outDir,
		},
		Libraries: []*config.Library{
			{
				Name: "google-cloud-secretmanager-v1",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
		},
	}

	if err := writeDocIndex(cfg, absGoogleapisDir); err != nil {
		t.Fatalf("writeDocIndex failed: %v", err)
	}

	outFile := filepath.Join(outDir, "_libraries.json")
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("ReadFile(_libraries.json) error = %v", err)
	}

	want := `{
  "google-cloud-secretmanager-v1": [
    {
      "PkgName": "google-cloud-secretmanager-v1",
      "Language": "Rust",
      "DocsURL": "https://docs.rs/google-cloud-secretmanager-v1/latest",
      "APIShortname": "secretmanager",
      "Product": "Secret Manager API"
    }
  ]
}
`
	if diff := cmp.Diff(want, string(content)); diff != "" {
		t.Errorf("writeDocIndex() mismatch (-want +got):\n%s", diff)
	}

	// Test skip when Default is nil or Output is empty or unsupported language
	noOutCfg := &config.Config{
		Language: config.LanguageRust,
	}
	if err := writeDocIndex(noOutCfg, absGoogleapisDir); err != nil {
		t.Fatalf("writeDocIndex with nil Default failed: %v", err)
	}

	unsupportedCfg := &config.Config{
		Language: config.LanguageGo,
		Default: &config.Default{
			Output: outDir,
		},
	}
	if err := writeDocIndex(unsupportedCfg, absGoogleapisDir); err != nil {
		t.Fatalf("writeDocIndex with unsupported language failed: %v", err)
	}
}

func TestIsEmbeddedAPIPath(t *testing.T) {
	for _, test := range []struct {
		path string
		want bool
	}{
		{"google/iam", true},
		{"google/iam/v1", true},
		{"google/iam_admin", false},
		{"google/type", true},
		{"google/type/money", true},
		{"google/type_custom", false},
		{"google/rpc", true},
		{"google/rpc/status", true},
		{"google/rpc_custom", false},
		{"google/longrunning", true},
		{"google/longrunning/operations", true},
		{"google/longrunning_custom", false},
		{"google/protobuf", true},
		{"google/protobuf/any", true},
		{"google/protobuf_custom", false},
		{"google/cloud/secretmanager/v1", false},
	} {
		if got := isEmbeddedAPIPath(test.path); got != test.want {
			t.Errorf("isEmbeddedAPIPath(%q) = %v, want %v", test.path, got, test.want)
		}
	}
}

func TestPrimaryAPIPath(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want string
	}{
		{
			name: "single explicit api",
			lib: &config.Library{
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			want: "google/cloud/secretmanager/v1",
		},
		{
			name: "multiple explicit apis sorted by SortAPIs",
			lib: &config.Library{
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1beta1"},
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			want: "google/cloud/secretmanager/v1",
		},
		{
			name: "rust veneer with storage control and storage selects storage",
			lib: &config.Library{
				Rust: &config.RustCrate{
					Modules: []*config.RustModule{
						{APIPath: "google/storage/control/v2"},
						{APIPath: "google/iam/v1"},
						{APIPath: "google/longrunning"},
						{APIPath: "google/storage/v2"},
						{APIPath: "google/type"},
					},
				},
			},
			want: "google/storage/v2",
		},
		{
			name: "swift veneer with storage control and storage selects storage",
			lib: &config.Library{
				Swift: &config.SwiftPackage{
					Modules: []*config.SwiftModule{
						{APIPath: "google/storage/control/v2"},
						{APIPath: "google/type"},
						{APIPath: "google/storage/v2"},
						{APIPath: "google/rpc"},
						{APIPath: "google/longrunning"},
						{APIPath: "google/iam/v1"},
					},
				},
			},
			want: "google/storage/v2",
		},
		{
			name: "veneer with embedded iam and unversioned location selects location",
			lib: &config.Library{
				Rust: &config.RustCrate{
					Modules: []*config.RustModule{
						{APIPath: "google/iam/v1"},
						{APIPath: "google/cloud/location"},
					},
				},
			},
			want: "google/cloud/location",
		},
		{
			name: "empty apis and modules returns empty",
			lib: &config.Library{
				Name: "google-cloud-auth",
			},
			want: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := primaryAPIPath(test.lib)
			if got != test.want {
				t.Errorf("primaryAPIPath() = %q, want %q", got, test.want)
			}
		})
	}
}
