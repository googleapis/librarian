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
	"errors"
	"os"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
)

func TestRun(t *testing.T) {
	for _, test := range []struct {
		name     string
		repoPath string
	}{
		{
			name:     "success",
			repoPath: "testdata/config-check",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := run([]string{test.repoPath}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRun_Error(t *testing.T) {
	for _, test := range []struct {
		name     string
		repoPath string
		wantErr  error
	}{
		{
			name:     "no state.yaml",
			repoPath: "testdata/no-state",
			wantErr:  os.ErrNotExist,
		},
		{
			name:     "no librarian.yaml",
			repoPath: "testdata/no-librarian",
			wantErr:  os.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := run([]string{test.repoPath})
			if !errors.Is(err, test.wantErr) {
				t.Errorf("got error %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestSyncNewLibrary(t *testing.T) {
	for _, test := range []struct {
		name  string
		state *legacyconfig.LibrarianState
		cfg   *config.Config
	}{
		{
			name: "library match in state.yaml and librarian.yaml",
			state: &legacyconfig.LibrarianState{
				Image: "test-image",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:          "lib-1",
						Version:     "1.0.0",
						SourceRoots: []string{"existing"},
						TagFormat:   "{id}/v{version}",
					},
					{
						ID:          "lib-2",
						Version:     "1.2.0",
						SourceRoots: []string{"another"},
						TagFormat:   "{id}/v{version}",
					},
				},
			},
			cfg: &config.Config{
				Libraries: []*config.Library{
					{Name: "lib-1", Version: "1.0.0"},
					{Name: "lib-2", Version: "1.2.0"},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := configCheck(test.state, test.cfg); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSyncNewLibrary_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		state   *legacyconfig.LibrarianState
		cfg     *config.Config
		wantErr error
	}{
		{
			name: "a library exists in state.yaml but not librarian.yaml",
			state: &legacyconfig.LibrarianState{
				Image: "test-image",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:          "lib-1",
						Version:     "1.0.0",
						SourceRoots: []string{"existing"},
						TagFormat:   "{id}/v{version}",
					},
					{
						ID:          "lib-2",
						Version:     "1.2.0",
						SourceRoots: []string{"another"},
						TagFormat:   "{id}/v{version}",
					},
				},
			},
			cfg: &config.Config{
				Libraries: []*config.Library{
					{Name: "lib-1", Version: "1.0.0"},
				},
			},
			wantErr: errLibNotFoundInLibrarianYAML,
		},
		{
			name: "a library exists in librarian.yaml but not state.yaml",
			state: &legacyconfig.LibrarianState{
				Image: "test-image",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:          "lib-1",
						Version:     "1.0.0",
						SourceRoots: []string{"existing"},
						TagFormat:   "{id}/v{version}",
					},
				},
			},
			cfg: &config.Config{
				Libraries: []*config.Library{
					{Name: "lib-1", Version: "1.0.0"},
					{Name: "lib-2", Version: "1.2.0"},
				},
			},
			wantErr: errLibNotFoundInStateYAML,
		},
		{
			name: "a library exists in two configs but version is different",
			state: &legacyconfig.LibrarianState{
				Image: "test-image",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:          "lib-1",
						Version:     "1.0.0",
						SourceRoots: []string{"existing"},
						TagFormat:   "{id}/v{version}",
					},
					{
						ID:          "lib-2",
						Version:     "1.1.0",
						SourceRoots: []string{"another"},
						TagFormat:   "{id}/v{version}",
					},
				},
			},
			cfg: &config.Config{
				Libraries: []*config.Library{
					{Name: "lib-1", Version: "1.0.0"},
					{Name: "lib-2", Version: "1.2.0"},
				},
			},
			wantErr: errLibraryVersionNotSame,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := configCheck(test.state, test.cfg)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("got error %v, want %v", err, test.wantErr)
			}
		})
	}
}
