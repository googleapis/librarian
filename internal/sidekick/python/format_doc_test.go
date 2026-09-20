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

func TestConvertMarkdownToRst(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "fenced code block",
			input: "Example code:\n```python\nx = 42\nprint(x)\n```",
			want:  "Example code:\n\n\n::\n\n   x = 42\n   print(x)\n\n",
		},
		{
			name:  "markdown link conversion",
			input: "See the [Cloud Storage documentation](https://cloud.google.com/storage).",
			want:  "See the `Cloud Storage documentation\uE000<https://cloud.google.com/storage>`__.",
		},
		{
			name:  "bullet normalization",
			input: "* item one\n* item two\n+ item three\n- item four",
			want:  "- item one\n- item two\n- item three\n- item four",
		},
		{
			name:  "strip raw HTML tags while preserving placeholder with underscore",
			input: "Path: <canonical service name>/<type> with <asset type> and <service_account_email>",
			want:  "Path: / with  and <service_account_email>",
		},
		{
			name:  "escape glob asterisk in prose",
			input: "Wildcard characters (such as * and ?) are supported.",
			want:  "Wildcard characters (such as \\* and ?) are supported.",
		},
		{
			name:  "escape domain glob asterisk",
			input: `"compute.googleapis.com.*" snapshots all compute resources.`,
			want:  `"compute.googleapis.com.\*" snapshots all compute resources.`,
		},
		{
			name:  "escape paired regex asterisk",
			input: `Pattern ".*Instance.*" matches instances.`,
			want:  `Pattern ".\ *Instance.*" matches instances.`,
		},
		{
			name:  "escape standalone underscore in parens",
			input: "Letters (A-Z), numbers (0-9), or underscores (_).",
			want:  "Letters (A-Z), numbers (0-9), or underscores (\\_).",
		},
		{
			name:  "in-word braced template emphasis",
			input: "Format is {SourceType}_{ACTION}_{DestType}.",
			want:  "Format is {SourceType}\\ *{ACTION}*\\ {DestType}.",
		},
		{
			name:  "quoted underscore replaced with asterisk",
			input: `Concatenated with "_" and separated by "_".`,
			want:  `Concatenated with "*" and separated by "*".`,
		},
		{
			name:  "code spans with single backticks become double",
			input: "Use `SecretPayload` to store data.",
			want:  "Use ``SecretPayload`` to store data.",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := convertMarkdownToRst(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

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
