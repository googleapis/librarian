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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
)

func TestFormatReleaseNotes(t *testing.T) {
	t.Parallel()

	today := time.Now().Format("2006-01-02")
	hash1 := plumbing.NewHash("1234567890abcdef")
	hash2 := plumbing.NewHash("fedcba0987654321")
	librarianVersion := cli.Version()

	for _, test := range []struct {
		name            string
		state           *config.LibrarianState
		repo            gitrepo.Repository
		wantReleaseNote string
		wantErr         bool
		wantErrPhrase   string
	}{
		{
			name: "single library release",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID:               "my-library",
						Version:          "1.0.0",
						ReleaseTriggered: true,
					},
				},
			},
			repo: &MockRepository{
				RemotesValue: []*git.Remote{git.NewRemote(nil, &gitconfig.RemoteConfig{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}})},
				GetCommitsForPathsSinceTagValueByTag: map[string][]*gitrepo.Commit{
					"my-library-1.0.0": {
						{Message: "feat: new feature", Hash: hash1},
						{Message: "fix: a bug fix", Hash: hash2},
					},
				},
				ChangedFilesInCommitValueByHash: map[string][]string{
					hash1.String(): {
						"path/to/file",
						"path/to/another/file",
					},
					hash2.String(): {
						"path/to/file",
					},
				},
			},
			wantReleaseNote: fmt.Sprintf(`Librarian Version: %s
Language Image: go:1.21

<details><summary>my-library: 1.1.0</summary>

## [1.1.0](https://github.com/owner/repo/compare/my-library-1.0.0...my-library-1.1.0) (%s)

### Features
* new feature ([1234567](https://github.com/owner/repo/commit/1234567890abcdef000000000000000000000000))

### Bug Fixes
* a bug fix ([fedcba0](https://github.com/owner/repo/commit/fedcba0987654321000000000000000000000000))

</details>
`,
				librarianVersion, today),
		},
		{
			name: "multiple library releases",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID:               "lib-a",
						Version:          "1.0.0",
						ReleaseTriggered: true,
					},
					{
						ID:               "lib-b",
						Version:          "2.0.0",
						ReleaseTriggered: true,
					},
				},
			},
			repo: &MockRepository{
				RemotesValue: []*git.Remote{git.NewRemote(nil, &gitconfig.RemoteConfig{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}})},
				GetCommitsForPathsSinceTagValueByTag: map[string][]*gitrepo.Commit{
					"lib-a-1.0.0": {
						{Message: "feat: feature for a", Hash: hash1},
					},
					"lib-b-2.0.0": {
						{Message: "fix: fix for b", Hash: hash2},
					},
				},
				ChangedFilesInCommitValueByHash: map[string][]string{
					hash1.String(): {"path/to/file"},
					hash2.String(): {"path/to/another/file"},
				},
			},
			wantReleaseNote: fmt.Sprintf(`Librarian Version: %s
Language Image: go:1.21

<details><summary>lib-a: 1.1.0</summary>

## [1.1.0](https://github.com/owner/repo/compare/lib-a-1.0.0...lib-a-1.1.0) (%s)

### Features
* feature for a ([1234567](https://github.com/owner/repo/commit/1234567890abcdef000000000000000000000000))

</details>
<details><summary>lib-b: 2.0.1</summary>

## [2.0.1](https://github.com/owner/repo/compare/lib-b-2.0.0...lib-b-2.0.1) (%s)

### Bug Fixes
* fix for b ([fedcba0](https://github.com/owner/repo/commit/fedcba0987654321000000000000000000000000))

</details>
`,
				librarianVersion, today, today),
		},
		{
			name: "release with ignored commit types",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID:               "my-library",
						Version:          "1.0.0",
						ReleaseTriggered: true,
					},
				},
			},
			repo: &MockRepository{
				RemotesValue: []*git.Remote{git.NewRemote(nil, &gitconfig.RemoteConfig{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}})},
				GetCommitsForPathsSinceTagValueByTag: map[string][]*gitrepo.Commit{
					"my-library-1.0.0": {
						{Message: "feat: new feature", Hash: hash1},
						{Message: "ci: a ci change", Hash: hash2},
					},
				},
				ChangedFilesInCommitValueByHash: map[string][]string{
					hash1.String(): {"path/to/file"},
					hash2.String(): {"path/to/another/file"},
				},
			},
			wantReleaseNote: fmt.Sprintf(`Librarian Version: %s
Language Image: go:1.21

<details><summary>my-library: 1.1.0</summary>

## [1.1.0](https://github.com/owner/repo/compare/my-library-1.0.0...my-library-1.1.0) (%s)

### Features
* new feature ([1234567](https://github.com/owner/repo/commit/1234567890abcdef000000000000000000000000))

</details>
`,
				librarianVersion, today),
		},
		{
			name: "no releases",
			state: &config.LibrarianState{
				Image:     "go:1.21",
				Libraries: []*config.LibraryState{},
			},
			repo:            &MockRepository{},
			wantReleaseNote: fmt.Sprintf("Librarian Version: %s\nLanguage Image: go:1.21\n\n", librarianVersion),
		},
		{
			name: "error getting commits",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID:               "my-library",
						Version:          "1.0.0",
						ReleaseTriggered: true,
					},
				},
			},
			repo: &MockRepository{
				RemotesValue:                    []*git.Remote{git.NewRemote(nil, &gitconfig.RemoteConfig{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}})},
				GetCommitsForPathsSinceTagError: fmt.Errorf("git error"),
			},
			wantErr:       true,
			wantErrPhrase: "failed to format release notes",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := FormatReleaseNotes(test.repo, test.state)
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				if !strings.Contains(err.Error(), test.wantErrPhrase) {
					t.Errorf("FormatReleaseNotes() returned error %q, want to contain %q", err.Error(), test.wantErrPhrase)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantReleaseNote, got); diff != "" {
				t.Errorf("FormatReleaseNotes() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
