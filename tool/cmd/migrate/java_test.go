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
	"errors"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestRunJavaMigration(t *testing.T) {
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
			repoPath: "testdata/run/non-existent",
			wantErr:  os.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outputPath := "librarian.yaml"
			t.Cleanup(func() {
				if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
					t.Fatalf("cleanup: %v", err)
				}
			})
			err := runJavaMigration(t.Context(), test.repoPath)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("expected error %v, got %v", test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestBuildConfig(t *testing.T) {
	for _, test := range []struct {
		name string
		gen  *GenerationConfig
		want *config.Config
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
				Default:  &config.Default{},
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "../../googleapis"},
				},
				Libraries: []*config.Library{
					{
						Name:   "language-v1",
						Output: "java-language-v1",
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
				Default:  &config.Default{},
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "../../googleapis"},
				},
				Libraries: []*config.Library{
					{
						Name:   "language",
						Output: "java-language",
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
				Default:  &config.Default{},
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "../../googleapis"},
				},
				Libraries: []*config.Library{
					{
						Name:   "vision",
						Output: "java-vision",
						APIs: []*config.API{
							{Path: "google/cloud/vision/v1"},
						},
						Java: &config.JavaModule{},
					},
					{
						Name:   "language",
						Output: "java-language",
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
				Default:  &config.Default{},
				Sources: &config.Sources{
					Googleapis: &config.Source{Dir: "../../googleapis"},
				},
				Libraries: []*config.Library{
					{
						Name:         "pubsub",
						Output:       "java-pubsub",
						ReleaseLevel: "stable",
						Transport:    "grpc",
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
							ExcludedPoms:                 "pom1,pom2",
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
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildConfig(test.gen)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
