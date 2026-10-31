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

func TestPathTemplate(t *testing.T) {
	s0 := *NewPathSegmentBuilder().WithLiteral("v1").Build()
	v1 := NewPathVariableBuilder("parent", "child").
		WithLiteral("projects").
		WithMatch().
		WithAllowReserved().
		WithLiteral("locations").
		WithMatchRecursive().
		Build()
	s1 := *NewPathSegmentBuilder().WithVariable(v1).Build()
	got := NewPathTemplateBuilder().
		WithSegments(s0).
		WithSegment(s1).
		WithLiteral("zones").
		WithVariable(NewPathVariableBuilder("zone").WithSegments("*").Build()).
		WithVariableNamed("v2", "field").
		WithVerb("verb").
		Build()
	want := &PathTemplate{
		segments: []PathSegment{
			s0,
			s1,
			*NewPathSegmentBuilder().WithLiteral("zones").Build(),
			*NewPathSegmentBuilder().WithVariable(NewPathVariableBuilder("zone").WithSegments("*").Build()).Build(),
			*NewPathSegmentBuilder().WithVariable(NewPathVariableBuilder("v2", "field").WithSegments("*").Build()).Build(),
		},
		verb: "verb",
	}
	if diff := cmp.Diff(want, got, allowUnexported()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.segments, got.Segments(), allowUnexported()); diff != "" {
		t.Errorf("Segments() mismatch (-want +got):\n%s", diff)
	}
	if got.Verb() != want.verb {
		t.Errorf("Verb() = %q, want %q", got.Verb(), want.verb)
	}
}
