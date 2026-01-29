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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestGenerate(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "dart")

	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()

	library := &config.Library{
		Name:          "google-cloud-secretmanager-v1",
		Version:       "0.1.0",
		Output:        outDir,
		CopyrightYear: "2025",
		APIs: []*config.API{
			{
				Path: "google/cloud/secretmanager/v1",
			},
		},
		Dart: &config.DartPackage{
			APIKeysEnvironmentVariables: "GOOGLE_API_KEY",
			IssueTrackerURL:             "https://github.com/googleapis/google-cloud-dart/issues",
			Packages: map[string]string{
				"googleapis_auth":           "^2.0.0",
				"http":                      "^1.3.0",
				"google_cloud_api":          "^0.4.0",
				"google_cloud_iam_v1":       "^0.4.0",
				"google_cloud_protobuf":     "^0.4.0",
				"google_cloud_location":     "^0.4.0",
				"google_cloud_longrunning":  "^0.4.0",
				"google_cloud_logging_type": "^0.4.0",
				"google_cloud_rpc":          "^0.4.0",
				"google_cloud_type":         "^0.4.0",
			},
		},
	}
	if err := Generate(t.Context(), &config.Config{}, library, googleapisDir); err != nil {
		t.Fatal(err)
	}
	if err := Format(t.Context(), library); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		path string
		want string
	}{
		{filepath.Join(outDir, "pubspec.yaml"), "name:"},
		{filepath.Join(outDir, "pubspec.yaml"), "google_cloud_secretmanager_v1"},
		{filepath.Join(outDir, "README.md"), "Secret Manager"},
		{filepath.Join(outDir, "lib", "secretmanager.dart"), "library"},
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

func TestFormat(t *testing.T) {
	testhelper.RequireCommand(t, "dart")
	outDir := t.TempDir()
	dartFile := filepath.Join(outDir, "test.dart")
	if err := os.WriteFile(dartFile, []byte("void main() { print('hello'); }"), 0644); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Output: outDir,
	}
	if err := Format(t.Context(), library); err != nil {
		t.Fatal(err)
	}
}

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
						"googleapis_auth": "^2.0.0",
						"http":            "^1.3.0",
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
						"google.protobuf": "pb",
						"google.api":      "api",
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
						"google.api":      "package:google_cloud_api/api.dart",
						"google.protobuf": "package:protobuf/protobuf.dart",
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
						"googleapis_auth": "^2.0.0",
					},
					Prefixes: map[string]string{
						"google.protobuf": "pb",
					},
					Protos: map[string]string{
						"google.api": "package:google_cloud_api/api.dart",
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
