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

func allowUnexported() cmp.Option {
	return cmp.AllowUnexported(
		PathBinding{},
		PathInfo{},
		PathSegment{},
		PathTemplate{},
		PathVariable{},
		TargetResource{},
	)
}

func TestRoutingCombosSimpleOr(t *testing.T) {
	v1 := NewTestRoutingInfoVariant("v1")
	v2 := NewTestRoutingInfoVariant("v2")
	info := NewTestRoutingInfo("key", v1, v2)
	method := NewTestMethod("test").WithRouting(info)

	want := []*RoutingInfoCombo{
		NewRoutingInfoCombo(NewRoutingInfoComboItem("key", v1)),
		NewRoutingInfoCombo(NewRoutingInfoComboItem("key", v2)),
	}
	if diff := cmp.Diff(want, method.RoutingCombos(), allowUnexported()); diff != "" {
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
		NewRoutingInfoCombo(
			NewRoutingInfoComboItem("key1", v1),
			NewRoutingInfoComboItem("key2", v2),
		),
	}
	if diff := cmp.Diff(want, method.RoutingCombos(), allowUnexported()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	combo := method.RoutingCombos()[0]
	if len(combo.Items()) != 2 {
		t.Fatalf("len(combo.Items()) = %d, want 2", len(combo.Items()))
	}
	if combo.Items()[0].Name() != "key1" || combo.Items()[0].Variant() != v1 {
		t.Errorf("combo.Items()[0] = (%q, %v), want (%q, %v)", combo.Items()[0].Name(), combo.Items()[0].Variant(), "key1", v1)
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
		return NewRoutingInfoCombo(
			NewRoutingInfoComboItem("a", va),
			NewRoutingInfoComboItem("b", vb),
			NewRoutingInfoComboItem("c", vc),
		)
	}
	want := []*RoutingInfoCombo{
		make_combo(va1, vb1, vc1),
		make_combo(va1, vb2, vc1),
		make_combo(va2, vb1, vc1),
		make_combo(va2, vb2, vc1),
		make_combo(va3, vb1, vc1),
		make_combo(va3, vb2, vc1),
	}
	if diff := cmp.Diff(want, method.RoutingCombos(), allowUnexported()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRoutingInfoVariantFieldName(t *testing.T) {
	variant := NewRoutingInfoVariantBuilder().
		WithFieldPath("request", "b", "c").
		Build()
	got := variant.FieldName()
	want := "request.b.c"
	if got != want {
		t.Errorf("mismatch in FieldName got=%q, want=%q", got, want)
	}
}

func TestRoutingInfoVariantTemplateAsString(t *testing.T) {
	variant := NewRoutingInfoVariantBuilder().
		WithPrefixSegments("a", "b", "c").
		WithMatchingSegments("d", "*").
		WithSuffixSegments("e", "**").
		Build()
	got := variant.TemplateAsString()
	want := "a/b/c/d/*/e/**"
	if got != want {
		t.Errorf("mismatch in TemplateAsString got=%q, want=%q", got, want)
	}
}

func TestPathBindingHeuristic(t *testing.T) {
	heuristic := NewTargetResource([][]string{{"project"}, {"zone"}, {"instance"}}, nil)
	binding := NewPathBindingBuilder().
		WithVerb("GET").
		WithTargetResource(heuristic).
		Build()

	if binding.Verb() != "GET" {
		t.Errorf("expected GET, got %s", binding.Verb())
	}

	if binding.TargetResource().FieldPaths()[0][0] != "project" {
		t.Errorf("expected project, got %s", binding.TargetResource().FieldPaths()[0][0])
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
