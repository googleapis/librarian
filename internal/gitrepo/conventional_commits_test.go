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

func TestParseCommit(t *testing.T) {
	for _, test := range []struct {
		name          string
		message       string
		want          *ConventionalCommit
		wantErr       bool
		wantErrPhrase string
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
			name:    "feat with multiple footers for generated changes",
			message: "feat: [library-name] add new feature\nThis is the body.\n...\n\nPiperOrigin-RevId: piper_cl_number\n\nSource-Link: [googleapis/googleapis@{source_commit_hash}](https://github.com/googleapis/googleapis/commit/{source_commit_hash})",
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
			name:    "feat with wrong breaking change footer",
			message: "feat: add new feature\n\nBreaking change: this is a breaking change",
			want: &ConventionalCommit{
				Type:        "feat",
				Description: "add new feature",
				Body:        "Breaking change: this is a breaking change",
				IsBreaking:  false,
				Footers:     map[string]string{},
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
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseCommit(test.message, "fake-sha")
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				if !strings.Contains(err.Error(), test.wantErrPhrase) {
					t.Errorf("ParseCommit(%q) returned error %q, want to contain %q", test.message, err.Error(), test.wantErrPhrase)
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
