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

package swift

import (
	"testing"
)

func TestEscapeKeyword(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		// Keywords requested to be escaped
		{input: "let", want: "`let`"},
		{input: "protocol", want: "`protocol`"},
		{input: "class", want: "`class`"},
		{input: "enum", want: "`enum`"},
		{input: "func", want: "`func`"},
		{input: "if", want: "`if`"},
		{input: "while", want: "`while`"},

		// Non-keywords requested NOT to be escaped
		{input: "secret", want: "secret"},
		{input: "volume", want: "volume"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeKeyword(tt.input)
			if got != tt.want {
				t.Errorf("escapeKeyword(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
