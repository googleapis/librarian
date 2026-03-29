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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
)

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
						ReleaseExcludePaths: []string{"internal/generated/snippets/aiplatform"},
					},
					{
						ID:          "existing",
						Version:     "1.0.0",
						SourceRoots: []string{"existing"}},
					{
						ID:                  "secretmanager",
						Version:             "1.2.0",
						SourceRoots:         []string{"secretmanager", "internal/generated/snippets/secretmanager"},
						ReleaseExcludePaths: []string{"internal/generated/snippets/secretmanager"},
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
