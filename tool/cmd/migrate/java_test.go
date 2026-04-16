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

package main

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
)

func TestApplyJavaProtoOverrides(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		want *config.JavaAPI
	}{
		{
			name: "google/cloud",
			path: "google/cloud",
			want: &config.JavaAPI{
				Path:           "google/cloud",
				ExcludedProtos: []string{"google/cloud/common_resources.proto"},
			},
		},
		{
			name: "aiplatform/v1beta1",
			path: "google/cloud/aiplatform/v1beta1",
			want: &config.JavaAPI{
				Path: "google/cloud/aiplatform/v1beta1",
				ExcludedProtos: []string{
					"google/cloud/aiplatform/v1beta1/schema/io_format.proto",
					"google/cloud/aiplatform/v1beta1/schema/annotation_payload.proto",
					"google/cloud/aiplatform/v1beta1/schema/annotation_spec_color.proto",
					"google/cloud/aiplatform/v1beta1/schema/data_item_payload.proto",
					"google/cloud/aiplatform/v1beta1/schema/dataset_metadata.proto",
					"google/cloud/aiplatform/v1beta1/schema/geometry.proto",
				},
			},
		},
		{
			name: "filestore",
			path: "google/cloud/filestore/v1",
			want: &config.JavaAPI{
				Path:             "google/cloud/filestore/v1",
				AdditionalProtos: []string{"google/cloud/common/operation_metadata.proto"},
			},
		},
		{
			name: "oslogin",
			path: "google/cloud/oslogin/v1",
			want: &config.JavaAPI{
				Path:             "google/cloud/oslogin/v1",
				AdditionalProtos: []string{"google/cloud/oslogin/common/common.proto"},
			},
		},
		{
			name: "google/rpc",
			path: "google/rpc",
			want: &config.JavaAPI{
				Path:           "google/rpc",
				ExcludedProtos: []string{"google/rpc/http.proto"},
			},
		},
		{
			name: "no override",
			path: "google/cloud/language/v1",
			want: &config.JavaAPI{
				Path: "google/cloud/language/v1",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := &config.JavaAPI{Path: test.path}
			applyJavaProtoOverrides(got)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunJavaMigration(t *testing.T) {
	fetchSourceWithCommit = func(ctx context.Context, endpoints *fetch.Endpoints, commitish string) (*config.Source, error) {
		return &config.Source{
			Commit: commitish,
			SHA256: "sha123",
			Dir:    "../../internal/testdata/googleapis",
		}, nil
	}
	for _, test := range []struct {
		name     string
		repoPath string
		wantErr  error
		insert   bool
	}{
		{
			name:     "success",
			repoPath: "testdata/run/success-java",
		},
		{
			name:     "tidy_failed",
			repoPath: "testdata/run/tidy-fails-java",
			wantErr:  errTidyFailed,
		},
		{
			name:     "no_generation_config",
			repoPath: "testdata/run/no-config",
			wantErr:  fs.ErrNotExist,
		},
		{
			name:     "insert_markers",
			repoPath: "testdata/run/success-java",
			insert:   true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.CopyFS(dir, os.DirFS(test.repoPath)); err != nil {
				t.Fatal(err)
			}
			writeVersionsFile(t, dir, "")
			if test.insert {
				// Create dummy pom.xml to be updated
				libDir := filepath.Join(dir, "java-language-v1")
				clientDir := filepath.Join(libDir, "google-cloud-language-v1")
				if err := os.MkdirAll(clientDir, 0755); err != nil {
					t.Fatal(err)
				}
				pomContent := `<project>
  <dependencies>
    <dependency>
      <groupId>com.google.api.grpc</groupId>
      <artifactId>proto-google-cloud-language-v1-v1</artifactId>
    </dependency>
  </dependencies>
</project>`
				if err := os.WriteFile(filepath.Join(clientDir, "pom.xml"), []byte(pomContent), 0644); err != nil {
					t.Fatal(err)
				}
			}
			err := runJavaMigration(t.Context(), dir, test.insert)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error %v, got %v", test.wantErr, err)
			}
			if test.insert {
				// Verify markers were inserted
				pomPath := filepath.Join(dir, "java-language-v1", "google-cloud-language-v1", "pom.xml")
				content, err := os.ReadFile(pomPath)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(content), managedProtoStartMarker) {
					t.Errorf("markers not found in %s", pomPath)
				}
			}
		})
	}
}

