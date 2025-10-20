// Copyright 2025 Google LLC
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
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/internal/api"
)

func TestFieldAnnotations(t *testing.T) {
	key_field := &api.Field{Name: "key", Typez: api.INT32_TYPE}
	value_field := &api.Field{Name: "value", Typez: api.INT64_TYPE}
	map_message := &api.Message{
		Name:    "$Map",
		ID:      ".test.v1.$Map",
		IsMap:   true,
		Package: "test.v1",
		Fields:  []*api.Field{key_field, value_field},
	}
	singular_field := &api.Field{
		Name:     "singular_field",
		JSONName: "singularField",
		ID:       ".test.v1.Message.singular_field",
		Typez:    api.STRING_TYPE,
	}
	repeated_field := &api.Field{
		Name:     "repeated_field",
		JSONName: "repeatedField",
		ID:       ".test.v1.Message.repeated_field",
		Typez:    api.STRING_TYPE,
		Repeated: true,
	}
	map_field := &api.Field{
		Name:     "map_field",
		JSONName: "mapField",
		ID:       ".test.v1.Message.map_field",
		Typez:    api.MESSAGE_TYPE,
		TypezID:  ".test.v1.$Map",
		Repeated: false,
	}
	boxed_field := &api.Field{
		Name:     "boxed_field",
		JSONName: "boxedField",
		ID:       ".test.v1.Message.boxed_field",
		Typez:    api.MESSAGE_TYPE,
		TypezID:  ".test.v1.TestMessage",
		Optional: true,
	}
	message := &api.Message{
		Name:          "TestMessage",
		Package:       "test.v1",
		ID:            ".test.v1.TestMessage",
		Documentation: "A test message.",
		Fields:        []*api.Field{singular_field, repeated_field, map_field, boxed_field},
	}

	model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{}, []*api.Service{})
	model.State.MessageByID[map_message.ID] = map_message
	api.CrossReference(model)
	api.LabelRecursiveFields(model)
	codec, err := newCodec("protobuf", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)
	wantMessage := &messageAnnotation{
		Name:              "TestMessage",
		ModuleName:        "test_message",
		QualifiedName:     "crate::model::TestMessage",
		RelativeName:      "TestMessage",
		NameInExamples:    "google_cloud_test_v1::model::TestMessage",
		PackageModuleName: "test::v1",
		SourceFQN:         "test.v1.TestMessage",
		DocLines:          []string{"/// A test message."},
		BasicFields:       []*api.Field{singular_field, repeated_field, map_field, boxed_field},
	}
	// We ignore the Parent.Codec and MessageType.Codec fields of Fields,
	// as those point to the message annotations itself and was causing
	// the test to fail because of cyclic dependencies.
	if diff := cmp.Diff(wantMessage, message.Codec, cmpopts.IgnoreFields(api.Field{}, "Parent.Codec", "MessageType.Codec")); diff != "" {
		t.Errorf("mismatch in message annotations (-want, +got)\n:%s", diff)
	}

	wantField := &fieldAnnotations{
		FieldName:          "singular_field",
		SetterName:         "singular_field",
		BranchName:         "SingularField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "std::string::String",
		PrimitiveFieldType: "std::string::String",
		AddQueryParameter:  `let builder = builder.query(&[("singularField", &req.singular_field)]);`,
	}
	if diff := cmp.Diff(wantField, singular_field.Codec); diff != "" {
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
	}

	wantField = &fieldAnnotations{
		FieldName:          "repeated_field",
		SetterName:         "repeated_field",
		BranchName:         "RepeatedField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "std::vec::Vec<std::string::String>",
		PrimitiveFieldType: "std::string::String",
		AddQueryParameter:  `let builder = req.repeated_field.iter().fold(builder, |builder, p| builder.query(&[("repeatedField", p)]));`,
	}
	if diff := cmp.Diff(wantField, repeated_field.Codec); diff != "" {
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
	}

	wantField = &fieldAnnotations{
		FieldName:          "map_field",
		SetterName:         "map_field",
		BranchName:         "MapField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "std::collections::HashMap<i32,i64>",
		PrimitiveFieldType: "std::collections::HashMap<i32,i64>",
		AddQueryParameter:  `let builder = { use gaxi::query_parameter::QueryParameter; serde_json::to_value(&req.map_field).map_err(Error::ser)?.add(builder, "mapField") };`,
		KeyType:            "i32",
		KeyField:           key_field,
		ValueType:          "i64",
		ValueField:         value_field,
		SerdeAs:            "std::collections::HashMap<wkt::internal::I32, wkt::internal::I64>",
		SkipIfIsDefault:    true,
	}
	if diff := cmp.Diff(wantField, map_field.Codec); diff != "" {
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
	}

	wantField = &fieldAnnotations{
		FieldName:          "boxed_field",
		SetterName:         "boxed_field",
		BranchName:         "BoxedField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "std::option::Option<std::boxed::Box<crate::model::TestMessage>>",
		PrimitiveFieldType: "crate::model::TestMessage",
		AddQueryParameter:  `let builder = req.boxed_field.as_ref().map(|p| serde_json::to_value(p).map_err(Error::ser) ).transpose()?.into_iter().fold(builder, |builder, v| { use gaxi::query_parameter::QueryParameter; v.add(builder, "boxedField") });`,
		IsBoxed:            true,
		SkipIfIsDefault:    true,
	}
	if diff := cmp.Diff(wantField, boxed_field.Codec); diff != "" {
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
	}
}

