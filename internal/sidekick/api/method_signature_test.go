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

func TestMethodSignature(t *testing.T) {
	f1 := NewFieldBuilder().WithName("parent").Build()
	f2 := NewFieldBuilder().WithName("book").Build()

	got := NewMethodSignatureBuilder().
		WithNames("parent", "book").
		WithFields(f1, f2).
		Build()

	want := &MethodSignature{
		names:  []string{"parent", "book"},
		fields: []*Field{f1, f2},
	}
	if diff := cmp.Diff(want, got, allowUnexported()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.names, got.Names()); diff != "" {
		t.Errorf("Names() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.fields, got.Fields(), allowUnexported()); diff != "" {
		t.Errorf("Fields() mismatch (-want +got):\n%s", diff)
	}
	if got.Method() != nil {
		t.Errorf("Method() = %v, want nil", got.Method())
	}

	got.Names()[0] = "mutated"
	got.Fields()[0] = nil
	if got.Names()[0] != "parent" || got.Fields()[0] != f1 {
		t.Errorf("MethodSignature getters should return defensive copies")
	}
}
