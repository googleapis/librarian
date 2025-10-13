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
	cases := []struct {
		name              string
		sourceFiles       []string
		languageRepoFiles []string
		library           *config.LibraryState
		want              bool
	}{
		{
			name:              "include: source and language files in path",
			sourceFiles:       []string{"google/cloud/aiplatform/v1/featurestore_service.proto"},
			languageRepoFiles: []string{"google/cloud/aiplatform/featurestore_client.go"},
			library: &config.LibraryState{
				APIs:        []*config.API{{Path: "google/cloud/aiplatform/v1"}},
				SourceRoots: []string{"google/cloud/aiplatform"},
			},
			want: true,
		},
		{
			name:              "exclude: no source files in path",
			sourceFiles:       []string{"google/cloud/vision/v1/image_annotator.proto"},
			languageRepoFiles: []string{"google/cloud/aiplatform/featurestore_client.go"},
			library: &config.LibraryState{
				APIs:        []*config.API{{Path: "google/cloud/aiplatform/v1"}},
				SourceRoots: []string{"google/cloud/aiplatform"},
			},
			want: false,
		},
		{
			name:              "exclude: no language repo files in path",
			sourceFiles:       []string{"google/cloud/aiplatform/v1/featurestore_service.proto"},
			languageRepoFiles: []string{"google/cloud/vision/image_annotator_client.go"},
			library: &config.LibraryState{
				APIs:        []*config.API{{Path: "google/cloud/aiplatform/v1"}},
				SourceRoots: []string{"google/cloud/aiplatform"},
			},
			want: false,
		},
		{
			name:              "exclude: language repo file in excluded path",
			sourceFiles:       []string{"google/cloud/aiplatform/v1/featurestore_service.proto"},
			languageRepoFiles: []string{"google/cloud/aiplatform/internal/generated/featurestore_client.go"},
			library: &config.LibraryState{
				APIs:                []*config.API{{Path: "google/cloud/aiplatform/v1"}},
				SourceRoots:         []string{"google/cloud/aiplatform"},
				ReleaseExcludePaths: []string{"google/cloud/aiplatform/internal"},
			},
			want: false,
		},
		{
			name:              "include: multiple files, some matching",
			sourceFiles:       []string{"google/cloud/aiplatform/v1/featurestore_service.proto", "unrelated/file"},
			languageRepoFiles: []string{"google/cloud/aiplatform/featurestore_client.go", "unrelated/file"},
			library: &config.LibraryState{
				APIs:        []*config.API{{Path: "google/cloud/aiplatform/v1"}},
				SourceRoots: []string{"google/cloud/aiplatform"},
			},
			want: true,
		},
		{
			name:              "include: multiple source roots and APIs",
			sourceFiles:       []string{"google/cloud/vision/v1/image_annotator.proto"},
			languageRepoFiles: []string{"google/cloud/aiplatform/featurestore_client.go"},
			library: &config.LibraryState{
				APIs:        []*config.API{{Path: "google/cloud/aiplatform/v1"}, {Path: "google/cloud/vision/v1"}},
				SourceRoots: []string{"google/cloud/aiplatform", "google/cloud/vision"},
			},
			want: true,
		},
		{
			name:              "include: source root all files",
			sourceFiles:       []string{"google/cloud/vision/v1/image_annotator.proto"},
			languageRepoFiles: []string{"google/cloud/vision/v1/image_annotator.go"},
			library: &config.LibraryState{
				APIs:        []*config.API{{Path: "google/cloud/vision/v1"}},
				SourceRoots: []string{"."},
			},
			want: true,
		},
		{
			name:              "exclude: empty source files",
			languageRepoFiles: []string{"google/cloud/aiplatform/featurestore_client.go"},
			library: &config.LibraryState{
				APIs:        []*config.API{{Path: "google/cloud/aiplatform/v1"}},
				SourceRoots: []string{"google/cloud/aiplatform"},
			},
			want: false,
		},
		{
			name:        "exclude: empty language repo files",
			sourceFiles: []string{"google/cloud/aiplatform/v1/featurestore_service.proto"},
			library: &config.LibraryState{
				APIs:        []*config.API{{Path: "google/cloud/aiplatform/v1"}},
				SourceRoots: []string{"google/cloud/aiplatform"},
			},
			want: false,
		},
		{
			name: "exclude: both file lists empty",
			library: &config.LibraryState{
				APIs:        []*config.API{{Path: "google/cloud/aiplatform/v1"}},
				SourceRoots: []string{"google/cloud/aiplatform"},
			},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := shouldIncludeForGeneration(tc.sourceFiles, tc.languageRepoFiles, tc.library)
			if got != tc.want {
				t.Errorf("shouldIncludeForGeneration() = %v, want %v", got, tc.want)
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
		{
			path: "foo/something.txt",
			message: `BEGIN_COMMIT_OVERRIDE 

BEGIN_NESTED_COMMIT
fix: a bug1 fix

This is another body.

PiperOrigin-RevId: 573342
Library-IDs: foo
Source-link: [googleapis/googleapis@fedcba0](https://github.com/googleapis/googleapis/commit/fedcba0)
END_NESTED_COMMIT

BEGIN_NESTED_COMMIT
fix: a bug2 fix

This is another body.

PiperOrigin-RevId: 573342
Library-IDs: bar
Source-link: [googleapis/googleapis@fedcba0](https://github.com/googleapis/googleapis/commit/fedcba0)
END_NESTED_COMMIT

BEGIN_NESTED_COMMIT
fix: a bug3 fix

This is another body.

PiperOrigin-RevId: 573342
Library-IDs: foo, bar
Source-link: [googleapis/googleapis@fedcba0](https://github.com/googleapis/googleapis/commit/fedcba0)
END_NESTED_COMMIT

END_COMMIT_OVERRIDE`,
		},
	}

	repoWithCommits := setupRepoForGetCommits(t, pathAndMessages, []string{"foo-v1.0.0"})

	for _, test := range []struct {
		name          string
		repo          gitrepo.Repository
		library       *config.LibraryState
		want          []*gitrepo.ConventionalCommit
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
			want: []*gitrepo.ConventionalCommit{
				{
					Type:      "fix",
					Subject:   "a bug1 fix",
					LibraryID: "foo",
					Footers: map[string]string{
						"PiperOrigin-RevId": "573342",
						"Library-IDs":       "foo",
						"Source-link":       "[googleapis/googleapis@fedcba0](https://github.com/googleapis/googleapis/commit/fedcba0)",
					},
					IsNested: true,
				},
				{
					Type:      "fix",
					Subject:   "a bug3 fix",
					LibraryID: "foo",
					Footers: map[string]string{
						"PiperOrigin-RevId": "573342",
						"Library-IDs":       "foo, bar",
						"Source-link":       "[googleapis/googleapis@fedcba0](https://github.com/googleapis/googleapis/commit/fedcba0)",
					},
					IsNested: true,
				},
				{
					Type:      "feat",
					Subject:   "another feature for foo",
					LibraryID: "foo",
					Footers:   make(map[string]string),
				},
				{
					Type:      "fix",
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
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(gitrepo.ConventionalCommit{}, "CommitHash", "Body", "IsBreaking", "When")); diff != "" {
				t.Errorf("getConventionalCommitsSinceLastRelease() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetConventionalCommitsSinceLastGeneration(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name           string
		sourceRepo     gitrepo.Repository
		languageRepo   gitrepo.Repository
		library        *config.LibraryState
		want           []*gitrepo.ConventionalCommit
		wantErr        bool
		wantErrMessage string
	}{
		{
			name: "found_matching_file_changes_for_foo",
			library: &config.LibraryState{
				ID:          "foo",
				SourceRoots: []string{"foo"},
				APIs: []*config.API{
					{
						Path: "foo",
					},
				},
			},
			sourceRepo: &MockRepository{
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*gitrepo.Commit{
					"1234": {
						{Message: "feat(foo): a feature"},
					},
				},
				ChangedFilesInCommitValue: []string{"foo/a.proto"},
			},
			languageRepo: &MockRepository{
				IsCleanValue:              true,
				HeadHashValue:             "5678",
				ChangedFilesInCommitValue: []string{"foo/a.go"},
			},
			want: []*gitrepo.ConventionalCommit{
				{
					Type:      "feat",
					Subject:   "a feature",
					LibraryID: "foo",
					Footers:   map[string]string{},
				},
			},
		},
		{
			name: "found_matching_file_changes_for_foo_with_unclean_repo",
			library: &config.LibraryState{
				ID:          "foo",
				SourceRoots: []string{"foo"},
				APIs: []*config.API{
					{
						Path: "foo",
					},
				},
			},
			sourceRepo: &MockRepository{
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*gitrepo.Commit{
					"1234": {
						{Message: "feat(foo): a feature"},
					},
				},
				ChangedFilesInCommitValue: []string{"foo/a.proto"},
			},
			languageRepo: &MockRepository{
				IsCleanValue:      false,
				ChangedFilesValue: []string{"foo/a.go"},
			},
			want: []*gitrepo.ConventionalCommit{
				{
					Type:      "feat",
					Subject:   "a feature",
					LibraryID: "foo",
					Footers:   map[string]string{},
				},
			},
		},
		{
			name: "no_matching_file_changes_for_foo",
			library: &config.LibraryState{
				ID:          "foo",
				SourceRoots: []string{"foo"},
				APIs: []*config.API{
					{
						Path: "foo",
					},
				},
			},
			sourceRepo: &MockRepository{
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*gitrepo.Commit{
					"1234": {
						{Message: "feat(baz): a feature"},
					},
				},
				ChangedFilesInCommitValue: []string{"baz/a.proto", "baz/b.proto", "bar/a.proto"}, // file changed is not in foo/*
			},
			languageRepo: &MockRepository{
				HeadHashValue:             "5678",
				ChangedFilesInCommitValue: []string{"foo/a.go"},
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
			sourceRepo: &MockRepository{
				GetCommitsForPathsSinceLastGenByCommit: map[string][]*gitrepo.Commit{
					"1234": {
						{Message: "feat(baz): a feature"},
					},
				},
				ChangedFilesInCommitValue: []string{"baz/a.proto", "baz/b.proto", "bar/a.proto"}, // file changed is not in foo/*
			},
			languageRepo: &MockRepository{
				HeadHashValue:             "5678",
				ChangedFilesInCommitValue: []string{"foo/a.go"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := getConventionalCommitsSinceLastGeneration(test.sourceRepo, test.languageRepo, test.library, "1234")
			if test.wantErr {
				if err == nil {
					t.Fatal("getConventionalCommitsSinceLastGeneration() should have failed")
				}
				if !strings.Contains(err.Error(), test.wantErrMessage) {
					t.Errorf("getConventionalCommitsSinceLastRelease() returned error %q, want to contain %q", err.Error(), test.wantErrMessage)
				}
				return
			}
			if err != nil {
				t.Fatalf("getConventionalCommitsSinceLastRelease() failed: %v", err)
			}
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(gitrepo.ConventionalCommit{}, "CommitHash", "Body", "IsBreaking", "When")); diff != "" {
				t.Errorf("getConventionalCommitsSinceLastRelease() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetHighestChange(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name           string
		commits        []*gitrepo.ConventionalCommit
		expectedChange semver.ChangeLevel
	}{
		{
			name: "major change",
			commits: []*gitrepo.ConventionalCommit{
				{Type: "feat", IsBreaking: true},
				{Type: "feat"},
				{Type: "fix"},
			},
			expectedChange: semver.Major,
		},
		{
			name: "minor change",
			commits: []*gitrepo.ConventionalCommit{
				{Type: "feat"},
				{Type: "fix"},
			},
			expectedChange: semver.Minor,
		},
		{
			name: "patch change",
			commits: []*gitrepo.ConventionalCommit{
				{Type: "fix"},
			},
			expectedChange: semver.Patch,
		},
		{
			name: "no change",
			commits: []*gitrepo.ConventionalCommit{
				{Type: "docs"},
				{Type: "chore"},
			},
			expectedChange: semver.None,
		},
		{
			name:           "no commits",
			commits:        []*gitrepo.ConventionalCommit{},
			expectedChange: semver.None,
		},
		{
			name: "nested commit forces minor bump",
			commits: []*gitrepo.ConventionalCommit{
				{Type: "fix"},
				{Type: "feat", IsNested: true},
			},
			expectedChange: semver.Minor,
		},
		{
			name: "nested commit with breaking change forces minor bump",
			commits: []*gitrepo.ConventionalCommit{
				{Type: "feat", IsBreaking: true, IsNested: true},
				{Type: "feat"},
			},
			expectedChange: semver.Minor,
		},
		{
			name: "major change and nested commit",
			commits: []*gitrepo.ConventionalCommit{
				{Type: "feat", IsBreaking: true},
				{Type: "fix", IsNested: true},
			},
			expectedChange: semver.Major,
		},
		{
			name: "nested commit before major change",
			commits: []*gitrepo.ConventionalCommit{
				{Type: "fix", IsNested: true},
				{Type: "feat", IsBreaking: true},
			},
			expectedChange: semver.Major,
		},
		{
			name: "nested commit with only fixes forces minor bump",
			commits: []*gitrepo.ConventionalCommit{
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
		commits        []*gitrepo.ConventionalCommit
		currentVersion string
		wantVersion    string
		wantErr        bool
	}{
		{
			name: "without override version",
			commits: []*gitrepo.ConventionalCommit{
				{Type: "feat"},
			},
			currentVersion: "1.0.0",
			wantVersion:    "1.1.0",
		},
		{
			name: "derive next returns error",
			commits: []*gitrepo.ConventionalCommit{
				{Type: "feat"},
			},
			currentVersion: "invalid-version",
			wantVersion:    "",
			wantErr:        true,
		},
		{
			name: "breaking change on nested commit results in minor bump",
			commits: []*gitrepo.ConventionalCommit{
				{Type: "feat", IsBreaking: true, IsNested: true},
			},
			currentVersion: "1.2.3",
			wantVersion:    "1.3.0",
		},
		{
			name: "major change before nested commit results in major bump",
			commits: []*gitrepo.ConventionalCommit{
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

func TestLibraryFilter(t *testing.T) {
	t.Parallel()
	commits := []*gitrepo.ConventionalCommit{
		{
			LibraryID: "foo",
			Footers:   map[string]string{},
		},
		{
			LibraryID: "bar",
			Footers:   map[string]string{},
		},
		{
			Footers: map[string]string{
				"Library-IDs": "foo",
			},
		},
		{
			Footers: map[string]string{
				"Library-IDs": "bar",
			},
		},
		{
			Footers: map[string]string{
				"Library-IDs": "foo, bar",
			},
		},
		{
			Footers: map[string]string{
				"Library-IDs": "foo,bar",
			},
		},
	}
	for _, test := range []struct {
		name      string
		libraryID string
		want      []*gitrepo.ConventionalCommit
	}{
		{
			name:      "filter by foo",
			libraryID: "foo",
			want: []*gitrepo.ConventionalCommit{
				commits[0],
				commits[2],
				commits[4],
				commits[5],
			},
		},
		{
			name:      "filter by bar",
			libraryID: "bar",
			want: []*gitrepo.ConventionalCommit{
				commits[1],
				commits[3],
				commits[4],
				commits[5],
			},
		},
		{
			name:      "filter by baz",
			libraryID: "baz",
			want:      nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := libraryFilter(commits, test.libraryID)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("libraryFilter() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
