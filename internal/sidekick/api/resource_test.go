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

func TestTargetResource(t *testing.T) {
	seg := *NewPathSegmentBuilder().WithLiteral("items").Build()
	got := NewTargetResource([][]string{{"parent"}, {"child"}}, []PathSegment{seg})
	want := &TargetResource{
		fieldPaths: [][]string{{"parent"}, {"child"}},
		template:   []PathSegment{seg},
	}
	opts := cmp.AllowUnexported(TargetResource{}, PathSegment{}, PathVariable{})
	if diff := cmp.Diff(want, got, opts); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.fieldPaths, got.FieldPaths()); diff != "" {
		t.Errorf("FieldPaths() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.template, got.Template(), opts); diff != "" {
		t.Errorf("Template() mismatch (-want +got):\n%s", diff)
	}
}
