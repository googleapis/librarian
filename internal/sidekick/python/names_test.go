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

func TestSnakeCase(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "camelCase", input: "getSecret", want: "get_secret"},
		{name: "pascalCase", input: "SecretManager", want: "secret_manager"},
		{name: "already_snake", input: "already_snake", want: "already_snake"},
		{name: "already_snake_with_digits", input: "data_crc32c", want: "data_crc32c"},
		{name: "with_acronym", input: "GetIAMPolicy", want: "get_iam_policy"},
		{name: "kebab-case", input: "foo-bar", want: "foo_bar"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := snakeCase(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPascalCase(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "snake_case", input: "secret_manager", want: "SecretManager"},
		{name: "pascal", input: "SecretManager", want: "SecretManager"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := pascalCase(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPythonIdentifier(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "keyword_from", input: "from", want: "from_"},
		{name: "keyword_import", input: "import", want: "import_"},
		{name: "non_keyword", input: "name", want: "name"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := pythonIdentifier(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
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
