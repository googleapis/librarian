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
	"os"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestRustWorkflow(t *testing.T) {
	testhelper.RequireCommand(t, "cargo")
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "taplo")

	for _, test := range []struct {
		name  string
		steps [][]string
		want  wantLibrary
	}{
		{
			name: "generate",
			steps: [][]string{
				{"librarian", "generate", testLibraryName},
			},
			want: wantLibrary{
				libraryName: testLibraryName,
				outputDir:   testOutputDir,
			},
		},
		{
			name: "generate all",
			steps: [][]string{
				{"librarian", "generate", "--all"},
			},
			want: wantLibrary{
				libraryName: testLibraryName,
				outputDir:   testOutputDir,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupRustRepo(t)
			for i, args := range test.steps {
				if err := Run(t.Context(), args...); err != nil {
					t.Fatalf("step %d (%v) failed: %v", i, args, err)
				}
			}
			verifyLibrary(t, test.want)
		})
	}
}

func setupRustRepo(t *testing.T) *config.Config {
	t.Helper()
	cfg := testConfig(t, languageRust, googleapisTestDir(t))
	cfg.Libraries = []*config.Library{
		{
			Name:    testLibraryName,
			Version: sample.InitialVersion,
			Output:  testOutputDir,
			Channels: []*config.Channel{
				{Path: "google/cloud/secretmanager/v1"},
			},
			Rust: &config.RustCrate{
				RustDefault: config.RustDefault{
					PackageDependencies: []*config.RustPackageDependency{
						{
							Name:      "wkt",
							Package:   "google-cloud-wkt",
							Source:    "google.protobuf",
							ForceUsed: true,
						},
						{
							Name:    "iam_v1",
							Package: "google-cloud-iam-v1",
							Source:  "google.iam.v1",
						},
						{
							Name:    "location",
							Package: "google-cloud-location",
							Source:  "google.cloud.location",
						},
					},
				},
				Modules: []*config.RustModule{
					{
						Source:   "google/cloud/secretmanager/v1",
						Template: "grpc-client",
					},
				},
			},
		},
	}
	setupRepo(t, cfg, func(t *testing.T) {
		workspaceCargoToml := `[workspace]
members = []
resolver = "2"

[workspace.package]
edition = "2024"
authors = ["Google LLC"]
license = "Apache-2.0"
repository = "https://github.com/googleapis/google-cloud-rust"
rust-version = "1.85.0"
keywords = []
categories = []

[workspace.dependencies]
iam_v1 = { version = "1", package = "google-cloud-iam-v1" }
location = { version = "1", package = "google-cloud-location" }
tokio-test = "0.4"
wkt = { version = "1", package = "google-cloud-wkt" }
`
		if err := os.WriteFile("Cargo.toml", []byte(workspaceCargoToml), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile("README.md", []byte("# Test Repo"), 0644); err != nil {
			t.Fatal(err)
		}
	})
	return cfg
}
