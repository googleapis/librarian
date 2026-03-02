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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestFromLibrary(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		want    *RepoMetadata
	}{
		{
			name: "no overrides",
			library: &config.Library{
				Name:         "google-cloud-secret-manager",
				APIs:         []*config.API{{Path: "google/cloud/secretmanager/v1"}},
				ReleaseLevel: "stable",
			},
			want: &RepoMetadata{
				Name:                 "secretmanager",
				NamePretty:           "Secret Manager",
				ProductDocumentation: "https://cloud.google.com/secret-manager/",
				IssueTracker:         "https://issuetracker.google.com/issues/new?component=784854&template=1380926",
				ReleaseLevel:         "stable",
				Language:             config.LanguagePython,
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
			want: &RepoMetadata{
				Name:                 "secretmanager",
				NamePretty:           "Secret Manager",
				ProductDocumentation: "https://cloud.google.com/secret-manager/",
				IssueTracker:         "https://issuetracker.google.com/issues/new?component=784854&template=1380926",
				ReleaseLevel:         "stable",
				Language:             config.LanguagePython,
				Repo:                 "googleapis/google-cloud-python",
				DistributionName:     "google-cloud-secret-manager",
				APIID:                "secretmanager.googleapis.com",
				APIShortname:         "secretmanager",
				APIDescription:       "Stores, manages, and secures access to application secrets.",
			},
		},
		{
			name: "no service config",
			library: &config.Library{
				Name:         "google-longrunning",
				ReleaseLevel: "stable",
				APIs:         []*config.API{{Path: "google/longrunning"}},
			},
			want: &RepoMetadata{
				ReleaseLevel:     "stable",
				Language:         config.LanguagePython,
				Repo:             "googleapis/google-cloud-python",
				DistributionName: "google-longrunning",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			outDir := filepath.Join(tmpDir, "output")
			if err := os.MkdirAll(outDir, 0755); err != nil {
				t.Fatal(err)
			}

			cfg := &config.Config{
				Language: config.LanguagePython,
				Repo:     "googleapis/google-cloud-python",
			}

			got, err := FromLibrary(cfg, test.library, "../testdata/googleapis")
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFromLibrary_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		wantErr error
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
				APIs:         []*config.API{{Path: "android/notallowed/v1"}},
			},
			// Error returned by serviceconfig.Find isn't easily distinguished
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{Language: config.LanguagePython}
			_, gotErr := FromLibrary(cfg, test.library, "../testdata/googleapis")
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
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
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
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestWriteAndRead tests the happy path for both the Write and Read functions.
// While it's odd to test two functions in one test, separating the tests would
// basically involve copy-pasting the production code for the "other" function
// into each separate test.
func TestWriteAndRead(t *testing.T) {
	want := &RepoMetadata{
		Name:       "test-library",
		NamePretty: "Test Library",
		Language:   config.LanguageGo,
	}
	tmpDir := t.TempDir()
	if err := want.Write(tmpDir); err != nil {
		t.Fatal(err)
	}
	got, err := Read(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestWrite_Error(t *testing.T) {
	metadata := &RepoMetadata{
		Name: "library",
	}
	dir := t.TempDir()
	gotErr := metadata.Write(filepath.Join(dir, "non-existent"))
	wantErr := os.ErrNotExist
	if !errors.Is(gotErr, wantErr) {
		t.Errorf("Write() error = %v, wantErr %v", gotErr, wantErr)
	}
}

func TestRead_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		setup   func(*testing.T, string)
		wantErr error
	}{
		{
			name: "not JSON",
			setup: func(t *testing.T, dir string) {
				if err := os.WriteFile(filepath.Join(dir, repoMetadataFile), []byte("not json"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			// We can't specify the exact error here.
		},
		{
			name:    "no file",
			setup:   func(t *testing.T, dir string) {},
			wantErr: os.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			test.setup(t, dir)
			_, gotErr := Read(dir)
			if gotErr == nil {
				t.Fatal("expected error; got nil")
			}
			if test.wantErr != nil && !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Read() error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}
