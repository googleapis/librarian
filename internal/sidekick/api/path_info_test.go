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

func TestPathInfo(t *testing.T) {
	b1 := NewPathBindingBuilder().WithVerb("GET").Build()
	b2 := NewPathBindingBuilder().WithVerb("POST").Build()
	got := NewPathInfoBuilder().
		WithBindings(b1).
		WithBinding(b2).
		WithBodyFieldPath("body").
		Build()
	got.SetAnnotations("pi-codec")
	want := &PathInfo{
		bindings:      []*PathBinding{b1, b2},
		bodyFieldPath: "body",
		codec:         "pi-codec",
	}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(PathInfo{}, PathBinding{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.bindings, got.Bindings(), cmp.AllowUnexported(PathBinding{})); diff != "" {
		t.Errorf("Bindings() mismatch (-want +got):\n%s", diff)
	}
	if got.BodyFieldPath() != "body" {
		t.Errorf("BodyFieldPath() = %q, want %q", got.BodyFieldPath(), "body")
	}
	if got.Codec() != "pi-codec" {
		t.Errorf("Codec() = %v, want %q", got.Codec(), "pi-codec")
	}
}
