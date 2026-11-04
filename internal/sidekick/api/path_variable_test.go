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

func TestPathVariable(t *testing.T) {
	got := NewPathVariableBuilder().
		WithFieldPath("parent", "child").
		WithLiteral("projects").
		WithMatch().
		WithAllowReserved().
		WithSegments("locations").
		WithMatchRecursive().
		Build()
	want := &PathVariable{
		fieldPath:     []string{"parent", "child"},
		segments:      []string{"projects", "*", "locations", "**"},
		allowReserved: true,
	}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(PathVariable{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.fieldPath, got.FieldPath()); diff != "" {
		t.Errorf("FieldPath() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.segments, got.Segments()); diff != "" {
		t.Errorf("Segments() mismatch (-want +got):\n%s", diff)
	}
	if got.AllowReserved() != want.allowReserved {
		t.Errorf("AllowReserved() = %v, want %v", got.AllowReserved(), want.allowReserved)
	}
}
