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
	"errors"
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

func TestFormatGenerationPRBody(t *testing.T) {
	t.Parallel()

	today := time.Now()
	hash1 := plumbing.NewHash("1234567890abcdef")
	hash2 := plumbing.NewHash("fedcba0987654321")
	librarianVersion := cli.Version()

	for _, test := range []struct {
		name          string
		state         *config.LibrarianState
		repo          gitrepo.Repository
		want          string
		wantErr       bool
		wantErrPhrase string
	}{
		{
			name: "single library generation",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID:                  "one-library",
						LastGeneratedCommit: "1234567890",
					},
				},
			},
			repo: &MockRepository{
				RemotesValue: []*git.Remote{git.NewRemote(nil, &gitconfig.RemoteConfig{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}})},
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*gitrepo.Commit{
					"1234567890": {
						{
							Message: "feat: new feature\n\nThis is body.\n\nPiperOrigin-RevId: 98765",
							Hash:    hash1,
							When:    today,
						},
						{
							Message: "fix: a bug fix\n\nThis is another body.\n\nPiperOrigin-RevId: 573342",
							Hash:    hash2,
							When:    today.Add(time.Hour),
						},
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
			want: fmt.Sprintf(`This pull request is generated with proto changes between
[googleapis/googleapis@1234567](https://github.com/googleapis/googleapis/commit/1234567890abcdef000000000000000000000000)
(exclusive) and
[googleapis/googleapis@fedcba0](https://github.com/googleapis/googleapis/commit/fedcba0987654321000000000000000000000000)
(inclusive).

Librarian Version: %s
Language Image: %s

BEGIN_COMMIT_OVERRIDE
BEGIN_NESTED_COMMIT
fix: [one-library] a bug fix
This is another body.

PiperOrigin-RevId: 573342

Source-link: [googleapis/googleapis@fedcba0](https://github.com/googleapis/googleapis/commit/fedcba0987654321000000000000000000000000)
END_NESTED_COMMIT
BEGIN_NESTED_COMMIT
feat: [one-library] new feature
This is body.

PiperOrigin-RevId: 98765

Source-link: [googleapis/googleapis@1234567](https://github.com/googleapis/googleapis/commit/1234567890abcdef000000000000000000000000)
END_NESTED_COMMIT
END_COMMIT_OVERRIDE`,
				librarianVersion, "go:1.21"),
		},
		{
			name: "failed to get conventional commits",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID:                  "one-library",
						LastGeneratedCommit: "1234567890",
					},
				},
			},
			repo: &MockRepository{
				GetCommitsForPathsSinceLastGenError: errors.New("simulated error"),
			},
			wantErr:       true,
			wantErrPhrase: "failed to fetch conventional commits for library",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := formatGenerationPRBody(test.repo, test.state)
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				if !strings.Contains(err.Error(), test.wantErrPhrase) {
					t.Errorf("formatGenerationPRBody() returned error %q, want to contain %q", err.Error(), test.wantErrPhrase)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("formatGenerationPRBody() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

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
			got, err := formatReleaseNotes(test.repo, test.state)
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				if !strings.Contains(err.Error(), test.wantErrPhrase) {
					t.Errorf("formatReleaseNotes() returned error %q, want to contain %q", err.Error(), test.wantErrPhrase)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantReleaseNote, got); diff != "" {
				t.Errorf("formatReleaseNotes() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
