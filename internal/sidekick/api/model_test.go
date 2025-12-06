// Copyright 2025 Google LLC
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

func TestRoutingCombosSimpleOr(t *testing.T) {
	v1 := &RoutingInfoVariant{
		FieldPath: []string{"v1"},
	}
	v2 := &RoutingInfoVariant{
		FieldPath: []string{"v2"},
	}
	info := &RoutingInfo{
		Name:     "key",
		Variants: []*RoutingInfoVariant{v1, v2},
	}
	method := Method{
		Routing: []*RoutingInfo{info},
	}

	want := []*RoutingInfoCombo{
		{
			Items: []*RoutingInfoComboItem{
				{
					Name:    "key",
					Variant: v1,
				},
			},
		},
		{
			Items: []*RoutingInfoComboItem{
				{
					Name:    "key",
					Variant: v2,
				},
			},
		},
	}
	if diff := cmp.Diff(want, method.RoutingCombos()); diff != "" {
		t.Errorf("Incorrect routing combos (-want, +got):\n%s", diff)
	}
}

func TestRoutingCombosSimpleAnd(t *testing.T) {
	v1 := &RoutingInfoVariant{
		FieldPath: []string{"v1"},
	}
	i1 := &RoutingInfo{
		Name:     "key1",
		Variants: []*RoutingInfoVariant{v1},
	}
	v2 := &RoutingInfoVariant{
		FieldPath: []string{"v2"},
	}
	i2 := &RoutingInfo{
		Name:     "key2",
		Variants: []*RoutingInfoVariant{v2},
	}
	method := Method{
		Routing: []*RoutingInfo{i1, i2},
	}

	want := []*RoutingInfoCombo{
		{
			Items: []*RoutingInfoComboItem{
				{
					Name:    "key1",
					Variant: v1,
				},
				{
					Name:    "key2",
					Variant: v2,
				},
			},
		},
	}
	if diff := cmp.Diff(want, method.RoutingCombos()); diff != "" {
		t.Errorf("Incorrect routing combos (-want, +got):\n%s", diff)
	}
}

func TestRoutingCombosFull(t *testing.T) {
	va1 := &RoutingInfoVariant{
		FieldPath: []string{"va1"},
	}
	va2 := &RoutingInfoVariant{
		FieldPath: []string{"va2"},
	}
	va3 := &RoutingInfoVariant{
		FieldPath: []string{"va3"},
	}
	a := &RoutingInfo{
		Name:     "a",
		Variants: []*RoutingInfoVariant{va1, va2, va3},
	}

	vb1 := &RoutingInfoVariant{
		FieldPath: []string{"vb1"},
	}
	vb2 := &RoutingInfoVariant{
		FieldPath: []string{"vb2"},
	}
	b := &RoutingInfo{
		Name:     "b",
		Variants: []*RoutingInfoVariant{vb1, vb2},
	}

	vc1 := &RoutingInfoVariant{
		FieldPath: []string{"vc1"},
	}
	c := &RoutingInfo{
		Name:     "c",
		Variants: []*RoutingInfoVariant{vc1},
	}

	method := Method{
		Routing: []*RoutingInfo{a, b, c},
	}

	make_combo := func(va *RoutingInfoVariant, vb *RoutingInfoVariant, vc *RoutingInfoVariant) *RoutingInfoCombo {
		return &RoutingInfoCombo{
			Items: []*RoutingInfoComboItem{
				{
					Name:    "a",
					Variant: va,
				},
				{
					Name:    "b",
					Variant: vb,
				},
				{
					Name:    "c",
					Variant: vc,
				},
			},
		}
	}
	want := []*RoutingInfoCombo{
		make_combo(va1, vb1, vc1),
		make_combo(va1, vb2, vc1),
		make_combo(va2, vb1, vc1),
		make_combo(va2, vb2, vc1),
		make_combo(va3, vb1, vc1),
		make_combo(va3, vb2, vc1),
	}
	if diff := cmp.Diff(want, method.RoutingCombos()); diff != "" {
		t.Errorf("Incorrect routing combos (-want, +got):\n%s", diff)
	}
}

func TestRoutingInfoVarianFieldName(t *testing.T) {
	variant := &RoutingInfoVariant{
		FieldPath: []string{"request", "b", "c"},
	}
	got := variant.FieldName()
	want := "request.b.c"
	if got != want {
		t.Errorf("mismatch in FieldName got=%q, want=%q", got, want)
	}
}

func TestRoutingInfoVariantTemplateAsString(t *testing.T) {
	variant := &RoutingInfoVariant{
		Prefix: RoutingPathSpec{
			Segments: []string{"a", "b", "c"},
		},
		Matching: RoutingPathSpec{
			Segments: []string{"d", "*"},
		},
		Suffix: RoutingPathSpec{
			Segments: []string{"e", "**"},
		},
	}
	got := variant.TemplateAsString()
	want := "a/b/c/d/*/e/**"
	if got != want {
		t.Errorf("mismatch in TemplateAsString got=%q, want=%q", got, want)
	}
}

