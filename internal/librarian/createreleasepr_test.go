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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/statepb"
)

func TestCalculateNextVersion(t *testing.T) {
	tests := []struct {
		current string
		want    string // Empty for an error
	}{
		{
			current: "1.0.0",
			want:    "1.1.0",
		},
		{
			current: "1.2.3-beta01",
			want:    "1.2.3-beta02",
		},
		{
			current: "1.2.3-alpha99",
			want:    "1.2.3-alpha100",
		},
		{
			current: "1.2.3-beta.1",
			want:    "1.2.3-beta.2",
		},
		{
			current: "1.2.3-beta.9",
			want:    "1.2.3-beta.10",
		},
		{
			current: "1.2.3-bad",
			want:    "",
		},
		{
			current: "bad",
			want:    "",
		},
	}
	for _, test := range tests {
		library := &statepb.LibraryState{
			CurrentVersion: test.current,
		}
		got, err := calculateNextVersion(library, "")
		if test.want == "" {
			if err == nil {
				t.Errorf("calculateNextVersion(%s); error expected", test.current)
			}
			continue
		}
		if err != nil {
			t.Errorf("calculateNextVersion(%s); got error %v", test.current, err)
			continue
		}

		if test.want != got {
			t.Errorf("calculateNextVersion(%s) expected %s, got %s", test.current, test.want, got)
		}
	}
}

func TestFormatReleaseNotes(t *testing.T) {
	for _, test := range []struct {
		name    string
		commits []*CommitMessage
		want    string
	}{
		{
			"Basic",
			[]*CommitMessage{
				{
					Features: []string{"feature A"},
				},
			},
			"### New features\n\n- Feature A\n\n",
		},
		{
			"Duplicated Feature",
			[]*CommitMessage{
				{
					Features: []string{"feature A", "feature A"},
				},
			},
			"### New features\n\n- Feature A\n\n",
		},
		{
			"Duplicated Docs",
			[]*CommitMessage{
				{
					Docs: []string{"Doc A", "Doc A"},
				},
			},
			"### Documentation improvements\n\n- Doc A\n\n",
		},
		{
			"Duplicated Bugs",
			[]*CommitMessage{
				{
					Fixes: []string{"Bugfix A", "Bugfix A"},
				},
			},
			"### Bug fixes\n\n- Bugfix A\n\n",
		},
		{
			"Sequential Sorting",
			[]*CommitMessage{
				{
					Fixes: []string{"Bugfix B"},
				},
				{
					Fixes: []string{"Bugfix A"},
				},
			},
			"### Bug fixes\n\n- Bugfix B\n- Bugfix A\n\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatReleaseNotes(test.commits)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
