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

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

func TestFormatReleaseNotes(t *testing.T) {
	t.Parallel()

	today := time.Now().Format("2006-01-02")
	hash1 := plumbing.NewHash("1234567890abcdef")
	hash2 := plumbing.NewHash("fedcba0987654321")
	hash3 := plumbing.NewHash("abcdefg123456789")
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
						Changes: []*config.Commit{
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

* new feature ([12345678](https://github.com/owner/repo/commit/12345678))

### Bug Fixes

* a bug fix ([fedcba09](https://github.com/owner/repo/commit/fedcba09))

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
						Changes: []*config.Commit{
							{
								Type:          "feat",
								Subject:       "new feature",
								CommitHash:    hash1.String(),
								PiperCLNumber: "123456",
							},
							{
								Type:          "fix",
								Subject:       "a bug fix",
								CommitHash:    hash2.String(),
								PiperCLNumber: "987654",
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

* new feature (PiperOrigin-RevId: 123456) ([12345678](https://github.com/owner/repo/commit/12345678))

### Bug Fixes

* a bug fix (PiperOrigin-RevId: 987654) ([fedcba09](https://github.com/owner/repo/commit/fedcba09))

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
						Changes: []*config.Commit{
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

* new feature ([12345678](https://github.com/owner/repo/commit/12345678))

* another new feature ([fedcba09](https://github.com/owner/repo/commit/fedcba09))

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
						Changes: []*config.Commit{
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
						Changes: []*config.Commit{
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

* feature for a ([12345678](https://github.com/owner/repo/commit/12345678))

</details>


<details><summary>lib-b: 2.0.1</summary>

## [2.0.1](https://github.com/owner/repo/compare/lib-b-2.0.0...lib-b-2.0.1) (%s)

### Bug Fixes

* fix for b ([fedcba09](https://github.com/owner/repo/commit/fedcba09))

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
						Changes: []*config.Commit{
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

* new feature ([12345678](https://github.com/owner/repo/commit/12345678))

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
						Changes: []*config.Commit{
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

* new feature ([12345678](https://github.com/owner/repo/commit/12345678))

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
		{
			name: "generate with chore",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID: "my-library",
						// this is the newVersion in the release note.
						Version:          "1.1.0",
						PreviousVersion:  "1.0.0",
						ReleaseTriggered: true,
						Changes: []*config.Commit{
							{
								Type:       "chore",
								Subject:    "some chore",
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

### Miscellaneous Chores

* some chore ([12345678](https://github.com/owner/repo/commit/12345678))

</details>`,
				librarianVersion, today),
		},
		{
			name: "release with bulk commits",
			state: &config.LibrarianState{
				Image: "go:1.21",
				Libraries: []*config.LibraryState{
					{
						ID: "j",
						// this is the newVersion in the release note.
						Version:          "1.1.0",
						PreviousVersion:  "1.0.0",
						ReleaseTriggered: true,
						Changes: []*config.Commit{
							{
								Type:       "feat",
								Subject:    "new feature",
								CommitHash: hash1.String(),
							},
							{
								Type:       "fix",
								Subject:    "bulk change",
								CommitHash: hash2.String(),
								LibraryIDs: "a,b,c,d,e,f,g,h,i,j,k",
							},
							{
								Type:          "chore",
								Subject:       "bulk change 2",
								CommitHash:    hash3.String(),
								LibraryIDs:    "j,k,l,m,n,o,p,q,r,s",
								PiperCLNumber: "12345",
							},
						},
					},
					{
						ID: "j",
						// this is the newVersion in the release note.
						Version:          "1.1.0",
						PreviousVersion:  "1.0.0",
						ReleaseTriggered: true,
						Changes: []*config.Commit{
							{
								Type:       "feat",
								Subject:    "new feature",
								CommitHash: hash1.String(),
							},
							{
								Type:       "fix",
								Subject:    "bulk change",
								CommitHash: hash2.String(),
								LibraryIDs: "a,b,c,d,e,f,g,h,i,j,k",
							},
						},
					},
				},
			},
			ghRepo: &github.Repository{Owner: "owner", Name: "repo"},
			wantReleaseNote: fmt.Sprintf(`Librarian Version: %s
Language Image: go:1.21
<details><summary>j: 1.1.0</summary>

## [1.1.0](https://github.com/owner/repo/compare/j-1.0.0...j-1.1.0) (%s)

### Features

* new feature ([12345678](https://github.com/owner/repo/commit/12345678))

</details>


<details><summary>j: 1.1.0</summary>

## [1.1.0](https://github.com/owner/repo/compare/j-1.0.0...j-1.1.0) (2025-10-15)

### Features

* new feature ([12345678](https://github.com/owner/repo/commit/12345678))

</details>


<details><summary>Bulk Changes</summary>

* chore: bulk change 2 (PiperOrigin-RevId: 12345) ([abcdef00](https://github.com/owner/repo/commit/abcdef00))
  Libraries: j,k,l,m,n,o,p,q,r,s
* fix: bulk change ([fedcba09](https://github.com/owner/repo/commit/fedcba09))
  Libraries: a,b,c,d,e,f,g,h,i,j,k
</details>`,
				librarianVersion, today),
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

func TestFindPiperIDFrom(t *testing.T) {
	for _, test := range []struct {
		name    string
		commit  *gitrepo.Commit
		want    string
		wantErr error
	}{
		{
			name: "found_piper_id",
			commit: &gitrepo.Commit{
				Message: "feat: add a new API\n\nPiperOrigin-RevId: 745187558",
			},
			want: "745187558",
		},
		{
			name: "invalid_commit",
			commit: &gitrepo.Commit{
				Message: "",
			},
			wantErr: gitrepo.ErrEmptyCommitMessage,
		},
		{
			name: "unconventional_commit",
			commit: &gitrepo.Commit{
				Message: "unconventional commit message",
			},
			wantErr: errPiperNotFound,
		},
		{
			name: "does_not_contain_piper_id",
			commit: &gitrepo.Commit{
				Message: "feat: add a new API",
			},
			wantErr: errPiperNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := findPiperIDFrom(test.commit, "example-id")
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("unexpected error type: got %v, want %v", err, test.wantErr)
				}

				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("findPiperIDFrom() mismatch (-want +got):%s", diff)
			}
		})
	}
}

func TestLanguageRepoChangedFiles(t *testing.T) {
	for _, test := range []struct {
		name    string
		repo    gitrepo.Repository
		want    []string
		wantErr bool
	}{
		{
			name: "IsClean fails",
			repo: &MockRepository{
				IsCleanError: fmt.Errorf("mock failure from IsClean"),
			},
			wantErr: true,
		},
		{
			name: "clean, HeadHash fails",
			repo: &MockRepository{
				IsCleanValue:  true,
				HeadHashError: fmt.Errorf("mock failure from HeadHash"),
			},
			wantErr: true,
		},
		{
			name: "clean, ChangedFilesInCommit fails",
			repo: &MockRepository{
				IsCleanValue:              true,
				HeadHashValue:             "1234",
				ChangedFilesInCommitError: fmt.Errorf("mock failure from ChangedFilesInCommit"),
			},
			wantErr: true,
		},
		{
			name: "dirty, ChangedFiles fails",
			repo: &MockRepository{
				ChangedFilesError: fmt.Errorf("mock failure from ChangedFiles"),
			},
			wantErr: true,
		},
		{
			name: "clean success",
			repo: &MockRepository{
				IsCleanValue:  true,
				HeadHashValue: "1234",
				ChangedFilesInCommitValueByHash: map[string][]string{
					"abcd": []string{"a/b/c", "d/e/f"},
					"1234": []string{"g/h/i", "j/k/l"},
				},
			},
			want: []string{"g/h/i", "j/k/l"},
		},
		{
			name: "dirty success",
			repo: &MockRepository{
				ChangedFilesValue: []string{"a/b/c", "d/e/f"},
			},
			want: []string{"a/b/c", "d/e/f"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := languageRepoChangedFiles(test.repo)
			if (err != nil) != test.wantErr {
				t.Errorf("languageRepoChangedFiles() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
