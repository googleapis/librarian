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
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
)

func TestParseCommit(t *testing.T) {
	for _, test := range []struct {
		name    string
		message string
		want    *ConventionalCommit
		wantErr bool
	}{
		{
			name:    "simple feat",
			message: "feat: add new feature",
			want: &ConventionalCommit{
				Type:        "feat",
				Description: "add new feature",
				Footers:     make(map[string]string),
				SHA:         "fake-sha",
			},
		},
		{
			name:    "feat with scope",
			message: "feat(scope): add new feature",
			want: &ConventionalCommit{
				Type:        "feat",
				Scope:       "scope",
				Description: "add new feature",
				Footers:     make(map[string]string),
				SHA:         "fake-sha",
			},
		},
		{
			name:    "feat with breaking change",
			message: "feat!: add new feature",
			want: &ConventionalCommit{
				Type:        "feat",
				Description: "add new feature",
				IsBreaking:  true,
				Footers:     make(map[string]string),
				SHA:         "fake-sha",
			},
		},
		{
			name:    "feat with single footer",
			message: "feat: add new feature\n\nCo-authored-by: John Doe <john.doe@example.com>",
			want: &ConventionalCommit{
				Type:        "feat",
				Description: "add new feature",
				Footers:     map[string]string{"Co-authored-by": "John Doe <john.doe@example.com>"},
				SHA:         "fake-sha",
			},
		},
		{
			name:    "feat with multiple footers",
			message: "feat: add new feature\n\nCo-authored-by: John Doe <john.doe@example.com>\nReviewed-by: Jane Smith <jane.smith@example.com>",
			want: &ConventionalCommit{
				Type:        "feat",
				Description: "add new feature",
				Footers: map[string]string{
					"Co-authored-by": "John Doe <john.doe@example.com>",
					"Reviewed-by":    "Jane Smith <jane.smith@example.com>",
				},
				SHA: "fake-sha",
			},
		},
		{
			name: "feat with multiple footers for generated changes",
			message: `feat: [library-name] add new feature
This is the body.
...

PiperOrigin-RevId: piper_cl_number

Source-Link: [googleapis/googleapis@{source_commit_hash}](https://github.com/googleapis/googleapis/commit/{source_commit_hash})
`,
			want: &ConventionalCommit{
				Type:        "feat",
				Description: "[library-name] add new feature",
				Body:        "This is the body.\n...",
				IsBreaking:  false,
				Footers: map[string]string{
					"PiperOrigin-RevId": "piper_cl_number",
					"Source-Link":       "[googleapis/googleapis@{source_commit_hash}](https://github.com/googleapis/googleapis/commit/{source_commit_hash})",
				},
				SHA: "fake-sha",
			},
		},
		{
			name:    "feat with breaking change footer",
			message: "feat: add new feature\n\nBREAKING CHANGE: this is a breaking change",
			want: &ConventionalCommit{
				Type:        "feat",
				Description: "add new feature",
				Body:        "",
				IsBreaking:  true,
				Footers:     map[string]string{"BREAKING CHANGE": "this is a breaking change"},
				SHA:         "fake-sha",
			},
		},
		{
			name:    "feat with body and footers",
			message: "feat: add new feature\n\nThis is the body of the commit message.\nIt can span multiple lines.\n\nCo-authored-by: John Doe <john.doe@example.com>",
			want: &ConventionalCommit{
				Type:        "feat",
				Description: "add new feature",
				Body:        "This is the body of the commit message.\nIt can span multiple lines.",
				Footers:     map[string]string{"Co-authored-by": "John Doe <john.doe@example.com>"},
				SHA:         "fake-sha",
			},
		},
		{
			name:    "feat with multi-line footer",
			message: "feat: add new feature\n\nThis is the body.\n\nBREAKING CHANGE: this is a breaking change\nthat spans multiple lines.",
			want: &ConventionalCommit{
				Type:        "feat",
				Description: "add new feature",
				Body:        "This is the body.",
				IsBreaking:  true,
				Footers:     map[string]string{"BREAKING CHANGE": "this is a breaking change\nthat spans multiple lines."},
				SHA:         "fake-sha",
			},
		},
		{
			name:    "invalid commit",
			message: "this is not a conventional commit",
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseCommit(test.message, "fake-sha")
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("ParseCommit(%q) returned diff (-want +got):\n%s", test.message, diff)
			}
		})
	}
}

func TestShouldExclude(t *testing.T) {
	testCases := []struct {
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldExclude(tc.files, tc.excludePaths)
			if got != tc.want {
				t.Errorf("shouldExclude(%v, %v) = %v, want %v", tc.files, tc.excludePaths, got, tc.want)
			}
		})
	}
}

func TestFormatTag(t *testing.T) {
	testCases := []struct {
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatTag(tc.library)
			if got != tc.want {
				t.Errorf("formatTag() = %q, want %q", got, tc.want)
			}
		})
	}
}

func setupRepoForGetCommits(t *testing.T) *gitrepo.LocalRepository {
	t.Helper()
	dir := t.TempDir()
	gitRepo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("git.PlainInit failed: %v", err)
	}

	createAndCommit := func(path, msg string) {
		w, err := gitRepo.Worktree()
		if err != nil {
			t.Fatalf("Worktree() failed: %v", err)
		}
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("os.MkdirAll failed: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("content"), 0644); err != nil {
			t.Fatalf("os.WriteFile failed: %v", err)
		}
		if _, err := w.Add(path); err != nil {
			t.Fatalf("w.Add failed: %v", err)
		}
		_, err = w.Commit(msg, &git.CommitOptions{
			Author: &object.Signature{Name: "Test", Email: "test@example.com"},
		})
		if err != nil {
			t.Fatalf("w.Commit failed: %v", err)
		}
	}

	createAndCommit("foo/a.txt", "feat(foo): initial commit for foo")
	head, err := gitRepo.Head()
	if err != nil {
		t.Fatalf("repo.Head() failed: %v", err)
	}
	if _, err := gitRepo.CreateTag("foo-v1.0.0", head.Hash(), nil); err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	createAndCommit("bar/a.txt", "feat(bar): initial commit for bar")
	createAndCommit("foo/b.txt", "fix(foo): a fix for foo")
	createAndCommit("foo/README.md", "docs(foo): update README")
	createAndCommit("foo/c.txt", "feat(foo): another feature for foo")

	r, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{Dir: dir})
	if err != nil {
		t.Fatalf("gitrepo.NewRepository failed: %v", err)
	}
	return r
}

func TestGetConventionalCommitsSinceLastRelease(t *testing.T) {
	repo := setupRepoForGetCommits(t)
	for _, test := range []struct {
		name    string
		library *config.LibraryState
		want    []*ConventionalCommit
		wantErr bool
	}{
		{
			name: "get commits for foo",
			library: &config.LibraryState{
				ID:                  "foo",
				Version:             "1.0.0",
				TagFormat:           "{id}-v{version}",
				SourceRoots:         []string{"foo"},
				ReleaseExcludePaths: []string{"foo/README.md"},
			},
			want: []*ConventionalCommit{
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
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := GetConventionalCommitsSinceLastRelease(repo, test.library)
			if (err != nil) != test.wantErr {
				t.Errorf("GetCommits() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(ConventionalCommit{}, "SHA", "Body", "IsBreaking")); diff != "" {
				t.Errorf("GetCommits() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
