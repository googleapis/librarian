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

package conventionalcommits

import (
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/gitrepo"
)

func TestParseCommits(t *testing.T) {
	now := time.Now()
	sha := plumbing.NewHash("fake-sha")
	for _, test := range []struct {
		name          string
		message       string
		want          []*ConventionalCommit
		wantErr       bool
		wantErrPhrase string
	}{
		{
			name:    "simple_commit_with_no_library_association",
			message: "feat: add new feature",
			want: []*ConventionalCommit{
				{
					Type:       "feat",
					Subject:    "add new feature",
					LibraryID:  "example-id",
					IsNested:   false,
					Footers:    make(map[string]string),
					CommitHash: sha.String(),
					When:       now,
				},
			},
		},
		{
			name:    "simple_commit_with_scope",
			message: "feat(scope): add new feature",
			want: []*ConventionalCommit{
				{
					Type:       "feat",
					Subject:    "add new feature",
					LibraryID:  "example-id",
					IsNested:   false,
					Footers:    make(map[string]string),
					CommitHash: sha.String(),
					When:       now,
				},
			},
		},
		{
			name:    "simple_commit_with_breaking_change",
			message: "feat!: add new feature",
			want: []*ConventionalCommit{
				{
					Type:       "feat",
					Subject:    "add new feature",
					LibraryID:  "example-id",
					IsBreaking: true,
					IsNested:   false,
					Footers:    make(map[string]string),
					CommitHash: sha.String(),
					When:       now,
				},
			},
		},
		{
			name:    "commit_with_single_footer",
			message: "feat: add new feature\n\nCo-authored-by: John Doe <john.doe@example.com>",
			want: []*ConventionalCommit{
				{
					Type:       "feat",
					Subject:    "add new feature",
					LibraryID:  "example-id",
					IsNested:   false,
					Footers:    map[string]string{"Co-authored-by": "John Doe <john.doe@example.com>"},
					CommitHash: sha.String(),
					When:       now,
				},
			},
		},
		{
			name:    "commit_with_multiple_footers",
			message: "feat: add new feature\n\nCo-authored-by: John Doe <john.doe@example.com>\nReviewed-by: Jane Smith <jane.smith@example.com>",
			want: []*ConventionalCommit{
				{
					Type:      "feat",
					Subject:   "add new feature",
					LibraryID: "example-id",
					IsNested:  false,
					Footers: map[string]string{
						"Co-authored-by": "John Doe <john.doe@example.com>",
						"Reviewed-by":    "Jane Smith <jane.smith@example.com>",
					},
					CommitHash: sha.String(),
					When:       now,
				},
			},
		},
		{
			name:    "commit_with_multiple_footers_for_generated_changes",
			message: "feat: [library-name] add new feature\n\nThis is the body.\n...\n\nPiperOrigin-RevId: piper_cl_number\n\nSource-Link: [googleapis/googleapis@{source_commit_hash}](https://github.com/googleapis/googleapis/commit/abcdefg1234567)",
			want: []*ConventionalCommit{
				{
					Type:       "feat",
					Subject:    "[library-name] add new feature",
					Body:       "This is the body.\n...",
					LibraryID:  "library-name",
					IsNested:   false,
					IsBreaking: false,
					Footers: map[string]string{
						"PiperOrigin-RevId": "piper_cl_number",
						"Source-Link":       "abcdefg1234567",
					},
					CommitHash: sha.String(),
					When:       now,
				},
			},
		},
		{
			name:    "commit_with_breaking_change_footer",
			message: "feat: add new feature\n\nBREAKING CHANGE: this is a breaking change",
			want: []*ConventionalCommit{
				{
					Type:       "feat",
					Subject:    "add new feature",
					Body:       "",
					LibraryID:  "example-id",
					IsNested:   false,
					IsBreaking: true,
					Footers:    map[string]string{"BREAKING CHANGE": "this is a breaking change"},
					CommitHash: sha.String(),
					When:       now,
				},
			},
		},
		{
			name:    "commit_with_wrong_breaking_change_footer",
			message: "feat: add new feature\n\nBreaking change: this is a breaking change",
			want: []*ConventionalCommit{
				{
					Type:       "feat",
					Subject:    "add new feature",
					Body:       "Breaking change: this is a breaking change",
					LibraryID:  "example-id",
					IsNested:   false,
					IsBreaking: false,
					Footers:    map[string]string{},
					CommitHash: sha.String(),
					When:       now,
				},
			},
		},
		{
			name:    "commit_with_body_and_footers",
			message: "feat: add new feature\n\nThis is the body of the commit message.\nIt can span multiple lines.\n\nCo-authored-by: John Doe <john.doe@example.com>",
			want: []*ConventionalCommit{
				{
					Type:       "feat",
					Subject:    "add new feature",
					Body:       "This is the body of the commit message.\nIt can span multiple lines.",
					LibraryID:  "example-id",
					IsNested:   false,
					Footers:    map[string]string{"Co-authored-by": "John Doe <john.doe@example.com>"},
					CommitHash: sha.String(),
					When:       now,
				},
			},
		},
		{
			name:    "commit_with_multi-line_footer",
			message: "feat: add new feature\n\nThis is the body.\n\nBREAKING CHANGE: this is a breaking change\nthat spans multiple lines.",
			want: []*ConventionalCommit{
				{
					Type:       "feat",
					Subject:    "add new feature",
					Body:       "This is the body.",
					LibraryID:  "example-id",
					IsNested:   false,
					IsBreaking: true,
					Footers:    map[string]string{"BREAKING CHANGE": "this is a breaking change\nthat spans multiple lines."},
					CommitHash: sha.String(),
					When:       now,
				},
			},
		},
		{
			name: "commit_override",
			message: `feat: original message

BEGIN_COMMIT_OVERRIDE
fix(override): this is the override message

This is the body of the override.

Reviewed-by: Jane Doe
END_COMMIT_OVERRIDE`,
			want: []*ConventionalCommit{
				{
					Type:       "fix",
					Subject:    "this is the override message",
					Body:       "This is the body of the override.",
					LibraryID:  "example-id",
					IsNested:   false,
					Footers:    map[string]string{"Reviewed-by": "Jane Doe"},
					CommitHash: sha.String(),
					When:       now,
				},
			},
		},
		{
			name:    "invalid_conventional_commit",
			message: "this is not a conventional commit",
			wantErr: false,
			want:    nil,
		},
		{
			name:          "empty_commit_message",
			message:       "",
			wantErr:       true,
			wantErrPhrase: "empty commit",
		},
		{
			name: "commit_with_nested_commit",
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
					Type:       "feat",
					Subject:    "main feature",
					Body:       "main commit body",
					LibraryID:  "example-id",
					IsNested:   false,
					Footers:    map[string]string{},
					CommitHash: sha.String(),
					When:       now,
				},
				{
					Type:       "fix",
					Subject:    "fix a bug",
					Body:       "some details for the fix",
					LibraryID:  "example-id",
					IsNested:   true,
					Footers:    map[string]string{},
					CommitHash: sha.String(),
					When:       now,
				},
				{
					Type:       "chore",
					Subject:    "update deps",
					Body:       "",
					LibraryID:  "example-id",
					IsNested:   true,
					Footers:    map[string]string{},
					CommitHash: sha.String(),
					When:       now,
				},
			},
		},
		{
			name: "commit_with_empty_nested_commit",
			message: `feat(parser): main feature
2nd line of title

BEGIN_NESTED_COMMIT
END_NESTED_COMMIT
`,
			want: []*ConventionalCommit{
				{
					Type:       "feat",
					Subject:    "main feature 2nd line of title",
					LibraryID:  "example-id",
					IsNested:   false,
					Footers:    map[string]string{},
					CommitHash: sha.String(),
					When:       now,
				},
			},
		},
		{
			name: "commit_override_with_nested_commits",
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

Source-Link: fake-link
END_NESTED_COMMIT
BEGIN_NESTED_COMMIT
feat: [abc] nested commit 2

body of nested commit 2
...

PiperOrigin-RevId: 654321

Source-Link: fake-link
END_NESTED_COMMIT
END_COMMIT_OVERRIDE
`,
			want: []*ConventionalCommit{
				{
					Type:       "feat",
					Subject:    "[abc] nested commit 1",
					Body:       "body of nested commit 1\n...",
					LibraryID:  "abc",
					IsNested:   true,
					Footers:    map[string]string{"PiperOrigin-RevId": "123456", "Source-Link": "fake-link"},
					CommitHash: sha.String(),
					When:       now,
				},
				{
					Type:       "feat",
					Subject:    "[abc] nested commit 2",
					IsNested:   true,
					Body:       "body of nested commit 2\n...",
					LibraryID:  "abc",
					Footers:    map[string]string{"PiperOrigin-RevId": "654321", "Source-Link": "fake-link"},
					CommitHash: sha.String(),
					When:       now,
				},
			},
		},
		{
			name: "nest_commit_outside_of_override_ignored",
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
					Type:       "fix",
					Subject:    "this is the override message",
					Body:       "This is the body of the override.",
					LibraryID:  "example-id",
					IsNested:   false,
					Footers:    map[string]string{"Reviewed-by": "Jane Doe"},
					CommitHash: sha.String(),
					When:       now,
				},
			},
		},
		{
			name: "parse_multiple_lines_message_inside_nested_commit,_one_line_header",
			message: `
chore: Update generation configuration at Tue Aug 26 02:31:23 UTC 2025 (#11734)

This pull request is generated with proto changes between
[googleapis/googleapis@525c95a](https://github.com/googleapis/googleapis/commit/525c95a7a122ec2869ae06cd02fa5013819463f6)
(exclusive) and
[googleapis/googleapis@b738e78](https://github.com/googleapis/googleapis/commit/b738e78ed63effb7d199ed2d61c9e03291b6077f)
(inclusive).

BEGIN_COMMIT_OVERRIDE
BEGIN_NESTED_COMMIT
feat: [texttospeech] Support promptable voices by specifying a model name and a prompt
feat: [texttospeech] Add enum value M4A to enum AudioEncoding
docs: [texttospeech] A comment for method 'StreamingSynthesize' in service 'TextToSpeech' is changed

PiperOrigin-RevId: 799242210

Source-Link: [googleapis/googleapis@b738e78](https://github.com/googleapis/googleapis/commit/b738e78ed63effb7d199ed2d61c9e03291b6077f)
END_NESTED_COMMIT
END_COMMIT_OVERRIDE`,
			want: []*ConventionalCommit{
				{
					Type:      "feat",
					Subject:   "[texttospeech] Support promptable voices by specifying a model name and a prompt",
					LibraryID: "texttospeech",
					IsNested:  true,
					Footers: map[string]string{
						"PiperOrigin-RevId": "799242210",
						"Source-Link":       "b738e78ed63effb7d199ed2d61c9e03291b6077f",
					},
					CommitHash: sha.String(),
					When:       now,
				},
				{
					Type:      "feat",
					Subject:   "[texttospeech] Add enum value M4A to enum AudioEncoding",
					LibraryID: "texttospeech",
					IsNested:  true,
					Footers: map[string]string{
						"PiperOrigin-RevId": "799242210",
						"Source-Link":       "b738e78ed63effb7d199ed2d61c9e03291b6077f",
					},
					CommitHash: sha.String(),
					When:       now,
				},
				{
					Type:      "docs",
					Subject:   "[texttospeech] A comment for method 'StreamingSynthesize' in service 'TextToSpeech' is changed",
					LibraryID: "texttospeech",
					IsNested:  true,
					Footers: map[string]string{
						"PiperOrigin-RevId": "799242210",
						"Source-Link":       "b738e78ed63effb7d199ed2d61c9e03291b6077f",
					},
					CommitHash: sha.String(),
					When:       now,
				},
			},
		},
		{
			name: "parse_multiple_lines_message_inside_nested_commit_multi_line_headers",
			message: `
chore: Update generation configuration at Tue Aug 26 02:31:23 UTC 2025 (#11734)

This pull request is generated with proto changes between
[googleapis/googleapis@525c95a](https://github.com/googleapis/googleapis/commit/525c95a7a122ec2869ae06cd02fa5013819463f6)
(exclusive) and
[googleapis/googleapis@b738e78](https://github.com/googleapis/googleapis/commit/b738e78ed63effb7d199ed2d61c9e03291b6077f)
(inclusive).

BEGIN_COMMIT_OVERRIDE
BEGIN_NESTED_COMMIT
feat: [texttospeech] Support promptable voices by specifying a model
name and a prompt
feat: [texttospeech] Add enum value M4A to enum AudioEncoding
docs: [texttospeech] A comment for method 'StreamingSynthesize' in
service 'TextToSpeech' is changed

END_NESTED_COMMIT
END_COMMIT_OVERRIDE`,
			want: []*ConventionalCommit{
				{
					Type:       "feat",
					Subject:    "[texttospeech] Support promptable voices by specifying a model name and a prompt",
					LibraryID:  "texttospeech",
					IsNested:   true,
					Footers:    map[string]string{},
					CommitHash: sha.String(),
					When:       now,
				},
				{
					Type:       "feat",
					Subject:    "[texttospeech] Add enum value M4A to enum AudioEncoding",
					LibraryID:  "texttospeech",
					IsNested:   true,
					Footers:    map[string]string{},
					CommitHash: sha.String(),
					When:       now,
				},
				{
					Type:       "docs",
					Subject:    "[texttospeech] A comment for method 'StreamingSynthesize' in service 'TextToSpeech' is changed",
					LibraryID:  "texttospeech",
					IsNested:   true,
					Footers:    map[string]string{},
					CommitHash: sha.String(),
					When:       now,
				},
			},
		},
		{
			name: "generation_commit_has_incorrect_libraryID_passed_in",
			message: `
chore: librarian generate pull request: 20250919T072957Z (#14501)

This pull request is generated with proto changes between

[googleapis/googleapis@f8776fe](googleapis/googleapis@f8776fe)
(exclusive) and

[googleapis/googleapis@36533b0](googleapis/googleapis@36533b0)
(inclusive).

BEGIN_COMMIT_OVERRIDE
BEGIN_NESTED_COMMIT
docs: [google-cloud-video-live-stream] Update requirements of resource ID fields to be more clear

END_NESTED_COMMIT

BEGIN_NESTED_COMMIT
feat: [google-cloud-eventarc] add new fields to Eventarc resources

END_NESTED_COMMIT
END_COMMIT_OVERRIDE`,
			want: []*ConventionalCommit{
				{
					Type:       "docs",
					Subject:    "[google-cloud-video-live-stream] Update requirements of resource ID fields to be more clear",
					LibraryID:  "google-cloud-video-live-stream",
					IsNested:   true,
					Footers:    map[string]string{},
					CommitHash: sha.String(),
					When:       now,
				},
				{
					Type:       "feat",
					Subject:    "[google-cloud-eventarc] add new fields to Eventarc resources",
					LibraryID:  "google-cloud-eventarc",
					IsNested:   true,
					Footers:    map[string]string{},
					CommitHash: sha.String(),
					When:       now,
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			commit := &gitrepo.Commit{
				Message: test.message,
				Hash:    plumbing.NewHash("fake-sha"),
				When:    now,
			}
			got, err := ParseCommits(commit, "example-id")
			if test.wantErr {
				if err == nil {
					t.Fatalf("%s should return error", test.name)
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
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractCommitParts(t *testing.T) {
	for _, test := range []struct {
		name    string
		message string
		want    []commitPart
	}{
		{
			name:    "empty message",
			message: "",
			want:    nil,
		},
		{
			name:    "no nested commits",
			message: "feat: hello world",
			want: []commitPart{
				{message: "feat: hello world", isNested: false},
			},
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
			want: []commitPart{
				{message: "fix(sub): fix a bug", isNested: true},
				{message: "chore(deps): update deps", isNested: true},
			},
		},
		{
			name: "primary and nested commits",
			message: `feat(parser): main feature
BEGIN_NESTED_COMMIT
fix(sub): fix a bug
END_NESTED_COMMIT
`,
			want: []commitPart{
				{message: "feat(parser): main feature", isNested: false},
				{message: "fix(sub): fix a bug", isNested: true},
			},
		},
		{
			name: "malformed nested commit without end marker, with primary commit",
			message: `feat(parser): main feature
BEGIN_NESTED_COMMIT
fix(sub): fix a bug that is never closed`,
			want: []commitPart{
				{message: "feat(parser): main feature", isNested: false},
			},
		},
		{
			name: "malformed nested commit without end marker, without primary commit",
			message: `BEGIN_NESTED_COMMIT
fix(sub): fix a bug that is never closed`,
			want: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := extractCommitParts(test.message)
			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(commitPart{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestConventionalCommit_MarshalJSON(t *testing.T) {
	c := &ConventionalCommit{
		Type:    "feat",
		Subject: "new feature",
		Body:    "body of feature",
		Footers: map[string]string{
			"PiperOrigin-RevId": "12345",
		},
		CommitHash: "1234",
	}
	b, err := c.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() failed: %v", err)
	}
	want := `{"type":"feat","subject":"new feature","body":"body of feature","commit_hash":"1234","piper_cl_number":"12345"}`
	if diff := cmp.Diff(want, string(b)); diff != "" {
		t.Errorf("MarshalJSON() mismatch (-want +got):\n%s", diff)
	}
}

func TestParseFooters(t *testing.T) {
	for _, test := range []struct {
		name           string
		footerLines    []string
		wantFooters    map[string]string
		wantIsBreaking bool
	}{
		{
			name: "single footer",
			footerLines: []string{
				"Reviewed-by: G. Gemini",
			},
			wantFooters: map[string]string{
				"Reviewed-by": "G. Gemini",
			},
		},
		{
			name: "multiple footers",
			footerLines: []string{
				"Reviewed-by: G. Gemini",
				"Co-authored-by: Another Person <another@person.com>",
			},
			wantFooters: map[string]string{
				"Reviewed-by":    "G. Gemini",
				"Co-authored-by": "Another Person <another@person.com>",
			},
		},
		{
			name: "repeated footer keys, keep first",
			footerLines: []string{
				"PiperOrigin-RevId: 123456",
				"Source-Link: first value",
				"Source-Link: second value",
			},
			wantFooters: map[string]string{
				"PiperOrigin-RevId": "123456",
				"Source-Link":       "first value",
			},
		},
		{
			name: "multiline footer",
			footerLines: []string{
				"BREAKING CHANGE: something broke",
				"  and it was a big deal",
			},
			wantFooters: map[string]string{
				"BREAKING CHANGE": "something broke\n  and it was a big deal",
			},
			wantIsBreaking: true,
		},
		{
			name: "empty lines",
			footerLines: []string{
				"Reviewed-by: G. Gemini",
				"",
				"",
				"Co-authored-by: Another Person <another@person.com>",
			},
			wantFooters: map[string]string{
				"Reviewed-by":    "G. Gemini",
				"Co-authored-by": "Another Person <another@person.com>",
			},
		},
		{
			name: "multi-line footers with key on one line, value on the next",
			footerLines: []string{
				"PiperOrigin-RevId: 123456",
				"Library-IDs:",
				"library-one,library-two",
				"Source-Link:",
				"",
				"googleapis/googleapis@a12b345",
				"",
				"",
				"Copy-Tag:",
				"eyJwIjoic",
			},
			wantFooters: map[string]string{
				"PiperOrigin-RevId": "123456",
				"Library-IDs":       "library-one,library-two",
				"Source-Link":       "googleapis/googleapis@a12b345",
				"Copy-Tag":          "eyJwIjoic",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotFooters, gotIsBreaking := parseFooters(test.footerLines)
			if diff := cmp.Diff(test.wantFooters, gotFooters); diff != "" {
				t.Errorf("parseFooters() footers mismatch (-want +got):%s", diff)
			}
			if gotIsBreaking != test.wantIsBreaking {
				t.Errorf("parseFooters() isBreaking = %v, want %v", gotIsBreaking, test.wantIsBreaking)
			}
		})
	}
}
