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
)

func TestShouldExclude(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
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
		t.Run(tc.name, func(t *testing.T) {
			got := shouldExclude(tc.files, tc.excludePaths)
			if got != tc.want {
				t.Errorf("shouldExclude(%v, %v) = %v, want %v", tc.files, tc.excludePaths, got, tc.want)
			}
		})
	}
}

func TestFormatTag(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
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
		t.Run(tc.name, func(t *testing.T) {
			got := formatTag(tc.library)
			if got != tc.want {
				t.Errorf("formatTag() = %q, want %q", got, tc.want)
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
					Footers:     make(map[string]string),
				},
				{
					Type:        "fix",
					Scope:       "foo",
					Description: "a fix for foo",
					Footers:     make(map[string]string),
				},
			},
			wantErr: false,
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
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(conventionalcommits.ConventionalCommit{}, "SHA", "Body", "IsBreaking")); diff != "" {
				t.Errorf("GetConventionalCommitsSinceLastRelease() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
