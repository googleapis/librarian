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
	empty := NewTargetResource(nil, nil)
	if empty.FieldPaths() != nil {
		t.Errorf("empty.FieldPaths() = %v, want nil", empty.FieldPaths())
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

func TestResourceReferenceSetAnnotationsPanicsWhenAlreadySet(t *testing.T) {
	rr := NewResourceReferenceBuilder().Build()
	rr.SetAnnotations("first")
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when calling SetAnnotations a second time, got none")
		}
	}()
	rr.SetAnnotations("second")
}

func TestResource(t *testing.T) {
	msg := NewMessageBuilder().WithName("Book").Build()
	segProjects := *NewPathSegmentBuilder().WithLiteral("projects").Build()
	got := NewResourceBuilder().
		WithType("test.googleapis.com/Book").
		WithPlural("books").
		WithSingular("book").
		WithSelf(msg).
		WithPatterns(ResourcePattern{segProjects}).
		Build()

	got.SetAnnotations("res-codec")

	want := &Resource{
		typez:    "test.googleapis.com/Book",
		plural:   "books",
		singular: "book",
		self:     msg,
		patterns: []ResourcePattern{{segProjects}},
		codec:    "res-codec",
	}
	if diff := cmp.Diff(want, got, allowUnexported()); diff != "" {
		t.Errorf("Resource mismatch (-want +got):\n%s", diff)
	}
	if got.Type() != want.typez || got.Plural() != want.plural || got.Singular() != want.singular || got.Self() != msg || got.Codec() != want.codec {
		t.Errorf("unexpected Resource getters: %+v", got)
	}
	if diff := cmp.Diff(want.patterns, got.Patterns(), allowUnexported()); diff != "" {
		t.Errorf("Patterns() mismatch (-want +got):\n%s", diff)
	}
	got.Patterns()[0][0] = *NewPathSegmentBuilder().WithLiteral("mutated").Build()
	if got.Patterns()[0][0].Literal() != "projects" {
		t.Errorf("Patterns() getter should return a deep defensive copy")
	}
}

func TestResourceSetAnnotationsPanicsWhenAlreadySet(t *testing.T) {
	r := NewResourceBuilder().Build()
	r.SetAnnotations("first")
	defer func() {
		if rec := recover(); rec == nil {
			t.Errorf("expected panic when calling SetAnnotations a second time, got none")
		}
	}()
	r.SetAnnotations("second")
}
