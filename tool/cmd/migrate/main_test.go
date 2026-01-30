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

package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestReadRootSidekick(t *testing.T) {
	for _, test := range []struct {
		name    string
		path    string
		want    *config.Config
		wantErr error
	}{
		{
			name: "success",
			path: "testdata/root-sidekick/success",
			want: &config.Config{
				Language: "dart",
				Version:  "0.4.0",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "211d22fa6dfabfa52cbda04d1aee852a01301edf",
						SHA256: "9aa6e5167f76b869b53b71f9fe963e6e17ec58b2cbdeb30715ef95b92faabfc5",
					},
					Showcase: &config.Source{
						Commit: "69bdd62035d793f3d23a0c960dee547023c1c5ac",
						SHA256: "96491310ba1b5c0c71738d3d80327a95196c1b6ac16f033e3fa440870efbbf5c",
					},
					ProtobufSrc: &config.Source{
						Commit:  "b407e8416e3893036aee5af9a12bd9b6a0e2b2e6",
						SHA256:  "55912546338433f465a552e9ef09930c63b9eb697053937416890cff83a8622d",
						Subpath: "src",
					},
					Conformance: &config.Source{
						Commit: "b407e8416e3893036aee5af9a12bd9b6a0e2b2e6",
						SHA256: "55912546338433f465a552e9ef09930c63b9eb697053937416890cff83a8622d",
					},
				},
				Default: &config.Default{
					Output:       "generated/",
					ReleaseLevel: "",
					Dart: &config.DartPackage{
						APIKeysEnvironmentVariables: "GOOGLE_API_KEY",
						IssueTrackerURL:             "https://github.com/googleapis/google-cloud-dart/issues",
						Packages: map[string]string{
							"package:google_cloud_api":          "^0.4.0",
							"package:google_cloud_iam_v1":       "^0.4.0",
							"package:google_cloud_location":     "^0.4.0",
							"package:google_cloud_logging_type": "^0.4.0",
							"package:google_cloud_longrunning":  "^0.4.0",
							"package:google_cloud_protobuf":     "^0.4.0",
							"package:google_cloud_rpc":          "^0.4.0",
							"package:google_cloud_type":         "^0.4.0",
							"package:googleapis_auth":           "^2.0.0",
							"package:http":                      "^1.3.0",
						},
						Prefixes: map[string]string{"prefix:google.logging.type": "logging_type"},
						Protos: map[string]string{
							"proto:google.api":            "package:google_cloud_api/api.dart",
							"proto:google.cloud.common":   "package:google_cloud_common/common.dart",
							"proto:google.cloud.location": "package:google_cloud_location/location.dart",
							"proto:google.iam.v1":         "package:google_cloud_iam_v1/iam.dart",
							"proto:google.logging.type":   "package:google_cloud_logging_type/logging_type.dart",
							"proto:google.longrunning":    "package:google_cloud_longrunning/longrunning.dart",
							"proto:google.protobuf":       "package:google_cloud_protobuf/protobuf.dart",
							"proto:google.rpc":            "package:google_cloud_rpc/rpc.dart",
							"proto:google.type":           "package:google_cloud_type/type.dart",
						},
					},
				},
			},
		},
		{
			name:    "no_sidekick_file",
			path:    "testdata/root-sidekick/no_sidekick_file",
			wantErr: errSidekickNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := readRootSidekick(test.path)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("got error %v, want %v", err, test.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("got error %v, want nil", err)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindSidekickFiles(t *testing.T) {
	for _, test := range []struct {
		name    string
		path    string
		want    []string
		wantErr error
	}{
		{
			name: "found_sidekick_files",
			path: "testdata/find-sidekick-files/success",
			want: []string{
				"testdata/find-sidekick-files/success/src/generated/sub-1/.sidekick.toml",
				"testdata/find-sidekick-files/success/src/generated/sub-1/subsub-1/.sidekick.toml",
			},
		},
		{
			name:    "no_src_directory",
			path:    "testdata/find-sidekick-files/no-src",
			wantErr: os.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := findSidekickFiles(test.path)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("got error %v, want %v", err, test.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("got error %v, want nil", err)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildGAPIC(t *testing.T) {
	for _, test := range []struct {
		name     string
		files    []string
		repoPath string
		want     []*config.Library
		wantErr  error
	}{
		{
			name: "read_sidekick_files",
			files: []string{
				"testdata/read-sidekick-files/success-read/library-a/.sidekick.toml",
				"testdata/read-sidekick-files/success-read/library-b/.sidekick.toml",
			},
			want: []*config.Library{
				{
					APIs: []*config.API{
						{
							Path: "google/ai/generativelanguage/v1beta",
						},
					},
					CopyrightYear:       "2025",
					Output:              "testdata/read-sidekick-files/success-read/library-a",
					SpecificationFormat: "protobuf",
					Dart: &config.DartPackage{
						APIKeysEnvironmentVariables: "GOOGLE_API_KEY,GEMINI_API_KEY",
						DevDependencies:             "googleapis_auth,test,test_utils",
						ReadmeAfterTitleText: `> [!TIP]
> Flutter applications should use
> [Firebase AI Logic](https://firebase.google.com/products/firebase-ai-logic).
>
> The Generative Language API is meant for Dart command-line, cloud, and server applications.
> For mobile and web applications, see instead
> [Firebase AI Logic](https://firebase.google.com/products/firebase-ai-logic), which provides
> client-side access to both the Gemini Developer API and Vertex AI.`,
						ReadmeQuickstartText: `## Quickstart

This quickstart shows you how to install the package and make your first
Gemini API request.

### Before you begin

You need a Gemini API key. If you don't already have one, you can get it for
free in [Google AI Studio](https://aistudio.google.com/app/api-keys)
([step-by-step instructions](https://ai.google.dev/gemini-api/docs/api-key)).

### Installing the package into your application

> [!TIP]
> You can create a skeleton application by running the terminal command: ` + "`dart create myapp`\n\n" +
							`Run the terminal command:
` + "\n```sh\ndart pub add google_cloud_ai_generativelanguage_v1beta\n```\n\n" +
							`### Make your first request

Here is an example that uses the generateContent method to send a request to
the Gemini API using the Gemini 2.5 Flash model.

If you set your API key as the environment variable ` + "`GEMINI_API_KEY` or\n" +
							"`GOOGLE_API_KEY`" + `, the API key will be picked up automatically by the client
when using the Gemini API libraries. Otherwise you will need to pass your
API key as an argument when initializing the client.
`,
						RepositoryURL: "https://github.com/googleapis/google-cloud-dart/tree/main/generated/google_cloud_ai_generativelanguage_v1beta",
					},
				},
				{
					APIs:                []*config.API{{Path: "google/rpc"}},
					CopyrightYear:       "2025",
					Output:              "testdata/read-sidekick-files/success-read/library-b",
					SpecificationFormat: "protobuf",
					Dart: &config.DartPackage{
						Dependencies:    "googleapis_auth,http",
						DevDependencies: "test",
						PartFile:        "src/rpc.p.dart",
						RepositoryURL:   "https://github.com/googleapis/google-cloud-dart/tree/main/generated/google_cloud_rpc",
					},
				},
			},
		},
		{
			name: "unable_to_calculate_output_path",
			files: []string{
				"testdata/read-sidekick-files/success-read/library-a/.sidekick.toml",
			},
			repoPath: "/invalid/repo/path",
			wantErr:  errUnableToCalculateOutputPath,
		},
		{
			name: "no_api_path",
			files: []string{
				"testdata/read-sidekick-files/no-api-path/.sidekick.toml",
			},
			want: []*config.Library{},
		},
		{
			name: "no_package_name",
			files: []string{
				"testdata/read-sidekick-files/no-package-name/.sidekick.toml",
			},
			wantErr: errLibraryNameNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildGAPIC(test.files, test.repoPath)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("got error %v, want %v", err, test.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("got error %v, want nil", err)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildConfig(t *testing.T) {
	for _, test := range []struct {
		name      string
		libraries []*config.Library
		defaults  *config.Config
		want      *config.Config
		wantErr   error
	}{
		{
			name: "rust_defaults",
			defaults: &config.Config{
				Default: &config.Default{
					Output: "src/generated/",
					Rust: &config.RustDefault{
						DisabledRustdocWarnings: []string{"bare_urls", "broken_intra_doc_links", "redundant_explicit_links"},
					},
				},
			},
			want: &config.Config{
				Default: &config.Default{
					Output: "src/generated/",
					Rust: &config.RustDefault{
						DisabledRustdocWarnings: []string{"bare_urls", "broken_intra_doc_links", "redundant_explicit_links"},
					},
				},
			},
		},
		{
			name:     "copy_libraries",
			defaults: &config.Config{},
			libraries: []*config.Library{
				{
					Name: "google-cloud-security-publicca-v1",
					APIs: []*config.API{
						{
							Path: "google/cloud/security/publicca/v1",
						},
					},
					Version:       "1.1.0",
					CopyrightYear: "2025",
					Rust: &config.RustCrate{
						RustDefault: config.RustDefault{
							DisabledRustdocWarnings: []string{"bare_urls", "broken_intra_doc_links", "redundant_explicit_links"},
							GenerateSetterSamples:   "true",
							GenerateRpcSamples:      "true",
						},
						PerServiceFeatures: true,
						NameOverrides:      ".google.cloud.security/publicca.v1.Storage=StorageControl",
					},
				},
				{
					Name: "google-cloud-sql-v1",
					APIs: []*config.API{
						{
							Path: "google/cloud/sql/v1",
						},
					},
					SkipPublish: true,
					Version:     "1.2.0",
				},
			},
			want: &config.Config{
				Libraries: []*config.Library{
					{
						Name: "google-cloud-security-publicca-v1",
						APIs: []*config.API{
							{
								Path: "google/cloud/security/publicca/v1",
							},
						},
						Version:       "1.1.0",
						CopyrightYear: "2025",
						Rust: &config.RustCrate{
							RustDefault: config.RustDefault{
								DisabledRustdocWarnings: []string{"bare_urls", "broken_intra_doc_links", "redundant_explicit_links"},
								GenerateSetterSamples:   "true",
								GenerateRpcSamples:      "true",
							},
							PerServiceFeatures: true,
							NameOverrides:      ".google.cloud.security/publicca.v1.Storage=StorageControl",
						},
					},
				},
			},
		},
		{
			name:     "service does not exist",
			defaults: &config.Config{},
			libraries: []*config.Library{
				{
					Name: "google-cloud-orgpolicy-v1",
					APIs: []*config.API{
						{
							Path: "google/cloud/orgpolicy/v1",
						},
					},
					Version:       "1.1.0",
					CopyrightYear: "2025",
					Rust: &config.RustCrate{
						RustDefault: config.RustDefault{
							DisabledRustdocWarnings: []string{"bare_urls", "broken_intra_doc_links", "redundant_explicit_links"},
							GenerateSetterSamples:   "true",
							GenerateRpcSamples:      "true",
						},
						PerServiceFeatures: true,
						NameOverrides:      ".google.cloud.orgpolicy.v1.OrgPolicy=OrgPolicyControl",
					},
				},
			},
			want: &config.Config{
				Libraries: []*config.Library{
					{
						Name: "google-cloud-orgpolicy-v1",
						APIs: []*config.API{
							{
								Path: "google/cloud/orgpolicy/v1",
							},
						},
						Version:       "1.1.0",
						CopyrightYear: "2025",
						Rust: &config.RustCrate{
							RustDefault: config.RustDefault{
								DisabledRustdocWarnings: []string{"bare_urls", "broken_intra_doc_links", "redundant_explicit_links"},
								GenerateSetterSamples:   "true",
								GenerateRpcSamples:      "true",
							},
							PerServiceFeatures: true,
							NameOverrides:      ".google.cloud.orgpolicy.v1.OrgPolicy=OrgPolicyControl",
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildConfig(test.libraries, test.defaults)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunMigrateCommand(t *testing.T) {
	for _, test := range []struct {
		name                         string
		path                         string
		wantErr                      error
		checkDocumentOverrideReplace []string
		checkDocumentOverrideMatch   []string
	}{
		{
			name: "success",
			path: "testdata/run/success",
		},
		{
			name:    "tidy_command_fails",
			path:    "testdata/run/tidy-fails",
			wantErr: errTidyFailed,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// ensure librarian.yaml generated is removed after the test,
			// even if the test fails
			outputPath := "librarian.yaml"
			t.Cleanup(func() {
				if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
					t.Logf("cleanup: remove %s: %v", outputPath, err)
				}
			})
			abs, err := filepath.Abs(test.path)
			if err != nil {
				t.Fatal(err)
			}
			if err := runSidekickMigration(t.Context(), abs); err != nil {
				if test.wantErr == nil {
					t.Fatal(err)
				}
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("expected error containing %q, got: %v", test.wantErr, err)
				}
			} else if test.wantErr != nil {
				t.Fatalf("expected error containing %q, got nil", test.wantErr)
			} else {
				data, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("librarian file does not exist")
				}

				librarianConfig, err := yaml.Unmarshal[config.Config](data)
				if err != nil {
					t.Fatalf("unable to parse librarian file")
				}
				if len(librarianConfig.Libraries) != 1 {
					t.Fatalf("librarian yaml does not contain library")
				}
			}

		})
	}
}

func TestParseWithPrefix(t *testing.T) {
	for _, test := range []struct {
		name   string
		prefix string
		want   map[string]string
	}{
		{
			name:   "prefix: as a prefix",
			prefix: "prefix:",
			want: map[string]string{
				"prefix:google.logging.type": "logging_type",
			},
		},
		{
			name:   "package: as a prefix",
			prefix: "package:",
			want: map[string]string{
				"package:googleapis_auth": "^2.0.0",
			},
		},
		{
			name:   "proto: as a prefix",
			prefix: "proto:",
			want: map[string]string{
				"proto:google.protobuf": "package:google_cloud_protobuf/protobuf.dart",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			codec := map[string]string{
				"prefix:google.logging.type": "logging_type",
				"package:googleapis_auth":    "^2.0.0",
				"proto:google.protobuf":      "package:google_cloud_protobuf/protobuf.dart",
			}
			got := parseKeyWithPrefix(codec, test.prefix)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
