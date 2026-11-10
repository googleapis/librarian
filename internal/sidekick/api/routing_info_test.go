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

func TestRoutingInfoVariant(t *testing.T) {
	got := NewRoutingInfoVariantBuilder().
		WithFieldPath("app_profile_id").
		WithPrefixSegments("projects", "*").
		WithMatchingSegments("instances", "*").
		WithSuffixSegments("tables", "*").
		Build()
	got.SetAnnotations("rv-codec")

	want := &RoutingInfoVariant{
		fieldPath: []string{"app_profile_id"},
		prefix:    RoutingPathSpec{segments: []string{"projects", "*"}},
		matching:  RoutingPathSpec{segments: []string{"instances", "*"}},
		suffix:    RoutingPathSpec{segments: []string{"tables", "*"}},
		codec:     "rv-codec",
	}
	if diff := cmp.Diff(want, got, allowUnexported()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.fieldPath, got.FieldPath()); diff != "" {
		t.Errorf("FieldPath() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.prefix.segments, got.Prefix().Segments()); diff != "" {
		t.Errorf("Prefix().Segments() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.matching.segments, got.Matching().Segments()); diff != "" {
		t.Errorf("Matching().Segments() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.suffix.segments, got.Suffix().Segments()); diff != "" {
		t.Errorf("Suffix().Segments() mismatch (-want +got):\n%s", diff)
	}
	if got.FieldName() != "app_profile_id" {
		t.Errorf("FieldName() = %q, want %q", got.FieldName(), "app_profile_id")
	}
	if got.TemplateAsString() != "projects/*/instances/*/tables/*" {
		t.Errorf("TemplateAsString() = %q, want %q", got.TemplateAsString(), "projects/*/instances/*/tables/*")
	}
	if got.Codec() != "rv-codec" {
		t.Errorf("Codec() = %v, want rv-codec", got.Codec())
	}

	got.FieldPath()[0] = "mutated"
	got.Prefix().Segments()[0] = "mutated"
	got.Matching().Segments()[0] = "mutated"
	got.Suffix().Segments()[0] = "mutated"
	if got.FieldPath()[0] != "app_profile_id" || got.Prefix().Segments()[0] != "projects" || got.Matching().Segments()[0] != "instances" || got.Suffix().Segments()[0] != "tables" {
		t.Errorf("RoutingInfoVariant slice getters should return defensive copies")
	}
}

func TestRoutingInfoVariantSetAnnotationsPanicsWhenAlreadySet(t *testing.T) {
	v := NewRoutingInfoVariantBuilder().Build()
	v.SetAnnotations("first")
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when calling SetAnnotations a second time, got none")
		}
	}()
	v.SetAnnotations("second")
}

func TestRoutingInfoVariantClone(t *testing.T) {
	var nilVariant *RoutingInfoVariant
	if got := nilVariant.Clone(); got != nil {
		t.Errorf("nilVariant.Clone() = %v, want nil", got)
	}

	orig := NewRoutingInfoVariantBuilder().
		WithFieldPath("app_profile_id").
		WithPrefixSegments("projects", "*").
		WithMatchingSegments("instances", "*").
		WithSuffixSegments("tables", "*").
		Build()
	orig.SetAnnotations("rv-codec")

	got := orig.Clone()
	want := &RoutingInfoVariant{
		fieldPath: []string{"app_profile_id"},
		prefix:    RoutingPathSpec{segments: []string{"projects", "*"}},
		matching:  RoutingPathSpec{segments: []string{"instances", "*"}},
		suffix:    RoutingPathSpec{segments: []string{"tables", "*"}},
	}
	if diff := cmp.Diff(want, got, allowUnexported()); diff != "" {
		t.Errorf("Clone() mismatch (-want +got):\n%s", diff)
	}
}

func TestRoutingInfo(t *testing.T) {
	v1 := NewRoutingInfoVariantBuilder().
		WithFieldPath("table_name").
		WithMatchingSegments("projects", "*").
		Build()
	v2 := NewRoutingInfoVariantBuilder().
		WithFieldPath("app_profile_id").
		WithMatchingSegments("Profiles", "*").
		Build()

	got := NewRoutingInfoBuilder().
		WithName("project_id").
		WithVariants(v1).
		WithVariant(v2).
		Build()

	want := &RoutingInfo{
		name:     "project_id",
		variants: []*RoutingInfoVariant{v1, v2},
	}
	if diff := cmp.Diff(want, got, allowUnexported()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if got.Name() != want.name {
		t.Errorf("Name() = %q, want %q", got.Name(), want.name)
	}
	if diff := cmp.Diff(want.variants, got.Variants(), allowUnexported()); diff != "" {
		t.Errorf("Variants() mismatch (-want +got):\n%s", diff)
	}

	got.Variants()[0] = nil
	if got.Variants()[0] != v1 {
		t.Errorf("Variants() getter should return a defensive copy")
	}
}

func TestRoutingInfoClone(t *testing.T) {
	var nilRouting *RoutingInfo
	if got := nilRouting.Clone(); got != nil {
		t.Errorf("nilRouting.Clone() = %v, want nil", got)
	}

	v1 := NewRoutingInfoVariantBuilder().
		WithFieldPath("table_name").
		WithMatchingSegments("projects", "*").
		Build()
	v1.SetAnnotations("v1-codec")
	orig := NewRoutingInfoBuilder().
		WithName("project_id").
		WithVariant(v1).
		Build()

	got := orig.Clone()
	want := &RoutingInfo{
		name: "project_id",
		variants: []*RoutingInfoVariant{
			{
				fieldPath: []string{"table_name"},
				matching:  RoutingPathSpec{segments: []string{"projects", "*"}},
			},
		},
	}
	if diff := cmp.Diff(want, got, allowUnexported()); diff != "" {
		t.Errorf("Clone() mismatch (-want +got):\n%s", diff)
	}
}

func TestRoutingInfoCombo(t *testing.T) {
	v1 := NewRoutingInfoVariantBuilder().WithFieldPath("table_name").Build()
	gotItem := NewRoutingInfoComboItem("project_id", v1)
	wantItem := &RoutingInfoComboItem{
		name:    "project_id",
		variant: v1,
	}
	if diff := cmp.Diff(wantItem, gotItem, allowUnexported()); diff != "" {
		t.Errorf("item mismatch (-want +got):\n%s", diff)
	}
	if gotItem.Name() != wantItem.name || gotItem.Variant() != v1 {
		t.Errorf("unexpected RoutingInfoComboItem getters: (%q, %v)", gotItem.Name(), gotItem.Variant())
	}

	gotCombo := NewRoutingInfoCombo(gotItem)
	wantCombo := &RoutingInfoCombo{
		items: []*RoutingInfoComboItem{gotItem},
	}
	if diff := cmp.Diff(wantCombo, gotCombo, allowUnexported()); diff != "" {
		t.Errorf("combo mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantCombo.items, gotCombo.Items(), allowUnexported()); diff != "" {
		t.Errorf("Items() mismatch (-want +got):\n%s", diff)
	}
	gotCombo.Items()[0] = nil
	if gotCombo.Items()[0] != gotItem {
		t.Errorf("Items() getter should return a defensive copy")
	}
}