func TestPathTemplateBuilder(t *testing.T) {
	got := NewPathTemplate().
		WithLiteral("v1").
		WithVariable(NewPathVariable("parent", "child").
			WithLiteral("projects").
			WithMatch().
			WithLiteral("locations").
			WithMatchRecursive()).
		WithVariableNamed("v2", "field").
		WithVerb("verb")
	name := "v1"
	verb := "verb"
	want := &PathTemplate{
		Segments: []PathSegment{
			{
				Literal: &name,
			},
			{
				Variable: &PathVariable{
					FieldPath: []string{"parent", "child"},
					Segments:  []string{"projects", "*", "locations", "**"},
				},
			},
			{
				Variable: &PathVariable{
					FieldPath: []string{"v2", "field"},
					Segments:  []string{"*"},
				},
			},
		},
		Verb: &verb,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("bad builder result (-want, +got):\n%s", diff)
	}
}

func TestFieldTypePredicates(t *testing.T) {
	type TestCase struct {
		field    *Field
		isString bool
		isBytes  bool
		isBool   bool
		isInt    bool
		isUInt   bool
		isFloat  bool
		isEnum   bool
		isObject bool
	}
	testCases := []TestCase{
		{field: &Field{Typez: STRING_TYPE}, isString: true},
		{field: &Field{Typez: BYTES_TYPE}, isBytes: true},
		{field: &Field{Typez: BOOL_TYPE}, isBool: true},
		{field: &Field{Typez: INT32_TYPE}, isInt: true},
		{field: &Field{Typez: INT64_TYPE}, isInt: true},
		{field: &Field{Typez: SINT32_TYPE}, isInt: true},
		{field: &Field{Typez: SINT64_TYPE}, isInt: true},
		{field: &Field{Typez: SFIXED32_TYPE}, isInt: true},
		{field: &Field{Typez: SFIXED64_TYPE}, isInt: true},
		{field: &Field{Typez: UINT32_TYPE}, isUInt: true},
		{field: &Field{Typez: UINT64_TYPE}, isUInt: true},
		{field: &Field{Typez: FIXED32_TYPE}, isUInt: true},
		{field: &Field{Typez: FIXED64_TYPE}, isUInt: true},
		{field: &Field{Typez: FLOAT_TYPE}, isFloat: true},
		{field: &Field{Typez: DOUBLE_TYPE}, isFloat: true},
		{field: &Field{Typez: ENUM_TYPE}, isEnum: true},
		{field: &Field{Typez: MESSAGE_TYPE}, isObject: true},
	}
	for _, tc := range testCases {
		if tc.field.IsString() != tc.isString {
			t.Errorf("IsString() for %v should be %v", tc.field.Typez, tc.isString)
		}
		if tc.field.IsBytes() != tc.isBytes {
			t.Errorf("IsBytes() for %v should be %v", tc.field.Typez, tc.isBytes)
		}
		if tc.field.IsBool() != tc.isBool {
			t.Errorf("IsBool() for %v should be %v", tc.field.Typez, tc.isBool)
		}
		if tc.field.IsLikeInt() != tc.isInt {
			t.Errorf("IsLikeInt() for %v should be %v", tc.field.Typez, tc.isInt)
		}
		if tc.field.IsLikeUInt() != tc.isUInt {
			t.Errorf("IsLikeUInt() for %v should be %v", tc.field.Typez, tc.isUInt)
		}
		if tc.field.IsLikeFloat() != tc.isFloat {
			t.Errorf("IsLikeFloat() for %v should be %v", tc.field.Typez, tc.isFloat)
		}
		if tc.field.IsEnum() != tc.isEnum {
			t.Errorf("IsEnum() for %v should be %v", tc.field.Typez, tc.isEnum)
		}
		if tc.field.IsObject() != tc.isObject {
			t.Errorf("IsObject() for %v should be %v", tc.field.Typez, tc.isObject)
		}
	}
}

func TestFlatPath(t *testing.T) {
	for _, test := range []struct {
		Input *PathTemplate
		Want  string
	}{
		{
			Input: NewPathTemplate(),
			Want:  "",
		},
		{
			Input: NewPathTemplate().
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("zones").
				WithVariableNamed("zone"),
			Want: "projects/{project}/zones/{zone}",
		},
		{
			Input: NewPathTemplate().
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("global").
				WithLiteral("location"),
			Want: "projects/{project}/global/location",
		},
		{
			Input: NewPathTemplate().
				WithLiteral("projects").
				WithVariable(NewPathVariable("a", "b", "c").WithMatchRecursive()),
			Want: "projects/{a.b.c}",
		},
	} {
		got := test.Input.FlatPath()
		if got != test.Want {
			t.Errorf("mismatch want=%q, got=%q", test.Want, got)
		}
	}
}

func TestField_IsResourceReference(t *testing.T) {
	for _, test := range []struct {
		name  string
		field *Field
		want  bool
	}{
		{
			name:  "nil ResourceReference",
			field: &Field{},
			want:  false,
		},
		{
			name:  "non-nil ResourceReference",
			field: &Field{ResourceReference: &ResourceReference{}},
			want:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.field.IsResourceReference()
			if got != test.want {
				t.Errorf("IsResourceReference() got = %v, want %v", got, test.want)
			}
		})
	}
}