func TestPrimitiveFieldAnnotations(t *testing.T) {
	for _, test := range []struct {
		wantType    string
		wantSerdeAs string
		typez       api.Typez
	}{
		{"i32", "wkt::internal::I32", api.INT32_TYPE},
		{"i32", "wkt::internal::I32", api.SFIXED32_TYPE},
		{"i32", "wkt::internal::I32", api.SINT32_TYPE},
		{"i64", "wkt::internal::I64", api.INT64_TYPE},
		{"i64", "wkt::internal::I64", api.SFIXED64_TYPE},
		{"i64", "wkt::internal::I64", api.SINT64_TYPE},
		{"u32", "wkt::internal::U32", api.UINT32_TYPE},
		{"u32", "wkt::internal::U32", api.FIXED32_TYPE},
		{"u64", "wkt::internal::U64", api.UINT64_TYPE},
		{"u64", "wkt::internal::U64", api.FIXED64_TYPE},
		{"f32", "wkt::internal::F32", api.FLOAT_TYPE},
		{"f64", "wkt::internal::F64", api.DOUBLE_TYPE},
	} {
		singular_field := &api.Field{
			Name:     "singular_field",
			JSONName: "singularField",
			ID:       ".test.Message.singular_field",
			Typez:    test.typez,
		}
		message := &api.Message{
			Name:          "TestMessage",
			Package:       "test",
			ID:            ".test.TestMessage",
			Documentation: "A test message.",
			Fields:        []*api.Field{singular_field},
		}
		model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{}, []*api.Service{})
		api.CrossReference(model)
		api.LabelRecursiveFields(model)
		codec, err := newCodec("protobuf", map[string]string{})
		if err != nil {
			t.Fatal(err)
		}
		annotateModel(model, codec)

		wantField := &fieldAnnotations{
			FieldName:          "singular_field",
			SetterName:         "singular_field",
			BranchName:         "SingularField",
			FQMessageName:      "crate::model::TestMessage",
			FieldType:          test.wantType,
			PrimitiveFieldType: test.wantType,
			SerdeAs:            test.wantSerdeAs,
			AddQueryParameter:  `let builder = builder.query(&[("singularField", &req.singular_field)]);`,
			SkipIfIsDefault:    true,
		}
		if diff := cmp.Diff(wantField, singular_field.Codec); diff != "" {
			t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
		}

	}
}

func TestBytesAnnotations(t *testing.T) {
	for _, test := range []struct {
		sourceSpecification string
		wantType            string
		wantSerdeAs         string
	}{
		{"protobuf", "::bytes::Bytes", "serde_with::base64::Base64"},
		{"openapi", "::bytes::Bytes", "serde_with::base64::Base64"},
		{"disco", "::bytes::Bytes", "serde_with::base64::Base64<serde_with::base64::UrlSafe>"},
	} {
		singular_field := &api.Field{
			Name:     "singular_field",
			JSONName: "singularField",
			ID:       ".test.Message.singular_field",
			Typez:    api.BYTES_TYPE,
			TypezID:  "bytes",
		}
		message := &api.Message{
			Name:          "TestMessage",
			Package:       "test",
			ID:            ".test.TestMessage",
			Documentation: "A test message.",
			Fields:        []*api.Field{singular_field},
		}
		model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{}, []*api.Service{})
		api.CrossReference(model)
		api.LabelRecursiveFields(model)
		codec, err := newCodec(test.sourceSpecification, map[string]string{})
		if err != nil {
			t.Fatal(err)
		}
		annotateModel(model, codec)

		wantField := &fieldAnnotations{
			FieldName:          "singular_field",
			SetterName:         "singular_field",
			BranchName:         "SingularField",
			FQMessageName:      "crate::model::TestMessage",
			FieldType:          test.wantType,
			PrimitiveFieldType: test.wantType,
			SerdeAs:            test.wantSerdeAs,
			AddQueryParameter:  `let builder = builder.query(&[("singularField", &req.singular_field)]);`,
		}
		if diff := cmp.Diff(wantField, singular_field.Codec); diff != "" {
			t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
		}
	}
}

