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

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestEscapeKeyword(t *testing.T) {
	for _, test := range []struct {
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
		// Metatype-related keywords, need custom escaping
		{input: "Type", want: "Type_"},
		{input: "Protocol", want: "Protocol_"},
		{input: "self", want: "self_"},

		// Non-keywords requested NOT to be escaped
		{input: "secret", want: "secret"},
		{input: "volume", want: "volume"},
	} {
		t.Run(test.input, func(t *testing.T) {
			got := escapeKeyword(test.input)
			if got != test.want {
				t.Errorf("escapeKeyword(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestCamelCase(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "secret_version", want: "secretVersion"},
		{input: "display_name", want: "displayName"},
		{input: "iam_policy", want: "iamPolicy"},
		{input: "Type", want: "type"},

		// Keywords that should be escaped after camelCase
		{input: "protocol", want: "`protocol`"},
		{input: "will_set", want: "`willSet`"},
		{input: "Self", want: "self_"},
	} {
		t.Run(test.input, func(t *testing.T) {
			got := camelCase(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPascalCase(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "SecretManagerService", want: "SecretManagerService"},
		{input: "CreateSecretRequest", want: "CreateSecretRequest"},
		{input: "IAMPolicy", want: "IAMPolicy"},
		{input: "IAM", want: "IAM"},

		// Keywords that should be escaped after pascalCase
		{input: "Protocol", want: "Protocol_"},
		{input: "Type", want: "Type_"},
		{input: "Self", want: "`Self`"},
		{input: "Any", want: "`Any`"},
	} {
		t.Run(test.input, func(t *testing.T) {
			got := pascalCase(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEnumValueCaseName(t *testing.T) {
	tests := []struct {
		name     string
		enumName string
		valName  string
		want     string
	}{
		{
			name:     "simple",
			enumName: "Color",
			valName:  "COLOR_RED",
			want:     "red",
		},
		{
			name:     "no prefix",
			enumName: "Color",
			valName:  "RED",
			want:     "red",
		},
		{
			name:     "numbers in prefix",
			enumName: "InstancePrivateIpv6GoogleAccess",
			valName:  "INSTANCE_PRIVATE_IPV6_GOOGLE_ACCESS_ENABLED",
			want:     "enabled",
		},
		{
			name:     "keyword",
			enumName: "Planet",
			valName:  "PLANET_SELF",
			want:     "self_", // keyword escaped
		},
		{
			name:     "number suffix after strip",
			enumName: "Foo",
			valName:  "FOO_VALUE_1",
			want:     "value1",
		},
		{
			name:     "number only after strip falls back to full name",
			enumName: "Foo",
			valName:  "FOO_1",
			want:     "foo1",
		},
		{
			name:     "acronym in enum name",
			enumName: "IAM",
			valName:  "IAM_POLICY",
			want:     "policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enum := &api.Enum{Name: tt.enumName}
			ev := &api.EnumValue{Name: tt.valName, Parent: enum}
			got := enumValueCaseName(ev)
			if got != tt.want {
				t.Errorf("enumValueCaseName() = %q, want %q", got, tt.want)
			}
		})
	}
}
