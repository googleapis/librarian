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

func TestRustRead(t *testing.T) {
	got, err := yaml.Read[Config]("testdata/rust/librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := &Config{
		Language: LanguageRust,
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
			Output:    "src/generated/",
			TagFormat: "{name}/v{version}",
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

func TestDotnetRead(t *testing.T) {
	got, err := yaml.Read[Config]("testdata/dotnet/librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := &Config{
		Language: LanguageDotnet,
		Sources: &Sources{
			Googleapis: &Source{
				Commit: "9fcfbea0aa5b50fa22e190faceb073d74504172b",
				SHA256: "81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98",
			},
		},
		Default: &Default{
			Output: "apis",
		},
		Libraries: []*Library{
			{
				Name:    "Google.Cloud.SecretManager.V1",
				Version: "2.0.0",
				Dotnet: &DotnetPackage{
					Dependencies: map[string]string{
						"Google.Cloud.Iam.V1":   "3.5.0",
						"Google.Cloud.Location": "2.4.0",
					},
				},
			},
			{
				Name:    "Google.Cloud.AIPlatform.V1",
				Version: "1.0.0",
				Dotnet: &DotnetPackage{
					Pregeneration: []*DotnetPregeneration{
						{
							RenameMessage: &DotnetRenameMessage{
								From: "Schema",
								To:   "OpenApiSchema",
							},
						},
						{
							RemoveField: &DotnetRemoveField{
								Message: "QueryDeployedModelsResponse",
								Field:   "deployed_models",
							},
						},
					},
				},
			},
			{
				Name:    "Google.Cloud.Logging.V2",
				Version: "1.0.0",
				Dotnet: &DotnetPackage{
					Pregeneration: []*DotnetPregeneration{
						{
							RenameRPC: &DotnetRenameRPC{
								From:     "UpdateBucketAsync",
								To:       "UpdateBucketLongRunning",
								WireName: "UpdateBucketAsync",
							},
						},
					},
				},
			},
			{
				Name:    "Google.Cloud.Bigtable.V2",
				Version: "1.0.0",
				Dotnet: &DotnetPackage{
					Postgeneration: []*DotnetPostgeneration{
						{
							Run: "dotnet run --project tools/BigtableClient.GenerateClient",
						},
					},
				},
			},
			{
				Name:    "Google.LongRunning",
				Version: "1.0.0",
				Dotnet: &DotnetPackage{
					Postgeneration: []*DotnetPostgeneration{
						{
							ExtraProto: "google/cloud/extended_operations.proto",
						},
					},
				},
			},
			{
				Name:    "Google.Cloud.DevTools.ContainerAnalysis.V1",
				Version: "1.0.0",
				Dotnet: &DotnetPackage{
					AdditionalServiceDescriptors: []string{
						"Grafeas.V1.GrafeasReflection.Descriptor",
					},
				},
			},
			{
				Name:    "Google.Cloud.Vision.V1",
				Version: "1.0.0",
				Dotnet: &DotnetPackage{
					Csproj: &DotnetCsproj{
						Snippets: &DotnetCsprojSnippets{
							EmbeddedResources: []string{"*.jpg", "*.png"},
						},
						IntegrationTests: &DotnetCsprojSnippets{
							EmbeddedResources: []string{"vision_eiffel_tower.jpg"},
						},
					},
				},
			},
			{
				Name:    "Google.Cloud.Spanner.V1",
				Version: "5.0.0",
				Dotnet: &DotnetPackage{
					PackageGroup: []string{
						"Google.Cloud.Spanner.Admin.Database.V1",
						"Google.Cloud.Spanner.Admin.Instance.V1",
						"Google.Cloud.Spanner.Common.V1",
						"Google.Cloud.Spanner.Data",
						"Google.Cloud.Spanner.V1",
					},
				},
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestNodejsRead(t *testing.T) {
	got, err := yaml.Read[Config]("testdata/nodejs/librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := &Config{
		Language: LanguageNodejs,
		Sources: &Sources{
			Googleapis: &Source{
				Commit: "9fcfbea0aa5b50fa22e190faceb073d74504172b",
				SHA256: "81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98",
			},
		},
		Default: &Default{
			Output: "packages",
			Keep:   []string{"CHANGELOG.md"},
		},
		Libraries: []*Library{
			{
				Name:    "google-cloud-batch",
				Version: "1.5.0",
			},
			{
				Name:    "google-cloud-monitoring",
				Version: "1.5.0",
				APIs: []*API{
					{Path: "google/monitoring/v3"},
				},
			},
			{
				Name:    "google-cloud-accessapproval",
				Version: "4.2.0",
				Nodejs: &NodejsPackage{
					PackageName: "@google-cloud/access-approval",
				},
			},
			{
				Name:    "google-cloud-speech",
				Version: "7.0.0",
				Nodejs: &NodejsPackage{
					Dependencies: map[string]string{
						"@google-cloud/common": "^5.0.0",
						"pumpify":              "^2.0.1",
					},
				},
			},
			{
				Name:    "google-cloud-aiplatform",
				Version: "4.0.0",
				APIs: []*API{
					{Path: "google/cloud/aiplatform/v1"},
					{Path: "google/cloud/aiplatform/v1beta1"},
				},
				Keep: []string{
					"src/decorator.ts",
					"src/helpers.ts",
					"src/index.ts",
					"src/value-converter.ts",
				},
			},
			{
				Name:    "google-cloud-translate",
				Version: "9.1.0",
				Nodejs: &NodejsPackage{
					BundleConfig: "google/cloud/translate/v3/translate_gapic.yaml",
					ExtraProtocParameters: []string{
						"metadata",
						"auto-populate-field-oauth-scope",
					},
					HandwrittenLayer: true,
					MainService:      "translate",
					Mixins:           "none",
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
		t.Fatal(err)
	}

	got, err := yaml.Read[Config](newFile)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
