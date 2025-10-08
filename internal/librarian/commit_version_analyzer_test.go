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

func TestShouldIncludeForRelease(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name         string
		files        []string
		sourceRoots  []string
		excludePaths []string
		want         bool
	}{
		{
			name:         "file in source root, not excluded",
			files:        []string{"a/b/c.go"},
			sourceRoots:  []string{"a"},
			excludePaths: []string{},
			want:         true,
		},
		{
			name:         "file in source root, and excluded",
			files:        []string{"a/b/c.go"},
			sourceRoots:  []string{"a"},
			excludePaths: []string{"a/b"},
		},
		{
			name:         "file not in source root",
			files:        []string{"x/y/z.go"},
			sourceRoots:  []string{"a"},
			excludePaths: []string{},
		},
		{
			name:         "one file included, one file not in source root",
			files:        []string{"a/b/c.go", "x/y/z.go"},
			sourceRoots:  []string{"a"},
			excludePaths: []string{},
			want:         true,
		},
		{
			name:         "one file included, one file excluded",
			files:        []string{"a/b/c.go", "a/d/e.go"},
			sourceRoots:  []string{"a"},
			excludePaths: []string{"a/d"},
			want:         true,
		},
		{
			name:         "all files excluded",
			files:        []string{"a/b/c.go", "a/d/e.go"},
			sourceRoots:  []string{"a"},
			excludePaths: []string{"a/b", "a/d"},
		},
		{
			name:         "all files not in source root",
			files:        []string{"x/y/c.go", "w/z/e.go"},
			sourceRoots:  []string{"a"},
			excludePaths: []string{},
		},
		{
			name:         "a file not in source root and a file in exclude path",
			files:        []string{"a/b/c.go", "w/z/e.go"},
			sourceRoots:  []string{"a"},
			excludePaths: []string{"a/b"},
		},
		{
			name:         "a file in source root and not in exclude path, one file in exclude path and a file outside of source",
			files:        []string{"a/d/c.go", "a/b/c.go", "w/z/e.go"},
			sourceRoots:  []string{"a"},
			excludePaths: []string{"a/b"},
			want:         true,
		},
		{
			name:         "no source roots",
			files:        []string{"a/b/c.go"},
			sourceRoots:  []string{},
			excludePaths: []string{},
		},
		{
			name:         "source root as prefix of another source root",
			files:        []string{"aiplatform/file.go"},
			sourceRoots:  []string{"ai"},
			excludePaths: []string{},
		},
		{
			name:         "excluded path is a directory",
			files:        []string{"foo/bar/baz.go"},
			sourceRoots:  []string{"foo"},
			excludePaths: []string{"foo/bar"},
		},
		{
			name:         "excluded path is a file, file matching it",
			files:        []string{"foo/bar/go.mod"},
			sourceRoots:  []string{"foo"},
			excludePaths: []string{"foo/bar/go.mod"},
		},
		{
			name:         "excluded path is a file, file does not match it",
			files:        []string{"foo/go.mod"},
			sourceRoots:  []string{"foo"},
			excludePaths: []string{"foo/bar/go.mod"},
			want:         true,
		},
		{
			name:         "excluded path is a file with similar name",
			files:        []string{"foo/bar/go.mod.bak"},
			sourceRoots:  []string{"foo"},
			excludePaths: []string{"foo/bar/go.mod"},
			want:         true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := shouldIncludeForRelease(test.files, test.sourceRoots, test.excludePaths)
			if got != test.want {
				t.Errorf("shouldIncludeForRelease(%v, %v, %v) = %v, want %v", test.files, test.sourceRoots, test.excludePaths, got, test.want)
			}
		})
	}
}

