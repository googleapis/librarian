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

package dart

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/common"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
)

func TestBuildCodec(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		want    map[string]string
	}{
		{
			name: "nil dart package",
			library: &config.Library{
				CopyrightYear: "2025",
				Version:       "0.1.0",
			},
			want: map[string]string{
				"copyright-year": "2025",
				"version":        "0.1.0",
			},
		},
		{
			name: "empty library",
			library: &config.Library{
				Dart: &config.DartPackage{},
			},
			want: map[string]string{},
		},
		{
			name: "dependencies",
			library: &config.Library{
				Dart: &config.DartPackage{
					Dependencies: "http: ^1.3.0",
				},
			},
			want: map[string]string{
				"dependencies": "http: ^1.3.0",
			},
		},
		{
			name: "dev dependencies",
			library: &config.Library{
				Dart: &config.DartPackage{
					DevDependencies: "test: ^1.0.0",
				},
			},
			want: map[string]string{
				"dev-dependencies": "test: ^1.0.0",
			},
		},
		{
			name: "extra imports",
			library: &config.Library{
				Dart: &config.DartPackage{
					ExtraImports: "package:googleapis_auth/auth.dart",
				},
			},
			want: map[string]string{
				"extra-imports": "package:googleapis_auth/auth.dart",
			},
		},
		{
			name: "library path override",
			library: &config.Library{
				Dart: &config.DartPackage{
					LibraryPathOverride: "lib/custom.dart",
				},
			},
			want: map[string]string{
				"library-path-override": "lib/custom.dart",
			},
		},
		{
			name: "not for publication",
			library: &config.Library{
				Dart: &config.DartPackage{
					NotForPublication: "true",
				},
			},
			want: map[string]string{
				"not-for-publication": "true",
			},
		},
		{
			name: "part file",
			library: &config.Library{
				Dart: &config.DartPackage{
					PartFile: "part 'src/common.dart';",
				},
			},
			want: map[string]string{
				"part-file": "part 'src/common.dart';",
			},
		},
		{
			name: "readme after title text",
			library: &config.Library{
				Dart: &config.DartPackage{
					ReadmeAfterTitleText: "**Note:** This package is experimental.",
				},
			},
			want: map[string]string{
				"readme-after-title-text": "**Note:** This package is experimental.",
			},
		},
		{
			name: "readme quickstart text",
			library: &config.Library{
				Dart: &config.DartPackage{
					ReadmeQuickstartText: "Run `dart pub add` to install this package.",
				},
			},
			want: map[string]string{
				"readme-quickstart-text": "Run `dart pub add` to install this package.",
			},
		},
		{
			name: "repository url",
			library: &config.Library{
				Dart: &config.DartPackage{
					RepositoryURL: "https://github.com/googleapis/google-cloud-dart",
				},
			},
			want: map[string]string{
				"repository-url": "https://github.com/googleapis/google-cloud-dart",
			},
		},
		{
			name: "packages map",
			library: &config.Library{
				Dart: &config.DartPackage{
					Packages: map[string]string{
						"package:googleapis_auth": "^2.0.0",
						"package:http":            "^1.3.0",
					},
				},
			},
			want: map[string]string{
				"package:googleapis_auth": "^2.0.0",
				"package:http":            "^1.3.0",
			},
		},
		{
			name: "prefixes map",
			library: &config.Library{
				Dart: &config.DartPackage{
					Prefixes: map[string]string{
						"prefix:google.protobuf": "pb",
						"prefix:google.api":      "api",
					},
				},
			},
			want: map[string]string{
				"prefix:google.protobuf": "pb",
				"prefix:google.api":      "api",
			},
		},
		{
			name: "protos map",
			library: &config.Library{
				Dart: &config.DartPackage{
					Protos: map[string]string{
						"proto:google.api":      "package:google_cloud_api/api.dart",
						"proto:google.protobuf": "package:protobuf/protobuf.dart",
					},
				},
			},
			want: map[string]string{
				"proto:google.api":      "package:google_cloud_api/api.dart",
				"proto:google.protobuf": "package:protobuf/protobuf.dart",
			},
		},
		{
			name: "all fields",
			library: &config.Library{
				CopyrightYear: "2025",
				Version:       "1.0.0",
				Dart: &config.DartPackage{
					APIKeysEnvironmentVariables: "GOOGLE_API_KEY",
					Dependencies:                "http: ^1.3.0",
					DevDependencies:             "test: ^1.0.0",
					ExtraImports:                "package:googleapis_auth/auth.dart",
					IssueTrackerURL:             "https://github.com/googleapis/google-cloud-dart/issues",
					LibraryPathOverride:         "lib/custom.dart",
					NotForPublication:           "false",
					PartFile:                    "part 'src/common.dart';",
					ReadmeAfterTitleText:        "**Note:** This package is experimental.",
					ReadmeQuickstartText:        "Run `dart pub add` to install.",
					RepositoryURL:               "https://github.com/googleapis/google-cloud-dart",
					Packages: map[string]string{
						"package:googleapis_auth": "^2.0.0",
					},
					Prefixes: map[string]string{
						"prefix:google.protobuf": "pb",
					},
					Protos: map[string]string{
						"proto:google.api": "package:google_cloud_api/api.dart",
					},
				},
			},
			want: map[string]string{
				"copyright-year":                 "2025",
				"version":                        "1.0.0",
				"api-keys-environment-variables": "GOOGLE_API_KEY",
				"dependencies":                   "http: ^1.3.0",
				"dev-dependencies":               "test: ^1.0.0",
				"extra-imports":                  "package:googleapis_auth/auth.dart",
				"issue-tracker-url":              "https://github.com/googleapis/google-cloud-dart/issues",
				"library-path-override":          "lib/custom.dart",
				"not-for-publication":            "false",
				"part-file":                      "part 'src/common.dart';",
				"readme-after-title-text":        "**Note:** This package is experimental.",
				"readme-quickstart-text":         "Run `dart pub add` to install.",
				"repository-url":                 "https://github.com/googleapis/google-cloud-dart",
				"package:googleapis_auth":        "^2.0.0",
				"prefix:google.protobuf":         "pb",
				"proto:google.api":               "package:google_cloud_api/api.dart",
			},
		},
		{
			name: "empty string fields not included",
			library: &config.Library{
				CopyrightYear: "",
				Version:       "1.0.0",
				Dart: &config.DartPackage{
					Dependencies:    "",
					DevDependencies: "",
					IssueTrackerURL: "https://github.com/googleapis/google-cloud-dart/issues",
					RepositoryURL:   "",
				},
			},
			want: map[string]string{
				"version":           "1.0.0",
				"issue-tracker-url": "https://github.com/googleapis/google-cloud-dart/issues",
			},
		},
		{
			name: "empty maps",
			library: &config.Library{
				Dart: &config.DartPackage{
					Packages: map[string]string{},
					Prefixes: map[string]string{},
					Protos:   map[string]string{},
				},
			},
			want: map[string]string{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildCodec(test.library)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestToSidekickConfig(t *testing.T) {
	googleapisDir := t.TempDir()
	for _, test := range []struct {
		name          string
		library       *config.Library
		channel       *config.API
		googleapisDir string
		want          *sidekickconfig.Config
		wantErr       error
	}{
		{
			name:    "empty library",
			library: &config.Library{},
			channel: &config.API{
				Path: "google/example/v1",
			},
			googleapisDir: googleapisDir,
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "dart",
					SpecificationFormat: "protobuf",
					ServiceConfig:       "",
					SpecificationSource: "google/example/v1",
				},
				Source: map[string]string{
					"googleapis-root": googleapisDir,
					"roots":           "googleapis",
				},
				Codec: map[string]string{},
			},
		},
		{
			name: "with description-override",
			library: &config.Library{
				DescriptionOverride: "this is a description override",
			},
			channel: &config.API{
				Path: "google/example/v1",
			},
			googleapisDir: googleapisDir,
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "dart",
					SpecificationFormat: "protobuf",
					ServiceConfig:       "",
					SpecificationSource: "google/example/v1",
				},
				Source: map[string]string{
					"googleapis-root":      googleapisDir,
					"description-override": "this is a description override",
					"roots":                "googleapis",
				},
				Codec: map[string]string{},
			},
		},
		{
			name: "with name-override",
			library: &config.Library{
				Dart: &config.DartPackage{
					NameOverride: "override-name",
				},
			},
			channel: &config.API{
				Path: "google/example/v1",
			},
			googleapisDir: googleapisDir,
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "dart",
					SpecificationFormat: "protobuf",
					ServiceConfig:       "",
					SpecificationSource: "google/example/v1",
				},
				Source: map[string]string{
					"googleapis-root": googleapisDir,
					"name-override":   "override-name",
					"roots":           "googleapis",
				},
				Codec: map[string]string{},
			},
		},
		{
			name: "with dart package",
			library: &config.Library{
				Dart: &config.DartPackage{
					APIKeysEnvironmentVariables: "GOOGLE_API_KEY",
					Dependencies:                "dep-1,dep-2",
					DevDependencies:             "dev-dep-1,dev-dep-2",
					ExtraImports:                "extra-imports",
					IssueTrackerURL:             "https://tracker/issues",
					LibraryPathOverride:         "library-path-override",
					NotForPublication:           "false",
					PartFile:                    "part-file",
					ReadmeAfterTitleText:        "readme-after-title-text",
					ReadmeQuickstartText:        "readme-quickstart-text",
					RepositoryURL:               "https://github.com/googleapis/google-cloud-dart",
					Packages: map[string]string{
						"package:googleapis_auth": "^2.0.0",
					},
					Prefixes: map[string]string{
						"prefix:google.logging.type": "logging_type",
					},
					Protos: map[string]string{
						"proto:google.api": "package:google_cloud_api/api.dart",
					},
					TitleOverride: "library-title-override",
				},
			},
			channel: &config.API{
				Path: "google/example/v1",
			},
			googleapisDir: googleapisDir,
			want: &sidekickconfig.Config{
				General: sidekickconfig.GeneralConfig{
					Language:            "dart",
					SpecificationFormat: "protobuf",
					ServiceConfig:       "",
					SpecificationSource: "google/example/v1",
				},
				Source: map[string]string{
					"googleapis-root": googleapisDir,
					"roots":           "googleapis",
					"title-override":  "library-title-override",
				},
				Codec: map[string]string{
					"api-keys-environment-variables": "GOOGLE_API_KEY",
					"dependencies":                   "dep-1,dep-2",
					"dev-dependencies":               "dev-dep-1,dev-dep-2",
					"extra-imports":                  "extra-imports",
					"issue-tracker-url":              "https://tracker/issues",
					"library-path-override":          "library-path-override",
					"not-for-publication":            "false",
					"package:googleapis_auth":        "^2.0.0",
					"part-file":                      "part-file",
					"prefix:google.logging.type":     "logging_type",
					"proto:google.api":               "package:google_cloud_api/api.dart",
					"readme-after-title-text":        "readme-after-title-text",
					"readme-quickstart-text":         "readme-quickstart-text",
					"repository-url":                 "https://github.com/googleapis/google-cloud-dart",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			sources := &common.Sources{
				Googleapis: test.googleapisDir,
			}
			got, err := toSidekickConfig(test.library, test.channel, sources)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("toSidekickConfig() error = %v, wantErr %v", err, test.wantErr)
				}
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
