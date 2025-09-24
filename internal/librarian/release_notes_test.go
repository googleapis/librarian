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

	"github.com/googleapis/librarian/internal/conventionalcommits"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

func TestFormatGenerationPRBody(t *testing.T) {
	t.Parallel()

	today := time.Now()
	hash1 := plumbing.NewHash("1234567890abcdef")
	hash2 := plumbing.NewHash("fedcba0987654321")
	librarianVersion := cli.Version()

	for _, test := range []struct {
		name            string
		state           *config.LibrarianState
		repo            gitrepo.Repository
		idToCommits     map[string]string
		failedLibraries []string
		want            string
		wantErr         bool
		wantErrPhrase   string
	}{
		{
			// This test verifies that only changed libraries appear in the pull request
			// body.
			name: "multiple libraries generation",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID: "one-library",
					},
					{
						ID: "another-library",
					},
				},
			},
			repo: &MockRepository{
				RemotesValue: []*git.Remote{git.NewRemote(nil, &gitconfig.RemoteConfig{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}})},
				GetCommitByHash: map[string]*gitrepo.Commit{
					"1234567890": {
						Hash: plumbing.NewHash("1234567890"),
						When: time.UnixMilli(200),
					},
					"abcdefg": {
						Hash: plumbing.NewHash("abcdefg"),
						When: time.UnixMilli(300),
					},
				},
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*gitrepo.Commit{
					"1234567890": {
						{
							Message: "fix: a bug fix\n\nThis is another body.\n\nPiperOrigin-RevId: 573342",
							Hash:    hash2,
							When:    today.Add(time.Hour),
						},
					},
					"abcdefg": {}, // no new commits since commit "abcdefg".
				},
				ChangedFilesInCommitValueByHash: map[string][]string{
					hash2.String(): {
						"path/to/file",
					},
				},
			},
			idToCommits: map[string]string{
				"one-library":     "1234567890",
				"another-library": "abcdefg",
			},
			failedLibraries: []string{},
			want: fmt.Sprintf(`BEGIN_COMMIT_OVERRIDE

BEGIN_NESTED_COMMIT
fix: [one-library] a bug fix
This is another body.

PiperOrigin-RevId: 573342

Source-link: [googleapis/googleapis@fedcba0](https://github.com/googleapis/googleapis/commit/fedcba0987654321000000000000000000000000)
END_NESTED_COMMIT

END_COMMIT_OVERRIDE

This pull request is generated with proto changes between
[googleapis/googleapis@abcdef0](https://github.com/googleapis/googleapis/commit/abcdef0000000000000000000000000000000000)
(exclusive) and
[googleapis/googleapis@fedcba0](https://github.com/googleapis/googleapis/commit/fedcba0987654321000000000000000000000000)
(inclusive).

Librarian Version: %s
Language Image: %s`,
				librarianVersion, "go:1.21"),
		},
		{
			name: "multiple libraries generation with failed libraries",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID: "one-library",
					},
					{
						ID: "another-library",
					},
				},
			},
			repo: &MockRepository{
				RemotesValue: []*git.Remote{git.NewRemote(nil, &gitconfig.RemoteConfig{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}})},
				GetCommitByHash: map[string]*gitrepo.Commit{
					"1234567890": {
						Hash: plumbing.NewHash("1234567890"),
						When: time.UnixMilli(200),
					},
					"abcdefg": {
						Hash: plumbing.NewHash("abcdefg"),
						When: time.UnixMilli(300),
					},
				},
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*gitrepo.Commit{
					"1234567890": {
						{
							Message: "fix: a bug fix\n\nThis is another body.\n\nPiperOrigin-RevId: 573342",
							Hash:    hash2,
							When:    today.Add(time.Hour),
						},
					},
					"abcdefg": {}, // no new commits since commit "abcdefg".
				},
				ChangedFilesInCommitValueByHash: map[string][]string{
					hash2.String(): {
						"path/to/file",
					},
				},
			},
			idToCommits: map[string]string{
				"one-library":     "1234567890",
				"another-library": "abcdefg",
			},
			failedLibraries: []string{
				"failed-library-a",
				"failed-library-b",
			},
			want: fmt.Sprintf(`BEGIN_COMMIT_OVERRIDE

BEGIN_NESTED_COMMIT
fix: [one-library] a bug fix
This is another body.

PiperOrigin-RevId: 573342

Source-link: [googleapis/googleapis@fedcba0](https://github.com/googleapis/googleapis/commit/fedcba0987654321000000000000000000000000)
END_NESTED_COMMIT

END_COMMIT_OVERRIDE

This pull request is generated with proto changes between
[googleapis/googleapis@abcdef0](https://github.com/googleapis/googleapis/commit/abcdef0000000000000000000000000000000000)
(exclusive) and
[googleapis/googleapis@fedcba0](https://github.com/googleapis/googleapis/commit/fedcba0987654321000000000000000000000000)
(inclusive).

Librarian Version: %s
Language Image: %s

## Generation failed for
- failed-library-a
- failed-library-b`,
				librarianVersion, "go:1.21"),
		},
		{
			name: "single library generation",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID: "one-library",
					},
				},
			},
			repo: &MockRepository{
				RemotesValue: []*git.Remote{git.NewRemote(nil, &gitconfig.RemoteConfig{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}})},
				GetCommitByHash: map[string]*gitrepo.Commit{
					"1234567890": {
						Hash: plumbing.NewHash("1234567890"),
						When: time.UnixMilli(200),
					},
				},
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
			idToCommits: map[string]string{
				"one-library": "1234567890",
			},
			failedLibraries: []string{},
			want: fmt.Sprintf(`BEGIN_COMMIT_OVERRIDE

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

END_COMMIT_OVERRIDE

This pull request is generated with proto changes between
[googleapis/googleapis@1234567](https://github.com/googleapis/googleapis/commit/1234567890000000000000000000000000000000)
(exclusive) and
[googleapis/googleapis@fedcba0](https://github.com/googleapis/googleapis/commit/fedcba0987654321000000000000000000000000)
(inclusive).

Librarian Version: %s
Language Image: %s`,
				librarianVersion, "go:1.21"),
		},
		{
			name: "no conventional commit is found since last generation",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID: "one-library",
						// Intentionally set this value to verify the test can pass.
						LastGeneratedCommit: "randomCommit",
					},
				},
			},
			repo: &MockRepository{
				RemotesValue:   []*git.Remote{git.NewRemote(nil, &gitconfig.RemoteConfig{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}})},
				GetCommitError: errors.New("simulated get commit error"),
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
			idToCommits: map[string]string{
				"one-library": "1234567890",
			},
			wantErr:       true,
			wantErrPhrase: "failed to find the start commit",
		},
		{
			name: "no conventional commits since last generation",
			state: &config.LibrarianState{
				Image:     "go:1.21",
				Libraries: []*config.LibraryState{{ID: "one-library"}},
			},
			repo: &MockRepository{},
			idToCommits: map[string]string{
				"one-library": "",
			},
			want: "No commit is found since last generation",
		},
		{
			name: "failed to get conventional commits",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID: "one-library",
					},
				},
			},
			repo: &MockRepository{
				GetCommitsForPathsSinceLastGenError: errors.New("simulated error"),
			},
			idToCommits: map[string]string{
				"one-library": "1234567890",
			},
			wantErr:       true,
			wantErrPhrase: "failed to fetch conventional commits for library",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := formatGenerationPRBody(test.repo, test.state, test.idToCommits, test.failedLibraries)
			if test.wantErr {
				if err == nil {
					t.Fatalf("%s should return error", test.name)
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

func TestFindLatestCommit(t *testing.T) {
	t.Parallel()

	today := time.Now()
	hash1 := plumbing.NewHash("1234567890abcdef")
	hash2 := plumbing.NewHash("fedcba0987654321")
	hash3 := plumbing.NewHash("ghfgsfgshfsdf232")
	for _, test := range []struct {
		name          string
		state         *config.LibrarianState
		repo          gitrepo.Repository
		idToCommits   map[string]string
		want          *gitrepo.Commit
		wantErr       bool
		wantErrPhrase string
	}{
		{
			name: "find the last generated commit",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "one-library",
					},
					{
						ID: "another-library",
					},
					{
						ID: "yet-another-library",
					},
					{
						ID: "skipped-library",
					},
				},
			},
			repo: &MockRepository{
				GetCommitByHash: map[string]*gitrepo.Commit{
					hash1.String(): {
						Hash:    hash1,
						Message: "this is a message",
						When:    today.Add(time.Hour),
					},
					hash2.String(): {
						Hash:    hash2,
						Message: "this is another message",
						When:    today.Add(2 * time.Hour).Add(time.Minute),
					},
					hash3.String(): {
						Hash:    hash3,
						Message: "yet another message",
						When:    today.Add(2 * time.Hour),
					},
				},
			},
			idToCommits: map[string]string{
				"one-library":         hash1.String(),
				"another-library":     hash2.String(),
				"yet-another-library": hash3.String(),
			},
			want: &gitrepo.Commit{
				Hash:    hash2,
				Message: "this is another message",
				When:    today.Add(2 * time.Hour).Add(time.Minute),
			},
		},
		{
			name: "failed to find last generated commit",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "one-library",
					},
				},
			},
			repo: &MockRepository{
				GetCommitError: errors.New("simulated error"),
			},
			idToCommits: map[string]string{
				"one-library": "1234567890",
			},
			wantErr:       true,
			wantErrPhrase: "can't find last generated commit for",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := findLatestGenerationCommit(test.repo, test.state, test.idToCommits)
			if test.wantErr {
				if err == nil {
					t.Fatalf("%s should return error", test.name)
				}
				if !strings.Contains(err.Error(), test.wantErrPhrase) {
					t.Errorf("findLatestCommit() returned error %q, want to contain %q", err.Error(), test.wantErrPhrase)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("findLatestCommit() mismatch (-want +got):\n%s", diff)
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
		ghRepo          *github.Repository
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
						ID: "my-library",
						// this is the NewVersion in the release note.
						Version:         "1.1.0",
						PreviousVersion: "1.0.0",
						Changes: []*conventionalcommits.ConventionalCommit{
							{
								Type:       "feat",
								Subject:    "new feature",
								CommitHash: hash1.String(),
							},
							{
								Type:       "fix",
								Subject:    "a bug fix",
								CommitHash: hash2.String(),
							},
						},
						ReleaseTriggered: true,
					},
				},
			},
			ghRepo: &github.Repository{Owner: "owner", Name: "repo"},
			wantReleaseNote: fmt.Sprintf(`Librarian Version: %s
Language Image: go:1.21
<details><summary>my-library: 1.1.0</summary>

## [1.1.0](https://github.com/owner/repo/compare/my-library-1.0.0...my-library-1.1.0) (%s)

### Features

* new feature ([1234567](https://github.com/owner/repo/commit/1234567))

### Bug Fixes

* a bug fix ([fedcba0](https://github.com/owner/repo/commit/fedcba0))

</details>`,
				librarianVersion, today),
		},
		{
			name: "single library release, with cl num",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID: "my-library",
						// this is the NewVersion in the release note.
						Version:         "1.1.0",
						PreviousVersion: "1.0.0",
						Changes: []*conventionalcommits.ConventionalCommit{
							{
								Type:       "feat",
								Subject:    "new feature",
								CommitHash: hash1.String(),
								Footers: map[string]string{
									"PiperOrigin-RevId": "123456",
								},
							},
							{
								Type:       "fix",
								Subject:    "a bug fix",
								CommitHash: hash2.String(),
								Footers: map[string]string{
									"PiperOrigin-RevId": "987654",
								},
							},
						},
						ReleaseTriggered: true,
					},
				},
			},
			ghRepo: &github.Repository{Owner: "owner", Name: "repo"},
			wantReleaseNote: fmt.Sprintf(`Librarian Version: %s
Language Image: go:1.21
<details><summary>my-library: 1.1.0</summary>

## [1.1.0](https://github.com/owner/repo/compare/my-library-1.0.0...my-library-1.1.0) (%s)

### Features

* new feature (PiperOrigin-RevId: 123456) ([1234567](https://github.com/owner/repo/commit/1234567))

### Bug Fixes

* a bug fix (PiperOrigin-RevId: 987654) ([fedcba0](https://github.com/owner/repo/commit/fedcba0))

</details>`,
				librarianVersion, today),
		},
		{
			name: "single_library_with_multiple_features",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID: "my-library",
						// this is the NewVersion in the release note.
						Version:         "1.1.0",
						PreviousVersion: "1.0.0",
						Changes: []*conventionalcommits.ConventionalCommit{
							{
								Type:       "feat",
								Subject:    "new feature",
								CommitHash: hash1.String(),
							},
							{
								Type:       "feat",
								Subject:    "another new feature",
								CommitHash: hash2.String(),
							},
						},
						ReleaseTriggered: true,
					},
				},
			},
			ghRepo: &github.Repository{Owner: "owner", Name: "repo"},
			wantReleaseNote: fmt.Sprintf(`Librarian Version: %s
Language Image: go:1.21
<details><summary>my-library: 1.1.0</summary>

## [1.1.0](https://github.com/owner/repo/compare/my-library-1.0.0...my-library-1.1.0) (%s)

### Features

* new feature ([1234567](https://github.com/owner/repo/commit/1234567))

* another new feature ([fedcba0](https://github.com/owner/repo/commit/fedcba0))

</details>`,
				librarianVersion, today),
		},
		{
			name: "multiple library releases",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID: "lib-a",
						// this is the NewVersion in the release note.
						Version:          "1.1.0",
						PreviousVersion:  "1.0.0",
						ReleaseTriggered: true,
						Changes: []*conventionalcommits.ConventionalCommit{
							{
								Type:       "feat",
								Subject:    "feature for a",
								CommitHash: hash1.String(),
							},
						},
					},
					{
						ID: "lib-b",
						// this is the NewVersion in the release note.
						Version:          "2.0.1",
						PreviousVersion:  "2.0.0",
						ReleaseTriggered: true,
						Changes: []*conventionalcommits.ConventionalCommit{
							{
								Type:       "fix",
								Subject:    "fix for b",
								CommitHash: hash2.String(),
							},
						},
					},
				},
			},
			ghRepo: &github.Repository{Owner: "owner", Name: "repo"},
			wantReleaseNote: fmt.Sprintf(`Librarian Version: %s
Language Image: go:1.21
<details><summary>lib-a: 1.1.0</summary>

## [1.1.0](https://github.com/owner/repo/compare/lib-a-1.0.0...lib-a-1.1.0) (%s)

### Features

* feature for a ([1234567](https://github.com/owner/repo/commit/1234567))

</details>


<details><summary>lib-b: 2.0.1</summary>

## [2.0.1](https://github.com/owner/repo/compare/lib-b-2.0.0...lib-b-2.0.1) (%s)

### Bug Fixes

* fix for b ([fedcba0](https://github.com/owner/repo/commit/fedcba0))

</details>`,
				librarianVersion, today, today),
		},
		{
			name: "release with ignored commit types",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID: "my-library",
						// this is the newVersion in the release note.
						Version:          "1.1.0",
						PreviousVersion:  "1.0.0",
						ReleaseTriggered: true,
						Changes: []*conventionalcommits.ConventionalCommit{
							{
								Type:       "feat",
								Subject:    "new feature",
								CommitHash: hash1.String(),
							},
							{
								Type:       "ci",
								Subject:    "a ci change",
								CommitHash: hash2.String(),
							},
						},
					},
				},
			},
			ghRepo: &github.Repository{Owner: "owner", Name: "repo"},
			wantReleaseNote: fmt.Sprintf(`Librarian Version: %s
Language Image: go:1.21
<details><summary>my-library: 1.1.0</summary>

## [1.1.0](https://github.com/owner/repo/compare/my-library-1.0.0...my-library-1.1.0) (%s)

### Features

* new feature ([1234567](https://github.com/owner/repo/commit/1234567))

</details>`,
				librarianVersion, today),
		},
		{
			name: "release_with_commit_description_and_body",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID: "my-library",
						// this is the newVersion in the release note.
						Version:          "1.1.0",
						PreviousVersion:  "1.0.0",
						ReleaseTriggered: true,
						Changes: []*conventionalcommits.ConventionalCommit{
							{
								Type:       "feat",
								Subject:    "new feature",
								Body:       "this is the body",
								CommitHash: hash1.String(),
							},
						},
					},
				},
			},
			ghRepo: &github.Repository{Owner: "owner", Name: "repo"},
			wantReleaseNote: fmt.Sprintf(`Librarian Version: %s
Language Image: go:1.21
<details><summary>my-library: 1.1.0</summary>

## [1.1.0](https://github.com/owner/repo/compare/my-library-1.0.0...my-library-1.1.0) (%s)

### Features

* new feature ([1234567](https://github.com/owner/repo/commit/1234567))

</details>`,
				librarianVersion, today),
		},
		{
			name: "no releases",
			state: &config.LibrarianState{
				Image:     "go:1.21",
				Libraries: []*config.LibraryState{},
			},
			ghRepo:          &github.Repository{},
			wantReleaseNote: fmt.Sprintf("Librarian Version: %s\nLanguage Image: go:1.21", librarianVersion),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := formatReleaseNotes(test.state, test.ghRepo)
			if test.wantErr {
				if err == nil {
					t.Fatalf("%s should return error", test.name)
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
