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

package gitrepo

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseCommits(t *testing.T) {
	tests := []struct {
		name          string
		message       string
		want          []*ConventionalCommit
		wantErr       bool
		wantErrPhrase string
	}{
		{
			name:    "simple feat",
			message: "feat: add new feature",
			want: []*ConventionalCommit{
				{
					Type:        "feat",
					Description: "add new feature",
					Footers:     make(map[string]string),
					SHA:         "fake-sha",
				},
			},
		},
		{
			name:    "feat with scope",
			message: "feat(scope): add new feature",
			want: []*ConventionalCommit{
				{
					Type:        "feat",
					Scope:       "scope",
					Description: "add new feature",
					Footers:     make(map[string]string),
					SHA:         "fake-sha",
				},
			},
		},
		{
			name:    "feat with breaking change",
			message: "feat!: add new feature",
			want: []*ConventionalCommit{
				{
					Type:        "feat",
					Description: "add new feature",
					IsBreaking:  true,
					Footers:     make(map[string]string),
					SHA:         "fake-sha",
				},
			},
		},
		{
			name:    "feat with single footer",
			message: "feat: add new feature\n\nCo-authored-by: John Doe <john.doe@example.com>",
			want: []*ConventionalCommit{
				{
					Type:        "feat",
					Description: "add new feature",
					Footers:     map[string]string{"Co-authored-by": "John Doe <john.doe@example.com>"},
					SHA:         "fake-sha",
				},
			},
		},
		{
			name:    "feat with multiple footers",
			message: "feat: add new feature\n\nCo-authored-by: John Doe <john.doe@example.com>\nReviewed-by: Jane Smith <jane.smith@example.com>",
			want: []*ConventionalCommit{
				{
					Type:        "feat",
					Description: "add new feature",
					Footers: map[string]string{
						"Co-authored-by": "John Doe <john.doe@example.com>",
						"Reviewed-by":    "Jane Smith <jane.smith@example.com>",
					},
					SHA: "fake-sha",
				},
			},
		},
		{
			name:    "feat with multiple footers for generated changes",
			message: "feat: [library-name] add new feature\nThis is the body.\n...\n\nPiperOrigin-RevId: piper_cl_number\n\nSource-Link: [googleapis/googleapis@{source_commit_hash}](https://github.com/googleapis/googleapis/commit/{source_commit_hash})",
			want: []*ConventionalCommit{
				{
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
		},
		{
			name:    "feat with breaking change footer",
			message: "feat: add new feature\n\nBREAKING CHANGE: this is a breaking change",
			want: []*ConventionalCommit{
				{
					Type:        "feat",
					Description: "add new feature",
					Body:        "",
					IsBreaking:  true,
					Footers:     map[string]string{"BREAKING CHANGE": "this is a breaking change"},
					SHA:         "fake-sha",
				},
			},
		},
		{
			name:    "feat with wrong breaking change footer",
			message: "feat: add new feature\n\nBreaking change: this is a breaking change",
			want: []*ConventionalCommit{
				{
					Type:        "feat",
					Description: "add new feature",
					Body:        "Breaking change: this is a breaking change",
					IsBreaking:  false,
					Footers:     map[string]string{},
					SHA:         "fake-sha",
				},
			},
		},
		{
			name:    "feat with body and footers",
			message: "feat: add new feature\n\nThis is the body of the commit message.\nIt can span multiple lines.\n\nCo-authored-by: John Doe <john.doe@example.com>",
			want: []*ConventionalCommit{
				{
					Type:        "feat",
					Description: "add new feature",
					Body:        "This is the body of the commit message.\nIt can span multiple lines.",
					Footers:     map[string]string{"Co-authored-by": "John Doe <john.doe@example.com>"},
					SHA:         "fake-sha",
				},
			},
		},
		{
			name:    "feat with multi-line footer",
			message: "feat: add new feature\n\nThis is the body.\n\nBREAKING CHANGE: this is a breaking change\nthat spans multiple lines.",
			want: []*ConventionalCommit{
				{
					Type:        "feat",
					Description: "add new feature",
					Body:        "This is the body.",
					IsBreaking:  true,
					Footers:     map[string]string{"BREAKING CHANGE": "this is a breaking change\nthat spans multiple lines."},
					SHA:         "fake-sha",
				},
			},
		},
		{
			name: "commit override",
			message: `feat: original message

BEGIN_COMMIT_OVERRIDE
fix(override): this is the override message

This is the body of the override.

Reviewed-by: Jane Doe
END_COMMIT_OVERRIDE`,
			want: []*ConventionalCommit{
				{
					Type:        "fix",
					Scope:       "override",
					Description: "this is the override message",
					Body:        "This is the body of the override.",
					Footers:     map[string]string{"Reviewed-by": "Jane Doe"},
					SHA:         "fake-sha",
				},
			},
		},
		{
			name:    "invalid conventional commit",
			message: "this is not a conventional commit",
			wantErr: false,
			want:    nil,
		},
		{
			name:          "empty commit message",
			message:       "",
			wantErr:       true,
			wantErrPhrase: "empty commit",
		},
		{
			name: "commit with nested commit",
			message: `feat(parser): main feature
main commit body

BEGIN_NESTED_COMMIT
fix(sub): fix a bug

some details for the fix
END_NESTED_COMMIT
BEGIN_NESTED_COMMIT
chore(deps): update deps
END_NESTED_COMMIT
`,
			want: []*ConventionalCommit{
				{
					Type:        "feat",
					Scope:       "parser",
					Description: "main feature",
					Body:        "main commit body",
					Footers:     map[string]string{},
					SHA:         "fake-sha",
				},
				{
					Type:        "fix",
					Scope:       "sub",
					Description: "fix a bug",
					Body:        "some details for the fix",
					Footers:     map[string]string{},
					SHA:         "fake-sha",
				},
				{
					Type:        "chore",
					Scope:       "deps",
					Description: "update deps",
					Body:        "",
					Footers:     map[string]string{},
					SHA:         "fake-sha",
				},
			},
		},

		{
			name: "commit with empty nested commit",
			message: `feat(parser): main feature
main commit body

BEGIN_NESTED_COMMIT
END_NESTED_COMMIT
`,
			want: []*ConventionalCommit{
				{
					Type:        "feat",
					Scope:       "parser",
					Description: "main feature",
					Body:        "main commit body",
					Footers:     map[string]string{},
					SHA:         "fake-sha",
				},
			},
		},
		{
			name: "commit override with nested commits",
			message: `feat: API regeneration main commit

This pull request is generated with proto changes between
... ...

Librarian Version: {librarian_version}
Language Image: {language_image_name_and_digest}

BEGIN_COMMIT_OVERRIDE
BEGIN_NESTED_COMMIT
feat: [abc] nested commit 1
body of nested commit 1
...

PiperOrigin-RevId: 123456

Source-link: fake-link
END_NESTED_COMMIT
BEGIN_NESTED_COMMIT
feat: [abc] nested commit 2
body of nested commit 2
...

PiperOrigin-RevId: 654321

Source-link: fake-link
END_NESTED_COMMIT
END_COMMIT_OVERRIDE

`,
			want: []*ConventionalCommit{
				{
					Type:        "feat",
					Description: "[abc] nested commit 1",
					Body:        "body of nested commit 1\n...",
					Footers:     map[string]string{"PiperOrigin-RevId": "123456", "Source-link": "fake-link"},
					SHA:         "fake-sha",
				},
				{
					Type:        "feat",
					Description: "[abc] nested commit 2",
					Body:        "body of nested commit 2\n...",
					Footers:     map[string]string{"PiperOrigin-RevId": "654321", "Source-link": "fake-link"},
					SHA:         "fake-sha",
				},
			},
		},
		{
			name: "nest commit outside of override ignored",
			message: `feat: original message

BEGIN_NESTED_COMMIT
ignored line
BEGIN_COMMIT_OVERRIDE
fix(override): this is the override message

This is the body of the override.

Reviewed-by: Jane Doe
END_COMMIT_OVERRIDE
END_NESTED_COMMIT`,
			want: []*ConventionalCommit{
				{
					Type:        "fix",
					Scope:       "override",
					Description: "this is the override message",
					Body:        "This is the body of the override.",
					Footers:     map[string]string{"Reviewed-by": "Jane Doe"},
					SHA:         "fake-sha",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseCommits(test.message, "fake-sha")
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				if !strings.Contains(err.Error(), test.wantErrPhrase) {
					t.Errorf("ParseCommits(%q) returned error %q, want to contain %q", test.message, err.Error(), test.wantErrPhrase)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("ParseCommits(%q) returned diff (-want +got):\n%s", test.message, diff)
			}
		})
	}
}

func TestExtractCommitParts(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    []string
	}{
		{
			name:    "empty message",
			message: "",
			want:    []string(nil),
		},
		{
			name:    "no nested commits",
			message: "feat: hello world",
			want:    []string{"feat: hello world"},
		},
		{
			name: "only nested commits",
			message: `BEGIN_NESTED_COMMIT
fix(sub): fix a bug
END_NESTED_COMMIT
BEGIN_NESTED_COMMIT
chore(deps): update deps
END_NESTED_COMMIT
`,
			want: []string{
				"fix(sub): fix a bug",
				"chore(deps): update deps",
			},
		},
		{
			name: "empty nested commits",
			message: `feat(parser): main feature
BEGIN_NESTED_COMMIT
END_NESTED_COMMIT
`,
			want: []string{"feat(parser): main feature"},
		},
		{
			name: "malformed nested commits",
			message: `feat(parser): main feature
BEGIN_NESTED_COMMIT
fix(sub): fix a bug
`,
			want: []string{"feat(parser): main feature"},
		},
		{
			name: "malformed nested commits - reversed",
			message: `feat(parser): main feature
END_NESTED_COMMIT
fix(sub): fix a bug
BEGIN_NESTED_COMMIT
`,
			want: []string{"feat(parser): main feature\nEND_NESTED_COMMIT\nfix(sub): fix a bug"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCommitParts(tt.message)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("extractCommitParts(%q) returned diff (-want +got):\n%s", tt.message, diff)
			}
		})
	}
}
