// Copyright 2026 Google LLC
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

package python

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFormatMessageDocLines(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty documentation",
			input: "",
			want:  nil,
		},
		{
			name:  "single line documentation",
			input: "A secret object with sensitive information.",
			want:  []string{"A secret object with sensitive information."},
		},
		{
			name:  "multiline documentation with bullet points",
			input: "A request message.\n\n* Option A\n* Option B",
			want: []string{
				"A request message.",
				"",
				"- Option A",
				"- Option B",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatMessageDocLines(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatFieldDocLines(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty doc",
			input: "",
			want:  nil,
		},
		{
			name:  "field doc without leading 12 spaces in returned slice",
			input: "The name of the secret resource.",
			want:  []string{"The name of the secret resource."},
		},
		{
			name:  "indented example preserved",
			input: "Optional. For example:\n\n  \"123/environment\": \"production\",\n  \"123/costCenter\": \"marketing\"\n\nTags are used.",
			want: []string{
				"Optional. For example:",
				"",
				"\"123/environment\": \"production\",",
				"  \"123/costCenter\": \"marketing\"",
				"",
				"Tags are used.",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatFieldDocLines(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			for _, line := range got {
				if strings.HasPrefix(line, "            ") {
					t.Errorf("formatFieldDocLines(%q) got line with 12 leading spaces %q, want unindented line (template provides indentation)", test.input, line)
				}
			}
		})
	}
}

func TestFormatDocLines(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty",
			input: "",
			want:  nil,
		},
		{
			name:  "single line",
			input: "Single line doc.",
			want:  []string{"Single line doc."},
		},
		{
			name:  "multiline with trailing whitespace",
			input: "Line 1   \nLine 2\t\nLine 3",
			want:  []string{"Line 1", "Line 2", "Line 3"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatDocLines(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatRstDoc(t *testing.T) {
	for _, test := range []struct {
		name   string
		input  string
		width  int
		indent int
		want   []string
	}{
		{
			name:   "empty",
			input:  "",
			width:  60,
			indent: 4,
			want:   nil,
		},
		{
			name:   "plain text wrapping",
			input:  "This is a plain documentation string that should be wrapped.",
			width:  30,
			indent: 4,
			want: []string{
				"This is a plain",
				"documentation string that",
				"should be wrapped.",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatRstDoc(test.input, test.width, test.indent)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
