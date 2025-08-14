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

package config

import (
	"strings"
	"testing"
)

func TestLibrarianState_Validate(t *testing.T) {
	for _, test := range []struct {
		name       string
		state      *LibrarianState
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "valid state",
			state: &LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*LibraryState{
					{
						ID:          "a/b",
						SourceRoots: []string{"src/a", "src/b"},
						APIs: []*API{
							{
								Path: "a/b/v1",
							},
						},
					},
				},
			},
		},
		{
			name: "missing image",
			state: &LibrarianState{
				Libraries: []*LibraryState{
					{
						ID:          "a/b",
						SourceRoots: []string{"src/a", "src/b"},
						APIs: []*API{
							{
								Path: "a/b/v1",
							},
						},
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "image is required",
		},
		{
			name: "missing libraries",
			state: &LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
			},
			wantErr:    true,
			wantErrMsg: "libraries cannot be empty",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.state.Validate()
			if test.wantErr {
				if err == nil {
					t.Error("Librarian.Validate() should fail")
				}
				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message %q, got %q", test.wantErrMsg, err.Error())
				}

				return
			}

			if err != nil {
				t.Errorf("Librarian.Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestLibrary_Validate(t *testing.T) {
	for _, test := range []struct {
		name       string
		library    *LibraryState
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "valid library",
			library: &LibraryState{
				ID:          "a/b",
				SourceRoots: []string{"src/a", "src/b"},
				APIs: []*API{
					{
						Path: "a/b/v1",
					},
				},
			},
		},
		{
			name:       "missing id",
			library:    &LibraryState{},
			wantErr:    true,
			wantErrMsg: "id is required",
		},
		{
			name: "id is dot",
			library: &LibraryState{
				ID: ".",
			},
			wantErr:    true,
			wantErrMsg: "id cannot be",
		},
		{
			name: "id is double dot",
			library: &LibraryState{
				ID: "..",
			},
			wantErr:    true,
			wantErrMsg: "id cannot be",
		},
		{
			name: "missing source paths",
			library: &LibraryState{
				ID: "a/b",
				APIs: []*API{
					{
						Path: "a/b/v1",
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "source_roots cannot be empty",
		},
		{
			name: "missing apis",
			library: &LibraryState{
				ID:          "a/b",
				SourceRoots: []string{"src/a", "src/b"},
			},
			wantErr:    true,
			wantErrMsg: "apis cannot be empty",
		},
		{
			name: "valid version without v prefix",
			library: &LibraryState{
				ID:          "a/b",
				Version:     "1.2.3",
				SourceRoots: []string{"src/a", "src/b"},
				APIs: []*API{
					{
						Path: "a/b/v1",
					},
				},
			},
		},
		{
			name: "invalid id characters",
			library: &LibraryState{
				ID: "a/b!",
			},
			wantErr:    true,
			wantErrMsg: "invalid id",
		},
		{
			name: "invalid last generated commit non-hex",
			library: &LibraryState{
				ID:                  "a/b",
				LastGeneratedCommit: "not-a-hex-string",
			},
			wantErr:    true,
			wantErrMsg: "last_generated_commit must be a hex string",
		},
		{
			name: "invalid last generated commit wrong length",
			library: &LibraryState{
				ID:                  "a/b",
				LastGeneratedCommit: "deadbeef",
			},
			wantErr:    true,
			wantErrMsg: "last_generated_commit must be 40 characters",
		},
		{
			name: "valid preserve_regex",
			library: &LibraryState{
				ID:            "a/b",
				SourceRoots:   []string{"src/a"},
				APIs:          []*API{{Path: "a/b/v1"}},
				PreserveRegex: []string{".*\\.txt"},
			},
		},
		{
			name: "invalid preserve_regex",
			library: &LibraryState{
				ID:            "a/b",
				SourceRoots:   []string{"src/a"},
				APIs:          []*API{{Path: "a/b/v1"}},
				PreserveRegex: []string{"["},
			},
			wantErr:    true,
			wantErrMsg: "invalid preserve_regex at index",
		},
		{
			name: "valid remove_regex",
			library: &LibraryState{
				ID:          "a/b",
				SourceRoots: []string{"src/a"},
				APIs:        []*API{{Path: "a/b/v1"}},
				RemoveRegex: []string{".*\\.log"},
			},
		},
		{
			name: "invalid remove_regex",
			library: &LibraryState{
				ID:          "a/b",
				SourceRoots: []string{"src/a"},
				APIs:        []*API{{Path: "a/b/v1"}},
				RemoveRegex: []string{"("},
			},
			wantErr:    true,
			wantErrMsg: "invalid remove_regex at index",
		},
		{
			name: "valid release_exclude_path",
			library: &LibraryState{
				ID:                  "a/b",
				SourceRoots:         []string{"src/a"},
				APIs:                []*API{{Path: "a/b/v1"}},
				ReleaseExcludePaths: []string{"a/b", "c"},
			},
		},
		{
			name: "invalid release_exclude_path",
			library: &LibraryState{
				ID:                  "a/b",
				SourceRoots:         []string{"src/a"},
				APIs:                []*API{{Path: "a/b/v1"}},
				ReleaseExcludePaths: []string{"/a/b"},
			},
			wantErr:    true,
			wantErrMsg: "invalid release_exclude_path at index",
		},
		{
			name: "valid tag_format",
			library: &LibraryState{
				ID:          "a/b",
				SourceRoots: []string{"src/a"},
				APIs:        []*API{{Path: "a/b/v1"}},
				TagFormat:   "v{id}-{version}",
			},
		},
		{
			name: "invalid tag_format placeholder",
			library: &LibraryState{
				ID:          "a/b",
				SourceRoots: []string{"src/a"},
				APIs:        []*API{{Path: "a/b/v1"}},
				TagFormat:   "{id}-{foo}",
			},
			wantErr:    true,
			wantErrMsg: "invalid placeholder in tag_format",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.library.Validate()
			if test.wantErr {
				if err == nil {
					t.Error("Library.Validate() should fail")
				}
				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message %q, got %q", test.wantErrMsg, err.Error())
				}

				return
			}

			if err != nil {
				t.Errorf("Library.Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestAPI_Validate(t *testing.T) {
	for _, test := range []struct {
		name       string
		api        *API
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "new api",
			api: &API{
				Path: "a/b/v1",
			},
		},
		{
			name: "existing api",
			api: &API{
				Path: "a/b/v1",
			},
		},
		{
			name:       "missing path",
			api:        &API{},
			wantErr:    true,
			wantErrMsg: "invalid path",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.api.Validate()
			if test.wantErr {
				if err == nil {
					t.Error("API.Validate() should fail")
				}
				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message %q, got %q", test.wantErrMsg, err.Error())
				}

				return
			}

			if err != nil {
				t.Errorf("API.Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestIsValidDirPath(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		want bool
	}{
		{"valid", "a/b/c", true},
		{"valid with dots", "a/./b/../c", true},
		{"empty", "", false},
		{"absolute", "/a/b", false},
		{"up traversal", "../a", false},
		{"double dot", "..", false},
		{"single dot", ".", false},
		{"invalid chars", "a/b<c", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := isValidDirPath(test.path); got != test.want {
				t.Errorf("isValidDirPath(%q) = %v, want %v", test.path, got, test.want)
			}
		})
	}
}

func TestIsValidImage(t *testing.T) {
	for _, test := range []struct {
		name  string
		image string
		want  bool
	}{
		{"valid with tag", "gcr.io/google/go-container:v1", true},
		{"valid with latest tag", "ubuntu:latest", true},
		{"valid with port and tag", "my-registry:5000/my/image:v1", true},
		{"invalid no tag", "gcr.io/google/go-container", false},
		{"invalid with port no tag", "my-registry:5000/my/image", false},
		{"invalid with spaces", "gcr.io/google/go-container with spaces", false},
		{"invalid no repo", ":v1", false},
		{"invalid empty tag", "my-image:", false},
		{"invalid empty", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := isValidImage(test.image); got != test.want {
				t.Errorf("isValidImage(%q) = %v, want %v", test.image, got, test.want)
			}
		})
	}
}
