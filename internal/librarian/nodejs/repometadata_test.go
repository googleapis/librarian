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

package nodejs

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/sample"
)

func TestGenerateRepoMetadata(t *testing.T) {
	absGoogleapisDir, err := filepath.Abs(googleapisDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Language: config.LanguageNodejs,
		Repo:     "googleapis/google-cloud-node",
	}
	for _, test := range []struct {
		name    string
		library *config.Library
		want    func() *repometadata.RepoMetadata
	}{
		{
			name: "no overrides",
			library: &config.Library{
				Name: "google-cloud-secretmanager",
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			},
			want: func() *repometadata.RepoMetadata {
				w := sample.RepoMetadata()
				w.DistributionName = "@google-cloud/secretmanager"
				w.Language = cfg.Language
				w.Repo = cfg.Repo
				w.ClientDocumentation = "https://cloud.google.com/nodejs/docs/reference/secretmanager/latest"
				w.ProductDocumentation = "https://cloud.google.com/secret-manager/docs"
				return w
			},
		},
		{
			name: "client documentation override",
			library: &config.Library{
				Name: "google-cloud-secretmanager",
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
				Nodejs: &config.NodejsPackage{
					ClientDocumentationOverride: "https://custom.docs.com/ref",
				},
			},
			want: func() *repometadata.RepoMetadata {
				w := sample.RepoMetadata()
				w.DistributionName = "@google-cloud/secretmanager"
				w.Language = cfg.Language
				w.Repo = cfg.Repo
				w.ClientDocumentation = "https://custom.docs.com/ref"
				w.ProductDocumentation = "https://cloud.google.com/secret-manager/docs"
				return w
			},
		},
		{
			name: "default version override",
			library: &config.Library{
				Name: "google-cloud-secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
					{Path: "google/cloud/secretmanager/v1beta"},
				},
				Nodejs: &config.NodejsPackage{
					DefaultVersion: "v1beta",
				},
			},
			want: func() *repometadata.RepoMetadata {
				w := sample.RepoMetadata()
				w.DistributionName = "@google-cloud/secretmanager"
				w.Language = cfg.Language
				w.Repo = cfg.Repo
				w.ClientDocumentation = "https://cloud.google.com/nodejs/docs/reference/secretmanager/latest"
				w.ProductDocumentation = "https://cloud.google.com/secret-manager/docs"
				w.DefaultVersion = "v1beta"
				return w
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := generateRepoMetadata(cfg, test.library, absGoogleapisDir)
			if err != nil {
				t.Fatal(err)
			}
			want := test.want()
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateRepoMetadata_Error(t *testing.T) {
	cfg := &config.Config{
		Language: config.LanguageNodejs,
		Repo:     "googleapis/google-cloud-node",
	}
	library := &config.Library{Name: "google-cloud-secretmanager"}
	_, err := generateRepoMetadata(cfg, library, googleapisDir)
	if !errors.Is(err, repometadata.ErrNoAPIs) {
		t.Errorf("error: %v; want %v", err, repometadata.ErrNoAPIs)
	}
}
