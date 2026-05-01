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

package gcloud

import (
	"testing"
)

func TestEscapeKeyword(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		// Keywords requested to be escaped
		{input: "break", want: "break_"},
		{input: "default", want: "default_"},
		{input: "func", want: "func_"},
		{input: "interface", want: "interface_"},
		{input: "map", want: "map_"},
		{input: "struct", want: "struct_"},
		{input: "int", want: "int_"},

		// Non-keywords requested NOT to be escaped
		{input: "foo", want: "foo"},
		{input: "bar", want: "bar"},
	} {
		t.Run(test.input, func(t *testing.T) {
			got := escapeKeyword(test.input)
			if got != test.want {
				t.Errorf("escapeKeyword(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
