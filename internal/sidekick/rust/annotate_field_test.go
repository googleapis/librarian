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
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func newTestCodec(t *testing.T, specificationFormat, packageName string, options map[string]string) *codec {
	t.Helper()
	codec, err := newCodec(specificationFormat, options)
	if err != nil {
		t.Fatal(err)
	}
	codec.packageMapping = map[string]*packagez{
		"google.protobuf": {name: "wkt"},
	}
	if packageName != "" {
		codec.packageMapping[packageName] = &packagez{name: "external-rust-pkg"}
	}
	return codec
}

func TestFieldAnnotations(t *testing.T) {
	keyField := api.NewTestField("key").WithType(api.TypezInt32)
	valueField := api.NewTestField("value").WithType(api.TypezInt64)
	mapMessage := api.NewTestMessage("$Map").
		WithPackage("test.v1").
		WithFields(keyField, valueField).
		WithIsMap()

	message := api.NewTestMessage("TestMessage").
		WithPackage("test.v1").
		WithDocumentation("A test message.")

	singularField := api.NewTestField("singular_field").WithType(api.TypezString)
	repeatedField := api.NewTestField("repeated_field").WithType(api.TypezString).WithRepeated()
	mapField := api.NewTestField("map_field").WithMessageType(mapMessage)
	boxedField := api.NewTestField("boxed_field").
		WithMessageType(message).
		WithOptional()
	message.WithFields(singularField, repeatedField, mapField, boxedField)

	model := api.NewTestAPI([]*api.Message{message, mapMessage}, nil, nil)
	api.CrossReference(model)
	api.LabelRecursiveFields(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "test", map[string]string{})
	annotateModel(model, codec)
	wantMessage := &messageAnnotation{
		Name:              "TestMessage",
		ModuleName:        "test_message",
		QualifiedName:     "crate::model::TestMessage",
		RelativeName:      "TestMessage",
		ProstRelativeName: "TestMessage",
		NameInExamples:    "google_cloud_test_v1::model::TestMessage",
		PackageModuleName: "test::v1",
		SourceFQN:         "test.v1.TestMessage",
		DocLines:          []string{"/// A test message."},
		BasicFields:       []*api.Field{singularField, repeatedField, mapField, boxedField},
	}
	// We ignore the Parent.Codec and MessageType.Codec fields of Fields,
	// as those point to the message annotations itself and was causing
	// the test to fail because of cyclic dependencies.
	if diff := cmp.Diff(wantMessage, message.Codec, cmpopts.IgnoreFields(api.Field{}, "Parent.Codec", "MessageType.Codec", "Codec")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantField := &fieldAnnotations{
		FieldName:          "singular_field",
		SetterName:         "singular_field",
		BranchName:         "SingularField",
		ProstBranchName:    "SingularField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "std::string::String",
		PrimitiveFieldType: "std::string::String",
		AddQueryParameter:  `let builder = builder.query(&[("singularField", &req.singular_field)]);`,
	}
	if diff := cmp.Diff(wantField, singularField.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantMessageNameInExamples := ""
	gotFA, _ := singularField.Codec.(*fieldAnnotations)
	gotMessageNameInExamples := gotFA.MessageNameInExamples()
	if wantMessageNameInExamples != gotMessageNameInExamples {
		t.Errorf("mismatch in MessageNameInExamples, want %s, got %s", wantMessageNameInExamples, gotMessageNameInExamples)
	}

	wantField = &fieldAnnotations{
		FieldName:          "repeated_field",
		SetterName:         "repeated_field",
		BranchName:         "RepeatedField",
		ProstBranchName:    "RepeatedField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "std::vec::Vec<std::string::String>",
		PrimitiveFieldType: "std::string::String",
		AddQueryParameter:  `let builder = req.repeated_field.iter().fold(builder, |builder, p| builder.query(&[("repeatedField", p)]));`,
	}
	if diff := cmp.Diff(wantField, repeatedField.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantMessageNameInExamples = ""
	gotFA, _ = repeatedField.Codec.(*fieldAnnotations)
	gotMessageNameInExamples = gotFA.MessageNameInExamples()
	if wantMessageNameInExamples != gotMessageNameInExamples {
		t.Errorf("mismatch in MessageNameInExamples, want %s, got %s", wantMessageNameInExamples, gotMessageNameInExamples)
	}

	wantField = &fieldAnnotations{
		FieldName:          "map_field",
		SetterName:         "map_field",
		BranchName:         "MapField",
		ProstBranchName:    "MapField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "std::collections::HashMap<i32,i64>",
		PrimitiveFieldType: "std::collections::HashMap<i32,i64>",
		AddQueryParameter:  `let builder = { use gaxi::query_parameter::QueryParameter; serde_json::to_value(&req.map_field).map_err(Error::ser)?.add(builder, "mapField") };`,
		KeyType:            "i32",
		KeyField:           keyField,
		ValueType:          "i64",
		ValueField:         valueField,
		SerdeAs:            "std::collections::HashMap<wkt::internal::I32, wkt::internal::I64>",
		SkipIfIsDefault:    true,
	}
	if diff := cmp.Diff(wantField, mapField.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantMessageNameInExamples = ""
	gotFA, _ = mapField.Codec.(*fieldAnnotations)
	gotMessageNameInExamples = gotFA.MessageNameInExamples()
	if wantMessageNameInExamples != gotMessageNameInExamples {
		t.Errorf("mismatch in MessageNameInExamples, want %s, got %s", wantMessageNameInExamples, gotMessageNameInExamples)
	}

	wantField = &fieldAnnotations{
		FieldName:             "boxed_field",
		SetterName:            "boxed_field",
		BranchName:            "BoxedField",
		ProstBranchName:       "BoxedField",
		FQMessageName:         "crate::model::TestMessage",
		FieldType:             "std::option::Option<std::boxed::Box<crate::model::TestMessage>>",
		MessageType:           message,
		PrimitiveFieldType:    "crate::model::TestMessage",
		AddQueryParameter:     `let builder = req.boxed_field.as_ref().map(|p| serde_json::to_value(p).map_err(Error::ser) ).transpose()?.into_iter().fold(builder, |builder, v| { use gaxi::query_parameter::QueryParameter; v.add(builder, "boxedField") });`,
		IsBoxed:               true,
		MapToBoxed:            true,
		SkipIfIsDefault:       true,
		FieldTypeIsParentType: true,
	}
	if diff := cmp.Diff(wantField, boxedField.Codec, cmpopts.IgnoreFields(api.Field{}, "Codec")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantMessageNameInExamples = "TestMessage"
	gotFA, _ = boxedField.Codec.(*fieldAnnotations)
	gotMessageNameInExamples = gotFA.MessageNameInExamples()
	if wantMessageNameInExamples != gotMessageNameInExamples {
		t.Errorf("mismatch in MessageNameInExamples, want %s, got %s", wantMessageNameInExamples, gotMessageNameInExamples)
	}
}

func TestRecursiveFieldAnnotations(t *testing.T) {
	message := api.NewTestMessage("TestMessage").
		WithPackage("test.v1").
		WithDocumentation("A test message.")

	keyField := api.NewTestField("key").WithType(api.TypezInt32)
	valueField := api.NewTestField("value").WithMessageType(message)
	mapMessage := api.NewTestMessage("$Map").
		WithPackage("test.v1").
		WithFields(keyField, valueField).
		WithIsMap()

	mapField := api.NewTestField("map_field").WithMessageType(mapMessage)
	oneOfField := api.NewTestField("oneof_field").WithMessageType(message)
	group := api.NewTestOneOf("oneof_type").WithFields(oneOfField)
	repeatedField := api.NewTestField("repeated_field").
		WithMessageType(message).
		WithRepeated()
	messageField := api.NewTestField("message_field").WithMessageType(message)

	message.WithFields(mapField).
		WithOneOfs(group).
		WithFields(repeatedField, messageField)

	model := api.NewTestAPI([]*api.Message{message, mapMessage}, nil, nil)
	api.CrossReference(model)
	api.LabelRecursiveFields(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "test", map[string]string{})
	annotateModel(model, codec)
	wantMessage := &messageAnnotation{
		Name:              "TestMessage",
		ModuleName:        "test_message",
		QualifiedName:     "crate::model::TestMessage",
		RelativeName:      "TestMessage",
		ProstRelativeName: "TestMessage",
		NameInExamples:    "google_cloud_test_v1::model::TestMessage",
		PackageModuleName: "test::v1",
		SourceFQN:         "test.v1.TestMessage",
		HasNestedTypes:    true,
		DocLines:          []string{"/// A test message."},
		BasicFields:       []*api.Field{mapField, repeatedField, messageField},
	}
	// We ignore the Parent.Codec and MessageType.Codec fields of Fields,
	// as those point to the message annotations itself and was causing
	// the test to fail because of cyclic dependencies.
	if diff := cmp.Diff(wantMessage, message.Codec, cmpopts.IgnoreFields(api.Field{}, "Parent.Codec", "MessageType.Codec", "Codec")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantField := &fieldAnnotations{
		FieldName:             "map_field",
		SetterName:            "map_field",
		BranchName:            "MapField",
		ProstBranchName:       "MapField",
		FQMessageName:         "crate::model::TestMessage",
		FieldType:             "std::collections::HashMap<i32,crate::model::TestMessage>",
		PrimitiveFieldType:    "std::collections::HashMap<i32,crate::model::TestMessage>",
		AddQueryParameter:     `let builder = { use gaxi::query_parameter::QueryParameter; serde_json::to_value(&req.map_field).map_err(Error::ser)?.add(builder, "mapField") };`,
		KeyType:               "i32",
		KeyField:              keyField,
		ValueType:             "crate::model::TestMessage",
		ValueField:            valueField,
		SerdeAs:               "std::collections::HashMap<wkt::internal::I32, serde_with::Same>",
		IsBoxed:               true,
		MapToBoxed:            true,
		SkipIfIsDefault:       true,
		FieldTypeIsParentType: true,
	}
	if diff := cmp.Diff(wantField, mapField.Codec, cmpopts.IgnoreFields(api.Field{}, "Codec")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantMessageNameInExamples := "TestMessage"
	gotFA, _ := mapField.Codec.(*fieldAnnotations)
	gotMessageNameInExamples := gotFA.MessageNameInExamples()
	if wantMessageNameInExamples != gotMessageNameInExamples {
		t.Errorf("mismatch in MessageNameInExamples, want %s, got %s", wantMessageNameInExamples, gotMessageNameInExamples)
	}

	wantField = &fieldAnnotations{
		FieldName:             "oneof_field",
		SetterName:            "oneof_field",
		BranchName:            "OneofField",
		ProstBranchName:       "OneofField",
		FQMessageName:         "crate::model::TestMessage",
		FieldType:             "std::boxed::Box<crate::model::TestMessage>",
		MessageType:           message,
		PrimitiveFieldType:    "crate::model::TestMessage",
		AddQueryParameter:     `let builder = req.oneof_field().map(|p| serde_json::to_value(p).map_err(Error::ser) ).transpose()?.into_iter().fold(builder, |builder, p| { use gaxi::query_parameter::QueryParameter; p.add(builder, "oneofField") });`,
		IsBoxed:               true,
		MapToBoxed:            true,
		SkipIfIsDefault:       true,
		OtherFieldsInGroup:    []*api.Field{},
		FieldTypeIsParentType: true,
	}
	if diff := cmp.Diff(wantField, oneOfField.Codec, cmpopts.IgnoreFields(api.Field{}, "Codec")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantMessageNameInExamples = "TestMessage"
	gotFA, _ = oneOfField.Codec.(*fieldAnnotations)
	gotMessageNameInExamples = gotFA.MessageNameInExamples()
	if wantMessageNameInExamples != gotMessageNameInExamples {
		t.Errorf("mismatch in MessageNameInExamples, want %s, got %s", wantMessageNameInExamples, gotMessageNameInExamples)
	}

	wantField = &fieldAnnotations{
		FieldName:             "repeated_field",
		SetterName:            "repeated_field",
		BranchName:            "RepeatedField",
		ProstBranchName:       "RepeatedField",
		FQMessageName:         "crate::model::TestMessage",
		FieldType:             "std::vec::Vec<crate::model::TestMessage>",
		MessageType:           message,
		PrimitiveFieldType:    "crate::model::TestMessage",
		AddQueryParameter:     `let builder = req.repeated_field.as_ref().map(|p| serde_json::to_value(p).map_err(Error::ser) ).transpose()?.into_iter().fold(builder, |builder, v| { use gaxi::query_parameter::QueryParameter; v.add(builder, "repeatedField") });`,
		IsBoxed:               true,
		MapToBoxed:            false,
		SkipIfIsDefault:       true,
		FieldTypeIsParentType: true,
	}
	if diff := cmp.Diff(wantField, repeatedField.Codec, cmpopts.IgnoreFields(api.Field{}, "Codec")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantMessageNameInExamples = "TestMessage"
	gotFA, _ = repeatedField.Codec.(*fieldAnnotations)
	gotMessageNameInExamples = gotFA.MessageNameInExamples()
	if wantMessageNameInExamples != gotMessageNameInExamples {
		t.Errorf("mismatch in MessageNameInExamples, want %s, got %s", wantMessageNameInExamples, gotMessageNameInExamples)
	}

	wantField = &fieldAnnotations{
		FieldName:             "message_field",
		SetterName:            "message_field",
		BranchName:            "MessageField",
		ProstBranchName:       "MessageField",
		FQMessageName:         "crate::model::TestMessage",
		FieldType:             "std::boxed::Box<crate::model::TestMessage>",
		MessageType:           message,
		PrimitiveFieldType:    "crate::model::TestMessage",
		AddQueryParameter:     `let builder = { use gaxi::query_parameter::QueryParameter; serde_json::to_value(&req.message_field).map_err(Error::ser)?.add(builder, "messageField") };`,
		IsBoxed:               true,
		MapToBoxed:            true,
		SkipIfIsDefault:       true,
		FieldTypeIsParentType: true,
	}
	if diff := cmp.Diff(wantField, messageField.Codec, cmpopts.IgnoreFields(api.Field{}, "Codec")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantMessageNameInExamples = "TestMessage"
	gotFA, _ = messageField.Codec.(*fieldAnnotations)
	gotMessageNameInExamples = gotFA.MessageNameInExamples()
	if wantMessageNameInExamples != gotMessageNameInExamples {
		t.Errorf("mismatch in MessageNameInExamples, want %s, got %s", wantMessageNameInExamples, gotMessageNameInExamples)
	}
}

func TestSameTypeNameFieldAnnotations(t *testing.T) {
	// A message with the same unqualified name as the message containing the fields.
	innerMessage := api.NewTestMessage("TestMessage").WithPackage("test.v1.inner")

	keyField := api.NewTestField("key").WithType(api.TypezInt32)
	valueField := api.NewTestField("value").WithMessageType(innerMessage)
	mapMessage := api.NewTestMessage("$Map").
		WithPackage("test.v1").
		WithFields(keyField, valueField).
		WithIsMap()

	mapField := api.NewTestField("map_field").WithMessageType(mapMessage)
	oneOfField := api.NewTestField("oneof_field").WithMessageType(innerMessage)
	group := api.NewTestOneOf("oneof_type").WithFields(oneOfField)
	repeatedField := api.NewTestField("repeated_field").
		WithMessageType(innerMessage).
		WithRepeated()
	messageField := api.NewTestField("message_field").WithMessageType(innerMessage)
	message := api.NewTestMessage("TestMessage").
		WithPackage("test.v1").
		WithDocumentation("A test message.").
		WithFields(mapField).
		WithOneOfs(group).
		WithFields(repeatedField, messageField)

	model := api.NewTestAPI([]*api.Message{message, mapMessage}, nil, nil)
	model.AddMessage(innerMessage)
	api.CrossReference(model)
	api.LabelRecursiveFields(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "test", map[string]string{})
	codec.packageMapping["test.v1.inner"] = &packagez{name: "rusty-test-inner-v1"}
	if _, err := annotateModel(model, codec); err != nil {
		t.Fatal(err)
	}
	wantMessage := &messageAnnotation{
		Name:              "TestMessage",
		ModuleName:        "test_message",
		QualifiedName:     "crate::model::TestMessage",
		RelativeName:      "TestMessage",
		ProstRelativeName: "TestMessage",
		NameInExamples:    "google_cloud_test_v1::model::TestMessage",
		PackageModuleName: "test::v1",
		SourceFQN:         "test.v1.TestMessage",
		HasNestedTypes:    true,
		DocLines:          []string{"/// A test message."},
		BasicFields:       []*api.Field{mapField, repeatedField, messageField},
	}
	// We ignore the Parent.Codec and MessageType.Codec fields of Fields,
	// as those point to the message annotations itself and was causing
	// the test to fail because of cyclic dependencies.
	if diff := cmp.Diff(wantMessage, message.Codec, cmpopts.IgnoreFields(api.Field{}, "Parent.Codec", "MessageType.Codec")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantField := &fieldAnnotations{
		FieldName:          "map_field",
		SetterName:         "map_field",
		BranchName:         "MapField",
		ProstBranchName:    "MapField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "std::collections::HashMap<i32,rusty_test_inner_v1::model::TestMessage>",
		PrimitiveFieldType: "std::collections::HashMap<i32,rusty_test_inner_v1::model::TestMessage>",
		AddQueryParameter:  `let builder = { use gaxi::query_parameter::QueryParameter; serde_json::to_value(&req.map_field).map_err(Error::ser)?.add(builder, "mapField") };`,
		KeyType:            "i32",
		KeyField:           keyField,
		ValueType:          "rusty_test_inner_v1::model::TestMessage",
		ValueField:         valueField,
		SerdeAs:            "std::collections::HashMap<wkt::internal::I32, serde_with::Same>",
		SkipIfIsDefault:    true,
		AliasInExamples:    "MapField",
	}
	if diff := cmp.Diff(wantField, mapField.Codec, cmpopts.IgnoreFields(api.Field{}, "Codec")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantMessageNameInExamples := "MapField"
	gotFA, _ := mapField.Codec.(*fieldAnnotations)
	gotMessageNameInExamples := gotFA.MessageNameInExamples()
	if wantMessageNameInExamples != gotMessageNameInExamples {
		t.Errorf("mismatch in MessageNameInExamples, want %s, got %s", wantMessageNameInExamples, gotMessageNameInExamples)
	}

	wantField = &fieldAnnotations{
		FieldName:          "oneof_field",
		SetterName:         "oneof_field",
		BranchName:         "OneofField",
		ProstBranchName:    "OneofField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "std::boxed::Box<rusty_test_inner_v1::model::TestMessage>",
		MessageType:        innerMessage,
		PrimitiveFieldType: "rusty_test_inner_v1::model::TestMessage",
		AddQueryParameter:  `let builder = req.oneof_field().map(|p| serde_json::to_value(p).map_err(Error::ser) ).transpose()?.into_iter().fold(builder, |builder, p| { use gaxi::query_parameter::QueryParameter; p.add(builder, "oneofField") });`,
		IsBoxed:            true,
		MapToBoxed:         false,
		SkipIfIsDefault:    true,
		OtherFieldsInGroup: []*api.Field{},
		AliasInExamples:    "OneofField",
	}
	if diff := cmp.Diff(wantField, oneOfField.Codec, cmpopts.IgnoreFields(api.Field{}, "Codec")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantMessageNameInExamples = "OneofField"
	gotFA, _ = oneOfField.Codec.(*fieldAnnotations)
	gotMessageNameInExamples = gotFA.MessageNameInExamples()
	if wantMessageNameInExamples != gotMessageNameInExamples {
		t.Errorf("mismatch in MessageNameInExamples, want %s, got %s", wantMessageNameInExamples, gotMessageNameInExamples)
	}

	wantField = &fieldAnnotations{
		FieldName:          "repeated_field",
		SetterName:         "repeated_field",
		BranchName:         "RepeatedField",
		ProstBranchName:    "RepeatedField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "std::vec::Vec<rusty_test_inner_v1::model::TestMessage>",
		MessageType:        innerMessage,
		PrimitiveFieldType: "rusty_test_inner_v1::model::TestMessage",
		AddQueryParameter:  `let builder = req.repeated_field.as_ref().map(|p| serde_json::to_value(p).map_err(Error::ser) ).transpose()?.into_iter().fold(builder, |builder, v| { use gaxi::query_parameter::QueryParameter; v.add(builder, "repeatedField") });`,
		SkipIfIsDefault:    true,
		AliasInExamples:    "RepeatedField",
	}
	if diff := cmp.Diff(wantField, repeatedField.Codec, cmpopts.IgnoreFields(api.Field{}, "Codec")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantMessageNameInExamples = "RepeatedField"
	gotFA, _ = repeatedField.Codec.(*fieldAnnotations)
	gotMessageNameInExamples = gotFA.MessageNameInExamples()
	if wantMessageNameInExamples != gotMessageNameInExamples {
		t.Errorf("mismatch in MessageNameInExamples, want %s, got %s", wantMessageNameInExamples, gotMessageNameInExamples)
	}

	wantField = &fieldAnnotations{
		FieldName:          "message_field",
		SetterName:         "message_field",
		BranchName:         "MessageField",
		ProstBranchName:    "MessageField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "rusty_test_inner_v1::model::TestMessage",
		MessageType:        innerMessage,
		PrimitiveFieldType: "rusty_test_inner_v1::model::TestMessage",
		AddQueryParameter:  `let builder = { use gaxi::query_parameter::QueryParameter; serde_json::to_value(&req.message_field).map_err(Error::ser)?.add(builder, "messageField") };`,
		SkipIfIsDefault:    true,
		AliasInExamples:    "MessageField",
	}
	if diff := cmp.Diff(wantField, messageField.Codec, cmpopts.IgnoreFields(api.Field{}, "Codec")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantMessageNameInExamples = "MessageField"
	gotFA, _ = messageField.Codec.(*fieldAnnotations)
	gotMessageNameInExamples = gotFA.MessageNameInExamples()
	if wantMessageNameInExamples != gotMessageNameInExamples {
		t.Errorf("mismatch in MessageNameInExamples, want %s, got %s", wantMessageNameInExamples, gotMessageNameInExamples)
	}
}

func TestPrimitiveFieldAnnotations(t *testing.T) {
	for _, test := range []struct {
		wantType    string
		wantSerdeAs string
		typez       api.Typez
	}{
		{"i32", "wkt::internal::I32", api.TypezInt32},
		{"i32", "wkt::internal::I32", api.TypezSfixed32},
		{"i32", "wkt::internal::I32", api.TypezSint32},
		{"i64", "wkt::internal::I64", api.TypezInt64},
		{"i64", "wkt::internal::I64", api.TypezSfixed64},
		{"i64", "wkt::internal::I64", api.TypezSint64},
		{"u32", "wkt::internal::U32", api.TypezUint32},
		{"u32", "wkt::internal::U32", api.TypezFixed32},
		{"u64", "wkt::internal::U64", api.TypezUint64},
		{"u64", "wkt::internal::U64", api.TypezFixed64},
		{"f32", "wkt::internal::F32", api.TypezFloat},
		{"f64", "wkt::internal::F64", api.TypezDouble},
	} {
		t.Run(fmt.Sprintf("%s_%v", test.wantType, test.typez), func(t *testing.T) {
			singularField := api.NewTestField("singular_field").
				WithType(test.typez)
			message := api.NewTestMessage("TestMessage").
				WithFields(singularField).
				WithDocumentation("A test message.")
			model := api.NewTestAPI([]*api.Message{message}, nil, nil)
			api.CrossReference(model)
			api.LabelRecursiveFields(model)
			codec := newTestCodec(t, libconfig.SpecProtobuf, "test", map[string]string{})
			annotateModel(model, codec)

			wantField := &fieldAnnotations{
				FieldName:          "singular_field",
				SetterName:         "singular_field",
				BranchName:         "SingularField",
				ProstBranchName:    "SingularField",
				FQMessageName:      "crate::model::TestMessage",
				FieldType:          test.wantType,
				PrimitiveFieldType: test.wantType,
				SerdeAs:            test.wantSerdeAs,
				AddQueryParameter:  `let builder = builder.query(&[("singularField", &req.singular_field)]);`,
				SkipIfIsDefault:    true,
			}
			if diff := cmp.Diff(wantField, singularField.Codec); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBytesAnnotations(t *testing.T) {
	for _, test := range []struct {
		sourceSpecification string
		wantType            string
		wantSerdeAs         string
	}{
		{libconfig.SpecProtobuf, "::bytes::Bytes", "serde_with::base64::Base64"},
		{libconfig.SpecOpenAPI, "::bytes::Bytes", "serde_with::base64::Base64"},
		{libconfig.SpecDiscovery, "::bytes::Bytes", "serde_with::base64::Base64<serde_with::base64::UrlSafe>"},
	} {
		t.Run(test.sourceSpecification, func(t *testing.T) {
			singularField := api.NewTestField("singular_field").
				WithType(api.TypezBytes)
			message := api.NewTestMessage("TestMessage").
				WithFields(singularField).
				WithDocumentation("A test message.")
			model := api.NewTestAPI([]*api.Message{message}, nil, nil)
			api.CrossReference(model)
			api.LabelRecursiveFields(model)
			codec := newTestCodec(t, test.sourceSpecification, "test", map[string]string{})
			annotateModel(model, codec)

			wantField := &fieldAnnotations{
				FieldName:          "singular_field",
				SetterName:         "singular_field",
				BranchName:         "SingularField",
				ProstBranchName:    "SingularField",
				FQMessageName:      "crate::model::TestMessage",
				FieldType:          test.wantType,
				PrimitiveFieldType: test.wantType,
				SerdeAs:            test.wantSerdeAs,
				AddQueryParameter:  `let builder = builder.query(&[("singularField", &req.singular_field)]);`,
			}
			if diff := cmp.Diff(wantField, singularField.Codec); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWrapperFieldAnnotations(t *testing.T) {
	for _, test := range []struct {
		wantType    string
		wantSerdeAs string
		name        string
	}{
		{"wkt::BytesValue", "serde_with::base64::Base64", "BytesValue"},
		{"wkt::UInt64Value", "wkt::internal::U64", "UInt64Value"},
		{"wkt::Int64Value", "wkt::internal::I64", "Int64Value"},
		{"wkt::UInt32Value", "wkt::internal::U32", "UInt32Value"},
		{"wkt::Int32Value", "wkt::internal::I32", "Int32Value"},
		{"wkt::FloatValue", "wkt::internal::F32", "FloatValue"},
		{"wkt::DoubleValue", "wkt::internal::F64", "DoubleValue"},
		{"wkt::BoolValue", "", "BoolValue"},
	} {
		t.Run(test.name, func(t *testing.T) {
			wkt := api.NewTestMessage(test.name).WithPackage("google.protobuf")
			singularField := api.NewTestField("singular_field").
				WithMessageType(wkt).
				WithOptional()
			message := api.NewTestMessage("TestMessage").
				WithFields(singularField).
				WithDocumentation("A test message.")
			model := api.NewTestAPI([]*api.Message{message}, nil, nil)
			api.CrossReference(model)
			api.LabelRecursiveFields(model)
			codec := createRustCodec()
			annotateModel(model, codec)

			wantField := &fieldAnnotations{
				FieldName:          "singular_field",
				SetterName:         "singular_field",
				BranchName:         "SingularField",
				ProstBranchName:    "SingularField",
				FQMessageName:      "crate::model::TestMessage",
				FieldType:          fmt.Sprintf("std::option::Option<%s>", test.wantType),
				PrimitiveFieldType: test.wantType,
				SerdeAs:            test.wantSerdeAs,
				SkipIfIsDefault:    true,
			}
			if diff := cmp.Diff(wantField, singularField.Codec, cmpopts.IgnoreFields(fieldAnnotations{}, "AddQueryParameter", "MessageType")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			got, _ := singularField.Codec.(*fieldAnnotations)
			if got.MessageType.ID != wkt.ID {
				t.Errorf("mismatch in field annotations MessageType.ID, want %s, got %s", wkt.ID, got.MessageType.ID)
			}
		})
	}
}

func TestEnumFieldAnnotations(t *testing.T) {
	enumz := api.NewTestEnum("TestEnum").WithPackage("test.v1")
	singularField := api.NewTestField("singular_field").
		WithType(api.TypezEnum).
		WithTypezID(enumz.ID)
	repeatedField := api.NewTestField("repeated_field").
		WithType(api.TypezEnum).
		WithTypezID(enumz.ID).
		WithRepeated()
	optionalField := api.NewTestField("optional_field").
		WithType(api.TypezEnum).
		WithTypezID(enumz.ID).
		WithOptional()
	nullValueField := api.NewTestField("null_value_field").
		WithType(api.TypezEnum).
		WithTypezID(".google.protobuf.NullValue")
	// TODO(#1381) - this is closer to what map message should be called.
	keyField := api.NewTestField("key").
		WithType(api.TypezString)
	valueField := api.NewTestField("value").
		WithType(api.TypezEnum).
		WithTypezID(enumz.ID)
	mapMessage := api.NewTestMessage("$map<string, .test.v1.TestEnum>").
		WithPackage("test.v1").
		WithID("$map<string, .test.v1.TestEnum>").
		WithFields(keyField, valueField).
		WithIsMap()
	mapField := api.NewTestField("map_field").
		WithMessageType(mapMessage)
	message := api.NewTestMessage("TestMessage").
		WithPackage("test.v1").
		WithFields(singularField, repeatedField, optionalField, nullValueField, mapField).
		WithDocumentation("A test message.")

	model := api.NewTestAPI([]*api.Message{message, mapMessage}, []*api.Enum{enumz}, nil)
	api.CrossReference(model)
	api.LabelRecursiveFields(model)
	codec, err := newCodec(libconfig.SpecProtobuf, map[string]string{
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
		ProstRelativeName: "TestMessage",
		NameInExamples:    "google_cloud_test_v1::model::TestMessage",
		PackageModuleName: "test::v1",
		SourceFQN:         "test.v1.TestMessage",
		DocLines:          []string{"/// A test message."},
		BasicFields:       []*api.Field{singularField, repeatedField, optionalField, nullValueField, mapField},
	}
	// We ignore the Parent.Codec field of Fields, as that points to the message annotations itself and was causing
	// the test to fail because of cyclic dependencies.
	if diff := cmp.Diff(wantMessage, message.Codec, cmpopts.IgnoreFields(api.Field{}, "Parent.Codec")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantField := &fieldAnnotations{
		FieldName:          "singular_field",
		SetterName:         "singular_field",
		BranchName:         "SingularField",
		ProstBranchName:    "SingularField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "crate::model::TestEnum",
		PrimitiveFieldType: "crate::model::TestEnum",
		AddQueryParameter:  `let builder = builder.query(&[("singularField", &req.singular_field)]);`,
		SkipIfIsDefault:    true,
	}
	if diff := cmp.Diff(wantField, singularField.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantField = &fieldAnnotations{
		FieldName:          "repeated_field",
		SetterName:         "repeated_field",
		BranchName:         "RepeatedField",
		ProstBranchName:    "RepeatedField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "std::vec::Vec<crate::model::TestEnum>",
		PrimitiveFieldType: "crate::model::TestEnum",
		AddQueryParameter:  `let builder = req.repeated_field.iter().fold(builder, |builder, p| builder.query(&[("repeatedField", p)]));`,
		SkipIfIsDefault:    true,
	}
	if diff := cmp.Diff(wantField, repeatedField.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantField = &fieldAnnotations{
		FieldName:          "optional_field",
		SetterName:         "optional_field",
		BranchName:         "OptionalField",
		ProstBranchName:    "OptionalField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "std::option::Option<crate::model::TestEnum>",
		PrimitiveFieldType: "crate::model::TestEnum",
		AddQueryParameter:  `let builder = req.optional_field.iter().fold(builder, |builder, p| builder.query(&[("optionalField", p)]));`,
		SkipIfIsDefault:    true,
	}
	if diff := cmp.Diff(wantField, optionalField.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// In the .proto specification this is represented as an enum. Which we
	// map to a unit struct.
	wantField = &fieldAnnotations{
		FieldName:          "null_value_field",
		SetterName:         "null_value_field",
		BranchName:         "NullValueField",
		ProstBranchName:    "NullValueField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "wkt::NullValue",
		PrimitiveFieldType: "wkt::NullValue",
		AddQueryParameter:  `let builder = builder.query(&[("nullValueField", &req.null_value_field)]);`,
		SkipIfIsDefault:    true,
		IsWktNullValue:     true,
	}
	if diff := cmp.Diff(wantField, nullValueField.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantField = &fieldAnnotations{
		FieldName:          "map_field",
		SetterName:         "map_field",
		BranchName:         "MapField",
		ProstBranchName:    "MapField",
		FQMessageName:      "crate::model::TestMessage",
		FieldType:          "std::collections::HashMap<std::string::String,crate::model::TestEnum>",
		PrimitiveFieldType: "std::collections::HashMap<std::string::String,crate::model::TestEnum>",
		AddQueryParameter:  `let builder = { use gaxi::query_parameter::QueryParameter; serde_json::to_value(&req.map_field).map_err(Error::ser)?.add(builder, "mapField") };`,
		KeyType:            "std::string::String",
		KeyField:           keyField,
		ValueType:          "crate::model::TestEnum",
		ValueField:         valueField,
		SkipIfIsDefault:    true,
	}
	if diff := cmp.Diff(wantField, mapField.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFormattedResourceAnnotations(t *testing.T) {
	for _, test := range []struct {
		name       string
		segments   []api.ResourceNameSegment
		wantString string
		wantArgs   []string
	}{
		{name: "standard",
			segments: []api.ResourceNameSegment{
				{Literal: "projects"},
				{Variable: "project"},
				{Literal: "locations"},
				{Variable: "location"},
				{Literal: "secrets"},
				{Variable: "secret"},
			},
			wantString: "projects/{project_id}/locations/{location_id}/secrets/{secret_id}",
			wantArgs:   []string{"project_id", "location_id", "secret_id"},
		},
		{
			name: "with name suffix",
			segments: []api.ResourceNameSegment{
				{Literal: "topics"},
				{Variable: "topicName"},
			},
			wantString: "topics/{topic_name}",
			wantArgs:   []string{"topic_name"},
		},
		{
			name: "already has id suffix",
			segments: []api.ResourceNameSegment{
				{Literal: "projects"},
				{Variable: "projectId"},
			},
			wantString: "projects/{project_id}",
			wantArgs:   []string{"project_id"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			field := api.NewTestField("name").
				WithType(api.TypezString)
			field.ResourceNamePattern = &api.ResourceNamePattern{
				Segments: test.segments,
			}
			message := api.NewTestMessage("TestMessage").
				WithPackage("test.v1").
				WithFields(field)

			model := api.NewTestAPI([]*api.Message{message}, nil, nil)
			api.CrossReference(model)
			codec := newTestCodec(t, libconfig.SpecProtobuf, "test", map[string]string{})
			annotateModel(model, codec)

			got := field.Codec.(*fieldAnnotations).FormattedResource
			want := &FormattedResource{
				FormatString: test.wantString,
				FormatArgs:   test.wantArgs,
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestJsonNameAnnotations(t *testing.T) {
	parent := api.NewTestField("parent").
		WithType(api.TypezString)
	publicKey := api.NewTestField("public_key").
		WithType(api.TypezString).
		WithJSONName("public_key")
	readTime := api.NewTestField("read_time").
		WithType(api.TypezInt32)
	optional := api.NewTestField("optional").
		WithType(api.TypezInt32).
		WithOptional()
	repeated := api.NewTestField("repeated").
		WithType(api.TypezInt32).
		WithRepeated()
	message := api.NewTestMessage("Request").
		WithFields(parent, publicKey, readTime, optional, repeated).
		WithDocumentation("A test message.")
	model := api.NewTestAPI([]*api.Message{message}, nil, nil)
	api.CrossReference(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"name-overrides": ".test.Request.public_key=custom_key",
	})
	annotateModel(model, codec)

	want := &fieldAnnotations{
		FieldName:          "parent",
		SetterName:         "parent",
		BranchName:         "Parent",
		ProstBranchName:    "Parent",
		FQMessageName:      "crate::model::Request",
		DocLines:           nil,
		FieldType:          "std::string::String",
		PrimitiveFieldType: "std::string::String",
		AddQueryParameter:  `let builder = builder.query(&[("parent", &req.parent)]);`,
		KeyType:            "",
		ValueType:          "",
	}
	if diff := cmp.Diff(want, parent.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	want = &fieldAnnotations{
		FieldName:          "custom_key",
		SetterName:         "custom_key",
		BranchName:         "CustomKey",
		ProstBranchName:    "PublicKey",
		FQMessageName:      "crate::model::Request",
		DocLines:           nil,
		FieldType:          "std::string::String",
		PrimitiveFieldType: "std::string::String",
		AddQueryParameter:  `let builder = builder.query(&[("public_key", &req.custom_key)]);`,
		KeyType:            "",
		ValueType:          "",
	}
	if diff := cmp.Diff(want, publicKey.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	want = &fieldAnnotations{
		FieldName:          "read_time",
		SetterName:         "read_time",
		BranchName:         "ReadTime",
		ProstBranchName:    "ReadTime",
		FQMessageName:      "crate::model::Request",
		DocLines:           nil,
		FieldType:          "i32",
		PrimitiveFieldType: "i32",
		AddQueryParameter:  `let builder = builder.query(&[("readTime", &req.read_time)]);`,
		SerdeAs:            "wkt::internal::I32",
		SkipIfIsDefault:    true,
	}
	if diff := cmp.Diff(want, readTime.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	want = &fieldAnnotations{
		FieldName:          "optional",
		SetterName:         "optional",
		BranchName:         "Optional",
		ProstBranchName:    "Optional",
		FQMessageName:      "crate::model::Request",
		DocLines:           nil,
		FieldType:          "std::option::Option<i32>",
		PrimitiveFieldType: "i32",
		AddQueryParameter:  `let builder = req.optional.iter().fold(builder, |builder, p| builder.query(&[("optional", p)]));`,
		SerdeAs:            "wkt::internal::I32",
		SkipIfIsDefault:    true,
	}
	if diff := cmp.Diff(want, optional.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	want = &fieldAnnotations{
		FieldName:          "repeated",
		SetterName:         "repeated",
		BranchName:         "Repeated",
		ProstBranchName:    "Repeated",
		FQMessageName:      "crate::model::Request",
		DocLines:           nil,
		FieldType:          "std::vec::Vec<i32>",
		PrimitiveFieldType: "i32",
		AddQueryParameter:  `let builder = req.repeated.iter().fold(builder, |builder, p| builder.query(&[("repeated", p)]));`,
		SerdeAs:            "wkt::internal::I32",
		SkipIfIsDefault:    true,
	}
	if diff := cmp.Diff(want, repeated.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFieldNameConflictWithNestedMessage(t *testing.T) {
	// Top-level message: google.cloud.compute.v1.CapacityHistoryRequest
	topLevelMsg := api.NewTestMessage("CapacityHistoryRequest").
		WithPackage("test.v1")

	// Nested message: google.cloud.compute.v1.Advice.CapacityHistoryRequest
	nestedMsg := api.NewTestMessage("CapacityHistoryRequest")

	// Message: google.cloud.compute.v1.Advice
	adviceMsg := api.NewTestMessage("Advice").
		WithPackage("test.v1").
		WithMessages(nestedMsg)

	// The "body" field of nestedMsg whose type is topLevelMsg
	bodyField := api.NewTestField("body").
		WithMessageType(topLevelMsg)
	overrideField := api.NewTestField("override_body").
		WithMessageType(topLevelMsg)
	nestedMsg.WithFields(bodyField, overrideField)

	model := api.NewTestAPI([]*api.Message{topLevelMsg, adviceMsg}, nil, nil)
	api.CrossReference(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "test", map[string]string{
		"generate-setter-samples": "true",
		"name-overrides":          ".test.v1.Advice.CapacityHistoryRequest.override_body=custom_body",
	})
	annotateModel(model, codec)

	gotFA, ok := bodyField.Codec.(*fieldAnnotations)
	if !ok {
		t.Fatalf("bodyField.Codec is not *fieldAnnotations")
	}
	if gotFA.AliasInExamples != "Body" {
		t.Errorf("mismatch in AliasInExamples, want Body, got %s", gotFA.AliasInExamples)
	}

	gotOverrideFA, ok := overrideField.Codec.(*fieldAnnotations)
	if !ok {
		t.Fatalf("overrideField.Codec is not *fieldAnnotations")
	}
	if gotOverrideFA.AliasInExamples != "CustomBody" {
		t.Errorf("mismatch in AliasInExamples, want CustomBody, got %s", gotOverrideFA.AliasInExamples)
	}
}
