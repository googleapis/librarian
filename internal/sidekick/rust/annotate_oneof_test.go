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

package rust

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestOneOfAnnotations(t *testing.T) {
	singular := &api.Field{
		Name:     "oneof_field",
		JSONName: "oneofField",
		ID:       ".test.Message.oneof_field",
		Typez:    api.TypezString,
		IsOneOf:  true,
	}
	repeated := &api.Field{
		Name:     "oneof_field_repeated",
		JSONName: "oneofFieldRepeated",
		ID:       ".test.Message.oneof_field_repeated",
		Typez:    api.TypezString,
		Repeated: true,
		IsOneOf:  true,
	}
	map_field := &api.Field{
		Name:     "oneof_field_map",
		JSONName: "oneofFieldMap",
		ID:       ".test.Message.oneof_field_map",
		Typez:    api.TypezMessage,
		TypezID:  ".test.$Map",
		Repeated: false,
		IsOneOf:  true,
	}
	integer_field := &api.Field{
		Name:     "oneof_field_integer",
		JSONName: "oneofFieldInteger",
		ID:       ".test.Message.oneof_field_integer",
		Typez:    api.TypezInt64,
		IsOneOf:  true,
	}
	boxed_field := &api.Field{
		Name:     "oneof_field_boxed",
		JSONName: "oneofFieldBoxed",
		ID:       ".test.Message.oneof_field_boxed",
		Typez:    api.TypezMessage,
		TypezID:  ".google.protobuf.DoubleValue",
		Optional: true,
		IsOneOf:  true,
	}

	group := &api.OneOf{
		Name:          "type",
		ID:            ".test.Message.type",
		Documentation: "Say something clever about this oneof.",
		Fields:        []*api.Field{singular, repeated, map_field, integer_field, boxed_field},
	}
	message := &api.Message{
		Name:    "Message",
		ID:      ".test.Message",
		Package: "test",
		Fields:  []*api.Field{singular, repeated, map_field, integer_field, boxed_field},
		OneOfs:  []*api.OneOf{group},
	}
	key_field := &api.Field{Name: "key", Typez: api.TypezInt32}
	value_field := &api.Field{Name: "value", Typez: api.TypezFloat}
	map_message := &api.Message{
		Name:    "$Map",
		ID:      ".test.$Map",
		IsMap:   true,
		Package: "test",
		Fields:  []*api.Field{key_field, value_field},
	}
	model := api.NewTestAPI([]*api.Message{message, map_message}, []*api.Enum{}, []*api.Service{})
	api.CrossReference(model)
	codec := createRustCodec()
	annotateModel(model, codec)

	wantOneOfCodec := &oneOfAnnotation{
		FieldName:           "r#type",
		SetterName:          "type",
		EnumName:            "Type",
		EnumNameInExamples:  "Type",
		QualifiedName:       "crate::model::message::Type",
		RelativeName:        "message::Type",
		StructQualifiedName: "crate::model::Message",
		NameInExamples:      "google_cloud_test::model::message::Type",
		FieldType:           "crate::model::message::Type",
		DocLines:            []string{"/// Say something clever about this oneof."},
	}
	if diff := cmp.Diff(wantOneOfCodec, group.Codec, cmpopts.IgnoreFields(api.OneOf{}, "Codec")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Stops the recursion when comparing fields.
	ignore := cmpopts.IgnoreFields(api.Field{}, "Codec")
	wantFieldCodec := &fieldAnnotations{
		FieldName:          "oneof_field",
		SetterName:         "oneof_field",
		BranchName:         "OneofField",
		FQMessageName:      "crate::model::Message",
		DocLines:           nil,
		FieldType:          "std::string::String",
		PrimitiveFieldType: "std::string::String",
		AddQueryParameter:  `let builder = req.oneof_field().iter().fold(builder, |builder, p| builder.query(&[("oneofField", p)]));`,
		KeyType:            "",
		ValueType:          "",
		OtherFieldsInGroup: []*api.Field{repeated, map_field, integer_field, boxed_field},
	}
	if diff := cmp.Diff(wantFieldCodec, singular.Codec, ignore); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantFieldCodec = &fieldAnnotations{
		FieldName:          "oneof_field_repeated",
		SetterName:         "oneof_field_repeated",
		BranchName:         "OneofFieldRepeated",
		FQMessageName:      "crate::model::Message",
		DocLines:           nil,
		FieldType:          "std::vec::Vec<std::string::String>",
		PrimitiveFieldType: "std::string::String",
		AddQueryParameter:  `let builder = req.oneof_field_repeated().iter().fold(builder, |builder, p| builder.query(&[("oneofFieldRepeated", p)]));`,
		KeyType:            "",
		ValueType:          "",
		OtherFieldsInGroup: []*api.Field{singular, map_field, integer_field, boxed_field},
	}
	if diff := cmp.Diff(wantFieldCodec, repeated.Codec, ignore); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantFieldCodec = &fieldAnnotations{
		FieldName:          "oneof_field_map",
		SetterName:         "oneof_field_map",
		BranchName:         "OneofFieldMap",
		FQMessageName:      "crate::model::Message",
		DocLines:           nil,
		FieldType:          "std::collections::HashMap<i32,f32>",
		PrimitiveFieldType: "std::collections::HashMap<i32,f32>",
		AddQueryParameter:  `let builder = req.oneof_field_map().map(|p| serde_json::to_value(p).map_err(Error::ser) ).transpose()?.into_iter().fold(builder, |builder, p| { use gaxi::query_parameter::QueryParameter; p.add(builder, "oneofFieldMap") });`,
		KeyType:            "i32",
		KeyField:           key_field,
		ValueType:          "f32",
		ValueField:         value_field,
		IsBoxed:            true,
		SerdeAs:            "std::collections::HashMap<wkt::internal::I32, wkt::internal::F32>",
		SkipIfIsDefault:    true,
		OtherFieldsInGroup: []*api.Field{singular, repeated, integer_field, boxed_field},
	}
	if diff := cmp.Diff(wantFieldCodec, map_field.Codec, ignore); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantFieldCodec = &fieldAnnotations{
		FieldName:          "oneof_field_integer",
		SetterName:         "oneof_field_integer",
		BranchName:         "OneofFieldInteger",
		FQMessageName:      "crate::model::Message",
		DocLines:           nil,
		FieldType:          "i64",
		PrimitiveFieldType: "i64",
		AddQueryParameter:  `let builder = req.oneof_field_integer().iter().fold(builder, |builder, p| builder.query(&[("oneofFieldInteger", p)]));`,
		SerdeAs:            "wkt::internal::I64",
		SkipIfIsDefault:    true,
		OtherFieldsInGroup: []*api.Field{singular, repeated, map_field, boxed_field},
	}
	if diff := cmp.Diff(wantFieldCodec, integer_field.Codec, ignore); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantFieldCodec = &fieldAnnotations{
		FieldName:          "oneof_field_boxed",
		SetterName:         "oneof_field_boxed",
		BranchName:         "OneofFieldBoxed",
		FQMessageName:      "crate::model::Message",
		DocLines:           nil,
		FieldType:          "std::boxed::Box<wkt::DoubleValue>",
		MessageType:        boxed_field.MessageType,
		PrimitiveFieldType: "wkt::DoubleValue",
		AddQueryParameter:  `let builder = req.oneof_field_boxed().map(|p| serde_json::to_value(p).map_err(Error::ser) ).transpose()?.into_iter().fold(builder, |builder, p| { use gaxi::query_parameter::QueryParameter; p.add(builder, "oneofFieldBoxed") });`,
		IsBoxed:            true,
		SerdeAs:            "wkt::internal::F64",
		SkipIfIsDefault:    true,
		OtherFieldsInGroup: []*api.Field{singular, repeated, map_field, integer_field},
	}
	if diff := cmp.Diff(wantFieldCodec, boxed_field.Codec, ignore); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestOneOfConflictAnnotations(t *testing.T) {
	singular := &api.Field{
		Name:     "oneof_field",
		JSONName: "oneofField",
		ID:       ".test.Message.oneof_field",
		Typez:    api.TypezString,
		IsOneOf:  true,
	}
	group := &api.OneOf{
		Name:          "nested_thing",
		ID:            ".test.Message.nested_thing",
		Documentation: "Say something clever about this oneof.",
		Fields:        []*api.Field{singular},
	}
	child := &api.Message{
		Name:    "NestedThing",
		ID:      ".test.Message.NestedThing",
		Package: "test",
	}
	message := &api.Message{
		Name:     "Message",
		ID:       ".test.Message",
		Package:  "test",
		Fields:   []*api.Field{singular},
		OneOfs:   []*api.OneOf{group},
		Messages: []*api.Message{child},
	}
	model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{}, []*api.Service{})
	api.CrossReference(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"name-overrides": ".test.Message.nested_thing=NestedThingOneOf",
	})
	annotateModel(model, codec)

	// Stops the recursion when comparing fields.
	ignore := cmpopts.IgnoreFields(api.OneOf{}, "Codec")

	want := &oneOfAnnotation{
		FieldName:           "nested_thing",
		SetterName:          "nested_thing",
		EnumName:            "NestedThingOneOf",
		EnumNameInExamples:  "NestedThingOneOf",
		QualifiedName:       "crate::model::message::NestedThingOneOf",
		RelativeName:        "message::NestedThingOneOf",
		StructQualifiedName: "crate::model::Message",
		NameInExamples:      "google_cloud_test::model::message::NestedThingOneOf",
		FieldType:           "crate::model::message::NestedThingOneOf",
		DocLines:            []string{"/// Say something clever about this oneof."},
	}
	if diff := cmp.Diff(want, group.Codec, ignore); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestOneOfUnqualifiedConflictAnnotations(t *testing.T) {
	singular := &api.Field{
		Name:     "oneof_field",
		JSONName: "oneofField",
		ID:       ".test.Message.oneof_field",
		Typez:    api.TypezString,
		IsOneOf:  true,
	}
	group := &api.OneOf{
		Name:          "message",
		ID:            ".test.Message.message.message",
		Documentation: "Say something clever about this oneof.",
		Fields:        []*api.Field{singular},
	}
	message := &api.Message{
		Name:    "Message",
		ID:      ".test.Message",
		Package: "test",
		Fields:  []*api.Field{singular},
		OneOfs:  []*api.OneOf{group},
	}
	model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{}, []*api.Service{})
	api.CrossReference(model)
	codec := createRustCodec()
	annotateModel(model, codec)

	// Stops the recursion when comparing fields.
	ignore := cmpopts.IgnoreFields(api.OneOf{}, "Codec")

	want := &oneOfAnnotation{
		FieldName:           "message",
		SetterName:          "message",
		EnumName:            "Message",
		QualifiedName:       "crate::model::message::Message",
		RelativeName:        "message::Message",
		StructQualifiedName: "crate::model::Message",
		NameInExamples:      "google_cloud_test::model::message::Message",
		FieldType:           "crate::model::message::Message",
		DocLines:            []string{"/// Say something clever about this oneof."},
		AliasInExamples:     "MessageOneOf",
		EnumNameInExamples:  "MessageOneOf",
	}
	if diff := cmp.Diff(want, group.Codec, ignore); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
