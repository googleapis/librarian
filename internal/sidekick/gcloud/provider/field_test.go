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

package provider

import (
	"testing"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGetGcloudType(t *testing.T) {
	for _, test := range []struct {
		name  string
		typez api.Typez
		want  string
	}{
		{"String", api.TypezString, "str"},
		{"Int32", api.TypezInt32, "int"},
		{"Int64", api.TypezInt64, "long"},
		{"UInt32", api.TypezUint32, "int"},
		{"UInt64", api.TypezUint64, "long"},
		{"Bool", api.TypezBool, "bool"},
		{"Float", api.TypezFloat, "float"},
		{"Double", api.TypezDouble, "float"},
		{"Bytes", api.TypezBytes, "bytes"},
		{"Enum", api.TypezEnum, "str"},
		{"Message", api.TypezMessage, "arg_object"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := GetGcloudType(test.typez)
			if got != test.want {
				t.Errorf("GetGcloudType(%v) = %q, want %q", test.typez, got, test.want)
			}
		})
	}
}

func TestIsSafeName(t *testing.T) {
	for _, test := range []struct {
		name string
		want bool
	}{
		{"validName", true},
		{"valid_name", true},
		{"valid.name", true},
		{"valid123", true},
		{"invalid-name", false},
		{"invalid name", false},
		{"invalid$name", false},
		{"", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := IsSafeName(test.name)
			if got != test.want {
				t.Errorf("IsSafeName(%q) = %v, want %v", test.name, got, test.want)
			}
		})
	}
}

func TestCleanDocumentation(t *testing.T) {
	for _, test := range []struct {
		name string
		in   string
		want string
	}{
		{"NoPrefix", "This is help text.", "This is help text."},
		{"Required", "Required. This is help text.", "This is help text."},
		{"Identifier", "Identifier. This is help text.", "This is help text."},
		{"Optional", "Optional. This is help text.", "This is help text."},
		{"Both_RequiredFirst", "Required. Identifier. This is help text.", "This is help text."},
		{"Both_IdentifierFirst", "Identifier. Required. This is help text.", "This is help text."},
		{"OptionalAndRequired", "Optional. Required. This is help text.", "This is help text."},
		{"Repeated", "Required. Required. This is help text.", "This is help text."},
		{"Empty", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := CleanDocumentation(test.in)
			if got != test.want {
				t.Errorf("CleanDocumentation(%q) = %q, want %q", test.in, got, test.want)
			}
		})
	}
}

func TestGetGcloudType_Panic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("GetGcloudType() did not panic for unsupported type")
		}
	}()
	GetGcloudType(api.Typez(999))
}

func TestGetFieldHelpText(t *testing.T) {
	overrides := &Config{
		APIs: []API{
			{
				HelpText: &HelpTextRules{
					FieldRules: []*HelpTextRule{
						{
							Selector: "test.googleapis.com/Instance.name",
							HelpText: &HelpTextElement{
								Brief: "Override Brief",
							},
						},
					},
				},
			},
		},
	}

	for _, test := range []struct {
		name      string
		overrides *Config
		field     *api.Field
		want      string
	}{
		{
			name:      "Override",
			overrides: overrides,
			field: &api.Field{
				ID:   "test.googleapis.com/Instance.name",
				Name: "name",
			},
			want: "Override Brief",
		},
		{
			name: "Documentation",
			field: &api.Field{
				Name:          "description",
				Documentation: "My proto comment.",
			},
			want: "My proto comment.",
		},
		{
			name: "CleanDocumentation",
			field: &api.Field{
				Name:          "description",
				Documentation: "Required. My proto comment.",
			},
			want: "My proto comment.",
		},
		{
			name: "Fallback",
			field: &api.Field{
				Name: "description",
			},
			want: "Value for the `description` field.",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := GetFieldHelpText(test.overrides, test.field)
			if got != test.want {
				t.Errorf("GetFieldHelpText() = %q, want %q", got, test.want)
			}
		})
	}
}