func TestWrapperFieldAnnotations(t *testing.T) {
	for _, test := range []struct {
		wantType    string
		wantSerdeAs string
		typezID     string
	}{
		{"wkt::BytesValue", "serde_with::base64::Base64", ".google.protobuf.BytesValue"},
		{"wkt::UInt64Value", "wkt::internal::U64", ".google.protobuf.UInt64Value"},
		{"wkt::Int64Value", "wkt::internal::I64", ".google.protobuf.Int64Value"},
		{"wkt::UInt32Value", "wkt::internal::U32", ".google.protobuf.UInt32Value"},
		{"wkt::Int32Value", "wkt::internal::I32", ".google.protobuf.Int32Value"},
		{"wkt::FloatValue", "wkt::internal::F32", ".google.protobuf.FloatValue"},
		{"wkt::DoubleValue", "wkt::internal::F64", ".google.protobuf.DoubleValue"},
		{"wkt::BoolValue", "", ".google.protobuf.BoolValue"},
	} {
		singular_field := &api.Field{
			Name:     "singular_field",
			JSONName: "singularField",
			ID:       ".test.Message.singular_field",
			Typez:    api.MESSAGE_TYPE,
			TypezID:  test.typezID,
			Optional: true,
		}
		message := &api.Message{
			Name:          "TestMessage",
			Package:       "test",
			ID:            ".test.TestMessage",
			Documentation: "A test message.",
			Fields:        []*api.Field{singular_field},
		}
		model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{}, []*api.Service{})
		api.CrossReference(model)
		api.LabelRecursiveFields(model)
		codec := createRustCodec()
		annotateModel(model, codec)

		wantField := &fieldAnnotations{
			FieldName:          "singular_field",
			SetterName:         "singular_field",
			BranchName:         "SingularField",
			FQMessageName:      "crate::model::TestMessage",
			FieldType:          fmt.Sprintf("std::option::Option<%s>", test.wantType),
			PrimitiveFieldType: test.wantType,
			SerdeAs:            test.wantSerdeAs,
			SkipIfIsDefault:    true,
		}
		if diff := cmp.Diff(wantField, singular_field.Codec, cmpopts.IgnoreFields(fieldAnnotations{}, "AddQueryParameter")); diff != "" {
			t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
		}

	}
}

