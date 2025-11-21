// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package legacyconfig

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDetermineTagFormat(t *testing.T) {
	for _, test := range []struct {
		name            string
		libraryState    *LibraryState
		librarianConfig *LibrarianConfig
		want            string
		wantErrMsg      string
	}{
		{
			name: "uses_default",
			libraryState: &LibraryState{
				ID: "example-library",
			},
			librarianConfig: &LibrarianConfig{},
			want:            defaultTagFormat,
		},
		{
			name: "prefers_per-library_from_config",
			libraryState: &LibraryState{
				ID:        "example-library",
				TagFormat: "per-library-tag-format-from-state",
			},
			librarianConfig: &LibrarianConfig{
				TagFormat: "from-config",
				Libraries: []*LibraryConfig{
					{
						LibraryID: "example-library",
						TagFormat: "per-library-tag-format-from-config",
					},
				},
			},
			want: "per-library-tag-format-from-config",
		},
		{
			name: "prefers_from_config",
			libraryState: &LibraryState{
				ID:        "example-library",
				TagFormat: "per-library-tag-format-from-state",
			},
			librarianConfig: &LibrarianConfig{
				TagFormat: "from-config",
				Libraries: []*LibraryConfig{
					{
						LibraryID: "example-library",
					},
				},
			},
			want: "from-config",
		},
		{
			name: "falls_back_to_per-library_from_state",
			libraryState: &LibraryState{
				ID:        "example-library",
				TagFormat: "per-library-tag-format-from-state",
			},
			librarianConfig: &LibrarianConfig{},
			want:            "per-library-tag-format-from-state",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DetermineTagFormat("example-library", test.libraryState, test.librarianConfig)

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("determineTagFormat() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatTag(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		library *LibraryState
		want    string
	}{
		{
			name: "default_format",
			library: &LibraryState{
				ID:      "google.cloud.foo.v1",
				Version: "1.2.3",
			},
			want: "google.cloud.foo.v1-1.2.3",
		},
		{
			name: "custom_format",
			library: &LibraryState{
				ID:        "google.cloud.foo.v1",
				Version:   "1.2.3",
				TagFormat: "v{version}-{id}",
			},
			want: "v1.2.3-google.cloud.foo.v1",
		},
		{
			name: "custom_format_version_only",
			library: &LibraryState{
				ID:        "google.cloud.foo.v1",
				Version:   "1.2.3",
				TagFormat: "v{version}",
			},
			want: "v1.2.3",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := FormatTag(test.library.TagFormat, test.library.ID, test.library.Version)
			if got != test.want {
				t.Errorf("FormatTag() = %q, want %q", got, test.want)
			}
		})
	}
}
