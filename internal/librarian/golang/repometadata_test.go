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

package golang

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

func TestGenerateRepoMetadata_Success(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	library := &config.Library{
		Name:         "secretmanager",
		Output:       tmpDir,
		ReleaseLevel: "stable",
	}
	api := &serviceconfig.API{
		ShortName: "secretmanager",
		Title:     "Secret Manager API",
		Path:      "google/cloud/secretmanager/v1",
	}
	path := filepath.Join(tmpDir, "secretmanager", "apiv1")
	want := &repometadata.RepoMetadata{
		APIShortname:        "secretmanager",
		ClientDocumentation: "https://cloud.google.com/go/docs/reference/cloud.google.com/go/secretmanager/latest/apiv1",
		ClientLibraryType:   "generated",
		Description:         "Secret Manager API",
		DistributionName:    "cloud.google.com/go/secretmanager/apiv1",
		Language:            "go",
		LibraryType:         repometadata.GAPICAutoLibraryType,
		ReleaseLevel:        "stable",
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := generateRepoMetadata(api, library); err != nil {
		t.Fatal(err)
	}

	got, err := repometadata.Read(path)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateRepoMetadata_Error(t *testing.T) {
	tmpDir := t.TempDir()
	library := &config.Library{
		Name:   "secretmanager",
		Output: tmpDir,
	}
	api := &serviceconfig.API{
		ShortName: "secretmanager",
		Path:      "google/cloud/secretmanager/v1",
	}

	dir, _ := resolveClientPath(library, api.Path)
	// Create a file where the directory should be so Write fails.
	if err := os.MkdirAll(filepath.Dir(dir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir, []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}

	err := generateRepoMetadata(api, library)
	if err == nil {
		t.Error("generateRepoMetadata() error = nil, want error")
	}
}

func TestGoClientDocURL(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		apiPath string
		want    string
	}{
		{
			name: "go",
			library: &config.Library{
				Name: "secretmanager",
			},
			apiPath: "google/cloud/secretmanager/v1",
			want:    "https://cloud.google.com/go/docs/reference/cloud.google.com/go/secretmanager/latest/apiv1",
		},
		{
			name: "has client directory",
			library: &config.Library{
				Name: "ai",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:            "google/ai/generativelanguage/v1",
							ClientDirectory: "generativelanguage",
						},
					},
				},
			},
			apiPath: "google/ai/generativelanguage/v1",
			want:    "https://cloud.google.com/go/docs/reference/cloud.google.com/go/ai/latest/generativelanguage/apiv1",
		},
		{
			name: "client directory with another api path",
			library: &config.Library{
				Name: "ai",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:            "google/ai/generativelanguage/v1beta1",
							ClientDirectory: "generativelanguage",
						},
					},
				},
			},
			apiPath: "google/ai/generativelanguage/v1",
			want:    "https://cloud.google.com/go/docs/reference/cloud.google.com/go/ai/latest/apiv1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := clientDocURL(test.library, test.apiPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGoDistributionName(t *testing.T) {
	for _, test := range []struct {
		name        string
		library     *config.Library
		apiPath     string
		serviceName string
		want        string
	}{
		{
			name: "has client directory",
			library: &config.Library{
				Name: "ai",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:            "google/ai/generativelanguage/v1",
							ClientDirectory: "generativelanguage",
						},
					},
				},
			},
			serviceName: "ai",
			apiPath:     "google/ai/generativelanguage/v1",
			want:        "cloud.google.com/go/ai/generativelanguage/apiv1",
		},
		{
			name: "does not have client directory",
			library: &config.Library{
				Name: "ai",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path: "google/ai/generativelanguage/v1beta1",
						},
					},
				},
			},
			serviceName: "ai",
			apiPath:     "google/ai/generativelanguage/v1",
			want:        "cloud.google.com/go/ai/apiv1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := distributionName(test.library, test.apiPath, test.serviceName)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
