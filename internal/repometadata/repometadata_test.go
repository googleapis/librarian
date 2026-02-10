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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestGenerate(t *testing.T) {
	for _, test := range []struct {
		name           string
		defaultVersion string
		library        *config.Library
		want           RepoMetadata
	}{
		{
			name: "no overrides",
			library: &config.Library{
				Name:         "google-cloud-secret-manager",
				APIs:         []*config.API{{Path: "google/cloud/secretmanager/v1"}},
				ReleaseLevel: "stable",
			},
			want: RepoMetadata{
				Name:                 "secretmanager",
				NamePretty:           "Secret Manager",
				ProductDocumentation: "https://cloud.google.com/secret-manager/",
				ClientDocumentation:  "https://cloud.google.com/python/docs/reference/secretmanager/latest",
				IssueTracker:         "https://issuetracker.google.com/issues/new?component=784854&template=1380926",
				ReleaseLevel:         "stable",
				Language:             "python",
				LibraryType:          "GAPIC_AUTO",
				Repo:                 "googleapis/google-cloud-python",
				DistributionName:     "google-cloud-secret-manager",
				APIID:                "secretmanager.googleapis.com",
				APIShortname:         "secretmanager",
				APIDescription:       "Stores sensitive data such as API keys, passwords, and certificates.\nProvides convenience while improving security.",
			},
		},
		{
			name: "description override",
			library: &config.Library{
				Name:                "google-cloud-secret-manager",
				ReleaseLevel:        "stable",
				APIs:                []*config.API{{Path: "google/cloud/secretmanager/v1"}},
				DescriptionOverride: "Stores, manages, and secures access to application secrets.",
			},
			defaultVersion: "v1",
			want: RepoMetadata{
				Name:                 "secretmanager",
				NamePretty:           "Secret Manager",
				ProductDocumentation: "https://cloud.google.com/secret-manager/",
				ClientDocumentation:  "https://cloud.google.com/python/docs/reference/secretmanager/latest",
				DefaultVersion:       "v1",
				IssueTracker:         "https://issuetracker.google.com/issues/new?component=784854&template=1380926",
				ReleaseLevel:         "stable",
				Language:             "python",
				LibraryType:          "GAPIC_AUTO",
				Repo:                 "googleapis/google-cloud-python",
				DistributionName:     "google-cloud-secret-manager",
				APIID:                "secretmanager.googleapis.com",
				APIShortname:         "secretmanager",
				APIDescription:       "Stores, manages, and secures access to application secrets.",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			outDir := filepath.Join(tmpDir, "output")
			if err := os.MkdirAll(outDir, 0755); err != nil {
				t.Fatal(err)
			}

			if err := FromLibrary(test.library, "python", "googleapis/google-cloud-python", "../testdata/googleapis", test.defaultVersion, outDir); err != nil {
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

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerate_Error(t *testing.T) {
	for _, test := range []struct {
		name           string
		defaultVersion string
		library        *config.Library
		wantErr        error
	}{
		{
			name: "no APIs",
			library: &config.Library{
				Name:         "google-cloud-secret-manager",
				ReleaseLevel: "stable",
			},
			wantErr: errNoAPIs,
		},
		{
			name: "non-allowlisted API",
			library: &config.Library{
				Name:         "google-cloud-secret-manager",
				ReleaseLevel: "stable",
				APIs:         []*config.API{{Path: "google/cloud/notallowed/v1"}},
			},
			// Error returned by serviceconfig.Find isn't easily distinguished
		},
		{
			name: "no service config",
			library: &config.Library{
				Name:         "google-longrunning",
				ReleaseLevel: "stable",
				APIs:         []*config.API{{Path: "google/longrunning"}},
			},
			wantErr: errNoServiceConfig,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			gotErr := FromLibrary(test.library, "python", "googleapis/google-cloud-python", "../testdata/googleapis", test.defaultVersion, tmpDir)
			if gotErr == nil {
				t.Fatal("expected error, got nil")
			}
			if test.wantErr != nil && !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Generate() error = %v, wantErr %v", gotErr, test.wantErr)
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

func TestBuildClientDocURL(t *testing.T) {
	for _, test := range []struct {
		name        string
		language    string
		serviceName string
		want        string
	}{
		{
			name:        "python",
			language:    "python",
			serviceName: "secretmanager",
			want:        "https://cloud.google.com/python/docs/reference/secretmanager/latest",
		},
		{
			name:        "rust",
			language:    "rust",
			serviceName: "secretmanager",
			want:        "https://docs.rs/google-cloud-secretmanager/latest",
		},
		{
			name:        "unknown language",
			language:    "vb",
			serviceName: "secretmanager",
			want:        "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildClientDocURL(test.language, test.serviceName)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}
