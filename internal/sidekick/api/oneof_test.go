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

func TestOneOf(t *testing.T) {
	f1 := NewFieldBuilder().WithName("f1").WithTypez(TypezMessage).Build()
	f2 := NewFieldBuilder().WithName("f2").WithTypez(TypezString).Build()

	b := NewOneOfBuilder().
		WithName("choice").
		WithID(".test.Msg.choice").
		WithDocumentation("OneOf docs.").
		WithFields(f1).
		AddField(f2).
		WithExampleField(f2).
		WithAnnotations("oneof-codec")
	got := b.Build()
	if got != b.Build() {
		t.Errorf("OneOfBuilder.Build() should preserve pointer identity across calls")
	}

	want := &OneOf{
		name:          "choice",
		id:            ".test.Msg.choice",
		documentation: "OneOf docs.",
		fields:        []*Field{f1, f2},
		exampleField:  f2,
		codec:         "oneof-codec",
	}
	opts := cmp.Options{
		cmp.AllowUnexported(OneOf{}, Field{}),
		cmp.FilterPath(func(p cmp.Path) bool {
			return len(p) > 2 && p.Last().String() == ".group"
		}, cmp.Ignore()),
	}
	if diff := cmp.Diff(want, got, opts); diff != "" {
		t.Errorf("OneOf mismatch (-want +got):\n%s", diff)
	}
	if got.Name() != want.name || got.ID() != want.id || got.Documentation() != want.documentation || got.ExampleField() != f2 || got.Codec() != want.codec {
		t.Errorf("unexpected OneOf getters: %+v", got)
	}
	if diff := cmp.Diff(want.fields, got.Fields(), opts); diff != "" {
		t.Errorf("Fields() mismatch (-want +got):\n%s", diff)
	}

	got.Fields()[0] = nil
	if got.Fields()[0] != f1 {
		t.Errorf("Fields() should return a defensive copy")
	}
}
