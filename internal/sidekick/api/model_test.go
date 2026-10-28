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
	v1 := NewTestRoutingInfoVariant("v1")
	v2 := NewTestRoutingInfoVariant("v2")
	info := NewTestRoutingInfo("key", v1, v2)
	method := NewTestMethod("test").WithRouting(info)

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
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRoutingCombosSimpleAnd(t *testing.T) {
	v1 := NewTestRoutingInfoVariant("v1")
	i1 := NewTestRoutingInfo("key1", v1)
	v2 := NewTestRoutingInfoVariant("v2")
	i2 := NewTestRoutingInfo("key2", v2)
	method := NewTestMethod("test").WithRouting(i1, i2)

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
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRoutingCombosFull(t *testing.T) {
	va1 := NewTestRoutingInfoVariant("va1")
	va2 := NewTestRoutingInfoVariant("va2")
	va3 := NewTestRoutingInfoVariant("va3")
	a := NewTestRoutingInfo("a", va1, va2, va3)

	vb1 := NewTestRoutingInfoVariant("vb1")
	vb2 := NewTestRoutingInfoVariant("vb2")
	b := NewTestRoutingInfo("b", vb1, vb2)

	vc1 := NewTestRoutingInfoVariant("vc1")
	c := NewTestRoutingInfo("c", vc1)

	method := NewTestMethod("test").WithRouting(a, b, c)

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
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRoutingInfoVariantFieldName(t *testing.T) {
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
	got := (&PathTemplate{}).
		WithLiteral("v1").
		WithVariable(NewPathVariableBuilder("parent", "child").
			WithLiteral("projects").
			WithMatch().
			WithAllowReserved().
			WithLiteral("locations").
			WithMatchRecursive().
			Build()).
		WithVariableNamed("v2", "field").
		WithVerb("verb")
	verb := "verb"
	want := &PathTemplate{
		Segments: []PathSegment{
			{
				Literal: "v1",
			},
			{
				Variable: &PathVariable{
					fieldPath:     []string{"parent", "child"},
					segments:      []string{"projects", "*", "locations", "**"},
					allowReserved: true,
				},
			},
			{
				Variable: &PathVariable{
					fieldPath: []string{"v2", "field"},
					segments:  []string{"*"},
				},
			},
		},
		Verb: verb,
	}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(PathVariable{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	v := got.Segments[1].Variable
	if diff := cmp.Diff([]string{"parent", "child"}, v.FieldPath()); diff != "" {
		t.Errorf("FieldPath() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"projects", "*", "locations", "**"}, v.Segments()); diff != "" {
		t.Errorf("Segments() mismatch (-want +got):\n%s", diff)
	}
	if !v.AllowReserved() {
		t.Errorf("AllowReserved() = false, want true")
	}
	vFromSegments := NewPathVariableBuilder().WithFieldPath("a", "b").WithSegments("x", "*").Build()
	if diff := cmp.Diff([]string{"a", "b"}, vFromSegments.FieldPath()); diff != "" {
		t.Errorf("FieldPath() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"x", "*"}, vFromSegments.Segments()); diff != "" {
		t.Errorf("Segments() mismatch (-want +got):\n%s", diff)
	}
}

func TestPathBindingHeuristic(t *testing.T) {
	heuristic := &TargetResource{
		FieldPaths: [][]string{{"project"}, {"zone"}, {"instance"}},
	}
	binding := &PathBinding{
		Verb:           "GET",
		TargetResource: heuristic,
	}

	if binding.Verb != "GET" {
		t.Errorf("expected GET, got %s", binding.Verb)
	}

	if binding.TargetResource.FieldPaths[0][0] != "project" {
		t.Errorf("expected project, got %s", binding.TargetResource.FieldPaths[0][0])
	}
}

func TestTypezString(t *testing.T) {
	for _, test := range []struct {
		name string
		t    Typez
		want string
	}{
		{"UNDEFINED", TypezUndefined, "UNDEFINED"},
		{"DOUBLE", TypezDouble, "DOUBLE"},
		{"FLOAT", TypezFloat, "FLOAT"},
		{"INT64", TypezInt64, "INT64"},
		{"UINT64", TypezUint64, "UINT64"},
		{"INT32", TypezInt32, "INT32"},
		{"FIXED64", TypezFixed64, "FIXED64"},
		{"FIXED32", TypezFixed32, "FIXED32"},
		{"BOOL", TypezBool, "BOOL"},
		{"STRING", TypezString, "STRING"},
		{"GROUP", TypezGroup, "GROUP"},
		{"MESSAGE", TypezMessage, "MESSAGE"},
		{"BYTES", TypezBytes, "BYTES"},
		{"UINT32", TypezUint32, "UINT32"},
		{"ENUM", TypezEnum, "ENUM"},
		{"SFIXED32", TypezSfixed32, "SFIXED32"},
		{"SFIXED64", TypezSfixed64, "SFIXED64"},
		{"SINT32", TypezSint32, "SINT32"},
		{"SINT64", TypezSint64, "SINT64"},
		{"Default", Typez(99), "Typez(99)"},
		{"Negative", Typez(-1), "Typez(-1)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.t.String()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestService_HasClientSideStreaming(t *testing.T) {
	for _, test := range []struct {
		name    string
		methods []*Method
		want    bool
	}{
		{
			name: "no methods",
			want: false,
		},
		{
			name: "no streaming methods",
			methods: []*Method{
				NewTestMethod("m1"),
				NewTestMethod("m2"),
			},
			want: false,
		},
		{
			name: "one streaming method",
			methods: []*Method{
				NewTestMethod("m1"),
				NewTestMethod("m2").WithClientSideStreaming(),
			},
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := NewTestService("test").WithMethods(test.methods...)
			got := s.HasClientSideStreaming()
			if got != test.want {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestService_HasBidiStreaming(t *testing.T) {
	for _, test := range []struct {
		name    string
		methods []*Method
		want    bool
	}{
		{
			name: "no methods",
			want: false,
		},
		{
			name: "unary only",
			methods: []*Method{
				NewTestMethod("m1"),
			},
			want: false,
		},
		{
			name: "server only streaming",
			methods: []*Method{
				NewTestMethod("m1").WithServerSideStreaming(),
			},
			want: false,
		},
		{
			name: "client only streaming",
			methods: []*Method{
				NewTestMethod("m1").WithClientSideStreaming(),
			},
			want: false,
		},
		{
			name: "bidi streaming",
			methods: []*Method{
				NewTestMethod("m1").WithBidiStreaming(),
			},
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := NewTestService("test").WithMethods(test.methods...)
			got := s.HasBidiStreaming()
			if got != test.want {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestService_HasServerSideStreaming(t *testing.T) {
	for _, test := range []struct {
		name    string
		methods []*Method
		want    bool
	}{
		{
			name: "no methods",
			want: false,
		},
		{
			name: "unary only",
			methods: []*Method{
				NewTestMethod("m1"),
			},
			want: false,
		},
		{
			name: "bidi streaming (not server-only)",
			methods: []*Method{
				NewTestMethod("m1").WithBidiStreaming(),
			},
			want: false,
		},
		{
			name: "client only streaming",
			methods: []*Method{
				NewTestMethod("m1").WithClientSideStreaming(),
			},
			want: false,
		},
		{
			name: "server-side streaming",
			methods: []*Method{
				NewTestMethod("m1").WithServerSideStreaming(),
			},
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := NewTestService("test").WithMethods(test.methods...)
			got := s.HasServerSideStreaming()
			if got != test.want {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}
