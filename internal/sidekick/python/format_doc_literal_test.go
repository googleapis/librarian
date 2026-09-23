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
)

func TestIsIndentedLiteralLine(t *testing.T) {
	for _, test := range []struct {
		name string
		line string
		want bool
	}{
		{name: "empty line", line: "", want: false},
		{name: "whitespace only", line: "    ", want: false},
		{name: "no leading space", line: "foo", want: false},
		{name: "two spaces", line: "  foo", want: false},
		{name: "three spaces", line: "   foo", want: true},
		{name: "four spaces", line: "    foo", want: true},
		{name: "tab indented", line: "\tfoo", want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isIndentedLiteralLine(test.line)
			if got != test.want {
				t.Errorf("isIndentedLiteralLine(%q) = %v, want %v", test.line, got, test.want)
			}
		})
	}
}

func TestExtractIndentedLiteralBlocks(t *testing.T) {
	for _, test := range []struct {
		name       string
		text       string
		wantBlocks int
	}{
		{
			name:       "empty string",
			text:       "",
			wantBlocks: 0,
		},
		{
			name:       "fenced code block",
			text:       "Before\n```\ncode line 1\ncode line 2\n```\nAfter",
			wantBlocks: 1,
		},
		{
			name:       "indented block with 4 spaces",
			text:       "Header:\n\n    block line 1\n    block line 2\n\nFooter",
			wantBlocks: 1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			clean, blocks := extractIndentedLiteralBlocks(test.text)
			if len(blocks) != test.wantBlocks {
				t.Errorf("len(blocks) = %d, want %d", len(blocks), test.wantBlocks)
			}
			if test.wantBlocks > 0 && clean == test.text {
				t.Errorf("expected placeholder token in text, got identical text")
			}
		})
	}
}

func TestParseLiteralBlockIndex(t *testing.T) {
	for _, test := range []struct {
		name    string
		token   string
		wantIdx int
		wantOk  bool
	}{
		{name: "valid token 0", token: "\uE003LITERAL_BLOCK_0\uE004", wantIdx: 0, wantOk: true},
		{name: "valid token 42", token: "\uE003LITERAL_BLOCK_42\uE004", wantIdx: 42, wantOk: true},
		{name: "invalid prefix", token: "LITERAL_BLOCK_0\uE004", wantIdx: -1, wantOk: false},
		{name: "invalid suffix", token: "\uE003LITERAL_BLOCK_0", wantIdx: -1, wantOk: false},
		{name: "non-numeric index", token: "\uE003LITERAL_BLOCK_abc\uE004", wantIdx: -1, wantOk: false},
		{name: "arbitrary text", token: "regular text", wantIdx: -1, wantOk: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotIdx, gotOk := parseLiteralBlockIndex(test.token)
			if gotIdx != test.wantIdx || gotOk != test.wantOk {
				t.Errorf("parseLiteralBlockIndex(%q) = (%d, %v), want (%d, %v)", test.token, gotIdx, gotOk, test.wantIdx, test.wantOk)
			}
		})
	}
}
