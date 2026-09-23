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
	keyField := api.NewTestField("key").WithType(api.TypezInt32)
	valueField := api.NewTestField("value").WithType(api.TypezFloat)
	mapMessage := api.NewTestMessage("$Map").
		WithPackage("test").
		WithFields(keyField, valueField).
		WithIsMap()

	doubleValue := api.NewTestMessage("DoubleValue").WithPackage("google.protobuf")

	singular := api.NewTestField("oneof_field").WithType(api.TypezString)
	repeated := api.NewTestField("oneof_field_repeated").WithType(api.TypezString).WithRepeated()
	mapField := api.NewTestField("oneof_field_map").WithMessageType(mapMessage)
	integerField := api.NewTestField("oneof_field_integer").WithType(api.TypezInt64)
	boxedField := api.NewTestField("oneof_field_boxed").
		WithMessageType(doubleValue).
		WithOptional()

	group := api.NewTestOneOf("type").
		WithDocumentation("Say something clever about this oneof.").
		WithFields(singular, repeated, mapField, integerField, boxedField)
	message := api.NewTestMessage("Message").
		WithOneOfs(group)
	model := api.NewTestAPI([]*api.Message{message, mapMessage}, nil, nil)
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
		ProstRelativeName:   "message::Type",
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
		ProstBranchName:    "OneofField",
		FQMessageName:      "crate::model::Message",
		DocLines:           nil,
		FieldType:          "std::string::String",
		PrimitiveFieldType: "std::string::String",
		AddQueryParameter:  `let builder = req.oneof_field().iter().fold(builder, |builder, p| builder.query(&[("oneofField", p)]));`,
		KeyType:            "",
		ValueType:          "",
		OtherFieldsInGroup: []*api.Field{repeated, mapField, integerField, boxedField},
	}
	if diff := cmp.Diff(wantFieldCodec, singular.Codec, ignore); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantFieldCodec = &fieldAnnotations{
		FieldName:          "oneof_field_repeated",
		SetterName:         "oneof_field_repeated",
		BranchName:         "OneofFieldRepeated",
		ProstBranchName:    "OneofFieldRepeated",
		FQMessageName:      "crate::model::Message",
		DocLines:           nil,
		FieldType:          "std::vec::Vec<std::string::String>",
		PrimitiveFieldType: "std::string::String",
		AddQueryParameter:  `let builder = req.oneof_field_repeated().iter().fold(builder, |builder, p| builder.query(&[("oneofFieldRepeated", p)]));`,
		KeyType:            "",
		ValueType:          "",
		OtherFieldsInGroup: []*api.Field{singular, mapField, integerField, boxedField},
	}
	if diff := cmp.Diff(wantFieldCodec, repeated.Codec, ignore); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantFieldCodec = &fieldAnnotations{
		FieldName:          "oneof_field_map",
		SetterName:         "oneof_field_map",
		BranchName:         "OneofFieldMap",
		ProstBranchName:    "OneofFieldMap",
		FQMessageName:      "crate::model::Message",
		DocLines:           nil,
		FieldType:          "std::collections::HashMap<i32,f32>",
		PrimitiveFieldType: "std::collections::HashMap<i32,f32>",
		AddQueryParameter:  `let builder = req.oneof_field_map().map(|p| serde_json::to_value(p).map_err(Error::ser) ).transpose()?.into_iter().fold(builder, |builder, p| { use gaxi::query_parameter::QueryParameter; p.add(builder, "oneofFieldMap") });`,
		KeyType:            "i32",
		KeyField:           keyField,
		ValueType:          "f32",
		ValueField:         valueField,
		IsBoxed:            true,
		SerdeAs:            "std::collections::HashMap<wkt::internal::I32, wkt::internal::F32>",
		SkipIfIsDefault:    true,
		OtherFieldsInGroup: []*api.Field{singular, repeated, integerField, boxedField},
	}
	if diff := cmp.Diff(wantFieldCodec, mapField.Codec, ignore); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantFieldCodec = &fieldAnnotations{
		FieldName:          "oneof_field_integer",
		SetterName:         "oneof_field_integer",
		BranchName:         "OneofFieldInteger",
		ProstBranchName:    "OneofFieldInteger",
		FQMessageName:      "crate::model::Message",
		DocLines:           nil,
		FieldType:          "i64",
		PrimitiveFieldType: "i64",
		AddQueryParameter:  `let builder = req.oneof_field_integer().iter().fold(builder, |builder, p| builder.query(&[("oneofFieldInteger", p)]));`,
		SerdeAs:            "wkt::internal::I64",
		SkipIfIsDefault:    true,
		OtherFieldsInGroup: []*api.Field{singular, repeated, mapField, boxedField},
	}
	if diff := cmp.Diff(wantFieldCodec, integerField.Codec, ignore); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantFieldCodec = &fieldAnnotations{
		FieldName:          "oneof_field_boxed",
		SetterName:         "oneof_field_boxed",
		BranchName:         "OneofFieldBoxed",
		ProstBranchName:    "OneofFieldBoxed",
		FQMessageName:      "crate::model::Message",
		DocLines:           nil,
		FieldType:          "std::boxed::Box<wkt::DoubleValue>",
		MessageType:        boxedField.MessageType,
		PrimitiveFieldType: "wkt::DoubleValue",
		AddQueryParameter:  `let builder = req.oneof_field_boxed().map(|p| serde_json::to_value(p).map_err(Error::ser) ).transpose()?.into_iter().fold(builder, |builder, p| { use gaxi::query_parameter::QueryParameter; p.add(builder, "oneofFieldBoxed") });`,
		IsBoxed:            true,
		SerdeAs:            "wkt::internal::F64",
		SkipIfIsDefault:    true,
		OtherFieldsInGroup: []*api.Field{singular, repeated, mapField, integerField},
	}
	if diff := cmp.Diff(wantFieldCodec, boxedField.Codec, ignore); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestOneOfConflictAnnotations(t *testing.T) {
	singular := api.NewTestField("oneof_field").WithType(api.TypezString)
	group := api.NewTestOneOf("nested_thing").
		WithDocumentation("Say something clever about this oneof.").
		WithFields(singular)
	child := api.NewTestMessage("NestedThing")
	message := api.NewTestMessage("Message").
		WithOneOfs(group).
		WithMessages(child)

	model := api.NewTestAPI([]*api.Message{message}, nil, nil)
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
		ProstRelativeName:   "message::NestedThingOneOf",
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
	singular := api.NewTestField("oneof_field").WithType(api.TypezString)
	group := api.NewTestOneOf("message").
		WithDocumentation("Say something clever about this oneof.").
		WithFields(singular)
	message := api.NewTestMessage("Message").WithOneOfs(group)
	model := api.NewTestAPI([]*api.Message{message}, nil, nil)
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
		ProstRelativeName:   "message::Message",
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
