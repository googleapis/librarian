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
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.CopyFS(dir, os.DirFS(test.repoPath)); err != nil {
				t.Fatal(err)
			}
			writeVersionsFile(t, dir, "")
			err := runJavaMigration(t.Context(), dir)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestBuildConfig(t *testing.T) {
	for _, test := range []struct {
		name     string
		gen      *GenerationConfig
		versions map[string]string
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
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildConfig(test.gen, ".", &config.Source{Dir: "../../internal/testdata/googleapis"}, test.versions)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
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
				NoSamples: false,
				AdditionalProtos: []string{
					"google/cloud/common_resources.proto",
				},
			},
		},
		{
			name:          "no GAPIC rules",
			googleapisDir: "testdata/parse-bazel/no-gapic-rule",
			want: &javaGAPICInfo{
				AdditionalProtos: []string{
					"google/cloud/common_resources.proto",
				},
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

func TestWrapDependencies(t *testing.T) {
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
		name        string
		artifactIDs []string
		start       string
		end         string
		want        []string
	}{
		{
			name:        "wrap existing proto",
			artifactIDs: []string{"proto-v1"},
			start:       managedProtoStart,
			end:         managedProtoEnd,
			want: []string{
				"<dependencies>",
				"  " + managedProtoStart,
				"  <dependency>",
				"    <artifactId>proto-v1</artifactId>",
				"  </dependency>",
				"  " + managedProtoEnd,
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
			name:        "wrap existing grpc",
			artifactIDs: []string{"grpc-v1"},
			start:       managedGrpcStart,
			end:         managedGrpcEnd,
			want: []string{
				"<dependencies>",
				"  <dependency>",
				"    <artifactId>proto-v1</artifactId>",
				"  </dependency>",
				"  " + managedGrpcStart,
				"  <dependency>",
				"    <artifactId>grpc-v1</artifactId>",
				"  </dependency>",
				"  " + managedGrpcEnd,
				"  <dependency>",
				"    <artifactId>other</artifactId>",
				"  </dependency>",
				"</dependencies>",
			},
		},
		{
			name:        "no match",
			artifactIDs: []string{"non-existent"},
			start:       managedProtoStart,
			end:         managedProtoEnd,
			want:        lines,
		},
		{
			name:        "empty artifactIDs",
			artifactIDs: []string{},
			start:       managedProtoStart,
			end:         managedProtoEnd,
			want:        lines,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := wrapDependencies(lines, test.artifactIDs, test.start, test.end)
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
	protoIDs, grpcIDs := getModuleArtifactIDs(lib)
	wantProto := []string{"proto-google-cloud-vision-v1", "proto-google-cloud-vision-v1p1beta1"}
	wantGrpc := []string{"grpc-google-cloud-vision-v1", "grpc-google-cloud-vision-v1p1beta1"}
	if diff := cmp.Diff(wantProto, protoIDs); diff != "" {
		t.Errorf("mismatch in protoIDs (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantGrpc, grpcIDs); diff != "" {
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
				managedProtoStart,
				"proto-google-cloud-vision-v1",
				managedProtoEnd,
				managedGrpcStart,
				"grpc-google-cloud-vision-v1",
				managedGrpcEnd,
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
				managedProtoStart,
				"proto-google-cloud-vision-v1",
				managedProtoEnd,
				"junit",
				managedGrpcStart,
				"grpc-google-cloud-vision-v1",
				managedGrpcEnd,
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
				managedProtoStart,
				"proto-google-cloud-vision-v1",
				"proto-google-cloud-vision-v1p1beta1",
				managedProtoEnd,
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
				managedProtoStart,
				"proto-google-cloud-vision-v1",
				"proto-google-cloud-vision-v1p1beta1",
				managedProtoEnd,
				managedGrpcStart,
				"grpc-google-cloud-vision-v1",
				managedGrpcEnd,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Setup test repo and call insertMarkers (simplified logic for test)
			lines := strings.Split(test.pomContent, "\n")
			gotLines := wrapDependencies(lines, test.protoIDs, managedProtoStart, managedProtoEnd)
			gotLines = wrapDependencies(gotLines, test.grpcIDs, managedGrpcStart, managedGrpcEnd)
			got := strings.Join(gotLines, "\n")

			for _, want := range test.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("%s: expected %q in output, but not found\nOutput:\n%s", test.name, want, got)
				}
			}

			// In non-contiguous case, verify that junit is NOT wrapped by markers if we fix it.
			if test.name == "multiple-proto-non-contiguous" {
				protoBlock := got[strings.Index(got, managedProtoStart):strings.Index(got, managedProtoEnd)]
				if strings.Contains(protoBlock, "junit") {
					t.Errorf("multiple-proto-non-contiguous: junit should NOT be inside proto markers, but found in block:\n%s", protoBlock)
				}
			}
		})
	}
}
