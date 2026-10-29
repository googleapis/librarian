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

func TestPathSegment(t *testing.T) {
	v := NewPathVariableBuilder("name").WithSegments("items", "*").Build()
	gotLiteral := NewPathSegmentBuilder().WithLiteral("v1").Build()
	wantLiteral := &PathSegment{
		literal: "v1",
	}
	if diff := cmp.Diff(wantLiteral, gotLiteral, cmp.AllowUnexported(PathSegment{}, PathVariable{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if gotLiteral.Literal() != wantLiteral.literal {
		t.Errorf("Literal() = %q, want %q", gotLiteral.Literal(), wantLiteral.literal)
	}
	if gotLiteral.Variable() != nil {
		t.Errorf("Variable() = %v, want nil", gotLiteral.Variable())
	}

	gotVar := NewPathSegmentBuilder().WithVariable(v).Build()
	wantVar := &PathSegment{
		variable: v,
	}
	if diff := cmp.Diff(wantVar, gotVar, cmp.AllowUnexported(PathSegment{}, PathVariable{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if gotVar.Literal() != "" {
		t.Errorf("Literal() = %q, want empty", gotVar.Literal())
	}
	if diff := cmp.Diff(wantVar.variable, gotVar.Variable(), cmp.AllowUnexported(PathVariable{})); diff != "" {
		t.Errorf("Variable() mismatch (-want +got):\n%s", diff)
	}
}
