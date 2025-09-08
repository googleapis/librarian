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

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/conventionalcommits"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/semver"
)

func TestShouldExclude(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name         string
		files        []string
		excludePaths []string
		want         bool
	}{
		{
			name:         "no exclude paths",
			files:        []string{"a/b/c.go"},
			excludePaths: []string{},
			want:         false,
		},
		{
			name:         "file in exclude path",
			files:        []string{"a/b/c.go"},
			excludePaths: []string{"a/b"},
			want:         true,
		},
		{
			name:         "file not in exclude path",
			files:        []string{"a/b/c.go"},
			excludePaths: []string{"d/e"},
			want:         false,
		},
		{
			name:         "one file in exclude path, one not",
			files:        []string{"a/b/c.go", "d/e/f.go"},
			excludePaths: []string{"a/b"},
			want:         false,
		},
		{
			name:         "all files in exclude paths",
			files:        []string{"a/b/c.go", "d/e/f.go"},
			excludePaths: []string{"a/b", "d/e"},
			want:         true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := shouldExclude(test.files, test.excludePaths)
			if got != test.want {
				t.Errorf("shouldExclude(%v, %v) = %v, want %v", test.files, test.excludePaths, got, test.want)
			}
		})
	}
}

func TestFormatTag(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		library *config.LibraryState
		want    string
	}{
		{
			name: "default format",
			library: &config.LibraryState{
				ID:      "google.cloud.foo.v1",
				Version: "1.2.3",
			},
			want: "google.cloud.foo.v1-1.2.3",
		},
		{
			name: "custom format",
			library: &config.LibraryState{
				ID:        "google.cloud.foo.v1",
				Version:   "1.2.3",
				TagFormat: "v{version}-{id}",
			},
			want: "v1.2.3-google.cloud.foo.v1",
		},
		{
			name: "custom format -- version only",
			library: &config.LibraryState{
				ID:        "google.cloud.foo.v1",
				Version:   "1.2.3",
				TagFormat: "v{version}",
			},
			want: "v1.2.3",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatTag(test.library, "")
			if got != test.want {
				t.Errorf("formatTag() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGetConventionalCommitsSinceLastRelease(t *testing.T) {
	t.Parallel()
	pathAndMessages := []pathAndMessage{
		{
			path:    "foo/a.txt",
			message: "feat(foo): initial commit for foo",
		},
		{
			path:    "bar/a.txt",
			message: "feat(bar): initial commit for bar",
		},
		{
			path:    "foo/b.txt",
			message: "fix(foo): a fix for foo",
		},
		{
			path:    "foo/README.md",
			message: "docs(foo): update README",
		},
		{
			path:    "foo/c.txt",
			message: "feat(foo): another feature for foo",
		},
	}
	repoWithCommits := setupRepoForGetCommits(t, pathAndMessages, []string{"foo-v1.0.0"})
	for _, test := range []struct {
		name          string
		repo          gitrepo.Repository
		library       *config.LibraryState
		want          []*conventionalcommits.ConventionalCommit
		wantErr       bool
		wantErrPhrase string
	}{
		{
			name: "get commits for foo",
			repo: repoWithCommits,
			library: &config.LibraryState{
				ID:                  "foo",
				Version:             "1.0.0",
				TagFormat:           "{id}-v{version}",
				SourceRoots:         []string{"foo"},
				ReleaseExcludePaths: []string{"foo/README.md"},
			},
			want: []*conventionalcommits.ConventionalCommit{
				{
					Type:        "feat",
					Scope:       "foo",
					Description: "another feature for foo",
					LibraryID:   "foo",
					Footers:     make(map[string]string),
				},
				{
					Type:        "fix",
					Scope:       "foo",
					Description: "a fix for foo",
					LibraryID:   "foo",
					Footers:     make(map[string]string),
				},
			},
		},
		{
			name: "GetCommitsForPathsSinceTag error",
			repo: &MockRepository{
				GetCommitsForPathsSinceTagError: fmt.Errorf("mock error from GetCommitsForPathsSinceTagError"),
			},
			library:       &config.LibraryState{ID: "foo"},
			wantErr:       true,
			wantErrPhrase: "mock error from GetCommitsForPathsSinceTagError",
		},
		{
			name: "ChangedFilesInCommit error",
			repo: &MockRepository{
				GetCommitsForPathsSinceTagValue: []*gitrepo.Commit{
					{Message: "feat(foo): a feature"},
				},
				ChangedFilesInCommitError: fmt.Errorf("mock error from ChangedFilesInCommit"),
			},
			library:       &config.LibraryState{ID: "foo"},
			wantErr:       true,
			wantErrPhrase: "mock error from ChangedFilesInCommit",
		},
		{
			name: "ParseCommit error",
			repo: &MockRepository{
				GetCommitsForPathsSinceTagValue: []*gitrepo.Commit{
					{Message: ""},
				},
				ChangedFilesInCommitValue: []string{"foo/a.txt"},
			},
			library:       &config.LibraryState{ID: "foo"},
			wantErr:       true,
			wantErrPhrase: "failed to parse commit",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := GetConventionalCommitsSinceLastRelease(test.repo, test.library)
			if test.wantErr {
				if err == nil {
					t.Fatal("GetConventionalCommitsSinceLastRelease() should have failed")
				}
				if !strings.Contains(err.Error(), test.wantErrPhrase) {
					t.Errorf("GetConventionalCommitsSinceLastRelease() returned error %q, want to contain %q", err.Error(), test.wantErrPhrase)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetConventionalCommitsSinceLastRelease() failed: %v", err)
			}
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(conventionalcommits.ConventionalCommit{}, "SHA", "Body", "IsBreaking", "When")); diff != "" {
				t.Errorf("GetConventionalCommitsSinceLastRelease() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetHighestChange(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name           string
		commits        []*conventionalcommits.ConventionalCommit
		expectedChange semver.ChangeLevel
	}{
		{
			name: "major change",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "feat", IsBreaking: true},
				{Type: "feat"},
				{Type: "fix"},
			},
			expectedChange: semver.Major,
		},
		{
			name: "minor change",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "feat"},
				{Type: "fix"},
			},
			expectedChange: semver.Minor,
		},
		{
			name: "patch change",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "fix"},
			},
			expectedChange: semver.Patch,
		},
		{
			name: "no change",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "docs"},
				{Type: "chore"},
			},
			expectedChange: semver.None,
		},
		{
			name:           "no commits",
			commits:        []*conventionalcommits.ConventionalCommit{},
			expectedChange: semver.None,
		},
		{
			name: "nested commit forces minor bump",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "fix"},
				{Type: "feat", IsNested: true},
			},
			expectedChange: semver.Minor,
		},
		{
			name: "nested commit with breaking change forces minor bump",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "feat", IsBreaking: true, IsNested: true},
				{Type: "feat"},
			},
			expectedChange: semver.Minor,
		},
		{
			name: "major change and nested commit",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "feat", IsBreaking: true},
				{Type: "fix", IsNested: true},
			},
			expectedChange: semver.Major,
		},
		{
			name: "nested commit before major change",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "fix", IsNested: true},
				{Type: "feat", IsBreaking: true},
			},
			expectedChange: semver.Major,
		},
		{
			name: "nested commit with only fixes forces minor bump",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "fix"},
				{Type: "fix", IsNested: true},
			},
			expectedChange: semver.Minor,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			highestChange := getHighestChange(test.commits)
			if diff := cmp.Diff(test.expectedChange, highestChange); diff != "" {
				t.Errorf("getHighestChange() returned diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNextVersion(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name                string
		commits             []*conventionalcommits.ConventionalCommit
		currentVersion      string
		overrideNextVersion string
		wantVersion         string
		wantErr             bool
	}{
		{
			name:                "with override version",
			commits:             []*conventionalcommits.ConventionalCommit{},
			currentVersion:      "1.0.0",
			overrideNextVersion: "2.0.0",
			wantVersion:         "2.0.0",
			wantErr:             false,
		},
		{
			name: "without override version",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "feat"},
			},
			currentVersion:      "1.0.0",
			overrideNextVersion: "",
			wantVersion:         "1.1.0",
			wantErr:             false,
		},
		{
			name: "derive next returns error",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "feat"},
			},
			currentVersion:      "invalid-version",
			overrideNextVersion: "",
			wantVersion:         "",
			wantErr:             true,
		},
		{
			name: "breaking change on nested commit results in minor bump",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "feat", IsBreaking: true, IsNested: true},
			},
			currentVersion:      "1.2.3",
			overrideNextVersion: "",
			wantVersion:         "1.3.0",
			wantErr:             false,
		},
		{
			name: "major change before nested commit results in major bump",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "feat", IsBreaking: true},
				{Type: "fix", IsNested: true},
			},
			currentVersion:      "1.2.3",
			overrideNextVersion: "",
			wantVersion:         "2.0.0",
			wantErr:             false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotVersion, err := NextVersion(test.commits, test.currentVersion, test.overrideNextVersion)
			if (err != nil) != test.wantErr {
				t.Errorf("NextVersion() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if gotVersion != test.wantVersion {
				t.Errorf("NextVersion() = %v, want %v", gotVersion, test.wantVersion)
			}
		})
	}
}
