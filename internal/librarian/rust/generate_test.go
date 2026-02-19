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
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/source"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestGenerateVeneer(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	outDir := t.TempDir()
	module1Dir := filepath.Join(outDir, "src", "generated", "v1")
	module2Dir := filepath.Join(outDir, "src", "generated", "v1beta")
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}

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
					{Name: "google-cloud-api", Package: "google-cloud-api", Source: "google.api"},
					{Name: "google-cloud-type", Package: "google-cloud-type", Source: "google.type"},
				},
			},
			Modules: []*config.RustModule{
				{
					APIPath:  "google/cloud/secretmanager/v1",
					Output:   module1Dir,
					Template: "grpc-client",
				},
				{
					APIPath:  "google/cloud/secretmanager/v1",
					Output:   module2Dir,
					Template: "grpc-client",
				},
			},
		},
	}
	sources := &source.Sources{
		Googleapis: googleapisDir,
	}
	if err := generate(t.Context(), library, sources); err != nil {
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

func TestSkipGenerateVeneer(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	outDir := t.TempDir()
	module1Dir := filepath.Join(outDir, "src", "generated", "v1")
	module2Dir := filepath.Join(outDir, "src", "generated", "v1beta")
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}

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
		},
	}
	sources := &source.Sources{
		Googleapis: googleapisDir,
	}
	if err := generate(t.Context(), library, sources); err != nil {
		t.Fatal(err)
	}

	for _, dir := range []string{module1Dir, module2Dir} {
		generatedFile := filepath.Join(dir, "model.rs")
		_, err := os.ReadFile(generatedFile)
		if err == nil {
			t.Errorf("want file %s to not exist, but it does", generatedFile)
		} else if !os.IsNotExist(err) {
			t.Errorf("unexpected error for file %s: %v", generatedFile, err)
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
	want := []string{"src/custom.rs"}
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

// TestGenerateLibraries performs simple testing that multiple libraries can
// be generated. Only the presence of a single expected file per library is
// performed; TestGenerate is responsible for more detailed testing of
// per-library generation.
func TestGenerateLibraries(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "rustfmt")
	testhelper.RequireCommand(t, "taplo")
	testhelper.RequireCommand(t, "cargo")

	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}

	workspaceDir := t.TempDir()
	cargoToml, err := os.ReadFile(filepath.Join("testdata", "Cargo.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspaceDir, "Cargo.toml"), cargoToml, 0644); err != nil {
		t.Fatal(err)
	}

	// Mock validate to speed up the test.
	oldValidate := validate
	validate = func(ctx context.Context, outputDir string) error { return nil }
	t.Cleanup(func() { validate = oldValidate })

	libraries := []*config.Library{
		{
			Name:          "google-cloud-secretmanager-v1",
			Version:       "0.1.0",
			ReleaseLevel:  "preview",
			CopyrightYear: "2025",
			APIs: []*config.API{
				{
					Path: "google/cloud/secretmanager/v1",
				},
			},
			Rust: &config.RustCrate{
				RustDefault: config.RustDefault{
					PackageDependencies: []*config.RustPackageDependency{
						{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
						{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
						{Name: "location", Package: "google-cloud-location", Source: "google.cloud.location"},
						{Name: "google-cloud-api", Package: "google-cloud-api", Source: "google.api"},
						{Name: "google-cloud-type", Package: "google-cloud-type", Source: "google.type"},
					},
				},
			},
		},
		{
			Name:          "google-cloud-configdelivery-v1",
			Version:       "0.1.0",
			ReleaseLevel:  "preview",
			CopyrightYear: "2025",
			APIs: []*config.API{
				{
					Path: "google/cloud/configdelivery/v1",
				},
			},
			Rust: &config.RustCrate{
				RustDefault: config.RustDefault{
					PackageDependencies: []*config.RustPackageDependency{
						{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
						{Name: "location", Package: "google-cloud-location", Source: "google.cloud.location"},
						{Name: "google-cloud-api", Package: "google-cloud-api", Source: "google.api"},
						{Name: "google-cloud-longrunning", Package: "google-cloud-longrunning", Source: "google.longrunning"},
						{Name: "google-cloud-rpc", Package: "google-cloud-rpc", Source: "google.rpc"},
					},
				},
			},
		},
	}
	sources := &source.Sources{
		Googleapis: googleapisDir,
	}
	t.Chdir(workspaceDir)
	for _, library := range libraries {
		library.Output = filepath.Join("generated", library.Name)
	}

	if err := GenerateLibraries(t.Context(), libraries, sources); err != nil {
		t.Fatal(err)
	}
	// Just check that a Cargo.toml has been created for each library.
	for _, library := range libraries {
		expectedPubspec := filepath.Join(library.Output, "Cargo.toml")
		_, err := os.Stat(expectedPubspec)
		if err != nil {
			t.Errorf("Stat(%s) returned error: %v", expectedPubspec, err)
		}
	}
}

func TestGenerateLibraries_Error(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "rustfmt")
	testhelper.RequireCommand(t, "taplo")
	testhelper.RequireCommand(t, "cargo")

	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}

	workspaceDir := t.TempDir()
	cargoToml, err := os.ReadFile(filepath.Join("testdata", "Cargo.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspaceDir, "Cargo.toml"), cargoToml, 0644); err != nil {
		t.Fatal(err)
	}

	// Mock validate to speed up the test.
	oldValidate := validate
	validate = func(ctx context.Context, outputDir string) error { return nil }
	t.Cleanup(func() { validate = oldValidate })

	libraries := []*config.Library{
		{
			Name:                "broken",
			Version:             "0.1.0",
			Output:              filepath.Join("generated", "broken"),
			ReleaseLevel:        "preview",
			CopyrightYear:       "2025",
			SpecificationFormat: "invalid",
			APIs: []*config.API{
				{
					Path: "google/cloud/secretmanager/v1",
				},
			},
		},
	}
	sources := &source.Sources{
		Googleapis: googleapisDir,
	}
	t.Chdir(workspaceDir)

	gotErr := GenerateLibraries(t.Context(), libraries, sources)
	wantErrMessage := "unknown specification format \"invalid\""
	if gotErr == nil {
		t.Fatalf("expected error with message %s", wantErrMessage)
	}
	if diff := cmp.Diff(wantErrMessage, gotErr.Error()); diff != "" {
		t.Errorf("error mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerate(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "rustfmt")
	testhelper.RequireCommand(t, "taplo")
	testhelper.RequireCommand(t, "cargo")

	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	workspaceDir, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		os.RemoveAll(filepath.Join(workspaceDir, "target"))
		os.Remove(filepath.Join(workspaceDir, "Cargo.lock"))
	})

	// Mock validate to speed up the test.
	oldValidate := validate
	validate = func(ctx context.Context, outputDir string) error { return nil }
	t.Cleanup(func() { validate = oldValidate })

	for _, test := range []struct {
		name      string
		preExists bool
	}{
		{
			name:      "directory exists",
			preExists: true,
		},
		{
			name:      "directory does not exist",
			preExists: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Change to testdata directory so cargo fmt can find Cargo.toml
			t.Chdir(workspaceDir)

			libName := "google-cloud-secretmanager-v1"
			outDir := filepath.Join(workspaceDir, libName)

			if err := os.RemoveAll(outDir); err != nil {
				t.Fatal(err)
			}
			if test.preExists {
				if err := os.MkdirAll(outDir, 0755); err != nil {
					t.Fatal(err)
				}
			}
			t.Cleanup(func() { os.RemoveAll(outDir) })

			library := &config.Library{
				Name:          libName,
				Version:       "0.1.0",
				Output:        outDir,
				ReleaseLevel:  "preview",
				CopyrightYear: "2025",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
					},
				},
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
							{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
							{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
							{Name: "location", Package: "google-cloud-location", Source: "google.cloud.location"},
							{Name: "google-cloud-api", Package: "google-cloud-api", Source: "google.api"},
							{Name: "google-cloud-type", Package: "google-cloud-type", Source: "google.type"},
						},
					},
				},
			}
			sources := &source.Sources{
				Googleapis: googleapisDir,
			}
			if err := generate(t.Context(), library, sources); err != nil {
				t.Fatal(err)
			}

			for _, check := range []struct {
				path string
				want string
			}{
				{filepath.Join(outDir, "Cargo.toml"), "name"},
				{filepath.Join(outDir, "Cargo.toml"), libName},
				{filepath.Join(outDir, "README.md"), "# Google Cloud Client Libraries for Rust - Secret Manager API"},
				{filepath.Join(outDir, "src", "lib.rs"), "pub mod model;"},
				{filepath.Join(outDir, "src", "lib.rs"), "pub mod client;"},
			} {
				t.Run(check.path, func(t *testing.T) {
					if _, err := os.Stat(check.path); err != nil {
						t.Fatal(err)
					}
					got, err := os.ReadFile(check.path)
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(string(got), check.want) {
						t.Errorf("%q missing expected string: %q", check.path, check.want)
					}
				})
			}
		})
	}
}

