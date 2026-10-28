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
	vGood1 := NewTestEnumValue("GOOD_1", 1)
	vGood2 := NewTestEnumValue("GOOD_2", 2)
	vGood3 := NewTestEnumValue("GOOD_3", 3)
	vGood4 := NewTestEnumValue("GOOD_4", 4)
	vBadDeprecated := NewTestEnumValue("BAD_DEPRECATED", 5).WithDeprecated(true)
	vBadDefault := NewTestEnumValue("BAD_DEFAULT", 0)

	for _, test := range []struct {
		name         string
		values       []*EnumValue
		wantExamples []*SampleValue
	}{
		{
			name:   "more than 3 good values",
			values: []*EnumValue{vGood1, vGood2, vGood3, vGood4},
			wantExamples: []*SampleValue{
				{EnumValue: vGood1, Index: 0},
				{EnumValue: vGood2, Index: 1},
				{EnumValue: vGood3, Index: 2},
			},
		},
		{
			name:   "less than 3 good values",
			values: []*EnumValue{vGood1, vGood2, vBadDeprecated},
			wantExamples: []*SampleValue{
				{EnumValue: vGood1, Index: 0},
				{EnumValue: vGood2, Index: 1},
			},
		},
		{
			name:   "no good values",
			values: []*EnumValue{vBadDefault, vBadDeprecated},
			wantExamples: []*SampleValue{
				{EnumValue: vBadDefault, Index: 0},
				{EnumValue: vBadDeprecated, Index: 1},
			},
		},
		{
			name:         "no values",
			values:       []*EnumValue{},
			wantExamples: []*SampleValue{},
		},
		{
			name:   "mixed good and bad values",
			values: []*EnumValue{vBadDefault, vGood1, vBadDeprecated, vGood2},
			wantExamples: []*SampleValue{
				{EnumValue: vGood1, Index: 0},
				{EnumValue: vGood2, Index: 1},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			enum := NewTestEnum("TestEnum").
				WithPackage("test.v1").
				WithValues(test.values...)
			model := NewTestAPI([]*Message{}, []*Enum{enum}, []*Service{})
			if err := CrossReference(model); err != nil {
				t.Fatal(err)
			}

			got := enum.ValuesForExamples
			if diff := cmp.Diff(test.wantExamples, got, cmpopts.IgnoreFields(EnumValue{}, "Parent")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEnrichSamplesOneOfExampleField(t *testing.T) {
	deprecated := NewTestField("deprecated_field").
		WithType(TypezString).
		WithOneOf().
		WithDeprecated(true)
	mapType := NewTestMap("$map<string, string>", TypezString, TypezString)
	mapField := NewTestField("map_field").
		WithMapType(mapType).
		WithOneOf()
	repeated := NewTestField("repeated_field").
		WithType(TypezString).
		WithRepeated().
		WithOneOf()
	scalar := NewTestField("scalar_field").
		WithType(TypezInt32).
		WithOneOf()
	oneMessage := NewTestMessage("OneMessage")
	messageField := NewTestField("message_field").
		WithMessageType(oneMessage).
		WithOneOf()
	anotherMessage := NewTestMessage("AnotherMessage")
	anotherMessageField := NewTestField("another_message_field").
		WithMessageType(anotherMessage).
		WithOneOf()

	for _, test := range []struct {
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
	} {
		t.Run(test.name, func(t *testing.T) {
			group := NewTestOneOf("test_oneof").WithFields(test.fields...)
			message := NewTestMessage("Message").
				WithFields(test.fields...).
				WithOneOfs(group)
			model := NewTestAPI([]*Message{message, oneMessage, anotherMessage}, nil, nil).WithMaps(mapType)
			if err := CrossReference(model); err != nil {
				t.Fatal(err)
			}

			got := group.ExampleField
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEnrichSamplesWithResourceNamePattern(t *testing.T) {
	t.Run("resource type resolution", func(t *testing.T) {
		res := NewTestResource("test.googleapis.com/Resource").
			WithPatterns(ResourcePattern{
				*(&PathSegment{}).WithLiteral("resources"),
				*(&PathSegment{}).WithVariable(NewPathVariable("resource").WithMatch()),
			})
		field := NewTestField("notName").
			WithType(TypezString).
			WithResourceReference("test.googleapis.com/Resource")
		message := NewTestMessage("ResourceMessage").
			WithFields(field).
			WithResource(res)
		model := NewTestAPI([]*Message{message}, []*Enum{}, []*Service{})

		if err := CrossReference(model); err != nil {
			t.Fatal(err)
		}

		got := field.ResourceNamePattern
		want := NewTestResourceNamePattern(ResourceNameSegment{Literal: "resources", Variable: "resource"})
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("child type resolution", func(t *testing.T) {
		res := NewTestResource("test.googleapis.com/Child").
			WithPatterns(ResourcePattern{
				*(&PathSegment{}).WithLiteral("parents"),
				*(&PathSegment{}).WithVariable(NewPathVariable("parent").WithMatch()),
				*(&PathSegment{}).WithLiteral("children"),
				*(&PathSegment{}).WithVariable(NewPathVariable("child").WithMatch()),
			})
		field := NewTestField("parent").
			WithType(TypezString).
			WithChildTypeReference("test.googleapis.com/Child")
		message := NewTestMessage("Message").WithFields(field)
		model := NewTestAPI([]*Message{message}, []*Enum{}, []*Service{})
		model.AddResource(res)

		if err := CrossReference(model); err != nil {
			t.Fatal(err)
		}

		got := field.ResourceNamePattern
		want := NewTestResourceNamePattern(ResourceNameSegment{Literal: "parents", Variable: "parent"})
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("name field resolution", func(t *testing.T) {
		res := NewTestResource("test.googleapis.com/Resource").
			WithPatterns(ResourcePattern{
				*(&PathSegment{}).WithLiteral("resources"),
				*(&PathSegment{}).WithVariable(NewPathVariable("resource").WithMatch()),
			})
		field := NewTestField("name").WithType(TypezString)
		message := NewTestMessage("ResourceMessage").
			WithFields(field).
			WithResource(res)
		field.MessageType = message
		model := NewTestAPI([]*Message{message}, []*Enum{}, []*Service{})

		if err := CrossReference(model); err != nil {
			t.Fatal(err)
		}

		got := field.ResourceNamePattern
		want := NewTestResourceNamePattern(ResourceNameSegment{Literal: "resources", Variable: "resource"})
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
			want: NewTestResourceNamePattern(
				ResourceNameSegment{Literal: "projects", Variable: "project"},
			),
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
			want: NewTestResourceNamePattern(
				ResourceNameSegment{Literal: "projects", Variable: "project"},
				ResourceNameSegment{Literal: "secrets", Variable: "secret"},
			),
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
			want: NewTestResourceNamePattern(
				ResourceNameSegment{Literal: "projects", Variable: "project"},
			),
		},
		{
			name: "with trailing literal",
			pattern: ResourcePattern{
				*(&PathSegment{}).WithLiteral("projects"),
				*(&PathSegment{}).WithVariable(NewPathVariable("project").WithMatch()),
				*(&PathSegment{}).WithLiteral("config"),
			},
			skipLast: false,
			want: NewTestResourceNamePattern(
				ResourceNameSegment{Literal: "projects", Variable: "project"},
				ResourceNameSegment{Literal: "config", Variable: ""},
			),
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
