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

package repometadata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestGenerateRepoMetadata(t *testing.T) {
	// Create a test service YAML
	yamlContent := `type: google.api.Service
config_version: 3
name: secretmanager.googleapis.com
title: Secret Manager API

documentation:
  summary: |-
    Stores sensitive data such as API keys, passwords, and certificates.
    Provides convenience while improving security.

publishing:
  documentation_uri: https://cloud.google.com/secret-manager/docs/overview
  api_short_name: secretmanager
`

	tmpDir := t.TempDir()
	serviceYAMLPath := filepath.Join(tmpDir, "service.yaml")
	if err := os.WriteFile(serviceYAMLPath, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(tmpDir, "output")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Name:         "google-cloud-secret-manager",
		ReleaseLevel: "stable",
	}

	if err := GenerateRepoMetadata(library, "python", "googleapis/google-cloud-python", serviceYAMLPath, outDir, []string{"google/cloud/secretmanager/v1", "google/cloud/secretmanager/v2"}); err != nil {
		t.Fatal(err)
	}

	// Read back the generated metadata
	data, err := os.ReadFile(filepath.Join(outDir, ".repo-metadata.json"))
	if err != nil {
		t.Fatal(err)
	}

	var got RepoMetadata
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	want := RepoMetadata{
		Name:                 "secretmanager",
		NamePretty:           "Secret Manager",
		ProductDocumentation: "https://cloud.google.com/secret-manager/",
		ClientDocumentation:  "https://cloud.google.com/python/docs/reference/secretmanager/latest",
		IssueTracker:         "",
		ReleaseLevel:         "stable",
		Language:             "python",
		LibraryType:          "GAPIC_AUTO",
		Repo:                 "googleapis/google-cloud-python",
		DistributionName:     "google-cloud-secret-manager",
		APIID:                "secretmanager.googleapis.com",
		DefaultVersion:       "v2",
		APIShortname:         "secretmanager",
		APIDescription:       "Stores sensitive data such as API keys, passwords, and certificates.\nProvides convenience while improving security.",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateRepoMetadata_WithOverrides(t *testing.T) {
	// Create a test service YAML
	yamlContent := `type: google.api.Service
config_version: 3
name: secretmanager.googleapis.com
title: Secret Manager API

documentation:
  summary: |-
    Stores sensitive data such as API keys, passwords, and certificates.
    Provides convenience while improving security.

publishing:
  documentation_uri: https://cloud.google.com/secret-manager/docs/overview
  api_short_name: secretmanager
`

	tmpDir := t.TempDir()
	serviceYAMLPath := filepath.Join(tmpDir, "service.yaml")
	if err := os.WriteFile(serviceYAMLPath, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(tmpDir, "output")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Name:                   "google-cloud-secret-manager",
		ReleaseLevel:           "stable",
		DescriptionOverride:    "Stores, manages, and secures access to application secrets.",
		DefaultVersionOverride: "v1",
	}

	if err := GenerateRepoMetadata(library, "python", "googleapis/google-cloud-python", serviceYAMLPath, outDir, []string{"google/cloud/secretmanager/v1", "google/cloud/secretmanager/v2"}); err != nil {
		t.Fatal(err)
	}

	// Read back the generated metadata
	data, err := os.ReadFile(filepath.Join(outDir, ".repo-metadata.json"))
	if err != nil {
		t.Fatal(err)
	}

	var got RepoMetadata
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	want := RepoMetadata{
		Name:                 "secretmanager",
		NamePretty:           "Secret Manager",
		ProductDocumentation: "https://cloud.google.com/secret-manager/",
		ClientDocumentation:  "https://cloud.google.com/python/docs/reference/secretmanager/latest",
		IssueTracker:         "",
		ReleaseLevel:         "stable",
		Language:             "python",
		LibraryType:          "GAPIC_AUTO",
		Repo:                 "googleapis/google-cloud-python",
		DistributionName:     "google-cloud-secret-manager",
		APIID:                "secretmanager.googleapis.com",
		DefaultVersion:       "v1",
		APIShortname:         "secretmanager",
		APIDescription:       "Stores, manages, and secures access to application secrets.",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestSelectDefaultVersion(t *testing.T) {
	for _, test := range []struct {
		name     string
		channels []string
		want     string
	}{
		{
			"prefer v1 over v1beta2",
			[]string{"google/cloud/secretmanager/v1beta2", "google/cloud/secretmanager/v1"},
			"v1",
		},
		{
			"prefer v1 over v1beta1",
			[]string{"google/cloud/secretmanager/v1", "google/cloud/secretmanager/v1beta1"},
			"v1",
		},
		{
			"prefer v2 over v1",
			[]string{"google/cloud/secretmanager/v1", "google/cloud/secretmanager/v2"},
			"v2",
		},
		{
			"select highest beta when no stable",
			[]string{"google/cloud/secretmanager/v1beta1", "google/cloud/secretmanager/v1beta2"},
			"v1beta2",
		},
		{
			"single version",
			[]string{"google/cloud/secretmanager/v1"},
			"v1",
		},
		{
			"multiple APIs with different versions",
			[]string{
				"google/cloud/secretmanager/v1",
				"google/cloud/secretmanager/v1beta2",
				"google/cloud/secrets/v1beta1",
			},
			"v1",
		},
		{
			"just v doesn't count as stable",
			[]string{
				"google/cloud/secretmanager/v",
				"google/cloud/secretmanager/v1beta2",
				"google/cloud/secrets/v1beta1",
			},
			"v1beta2",
		},
		{
			"empty",
			[]string{},
			"",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := selectDefaultVersion(test.channels)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestDeriveVersionComponent(t *testing.T) {
	for _, test := range []struct {
		name    string
		channel string
		want    string
	}{
		{"v1", "google/cloud/secretmanager/v1", "v1"},
		{"v1beta1", "google/cloud/aiplatform/v1beta1", "v1beta1"},
		{"v2", "google/analytics/admin/v2", "v2"},
		{"no version", "google/cloud/location", ""},
		{"empty", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := deriveVersionComponent(test.channel)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestCleanTitle(t *testing.T) {
	for _, test := range []struct {
		name  string
		title string
		want  string
	}{
		{"with API suffix", "Secret Manager API", "Secret Manager"},
		{"without suffix", "Secret Manager", "Secret Manager"},
		{"with trailing space", "Cloud Functions  API  ", "Cloud Functions"},
		{"empty", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := cleanTitle(test.title)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestExtractNameFromAPIID(t *testing.T) {
	for _, test := range []struct {
		name  string
		apiID string
		want  string
	}{
		{"standard", "secretmanager.googleapis.com", "secretmanager"},
		{"no domain", "secretmanager", "secretmanager"},
		{"empty", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := extractNameFromAPIID(test.apiID)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestExtractBaseProductURL(t *testing.T) {
	for _, test := range []struct {
		name   string
		docURI string
		want   string
	}{
		{
			"strip /docs/overview",
			"https://cloud.google.com/secret-manager/docs/overview",
			"https://cloud.google.com/secret-manager/",
		},
		{
			"strip /docs/reference",
			"https://cloud.google.com/storage/docs/reference",
			"https://cloud.google.com/storage/",
		},
		{
			"no /docs/ in URL",
			"https://cloud.google.com/secret-manager",
			"https://cloud.google.com/secret-manager",
		},
		{
			"empty",
			"",
			"",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := extractBaseProductURL(test.docURI)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}