func TestBuildConfig(t *testing.T) {
	for _, test := range []struct {
		name     string
		gen      *GenerationConfig
		versions map[string]string
		src      *config.Source
		want     *config.Config
	}{
		{
			name: "prioritize library_name over api_shortname",
			gen: &GenerationConfig{
				Libraries: []LibraryConfig{
					{
						LibraryName:  "language-v1",
						APIShortName: "language",
						GAPICs: []GAPICConfig{
							{ProtoPath: "google/cloud/language/v1"},
						},
					},
				},
			},
			src: &config.Source{Dir: "../../internal/testdata/googleapis"},
			want: &config.Config{
				Language: "java",
				Repo:     "googleapis/google-cloud-java",
				Default: &config.Default{
					Java: &config.JavaModule{},
				},
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "../../internal/testdata/googleapis"},
				},
				Libraries: []*config.Library{
					{
						Name: "language-v1",
						APIs: []*config.API{
							{Path: "google/cloud/language/v1"},
						},
						Java: &config.JavaModule{},
					},
				},
			},
		},
		{
			name: "fallback to api_shortname",
			gen: &GenerationConfig{
				Libraries: []LibraryConfig{
					{
						APIShortName: "language",
						GAPICs: []GAPICConfig{
							{ProtoPath: "google/cloud/language/v1"},
						},
					},
				},
			},
			src: &config.Source{Dir: "../../internal/testdata/googleapis"},
			want: &config.Config{
				Language: "java",
				Repo:     "googleapis/google-cloud-java",
				Default: &config.Default{
					Java: &config.JavaModule{},
				},
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "../../internal/testdata/googleapis"},
				},
				Libraries: []*config.Library{
					{
						Name: "language",
						APIs: []*config.API{
							{Path: "google/cloud/language/v1"},
						},
						Java: &config.JavaModule{},
					},
				},
			},
		},
		{
			name: "multiple libraries",
			gen: &GenerationConfig{
				Libraries: []LibraryConfig{
					{
						LibraryName: "vision",
						GAPICs: []GAPICConfig{
							{ProtoPath: "google/cloud/vision/v1"},
						},
					},
					{
						APIShortName: "language",
						GAPICs: []GAPICConfig{
							{ProtoPath: "google/cloud/language/v1"},
						},
					},
				},
			},
			src: &config.Source{Dir: "../../internal/testdata/googleapis"},
			want: &config.Config{
				Language: "java",
				Repo:     "googleapis/google-cloud-java",
				Default: &config.Default{
					Java: &config.JavaModule{},
				},
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "../../internal/testdata/googleapis"},
				},
				Libraries: []*config.Library{
					{
						Name: "vision",
						APIs: []*config.API{
							{Path: "google/cloud/vision/v1"},
						},
						Java: &config.JavaModule{},
					},
					{
						Name: "language",
						APIs: []*config.API{
							{Path: "google/cloud/language/v1"},
						},
						Java: &config.JavaModule{},
					},
				},
			},
		},
		{
			name: "all java fields and overrides",
			gen: &GenerationConfig{
				LibrariesBomVersion: "1.2.3",
				Libraries: []LibraryConfig{
					{
						LibraryName:           "pubsub",
						APIShortName:          "pubsub",
						APIID:                 "pubsub.googleapis.com",
						APIDescription:        "Pub/Sub description",
						APIReference:          "https://api-ref.com",
						ClientDocumentation:   "https://client-doc.com",
						CloudAPI:              func(b bool) *bool { return &b }(false),
						CodeownerTeam:         "team-pubsub",
						DistributionName:      "com.google.cloud:google-cloud-pubsub",
						ExcludedDependencies:  "dep1,dep2",
						ExcludedPoms:          "pom1,pom2",
						ExtraVersionedModules: "module1",
						GroupID:               "com.google.cloud",
						IssueTracker:          "https://tracker.com",
						LibraryType:           "GAPIC_AUTO",
						MinJavaVersion:        11,
						NamePretty:            "Pub/Sub API",
						ProductDocumentation:  "https://product-doc.com",
						RecommendedPackage:    "com.google.cloud.pubsub",
						ReleaseLevel:          "stable",
						RequiresBilling:       func(b bool) *bool { return &b }(false),
						RestDocumentation:     "https://rest-doc.com",
						RpcDocumentation:      "https://rpc-doc.com",
						Transport:             "grpc",
						GAPICs: []GAPICConfig{
							{ProtoPath: "google/pubsub/v1"},
						},
					},
				},
			},
			src: &config.Source{Dir: "../../internal/testdata/googleapis"},
			want: &config.Config{
				Language: "java",
				Repo:     "googleapis/google-cloud-java",
				Default: &config.Default{
					Java: &config.JavaModule{
						LibrariesBOMVersion: "1.2.3",
					},
				},
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "../../internal/testdata/googleapis"},
				},
				Libraries: []*config.Library{
					{
						Name: "pubsub",
						APIs: []*config.API{
							{Path: "google/pubsub/v1"},
						},
						Java: &config.JavaModule{
							APIIDOverride:                "pubsub.googleapis.com",
							APIReference:                 "https://api-ref.com",
							APIDescriptionOverride:       "Pub/Sub description",
							ClientDocumentationOverride:  "https://client-doc.com",
							NonCloudAPI:                  true,
							CodeownerTeam:                "team-pubsub",
							DistributionNameOverride:     "com.google.cloud:google-cloud-pubsub",
							ExcludedDependencies:         "dep1,dep2",
							ExcludedPOMs:                 "pom1,pom2",
							ExtraVersionedModules:        "module1",
							GroupID:                      "com.google.cloud",
							IssueTrackerOverride:         "https://tracker.com",
							LibraryTypeOverride:          "GAPIC_AUTO",
							MinJavaVersion:               11,
							NamePrettyOverride:           "Pub/Sub API",
							ProductDocumentationOverride: "https://product-doc.com",
							RecommendedPackage:           "com.google.cloud.pubsub",
							BillingNotRequired:           true,
							RestDocumentation:            "https://rest-doc.com",
							RpcDocumentation:             "https://rpc-doc.com",
						},
					},
				},
			},
		},
		{
			name: "version lookup",
			gen: &GenerationConfig{
				Libraries: []LibraryConfig{
					{
						APIShortName:     "accessapproval",
						DistributionName: "com.google.cloud:google-cloud-accessapproval",
						GAPICs: []GAPICConfig{
							{ProtoPath: "google/cloud/accessapproval/v1"},
						},
					},
					{
						APIShortName: "aiplatform",
						GAPICs: []GAPICConfig{
							{ProtoPath: "google/cloud/aiplatform/v1"},
						},
					},
				},
			},
			versions: map[string]string{
				"google-cloud-java":           "1.79.0",
				"google-cloud-accessapproval": "2.86.0",
				"google-cloud-aiplatform":     "3.86.0",
			},
			src: &config.Source{Dir: "../../internal/testdata/googleapis"},
			want: &config.Config{
				Language: "java",
				Repo:     "googleapis/google-cloud-java",
				Default: &config.Default{
					Java: &config.JavaModule{},
				},
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "../../internal/testdata/googleapis"},
				},
				Libraries: []*config.Library{
					{
						Name:         "google-cloud-java",
						Version:      "1.79.0",
						SkipGenerate: true,
					},
					{
						Name:    "accessapproval",
						Version: "2.86.0",
						APIs: []*config.API{
							{Path: "google/cloud/accessapproval/v1"},
						},
						Java: &config.JavaModule{
							DistributionNameOverride: "com.google.cloud:" + "google-cloud-accessapproval",
						},
					},
					{
						Name:    "aiplatform",
						Version: "3.86.0",
						APIs: []*config.API{
							{Path: "google/cloud/aiplatform/v1"},
						},
						Java: &config.JavaModule{},
					},
				},
			},
		},
		{
			name: "api shortname overrides",
			gen: &GenerationConfig{
				Libraries: []LibraryConfig{
					{APIShortName: "beyondcorp-appconnections"},
				},
			},
			src: &config.Source{Dir: "../../internal/testdata/googleapis"},
			want: &config.Config{
				Language: "java",
				Repo:     "googleapis/google-cloud-java",
				Default: &config.Default{
					Java: &config.JavaModule{},
				},
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "../../internal/testdata/googleapis"},
				},
				Libraries: []*config.Library{
					{
						Name: "beyondcorp-appconnections",
						Java: &config.JavaModule{
							APIShortnameOverride: "beyondcorp-appconnections",
						},
					},
				},
			},
		},
		{
			name: "proto-only API",
			gen: &GenerationConfig{
				Libraries: []LibraryConfig{
					{
						APIShortName: "gkehub",
						GAPICs: []GAPICConfig{
							{ProtoPath: "google/cloud/gkehub/policycontroller/v1beta"},
						},
					},
				},
			},
			src: &config.Source{Dir: "../../../internal/testdata/googleapis"},
			want: &config.Config{
				Language: "java",
				Repo:     "googleapis/google-cloud-java",
				Default: &config.Default{
					Java: &config.JavaModule{},
				},
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "../../../internal/testdata/googleapis"},
				},
				Libraries: []*config.Library{
					{
						Name: "gkehub",
						APIs: []*config.API{
							{Path: "google/cloud/gkehub/policycontroller/v1beta"},
						},
						Java: &config.JavaModule{
							JavaAPIs: []*config.JavaAPI{
								{
									Path:      "google/cloud/gkehub/policycontroller/v1beta",
									Samples:   new(false),
									ProtoOnly: true,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "proto artifact overrides",
			gen: &GenerationConfig{
				Libraries: []LibraryConfig{
					{
						APIShortName: "gsuite-addons",
						GAPICs: []GAPICConfig{
							{ProtoPath: "google/apps/script/type"},
						},
					},
				},
			},
			src: &config.Source{Dir: "testdata/googleapis"},
			want: &config.Config{
				Language: "java",
				Repo:     "googleapis/google-cloud-java",
				Default: &config.Default{
					Java: &config.JavaModule{},
				},
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "testdata/googleapis"},
				},
				Libraries: []*config.Library{
					{
						Name: "gsuite-addons",
						APIs: []*config.API{
							{Path: "google/apps/script/type"},
						},
						Java: &config.JavaModule{
							JavaAPIs: []*config.JavaAPI{
								{
									Path:                    "google/apps/script/type",
									ProtoArtifactIDOverride: "proto-google-apps-script-type-protos",
									ProtoOnly:               true,
									Samples:                 new(false),
								},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildConfig(test.gen, ".", test.src, test.versions)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestShouldExcludeSamples(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  string
		info *javaGAPICInfo
		want bool
	}{
		{
			name: "exclude if info.Samples is false",
			lib:  "any-lib",
			info: &javaGAPICInfo{Samples: false},
			want: true,
		},
		{
			name: "exclude if lib is in excludedSamplesLibraries",
			lib:  "datastore",
			info: &javaGAPICInfo{Samples: true},
			want: true,
		},
		{
			name: "include if info.Samples is true and lib not in map",
			lib:  "any-lib",
			info: &javaGAPICInfo{Samples: true},
			want: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldExcludeSamples(test.lib, test.info); got != test.want {
				t.Errorf("shouldExcludeSamples(%q, %+v) = %v, want %v", test.lib, test.info, got, test.want)
			}
		})
	}
}

func TestBuildConfig_ArtifactIDOverrides(t *testing.T) {
	gen := &GenerationConfig{
		Libraries: []LibraryConfig{
			{
				LibraryName: "datastore",
				GAPICs: []GAPICConfig{
					{ProtoPath: "google/datastore/admin/v1"},
				},
			},
		},
	}
	srcDir := t.TempDir()
	buildFile := filepath.Join(srcDir, "google/datastore/admin/v1", "BUILD.bazel")
	if err := os.MkdirAll(filepath.Dir(buildFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(buildFile, []byte(`java_gapic_library(name = "datastore_java_gapic")`), 0644); err != nil {
		t.Fatal(err)
	}

	want := &config.Config{
		Language: "java",
		Repo:     "googleapis/google-cloud-java",
		Default: &config.Default{
			Java: &config.JavaModule{},
		},
		Sources: &config.Sources{
			Googleapis: &config.Source{Dir: srcDir},
		},
		Libraries: []*config.Library{
			{
				Name: "datastore",
				APIs: []*config.API{
					{Path: "google/datastore/admin/v1"},
				},
				Java: &config.JavaModule{
					JavaAPIs: []*config.JavaAPI{
						{
							Path:                    "google/datastore/admin/v1",
							Samples:                 func(b bool) *bool { return &b }(false),
							ProtoArtifactIDOverride: "proto-google-cloud-datastore-admin-v1",
							GRPCArtifactIDOverride:  "grpc-google-cloud-datastore-admin-v1",
						},
					},
				},
			},
		},
	}

	got := buildConfig(gen, ".", &config.Source{Dir: srcDir}, nil)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestBuildConfig_OwlBotKeep(t *testing.T) {
	repoPath := "testdata/google-cloud-java"
	gen := &GenerationConfig{
		Libraries: []LibraryConfig{
			{
				APIShortName: "vision",
				GAPICs: []GAPICConfig{
					{ProtoPath: "google/cloud/vision/v1"},
				},
			},
		},
	}
	got := buildConfig(gen, repoPath, &config.Source{Dir: "../../internal/testdata/googleapis"}, nil)
	wantKeep := []string{
		"proto-google-cloud-vision-v1/src/main/java/com/google/cloud/vision/v1/ImageName.java",
		"google-cloud-vision/src/test/java/com/google/cloud/vision/it/ITSystemTest.java",
		"google-cloud-vision/src/test/resources/city.jpg",
		"google-cloud-vision/src/test/resources/face_no_surprise.jpg",
		"google-cloud-vision/src/test/resources/landmark.jpg",
		"google-cloud-vision/src/test/resources/logos.png",
		"google-cloud-vision/src/test/resources/puppies.jpg",
		"google-cloud-vision/src/test/resources/text.jpg",
		"google-cloud-vision/src/test/resources/wakeupcat.jpg",
	}
	if diff := cmp.Diff(wantKeep, got.Libraries[0].Keep); diff != "" {
		t.Errorf("mismatch in Keep field (-want +got):\n%s", diff)
	}
}

func TestReadVersions(t *testing.T) {
	path := writeVersionsFile(t, t.TempDir(), `# Format:
# module:released-version:current-version

google-cloud-accessapproval:2.86.0:2.87.0-SNAPSHOT
google-cloud-aiplatform:3.86.0:3.87.0-SNAPSHOT
`)

	got, err := readVersions(path)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"google-cloud-accessapproval": "2.87.0-SNAPSHOT",
		"google-cloud-aiplatform":     "3.87.0-SNAPSHOT",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestReadVersions_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
	}{
		{
			name:    "too few parts",
			content: "a:b",
		},
		{
			name:    "too many parts",
			content: "a:b:c:d",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := writeVersionsFile(t, t.TempDir(), test.content)
			if _, err := readVersions(path); err == nil {
				t.Errorf("readVersions(%q) error = nil, want error", test.content)
			}
		})
	}
}

func TestReadVersions_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "non-existent")
	if _, err := readVersions(path); err == nil {
		t.Error("readVersions() error = nil, want error")
	}
}

func writeVersionsFile(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "versions.txt")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseJavaBazel(t *testing.T) {
	for _, test := range []struct {
		name          string
		googleapisDir string
		buildPath     string
		want          *javaGAPICInfo
	}{
		{
			name:          "success",
			googleapisDir: "testdata/parse-bazel/success",
			buildPath:     "google/cloud/bigquery/analyticshub/v1",
			want: &javaGAPICInfo{
				Samples: true,
				AdditionalProtos: []string{
					"google/cloud/common_resources.proto",
				},
			},
		},
		{
			name:          "no GAPIC rules",
			googleapisDir: "testdata/parse-bazel/no-gapic-rule",
			want: &javaGAPICInfo{
				Samples: false,
				AdditionalProtos: []string{
					"google/cloud/common_resources.proto",
				},
				ProtoOnly: true,
			},
		},
		{
			name:          "complex-deps",
			googleapisDir: "testdata/parse-bazel/complex-deps",
			buildPath:     "google/cloud/aiplatform/v1",
			want: &javaGAPICInfo{
				AdditionalProtos: []string{
					"google/cloud/common_resources.proto",
					"google/cloud/location/locations.proto",
					"google/iam/v1/iam_policy.proto",
				},
				ProtoOnly: true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseJavaBazel(test.googleapisDir, test.buildPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWrapBlocks(t *testing.T) {
	lines := []string{
		"<dependencies>",
		"  <dependency>",
		"    <artifactId>proto-v1</artifactId>",
		"  </dependency>",
		"  <dependency>",
		"    <artifactId>grpc-v1</artifactId>",
		"  </dependency>",
		"  <dependency>",
		"    <artifactId>other</artifactId>",
		"  </dependency>",
		"</dependencies>",
	}
	for _, test := range []struct {
		name    string
		targets []string
		start   string
		end     string
		want    []string
	}{
		{
			name:    "wrap existing proto",
			targets: toArtifactTags([]string{"proto-v1"}),
			start:   managedProtoStartMarker,
			end:     managedProtoEndMarker,
			want: []string{
				"<dependencies>",
				"  " + managedProtoStartMarker,
				"  <dependency>",
				"    <artifactId>proto-v1</artifactId>",
				"  </dependency>",
				"  " + managedProtoEndMarker,
				"  <dependency>",
				"    <artifactId>grpc-v1</artifactId>",
				"  </dependency>",
				"  <dependency>",
				"    <artifactId>other</artifactId>",
				"  </dependency>",
				"</dependencies>",
			},
		},
		{
			name:    "wrap existing grpc",
			targets: toArtifactTags([]string{"grpc-v1"}),
			start:   managedGrpcStartMarker,
			end:     managedGrpcEndMarker,
			want: []string{
				"<dependencies>",
				"  <dependency>",
				"    <artifactId>proto-v1</artifactId>",
				"  </dependency>",
				"  " + managedGrpcStartMarker,
				"  <dependency>",
				"    <artifactId>grpc-v1</artifactId>",
				"  </dependency>",
				"  " + managedGrpcEndMarker,
				"  <dependency>",
				"    <artifactId>other</artifactId>",
				"  </dependency>",
				"</dependencies>",
			},
		},
		{
			name:    "no match",
			targets: toArtifactTags([]string{"non-existent"}),
			start:   managedProtoStartMarker,
			end:     managedProtoEndMarker,
			want:    lines,
		},
		{
			name:    "empty targets",
			targets: []string{},
			start:   managedProtoStartMarker,
			end:     managedProtoEndMarker,
			want:    lines,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := wrapBlocks(wrapArgs{
				lines:       lines,
				targets:     test.targets,
				startMarker: test.start,
				endMarker:   test.end,
				startTag:    "<dependency>",
				endTag:      "</dependency>",
			})
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetModuleArtifactIDs(t *testing.T) {
	lib := &config.Library{
		Name: "vision",
		APIs: []*config.API{
			{Path: "google/cloud/vision/v1"},
			{Path: "google/cloud/vision/v1p1beta1"},
		},
	}
	ids := getModuleArtifactIDs(lib)
	wantProto := []string{"proto-google-cloud-vision-v1", "proto-google-cloud-vision-v1p1beta1"}
	wantGrpc := []string{"grpc-google-cloud-vision-v1", "grpc-google-cloud-vision-v1p1beta1"}
	if diff := cmp.Diff(wantProto, ids.Protos); diff != "" {
		t.Errorf("mismatch in protoIDs (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantGrpc, ids.GRPCs); diff != "" {
		t.Errorf("mismatch in grpcIDs (-want +got):\n%s", diff)
	}
}

func TestInsertMarkers(t *testing.T) {
	for _, test := range []struct {
		name         string
		pomContent   string
		protoIDs     []string
		grpcIDs      []string
		wantContains []string
	}{
		{
			name: "contiguous",
			pomContent: `
<dependencies>
  <dependency>
    <groupId>com.google.api.grpc</groupId>
    <artifactId>proto-google-cloud-vision-v1</artifactId>
  </dependency>
  <dependency>
    <groupId>com.google.api.grpc</groupId>
    <artifactId>grpc-google-cloud-vision-v1</artifactId>
  </dependency>
</dependencies>
`,
			protoIDs: []string{"proto-google-cloud-vision-v1"},
			grpcIDs:  []string{"grpc-google-cloud-vision-v1"},
			wantContains: []string{
				managedProtoStartMarker,
				"proto-google-cloud-vision-v1",
				managedProtoEndMarker,
				managedGrpcStartMarker,
				"grpc-google-cloud-vision-v1",
				managedGrpcEndMarker,
			},
		},
		{
			name: "non-contiguous",
			pomContent: `
<dependencies>
  <dependency>
    <groupId>com.google.api.grpc</groupId>
    <artifactId>proto-google-cloud-vision-v1</artifactId>
  </dependency>
  <dependency>
    <groupId>junit</groupId>
    <artifactId>junit</artifactId>
    <scope>test</scope>
  </dependency>
  <dependency>
    <groupId>com.google.api.grpc</groupId>
    <artifactId>grpc-google-cloud-vision-v1</artifactId>
  </dependency>
</dependencies>
`,
			protoIDs: []string{"proto-google-cloud-vision-v1"},
			grpcIDs:  []string{"grpc-google-cloud-vision-v1"},
			wantContains: []string{
				managedProtoStartMarker,
				"proto-google-cloud-vision-v1",
				managedProtoEndMarker,
				"junit",
				managedGrpcStartMarker,
				"grpc-google-cloud-vision-v1",
				managedGrpcEndMarker,
			},
		},
		{
			name: "multiple-proto-non-contiguous",
			pomContent: `
<dependencies>
  <dependency>
    <groupId>com.google.api.grpc</groupId>
    <artifactId>proto-google-cloud-vision-v1</artifactId>
  </dependency>
  <dependency>
    <groupId>junit</groupId>
    <artifactId>junit</artifactId>
    <scope>test</scope>
  </dependency>
  <dependency>
    <groupId>com.google.api.grpc</groupId>
    <artifactId>proto-google-cloud-vision-v1p1beta1</artifactId>
  </dependency>
</dependencies>
`,
			protoIDs: []string{"proto-google-cloud-vision-v1", "proto-google-cloud-vision-v1p1beta1"},
			grpcIDs:  []string{},
			wantContains: []string{
				managedProtoStartMarker,
				"proto-google-cloud-vision-v1",
				"proto-google-cloud-vision-v1p1beta1",
				managedProtoEndMarker,
				"junit",
			},
		},
		{
			name: "mixed-types-non-contiguous",
			pomContent: `
<dependencies>
  <dependency>
    <groupId>com.google.api.grpc</groupId>
    <artifactId>proto-google-cloud-vision-v1</artifactId>
  </dependency>
  <dependency>
    <groupId>com.google.api.grpc</groupId>
    <artifactId>grpc-google-cloud-vision-v1</artifactId>
  </dependency>
  <dependency>
    <groupId>com.google.api.grpc</groupId>
    <artifactId>proto-google-cloud-vision-v1p1beta1</artifactId>
  </dependency>
</dependencies>
`,
			protoIDs: []string{"proto-google-cloud-vision-v1", "proto-google-cloud-vision-v1p1beta1"},
			grpcIDs:  []string{"grpc-google-cloud-vision-v1"},
			wantContains: []string{
				managedProtoStartMarker,
				"proto-google-cloud-vision-v1",
				"proto-google-cloud-vision-v1p1beta1",
				managedProtoEndMarker,
				managedGrpcStartMarker,
				"grpc-google-cloud-vision-v1",
				managedGrpcEndMarker,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Setup test repo and call insertMarkers (simplified logic for test)
			lines := strings.Split(test.pomContent, "\n")
			gotLines := wrapBlocks(wrapArgs{
				lines:       lines,
				targets:     toArtifactTags(test.protoIDs),
				startMarker: managedProtoStartMarker,
				endMarker:   managedProtoEndMarker,
				startTag:    "<dependency>",
				endTag:      "</dependency>",
			})
			gotLines = wrapBlocks(wrapArgs{
				lines:       gotLines,
				targets:     toArtifactTags(test.grpcIDs),
				startMarker: managedGrpcStartMarker,
				endMarker:   managedGrpcEndMarker,
				startTag:    "<dependency>",
				endTag:      "</dependency>",
			})
			got := strings.Join(gotLines, "\n")

			for _, want := range test.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("%s: expected %q in output, but not found\nOutput:\n%s", test.name, want, got)
				}
			}

			// In non-contiguous case, verify that junit is NOT wrapped by markers if we fix it.
			if test.name == "multiple-proto-non-contiguous" {
				protoBlock := got[strings.Index(got, managedProtoStartMarker):strings.Index(got, managedProtoEndMarker)]
				if strings.Contains(protoBlock, "junit") {
					t.Errorf("multiple-proto-non-contiguous: junit should NOT be inside proto markers, but found in block:\n%s", protoBlock)
				}
			}
		})
	}
}

func TestInsertMarkers_Full(t *testing.T) {
	dir := t.TempDir()
	libName := "vision"
	artifactID := "google-cloud-vision"
	repoPath := dir
	libDir := filepath.Join(repoPath, "java-"+libName)
	clientDir := filepath.Join(libDir, artifactID)
	bomDir := filepath.Join(libDir, artifactID+"-bom")

	if err := os.MkdirAll(clientDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bomDir, 0755); err != nil {
		t.Fatal(err)
	}

	clientPOM := `<project>
  <dependencies>
    <dependency>
      <groupId>com.google.api.grpc</groupId>
      <artifactId>proto-google-cloud-vision-v1</artifactId>
    </dependency>
  </dependencies>
</project>`

	parentPOM := `<project>
  <dependencyManagement>
    <dependencies>
      <dependency>
        <groupId>com.google.cloud</groupId>
        <artifactId>google-cloud-vision</artifactId>
      </dependency>
      <dependency>
        <groupId>com.google.cloud</groupId>
        <artifactId>google-cloud-vision-bom</artifactId>
      </dependency>
    </dependencies>
  </dependencyManagement>
  <modules>
    <module>google-cloud-vision</module>
    <module>google-cloud-vision-bom</module>
    <module>proto-google-cloud-vision-v1</module>
  </modules>
</project>`

	bomPOM := `<project>
  <dependencyManagement>
    <dependencies>
      <dependency>
        <groupId>com.google.cloud</groupId>
        <artifactId>google-cloud-vision</artifactId>
      </dependency>
    </dependencies>
  </dependencyManagement>
</project>`

	if err := os.WriteFile(filepath.Join(clientDir, "pom.xml"), []byte(clientPOM), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "pom.xml"), []byte(parentPOM), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bomDir, "pom.xml"), []byte(bomPOM), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Libraries: []*config.Library{
			{
				Name: libName,
				APIs: []*config.API{
					{Path: "google/cloud/vision/v1"},
				},
				Java: &config.JavaModule{
					DistributionNameOverride: "com.google.cloud:" + artifactID,
				},
			},
		},
	}
	if err := insertMarkers(repoPath, cfg); err != nil {
		t.Fatal(err)
	}

	// Verify Client
	clientGot, _ := os.ReadFile(filepath.Join(clientDir, "pom.xml"))
	if !strings.Contains(string(clientGot), managedProtoStartMarker) {
		t.Error("client pom missing proto markers")
	}

	// Verify Parent
	parentGot, _ := os.ReadFile(filepath.Join(libDir, "pom.xml"))
	if !strings.Contains(string(parentGot), managedDepsStartMarker) {
		t.Error("parent pom missing dependency markers")
	}
	if !strings.Contains(string(parentGot), managedModulesStartMarker) {
		t.Error("parent pom missing module markers")
	}
	if !strings.Contains(string(parentGot), "proto-google-cloud-vision-v1") {
		t.Error("parent pom missing proto module")
	}

	// Verify BOM is NOT inside module markers
	parentStr := string(parentGot)
	modulesStart := strings.Index(parentStr, managedModulesStartMarker)
	modulesEnd := strings.Index(parentStr, managedModulesEndMarker)
	if modulesStart != -1 && modulesEnd != -1 && modulesStart < modulesEnd {
		modulesBlock := parentStr[modulesStart:modulesEnd]
		if strings.Contains(modulesBlock, artifactID+"-bom") {
			t.Errorf("parent pom should not include BOM module %s inside markers", artifactID+"-bom")
		}
	}

	// Verify BOM
	bomGot, _ := os.ReadFile(filepath.Join(bomDir, "pom.xml"))
	if !strings.Contains(string(bomGot), managedDepsStartMarker) {
		t.Error("bom pom missing dependency markers")
	}
}
