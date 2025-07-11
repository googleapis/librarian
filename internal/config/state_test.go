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
	"testing"
)

func TestLibrarianState_Validate(t *testing.T) {
	for _, test := range []struct {
		name    string
		state   *LibrarianState
		wantErr bool
	}{
		{
			name: "valid state",
			state: &LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*LibraryState{
					{
						ID:          "a/b",
						SourcePaths: []string{"src/a", "src/b"},
						APIs: []API{
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
						SourcePaths: []string{"src/a", "src/b"},
						APIs: []API{
							{
								Path: "a/b/v1",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing libraries",
			state: &LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.state.Validate(); (err != nil) != test.wantErr {
				t.Errorf("LibrarianState.Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestLibrary_Validate(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *LibraryState
		wantErr bool
	}{
		{
			name: "valid library",
			library: &LibraryState{
				ID:          "a/b",
				SourcePaths: []string{"src/a", "src/b"},
				APIs: []API{
					{
						Path: "a/b/v1",
					},
				},
			},
		},
		{
			name: "missing id",
			library: &LibraryState{
				SourcePaths: []string{"src/a", "src/b"},
				APIs: []API{
					{
						Path: "a/b/v1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "id is dot",
			library: &LibraryState{
				ID:          ".",
				SourcePaths: []string{"src/a", "src/b"},
				APIs: []API{
					{
						Path: "a/b/v1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "id is double dot",
			library: &LibraryState{
				ID:          "..",
				SourcePaths: []string{"src/a", "src/b"},
				APIs: []API{
					{
						Path: "a/b/v1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing source paths",
			library: &LibraryState{
				ID: "a/b",
				APIs: []API{
					{
						Path: "a/b/v1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing apis",
			library: &LibraryState{
				ID:          "a/b",
				SourcePaths: []string{"src/a", "src/b"},
			},
			wantErr: true,
		},
		{
			name: "valid version without v prefix",
			library: &LibraryState{
				ID:          "a/b",
				Version:     "1.2.3",
				SourcePaths: []string{"src/a", "src/b"},
				APIs: []API{
					{
						Path: "a/b/v1",
					},
				},
			},
		},
		{
			name: "invalid id characters",
			library: &LibraryState{
				ID:          "a/b!",
				SourcePaths: []string{"src/a", "src/b"},
				APIs: []API{
					{
						Path: "a/b/v1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid last generated commit non-hex",
			library: &LibraryState{
				ID:                  "a/b",
				LastGeneratedCommit: "not-a-hex-string",
				SourcePaths:         []string{"src/a", "src/b"},
				APIs: []API{
					{
						Path: "a/b/v1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid last generated commit wrong length",
			library: &LibraryState{
				ID:                  "a/b",
				LastGeneratedCommit: "deadbeef",
				SourcePaths:         []string{"src/a", "src/b"},
				APIs: []API{
					{
						Path: "a/b/v1",
					},
				},
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.library.Validate(); (err != nil) != test.wantErr {
				t.Errorf("Library.Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestAPI_Validate(t *testing.T) {
	for _, test := range []struct {
		name    string
		api     *API
		wantErr bool
	}{
		{
			name: "valid api",
			api: &API{
				Path: "a/b/v1",
			},
		},
		{
			name:    "missing path",
			api:     &API{},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.api.Validate(); (err != nil) != test.wantErr {
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
