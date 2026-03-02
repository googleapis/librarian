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
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/semver"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

func TestGenerateRepoMetadata(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	library := &config.Library{
		Name:    "secretmanager",
		Output:  tmpDir,
		Version: "1.2.3",
		Go: &config.GoModule{
			GoAPIs: []*config.GoAPI{
				{
					ClientPackage: "secretmanager",
					ImportPath:    "secretmanager/apiv1",
					Path:          "google/cloud/secretmanager/v1",
				},
			},
		},
	}
	api := &serviceconfig.API{
		ShortName: "secretmanager",
		Title:     "Secret Manager API",
		Path:      "google/cloud/secretmanager/v1",
	}
	metadataDir := filepath.Join(tmpDir, "secretmanager", "apiv1")
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
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := generateRepoMetadata(api, library); err != nil {
		t.Fatal(err)
	}

	got, err := repometadata.Read(metadataDir)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateRepoMetadata_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		api     *serviceconfig.API
		library *config.Library
		setup   func(library *config.Library, api *serviceconfig.API, output string)
		wantErr error
	}{
		{
			name: "no go api",
			api: &serviceconfig.API{
				ShortName: "secretmanager",
				Path:      "google/cloud/secretmanager/v1",
			},
			library: &config.Library{
				Name: "secretmanager",
			},
			wantErr: errGoAPINotFound,
		},
		{
			name: "invalid version",
			api: &serviceconfig.API{
				ShortName: "secretmanager",
				Path:      "google/cloud/secretmanager/v1",
			},
			library: &config.Library{
				Name:    "secretmanager",
				Version: "invalid",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ClientPackage: "secretmanager",
							ImportPath:    "secretmanager/apiv1",
							Path:          "google/cloud/secretmanager/v1",
						},
					},
				},
			},
			wantErr: semver.ErrInvalidVersion,
		},
		{
			name: "invalid output directory",
			api: &serviceconfig.API{
				ShortName: "secretmanager",
				Path:      "google/cloud/secretmanager/v1",
			},
			library: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.3",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ClientPackage: "secretmanager",
							ImportPath:    "secretmanager/apiv1",
							Path:          "google/cloud/secretmanager/v1",
						},
					},
				},
			},
			setup: func(library *config.Library, api *serviceconfig.API, output string) {
				library.Output = output
				dir := filepath.Join(output, "secretmanager", "apiv1")
				// Create a file where the directory should be so Write fails.
				if err := os.MkdirAll(filepath.Dir(dir), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(dir, []byte("not a directory"), 0644); err != nil {
					t.Fatal(err)
				}

			},
			wantErr: syscall.ENOTDIR,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			if test.setup != nil {
				test.setup(test.library, test.api, tempDir)
			}
			err := generateRepoMetadata(test.api, test.library)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("metadataReleaseLevel() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestGoClientDocURL(t *testing.T) {
	for _, test := range []struct {
		name       string
		importPath string
		want       string
	}{
		{
			name:       "basic",
			importPath: "secretmanager/apiv1",
			want:       "https://cloud.google.com/go/docs/reference/cloud.google.com/go/secretmanager/latest/apiv1",
		},
		{
			name:       "spanner",
			importPath: "spanner/admin/database/apiv1",
			want:       "https://cloud.google.com/go/docs/reference/cloud.google.com/go/spanner/latest/admin/database/apiv1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := clientDocURL(test.importPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGoDistributionName(t *testing.T) {
	for _, test := range []struct {
		name       string
		importPath string
		want       string
	}{
		{
			name:       "basic",
			importPath: "secretmanager/apiv1",
			want:       "cloud.google.com/go/secretmanager/apiv1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := distributionName(test.importPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMetadataReleaseLevel(t *testing.T) {
	for _, test := range []struct {
		name    string
		api     *serviceconfig.API
		library *config.Library
		want    string
	}{
		{
			name: "stable",
			api: &serviceconfig.API{
				Path: "google/cloud/secretmanager/v1",
			},
			library: &config.Library{
				Version: "1.2.3",
			},
			want: "stable",
		},
		{
			name: "preview",
			api: &serviceconfig.API{
				Path: "google/cloud/secretmanager/v1beta1",
			},
			library: &config.Library{
				Version: "1.2.3",
			},
			want: "preview",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := metadataReleaseLevel(test.api, test.library)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("metadataReleaseLevel() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestMetadataReleaseLevel_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		api     *serviceconfig.API
		library *config.Library
		wantErr error
	}{
		{
			name: "invalid version",
			api: &serviceconfig.API{
				Path: "google/cloud/secretmanager/v1",
			},
			library: &config.Library{
				Version: "invalid",
			},
			wantErr: semver.ErrInvalidVersion,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := metadataReleaseLevel(test.api, test.library)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("metadataReleaseLevel() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
