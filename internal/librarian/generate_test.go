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

package librarian

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
)

func TestGenerateSingleLibrary(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name              string
		api               string
		repo              gitrepo.Repository
		state             *config.LibrarianState
		container         *mockContainerClient
		ghClient          GitHubClient
		wantLibraryID     string
		wantErr           bool
		wantGenerateCalls int
	}{
		{
			name:     "works",
			api:      "some/api",
			repo:     newTestGitRepo(t),
			ghClient: &mockGitHubClient{},
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []*config.API{{Path: "some/api"}},
					},
				},
			},
			container:         &mockContainerClient{},
			wantLibraryID:     "some-library",
			wantGenerateCalls: 1,
		},
		{
			name:     "works with no response",
			api:      "some/api",
			repo:     newTestGitRepo(t),
			ghClient: &mockGitHubClient{},
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []*config.API{{Path: "some/api"}},
					},
				},
			},
			container: &mockContainerClient{
				noGenerateResponse: true,
			},
			wantLibraryID:     "some-library",
			wantGenerateCalls: 1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			outputDir := t.TempDir()
			libraryID := "some-library"
			libraryState := test.state.LibraryByID(libraryID)
			err := generateSingleLibrary(t.Context(), test.container, test.state, libraryState, newTestGitRepo(t), test.repo, outputDir)
			if (err != nil) != test.wantErr {
				t.Errorf("generateSingleLibrary() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.wantGenerateCalls, test.container.generateCalls); diff != "" {
				t.Errorf("runGenerateCommand() generateCalls mismatch (-want +got):%s", diff)
			}
		})
	}
}

func TestGetExistingSrc(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		paths []string
		want  []string
	}{
		{
			name: "all_source_paths_existed",
			paths: []string{
				"a/path",
				"another/path",
			},
			want: []string{
				"a/path",
				"another/path",
			},
		},
		{
			name: "one_source_paths_existed",
			paths: []string{
				"a/path",
			},
			want: []string{
				"a/path",
			},
		},
		{
			name: "no_source_paths_existed",
			want: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repo := newTestGitRepo(t)
			state := &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "some-library",
						SourceRoots: []string{
							"a/path",
							"another/path",
						},
					},
				},
			}
			for _, path := range test.paths {
				relPath := filepath.Join(repo.GetDir(), path)
				if err := os.MkdirAll(relPath, 0755); err != nil {
					t.Fatal(err)
				}
			}

			r := &generateRunner{
				repo:  repo,
				state: state,
			}

			got := r.getExistingSrc("some-library")
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("getExistingSrc() mismatch (-want +got):%s", diff)
			}
		})
	}
}

func TestGetSafeDirectoryName(t *testing.T) {
	for _, test := range []struct {
		name string
		id   string
		want string
	}{
		{
			name: "simple",
			id:   "pubsub",
			want: "pubsub",
		},
		{
			name: "nested",
			id:   "pubsub/v2",
			want: "pubsub-slash-v2",
		},
		{
			name: "deeply nested",
			id:   "compute/metadata/v2",
			want: "compute-slash-metadata-slash-v2",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := getSafeDirectoryName(test.id)
			if test.want != got {
				t.Errorf("getSafeDirectoryName() = %q; want %q", got, test.want)
			}
		})
	}
}
