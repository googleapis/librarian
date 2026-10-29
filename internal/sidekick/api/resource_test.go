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

func TestResourceRefs(t *testing.T) {
	segProjects := *NewPathSegmentBuilder().WithLiteral("projects").Build()
	gotTR := NewTargetResource([][]string{{"project"}, {"zone"}}, []PathSegment{segProjects})

	wantTR := &TargetResource{
		fieldPaths: [][]string{{"project"}, {"zone"}},
		template:   []PathSegment{segProjects},
	}
	if diff := cmp.Diff(wantTR, gotTR, allowUnexported()); diff != "" {
		t.Errorf("TargetResource mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantTR.fieldPaths, gotTR.FieldPaths()); diff != "" {
		t.Errorf("FieldPaths() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantTR.template, gotTR.Template(), allowUnexported()); diff != "" {
		t.Errorf("Template() mismatch (-want +got):\n%s", diff)
	}
	gotTR.FieldPaths()[0][0] = "mutated"
	if gotTR.FieldPaths()[0][0] != "project" {
		t.Errorf("FieldPaths() should return a deep defensive copy")
	}
	gotTR.Template()[0] = *NewPathSegmentBuilder().WithLiteral("mutated").Build()
	if gotTR.Template()[0].Literal() != "projects" {
		t.Errorf("Template() should return a defensive copy")
	}

	gotPattern := NewResourceNamePattern(
		NewResourceNameSegment("projects", ""),
		NewResourceNameSegment("", "project"),
	)
	wantPattern := &ResourceNamePattern{
		segments: []ResourceNameSegment{
			{literal: "projects"},
			{variable: "project"},
		},
	}
	if diff := cmp.Diff(wantPattern, gotPattern, allowUnexported()); diff != "" {
		t.Errorf("ResourceNamePattern mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantPattern.segments, gotPattern.Segments(), allowUnexported()); diff != "" {
		t.Errorf("Segments() mismatch (-want +got):\n%s", diff)
	}
	if gotPattern.Segments()[0].Literal() != "projects" || gotPattern.Segments()[1].Variable() != "project" {
		t.Errorf("unexpected segments: %+v", gotPattern.Segments())
	}
	gotPattern.Segments()[0] = NewResourceNameSegment("mutated", "")
	if gotPattern.Segments()[0].Literal() != "projects" {
		t.Errorf("Segments() should return a defensive copy")
	}

	gotRR := NewResourceReferenceBuilder().
		WithType("test.googleapis.com/Book").
		WithChildType("test.googleapis.com/Page").
		Build()
	gotRR.SetAnnotations("rr-codec")
	wantRR := &ResourceReference{
		typez:     "test.googleapis.com/Book",
		childType: "test.googleapis.com/Page",
		codec:     "rr-codec",
	}
	if diff := cmp.Diff(wantRR, gotRR, allowUnexported()); diff != "" {
		t.Errorf("ResourceReference mismatch (-want +got):\n%s", diff)
	}
	if gotRR.Type() != wantRR.typez || gotRR.ChildType() != wantRR.childType || gotRR.Codec() != wantRR.codec {
		t.Errorf("unexpected ResourceReference getters: %+v", gotRR)
	}
}
