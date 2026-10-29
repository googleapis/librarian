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
				NewSampleValue(vGood1, 0),
				NewSampleValue(vGood2, 1),
				NewSampleValue(vGood3, 2),
			},
		},
		{
			name:   "less than 3 good values",
			values: []*EnumValue{vGood1, vGood2, vBadDeprecated},
			wantExamples: []*SampleValue{
				NewSampleValue(vGood1, 0),
				NewSampleValue(vGood2, 1),
			},
		},
		{
			name:   "no good values",
			values: []*EnumValue{vBadDefault, vBadDeprecated},
			wantExamples: []*SampleValue{
				NewSampleValue(vBadDefault, 0),
				NewSampleValue(vBadDeprecated, 1),
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
				NewSampleValue(vGood1, 0),
				NewSampleValue(vGood2, 1),
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

			got := enum.ValuesForExamples()
			ignoreParent := cmp.FilterPath(func(p cmp.Path) bool { return p.Last().String() == ".parent" }, cmp.Ignore())
			if diff := cmp.Diff(test.wantExamples, got, allowUnexported(), ignoreParent); diff != "" {
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

			got := group.ExampleField()
			ignoreFieldCycle := cmp.FilterPath(func(p cmp.Path) bool {
				return p.Last().String() == ".parent" || p.Last().String() == ".group"
			}, cmp.Ignore())
			if diff := cmp.Diff(test.want, got, allowUnexported(), ignoreFieldCycle); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEnrichSamplesWithResourceNamePattern(t *testing.T) {
	t.Run("resource type resolution", func(t *testing.T) {
		res := NewTestResource("test.googleapis.com/Resource").
			WithPatterns(ResourcePattern{
				*NewPathSegmentBuilder().WithLiteral("resources").Build(),
				*NewPathSegmentBuilder().WithVariable(NewPathVariableBuilder("resource").WithMatch().Build()).Build(),
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

		got := field.ResourceNamePattern()
		want := NewTestResourceNamePattern(NewResourceNameSegment("resources", "resource"))
		if diff := cmp.Diff(want, got, allowUnexported()); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("child type resolution", func(t *testing.T) {
		res := NewTestResource("test.googleapis.com/Child").
			WithPatterns(ResourcePattern{
				*NewPathSegmentBuilder().WithLiteral("parents").Build(),
				*NewPathSegmentBuilder().WithVariable(NewPathVariableBuilder("parent").WithMatch().Build()).Build(),
				*NewPathSegmentBuilder().WithLiteral("children").Build(),
				*NewPathSegmentBuilder().WithVariable(NewPathVariableBuilder("child").WithMatch().Build()).Build(),
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

		got := field.ResourceNamePattern()
		want := NewTestResourceNamePattern(NewResourceNameSegment("parents", "parent"))
		if diff := cmp.Diff(want, got, allowUnexported()); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("name field resolution", func(t *testing.T) {
		res := NewTestResource("test.googleapis.com/Resource").
			WithPatterns(ResourcePattern{
				*NewPathSegmentBuilder().WithLiteral("resources").Build(),
				*NewPathSegmentBuilder().WithVariable(NewPathVariableBuilder("resource").WithMatch().Build()).Build(),
			})
		field := NewTestField("name").WithType(TypezString)
		message := NewTestMessage("ResourceMessage").
			WithFields(field).
			WithResource(res)
		field.WithMessageType(message)
		model := NewTestAPI([]*Message{message}, []*Enum{}, []*Service{})

		if err := CrossReference(model); err != nil {
			t.Fatal(err)
		}

		got := field.ResourceNamePattern()
		want := NewTestResourceNamePattern(NewResourceNameSegment("resources", "resource"))
		if diff := cmp.Diff(want, got, allowUnexported()); diff != "" {
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
				*NewPathSegmentBuilder().WithLiteral("projects").Build(),
				*NewPathSegmentBuilder().WithVariable(NewPathVariableBuilder("project").WithMatch().Build()).Build(),
			},
			skipLast: false,
			want: NewTestResourceNamePattern(
				NewResourceNameSegment("projects", "project"),
			),
		},
		{
			name: "multi-segment",
			pattern: ResourcePattern{
				*NewPathSegmentBuilder().WithLiteral("projects").Build(),
				*NewPathSegmentBuilder().WithVariable(NewPathVariableBuilder("project").WithMatch().Build()).Build(),
				*NewPathSegmentBuilder().WithLiteral("secrets").Build(),
				*NewPathSegmentBuilder().WithVariable(NewPathVariableBuilder("secret").WithMatch().Build()).Build(),
			},
			skipLast: false,
			want: NewTestResourceNamePattern(
				NewResourceNameSegment("projects", "project"),
				NewResourceNameSegment("secrets", "secret"),
			),
		},
		{
			name: "multi-segment-skip-last",
			pattern: ResourcePattern{
				*NewPathSegmentBuilder().WithLiteral("projects").Build(),
				*NewPathSegmentBuilder().WithVariable(NewPathVariableBuilder("project").WithMatch().Build()).Build(),
				*NewPathSegmentBuilder().WithLiteral("secrets").Build(),
				*NewPathSegmentBuilder().WithVariable(NewPathVariableBuilder("secret").WithMatch().Build()).Build(),
			},
			skipLast: true,
			want: NewTestResourceNamePattern(
				NewResourceNameSegment("projects", "project"),
			),
		},
		{
			name: "with trailing literal",
			pattern: ResourcePattern{
				*NewPathSegmentBuilder().WithLiteral("projects").Build(),
				*NewPathSegmentBuilder().WithVariable(NewPathVariableBuilder("project").WithMatch().Build()).Build(),
				*NewPathSegmentBuilder().WithLiteral("config").Build(),
			},
			skipLast: false,
			want: NewTestResourceNamePattern(
				NewResourceNameSegment("projects", "project"),
				NewResourceNameSegment("config", ""),
			),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := toResourceNamePattern(test.pattern, test.skipLast)
			if diff := cmp.Diff(test.want, got, allowUnexported()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
