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
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacycli"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacygitrepo"
)

func TestFormatGenerationPRBody(t *testing.T) {
	t.Parallel()

	today := time.Now()
	hash1 := plumbing.NewHash("1234567890abcdef")
	hash2 := plumbing.NewHash("fedcba0987654321")
	librarianVersion := legacycli.Version()

	for _, test := range []struct {
		name            string
		state           *legacyconfig.LibrarianState
		sourceRepo      legacygitrepo.Repository
		languageRepo    legacygitrepo.Repository
		idToCommits     map[string]string
		failedLibraries []string
		api             string
		library         string
		apiOnboarding   bool
		want            string
		wantErr         bool
		wantErrPhrase   string
	}{
		{
			// This test verifies that only changed libraries appear in the pull request
			// body.
			name: "multiple libraries generation",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
						APIs: []*legacyconfig.API{
							{
								Path: "path/to",
							},
						},
					},
					{
						ID:          "another-library",
						SourceRoots: []string{"path/to"},
						APIs: []*legacyconfig.API{
							{
								Path: "path/to",
							},
						},
					},
				},
			},
			sourceRepo: &MockRepository{
				RemotesValue: []*legacygitrepo.Remote{{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}}},
				GetCommitByHash: map[string]*legacygitrepo.Commit{
					"1234567890": {
						Hash: plumbing.NewHash("1234567890"),
						When: time.UnixMilli(200),
					},
					"abcdefg": {
						Hash: plumbing.NewHash("abcdefg"),
						When: time.UnixMilli(300),
					},
				},
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*legacygitrepo.Commit{
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
			languageRepo: &MockRepository{
				IsCleanValue:              true,
				HeadHashValue:             "5678",
				ChangedFilesInCommitValue: []string{"path/to/a.go"},
			},
			idToCommits: map[string]string{
				"one-library":     "1234567890",
				"another-library": "abcdefg",
			},
			failedLibraries: []string{},
			want: fmt.Sprintf(`PR created by the Librarian CLI to generate Cloud Client Libraries code from protos.

BEGIN_COMMIT

BEGIN_NESTED_COMMIT
fix: a bug fix
This is another body.

PiperOrigin-RevId: 573342
Library-IDs: one-library
Source-link: [googleapis/googleapis@fedcba09](https://github.com/googleapis/googleapis/commit/fedcba09)
END_NESTED_COMMIT

END_COMMIT

This pull request is generated with proto changes between
[googleapis/googleapis@abcdef00](https://github.com/googleapis/googleapis/commit/abcdef0000000000000000000000000000000000)
(exclusive) and
[googleapis/googleapis@fedcba09](https://github.com/googleapis/googleapis/commit/fedcba0987654321000000000000000000000000)
(inclusive).

Librarian Version: %s
Language Image: %s`,
				librarianVersion, "go:1.21"),
		},
		{
			name: "group_commit_messages",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
						APIs: []*legacyconfig.API{
							{
								Path: "path/to",
							},
						},
					},
					{
						ID:          "another-library",
						SourceRoots: []string{"path/to"},
						APIs: []*legacyconfig.API{
							{
								Path: "path/to",
							},
						},
					},
				},
			},
			sourceRepo: &MockRepository{
				RemotesValue: []*legacygitrepo.Remote{{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}}},
				GetCommitByHash: map[string]*legacygitrepo.Commit{
					"1234567890": {
						Hash: plumbing.NewHash("1234567890"),
						When: time.UnixMilli(200),
					},
					"abcdefg": {
						Hash: plumbing.NewHash("abcdefg"),
						When: time.UnixMilli(300),
					},
				},
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*legacygitrepo.Commit{
					"1234567890": {
						{
							Message: "fix: a bug fix\n\nThis is another body.\n\nPiperOrigin-RevId: 573342",
							Hash:    hash2,
							When:    today.Add(time.Hour),
						},
					},
					"abcdefg": {
						{
							Message: "fix: a bug fix\n\nThis is another body.\n\nPiperOrigin-RevId: 573342",
							Hash:    hash2,
							When:    today.Add(time.Hour),
						},
					},
				},
				ChangedFilesInCommitValueByHash: map[string][]string{
					hash2.String(): {
						"path/to/file",
					},
				},
			},
			languageRepo: &MockRepository{
				IsCleanValue:              true,
				HeadHashValue:             "5678",
				ChangedFilesInCommitValue: []string{"path/to/a.go"},
			},
			idToCommits: map[string]string{
				"one-library":     "1234567890",
				"another-library": "abcdefg",
			},
			failedLibraries: []string{},
			want: fmt.Sprintf(`PR created by the Librarian CLI to generate Cloud Client Libraries code from protos.

BEGIN_COMMIT

BEGIN_NESTED_COMMIT
fix: a bug fix
This is another body.

PiperOrigin-RevId: 573342
Library-IDs: one-library,another-library
Source-link: [googleapis/googleapis@fedcba09](https://github.com/googleapis/googleapis/commit/fedcba09)
END_NESTED_COMMIT

END_COMMIT

This pull request is generated with proto changes between
[googleapis/googleapis@abcdef00](https://github.com/googleapis/googleapis/commit/abcdef0000000000000000000000000000000000)
(exclusive) and
[googleapis/googleapis@fedcba09](https://github.com/googleapis/googleapis/commit/fedcba0987654321000000000000000000000000)
(inclusive).

Librarian Version: %s
Language Image: %s`,
				librarianVersion, "go:1.21"),
		},
		{
			name: "multiple libraries generation with failed libraries",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
						APIs: []*legacyconfig.API{
							{
								Path: "path/to",
							},
						},
					},
					{
						ID:          "another-library",
						SourceRoots: []string{"path/to"},
						APIs: []*legacyconfig.API{
							{
								Path: "path/to",
							},
						},
					},
				},
			},
			sourceRepo: &MockRepository{
				RemotesValue: []*legacygitrepo.Remote{{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}}},
				GetCommitByHash: map[string]*legacygitrepo.Commit{
					"1234567890": {
						Hash: plumbing.NewHash("1234567890"),
						When: time.UnixMilli(200),
					},
					"abcdefg": {
						Hash: plumbing.NewHash("abcdefg"),
						When: time.UnixMilli(300),
					},
				},
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*legacygitrepo.Commit{
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
			languageRepo: &MockRepository{
				IsCleanValue:              true,
				HeadHashValue:             "5678",
				ChangedFilesInCommitValue: []string{"path/to/a.go"},
			},
			idToCommits: map[string]string{
				"one-library":     "1234567890",
				"another-library": "abcdefg",
			},
			failedLibraries: []string{
				"failed-library-a",
				"failed-library-b",
			},
			want: fmt.Sprintf(`PR created by the Librarian CLI to generate Cloud Client Libraries code from protos.

BEGIN_COMMIT

BEGIN_NESTED_COMMIT
fix: a bug fix
This is another body.

PiperOrigin-RevId: 573342
Library-IDs: one-library
Source-link: [googleapis/googleapis@fedcba09](https://github.com/googleapis/googleapis/commit/fedcba09)
END_NESTED_COMMIT

END_COMMIT

This pull request is generated with proto changes between
[googleapis/googleapis@abcdef00](https://github.com/googleapis/googleapis/commit/abcdef0000000000000000000000000000000000)
(exclusive) and
[googleapis/googleapis@fedcba09](https://github.com/googleapis/googleapis/commit/fedcba0987654321000000000000000000000000)
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
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
						APIs: []*legacyconfig.API{
							{
								Path: "path/to",
							},
						},
					},
				},
			},
			sourceRepo: &MockRepository{
				RemotesValue: []*legacygitrepo.Remote{{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}}},
				GetCommitByHash: map[string]*legacygitrepo.Commit{
					"1234567890": {
						Hash: plumbing.NewHash("1234567890"),
						When: time.UnixMilli(200),
					},
				},
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*legacygitrepo.Commit{
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
			languageRepo: &MockRepository{
				IsCleanValue:              true,
				HeadHashValue:             "5678",
				ChangedFilesInCommitValue: []string{"path/to/a.go"},
			},
			idToCommits: map[string]string{
				"one-library": "1234567890",
			},
			failedLibraries: []string{},
			want: fmt.Sprintf(`PR created by the Librarian CLI to generate Cloud Client Libraries code from protos.

BEGIN_COMMIT

BEGIN_NESTED_COMMIT
fix: a bug fix
This is another body.

PiperOrigin-RevId: 573342
Library-IDs: one-library
Source-link: [googleapis/googleapis@fedcba09](https://github.com/googleapis/googleapis/commit/fedcba09)
END_NESTED_COMMIT

BEGIN_NESTED_COMMIT
feat: new feature
This is body.

PiperOrigin-RevId: 98765
Library-IDs: one-library
Source-link: [googleapis/googleapis@12345678](https://github.com/googleapis/googleapis/commit/12345678)
END_NESTED_COMMIT

END_COMMIT

This pull request is generated with proto changes between
[googleapis/googleapis@12345678](https://github.com/googleapis/googleapis/commit/1234567890000000000000000000000000000000)
(exclusive) and
[googleapis/googleapis@fedcba09](https://github.com/googleapis/googleapis/commit/fedcba0987654321000000000000000000000000)
(inclusive).

Librarian Version: %s
Language Image: %s`,
				librarianVersion, "go:1.21"),
		},
		{
			name: "no conventional commit is found since last generation",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
						// Intentionally set this value to verify the test can pass.
						LastGeneratedCommit: "randomCommit",
						APIs: []*legacyconfig.API{
							{
								Path: "path/to",
							},
						},
					},
				},
			},
			sourceRepo: &MockRepository{
				RemotesValue:   []*legacygitrepo.Remote{{Name: "origin", URLs: []string{"https://github.com/owner/repo.git"}}},
				GetCommitError: errors.New("simulated get commit error"),
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*legacygitrepo.Commit{
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
			languageRepo: &MockRepository{
				IsCleanValue:              true,
				HeadHashValue:             "5678",
				ChangedFilesInCommitValue: []string{"path/to/a.go"},
			},
			idToCommits: map[string]string{
				"one-library": "1234567890",
			},
			wantErr:       true,
			wantErrPhrase: "failed to find the start commit",
		},
		{
			name: "no conventional commits since last generation",
			state: &legacyconfig.LibrarianState{
				Image:     "go:1.21",
				Libraries: []*legacyconfig.LibraryState{{ID: "one-library", SourceRoots: []string{"path/to"}}},
			},
			sourceRepo: &MockRepository{},
			languageRepo: &MockRepository{
				HeadHashValue:             "5678",
				ChangedFilesInCommitValue: []string{"path/to/a.go"},
			},
			idToCommits: map[string]string{
				"one-library": "",
			},
			want: "No commit is found since last generation",
		},
		{
			name: "failed to get language repo changes commits",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
					},
				},
			},
			sourceRepo: &MockRepository{},
			languageRepo: &MockRepository{
				IsCleanError: errors.New("simulated error"),
			},
			idToCommits: map[string]string{
				"one-library": "1234567890",
			},
			wantErr:       true,
			wantErrPhrase: "failed to fetch changes in language repo",
		},
		{
			name: "failed to get conventional commits",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
					},
				},
			},
			sourceRepo: &MockRepository{
				GetCommitsForPathsSinceLastGenError: errors.New("simulated error"),
			},
			languageRepo: &MockRepository{
				IsCleanValue:              true,
				HeadHashValue:             "5678",
				ChangedFilesInCommitValue: []string{"path/to/a.go"},
			},
			idToCommits: map[string]string{
				"one-library": "1234567890",
			},
			wantErr:       true,
			wantErrPhrase: "failed to fetch conventional commits for library",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := &generationPRRequest{
				sourceRepo:      test.sourceRepo,
				languageRepo:    test.languageRepo,
				state:           test.state,
				idToCommits:     test.idToCommits,
				failedLibraries: test.failedLibraries,
			}
			got, err := formatGenerationPRBody(req)
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

func TestFormatOnboardPRBody(t *testing.T) {
	t.Parallel()
	librarianVersion := legacycli.Version()

	for _, test := range []struct {
		name          string
		state         *legacyconfig.LibrarianState
		sourceRepo    legacygitrepo.Repository
		api           string
		library       string
		want          string
		wantErr       bool
		wantErrPhrase string
	}{
		{
			name: "onboarding_new_api",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
						APIs: []*legacyconfig.API{
							{
								Path:          "path/to",
								ServiceConfig: "library_v1.yaml",
							},
						},
					},
				},
			},
			sourceRepo: &MockRepository{
				GetLatestCommitByPath: map[string]*legacygitrepo.Commit{
					"path/to/library_v1.yaml": {
						Message: "feat: new feature\n\nThis is body.\n\nPiperOrigin-RevId: 98765",
					},
				},
			},
			api:     "path/to",
			library: "one-library",
			want: fmt.Sprintf(`PR created by the Librarian CLI to onboard a new Cloud Client Library.

BEGIN_COMMIT

feat: onboard a new library

PiperOrigin-RevId: 98765
Library-IDs: one-library

END_COMMIT

Librarian Version: %s
Language Image: %s`,
				librarianVersion, "go:1.21"),
		},
		{
			name: "no_latest_commit_during_api_onboarding",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
						APIs: []*legacyconfig.API{
							{
								Path:          "path/to",
								ServiceConfig: "library_v1.yaml",
							},
						},
					},
				},
			},
			sourceRepo: &MockRepository{
				GetLatestCommitError: errors.New("no latest commit"),
			},
			api:           "path/to",
			library:       "one-library",
			wantErr:       true,
			wantErrPhrase: "no latest commit",
		},
		{
			name: "latest_commit_does_not_contain_piper_during_api_onboarding",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID:          "one-library",
						SourceRoots: []string{"path/to"},
						APIs: []*legacyconfig.API{
							{
								Path:          "path/to",
								ServiceConfig: "library_v1.yaml",
							},
						},
					},
				},
			},
			sourceRepo: &MockRepository{
				GetLatestCommitByPath: map[string]*legacygitrepo.Commit{
					"path/to/library_v1.yaml": {
						Message: "feat: new feature\n\nThis is body.",
					},
				},
			},
			api:           "path/to",
			library:       "one-library",
			wantErr:       true,
			wantErrPhrase: errPiperNotFound.Error(),
		},
		{
			name: "library_not_found_in_state",
			state: &legacyconfig.LibrarianState{
				Image: "go:1.21",
				Libraries: []*legacyconfig.LibraryState{
					{
						ID: "one-library",
					},
				},
			},
			library:       "another-library",
			wantErr:       true,
			wantErrPhrase: "library another-library not found",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := &onboardPRRequest{
				sourceRepo: test.sourceRepo,
				state:      test.state,
				api:        test.api,
				library:    test.library,
			}
			got, err := formatOnboardPRBody(req)
			if test.wantErr {
				if err == nil {
					t.Fatalf("%s should return error", test.name)
				}
				if !strings.Contains(err.Error(), test.wantErrPhrase) {
					t.Errorf("formatOnboardPRBody() returned error %q, want to contain %q", err.Error(), test.wantErrPhrase)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("formatOnboardPRBody() mismatch (-want +got):\n%s", diff)
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
		state         *legacyconfig.LibrarianState
		repo          legacygitrepo.Repository
		idToCommits   map[string]string
		want          *legacygitrepo.Commit
		wantErr       bool
		wantErrPhrase string
	}{
		{
			name: "find the last generated commit",
			state: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
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
				GetCommitByHash: map[string]*legacygitrepo.Commit{
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
			want: &legacygitrepo.Commit{
				Hash:    hash2,
				Message: "this is another message",
				When:    today.Add(2 * time.Hour).Add(time.Minute),
			},
		},
		{
			name: "failed to find last generated commit",
			state: &legacyconfig.LibrarianState{
				Libraries: []*legacyconfig.LibraryState{
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
func TestGroupByPiperID(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		commits []*legacygitrepo.ConventionalCommit
		want    []*legacygitrepo.ConventionalCommit
	}{
		{
			name: "group_commits_with_same_piper_id_and_subject",
			commits: []*legacygitrepo.ConventionalCommit{
				{
					LibraryID: "library-1",
					Subject:   "one subject",
					Footers: map[string]string{
						"PiperOrigin-RevId": "123456",
					},
				},
				{
					LibraryID: "library-2",
					Subject:   "a different subject",
					Footers: map[string]string{
						"PiperOrigin-RevId": "123456",
					},
				},
				{
					LibraryID: "library-3",
					Subject:   "the same subject",
					Footers: map[string]string{
						"PiperOrigin-RevId": "987654",
					},
				},
				{
					LibraryID: "library-4",
					Subject:   "the same subject",
					Footers: map[string]string{
						"PiperOrigin-RevId": "987654",
					},
				},
				{
					LibraryID: "library-5",
				},
				{
					LibraryID: "library-6",
					Footers: map[string]string{
						"random-key": "random-value",
					},
				},
			},
			want: []*legacygitrepo.ConventionalCommit{
				{
					LibraryID: "library-1",
					Subject:   "one subject",
					Footers: map[string]string{
						"PiperOrigin-RevId": "123456",
						"Library-IDs":       "library-1",
					},
				},
				{
					LibraryID: "library-2",
					Subject:   "a different subject",
					Footers: map[string]string{
						"PiperOrigin-RevId": "123456",
						"Library-IDs":       "library-2",
					},
				},
				{
					LibraryID: "library-3",
					Subject:   "the same subject",
					Footers: map[string]string{
						"PiperOrigin-RevId": "987654",
						"Library-IDs":       "library-3,library-4",
					},
				},
				{
					LibraryID: "library-5",
					Footers: map[string]string{
						"Library-IDs": "library-5",
					},
				},
				{
					LibraryID: "library-6",
					Footers: map[string]string{
						"random-key":  "random-value",
						"Library-IDs": "library-6",
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := groupByIDAndSubject(test.commits)
			// We don't care the order in the slice but sorting makes the test deterministic.
			opts := cmpopts.SortSlices(func(a, b *legacygitrepo.ConventionalCommit) bool {
				return a.LibraryID < b.LibraryID
			})
			if diff := cmp.Diff(test.want, got, opts); diff != "" {
				t.Errorf("groupByIDAndSubject() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
