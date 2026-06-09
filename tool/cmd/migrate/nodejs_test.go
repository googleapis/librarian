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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
)

func TestBuildNodejsLibraries(t *testing.T) {
	got, err := buildNodejsLibraries("testdata/google-cloud-node", "testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	want := []*config.Library{
		{
			Name: "google-cloud-secretmanager", Version: "6.1.0",
			CopyrightYear: "2020",
			APIs: []*config.API{
				{Path: "google/cloud/secretmanager/v1"},
			},
			Nodejs: &config.NodejsPackage{
				PackageName: "@google-cloud/secret-manager",
			},
		},
		{
			Name:    "google-cloud-speech",
			Version: "7.2.0",
			APIs: []*config.API{
				{Path: "google/cloud/speech/v1"},
			},
			Nodejs: &config.NodejsPackage{
				Dependencies: map[string]string{
					"@google-cloud/common": "^6.0.0",
					"pumpify":              "^2.0.1",
				},
			},
		},
		{
			Name:    "google-cloud-translate",
			Version: "9.1.0",
			APIs: []*config.API{
				{Path: "google/cloud/translate/v3"},
			},
			Nodejs: &config.NodejsPackage{
				BundleConfig: "google/cloud/translate/v3/translate_gapic.yaml",
				ExtraProtocParameters: []string{
					"auto-populate-field-oauth-scope",
				},
				HandwrittenLayer: true,
				MainService:      "translate",
				Mixins:           "none",
			},
		},
		{
			Name:    "google-cloud-workstations",
			Version: "1.3.0",
			APIs: []*config.API{
				{Path: "google/cloud/workstations/v1"},
			},
			Nodejs: &config.NodejsPackage{},
		},
	}
	if diff := cmp.Diff(want, got, cmpopts.SortSlices(func(a, b *config.Library) bool { return a.Name < b.Name })); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestDeriveNpmPackageName(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "standard",
			input: "google-cloud-batch",
			want:  "@google-cloud/batch",
		},
		{
			name:  "multi-word suffix",
			input: "google-cloud-access-approval",
			want:  "@google-cloud/access-approval",
		},
		{
			name:  "no second dash",
			input: "google",
			want:  "google",
		},
		{
			name:  "one dash only",
			input: "google-cloud",
			want:  "google-cloud",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := deriveNpmPackageName(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseBazelNodejsInfo(t *testing.T) {
	for _, test := range []struct {
		name string
		api  string
		want *nodejsGapicInfo
	}{
		{
			name: "secretmanager",
			api:  "google/cloud/secretmanager/v1",
			want: &nodejsGapicInfo{
				packageName: "@google-cloud/secret-manager",
			},
		},
		{
			name: "translate with all fields",
			api:  "google/cloud/translate/v3",
			want: &nodejsGapicInfo{
				packageName:  "@google-cloud/translate",
				bundleConfig: "google/cloud/translate/v3/translate_gapic.yaml",
				extraProtocParameters: []string{
					"auto-populate-field-oauth-scope",
				},
				handwrittenLayer: true,
				mainService:      "translate",
				mixins:           "none",
			},
		},
		{
			name: "no nodejs rule",
			api:  "google/cloud/no-gapic",
			want: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseBazelNodejsInfo("testdata/googleapis", test.api)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(nodejsGapicInfo{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseBazelNodejsInfo_Diregapic(t *testing.T) {
	tmpDir := t.TempDir()
	apiPath := "google/cloud/compute/v1"
	fullPath := filepath.Join(tmpDir, apiPath)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		t.Fatal(err)
	}
	bazelContent := `
nodejs_gapic_library(
    name = "compute_nodejs_gapic",
    package_name = "@google-cloud/compute",
    diregapic = True,
)
`
	if err := os.WriteFile(filepath.Join(fullPath, "BUILD.bazel"), []byte(bazelContent), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := parseBazelNodejsInfo(tmpDir, apiPath)
	if err != nil {
		t.Fatal(err)
	}
	want := &nodejsGapicInfo{
		packageName: "@google-cloud/compute",
		diregapic:   true,
	}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(nodejsGapicInfo{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestParseOwlBotAPIPaths(t *testing.T) {
	tmpDir := t.TempDir()
	for _, p := range []string{
		"google/cloud/fake/v1",
		"google/cloud/fake/v2",
		"google/cloud/fake/dashboard/v1",
	} {
		fullPath := filepath.Join(tmpDir, p)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			t.Fatal(err)
		}
		bazelContent := `
nodejs_gapic_library(
    name = "fake_nodejs_gapic",
    package_name = "@google-cloud/fake",
)
`
		if err := os.WriteFile(filepath.Join(fullPath, "BUILD.bazel"), []byte(bazelContent), 0644); err != nil {
			t.Fatal(err)
		}
	}

	for _, test := range []struct {
		name   string
		owlBot *owlBotYAML
		want   []*config.API
	}{
		{
			name: "standard base path",
			owlBot: &owlBotYAML{
				DeepCopyRegex: []owlBotCopyRule{
					{Source: "/google/cloud/fake/(.*)/.*-nodejs"},
				},
			},
			want: []*config.API{
				{Path: "google/cloud/fake/v1"},
				{Path: "google/cloud/fake/v2"},
			},
		},
		{
			name: "with api-name metadata",
			owlBot: &owlBotYAML{
				DeepCopyRegex: []owlBotCopyRule{
					{Source: "/google/cloud/fake/(.*)/.*-nodejs"},
				},
				APIName: "dashboard",
			},
			want: []*config.API{
				{Path: "google/cloud/fake/dashboard/v1"},
			},
		},
		{
			name:   "empty copy regex",
			owlBot: &owlBotYAML{},
			want:   nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseOwlBotAPIPaths(test.owlBot, tmpDir)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("parseOwlBotAPIPaths mismatch for api-name %q (-want +got):\n%s", test.owlBot.APIName, diff)
			}
		})
	}
}

func TestBuildNodejsLibrary_ESMOverride(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		libraryName string
		pkgJSON     string
		wantESM     bool
	}{
		{
			name:        "tasks library receives ESM true",
			libraryName: "google-cloud-tasks",
			pkgJSON:     `{"name": "@google-cloud/tasks", "version": "6.2.2"}`,
			wantESM:     true,
		},
		{
			name:        "other standard library receives ESM false",
			libraryName: "google-cloud-other",
			pkgJSON:     `{"name": "@google-cloud/other", "version": "1.0.0"}`,
			wantESM:     false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			pkgDir := filepath.Join(tmpDir, "packages", test.libraryName)
			if err := os.MkdirAll(pkgDir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(test.pkgJSON), 0644); err != nil {
				t.Fatal(err)
			}
			got, err := buildNodejsLibrary(t.TempDir(), filepath.Join(tmpDir, "packages"), test.libraryName)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantESM, got.Nodejs.ESM); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestParseBazelNodejsInfo_OmitCommonResources validates the Bazel target dependency graph parser
// and verifies that the omit_common_resources flag is correctly auto-detected across different AST configurations.
func TestParseBazelNodejsInfo_OmitCommonResources(t *testing.T) {
	for _, test := range []struct {
		name         string
		bazelContent string
		wantOmit     bool
	}{
		{
			name: "case 1: relative path dependency present",
			bazelContent: `
nodejs_gapic_library(
    name = "secretmanager_nodejs_gapic",
    src = ":secretmanager_proto_with_info",
    package_name = "@google-cloud/secret-manager",
)
proto_library_with_info(
    name = "secretmanager_proto_with_info",
    deps = [
        "//google/cloud:common_resources_proto",
    ],
)
`,
			wantOmit: false,
		},
		{
			name: "case 2: dependency absent",
			bazelContent: `
nodejs_gapic_library(
    name = "billing_nodejs_gapic",
    src = ":billing_proto_with_info",
    package_name = "@google-cloud/billing",
)
proto_library_with_info(
    name = "billing_proto_with_info",
    deps = [
        ":billing_proto",
    ],
)
`,
			wantOmit: true,
		},
		{
			name: "case 3: external repository absolute namespace prefix",
			bazelContent: `
nodejs_gapic_library(
    name = "compute_nodejs_gapic",
    src = ":compute_proto_with_info",
    package_name = "@google-cloud/compute",
)
proto_library_with_info(
    name = "compute_proto_with_info",
    deps = [
        "@com_google_googleapis//google/cloud:common_resources_proto",
    ],
)
`,
			wantOmit: false,
		},
		{
			name: "case 4: list concatenation addition present",
			bazelContent: `
nodejs_gapic_library(
    name = "aiplatform_nodejs_gapic",
    src = ":aiplatform_proto_with_info",
    package_name = "@google-cloud/aiplatform",
)
proto_library_with_info(
    name = "aiplatform_proto_with_info",
    deps = [
        ":aiplatform_proto",
        "//google/cloud:common_resources_proto",
    ] + _PROTO_SUBPACKAGE_DEPS,
)
`,
			wantOmit: false,
		},
		{
			name: "case 5: list concatenation addition absent",
			bazelContent: `
nodejs_gapic_library(
    name = "quotas_nodejs_gapic",
    src = ":quotas_proto_with_info",
    package_name = "@google-cloud/quotas",
)
proto_library_with_info(
    name = "quotas_proto_with_info",
    deps = [
        ":quotas_proto",
    ] + _PROTO_SUBPACKAGE_DEPS,
)
`,
			wantOmit: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			apiPath := "google/cloud/test/v1"
			fullPath := filepath.Join(tmpDir, apiPath)
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(fullPath, "BUILD.bazel"), []byte(test.bazelContent), 0644); err != nil {
				t.Fatal(err)
			}

			got, err := parseBazelNodejsInfo(tmpDir, apiPath)
			if err != nil {
				t.Fatal(err)
			}
			if got == nil {
				t.Fatal("expected non-nil nodejsGapicInfo")
			}
			if got.omitCommonResources != test.wantOmit {
				t.Errorf("parseBazelNodejsInfo() omitCommonResources = %v, want %v", got.omitCommonResources, test.wantOmit)
			}
		})
	}
}

func TestBuildNodejsLibrary_MetadataOverrides(t *testing.T) {
	for _, test := range []struct {
		name        string
		packageName string
		repoMeta    string
		want        *config.NodejsPackage
	}{
		{
			name:        "standard google-cloud package with custom docs",
			packageName: "@google-cloud/secret-manager",
			repoMeta:    `{"client_documentation": "https://custom.docs.com/client-ref"}`,
			want: &config.NodejsPackage{
				PackageName:                 "@google-cloud/secret-manager",
				ClientDocumentationOverride: "https://custom.docs.com/client-ref",
			},
		},
		{
			name:        "non-google-cloud package preserves docs as override",
			packageName: "secretmanager-utility",
			// Even though this URL looks like a standard one, since the package name is not @google-cloud/,
			// the generator will NOT produce this by default. It must be preserved as an override.
			repoMeta: `{"client_documentation": "https://cloud.google.com/nodejs/docs/reference/secretmanager-utility/latest"}`,
			want: &config.NodejsPackage{
				PackageName:                 "secretmanager-utility",
				ClientDocumentationOverride: "https://cloud.google.com/nodejs/docs/reference/secretmanager-utility/latest",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			pkgDir := filepath.Join(tmpDir, "packages", "google-cloud-secretmanager")
			if err := os.MkdirAll(pkgDir, 0755); err != nil {
				t.Fatal(err)
			}

			pkgJSON := `{"name": "` + test.packageName + `", "version": "6.1.0"}`
			if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(pkgJSON), 0644); err != nil {
				t.Fatal(err)
			}

			owlBot := `
deep-copy-regex:
  - source: /google/cloud/secretmanager/(.*)/.*-nodejs
`
			if err := os.WriteFile(filepath.Join(pkgDir, ".OwlBot.yaml"), []byte(owlBot), 0644); err != nil {
				t.Fatal(err)
			}

			if err := os.WriteFile(filepath.Join(pkgDir, ".repo-metadata.json"), []byte(test.repoMeta), 0644); err != nil {
				t.Fatal(err)
			}

			// Running buildNodejsLibrary using testdata/googleapis (which contains secretmanager_v1.yaml)
			got, err := buildNodejsLibrary("testdata/googleapis", filepath.Join(tmpDir, "packages"), "google-cloud-secretmanager")
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, got.Nodejs); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
