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
		want     *legacyconfig.LibrarianState
	}{
		{
			name:     "success",
			repoPath: "testdata/sync-new-library",
			want: &legacyconfig.LibrarianState{
				Image: "test-image",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:                  "accessapproval",
						Version:             "1.9.0",
						APIs:                []*legacyconfig.API{},
						SourceRoots:         []string{"accessapproval", "internal/generated/snippets/accessapproval"},
						PreserveRegex:       []string{},
						RemoveRegex:         []string{},
						ReleaseExcludePaths: []string{"internal/generated/snippets/accessapproval/"},
						TagFormat:           "{id}/v{version}",
					},
					{
						ID:                  "accesscontextmanager",
						Version:             "1.9.7",
						APIs:                []*legacyconfig.API{},
						SourceRoots:         []string{"accesscontextmanager", "internal/generated/snippets/accesscontextmanager"},
						PreserveRegex:       []string{},
						RemoveRegex:         []string{},
						ReleaseExcludePaths: []string{"internal/generated/snippets/accesscontextmanager/"},
						TagFormat:           "{id}/v{version}",
					},
					{
						ID:                  "advisorynotifications",
						Version:             "1.5.6",
						APIs:                []*legacyconfig.API{},
						SourceRoots:         []string{"advisorynotifications", "internal/generated/snippets/advisorynotifications"},
						PreserveRegex:       []string{},
						RemoveRegex:         []string{},
						ReleaseExcludePaths: []string{"internal/generated/snippets/advisorynotifications/"},
						TagFormat:           "{id}/v{version}",
					},
					{
						ID:                  "apigeeconnect",
						Version:             "1.7.7",
						APIs:                []*legacyconfig.API{},
						SourceRoots:         []string{"apigeeconnect", "internal/generated/snippets/apigeeconnect"},
						PreserveRegex:       []string{},
						RemoveRegex:         []string{},
						ReleaseExcludePaths: []string{"internal/generated/snippets/apigeeconnect/"},
						TagFormat:           "{id}/v{version}",
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			statePath := filepath.Join(test.repoPath, ".librarian", "state.yaml")
			original, err := yaml.Read[legacyconfig.LibrarianState](statePath)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := yaml.Write(statePath, original); err != nil {
					t.Fatal()
				}
			})
			if err := run(t.Context(), []string{test.repoPath}); err != nil {
				t.Fatal(err)
			}
			got, err := yaml.Read[legacyconfig.LibrarianState](statePath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSyncNewLibrary(t *testing.T) {
	for _, test := range []struct {
		name  string
		state *legacyconfig.LibrarianState
		cfg   *config.Config
		want  *legacyconfig.LibrarianState
	}{
		{
			name: "sync new library",
			state: &legacyconfig.LibrarianState{
				Image: "test-image",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:          "existing",
						Version:     "1.0.0",
						SourceRoots: []string{"existing"},
						TagFormat:   "{id}/v{version}",
					},
				},
			},
			cfg: &config.Config{
				Libraries: []*config.Library{
					{Name: "aiplatform", Version: "1.0.0"},
					{Name: "secretmanager", Version: "1.2.0"},
				},
			},
			want: &legacyconfig.LibrarianState{
				Image: "test-image",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:                  "aiplatform",
						Version:             "1.0.0",
						SourceRoots:         []string{"aiplatform", "internal/generated/snippets/aiplatform"},
						ReleaseExcludePaths: []string{"internal/generated/snippets/aiplatform/"},
						TagFormat:           "{id}/v{version}",
					},
					{
						ID:          "existing",
						Version:     "1.0.0",
						SourceRoots: []string{"existing"},
						TagFormat:   "{id}/v{version}",
					},
					{
						ID:                  "secretmanager",
						Version:             "1.2.0",
						SourceRoots:         []string{"secretmanager", "internal/generated/snippets/secretmanager"},
						ReleaseExcludePaths: []string{"internal/generated/snippets/secretmanager/"},
						TagFormat:           "{id}/v{version}",
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := syncNewLibrary(test.state, test.cfg)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
