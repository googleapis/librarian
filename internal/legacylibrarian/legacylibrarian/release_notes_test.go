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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacycli"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacygithub"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacygitrepo"
)

func TestFormatReleaseNotes(t *testing.T) {
	t.Parallel()

	today := time.Now().Format("2006-01-02")
	hash1 := plumbing.NewHash("1234567890abcdef")
	hash2 := plumbing.NewHash("fedcba0987654321")
	hash3 := plumbing.NewHash("abcdefg123456789")
	hash4 := plumbing.NewHash("acdef12345678901")
	hash5 := plumbing.NewHash("bcdef12345678901")
	hash6 := plumbing.NewHash("cdefg12345678901")
	librarianVersion := legacycli.Version()

	for _, test := range []struct {
		name            string
		state           *legacyconfig.LibrarianState
		ghRepo          *legacygithub.Repository
		wantReleaseNote string
		wantErr         bool
		wantErrPhrase   string
	}{
		{
			name: "single library release",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID: "my-library",
						// this is the NewVersion in the release note.
						Version:         "1.1.0",
						PreviousVersion: "1.0.0",
						Changes: []*legacyconfig.Commit{
							{
								Type:       "feat",
								Subject:    "new feature",
								CommitHash: hash1.String(),
								LibraryIDs: "my-library",
							},
							{
								Type:       "fix",
								Subject:    "a bug fix",
								CommitHash: hash2.String(),
								LibraryIDs: "my-library",
							},
						},
						ReleaseTriggered: true,
					},
				},
			},
			ghRepo: &legacygithub.Repository{Owner: "owner", Name: "repo"},
			wantReleaseNote: fmt.Sprintf(`PR created by the Librarian CLI to initialize a release. Merging this PR will auto trigger a release.

Librarian Version: %s
Language Image: go:1.21
<details><summary>my-library: v1.1.0</summary>

## [v1.1.0](https://github.com/owner/repo/compare/my-library-1.0.0...my-library-1.1.0) (%s)

### Features

* new feature ([12345678](https://github.com/owner/repo/commit/12345678))

### Bug Fixes

* a bug fix ([fedcba09](https://github.com/owner/repo/commit/fedcba09))

</details>`,
				librarianVersion, today),
		},
		{
			name: "single library release, with cl num",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID: "my-library",
						// this is the NewVersion in the release note.
						Version:         "1.1.0",
						PreviousVersion: "1.0.0",
						Changes: []*legacyconfig.Commit{
							{
								Type:          "feat",
								Subject:       "new feature",
								CommitHash:    hash1.String(),
								PiperCLNumber: "123456",
								LibraryIDs:    "my-library",
							},
							{
								Type:          "fix",
								Subject:       "a bug fix",
								CommitHash:    hash2.String(),
								PiperCLNumber: "987654",
								LibraryIDs:    "my-library",
							},
						},
						ReleaseTriggered: true,
					},
				},
			},
			ghRepo: &legacygithub.Repository{Owner: "owner", Name: "repo"},
			wantReleaseNote: fmt.Sprintf(`PR created by the Librarian CLI to initialize a release. Merging this PR will auto trigger a release.

Librarian Version: %s
Language Image: go:1.21
<details><summary>my-library: v1.1.0</summary>

## [v1.1.0](https://github.com/owner/repo/compare/my-library-1.0.0...my-library-1.1.0) (%s)

### Features

* new feature (PiperOrigin-RevId: 123456) ([12345678](https://github.com/owner/repo/commit/12345678))

### Bug Fixes

* a bug fix (PiperOrigin-RevId: 987654) ([fedcba09](https://github.com/owner/repo/commit/fedcba09))

</details>`,
				librarianVersion, today),
		},
		{
			name: "single_library_with_multiple_features",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID: "my-library",
						// this is the NewVersion in the release note.
						Version:         "1.1.0",
						PreviousVersion: "1.0.0",
						Changes: []*legacyconfig.Commit{
							{
								Type:       "feat",
								Subject:    "new feature",
								CommitHash: hash1.String(),
								LibraryIDs: "my-library",
							},
							{
								Type:       "feat",
								Subject:    "another new feature",
								CommitHash: hash2.String(),
								LibraryIDs: "my-library",
							},
						},
						ReleaseTriggered: true,
					},
				},
			},
			ghRepo: &legacygithub.Repository{Owner: "owner", Name: "repo"},
			wantReleaseNote: fmt.Sprintf(`PR created by the Librarian CLI to initialize a release. Merging this PR will auto trigger a release.

Librarian Version: %s
Language Image: go:1.21
<details><summary>my-library: v1.1.0</summary>

## [v1.1.0](https://github.com/owner/repo/compare/my-library-1.0.0...my-library-1.1.0) (%s)

### Features

* new feature ([12345678](https://github.com/owner/repo/commit/12345678))

* another new feature ([fedcba09](https://github.com/owner/repo/commit/fedcba09))

</details>`,
				librarianVersion, today),
		},
		{
			name: "multiple library releases",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID: "lib-a",
						// this is the NewVersion in the release note.
						Version:          "1.1.0",
						PreviousVersion:  "1.0.0",
						ReleaseTriggered: true,
						Changes: []*legacyconfig.Commit{
							{
								Type:       "feat",
								Subject:    "feature for a",
								CommitHash: hash1.String(),
								LibraryIDs: "lib-a",
							},
						},
					},
					{
						ID: "lib-b",
						// this is the NewVersion in the release note.
						Version:          "2.0.1",
						PreviousVersion:  "2.0.0",
						ReleaseTriggered: true,
						Changes: []*legacyconfig.Commit{
							{
								Type:       "fix",
								Subject:    "fix for b",
								CommitHash: hash2.String(),
								LibraryIDs: "lib-b",
							},
						},
					},
				},
			},
			ghRepo: &legacygithub.Repository{Owner: "owner", Name: "repo"},
			wantReleaseNote: fmt.Sprintf(`PR created by the Librarian CLI to initialize a release. Merging this PR will auto trigger a release.

Librarian Version: %s
Language Image: go:1.21
<details><summary>lib-a: v1.1.0</summary>

## [v1.1.0](https://github.com/owner/repo/compare/lib-a-1.0.0...lib-a-1.1.0) (%s)

### Features

* feature for a ([12345678](https://github.com/owner/repo/commit/12345678))

</details>


<details><summary>lib-b: v2.0.1</summary>

## [v2.0.1](https://github.com/owner/repo/compare/lib-b-2.0.0...lib-b-2.0.1) (%s)

### Bug Fixes

* fix for b ([fedcba09](https://github.com/owner/repo/commit/fedcba09))

</details>`,
				librarianVersion, today, today),
		},
		{
			name: "release with ignored commit types",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID: "my-library",
						// this is the newVersion in the release note.
						Version:          "1.1.0",
						PreviousVersion:  "1.0.0",
						ReleaseTriggered: true,
						Changes: []*legacyconfig.Commit{
							{
								Type:       "feat",
								Subject:    "new feature",
								CommitHash: hash1.String(),
								LibraryIDs: "my-library",
							},
							{
								Type:       "ci",
								Subject:    "a ci change",
								CommitHash: hash2.String(),
								LibraryIDs: "my-library",
							},
						},
					},
				},
			},
			ghRepo: &legacygithub.Repository{Owner: "owner", Name: "repo"},
			wantReleaseNote: fmt.Sprintf(`PR created by the Librarian CLI to initialize a release. Merging this PR will auto trigger a release.

Librarian Version: %s
Language Image: go:1.21
<details><summary>my-library: v1.1.0</summary>

## [v1.1.0](https://github.com/owner/repo/compare/my-library-1.0.0...my-library-1.1.0) (%s)

### Features

* new feature ([12345678](https://github.com/owner/repo/commit/12345678))

</details>`,
				librarianVersion, today),
		},
		{
			name: "release_with_commit_description_and_body",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID: "my-library",
						// this is the newVersion in the release note.
						Version:          "1.1.0",
						PreviousVersion:  "1.0.0",
						ReleaseTriggered: true,
						Changes: []*legacyconfig.Commit{
							{
								Type:       "feat",
								Subject:    "new feature",
								Body:       "this is the body",
								CommitHash: hash1.String(),
								LibraryIDs: "my-library",
							},
						},
					},
				},
			},
			ghRepo: &legacygithub.Repository{Owner: "owner", Name: "repo"},
			wantReleaseNote: fmt.Sprintf(`PR created by the Librarian CLI to initialize a release. Merging this PR will auto trigger a release.

Librarian Version: %s
Language Image: go:1.21
<details><summary>my-library: v1.1.0</summary>

## [v1.1.0](https://github.com/owner/repo/compare/my-library-1.0.0...my-library-1.1.0) (%s)

### Features

* new feature ([12345678](https://github.com/owner/repo/commit/12345678))

</details>`,
				librarianVersion, today),
		},
		{
			name: "no releases",
			state: &legacyconfig.LibrarianState{
				Image:     "go:1.21",
				Libraries: []*legacyconfig.LibraryState{},
			},
			ghRepo: &legacygithub.Repository{},
			wantReleaseNote: fmt.Sprintf(`PR created by the Librarian CLI to initialize a release. Merging this PR will auto trigger a release.

Librarian Version: %s
Language Image: go:1.21`, librarianVersion),
		},
		{
			name: "generate with chore",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID: "my-library",
						// this is the newVersion in the release note.
						Version:          "1.1.0",
						PreviousVersion:  "1.0.0",
						ReleaseTriggered: true,
						Changes: []*legacyconfig.Commit{
							{
								Type:       "chore",
								Subject:    "some chore",
								Body:       "this is the body",
								CommitHash: hash1.String(),
								LibraryIDs: "my-library",
							},
						},
					},
				},
			},
			ghRepo: &legacygithub.Repository{Owner: "owner", Name: "repo"},
			wantReleaseNote: fmt.Sprintf(`PR created by the Librarian CLI to initialize a release. Merging this PR will auto trigger a release.

Librarian Version: %s
Language Image: go:1.21
<details><summary>my-library: v1.1.0</summary>

## [v1.1.0](https://github.com/owner/repo/compare/my-library-1.0.0...my-library-1.1.0) (%s)

</details>`,
				librarianVersion, today),
		},
		{
			name: "release_with_bulk_generation_commits",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:               "j",
						Version:          "1.1.0",
						PreviousVersion:  "1.0.0",
						ReleaseTriggered: true,
						Changes: []*legacyconfig.Commit{
							{
								Type:       "feat",
								Subject:    "new feature",
								CommitHash: hash1.String(),
								LibraryIDs: "j",
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
						ID:               "k",
						Version:          "2.4.0",
						PreviousVersion:  "2.3.0",
						ReleaseTriggered: true,
						Changes: []*legacyconfig.Commit{
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
			ghRepo: &legacygithub.Repository{Owner: "owner", Name: "repo"},
			wantReleaseNote: fmt.Sprintf(`PR created by the Librarian CLI to initialize a release. Merging this PR will auto trigger a release.

Librarian Version: %s
Language Image: go:1.21
<details><summary>j: v1.1.0</summary>

## [v1.1.0](https://github.com/owner/repo/compare/j-1.0.0...j-1.1.0) (%s)

### Features

* new feature ([12345678](https://github.com/owner/repo/commit/12345678))

</details>


<details><summary>k: v2.4.0</summary>

## [v2.4.0](https://github.com/owner/repo/compare/k-2.3.0...k-2.4.0) (%s)

</details>


<details><summary>Bulk Changes</summary>

* chore: bulk change 2 (PiperOrigin-RevId: 12345) ([abcdef00](https://github.com/owner/repo/commit/abcdef00))
  Libraries: j,k,l,m,n,o,p,q,r,s
* fix: bulk change ([fedcba09](https://github.com/owner/repo/commit/fedcba09))
  Libraries: a,b,c,d,e,f,g,h,i,j,k
</details>`,
				librarianVersion, today, today),
		},
		{
			name: "release_with_other_src_bulk_commits",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:              "ignored-library",
						Version:         "1.4.0",
						PreviousVersion: "1.3.0",
						// This library will not appear in the release notes because
						// release is not triggered for.
						ReleaseTriggered: false,
						Changes: []*legacyconfig.Commit{
							{
								Type:       "fix",
								Subject:    "this commit is ignored",
								CommitHash: "123456789012345",
								LibraryIDs: "ignored-library",
							},
						},
					},
					{
						ID:               "j",
						Version:          "1.1.0",
						PreviousVersion:  "1.0.0",
						ReleaseTriggered: true,
						Changes: []*legacyconfig.Commit{
							{
								Type:       "feat",
								Subject:    "new feature",
								CommitHash: hash1.String(),
								LibraryIDs: "j",
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
							{
								// This is a bulk commit, it appears in
								// bulk changes section.
								Type:       "chore",
								Subject:    "update dependency",
								CommitHash: hash4.String(),
								LibraryIDs: "j",
							},
						},
					},
					{
						ID:               "k",
						Version:          "2.4.0",
						PreviousVersion:  "2.3.0",
						ReleaseTriggered: true,
						Changes: []*legacyconfig.Commit{
							{
								Type:       "fix",
								Subject:    "bulk change",
								CommitHash: hash2.String(),
								LibraryIDs: "a,b,c,d,e,f,g,h,i,j,k",
							},
							{
								// This is a bulk commit, it appears in
								// bulk changes section.
								Type:       "chore",
								Subject:    "update dependency",
								CommitHash: hash4.String(),
								LibraryIDs: "k",
							},
						},
					},
					{
						ID:               "library-1",
						Version:          "2.4.0",
						PreviousVersion:  "2.3.0",
						ReleaseTriggered: true,
						Changes: []*legacyconfig.Commit{
							{
								// This is a bulk commit, it appears in
								// bulk changes section.
								Type:       "chore",
								Subject:    "update dependency",
								CommitHash: hash4.String(),
								LibraryIDs: "library-1",
							},
							{
								Type:       "docs",
								Subject:    "update README",
								CommitHash: hash5.String(),
								LibraryIDs: "library-1",
							},
						},
					},
					{
						ID:               "library-2",
						Version:          "2.4.0",
						PreviousVersion:  "2.3.0",
						ReleaseTriggered: true,
						Changes: []*legacyconfig.Commit{
							{
								// This is a bulk commit, it appears in
								// bulk changes section.
								Type:       "chore",
								Subject:    "update dependency",
								CommitHash: hash4.String(),
								LibraryIDs: "library-2",
							},
							{
								Type:       "docs",
								Subject:    "update README",
								CommitHash: hash5.String(),
								LibraryIDs: "library-2",
							},
						},
					},
					{
						ID:               "library-3",
						Version:          "2.4.0",
						PreviousVersion:  "2.3.0",
						ReleaseTriggered: true,
						Changes: []*legacyconfig.Commit{
							{
								// This is a bulk commit, it appears in
								// bulk changes section.
								Type:       "chore",
								Subject:    "update dependency",
								CommitHash: hash4.String(),
								LibraryIDs: "library-3",
							},
							{
								// This is a non-bulk commit because it only has two
								// library ids.
								// This commit will appear in library section.
								Type:       "fix",
								Subject:    "non bulk fix",
								CommitHash: hash6.String(),
								LibraryIDs: "library-3,library-4",
							},
						},
					},
					{
						ID:               "library-4",
						Version:          "2.4.0",
						PreviousVersion:  "2.3.0",
						ReleaseTriggered: true,
						Changes: []*legacyconfig.Commit{
							{
								// This is a bulk commit, it appears in
								// bulk changes section.
								Type:       "chore",
								Subject:    "update dependency",
								CommitHash: hash4.String(),
								LibraryIDs: "library-4",
							},
						},
					},
					{
						ID:               "library-5",
						Version:          "2.4.0",
						PreviousVersion:  "2.3.0",
						ReleaseTriggered: true,
						Changes: []*legacyconfig.Commit{
							{
								// This is a bulk commit, it appears in
								// bulk changes section.
								Type:       "chore",
								Subject:    "update dependency",
								CommitHash: hash4.String(),
								LibraryIDs: "library-5",
							},
						},
					},
					{
						ID:               "library-6",
						Version:          "2.4.0",
						PreviousVersion:  "2.3.0",
						ReleaseTriggered: true,
						Changes: []*legacyconfig.Commit{
							{
								// This is a bulk commit, it appears in
								// bulk changes section.
								Type:       "chore",
								Subject:    "update dependency",
								CommitHash: hash4.String(),
								LibraryIDs: "library-6",
							},
						},
					},
					{
						ID:               "library-7",
						Version:          "2.4.0",
						PreviousVersion:  "2.3.0",
						ReleaseTriggered: true,
						Changes: []*legacyconfig.Commit{
							{
								// This is a bulk commit, it appears in
								// bulk changes section.
								Type:       "chore",
								Subject:    "update dependency",
								CommitHash: hash4.String(),
								LibraryIDs: "library-7",
							},
						},
					},
					{
						ID:               "library-8",
						Version:          "2.4.0",
						PreviousVersion:  "2.3.0",
						ReleaseTriggered: true,
						Changes: []*legacyconfig.Commit{
							{
								// This is a bulk commit, it appears in
								// bulk changes section.
								Type:       "chore",
								Subject:    "update dependency",
								CommitHash: hash4.String(),
								LibraryIDs: "library-8",
							},
						},
					},
				},
			},
			ghRepo: &legacygithub.Repository{Owner: "owner", Name: "repo"},
			wantReleaseNote: fmt.Sprintf(`PR created by the Librarian CLI to initialize a release. Merging this PR will auto trigger a release.

Librarian Version: %s
Language Image: go:1.21
<details><summary>j: v1.1.0</summary>

## [v1.1.0](https://github.com/owner/repo/compare/j-1.0.0...j-1.1.0) (%s)

### Features

* new feature ([12345678](https://github.com/owner/repo/commit/12345678))

</details>


<details><summary>k: v2.4.0</summary>

## [v2.4.0](https://github.com/owner/repo/compare/k-2.3.0...k-2.4.0) (%s)

</details>


<details><summary>library-1: v2.4.0</summary>

## [v2.4.0](https://github.com/owner/repo/compare/library-1-2.3.0...library-1-2.4.0) (%s)

### Documentation

* update README ([bcdef123](https://github.com/owner/repo/commit/bcdef123))

</details>


<details><summary>library-2: v2.4.0</summary>

## [v2.4.0](https://github.com/owner/repo/compare/library-2-2.3.0...library-2-2.4.0) (%s)

### Documentation

* update README ([bcdef123](https://github.com/owner/repo/commit/bcdef123))

</details>


<details><summary>library-3: v2.4.0</summary>

## [v2.4.0](https://github.com/owner/repo/compare/library-3-2.3.0...library-3-2.4.0) (%s)

### Bug Fixes

* non bulk fix ([cdef0000](https://github.com/owner/repo/commit/cdef0000))

</details>


<details><summary>library-4: v2.4.0</summary>

## [v2.4.0](https://github.com/owner/repo/compare/library-4-2.3.0...library-4-2.4.0) (%s)

### Bug Fixes

* non bulk fix ([cdef0000](https://github.com/owner/repo/commit/cdef0000))

</details>


<details><summary>library-5: v2.4.0</summary>

## [v2.4.0](https://github.com/owner/repo/compare/library-5-2.3.0...library-5-2.4.0) (%s)

</details>


<details><summary>library-6: v2.4.0</summary>

## [v2.4.0](https://github.com/owner/repo/compare/library-6-2.3.0...library-6-2.4.0) (%s)

</details>


<details><summary>library-7: v2.4.0</summary>

## [v2.4.0](https://github.com/owner/repo/compare/library-7-2.3.0...library-7-2.4.0) (%s)

</details>


<details><summary>library-8: v2.4.0</summary>

## [v2.4.0](https://github.com/owner/repo/compare/library-8-2.3.0...library-8-2.4.0) (%s)

</details>


<details><summary>Bulk Changes</summary>

* chore: bulk change 2 (PiperOrigin-RevId: 12345) ([abcdef00](https://github.com/owner/repo/commit/abcdef00))
  Libraries: j,k,l,m,n,o,p,q,r,s
* chore: update dependency ([acdef123](https://github.com/owner/repo/commit/acdef123))
  Libraries: j,k,library-1,library-2,library-3,library-4,library-5,library-6,library-7,library-8
* fix: bulk change ([fedcba09](https://github.com/owner/repo/commit/fedcba09))
  Libraries: a,b,c,d,e,f,g,h,i,j,k
</details>`,
				librarianVersion, today, today, today, today, today, today, today, today, today, today),
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
		commit  *legacygitrepo.Commit
		want    string
		wantErr error
	}{
		{
			name: "found_piper_id",
			commit: &legacygitrepo.Commit{
				Message: "feat: add a new API\n\nPiperOrigin-RevId: 745187558",
			},
			want: "745187558",
		},
		{
			name: "invalid_commit",
			commit: &legacygitrepo.Commit{
				Message: "",
			},
			wantErr: legacygitrepo.ErrEmptyCommitMessage,
		},
		{
			name: "unconventional_commit",
			commit: &legacygitrepo.Commit{
				Message: "unconventional commit message",
			},
			wantErr: errPiperNotFound,
		},
		{
			name: "does_not_contain_piper_id",
			commit: &legacygitrepo.Commit{
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
		repo    legacygitrepo.Repository
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
					"abcd": {"a/b/c", "d/e/f"},
					"1234": {"g/h/i", "j/k/l"},
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
