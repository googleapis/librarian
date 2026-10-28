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

package swift

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/sources"
)

func TestCreateRepoMetadata(t *testing.T) {
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name    string
		library *config.Library
		sources *sources.Sources
		want    *repometadata.RepoMetadata
	}{
		{
			name: "creates swift repo metadata with prefix and version",
			library: &config.Library{
				Name:    "google-cloud-secretmanager-v1",
				Version: "0.1.0-preview",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			sources: &sources.Sources{
				Googleapis: googleapisDir,
			},
			want: &repometadata.RepoMetadata{
				Name:                 "secretmanager",
				NamePretty:           "Secret Manager",
				ProductDocumentation: "https://cloud.google.com/secret-manager/",
				ClientDocumentation:  "https://swiftpackageindex.com/googleapis/swift-google-cloud-secretmanager-v1/0.1.0-preview/documentation",
				IssueTracker:         "https://issuetracker.google.com/issues/new?component=784854&template=1380926",
				Repo:                 "googleapis/swift-google-cloud-secretmanager-v1",
				DistributionName:     "swift-google-cloud-secretmanager-v1",
				Language:             "swift",
				APIID:                "secretmanager.googleapis.com",
				APIShortname:         "secretmanager",
				APIDescription:       "Stores sensitive data such as API keys, passwords, and certificates.\nProvides convenience while improving security.",
				ReleaseLevel:         "preview",
				LibraryType:          "GAPIC_AUTO",
			},
		},
		{
			name: "creates swift repo metadata already prefixed without version",
			library: &config.Library{
				Name: "swift-google-cloud-secretmanager-v1",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			sources: &sources.Sources{
				Googleapis: googleapisDir,
			},
			want: &repometadata.RepoMetadata{
				Name:                 "secretmanager",
				NamePretty:           "Secret Manager",
				ProductDocumentation: "https://cloud.google.com/secret-manager/",
				ClientDocumentation:  "https://swiftpackageindex.com/googleapis/swift-google-cloud-secretmanager-v1/documentation",
				IssueTracker:         "https://issuetracker.google.com/issues/new?component=784854&template=1380926",
				Repo:                 "googleapis/swift-google-cloud-secretmanager-v1",
				DistributionName:     "swift-google-cloud-secretmanager-v1",
				Language:             "swift",
				APIID:                "secretmanager.googleapis.com",
				APIShortname:         "secretmanager",
				APIDescription:       "Stores sensitive data such as API keys, passwords, and certificates.\nProvides convenience while improving security.",
				ReleaseLevel:         "stable",
				LibraryType:          "GAPIC_AUTO",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				Language: config.LanguageSwift,
				Repo:     "googleapis/google-cloud-swift",
			}
			got, err := createRepoMetadata(cfg, test.library, test.sources)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPackageName(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		want    string
	}{
		{
			name: "package name override takes precedence",
			library: &config.Library{
				Name:   "google-cloud-auth",
				Output: "pkgs/swift-google-auth/Sources/GoogleCloudAuth/generated",
				Swift: &config.SwiftPackage{
					PackageNameOverride: "custom-auth-pkg",
				},
			},
			want: "custom-auth-pkg",
		},
		{
			name: "derived from output path segment starting with swift-",
			library: &config.Library{
				Name:   "google-cloud-auth",
				Output: "pkgs/swift-google-auth/Sources/GoogleCloudAuth/generated",
			},
			want: "swift-google-auth",
		},
		{
			name: "storage package derived from output",
			library: &config.Library{
				Name:   "google-cloud-storage",
				Output: "pkgs/swift-google-cloud-storage/Sources/generated/StorageControlProtos",
			},
			want: "swift-google-cloud-storage",
		},
		{
			name: "generated library output",
			library: &config.Library{
				Name:   "google-cloud-compute-v1",
				Output: "generated/swift-google-cloud-compute-v1",
			},
			want: "swift-google-cloud-compute-v1",
		},
		{
			name: "fallback prepends swift- when name has no swift- prefix",
			library: &config.Library{
				Name: "google-cloud-compute-v1",
			},
			want: "swift-google-cloud-compute-v1",
		},
		{
			name: "fallback preserves name when name already has swift- prefix",
			library: &config.Library{
				Name: "swift-google-cloud-compute-v1",
			},
			want: "swift-google-cloud-compute-v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := PackageName(test.library)
			if got != test.want {
				t.Errorf("PackageName() = %q, want %q", got, test.want)
			}
		})
	}
}
