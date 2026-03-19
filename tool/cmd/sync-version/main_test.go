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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
)

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
