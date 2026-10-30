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

func TestField(t *testing.T) {
	rr := NewResourceReferenceBuilder().WithType("test.googleapis.com/Book").Build()
	rnp := NewResourceNamePattern(NewResourceNameSegment("projects", ""))
	msg := NewTestMessage("Book")
	en := NewEnumBuilder().WithName("State").Build()
	mp := NewTestMap(".test.MapEntry", TypezString, TypezInt32)
	parent := NewTestMessage("Request")
	group := NewTestOneOf("group")

	got := NewFieldBuilder().
		WithName("book").
		WithID(".test.Request.book").
		WithDocumentation("Book field.").
		WithTypez(TypezMessage).
		WithTypezID(".test.Book").
		WithJSONName("book").
		WithOptional(true).
		WithProto2Optional(true).
		WithRepeated(true).
		WithMap(true).
		WithIsOneOf(true).
		WithDeprecated(true).
		WithRecursive(true).
		WithSkipProtoConversion(true).
		WithAutoPopulated(true).
		WithBehavior(FieldBehaviorRequired).
		WithResourceReference(rr).
		WithResourceNamePattern(rnp).
		WithMessageType(msg).
		WithEnumType(en).
		WithMapType(mp).
		WithParent(parent).
		WithGroup(group).
		Build()

	got.SetAnnotations("field-codec")

	want := &Field{
		name:                "book",
		id:                  ".test.Request.book",
		documentation:       "Book field.",
		typez:               TypezMessage,
		typezID:             ".test.Book",
		jsonName:            "book",
		optional:            true,
		proto2Optional:      true,
		repeated:            true,
		mapz:                true,
		isOneOf:             true,
		deprecated:          true,
		recursive:           true,
		skipProtoConversion: true,
		autoPopulated:       true,
		behavior:            []FieldBehavior{FieldBehaviorRequired},
		resourceReference:   rr,
		resourceNamePattern: rnp,
		messageType:         msg,
		enumType:            en,
		mapType:             mp,
		parent:              parent,
		group:               group,
		codec:               "field-codec",
	}
	if diff := cmp.Diff(want, got, allowUnexported()); diff != "" {
		t.Errorf("Field mismatch (-want +got):\n%s", diff)
	}
	if got.Name() != want.name || got.ID() != want.id || got.Documentation() != want.documentation || got.Typez() != want.typez || got.TypezID() != want.typezID || got.JSONName() != want.jsonName {
		t.Errorf("unexpected Field identity: %+v", got)
	}
	if !got.Optional() || !got.Proto2Optional() || !got.Repeated() || !got.Map() || !got.IsOneOf() || !got.Deprecated() || !got.Recursive() || !got.SkipProtoConversion() || !got.AutoPopulated() || !got.DocumentAsRequired() {
		t.Errorf("unexpected Field flags: %+v", got)
	}
	if got.ResourceReference() != rr || got.ResourceNamePattern() != rnp || got.MessageType() != msg || got.EnumType() != en || got.MapType() != mp || got.Parent() != parent || got.Group() != group || got.Codec() != "field-codec" {
		t.Errorf("unexpected Field references: %+v", got)
	}
	if diff := cmp.Diff(want.behavior, got.Behavior()); diff != "" {
		t.Errorf("Behavior() mismatch (-want +got):\n%s", diff)
	}

	got.Behavior()[0] = FieldBehaviorOptional
	if got.Behavior()[0] != FieldBehaviorRequired {
		t.Errorf("Behavior() should return a defensive copy")
	}

	f2 := NewFieldBuilderFrom(got).WithName("other").Build()
	if f2.Name() != "other" || !f2.DocumentAsRequired() {
		t.Errorf("unexpected f2: %+v", f2)
	}
}
