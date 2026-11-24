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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	sidekickrust "github.com/googleapis/librarian/internal/sidekick/rust"
	"github.com/googleapis/librarian/internal/sources"
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
	sources := &sources.Sources{
		Googleapis: googleapisDir,
	}
	if err := Generate(t.Context(), &config.Config{Language: "rust", Repo: "google-cloud-rust"}, library, sources); err != nil {
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

func TestIsMixedLibrary(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want bool
	}{
		{
			name: "rust modules with output",
			lib: &config.Library{
				Name:   "google-cloud-storage",
				Output: "src/storage",
				Rust: &config.RustCrate{
					Modules: []*config.RustModule{
						{Output: "src/storage/src/generated"},
					},
				},
			},
			want: true,
		},
		{
			name: "rust modules with api, no output",
			lib: &config.Library{
				Name: "google-cloud-storage",
				APIs: []*config.API{
					{Path: "google/storage/v2"},
				},
				Rust: &config.RustCrate{
					Modules: []*config.RustModule{
						{Output: "src/storage/src/generated"},
					},
				},
			},
			want: true,
		},
		{
			name: "output without api",
			lib: &config.Library{
				Name:   "storage-w1r3",
				Output: "src/storage/benchmarks/w1r3",
			},
			want: true,
		},
		{
			name: "nosvc library without rust modules",
			lib: &config.Library{
				Name:   "google-cloud-oslogin-common",
				Output: "src/generated/cloud/oslogin/common",
			},
			want: false,
		},
		{
			name: "output with api",
			lib: &config.Library{
				Name: "google-cloud-api",
				APIs: []*config.API{
					{Path: "google/api"},
				},
				Output: "src/generated/api/types",
			},
			want: false,
		},
		{
			name: "handwritten library not in sdk.yaml",
			lib: &config.Library{
				Name:   "google-cloud-auth",
				Output: "src/auth",
			},
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := IsMixedLibrary(test.lib); got != test.want {
				t.Errorf("IsMixedLibrary() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestGenerateVeneerNoModules(t *testing.T) {
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
	sources := &sources.Sources{
		Googleapis: googleapisDir,
	}
	if err := Generate(t.Context(), &config.Config{Language: "rust", Repo: "google-cloud-rust"}, library, sources); err != nil {
		t.Fatal(err)
	}

	for _, dir := range []string{module1Dir, module2Dir} {
		generatedFile := filepath.Join(dir, "model.rs")
		_, err := os.ReadFile(generatedFile)
		if err == nil {
			t.Errorf("want file %s to not exist, but it does", generatedFile)
		} else if !errors.Is(err, fs.ErrNotExist) {
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
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	library := &config.Library{
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

// TestGenerate performs simple testing that multiple libraries can be
// generated. Only the presence of a single expected file per library is
// performed; TestGenerateLibrary is responsible for more detailed testing of
// per-library generation.
func TestGenerate(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(workspaceDir, "Cargo.toml"), cargoToml, 0o644); err != nil {
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
	sources := &sources.Sources{
		Googleapis: googleapisDir,
	}
	t.Chdir(workspaceDir)
	for _, library := range libraries {
		library.Output = filepath.Join("generated", library.Name)
	}

	cfg := &config.Config{Language: "rust", Repo: "google-cloud-rust"}
	for _, library := range libraries {
		if err := Generate(t.Context(), cfg, library, sources); err != nil {
			t.Fatal(err)
		}
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

func TestGenerate_Error(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(workspaceDir, "Cargo.toml"), cargoToml, 0o644); err != nil {
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
			CopyrightYear:       "2025",
			SpecificationFormat: "invalid",
			APIs: []*config.API{
				{
					Path: "google/cloud/secretmanager/v1",
				},
			},
		},
	}
	sources := &sources.Sources{
		Googleapis: googleapisDir,
	}
	t.Chdir(workspaceDir)

	cfg := &config.Config{Language: "rust", Repo: "google-cloud-rust"}
	gotErr := Generate(t.Context(), cfg, libraries[0], sources)
	wantErrMessage := "unknown specification format \"invalid\""
	if gotErr == nil {
		t.Fatalf("expected error with message %s", wantErrMessage)
	}
	if diff := cmp.Diff(wantErrMessage, gotErr.Error()); diff != "" {
		t.Errorf("error mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateLibrary(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "rustfmt")
	testhelper.RequireCommand(t, "taplo")
	testhelper.RequireCommand(t, "cargo")

	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}

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
			// Do all the work in a temporary directory, to keep the source
			// directory unchanged.
			temp := t.TempDir()
			t.Chdir(temp)

			libName := "google-cloud-secretmanager-v1"
			outDir := "src/generated/cloud/secretmanager/v1"
			if test.preExists {
				if err := os.MkdirAll(outDir, 0o755); err != nil {
					t.Fatal(err)
				}
				contents := fmt.Appendf(nil, formatTestCargoToml, fmt.Sprintf(`  "%s",`, outDir))
				if err := os.WriteFile("Cargo.toml", contents, 0o644); err != nil {
					t.Fatal(err)
				}
			} else {
				contents := fmt.Appendf(nil, formatTestCargoToml, "")
				if err := os.WriteFile("Cargo.toml", contents, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			t.Cleanup(func() { os.RemoveAll(outDir) })

			library := &config.Library{
				Name:          libName,
				Version:       "0.1.0",
				Output:        outDir,
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
			sources := &sources.Sources{
				Googleapis: googleapisDir,
			}
			if err := Generate(t.Context(), &config.Config{Language: "rust", Repo: "google-cloud-rust"}, library, sources); err != nil {
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
				{filepath.Join(outDir, ".repo-metadata.json"), "secretmanager.googleapis.com"},
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

func TestDefaultOutput(t *testing.T) {
	for _, test := range []struct {
		name          string
		api           string
		defaultOutput string
		want          string
	}{
		{
			name:          "typical api with google prefix",
			api:           "google/cloud/secretmanager/v1",
			defaultOutput: "src/generated",
			want:          "src/generated/cloud/secretmanager/v1",
		},
		{
			name:          "api without google prefix",
			api:           "other/service/v1",
			defaultOutput: "src/generated",
			want:          "src/generated/other/service/v1",
		},
		{
			name:          "empty api",
			api:           "",
			defaultOutput: "src/generated",
			want:          "src/generated",
		},
		{
			name:          "empty default output",
			api:           "google/cloud/secretmanager/v1",
			defaultOutput: "",
			want:          "cloud/secretmanager/v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DefaultOutput(test.api, test.defaultOutput)
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
							Output: "target-output",
						},
						{
							Output: "other-output",
						},
					},
				},
			},
			output: "target-output",
			want: &config.RustModule{
				Output: "target-output",
			},
		},
		{
			name: "does not find the module",
			lib: &config.Library{
				Name: "test",
				Rust: &config.RustCrate{
					Modules: []*config.RustModule{
						{
							Output: "other-output",
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

func TestCreateRepoMetadata(t *testing.T) {
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		library *config.Library
		sources *sources.Sources
		needed  bool
		want    *repometadata.RepoMetadata
	}{
		{
			name: "googleapis",
			library: &config.Library{
				Name:    "google-cloud-secretmanager-v1",
				Version: "0.1.0",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
					},
				},
			},
			sources: &sources.Sources{
				Googleapis: googleapisDir,
			},
			needed: true,
			want: &repometadata.RepoMetadata{
				Name:                 "secretmanager",
				NamePretty:           "Secret Manager",
				ProductDocumentation: "https://cloud.google.com/secret-manager/",
				ClientDocumentation:  "https://docs.rs/google-cloud-secretmanager-v1/latest",
				IssueTracker:         "https://issuetracker.google.com/issues/new?component=784854&template=1380926",
				Language:             config.LanguageRust,
				Repo:                 "googleapis/google-cloud-rust",
				DistributionName:     "google-cloud-secretmanager-v1",
				APIID:                "secretmanager.googleapis.com",
				APIShortname:         "secretmanager",
				APIDescription:       "Stores sensitive data such as API keys, passwords, and certificates.\nProvides convenience while improving security.",
				ReleaseLevel:         "preview",
				LibraryType:          "GAPIC_AUTO",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				Language: config.LanguageRust,
				Repo:     "googleapis/google-cloud-rust",
			}
			got, err := createRepoMetadata(cfg, test.library, test.sources)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateVeneerRedactedDebugFields(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	srcs := &sources.Sources{Googleapis: googleapisDir}
	deps := []*config.RustPackageDependency{
		{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
		{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
		{Name: "location", Package: "google-cloud-location", Source: "google.cloud.location"},
		{Name: "google-cloud-api", Package: "google-cloud-api", Source: "google.api"},
		{Name: "google-cloud-type", Package: "google-cloud-type", Source: "google.type"},
	}

	t.Run("crate level scopes across multi-package modules", func(t *testing.T) {
		outDir := t.TempDir()
		smDir := filepath.Join(outDir, "src", "generated", "secretmanager")
		locDir := filepath.Join(outDir, "src", "generated", "location")
		library := &config.Library{
			Name:          "test-veneer",
			Output:        outDir,
			CopyrightYear: "2025",
			Rust: &config.RustCrate{
				RustDefault:         config.RustDefault{PackageDependencies: deps},
				RedactedDebugFields: []string{".google.cloud.secretmanager.v1.Secret.name"},
				Modules: []*config.RustModule{
					{APIPath: "google/cloud/secretmanager/v1", Output: smDir, Template: "grpc-client"},
					{APIPath: "google/cloud/location", Output: locDir, Template: "grpc-client"},
				},
			},
		}
		if err := Generate(t.Context(), &config.Config{Language: "rust", Repo: "google-cloud-rust"}, library, srcs); err != nil {
			t.Fatal(err)
		}
		debugBytes, err := os.ReadFile(filepath.Join(smDir, "model", "debug.rs"))
		if err != nil {
			t.Fatal(err)
		}
		secretImpl := extractRustBlock(t, string(debugBytes), "impl std::fmt::Debug for super::Secret {", "\n}")
		if !strings.Contains(secretImpl, `debug_struct.field("name", &"[censored]");`) {
			t.Errorf("expected Secret.name to be redacted in:\n%s", secretImpl)
		}
		if strings.Contains(secretImpl, "&self.name") {
			t.Errorf("expected Secret.name not to print &self.name in:\n%s", secretImpl)
		}
		getSecretReqImpl := extractRustBlock(t, string(debugBytes), "impl std::fmt::Debug for super::GetSecretRequest {", "\n}")
		if !strings.Contains(getSecretReqImpl, `debug_struct.field("name", &self.name);`) {
			t.Errorf("expected GetSecretRequest.name not to be redacted in:\n%s", getSecretReqImpl)
		}
		if strings.Contains(getSecretReqImpl, `"[censored]"`) {
			t.Errorf("expected GetSecretRequest.name not to be censored in:\n%s", getSecretReqImpl)
		}

		locDebugBytes, err := os.ReadFile(filepath.Join(locDir, "model", "debug.rs"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(locDebugBytes), `"[censored]"`) {
			t.Errorf("expected location debug.rs not to contain censored fields:\n%s", string(locDebugBytes))
		}
	})

	for _, test := range []struct {
		name        string
		crateFields []string
		modules     func(outDir string) []*config.RustModule
		checkErr    func(t *testing.T, err error)
	}{
		{
			name:        "crate level package typo fails",
			crateFields: []string{".google.cloud.secretmanager.v99.Secret.name"},
			modules: func(outDir string) []*config.RustModule {
				return []*config.RustModule{
					{APIPath: "google/cloud/secretmanager/v1", Output: filepath.Join(outDir, "sm"), Template: "grpc-client"},
				}
			},
			checkErr: func(t *testing.T, err error) {
				t.Helper()
				if !strings.Contains(err.Error(), ".google.cloud.secretmanager.v99.Secret.name") || !strings.Contains(err.Error(), "did not match") {
					t.Errorf("unexpected error: %v", err)
				}
			},
		},
		{
			name: "module level package typo fails",
			modules: func(outDir string) []*config.RustModule {
				return []*config.RustModule{
					{
						APIPath:             "google/cloud/secretmanager/v1",
						Output:              filepath.Join(outDir, "sm"),
						Template:            "grpc-client",
						RedactedDebugFields: []string{".google.cloud.secretmanager.v99.Secret.name"},
					},
				}
			},
			checkErr: func(t *testing.T, err error) {
				t.Helper()
				if !errors.Is(err, sidekickrust.ErrRedactedDebugFieldNotFound) {
					t.Errorf("expected ErrRedactedDebugFieldNotFound, got %v", err)
				}
			},
		},
		{
			name: "unsupported template rejects module redacted_debug_fields",
			modules: func(outDir string) []*config.RustModule {
				return []*config.RustModule{
					{
						APIPath:             "google/cloud/secretmanager/v1",
						Output:              filepath.Join(outDir, "conv"),
						Template:            "convert-prost",
						ModulePath:          "crate::model",
						RedactedDebugFields: []string{".google.cloud.secretmanager.v1.Secret.name"},
					},
				}
			},
			checkErr: func(t *testing.T, err error) {
				t.Helper()
				if !strings.Contains(err.Error(), "redacted_debug_fields is not supported for template") || !strings.Contains(err.Error(), "convert-prost") {
					t.Errorf("unexpected error: %v", err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			library := &config.Library{
				Name:          "test-veneer",
				Output:        outDir,
				CopyrightYear: "2025",
				Rust: &config.RustCrate{
					RustDefault:         config.RustDefault{PackageDependencies: deps},
					RedactedDebugFields: test.crateFields,
					Modules:             test.modules(outDir),
				},
			}
			err := Generate(t.Context(), &config.Config{Language: "rust", Repo: "google-cloud-rust"}, library, srcs)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			test.checkErr(t, err)
		})
	}
}

func extractRustBlock(t *testing.T, content, start, end string) string {
	t.Helper()
	begin := strings.Index(content, start)
	if begin == -1 {
		t.Fatalf("missing %q in:\n%s", start, content)
	}
	rest := content[begin:]
	stop := strings.Index(rest, end)
	if stop == -1 {
		t.Fatalf("missing closing %q after %q in:\n%s", end, start, rest)
	}
	return rest[:stop+len(end)]
}

func TestModuleGeneratesDebug(t *testing.T) {
	for _, test := range []struct {
		template string
		want     bool
	}{
		{template: "", want: true},
		{template: "crate", want: true},
		{template: "grpc-client", want: true},
		{template: "http-client", want: true},
		{template: "mod", want: true},
		{template: "nosvc", want: true},
		{template: "templates/crate", want: true},
		{template: "templates/nosvc", want: true},
		{template: "templates/grpc-client", want: true},
		{template: "templates/http-client", want: true},
		{template: "templates/mod", want: true},
		{template: "convert-prost", want: false},
		{template: "grpc-mock", want: false},
		{template: "prost", want: false},
		{template: "tonic", want: false},
		{template: "bigquery", want: false},
		{template: "storage", want: false},
		{template: "templates/convert-prost", want: false},
		{template: "templates/grpc-mock", want: false},
		{template: "templates/bigquery", want: false},
		{template: "custom-template", want: false},
	} {
		t.Run(test.template, func(t *testing.T) {
			if got := moduleGeneratesDebug(test.template); got != test.want {
				t.Errorf("moduleGeneratesDebug(%q) = %v, want %v", test.template, got, test.want)
			}
		})
	}
}

func TestScopeModuleRedactedDebugFields(t *testing.T) {
	t.Run("split-package modules", func(t *testing.T) {
		secretMsg := api.NewTestMessage("Secret").
			WithPackage("google.cloud.secretmanager.v1").
			WithFields(api.NewTestField("name").WithType(api.TypezString))
		secretModel := api.NewTestAPI([]*api.Message{secretMsg}, nil, nil).
			WithPackageName("google.cloud.secretmanager.v1")

		versionMsg := api.NewTestMessage("SecretVersion").
			WithPackage("google.cloud.secretmanager.v1").
			WithFields(api.NewTestField("payload").WithType(api.TypezBytes))
		versionModel := api.NewTestAPI([]*api.Message{versionMsg}, nil, nil).
			WithPackageName("google.cloud.secretmanager.v1")

		library := &config.Library{
			Rust: &config.RustCrate{
				RedactedDebugFields: []string{
					".google.cloud.secretmanager.v1.Secret.name",
					".google.cloud.secretmanager.v1.SecretVersion.payload",
				},
			},
		}
		matched := make(map[string]bool)

		secretCfg := &parser.ModelConfig{Codec: map[string]string{}}
		scopeModuleRedactedDebugFields(library, &config.RustModule{}, secretModel, secretCfg, matched)
		if got := secretCfg.Codec["redacted-debug-fields"]; got != ".google.cloud.secretmanager.v1.Secret.name" {
			t.Errorf("secret module redacted-debug-fields = %q, want %q", got, ".google.cloud.secretmanager.v1.Secret.name")
		}

		versionCfg := &parser.ModelConfig{Codec: map[string]string{}}
		scopeModuleRedactedDebugFields(library, &config.RustModule{}, versionModel, versionCfg, matched)
		if got := versionCfg.Codec["redacted-debug-fields"]; got != ".google.cloud.secretmanager.v1.SecretVersion.payload" {
			t.Errorf("version module redacted-debug-fields = %q, want %q", got, ".google.cloud.secretmanager.v1.SecretVersion.payload")
		}

		wantMatched := map[string]bool{
			".google.cloud.secretmanager.v1.Secret.name":           true,
			".google.cloud.secretmanager.v1.SecretVersion.payload": true,
		}
		if diff := cmp.Diff(wantMatched, matched); diff != "" {
			t.Errorf("matchedCrateFields mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("nested subpackage field not matched by parent package", func(t *testing.T) {
		iamMsg := api.NewTestMessage("Policy").
			WithPackage("google.iam.v1").
			WithFields(api.NewTestField("etag").WithType(api.TypezBytes))
		iamModel := api.NewTestAPI([]*api.Message{iamMsg}, nil, nil).
			WithPackageName("google.iam.v1")

		library := &config.Library{
			Rust: &config.RustCrate{
				RedactedDebugFields: []string{".google.iam.v1.logging.AuditData.secret"},
			},
		}
		matched := make(map[string]bool)
		cfg := &parser.ModelConfig{Codec: map[string]string{}}
		scopeModuleRedactedDebugFields(library, &config.RustModule{}, iamModel, cfg, matched)
		if got, ok := cfg.Codec["redacted-debug-fields"]; ok {
			t.Errorf("expected redacted-debug-fields to be unset, got %q", got)
		}
		if len(matched) != 0 {
			t.Errorf("expected no matched fields, got %v", matched)
		}
	})

	t.Run("package-less module", func(t *testing.T) {
		root := api.NewTestMessage("Root").
			WithPackage("").
			WithID(".Root").
			WithFields(api.NewTestField("secret").WithType(api.TypezBytes))
		model := api.NewTestAPI([]*api.Message{root}, nil, nil).WithPackageName("")

		library := &config.Library{
			Rust: &config.RustCrate{
				RedactedDebugFields: []string{".Root.secret"},
			},
		}
		matched := make(map[string]bool)
		cfg := &parser.ModelConfig{Codec: nil}
		scopeModuleRedactedDebugFields(library, &config.RustModule{}, model, cfg, matched)
		if got := cfg.Codec["redacted-debug-fields"]; got != ".Root.secret" {
			t.Errorf("redacted-debug-fields = %q, want %q", got, ".Root.secret")
		}
		if !matched[".Root.secret"] {
			t.Errorf("expected .Root.secret to be marked matched")
		}
	})

	t.Run("nested message fields and oneof IDs", func(t *testing.T) {
		nested := api.NewTestMessage("Nested").
			WithFields(api.NewTestField("raw_key").WithType(api.TypezBytes))
		oneofBranch := api.NewTestField("secret_token").WithType(api.TypezString).WithOneOf()
		oneofGroup := api.NewTestOneOf("credential").WithFields(oneofBranch)
		parent := api.NewTestMessage("Parent").
			WithPackage("test.v1").
			WithMessages(nested).
			WithOneOfs(oneofGroup)
		model := api.NewTestAPI([]*api.Message{parent}, nil, nil).WithPackageName("test.v1")

		library := &config.Library{
			Rust: &config.RustCrate{
				RedactedDebugFields: []string{
					".test.v1.Parent.Nested.raw_key",
					".test.v1.Parent.credential",
					".test.v1.Parent.secret_token",
				},
			},
		}
		matched := make(map[string]bool)
		cfg := &parser.ModelConfig{Codec: map[string]string{}}
		scopeModuleRedactedDebugFields(library, &config.RustModule{}, model, cfg, matched)
		want := ".test.v1.Parent.Nested.raw_key,.test.v1.Parent.credential,.test.v1.Parent.secret_token"
		if got := cfg.Codec["redacted-debug-fields"]; got != want {
			t.Errorf("redacted-debug-fields = %q, want %q", got, want)
		}
		for _, id := range library.Rust.RedactedDebugFields {
			if !matched[id] {
				t.Errorf("expected %q to be marked matched", id)
			}
		}
	})

	t.Run("service placeholder ignored while child message collected", func(t *testing.T) {
		child := api.NewTestMessage("SyntheticReq").
			WithFields(api.NewTestField("secret").WithType(api.TypezBytes))
		placeholder := api.NewTestMessage("ServicePlaceholder").
			WithPackage("test.v1").
			WithServicePlaceholder().
			WithFields(api.NewTestField("ignored").WithType(api.TypezString)).
			WithMessages(child)
		model := api.NewTestAPI([]*api.Message{placeholder}, nil, nil).WithPackageName("test.v1")

		library := &config.Library{
			Rust: &config.RustCrate{
				RedactedDebugFields: []string{
					".test.v1.ServicePlaceholder.ignored",
					".test.v1.ServicePlaceholder.SyntheticReq.secret",
				},
			},
		}
		matched := make(map[string]bool)
		cfg := &parser.ModelConfig{Codec: map[string]string{}}
		scopeModuleRedactedDebugFields(library, &config.RustModule{}, model, cfg, matched)
		if got := cfg.Codec["redacted-debug-fields"]; got != ".test.v1.ServicePlaceholder.SyntheticReq.secret" {
			t.Errorf("redacted-debug-fields = %q, want %q", got, ".test.v1.ServicePlaceholder.SyntheticReq.secret")
		}
		if matched[".test.v1.ServicePlaceholder.ignored"] {
			t.Errorf("expected placeholder field not to match")
		}
		if !matched[".test.v1.ServicePlaceholder.SyntheticReq.secret"] {
			t.Errorf("expected nested real child field to match")
		}
	})

	t.Run("merges crate and module IDs without duplicates", func(t *testing.T) {
		msg := api.NewTestMessage("Msg").
			WithPackage("test.v1").
			WithFields(
				api.NewTestField("crate_secret").WithType(api.TypezString),
				api.NewTestField("shared").WithType(api.TypezString),
				api.NewTestField("module_secret").WithType(api.TypezString),
			)
		model := api.NewTestAPI([]*api.Message{msg}, nil, nil).WithPackageName("test.v1")

		library := &config.Library{
			Rust: &config.RustCrate{
				RedactedDebugFields: []string{".test.v1.Msg.crate_secret", ".test.v1.Msg.shared"},
			},
		}
		module := &config.RustModule{
			RedactedDebugFields: []string{".test.v1.Msg.shared", ".test.v1.Msg.module_secret"},
		}
		matched := make(map[string]bool)
		cfg := &parser.ModelConfig{Codec: map[string]string{}}
		scopeModuleRedactedDebugFields(library, module, model, cfg, matched)
		want := ".test.v1.Msg.crate_secret,.test.v1.Msg.shared,.test.v1.Msg.module_secret"
		if got := cfg.Codec["redacted-debug-fields"]; got != want {
			t.Errorf("redacted-debug-fields = %q, want %q", got, want)
		}
	})

	t.Run("clears stale redacted-debug-fields when nothing matches", func(t *testing.T) {
		msg := api.NewTestMessage("Msg").
			WithPackage("test.v1").
			WithFields(api.NewTestField("name").WithType(api.TypezString))
		model := api.NewTestAPI([]*api.Message{msg}, nil, nil).WithPackageName("test.v1")

		library := &config.Library{
			Rust: &config.RustCrate{
				RedactedDebugFields: []string{".other.v1.Msg.secret"},
			},
		}
		cfg := &parser.ModelConfig{
			Codec: map[string]string{
				"redacted-debug-fields": ".stale.v1.Msg.old_secret",
			},
		}
		scopeModuleRedactedDebugFields(library, &config.RustModule{}, model, cfg, make(map[string]bool))
		if got, ok := cfg.Codec["redacted-debug-fields"]; ok {
			t.Errorf("expected stale redacted-debug-fields to be cleared, got %q", got)
		}
	})
}

func TestValidateMatchedCrateRedactedFields(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		matched map[string]bool
		wantErr string
	}{
		{
			name:    "Rust is nil",
			library: &config.Library{Name: "test-lib"},
		},
		{
			name: "empty fields",
			library: &config.Library{
				Name: "test-lib",
				Rust: &config.RustCrate{},
			},
		},
		{
			name: "all matched",
			library: &config.Library{
				Name: "test-lib",
				Rust: &config.RustCrate{
					RedactedDebugFields: []string{".test.v1.Msg.secret"},
				},
			},
			matched: map[string]bool{".test.v1.Msg.secret": true},
		},
		{
			name: "unmatched ID returns error",
			library: &config.Library{
				Name: "test-lib",
				Rust: &config.RustCrate{
					RedactedDebugFields: []string{".test.v1.Msg.secret", ".test.v1.Msg.missing"},
				},
			},
			matched: map[string]bool{".test.v1.Msg.secret": true},
			wantErr: ".test.v1.Msg.missing",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateMatchedCrateRedactedFields(test.library, test.matched)
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), test.wantErr) || !strings.Contains(err.Error(), "did not match") {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGenerateVeneerNoModulesRedactedDebugFields(t *testing.T) {
	t.Run("returns unmatched error when redacted_debug_fields is set", func(t *testing.T) {
		library := &config.Library{
			Name:   "test-veneer",
			Output: t.TempDir(),
			Rust: &config.RustCrate{
				RedactedDebugFields: []string{".google.cloud.secretmanager.v1.Secret.name"},
			},
		}
		err := Generate(t.Context(), &config.Config{Language: "rust"}, library, &sources.Sources{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), ".google.cloud.secretmanager.v1.Secret.name") || !strings.Contains(err.Error(), "did not match") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("succeeds without redacted_debug_fields", func(t *testing.T) {
		library := &config.Library{
			Name:   "test-veneer",
			Output: t.TempDir(),
			Rust:   &config.RustCrate{},
		}
		if err := Generate(t.Context(), &config.Config{Language: "rust"}, library, &sources.Sources{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestGenerateNonVeneerUnsupportedTemplateRedactedDebugFields(t *testing.T) {
	for _, template := range []string{"templates/convert-prost", "templates/bigquery"} {
		t.Run(template, func(t *testing.T) {
			library := &config.Library{
				Name:   "test-crate",
				Output: t.TempDir(),
				APIs:   []*config.API{{Path: "google/cloud/secretmanager/v1"}},
				Rust: &config.RustCrate{
					TemplateOverride:    template,
					RedactedDebugFields: []string{".google.cloud.secretmanager.v1.Secret.name"},
				},
			}
			err := Generate(t.Context(), &config.Config{Language: "rust"}, library, &sources.Sources{})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "redacted_debug_fields is not supported for template") || !strings.Contains(err.Error(), template) {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGenerateTracingTarget(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	srcs := &sources.Sources{Googleapis: googleapisDir}
	deps := []*config.RustPackageDependency{
		{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
		{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
		{Name: "location", Package: "google-cloud-location", Source: "google.cloud.location"},
		{Name: "google-cloud-api", Package: "google-cloud-api", Source: "google.api"},
		{Name: "google-cloud-type", Package: "google-cloud-type", Source: "google.type"},
	}

	outDir := t.TempDir()
	crateTargetDir := filepath.Join(outDir, "crate_target")
	modOverrideDir := filepath.Join(outDir, "mod_override")
	library := &config.Library{
		Name:          "test-veneer",
		Output:        outDir,
		CopyrightYear: "2025",
		Rust: &config.RustCrate{
			RustDefault:   config.RustDefault{PackageDependencies: deps},
			TracingTarget: "crate_target::tracing",
			Modules: []*config.RustModule{
				{
					APIPath:  "google/cloud/secretmanager/v1",
					Output:   crateTargetDir,
					Template: "grpc-client",
				},
				{
					APIPath:       "google/cloud/secretmanager/v1",
					Output:        modOverrideDir,
					Template:      "grpc-client",
					TracingTarget: "module_target::tracing",
				},
			},
		},
	}
	if err := Generate(t.Context(), &config.Config{Language: "rust", Repo: "google-cloud-rust"}, library, srcs); err != nil {
		t.Fatal(err)
	}

	crateTracingBytes, err := os.ReadFile(filepath.Join(crateTargetDir, "tracing.rs"))
	if err != nil {
		t.Fatal(err)
	}
	wantCrateAttr := `#[tracing::instrument(target = "crate_target::tracing", level = tracing::Level::DEBUG, ret)]`
	if !strings.Contains(string(crateTracingBytes), wantCrateAttr) {
		t.Errorf("crate-inherited tracing.rs missing %q in:\n%s", wantCrateAttr, string(crateTracingBytes))
	}

	modTracingBytes, err := os.ReadFile(filepath.Join(modOverrideDir, "tracing.rs"))
	if err != nil {
		t.Fatal(err)
	}
	wantModAttr := `#[tracing::instrument(target = "module_target::tracing", level = tracing::Level::DEBUG, ret)]`
	if !strings.Contains(string(modTracingBytes), wantModAttr) {
		t.Errorf("module-overridden tracing.rs missing %q in:\n%s", wantModAttr, string(modTracingBytes))
	}
	if strings.Contains(string(modTracingBytes), "crate_target::tracing") {
		t.Errorf("module-overridden tracing.rs unexpectedly used crate target:\n%s", string(modTracingBytes))
	}
}

func TestGenerateStorageTemplateRedactedDebugFields(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	srcs := &sources.Sources{Googleapis: googleapisDir}
	deps := []*config.RustPackageDependency{
		{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
		{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
		{Name: "location", Package: "google-cloud-location", Source: "google.cloud.location"},
		{Name: "google-cloud-api", Package: "google-cloud-api", Source: "google.api"},
		{Name: "google-cloud-type", Package: "google-cloud-type", Source: "google.type"},
	}

	t.Run("matched crate field succeeds", func(t *testing.T) {
		outDir := t.TempDir()
		storageOut := filepath.Join(outDir, "storage")
		library := &config.Library{
			Name:          "test-storage",
			Output:        outDir,
			CopyrightYear: "2025",
			Rust: &config.RustCrate{
				RustDefault:         config.RustDefault{PackageDependencies: deps},
				RedactedDebugFields: []string{".google.cloud.secretmanager.v1.Secret.name"},
				Modules: []*config.RustModule{
					{Output: storageOut, Template: "storage"},
					{APIPath: "google/cloud/secretmanager/v1", Output: "src/storage/src/generated/gapic", Template: "grpc-client"},
					{APIPath: "google/cloud/location", Output: "src/storage/src/generated/gapic_control", Template: "grpc-client"},
				},
			},
		}
		if err := Generate(t.Context(), &config.Config{Language: "rust", Repo: "google-cloud-rust"}, library, srcs); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"client.rs", "stub.rs"} {
			if _, err := os.Stat(filepath.Join(storageOut, name)); err != nil {
				t.Errorf("expected %s to exist: %v", name, err)
			}
		}
	})

	t.Run("unmatched crate field fails", func(t *testing.T) {
		outDir := t.TempDir()
		storageOut := filepath.Join(outDir, "storage")
		library := &config.Library{
			Name:          "test-storage",
			Output:        outDir,
			CopyrightYear: "2025",
			Rust: &config.RustCrate{
				RustDefault:         config.RustDefault{PackageDependencies: deps},
				RedactedDebugFields: []string{".google.cloud.secretmanager.v99.Secret.name"},
				Modules: []*config.RustModule{
					{Output: storageOut, Template: "storage"},
					{APIPath: "google/cloud/secretmanager/v1", Output: "src/storage/src/generated/gapic", Template: "grpc-client"},
					{APIPath: "google/cloud/location", Output: "src/storage/src/generated/gapic_control", Template: "grpc-client"},
				},
			},
		}
		err := Generate(t.Context(), &config.Config{Language: "rust", Repo: "google-cloud-rust"}, library, srcs)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), ".google.cloud.secretmanager.v99.Secret.name") || !strings.Contains(err.Error(), "did not match") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