func TestEnumFieldAnnotations(t *testing.T) {
	enumz := &api.Enum{
		Name:    "TestEnum",
		Package: "test.v1",
		ID:      ".test.v1.TestEnum",
	}
	singular_field := &api.Field{
		Name:     "singular_field",
		JSONName: "singularField",
		ID:       ".test.v1.Message.singular_field",
		Typez:    api.ENUM_TYPE,
		TypezID:  ".test.v1.TestEnum",
	}
	repeated_field := &api.Field{
		Name:     "repeated_field",
		JSONName: "repeatedField",
		ID:       ".test.v1.Message.repeated_field",
		Typez:    api.ENUM_TYPE,
		TypezID:  ".test.v1.TestEnum",
		Repeated: true,
	}
	optional_field := &api.Field{
		Name:     "optional_field",
		JSONName: "optionalField",
		ID:       ".test.v1.Message.optional_field",
		Typez:    api.ENUM_TYPE,
		TypezID:  ".test.v1.TestEnum",
		Optional: true,
	}
	null_value_field := &api.Field{
		Name:     "null_value_field",
		JSONName: "nullValueField",
		ID:       ".test.v1.Message.null_value_field",
		Typez:    api.ENUM_TYPE,
		TypezID:  ".google.protobuf.NullValue",
	}
	map_field := &api.Field{
		Name:     "map_field",
		JSONName: "mapField",
		ID:       ".test.v1.Message.map_field",
		Typez:    api.MESSAGE_TYPE,
		TypezID:  "$map<string, .test.v1.TestEnum>",
	}
	// TODO(#1381) - this is closer to what map message should be called.
	key_field := &api.Field{
		Name:     "key",
		JSONName: "key",
		ID:       "$map<string, .test.v1.TestEnum>.key",
		Typez:    api.STRING_TYPE,
	}
	value_field := &api.Field{
		Name:     "value",
		JSONName: "value",
		ID:       "$map<string, .test.v1.TestEnum>.value",
		Typez:    api.ENUM_TYPE,
		TypezID:  ".test.v1.TestEnum",
	}
	map_message := &api.Message{
		Name:    "$map<string, .test.v1.TestEnum>",
		ID:      "$map<string, .test.v1.TestEnum>",
		Package: "test.v1",
		IsMap:   true,
		Fields:  []*api.Field{key_field, value_field},
	}
	message := &api.Message{
		Name:          "TestMessage",
		Package:       "test.v1",
		ID:            ".test.v1.TestMessage",
		Documentation: "A test message.",
		Fields:        []*api.Field{singular_field, repeated_field, optional_field, null_value_field, map_field},
	}

	model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{enumz}, []*api.Service{})
	model.State.MessageByID[map_message.ID] = map_message
	api.CrossReference(model)
	api.LabelRecursiveFields(model)
	codec, err := newCodec("protobuf", map[string]string{
		"package:wkt": "force-used=true,package=google-cloud-wkt,source=google.protobuf",
	})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)
	wantMessage := &messageAnnotation{
		Name:              "TestMessage",
		ModuleName:        "test_message",
		QualifiedName:     "crate::model::TestMessage",
		RelativeName:      "TestMessage",
		NameInExamples:    "google_cloud_test_v1::model::TestMessage",
		PackageModuleName: "test::v1",
		SourceFQN:         "test.v1.TestMessage",
		DocLines:          []string{"/// A test message."},
		BasicFields:       []*api.Field{singular_field, repeated_field, optional_field, null_value_field, map_field},
	}
	// We ignore the Parent.Codec field of Fields, as that points to the message annotations itself and was causing
	// the test to fail because of cyclic dependencies.
	if diff := cmp.Diff(wantMessage, message.Codec, cmpopts.IgnoreFields(api.Field{}, "Parent.Codec")); diff != "" {
		t.Errorf("mismatch in message annotations (-want, +got)\n:%s", diff)
	}

	wantField := &fieldAnnotations{
		FieldName:          "singular_field",
		SetterName:         "singular_field",
		BranchName:         "SingularField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "crate::model::TestEnum",
		PrimitiveFieldType: "crate::model::TestEnum",
		AddQueryParameter:  `let builder = builder.query(&[("singularField", &req.singular_field)]);`,
		SkipIfIsDefault:    true,
	}
	if diff := cmp.Diff(wantField, singular_field.Codec); diff != "" {
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
	}

	wantField = &fieldAnnotations{
		FieldName:          "repeated_field",
		SetterName:         "repeated_field",
		BranchName:         "RepeatedField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "std::vec::Vec<crate::model::TestEnum>",
		PrimitiveFieldType: "crate::model::TestEnum",
		AddQueryParameter:  `let builder = req.repeated_field.iter().fold(builder, |builder, p| builder.query(&[("repeatedField", p)]));`,
		SkipIfIsDefault:    true,
	}
	if diff := cmp.Diff(wantField, repeated_field.Codec); diff != "" {
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
	}

	wantField = &fieldAnnotations{
		FieldName:          "optional_field",
		SetterName:         "optional_field",
		BranchName:         "OptionalField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "std::option::Option<crate::model::TestEnum>",
		PrimitiveFieldType: "crate::model::TestEnum",
		AddQueryParameter:  `let builder = req.optional_field.iter().fold(builder, |builder, p| builder.query(&[("optionalField", p)]));`,
		SkipIfIsDefault:    true,
	}
	if diff := cmp.Diff(wantField, optional_field.Codec); diff != "" {
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
	}

	// In the .proto specification this is represented as an enum. Which we
	// map to a unit struct.
	wantField = &fieldAnnotations{
		FieldName:          "null_value_field",
		SetterName:         "null_value_field",
		BranchName:         "NullValueField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "wkt::NullValue",
		PrimitiveFieldType: "wkt::NullValue",
		AddQueryParameter:  `let builder = builder.query(&[("nullValueField", &req.null_value_field)]);`,
		SkipIfIsDefault:    true,
		IsWktNullValue:     true,
	}
	if diff := cmp.Diff(wantField, null_value_field.Codec); diff != "" {
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
	}

	wantField = &fieldAnnotations{
		FieldName:          "map_field",
		SetterName:         "map_field",
		BranchName:         "MapField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "std::collections::HashMap<std::string::String,crate::model::TestEnum>",
		PrimitiveFieldType: "std::collections::HashMap<std::string::String,crate::model::TestEnum>",
		AddQueryParameter:  `let builder = { use gaxi::query_parameter::QueryParameter; serde_json::to_value(&req.map_field).map_err(Error::ser)?.add(builder, "mapField") };`,
		KeyType:            "std::string::String",
		KeyField:           key_field,
		ValueType:          "crate::model::TestEnum",
		ValueField:         value_field,
		SkipIfIsDefault:    true,
	}
	if diff := cmp.Diff(wantField, map_field.Codec); diff != "" {
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
	}
}
