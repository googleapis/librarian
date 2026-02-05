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

func TestBuildPythonLibraries(t *testing.T) {
	for _, test := range []struct {
		name  string
		input *MigrationInput
		want  []*config.Library
	}{
		{
			name: "secret manager (keep paths, description override)",
			input: &MigrationInput{
				repoPath: "testdata/google-cloud-python",
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID:          "google-cloud-secret-manager",
							APIs:        []*legacyconfig.API{{Path: "google/cloud/secretmanager/v1"}},
							SourceRoots: []string{"packages/google-cloud-secret-manager"},
							PreserveRegex: []string{
								"packages/google-cloud-secret-manager/CHANGELOG.md",
								"docs/CHANGELOG.md",
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
			},
			want: []*config.Library{
				{
					Name:                "google-cloud-secret-manager",
					DescriptionOverride: "Stores, manages, and secures access to application secrets.",
					APIs:                []*config.API{{Path: "google/cloud/secretmanager/v1"}},
					Keep: []string{
						"CHANGELOG.md",
						"docs/CHANGELOG.md",
					},
				},
			},
		},
		{
			name: "workstations (preview release level)",
			input: &MigrationInput{
				repoPath: "testdata/google-cloud-python",
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID:   "google-cloud-workstations",
							APIs: []*legacyconfig.API{{Path: "google/cloud/workstations/v1"}},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
			},
			want: []*config.Library{
				{
					Name:         "google-cloud-workstations",
					ReleaseLevel: "preview",
					APIs:         []*config.API{{Path: "google/cloud/workstations/v1"}},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildPythonLibraries(test.input, "testdata/googleapis")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildPythonLibraries_Error(t *testing.T) {
	for _, test := range []struct {
		name  string
		input *MigrationInput
	}{
		{
			name: "preserve regex but no source roots",
			input: &MigrationInput{
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "google-cloud-secret-manager",
							PreserveRegex: []string{
								"packages/google-cloud-secret-manager/CHANGELOG.md",
								"docs/CHANGELOG.md",
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
			},
		},
		{
			name: "invalid preserve regex",
			input: &MigrationInput{
				repoPath: "testdata/google-cloud-python",
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID:          "google-cloud-secret-manager",
							SourceRoots: []string{"packages/google-cloud-secret-manager"},
							PreserveRegex: []string{
								"*",
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
			},
		},
		{
			name: "source root doesn't exist",
			input: &MigrationInput{
				repoPath: "testdata/google-cloud-python",
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID:          "google-cloud-secret-manager",
							SourceRoots: []string{"packages/missing"},
							PreserveRegex: []string{
								"*",
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
			},
		},
		{
			name: "repo metadata missing",
			input: &MigrationInput{
				repoPath: "testdata/google-cloud-python",
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "google-cloud-missing-metadata",
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
			},
		},
		{
			name: "repo metadata invalid",
			input: &MigrationInput{
				repoPath: "testdata/google-cloud-python",
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID: "google-cloud-bad-metadata",
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
			},
		},
		{
			name: "api not allow-listed",
			input: &MigrationInput{
				repoPath: "testdata/google-cloud-python",
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID:   "google-example-api-v1",
							APIs: []*legacyconfig.API{{Path: "google/example/api/v1"}},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
			},
		},
		{
			name: "source root isn't in root",
			input: &MigrationInput{
				repoPath: "testdata/google-cloud-python",
				librarianState: &legacyconfig.LibrarianState{
					Libraries: []*legacyconfig.LibraryState{
						{
							ID:          "google-cloud-secret-manager",
							SourceRoots: []string{"../get-git-commit"},
							PreserveRegex: []string{
								"*",
							},
						},
					},
				},
				librarianConfig: &legacyconfig.LibrarianConfig{},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := buildPythonLibraries(test.input, "testdata/googleapis")
			if err == nil {
				t.Errorf("expected error; got none")
			}
		})
	}
}
