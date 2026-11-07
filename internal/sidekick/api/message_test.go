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

func TestMessage(t *testing.T) {
	f1 := NewFieldBuilder().WithName("f1").Build()
	f2 := NewFieldBuilder().WithName("f2").Build()
	skipped := NewFieldBuilder().WithName("skipped").Build()
	e1 := NewEnumBuilder().WithName("E1").Build()
	nested := NewMessageBuilder().WithName("Nested").Build()
	oo := NewOneOfBuilder().WithName("oo").Build()
	parent := NewMessageBuilder().WithName("Parent").Build()
	res := NewResourceBuilder().WithType("test.googleapis.com/Book").Build()
	page := NewPaginationInfo(f1, f2)

	b := NewMessageBuilder().
		WithName("Book").
		WithID(".test.Book").
		WithPackage("test").
		WithDocumentation("Book docs.").
		WithDeprecated(true).
		WithSyntheticRequest(true).
		WithServicePlaceholder(true).
		WithFields(f1).
		AddField(f2).
		WithSkippedFields(skipped).
		WithEnums(e1).
		WithMessages(nested).
		WithOneOfs(oo).
		WithParent(parent).
		WithPagination(page).
		WithResource(res).
		WithAnnotations("msg-codec")

	got := b.Build()
	if got != b.Build() {
		t.Errorf("MessageBuilder.Build() should preserve pointer identity")
	}

	want := &Message{
		name:               "Book",
		id:                 ".test.Book",
		packagez:           "test",
		documentation:      "Book docs.",
		deprecated:         true,
		syntheticRequest:   true,
		servicePlaceholder: true,
		fields:             []*Field{f1, f2},
		skippedFields:      []*Field{skipped},
		enums:              []*Enum{e1},
		messages:           []*Message{nested},
		oneOfs:             []*OneOf{oo},
		parent:             parent,
		pagination:         page,
		resource:           res,
		codec:              "msg-codec",
	}
	opts := cmp.Options{
		allowUnexported(),
		cmp.FilterPath(func(p cmp.Path) bool {
			return len(p) > 2 && p.Last().String() == ".parent"
		}, cmp.Ignore()),
	}
	if diff := cmp.Diff(want, got, opts); diff != "" {
		t.Errorf("Message mismatch (-want +got):\n%s", diff)
	}
	if got.Name() != want.name || got.ID() != want.id || got.Package() != want.packagez || got.Documentation() != want.documentation || !got.Deprecated() || !got.SyntheticRequest() || !got.ServicePlaceholder() {
		t.Errorf("unexpected Message scalar fields: %+v", got)
	}
	if got.Parent() != parent || got.Pagination() != page || got.Resource() != res || got.Codec() != "msg-codec" {
		t.Errorf("unexpected Message pointers: %+v", got)
	}
	if len(got.Fields()) != 2 || got.Fields()[0] != f1 || len(got.SkippedFields()) != 1 || got.SkippedFields()[0] != skipped || len(got.Enums()) != 1 || got.Enums()[0] != e1 || len(got.Messages()) != 1 || got.Messages()[0] != nested || len(got.OneOfs()) != 1 || got.OneOfs()[0] != oo {
		t.Errorf("Message slice inputs were not cloned")
	}

	got.Fields()[0] = nil
	got.SkippedFields()[0] = nil
	got.Enums()[0] = nil
	got.Messages()[0] = nil
	got.OneOfs()[0] = nil
	if got.Fields()[0] != f1 || got.SkippedFields()[0] != skipped || got.Enums()[0] != e1 || got.Messages()[0] != nested || got.OneOfs()[0] != oo {
		t.Errorf("Message slice getters should return defensive copies")
	}

	m2 := NewMessageBuilderFrom(got).WithName("Book2").Build()
	if m2.Name() != "Book2" || len(m2.Fields()) != 2 || len(m2.SkippedFields()) != 1 {
		t.Errorf("unexpected m2: %+v", m2)
	}
}
