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
	opts := cmp.AllowUnexported(RoutingInfoVariant{}, RoutingPathSpec{})
	if diff := cmp.Diff(want, got, opts); diff != "" {
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
