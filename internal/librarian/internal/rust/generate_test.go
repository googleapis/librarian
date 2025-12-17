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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelpers"
)

func TestGenerateVeneer(t *testing.T) {
	testhelpers.RequireCommand(t, "protoc")
	testdataDir, err := filepath.Abs("../../../sidekick/testdata")
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()
	module1Dir := filepath.Join(outDir, "src", "generated", "v1")
	module2Dir := filepath.Join(outDir, "src", "generated", "v1beta")
	googleapisDir := filepath.Join(testdataDir, "googleapis")

	library := &config.Library{
		Name:          "test-veneer",
		Veneer:        true,
		Output:        outDir,
		CopyrightYear: "2025",
		Rust: &config.RustCrate{
			RustDefault: config.RustDefault{
				PackageDependencies: []*config.RustPackageDependency{
					{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
					{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
					{Name: "location", Package: "google-cloud-location", Source: "google.cloud.location"},
				},
			},
			Modules: []*config.RustModule{
				{
					Source:        "google/cloud/secretmanager/v1",
					ServiceConfig: "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
					Output:        module1Dir,
					Template:      "grpc-client",
				},
				{
					Source:        "google/cloud/secretmanager/v1",
					ServiceConfig: "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
					Output:        module2Dir,
					Template:      "grpc-client",
				},
			},
		},
	}
	sources := &config.Sources{
		Googleapis: &config.Source{Dir: googleapisDir},
	}
	if err := Generate(t.Context(), library, sources); err != nil {
		t.Fatal(err)
	}

	for _, dir := range []string{module1Dir, module2Dir} {
		model, err := os.ReadFile(filepath.Join(dir, "model.rs"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(model), "SecretManagerService") {
			t.Errorf("%s/model.rs missing SecretManagerService", dir)
		}
	}
}

func TestKeepNonVeneer(t *testing.T) {
	library := &config.Library{
		Keep: []string{"src/custom.rs"},
	}
	got, err := Keep(library)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"src/custom.rs", "Cargo.toml"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestKeepVeneer(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{
		"Cargo.toml",
		"src/lib.rs",
		"src/generated/model.rs",
	} {
		path := filepath.Join(dir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
	}

	library := &config.Library{
		Veneer: true,
		Output: dir,
		Rust: &config.RustCrate{
			Modules: []*config.RustModule{
				{Output: filepath.Join(dir, "src", "generated")},
			},
		},
	}
	got, err := Keep(library)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)
	want := []string{"Cargo.toml", "src/lib.rs"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerate(t *testing.T) {
	testhelpers.RequireCommand(t, "protoc")
	testhelpers.RequireCommand(t, "rustfmt")
	testhelpers.RequireCommand(t, "taplo")
	testdataDir, err := filepath.Abs("../../../sidekick/testdata")
	if err != nil {
		t.Fatal(err)
	}

	// Change to testdata directory so cargo fmt can find Cargo.toml
	workspaceDir, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(workspaceDir, "google-cloud-secretmanager-v1")
	t.Cleanup(func() { os.RemoveAll(outDir) })
	t.Chdir(workspaceDir)
	googleapisDir := filepath.Join(testdataDir, "googleapis")
	library := &config.Library{
		Name:          "google-cloud-secretmanager-v1",
		Version:       "0.1.0",
		Output:        outDir,
		ReleaseLevel:  "preview",
		CopyrightYear: "2025",
		Channels: []*config.Channel{
			{
				Path:          "google/cloud/secretmanager/v1",
				ServiceConfig: "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			},
		},
		Rust: &config.RustCrate{
			RustDefault: config.RustDefault{
				PackageDependencies: []*config.RustPackageDependency{
					{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
					{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
					{Name: "location", Package: "google-cloud-location", Source: "google.cloud.location"},
				},
			},
		},
	}
	sources := &config.Sources{
		Googleapis: &config.Source{Dir: googleapisDir},
	}
	if err := Generate(t.Context(), library, sources); err != nil {
		t.Fatal(err)
	}
	if err := Format(t.Context(), library); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		path string
		want string
	}{
		{filepath.Join(outDir, "Cargo.toml"), "name"},
		{filepath.Join(outDir, "Cargo.toml"), "google-cloud-secretmanager-v1"},
		{filepath.Join(outDir, "README.md"), "# Google Cloud Client Libraries for Rust - Secret Manager API"},
		{filepath.Join(outDir, "src", "lib.rs"), "pub mod model;"},
		{filepath.Join(outDir, "src", "lib.rs"), "pub mod client;"},
	} {
		t.Run(test.path, func(t *testing.T) {
			if _, err := os.Stat(test.path); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(test.path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(got), test.want) {
				t.Errorf("%q missing expected string: %q", test.path, test.want)
			}
		})
	}
}
