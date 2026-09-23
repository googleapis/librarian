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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWrapUnformatted(t *testing.T) {
	for _, test := range []struct {
		name   string
		text   string
		width  int
		indent int
		want   []string
	}{
		{
			name:   "empty",
			text:   "",
			width:  60,
			indent: 4,
			want:   nil,
		},
		{
			name:   "single line text",
			text:   "A brief description of this resource.",
			width:  60,
			indent: 4,
			want:   []string{"A brief description of this resource."},
		},
		{
			name:   "multiline text with colon and bullet list",
			text:   "Options available:\n- Option 1\n- Option 2",
			width:  60,
			indent: 4,
			want: []string{
				"Options available:",
				"",
				"- Option 1",
				"- Option 2",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := wrapUnformatted(test.text, test.width, test.indent)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSplitWordsAndSpaces(t *testing.T) {
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
			name:  "words and spaces",
			input: "hello world  test",
			want:  []string{"hello", " ", "world", "  ", "test"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := splitWordsAndSpaces(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsListItem(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  bool
	}{
		{name: "short string", input: "-", want: false},
		{name: "bullet dash", input: "- item", want: true},
		{name: "bullet plus", input: "+ item", want: true},
		{name: "numbered list", input: "1. first item", want: true},
		{name: "plain sentence", input: "Just a regular sentence.", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isListItem(test.input)
			if got != test.want {
				t.Errorf("isListItem(%q) = %v, want %v", test.input, got, test.want)
			}
		})
	}
}

func TestGetSubsequentLineIndentLevel(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  int
	}{
		{name: "dash list", input: "- item", want: 2},
		{name: "plus list", input: "+ item", want: 2},
		{name: "numbered list", input: "1. item", want: 4},
		{name: "plain text", input: "regular text", want: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := getSubsequentLineIndentLevel(test.input)
			if got != test.want {
				t.Errorf("getSubsequentLineIndentLevel(%q) = %d, want %d", test.input, got, test.want)
			}
		})
	}
}
