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

package main

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestRun(t *testing.T) {
	for _, test := range []struct {
		name     string
		repoPath string
		want     *config.Config
	}{
		{
			name:     "success",
			repoPath: "testdata/sync-version",
			want: &config.Config{
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "6c3dce4a02401667e9dd6d28304fee3f98e20ff8",
						SHA256: "c119da87c44afc55cd7afc0829c7e321a0fb03ff5c8bdbe0c00d27e7f30a8c63",
					},
				},
				Libraries: []*config.Library{
					{
						Name:    "accessapproval",
						Version: "1.9.0",
						APIs:    []*config.API{{Path: "google/cloud/accessapproval/v1"}},
					},
					{
						Name:    "accesscontextmanager",
						Version: "1.10.0",
						APIs:    []*config.API{{Path: "google/identity/accesscontextmanager/v1"}},
						Go: &config.GoModule{
							GoAPIs: []*config.GoAPI{
								{
									Path:       "google/identity/accesscontextmanager/v1",
									ImportPath: "accesscontextmanager/apiv1",
								},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfgPath := filepath.Join(test.repoPath, "librarian.yaml")
			original, err := yaml.Read[config.Config](cfgPath)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := yaml.Write(cfgPath, original); err != nil {
					t.Fatal()
				}
			})
			if err := run(t.Context(), []string{test.repoPath}); err != nil {
				t.Fatal(err)
			}
			got, err := yaml.Read[config.Config](cfgPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSyncVersion(t *testing.T) {
	for _, test := range []struct {
		name            string
		legacyLibraries []*legacyconfig.LibraryState
		libraries       []*config.Library
		want            []*config.Library
	}{
		{
			name: "update versions for libraries in legacylibrarian state.yaml",
			legacyLibraries: []*legacyconfig.LibraryState{
				{ID: "lib1", Version: "1.1.0"},
				{ID: "lib2", Version: "2.1.0"},
			},
			libraries: []*config.Library{
				{Name: "lib1", Version: "1.0.0"},
				{Name: "lib3", Version: "3.0.0"},
			},
			want: []*config.Library{
				{Name: "lib1", Version: "1.1.0"},
				{Name: "lib3", Version: "3.0.0"},
			},
		},
		{
			name: "smaller version is not synced",
			legacyLibraries: []*legacyconfig.LibraryState{
				{ID: "lib1", Version: "1.1.0"},
				{ID: "lib2", Version: "2.1.0"},
			},
			libraries: []*config.Library{
				{Name: "lib1", Version: "1.0.0"},
				{Name: "lib2", Version: "2.2.0"},
			},
			want: []*config.Library{
				{Name: "lib1", Version: "1.1.0"},
				{Name: "lib2", Version: "2.2.0"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := &legacyconfig.LibrarianState{Libraries: test.legacyLibraries}
			cfg := &config.Config{Libraries: test.libraries}
			got := syncVersion(state, cfg)
			if diff := cmp.Diff(test.want, got.Libraries); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
