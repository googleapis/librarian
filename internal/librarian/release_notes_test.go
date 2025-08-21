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
	"testing"
	"time"

	"github.com/googleapis/librarian/internal/conventionalcommits"

	"github.com/google/go-cmp/cmp"
)

func TestFormatReleaseNotes(t *testing.T) {
	today := time.Now().Format("2006-01-02")

	tests := []struct {
		name      string
		releases  map[string]*LibraryRelease
		repoOwner string
		repoName  string

		librarianVersion string
		languageImage    string
		wantReleaseNote  string
	}{
		{
			name: "single library release",
			releases: map[string]*LibraryRelease{
				"my-library": {
					PreviousTag: "my-library-v1.0.0",
					NewTag:      "my-library-v1.1.0",
					NewVersion:  "1.1.0",
					Commits: []*conventionalcommits.ConventionalCommit{
						{Type: "feat", Description: "new feature", SHA: "1234567890abcdef"},
						{Type: "fix", Description: "a bug fix", SHA: "fedcba0987654321"},
					},
				},
			},
			repoOwner: "owner",
			repoName:  "repo",

			librarianVersion: "v1.2.3",
			languageImage:    "go:1.21",
			wantReleaseNote: fmt.Sprintf(`Librarian Version: v1.2.3
Language Image: go:1.21

<details><summary>my-library: 1.1.0</summary>

## [1.1.0](https://github.com/owner/repo/compare/my-library-v1.0.0...my-library-v1.1.0) (%s)

### Features
* new feature ([1234567](https://github.com/owner/repo/commit/1234567890abcdef))

### Bug Fixes
* a bug fix ([fedcba0](https://github.com/owner/repo/commit/fedcba0987654321))

</details>`, today),
		},
		{
			name: "multiple library releases",
			releases: map[string]*LibraryRelease{
				"lib-b": {
					PreviousTag: "lib-b-v2.0.0",
					NewTag:      "lib-b-v2.0.1",
					NewVersion:  "2.0.1",
					Commits: []*conventionalcommits.ConventionalCommit{
						{Type: "fix", Description: "fix for b", SHA: "bbbbbbbb"},
					},
				},
				"lib-a": {
					PreviousTag: "lib-a-v1.0.0",
					NewTag:      "lib-a-v1.1.0",
					NewVersion:  "1.1.0",
					Commits: []*conventionalcommits.ConventionalCommit{
						{Type: "feat", Description: "feature for a", SHA: "aaaaaaaa"},
					},
				},
			},
			repoOwner: "owner",
			repoName:  "repo",

			librarianVersion: "v1.2.3",
			languageImage:    "go:1.21",
			wantReleaseNote: fmt.Sprintf(`Librarian Version: v1.2.3
Language Image: go:1.21

<details><summary>lib-a: 1.1.0</summary>

## [1.1.0](https://github.com/owner/repo/compare/lib-a-v1.0.0...lib-a-v1.1.0) (%s)

### Features
* feature for a ([aaaaaaa](https://github.com/owner/repo/commit/aaaaaaaa))

</details>
<details><summary>lib-b: 2.0.1</summary>

## [2.0.1](https://github.com/owner/repo/compare/lib-b-v2.0.0...lib-b-v2.0.1) (%s)

### Bug Fixes
* fix for b ([bbbbbbb](https://github.com/owner/repo/commit/bbbbbbbb))

</details>`, today, today),
		},
		{
			name: "release with ignored commit types",
			releases: map[string]*LibraryRelease{
				"my-library": {
					PreviousTag: "my-library-v1.0.0",
					NewTag:      "my-library-v1.1.0",
					NewVersion:  "1.1.0",
					Commits: []*conventionalcommits.ConventionalCommit{
						{Type: "feat", Description: "new feature", SHA: "1234567890abcdef"},
						{Type: "ci", Description: "a ci change", SHA: "fedcba0987654321"},
					},
				},
			},
			repoOwner: "owner",
			repoName:  "repo",

			librarianVersion: "v1.2.3",
			languageImage:    "go:1.21",
			wantReleaseNote: fmt.Sprintf(`Librarian Version: v1.2.3
Language Image: go:1.21

<details><summary>my-library: 1.1.0</summary>

## [1.1.0](https://github.com/owner/repo/compare/my-library-v1.0.0...my-library-v1.1.0) (%s)

### Features
* new feature ([1234567](https://github.com/owner/repo/commit/1234567890abcdef))

</details>`, today),
		},
		{
			name:      "no releases",
			releases:  map[string]*LibraryRelease{},
			repoOwner: "owner",
			repoName:  "repo",

			librarianVersion: "v1.2.3",
			languageImage:    "go:1.21",
			wantReleaseNote:  "Librarian Version: v1.2.3\nLanguage Image: go:1.21\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatReleaseNotes(tt.releases, tt.repoOwner, tt.repoName, tt.librarianVersion, tt.languageImage)
			if diff := cmp.Diff(tt.wantReleaseNote, got); diff != "" {
				t.Errorf("FormatReleaseNotes() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