func TestShouldIncludeForGeneration(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		files    []string
		apiPaths []string
		want     bool
	}{
		{
			name:     "all_files_in_apiPaths",
			files:    []string{"a/b/c.proto"},
			apiPaths: []string{"a"},
			want:     true,
		},
		{
			name:     "some_files_in_apiPaths",
			files:    []string{"a/b/c.proto", "e/f/g.proto"},
			apiPaths: []string{"a"},
			want:     true,
		},
		{
			name:     "no_files_in_apiPaths",
			files:    []string{"a/b/c.proto"},
			apiPaths: []string{"b"},
			want:     false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := shouldIncludeForGeneration(test.files, test.apiPaths)
			if got != test.want {
				t.Errorf("shouldIncludeForGeneration(%v, %v) = %v, want %v", test.files, test.apiPaths, got, test.want)
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
			name: "found_matching_commits_for_foo",
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
					Type:      "feat",
					Scope:     "foo",
					Subject:   "another feature for foo",
					LibraryID: "foo",
					Footers:   make(map[string]string),
				},
				{
					Type:      "fix",
					Scope:     "foo",
					Subject:   "a fix for foo",
					LibraryID: "foo",
					Footers:   make(map[string]string),
				},
			},
		},
		{
			name: "no_matching_commits_for_foo",
			repo: repoWithCommits,
			library: &config.LibraryState{
				ID:          "foo",
				Version:     "1.0.0",
				TagFormat:   "{id}-v{version}",
				SourceRoots: []string{"no_matching_dir"},
			},
		},
		{
			name: "apiPaths_has_no_impact_on_release",
			repo: repoWithCommits,
			library: &config.LibraryState{
				ID:          "foo",
				Version:     "1.0.0",
				TagFormat:   "{id}-v{version}",
				SourceRoots: []string{"no_matching_dir"}, // For release, only this is considered
				APIs: []*config.API{
					{
						Path: "foo",
					},
					{
						Path: "bar",
					},
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
			library:       &config.LibraryState{ID: "foo", SourceRoots: []string{"foo"}},
			wantErr:       true,
			wantErrPhrase: "failed to parse commit",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := getConventionalCommitsSinceLastRelease(test.repo, test.library, "")
			if test.wantErr {
				if err == nil {
					t.Fatal("getConventionalCommitsSinceLastRelease() should have failed")
				}
				if !strings.Contains(err.Error(), test.wantErrPhrase) {
					t.Errorf("getConventionalCommitsSinceLastRelease() returned error %q, want to contain %q", err.Error(), test.wantErrPhrase)
				}
				return
			}
			if err != nil {
				t.Fatalf("getConventionalCommitsSinceLastRelease() failed: %v", err)
			}
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(conventionalcommits.ConventionalCommit{}, "SHA", "CommitHash", "Body", "IsBreaking", "When")); diff != "" {
				t.Errorf("getConventionalCommitsSinceLastRelease() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetConventionalCommitsSinceLastGeneration(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		repo          gitrepo.Repository
		library       *config.LibraryState
		want          []*conventionalcommits.ConventionalCommit
		wantErr       bool
		wantErrPhrase string
	}{
		{
			name: "found_matching_file_changes_for_foo",
			library: &config.LibraryState{
				ID: "foo",
				APIs: []*config.API{
					{
						Path: "foo",
					},
				},
			},
			repo: &MockRepository{
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*gitrepo.Commit{
					"1234": {
						{Message: "feat(foo): a feature"},
					},
				},
				ChangedFilesInCommitValue: []string{"foo/a.proto"},
			},
			want: []*conventionalcommits.ConventionalCommit{
				{
					Type:      "feat",
					Scope:     "foo",
					Subject:   "a feature",
					LibraryID: "foo",
					Footers:   map[string]string{},
				},
			},
		},
		{
			name: "no_matching_file_changes_for_foo",
			library: &config.LibraryState{
				ID: "foo",
				APIs: []*config.API{
					{
						Path: "foo",
					},
				},
			},
			repo: &MockRepository{
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*gitrepo.Commit{
					"1234": {
						{Message: "feat(baz): a feature"},
					},
				},
				ChangedFilesInCommitValue: []string{"baz/a.proto", "baz/b.proto", "bar/a.proto"}, // file changed is not in foo/*
			},
		},
		{
			name: "sources_root_has_no_impact",
			library: &config.LibraryState{
				ID: "foo",
				APIs: []*config.API{
					{
						Path: "foo", // For generation, only this is considered
					},
				},
				SourceRoots: []string{
					"baz/",
					"bar/",
				},
			},
			repo: &MockRepository{
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*gitrepo.Commit{
					"1234": {
						{Message: "feat(baz): a feature"},
					},
				},
				ChangedFilesInCommitValue: []string{"baz/a.proto", "baz/b.proto", "bar/a.proto"}, // file changed is not in foo/*
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := getConventionalCommitsSinceLastGeneration(test.repo, test.library, "1234")
			if test.wantErr {
				if err == nil {
					t.Fatal("getConventionalCommitsSinceLastGeneration() should have failed")
				}
				if !strings.Contains(err.Error(), test.wantErrPhrase) {
					t.Errorf("getConventionalCommitsSinceLastRelease() returned error %q, want to contain %q", err.Error(), test.wantErrPhrase)
				}
				return
			}
			if err != nil {
				t.Fatalf("getConventionalCommitsSinceLastRelease() failed: %v", err)
			}
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(conventionalcommits.ConventionalCommit{}, "SHA", "CommitHash", "Body", "IsBreaking", "When")); diff != "" {
				t.Errorf("getConventionalCommitsSinceLastRelease() mismatch (-want +got):\n%s", diff)
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
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNextVersion(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name           string
		commits        []*conventionalcommits.ConventionalCommit
		currentVersion string
		wantVersion    string
		wantErr        bool
	}{
		{
			name: "without override version",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "feat"},
			},
			currentVersion: "1.0.0",
			wantVersion:    "1.1.0",
		},
		{
			name: "derive next returns error",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "feat"},
			},
			currentVersion: "invalid-version",
			wantVersion:    "",
			wantErr:        true,
		},
		{
			name: "breaking change on nested commit results in minor bump",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "feat", IsBreaking: true, IsNested: true},
			},
			currentVersion: "1.2.3",
			wantVersion:    "1.3.0",
		},
		{
			name: "major change before nested commit results in major bump",
			commits: []*conventionalcommits.ConventionalCommit{
				{Type: "feat", IsBreaking: true},
				{Type: "fix", IsNested: true},
			},
			currentVersion: "1.2.3",
			wantVersion:    "2.0.0",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotVersion, err := NextVersion(test.commits, test.currentVersion)
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
