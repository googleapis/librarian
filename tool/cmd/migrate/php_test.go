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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestRunPHPMigration(t *testing.T) {
	oldFetchSource := phpFetchSource
	t.Cleanup(func() {
		phpFetchSource = oldFetchSource
	})
	absGoogleapis, err := filepath.Abs("../../internal/testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	// Override phpFetchSource.
	phpFetchSource = func(ctx context.Context) (*config.Source, error) {
		return &config.Source{
			Commit: "abcd123",
			SHA256: "sha123",
			Dir:    absGoogleapis,
		}, nil
	}
	dir := t.TempDir()
	// Create a fake library SecretManager.
	libDir := filepath.Join(dir, "SecretManager")
	if err := os.Mkdir(libDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "VERSION"), []byte("2.3.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "composer.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create a fake non-library directory to ensure it is ignored.
	ignoredDir := filepath.Join(dir, "dev")
	if err := os.Mkdir(ignoredDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ignoredDir, "composer.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	err = runPHPMigration(t.Context(), ".")
	if err != nil {
		t.Fatal(err)
	}
	// Verify librarian.yaml is written and contains the expected content.
	got, err := yaml.Read[config.Config](config.LibrarianYAML)
	if err != nil {
		t.Fatal(err)
	}
	commonResources := true
	want := &config.Config{
		Language: config.LanguagePhp,
		Sources: &config.Sources{
			Googleapis: &config.Source{
				Commit: "abcd123",
				SHA256: "sha123",
			},
		},
		Default: &config.Default{
			PHP: &config.PHPDefault{
				CommonResources: &commonResources,
			},
		},
		Libraries: []*config.Library{
			{
				Name:    "SecretManager",
				Version: "2.3.0",
			},
		},
		Tools: &config.Tools{
			Composer: []*config.ComposerTool{
				{
					Name:    "google/gapic-generator-php",
					Version: "v1.21.2",
					Repo:    "github.com/googleapis/gapic-generator-php",
					SHA256:  "29635b02c6e505fe31cba2f88ae999f00d2710fe1d65cb7cad521a82e7c5a518",
				},
			},
			Pip: []*config.PipTool{
				{
					Name:    "synthtool",
					Version: "e643ce8e20f8fe237a31a1754524ba987de72875",
					Package: "gcp-synthtool@git+https://github.com/googleapis/synthtool@e643ce8e20f8fe237a31a1754524ba987de72875",
				},
			},
			PNPM: []*config.PNPMTool{
				{
					Name:    "@prettier/plugin-php",
					Version: "0.19.2",
				},
				{
					Name:    "prettier",
					Version: "2.8.8",
				},
			},
			Protoc: &config.Protoc{
				Version: "31.0",
				SHA256:  "24e2ed32060b7c990d5eb00d642fde04869d7f77c6d443f609353f097799dd42",
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestExtractAPIPaths(t *testing.T) {
	for _, test := range []struct {
		name      string
		source    string
		wantPaths []string
	}{
		{
			name:      "versioned api",
			source:    "/google/cloud/ces/(v1)/.*-php/(.*)",
			wantPaths: []string{"google/cloud/ces/v1"},
		},
		{
			name:      "unversioned api",
			source:    "/google/identity/accesscontextmanager/type/.*-php/(.*)",
			wantPaths: []string{"google/identity/accesscontextmanager/type"},
		},
		{
			name:      "non-matching path",
			source:    "/some/other/path",
			wantPaths: nil,
		},
		{
			name:      "grafeas versioned",
			source:    "/grafeas/(v1)/.*-php/(.*)",
			wantPaths: []string{"grafeas/v1"},
		},
		{
			name:      "union versioned api",
			source:    "/google/cloud/secretmanager/(v1|v1beta2)/.*-php/(.*)",
			wantPaths: []string{"google/cloud/secretmanager/v1", "google/cloud/secretmanager/v1beta2"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotPaths := extractAPIPaths(test.source)
			if diff := cmp.Diff(test.wantPaths, gotPaths); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractAPIsFromOwlBot(t *testing.T) {
	for _, test := range []struct {
		name      string
		setupFile func(dir string) string
		want      []*config.API
	}{
		{
			name: "missing owlbot.yaml",
			setupFile: func(dir string) string {
				return filepath.Join(dir, "missing.yaml")
			},
			want: nil,
		},
		{
			name: "valid file",
			setupFile: func(dir string) string {
				content := `
deep-copy-regex:
  - source: /google/cloud/ces/(v1)/.*-php/(.*)
    dest: /owl-bot-staging/Ces/$1/$2
  - source: /google/identity/accesscontextmanager/type/.*-php/(.*)
    dest: /owl-bot-staging/AccessContextManager/type-protos/$1
  - source: /google/apps/card/(v1)/.*-php/(.*)
    dest: /owl-bot-staging/AppsChat/card-protos/$1/$2
api-name: Ces
`
				path := filepath.Join(dir, ".OwlBot.yaml")
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			want: []*config.API{
				{
					Path: "google/cloud/ces/v1",
					PHP: &config.PHPAPI{
						StagingSubdir: "v1",
					},
				},
				{
					Path: "google/identity/accesscontextmanager/type",
					PHP: &config.PHPAPI{
						StagingSubdir: "type-protos",
					},
				},
				{
					Path: "google/apps/card/v1",
					PHP: &config.PHPAPI{
						StagingSubdir: "card-protos/v1",
					},
				},
			},
		},
		{
			name: "destination with library root (dot)",
			setupFile: func(dir string) string {
				content := `
deep-copy-regex:
  - source: /google/geo/type/.*-php/(.*)
    dest: /owl-bot-staging/GeoCommonProtos/$1
api-name: GeoCommonProtos
`
				path := filepath.Join(dir, ".OwlBot.yaml")
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			want: []*config.API{
				{
					Path: "google/geo/type",
					PHP: &config.PHPAPI{
						StagingSubdir: ".",
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			path := test.setupFile(dir)
			got, err := extractAPIsFromOwlBot(path)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractAPIsFromOwlBot_Error(t *testing.T) {
	for _, test := range []struct {
		name      string
		setupFile func(dir string) string
		wantErr   error
	}{
		{
			name: "invalid file",
			setupFile: func(dir string) string {
				content := `{invalid`
				path := filepath.Join(dir, ".OwlBot.yaml")
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			// wantErr is nil as we only assert that a YAML parsing error is returned.
			wantErr: nil,
		},
		{
			name: "missing staging marker",
			setupFile: func(dir string) string {
				content := `
deep-copy-regex:
  - source: /google/cloud/ces/(v1)/.*-php/(.*)
    dest: /no-staging/Ces/$1/$2
api-name: Ces
`
				path := filepath.Join(dir, ".OwlBot.yaml")
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			wantErr: errUnableToResolveStagingSubdir,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			path := test.setupFile(dir)
			_, err := extractAPIsFromOwlBot(path)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Errorf("expected error %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestParsePHPBazel(t *testing.T) {
	for _, test := range []struct {
		name                string
		bazelRules          string
		want                []string
		wantCommonResources bool
	}{
		{
			name:                "no BUILD.bazel",
			bazelRules:          "",
			want:                nil,
			wantCommonResources: false,
		},
		{
			name: "valid BUILD.bazel with location and iam mixins",
			bazelRules: `
proto_library_with_info(
    name = "ces_proto_with_info",
    deps = [
        ":ces_proto",
        "//google/cloud:common_resources_proto",
        "//google/cloud/location:location_proto",
        "//google/iam/v1:iam_policy_proto",
        "//google/cloud/unrelated:unrelated_proto",
    ],
)
`,
			want: []string{
				"google/cloud/location/locations.proto",
				"google/iam/v1/iam_policy.proto",
			},
			wantCommonResources: true,
		},
		{
			name: "valid BUILD.bazel with no mixins",
			bazelRules: `
proto_library_with_info(
    name = "ces_proto_with_info",
    deps = [
        ":ces_proto",
        "//google/cloud:common_resources_proto",
    ],
)
`,
			want:                nil,
			wantCommonResources: true,
		},
		{
			name: "valid BUILD.bazel with no common resources",
			bazelRules: `
proto_library_with_info(
    name = "ces_proto_with_info",
    deps = [
        ":ces_proto",
    ],
)
`,
			want:                nil,
			wantCommonResources: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			if test.bazelRules != "" {
				apiDir := filepath.Join(tempDir, "google/cloud/ces/v1")
				if err := os.MkdirAll(apiDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(apiDir, "BUILD.bazel"), []byte(test.bazelRules), 0644); err != nil {
					t.Fatal(err)
				}
			}
			got, gotCommonResources, err := parsePHPBazel(tempDir, "google/cloud/ces/v1")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if gotCommonResources != test.wantCommonResources {
				t.Errorf("mismatch common resources flag: want %t, got %t", test.wantCommonResources, gotCommonResources)
			}
		})
	}
}

func TestFindPHPLibraries(t *testing.T) {
	googleapisDir := "testdata/googleapis"

	for _, test := range []struct {
		name                         string
		setupLib                     func(t *testing.T, dir string)
		globalDefaultCommonResources bool
		want                         []*config.Library
	}{
		{
			name: "common resources configuration",
			setupLib: func(t *testing.T, dir string) {
				libDir := filepath.Join(dir, "SecretManager")
				if err := os.Mkdir(libDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(libDir, "VERSION"), []byte("2.3.0\n"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(libDir, "composer.json"), []byte("{}"), 0644); err != nil {
					t.Fatal(err)
				}
				owlbotContent := `
deep-copy-regex:
  - source: /google/cloud/secretmanager/(v1)/.*-php/(.*)
    dest: /owl-bot-staging/SecretManager/$1/$2
  - source: /google/cloud/multipygapic/.*-php/(.*)
    dest: /owl-bot-staging/SecretManager/multipygapic/$1
`
				if err := os.WriteFile(filepath.Join(libDir, ".OwlBot.yaml"), []byte(owlbotContent), 0644); err != nil {
					t.Fatal(err)
				}
			},
			globalDefaultCommonResources: true,
			want: []*config.Library{
				{
					Name:    "SecretManager",
					Version: "2.3.0",
					APIs: []*config.API{
						{
							Path: "google/cloud/secretmanager/v1",
							PHP: &config.PHPAPI{
								StagingSubdir: "v1",
							},
						},
						{
							Path: "google/cloud/multipygapic",
							PHP: &config.PHPAPI{
								StagingSubdir:   "multipygapic",
								CommonResources: new(false),
							},
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			test.setupLib(t, dir)
			got, err := findPHPLibraries(dir, googleapisDir, test.globalDefaultCommonResources)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