func TestDefaultLibraryName(t *testing.T) {
	for _, test := range []struct {
		name string
		api  string
		want string
	}{
		{
			name: "simple",
			api:  "google/cloud/secretmanager/v1",
			want: "google-cloud-secretmanager-v1",
		},
		{
			name: "no slashes",
			api:  "name",
			want: "name",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DefaultLibraryName(test.api)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeriveAPIPath(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  string
		want string
	}{
		{
			name: "simple",
			lib:  "google-cloud-secretmanager-v1",
			want: "google/cloud/secretmanager/v1",
		},
		{
			name: "no dashes",
			lib:  "name",
			want: "name",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DeriveAPIPath(test.lib)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindModuleByOutput(t *testing.T) {
	for _, test := range []struct {
		name   string
		lib    *config.Library
		output string
		want   *config.RustModule
	}{
		{
			name: "find the module",
			lib: &config.Library{
				Name: "test",
				Rust: &config.RustCrate{
					Modules: []*config.RustModule{
						{
							Language: "target-language",
							Output:   "target-output",
						},
						{
							Language: "other-language",
							Output:   "other-output",
						},
					},
				},
			},
			output: "target-output",
			want: &config.RustModule{
				Language: "target-language",
				Output:   "target-output",
			},
		},
		{
			name: "does not find the module",
			lib: &config.Library{
				Name: "test",
				Rust: &config.RustCrate{
					Modules: []*config.RustModule{
						{
							Language: "other-language",
							Output:   "other-output",
						},
					},
				},
			},
			output: "target-output",
			want:   nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := findModuleByOutput(test.lib, test.output)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
