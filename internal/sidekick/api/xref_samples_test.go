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

package api

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestEnrichSamplesEnumValues(t *testing.T) {
	v_good1 := &EnumValue{Name: "GOOD_1", Number: 1}
	v_good2 := &EnumValue{Name: "GOOD_2", Number: 2}
	v_good3 := &EnumValue{Name: "GOOD_3", Number: 3}
	v_good4 := &EnumValue{Name: "GOOD_4", Number: 4}
	v_bad_deprecated := &EnumValue{Name: "BAD_DEPRECATED", Number: 5, Deprecated: true}
	v_bad_default := &EnumValue{Name: "BAD_DEFAULT", Number: 0}

	testCases := []struct {
		name         string
		values       []*EnumValue
		wantExamples []*SampleValue
	}{
		{
			name:   "more than 3 good values",
			values: []*EnumValue{v_good1, v_good2, v_good3, v_good4},
			wantExamples: []*SampleValue{
				{EnumValue: v_good1, Index: 0},
				{EnumValue: v_good2, Index: 1},
				{EnumValue: v_good3, Index: 2},
			},
		},
		{
			name:   "less than 3 good values",
			values: []*EnumValue{v_good1, v_good2, v_bad_deprecated},
			wantExamples: []*SampleValue{
				{EnumValue: v_good1, Index: 0},
				{EnumValue: v_good2, Index: 1},
			},
		},
		{
			name:   "no good values",
			values: []*EnumValue{v_bad_default, v_bad_deprecated},
			wantExamples: []*SampleValue{
				{EnumValue: v_bad_default, Index: 0},
				{EnumValue: v_bad_deprecated, Index: 1},
			},
		},
		{
			name:         "no values",
			values:       []*EnumValue{},
			wantExamples: []*SampleValue{},
		},
		{
			name:   "mixed good and bad values",
			values: []*EnumValue{v_bad_default, v_good1, v_bad_deprecated, v_good2},
			wantExamples: []*SampleValue{
				{EnumValue: v_good1, Index: 0},
				{EnumValue: v_good2, Index: 1},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enum := &Enum{
				Name:    "TestEnum",
				ID:      ".test.v1.TestEnum",
				Package: "test.v1",
				Values:  tc.values,
			}
			model := NewTestAPI([]*Message{}, []*Enum{enum}, []*Service{})
			if err := CrossReference(model); err != nil {
				t.Fatal(err)
			}

			got := enum.ValuesForExamples
			if diff := cmp.Diff(tc.wantExamples, got, cmpopts.IgnoreFields(EnumValue{}, "Parent")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEnrichSamplesOneOfExampleField(t *testing.T) {
	deprecated := &Field{
		Name:       "deprecated_field",
		ID:         ".test.Message.deprecated_field",
		Typez:      TypezString,
		IsOneOf:    true,
		Deprecated: true,
	}
	mapMessage := &Message{
		Name:  "$map<string, string>",
		ID:    "$map<string, string>",
		IsMap: true,
		Fields: []*Field{
			{Name: "key", ID: "$map<string, string>.key", Typez: TypezString},
			{Name: "value", ID: "$map<string, string>.value", Typez: TypezString},
		},
	}
	mapField := &Field{
		Name:    "map_field",
		ID:      ".test.Message.map_field",
		Typez:   TypezMessage,
		TypezID: "$map<string, string>",
		IsOneOf: true,
		Map:     true,
	}
	repeated := &Field{
		Name:     "repeated_field",
		ID:       ".test.Message.repeated_field",
		Typez:    TypezString,
		Repeated: true,
		IsOneOf:  true,
	}
	scalar := &Field{
		Name:    "scalar_field",
		ID:      ".test.Message.scalar_field",
		Typez:   TypezInt32,
		IsOneOf: true,
	}
	messageField := &Field{
		Name:    "message_field",
		ID:      ".test.Message.message_field",
		Typez:   TypezMessage,
		TypezID: ".test.OneMessage",
		IsOneOf: true,
	}
	anotherMessageField := &Field{
		Name:    "another_message_field",
		ID:      ".test.Message.another_message_field",
		Typez:   TypezMessage,
		TypezID: ".test.AnotherMessage",
		IsOneOf: true,
	}

	testCases := []struct {
		name   string
		fields []*Field
		want   *Field
	}{
		{
			name:   "all types",
			fields: []*Field{deprecated, mapField, repeated, scalar, messageField},
			want:   scalar,
		},
		{
			name:   "no primitives",
			fields: []*Field{deprecated, mapField, repeated, messageField},
			want:   messageField,
		},
		{
			name:   "only scalars and messages",
			fields: []*Field{messageField, scalar, anotherMessageField},
			want:   scalar,
		},
		{
			name:   "no scalars",
			fields: []*Field{deprecated, mapField, repeated},
			want:   repeated,
		},
		{
			name:   "only map and deprecated",
			fields: []*Field{deprecated, mapField},
			want:   mapField,
		},
		{
			name:   "only deprecated",
			fields: []*Field{deprecated},
			want:   deprecated,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			group := &OneOf{
				Name:   "test_oneof",
				ID:     ".test.Message.test_oneof",
				Fields: tc.fields,
			}
			message := &Message{
				Name:    "Message",
				ID:      ".test.Message",
				Package: "test",
				Fields:  tc.fields,
				OneOfs:  []*OneOf{group},
			}
			oneMessage := &Message{
				Name:    "OneMessage",
				ID:      ".test.OneMessage",
				Package: "test",
			}
			anotherMessage := &Message{
				Name:    "AnotherMessage",
				ID:      ".test.AnotherMessage",
				Package: "test",
			}
			model := NewTestAPI([]*Message{message, oneMessage, anotherMessage, mapMessage}, []*Enum{}, []*Service{})
			if err := CrossReference(model); err != nil {
				t.Fatal(err)
			}

			got := group.ExampleField
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEnrichSamplesWithResourceNamePattern(t *testing.T) {
	t.Run("resource type resolution", func(t *testing.T) {
		res := &Resource{
			Type: "test.googleapis.com/Resource",
			Patterns: []ResourcePattern{
				{
					*(&PathSegment{}).WithLiteral("resources"),
					*(&PathSegment{}).WithVariable(NewPathVariable("resource").WithMatch()),
				},
			},
		}
		field := &Field{
			Name:  "notName",
			ID:    ".test.ResourceMessage.name",
			Typez: TypezString,
			ResourceReference: &ResourceReference{
				Type: "test.googleapis.com/Resource",
			},
		}
		message := &Message{
			Name:     "ResourceMessage",
			ID:       ".test.ResourceMessage",
			Fields:   []*Field{field},
			Resource: res,
		}
		model := NewTestAPI([]*Message{message}, []*Enum{}, []*Service{})

		if err := CrossReference(model); err != nil {
			t.Fatal(err)
		}

		got := field.ResourceNamePattern
		want := &ResourceNamePattern{
			Segments: []ResourceNameSegment{
				{Literal: "resources", Variable: "resource"},
			},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("child type resolution", func(t *testing.T) {
		res := &Resource{
			Type: "test.googleapis.com/Child",
			Patterns: []ResourcePattern{
				{
					*(&PathSegment{}).WithLiteral("parents"),
					*(&PathSegment{}).WithVariable(NewPathVariable("parent").WithMatch()),
					*(&PathSegment{}).WithLiteral("children"),
					*(&PathSegment{}).WithVariable(NewPathVariable("child").WithMatch()),
				},
			},
		}
		field := &Field{
			Name:  "parent",
			ID:    ".test.Message.parent",
			Typez: TypezString,
			ResourceReference: &ResourceReference{
				ChildType: "test.googleapis.com/Child",
			},
		}
		message := &Message{
			Name:   "Message",
			ID:     ".test.Message",
			Fields: []*Field{field},
		}
		model := NewTestAPI([]*Message{message}, []*Enum{}, []*Service{})
		model.AddResource(res)

		if err := CrossReference(model); err != nil {
			t.Fatal(err)
		}

		got := field.ResourceNamePattern
		want := &ResourceNamePattern{
			Segments: []ResourceNameSegment{
				{Literal: "parents", Variable: "parent"},
			},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("name field resolution", func(t *testing.T) {
		res := &Resource{
			Type: "test.googleapis.com/Resource",
			Patterns: []ResourcePattern{
				{
					*(&PathSegment{}).WithLiteral("resources"),
					*(&PathSegment{}).WithVariable(NewPathVariable("resource").WithMatch()),
				},
			},
		}
		field := &Field{
			Name:  "name",
			ID:    ".test.ResourceMessage.name",
			Typez: TypezString,
		}
		message := &Message{
			Name:     "ResourceMessage",
			ID:       ".test.ResourceMessage",
			Fields:   []*Field{field},
			Resource: res,
		}
		field.MessageType = message
		model := NewTestAPI([]*Message{message}, []*Enum{}, []*Service{})

		if err := CrossReference(model); err != nil {
			t.Fatal(err)
		}

		got := field.ResourceNamePattern
		want := &ResourceNamePattern{
			Segments: []ResourceNameSegment{
				{Literal: "resources", Variable: "resource"},
			},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestToResourceNamePattern(t *testing.T) {
	for _, test := range []struct {
		name     string
		pattern  ResourcePattern
		skipLast bool
		want     *ResourceNamePattern
	}{
		{
			name: "simple",
			pattern: ResourcePattern{
				*(&PathSegment{}).WithLiteral("projects"),
				*(&PathSegment{}).WithVariable(NewPathVariable("project").WithMatch()),
			},
			skipLast: false,
			want: &ResourceNamePattern{
				Segments: []ResourceNameSegment{
					{Literal: "projects", Variable: "project"},
				},
			},
		},
		{
			name: "multi-segment",
			pattern: ResourcePattern{
				*(&PathSegment{}).WithLiteral("projects"),
				*(&PathSegment{}).WithVariable(NewPathVariable("project").WithMatch()),
				*(&PathSegment{}).WithLiteral("secrets"),
				*(&PathSegment{}).WithVariable(NewPathVariable("secret").WithMatch()),
			},
			skipLast: false,
			want: &ResourceNamePattern{
				Segments: []ResourceNameSegment{
					{Literal: "projects", Variable: "project"},
					{Literal: "secrets", Variable: "secret"},
				},
			},
		},
		{
			name: "multi-segment-skip-last",
			pattern: ResourcePattern{
				*(&PathSegment{}).WithLiteral("projects"),
				*(&PathSegment{}).WithVariable(NewPathVariable("project").WithMatch()),
				*(&PathSegment{}).WithLiteral("secrets"),
				*(&PathSegment{}).WithVariable(NewPathVariable("secret").WithMatch()),
			},
			skipLast: true,
			want: &ResourceNamePattern{
				Segments: []ResourceNameSegment{
					{Literal: "projects", Variable: "project"},
				},
			},
		},
		{
			name: "with trailing literal",
			pattern: ResourcePattern{
				*(&PathSegment{}).WithLiteral("projects"),
				*(&PathSegment{}).WithVariable(NewPathVariable("project").WithMatch()),
				*(&PathSegment{}).WithLiteral("config"),
			},
			skipLast: false,
			want: &ResourceNamePattern{
				Segments: []ResourceNameSegment{
					{Literal: "projects", Variable: "project"},
					{Literal: "config", Variable: ""},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := toResourceNamePattern(test.pattern, test.skipLast)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
