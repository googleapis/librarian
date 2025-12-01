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

package legacylibrarian

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
)

func TestBuildSingleLibrary(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name           string
		libraryID      string
		container      *mockContainerClient
		wantBuildCalls int
		wantErr        bool
	}{
		{
			name:           "build_with_library_id",
			libraryID:      "some-library",
			container:      &mockContainerClient{},
			wantBuildCalls: 1,
		},
		{
			name:           "build_with_no_library_id",
			container:      &mockContainerClient{},
			wantErr:        true,
			wantBuildCalls: 0,
		},
		{
			name:      "build_with_no_response",
			libraryID: "some-library",
			container: &mockContainerClient{
				noBuildResponse: true,
			},
			wantBuildCalls: 1,
		},
		{
			name:      "build_with_docker_command_error_files_restored",
			libraryID: "some-library",
			container: &mockContainerClient{
				buildErr: errors.New("simulate build error"),
			},
			wantErr: true,
		},
		{
			name:      "build_with_error_response_in_response",
			libraryID: "some-library",
			container: &mockContainerClient{
				wantErrorMsg: true,
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repo := newTestGitRepo(t)
			state := &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
					{
						ID: "some-library",
						SourceRoots: []string{
							"a/path",
							"another/path",
						},
					},
				},
			}

			// Create library files and commit the change.
			repoDir := repo.GetDir()
			for _, library := range state.Libraries {
				for _, srcPath := range library.SourceRoots {
					relPath := filepath.Join(repoDir, srcPath)
					if err := os.MkdirAll(relPath, 0755); err != nil {
						t.Fatal(err)
					}
					file := filepath.Join(relPath, "example.txt")
					if err := os.WriteFile(file, []byte("old content"), 0755); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err := repo.AddAll(); err != nil {
				t.Fatal(err)
			}
			if err := repo.Commit("test commit"); err != nil {
				t.Fatal(err)
			}

			libraryState := state.LibraryByID(test.libraryID)
			err := buildSingleLibrary(t.Context(), test.container, state, libraryState, repo)
			if test.wantErr {
				if err == nil {
					t.Fatal(err)
				}
				// Verify the library files are restore.
				for _, library := range state.Libraries {
					for _, srcPath := range library.SourceRoots {
						file := filepath.Join(repoDir, srcPath, "example.txt")
						readFile, err := os.ReadFile(file)
						if err != nil {
							t.Fatal(err)
						}
						if diff := cmp.Diff("old content", string(readFile)); diff != "" {
							t.Errorf("file content mismatch (-want +got):%s", diff)
						}

						newFile := filepath.Join(repoDir, srcPath, "another_example.txt")
						if _, err := os.Stat(newFile); !os.IsNotExist(err) {
							t.Fatal(err)
						}
					}
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantBuildCalls, test.container.buildCalls); diff != "" {
				t.Errorf("runBuildCommand() buildCalls mismatch (-want +got):%s", diff)
			}
		})
	}
}
