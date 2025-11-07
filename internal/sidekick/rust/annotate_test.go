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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestPackageNames(t *testing.T) {
	model := api.NewTestAPI(
		[]*api.Message{}, []*api.Enum{},
		[]*api.Service{{Name: "Workflows", Package: "google.cloud.workflows.v1"}})
	err := api.CrossReference(model)
	if err != nil {
		t.Fatal(err)
	}
	// Override the default name for test APIs ("Test").
	model.Name = "workflows-v1"
	codec, err := newCodec("protobuf", map[string]string{
		"version":                     "1.2.3",
		"release-level":               "stable",
		"copyright-year":              "2035",
		"per-service-features":        "true",
		"extra-modules":               "operation",
		"generate-setter-samples":     "true",
		"detailed-tracing-attributes": "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := annotateModel(model, codec)
	want := &modelAnnotations{
		PackageName:               "google-cloud-workflows-v1",
		PackageVersion:            "1.2.3",
		ReleaseLevel:              "stable",
		PackageNamespace:          "google_cloud_workflows_v1",
		RequiredPackages:          []string{},
		ExternPackages:            []string{},
		HasLROs:                   false,
		CopyrightYear:             "2035",
		Services:                  []*api.Service{},
		NameToLower:               "workflows-v1",
		PerServiceFeatures:        false, // no services
		ExtraModules:              []string{"operation"},
		GenerateSetterSamples:     true,
		DetailedTracingAttributes: true,
	}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(modelAnnotations{}, "BoilerPlate")); diff != "" {
		t.Errorf("mismatch in modelAnnotations list (-want, +got)\n:%s", diff)
	}
}

func serviceAnnotationsModel() *api.API {
	request := &api.Message{
		Name:    "Request",
		Package: "test.v1",
		ID:      ".test.v1.Request",
	}
	response := &api.Message{
		Name:    "Response",
		Package: "test.v1",
		ID:      ".test.v1.Response",
		Fields: []*api.Field{
			{
				Name:    "field",
				ID:      ".test.v1.Response.field",
				Typez:   api.ENUM_TYPE,
				TypezID: ".test.v1.UsedEnum",
			},
		},
	}
	method := &api.Method{
		Name:         "GetResource",
		ID:           ".test.v1.ResourceService.GetResource",
		InputType:    request,
		InputTypeID:  ".test.v1.Request",
		OutputTypeID: ".test.v1.Response",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb: "GET",
					PathTemplate: api.NewPathTemplate().
						WithLiteral("v1").
						WithLiteral("resource"),
				},
			},
		},
	}
	emptyMethod := &api.Method{
		Name:         "DeleteResource",
		ID:           ".test.v1.ResourceService.DeleteResource",
		InputType:    request,
		InputTypeID:  ".test.v1.Request",
		OutputTypeID: ".google.protobuf.Empty",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb: "DELETE",
					PathTemplate: api.NewPathTemplate().
						WithLiteral("v1").
						WithLiteral("resource"),
				},
			},
		},
		ReturnsEmpty: true,
	}
	noHttpMethod := &api.Method{
		Name:         "DoAThing",
		ID:           ".test.v1.ResourceService.DoAThing",
		InputTypeID:  ".test.v1.Request",
		OutputTypeID: ".test.v1.Response",
	}
	service := &api.Service{
		Name:    "ResourceService",
		ID:      ".test.v1.ResourceService",
		Package: "test.v1",
		Methods: []*api.Method{method, emptyMethod, noHttpMethod},
	}

	usedEnum := &api.Enum{
		Name:    "UsedEnum",
		ID:      ".test.v1.UsedEnum",
		Package: "test.v1",
	}
	extraEnum := &api.Enum{
		Name:    "ExtraEnum",
		ID:      ".test.v1.ExtraEnum",
		Package: "test.v1",
	}

	model := api.NewTestAPI(
		[]*api.Message{request, response},
		[]*api.Enum{usedEnum, extraEnum},
		[]*api.Service{service})
	api.CrossReference(model)
	return model
}

func TestServiceAnnotations(t *testing.T) {
	model := serviceAnnotationsModel()
	service, ok := model.State.ServiceByID[".test.v1.ResourceService"]
	if !ok {
		t.Fatal("cannot find .test.v1.ResourceService")
	}
	method, ok := model.State.MethodByID[".test.v1.ResourceService.GetResource"]
	if !ok {
		t.Fatal("cannot find .test.v1.ResourceService.GetResource")
	}
	emptyMethod, ok := model.State.MethodByID[".test.v1.ResourceService.DeleteResource"]
	if !ok {
		t.Fatal("cannot find .test.v1.ResourceService.DeleteResource")
	}
	codec, err := newCodec("protobuf", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)
	wantService := &serviceAnnotations{
		Name:              "ResourceService",
		PackageModuleName: "test::v1",
		ModuleName:        "resource_service",
		Incomplete:        true,
	}
	if diff := cmp.Diff(wantService, service.Codec, cmpopts.IgnoreFields(serviceAnnotations{}, "Methods")); diff != "" {
		t.Errorf("mismatch in service annotations (-want, +got)\n:%s", diff)
	}

	// The `noHttpMethod` should be excluded from the list of methods in the
	// Codec.
	serviceAnn := service.Codec.(*serviceAnnotations)
	wantMethodList := []*api.Method{method, emptyMethod}
	if diff := cmp.Diff(wantMethodList, serviceAnn.Methods, cmpopts.IgnoreFields(api.Method{}, "Model", "Service", "SourceService")); diff != "" {
		t.Errorf("mismatch in method list (-want, +got)\n:%s", diff)
	}

	wantMethod := &methodAnnotation{
		Name:           "get_resource",
		NameNoMangling: "get_resource",
		BuilderName:    "GetResource",
		Body:           "None::<gaxi::http::NoBody>",
		PathInfo:       method.PathInfo,
		SystemParameters: []systemParameter{
			{Name: "$alt", Value: "json;enum-encoding=int"},
		},
		ServiceNameToPascal: "ResourceService",
		ServiceNameToCamel:  "resourceService",
		ServiceNameToSnake:  "resource_service",
		ReturnType:          "crate::model::Response",
	}
	if diff := cmp.Diff(wantMethod, method.Codec); diff != "" {
		t.Errorf("mismatch in method annotations (-want, +got)\n:%s", diff)
	}

	wantMethod = &methodAnnotation{
		Name:           "delete_resource",
		NameNoMangling: "delete_resource",
		BuilderName:    "DeleteResource",
		Body:           "None::<gaxi::http::NoBody>",
		PathInfo:       emptyMethod.PathInfo,
		SystemParameters: []systemParameter{
			{Name: "$alt", Value: "json;enum-encoding=int"},
		},
		ServiceNameToPascal: "ResourceService",
		ServiceNameToCamel:  "resourceService",
		ServiceNameToSnake:  "resource_service",
		ReturnType:          "()",
	}
	if diff := cmp.Diff(wantMethod, emptyMethod.Codec); diff != "" {
		t.Errorf("mismatch in method annotations (-want, +got)\n:%s", diff)
	}
}

func TestServiceAnnotationsDetailedTracing(t *testing.T) {
	model := serviceAnnotationsModel()
	service, ok := model.State.ServiceByID[".test.v1.ResourceService"]
	if !ok {
		t.Fatal("cannot find .test.v1.ResourceService")
	}
	codec, err := newCodec("protobuf", map[string]string{
		"detailed-tracing-attributes": "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)
	got := service.Codec.(*serviceAnnotations)
	if !got.DetailedTracingAttributes {
		t.Errorf("serviceAnnotations.DetailedTracingAttributes = %v, want %v", got.DetailedTracingAttributes, true)
	}
}

func TestMethodAnnotationsDetailedTracing(t *testing.T) {
	model := serviceAnnotationsModel()
	method, ok := model.State.MethodByID[".test.v1.ResourceService.GetResource"]
	if !ok {
		t.Fatal("cannot find .test.v1.ResourceService.GetResource")
	}
	codec, err := newCodec("protobuf", map[string]string{
		"detailed-tracing-attributes": "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)
	got := method.Codec.(*methodAnnotation)
	if !got.DetailedTracingAttributes {
		t.Errorf("methodAnnotation.DetailedTracingAttributes = %v, want %v", got.DetailedTracingAttributes, true)
	}
}

func TestServiceAnnotationsPerServiceFeatures(t *testing.T) {
	model := serviceAnnotationsModel()
	service, ok := model.State.ServiceByID[".test.v1.ResourceService"]
	if !ok {
		t.Fatal("cannot find .test.v1.ResourceService")
	}
	codec, err := newCodec("protobuf", map[string]string{
		"per-service-features": "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)
	wantService := &serviceAnnotations{
		Name:               "ResourceService",
		PackageModuleName:  "test::v1",
		ModuleName:         "resource_service",
		PerServiceFeatures: true,
		Incomplete:         true,
	}
	if diff := cmp.Diff(wantService, service.Codec, cmpopts.IgnoreFields(serviceAnnotations{}, "Methods")); diff != "" {
		t.Errorf("mismatch in service annotations (-want, +got)\n:%s", diff)
	}
}

func TestServiceAnnotationsLROTypes(t *testing.T) {
	create := &api.Message{
		Name:    "CreateResourceRequest",
		ID:      ".test.CreateResourceRequest",
		Package: "test",
	}
	delete := &api.Message{
		Name:    "DeleteResourceRequest",
		ID:      ".test.DeleteResourceRequest",
		Package: "test",
	}
	resource := &api.Message{
		Name:    "Resource",
		ID:      ".test.Resource",
		Package: "test",
	}
	metadata := &api.Message{
		Name:    "OperationMetadata",
		ID:      ".test.OperationMetadata",
		Package: "test",
	}
	operation := &api.Message{
		Name:    "Operation",
		ID:      ".google.longrunning.Operation",
		Package: "google.longrunning",
	}
	service := &api.Service{
		Name:    "LroService",
		ID:      ".test.LroService",
		Package: "test",
		Methods: []*api.Method{
			{
				Name:         "CreateResource",
				ID:           ".test.LroService.CreateResource",
				PathInfo:     &api.PathInfo{},
				InputType:    create,
				InputTypeID:  ".test.CreateResourceRequest",
				OutputTypeID: ".google.longrunning.Operation",
				OperationInfo: &api.OperationInfo{
					MetadataTypeID: ".test.OperationMetadata",
					ResponseTypeID: ".test.Resource",
				},
			},
			{
				Name:         "DeleteResource",
				ID:           ".test.LroService.DeleteResource",
				PathInfo:     &api.PathInfo{},
				InputType:    delete,
				InputTypeID:  ".test.DeleteResourceRequest",
				OutputTypeID: ".google.longrunning.Operation",
				OperationInfo: &api.OperationInfo{
					MetadataTypeID: ".test.OperationMetadata",
					ResponseTypeID: ".google.protobuf.Empty",
				},
			},
		},
	}
	model := api.NewTestAPI([]*api.Message{create, delete, resource, metadata, operation}, []*api.Enum{}, []*api.Service{service})
	err := api.CrossReference(model)
	if err != nil {
		t.Fatal(err)
	}

	codec, err := newCodec("protobuf", map[string]string{
		"include-grpc-only-methods": "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)
	empty := model.State.MessageByID[".google.protobuf.Empty"]
	wantService := &serviceAnnotations{
		Name:              "LroService",
		PackageModuleName: "test",
		ModuleName:        "lro_service",
		LROTypes: []*api.Message{
			metadata,
			resource,
			empty,
		},
	}
	if !wantService.HasLROs() {
		t.Errorf("HasLRO should be true. The service has several LROs.")
	}
	if diff := cmp.Diff(wantService, service.Codec, cmpopts.IgnoreFields(serviceAnnotations{}, "Methods")); diff != "" {
		t.Errorf("mismatch in service annotations (-want, +got)\n:%s", diff)
	}
}

func TestServiceAnnotationsNameOverrides(t *testing.T) {
	model := serviceAnnotationsModel()
	service, ok := model.State.ServiceByID[".test.v1.ResourceService"]
	if !ok {
		t.Fatal("cannot find .test.v1.ResourceService")
	}
	method, ok := model.State.MethodByID[".test.v1.ResourceService.GetResource"]
	if !ok {
		t.Fatal("cannot find .test.v1.ResourceService.GetResource")
	}

	codec, err := newCodec("protobuf", map[string]string{
		"name-overrides": ".test.v1.ResourceService=Renamed",
	})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)

	serviceFilter := cmpopts.IgnoreFields(serviceAnnotations{}, "PackageModuleName", "Methods")
	wantService := &serviceAnnotations{
		Name:       "Renamed",
		ModuleName: "renamed",
		Incomplete: true,
	}
	if diff := cmp.Diff(wantService, service.Codec, serviceFilter); diff != "" {
		t.Errorf("mismatch in service annotations (-want, +got)\n:%s", diff)
	}

	methodFilter := cmpopts.IgnoreFields(methodAnnotation{}, "Name", "NameNoMangling", "BuilderName", "Body", "PathInfo", "SystemParameters", "ReturnType")
	wantMethod := &methodAnnotation{
		ServiceNameToPascal: "Renamed",
		ServiceNameToCamel:  "renamed",
		ServiceNameToSnake:  "renamed",
	}
	if diff := cmp.Diff(wantMethod, method.Codec, methodFilter); diff != "" {
		t.Errorf("mismatch in method annotations (-want, +got)\n:%s", diff)
	}
}

func TestOneOfAnnotations(t *testing.T) {
	singular := &api.Field{
		Name:     "oneof_field",
		JSONName: "oneofField",
		ID:       ".test.Message.oneof_field",
		Typez:    api.STRING_TYPE,
		IsOneOf:  true,
	}
	repeated := &api.Field{
		Name:     "oneof_field_repeated",
		JSONName: "oneofFieldRepeated",
		ID:       ".test.Message.oneof_field_repeated",
		Typez:    api.STRING_TYPE,
		Repeated: true,
		IsOneOf:  true,
	}
	map_field := &api.Field{
		Name:     "oneof_field_map",
		JSONName: "oneofFieldMap",
		ID:       ".test.Message.oneof_field_map",
		Typez:    api.MESSAGE_TYPE,
		TypezID:  ".test.$Map",
		Repeated: false,
		IsOneOf:  true,
	}
	integer_field := &api.Field{
		Name:     "oneof_field_integer",
		JSONName: "oneofFieldInteger",
		ID:       ".test.Message.oneof_field_integer",
		Typez:    api.INT64_TYPE,
		IsOneOf:  true,
	}
	boxed_field := &api.Field{
		Name:     "oneof_field_boxed",
		JSONName: "oneofFieldBoxed",
		ID:       ".test.Message.oneof_field_boxed",
		Typez:    api.MESSAGE_TYPE,
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
	key_field := &api.Field{Name: "key", Typez: api.INT32_TYPE}
	value_field := &api.Field{Name: "value", Typez: api.FLOAT_TYPE}
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
		QualifiedName:       "crate::model::message::Type",
		RelativeName:        "message::Type",
		StructQualifiedName: "crate::model::Message",
		ScopeInExamples:     "google_cloud_test::model::message",
		FieldType:           "crate::model::message::Type",
		DocLines:            []string{"/// Say something clever about this oneof."},
		ExampleField:        singular,
	}
	if diff := cmp.Diff(wantOneOfCodec, group.Codec, cmpopts.IgnoreFields(api.OneOf{}, "Codec")); diff != "" {
		t.Errorf("mismatch in oneof annotations (-want, +got)\n:%s", diff)
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
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
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
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
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
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
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
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
	}

	wantFieldCodec = &fieldAnnotations{
		FieldName:          "oneof_field_boxed",
		SetterName:         "oneof_field_boxed",
		BranchName:         "OneofFieldBoxed",
		FQMessageName:      "crate::model::Message",
		DocLines:           nil,
		FieldType:          "std::boxed::Box<wkt::DoubleValue>",
		PrimitiveFieldType: "wkt::DoubleValue",
		AddQueryParameter:  `let builder = req.oneof_field_boxed().map(|p| serde_json::to_value(p).map_err(Error::ser) ).transpose()?.into_iter().fold(builder, |builder, p| { use gaxi::query_parameter::QueryParameter; p.add(builder, "oneofFieldBoxed") });`,
		IsBoxed:            true,
		SerdeAs:            "wkt::internal::F64",
		SkipIfIsDefault:    true,
		OtherFieldsInGroup: []*api.Field{singular, repeated, map_field, integer_field},
	}
	if diff := cmp.Diff(wantFieldCodec, boxed_field.Codec, ignore); diff != "" {
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
	}
}

func TestOneOfConflictAnnotations(t *testing.T) {
	singular := &api.Field{
		Name:     "oneof_field",
		JSONName: "oneofField",
		ID:       ".test.Message.oneof_field",
		Typez:    api.STRING_TYPE,
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
	codec, err := newCodec("protobuf", map[string]string{
		"name-overrides": ".test.Message.nested_thing=NestedThingOneOf",
	})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)

	// Stops the recursion when comparing fields.
	ignore := cmpopts.IgnoreFields(api.OneOf{}, "Codec")

	want := &oneOfAnnotation{
		FieldName:           "nested_thing",
		SetterName:          "nested_thing",
		EnumName:            "NestedThingOneOf",
		QualifiedName:       "crate::model::message::NestedThingOneOf",
		RelativeName:        "message::NestedThingOneOf",
		StructQualifiedName: "crate::model::Message",
		ScopeInExamples:     "google_cloud_test::model::message",
		FieldType:           "crate::model::message::NestedThingOneOf",
		DocLines:            []string{"/// Say something clever about this oneof."},
		ExampleField:        singular,
	}
	if diff := cmp.Diff(want, group.Codec, ignore); diff != "" {
		t.Errorf("mismatch in oneof annotations (-want, +got)\n:%s", diff)
	}
}

func TestEnumAnnotations(t *testing.T) {
	// Verify we can handle values that are not in SCREAMING_SNAKE_CASE style.
	v0 := &api.EnumValue{
		Name:          "week5",
		ID:            ".test.v1.TestEnum.week5",
		Documentation: "week5 is also documented.",
		Number:        2,
	}
	v1 := &api.EnumValue{
		Name:          "MULTI_WORD_VALUE",
		ID:            ".test.v1.TestEnum.MULTI_WORD_VALUES",
		Documentation: "MULTI_WORD_VALUE is also documented.",
		Number:        1,
	}
	v2 := &api.EnumValue{
		Name:          "VALUE",
		ID:            ".test.v1.TestEnum.VALUE",
		Documentation: "VALUE is also documented.",
		Number:        0,
	}
	v3 := &api.EnumValue{
		Name:   "TEST_ENUM_V3",
		ID:     ".test.v1.TestEnum.TEST_ENUM_V3",
		Number: 3,
	}
	v4 := &api.EnumValue{
		Name:   "TEST_ENUM_2025",
		ID:     ".test.v1.TestEnum.TEST_ENUM_2025",
		Number: 4,
	}
	enum := &api.Enum{
		Name:          "TestEnum",
		ID:            ".test.v1.TestEnum",
		Package:       "test.v1",
		Documentation: "The enum is documented.",
		Values:        []*api.EnumValue{v0, v1, v2, v3, v4},
	}

	model := api.NewTestAPI(
		[]*api.Message{}, []*api.Enum{enum}, []*api.Service{})
	codec, err := newCodec("protobuf", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)

	wantEnumCodec := &enumAnnotation{
		Name:           "TestEnum",
		ModuleName:     "test_enum",
		QualifiedName:  "crate::model::TestEnum",
		RelativeName:   "TestEnum",
		DocLines:       []string{"/// The enum is documented."},
		UniqueNames:    []*api.EnumValue{v0, v1, v2, v3, v4},
		NameInExamples: "google_cloud_test_v1::model::TestEnum",
		ValuesForExamples: []*enumValueForExamples{
			{EnumValue: v0, Index: 0},
			{EnumValue: v1, Index: 1},
			{EnumValue: v3, Index: 2},
		},
	}
	if diff := cmp.Diff(wantEnumCodec, enum.Codec, cmpopts.IgnoreFields(api.EnumValue{}, "Codec", "Parent")); diff != "" {
		t.Errorf("mismatch in enum annotations (-want, +got)\n:%s", diff)
	}

	wantEnumValueCodec := &enumValueAnnotation{
		Name:        "WEEK_5",
		VariantName: "Week5",
		EnumType:    "TestEnum",
		DocLines:    []string{"/// week5 is also documented."},
	}
	if diff := cmp.Diff(wantEnumValueCodec, v0.Codec); diff != "" {
		t.Errorf("mismatch in enum annotations (-want, +got)\n:%s", diff)
	}

	wantEnumValueCodec = &enumValueAnnotation{
		Name:        "MULTI_WORD_VALUE",
		VariantName: "MultiWordValue",
		EnumType:    "TestEnum",
		DocLines:    []string{"/// MULTI_WORD_VALUE is also documented."},
	}
	if diff := cmp.Diff(wantEnumValueCodec, v1.Codec); diff != "" {
		t.Errorf("mismatch in enum annotations (-want, +got)\n:%s", diff)
	}

	wantEnumValueCodec = &enumValueAnnotation{
		Name:        "VALUE",
		VariantName: "Value",
		EnumType:    "TestEnum",
		DocLines:    []string{"/// VALUE is also documented."},
	}
	if diff := cmp.Diff(wantEnumValueCodec, v2.Codec); diff != "" {
		t.Errorf("mismatch in enum annotations (-want, +got)\n:%s", diff)
	}

	wantEnumValueCodec = &enumValueAnnotation{
		Name:        "TEST_ENUM_V3",
		VariantName: "V3",
		EnumType:    "TestEnum",
	}
	if diff := cmp.Diff(wantEnumValueCodec, v3.Codec); diff != "" {
		t.Errorf("mismatch in enum annotations (-want, +got)\n:%s", diff)
	}

	wantEnumValueCodec = &enumValueAnnotation{
		Name:        "TEST_ENUM_2025",
		VariantName: "TestEnum2025",
		EnumType:    "TestEnum",
	}
	if diff := cmp.Diff(wantEnumValueCodec, v4.Codec); diff != "" {
		t.Errorf("mismatch in enum annotations (-want, +got)\n:%s", diff)
	}
}

func TestDuplicateEnumValueAnnotations(t *testing.T) {
	// Verify we can handle values that are not in SCREAMING_SNAKE_CASE style.
	v0 := &api.EnumValue{
		Name:   "full",
		ID:     ".test.v1.TestEnum.full",
		Number: 1,
	}
	v1 := &api.EnumValue{
		Name:   "FULL",
		ID:     ".test.v1.TestEnum.FULL",
		Number: 1,
	}
	v2 := &api.EnumValue{
		Name:   "partial",
		ID:     ".test.v1.TestEnum.partial",
		Number: 2,
	}
	// This does not happen in practice, but we want to verify the code can
	// handle it if it ever does.
	v3 := &api.EnumValue{
		Name:   "PARTIAL",
		ID:     ".test.v1.TestEnum.PARTIAL",
		Number: 3,
	}
	enum := &api.Enum{
		Name:    "TestEnum",
		ID:      ".test.v1.TestEnum",
		Package: "test.v1",
		Values:  []*api.EnumValue{v0, v1, v2, v3},
	}

	model := api.NewTestAPI(
		[]*api.Message{}, []*api.Enum{enum}, []*api.Service{})
	codec, err := newCodec("protobuf", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)

	want := &enumAnnotation{
		Name:           "TestEnum",
		ModuleName:     "test_enum",
		QualifiedName:  "crate::model::TestEnum",
		RelativeName:   "TestEnum",
		UniqueNames:    []*api.EnumValue{v0, v2},
		NameInExamples: "google_cloud_test_v1::model::TestEnum",
		ValuesForExamples: []*enumValueForExamples{
			{EnumValue: v0, Index: 0},
			{EnumValue: v2, Index: 1},
		},
	}

	if diff := cmp.Diff(want, enum.Codec, cmpopts.IgnoreFields(api.EnumValue{}, "Codec", "Parent")); diff != "" {
		t.Errorf("mismatch in enum annotations (-want, +got)\n:%s", diff)
	}
}

func TestJsonNameAnnotations(t *testing.T) {
	parent := &api.Field{
		Name:     "parent",
		JSONName: "parent",
		ID:       ".test.Request.parent",
		Typez:    api.STRING_TYPE,
	}
	publicKey := &api.Field{
		Name:     "public_key",
		JSONName: "public_key",
		ID:       ".test.Request.public_key",
		Typez:    api.STRING_TYPE,
	}
	readTime := &api.Field{
		Name:     "read_time",
		JSONName: "readTime",
		ID:       ".test.Request.read_time",
		Typez:    api.INT32_TYPE,
	}
	optional := &api.Field{
		Name:     "optional",
		JSONName: "optional",
		ID:       ".test.Request.optional",
		Typez:    api.INT32_TYPE,
		Optional: true,
	}
	repeated := &api.Field{
		Name:     "repeated",
		JSONName: "repeated",
		ID:       ".test.Request.repeated",
		Typez:    api.INT32_TYPE,
		Repeated: true,
	}
	message := &api.Message{
		Name:          "Request",
		Package:       "test",
		ID:            ".test.Request",
		Documentation: "A test message.",
		Fields:        []*api.Field{parent, publicKey, readTime, optional, repeated},
	}
	model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{}, []*api.Service{})
	api.CrossReference(model)
	codec, err := newCodec("protobuf", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)

	want := &fieldAnnotations{
		FieldName:          "parent",
		SetterName:         "parent",
		BranchName:         "Parent",
		FQMessageName:      "crate::model::Request",
		DocLines:           nil,
		FieldType:          "std::string::String",
		PrimitiveFieldType: "std::string::String",
		AddQueryParameter:  `let builder = builder.query(&[("parent", &req.parent)]);`,
		KeyType:            "",
		ValueType:          "",
	}
	if diff := cmp.Diff(want, parent.Codec); diff != "" {
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
	}

	want = &fieldAnnotations{
		FieldName:          "public_key",
		SetterName:         "public_key",
		BranchName:         "PublicKey",
		FQMessageName:      "crate::model::Request",
		DocLines:           nil,
		FieldType:          "std::string::String",
		PrimitiveFieldType: "std::string::String",
		AddQueryParameter:  `let builder = builder.query(&[("public_key", &req.public_key)]);`,
		KeyType:            "",
		ValueType:          "",
	}
	if diff := cmp.Diff(want, publicKey.Codec); diff != "" {
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
	}

	want = &fieldAnnotations{
		FieldName:          "read_time",
		SetterName:         "read_time",
		BranchName:         "ReadTime",
		FQMessageName:      "crate::model::Request",
		DocLines:           nil,
		FieldType:          "i32",
		PrimitiveFieldType: "i32",
		AddQueryParameter:  `let builder = builder.query(&[("readTime", &req.read_time)]);`,
		SerdeAs:            "wkt::internal::I32",
		SkipIfIsDefault:    true,
	}
	if diff := cmp.Diff(want, readTime.Codec); diff != "" {
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
	}

	want = &fieldAnnotations{
		FieldName:          "optional",
		SetterName:         "optional",
		BranchName:         "Optional",
		FQMessageName:      "crate::model::Request",
		DocLines:           nil,
		FieldType:          "std::option::Option<i32>",
		PrimitiveFieldType: "i32",
		AddQueryParameter:  `let builder = req.optional.iter().fold(builder, |builder, p| builder.query(&[("optional", p)]));`,
		SerdeAs:            "wkt::internal::I32",
		SkipIfIsDefault:    true,
	}
	if diff := cmp.Diff(want, optional.Codec); diff != "" {
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
	}

	want = &fieldAnnotations{
		FieldName:          "repeated",
		SetterName:         "repeated",
		BranchName:         "Repeated",
		FQMessageName:      "crate::model::Request",
		DocLines:           nil,
		FieldType:          "std::vec::Vec<i32>",
		PrimitiveFieldType: "i32",
		AddQueryParameter:  `let builder = req.repeated.iter().fold(builder, |builder, p| builder.query(&[("repeated", p)]));`,
		SerdeAs:            "wkt::internal::I32",
		SkipIfIsDefault:    true,
	}
	if diff := cmp.Diff(want, repeated.Codec); diff != "" {
		t.Errorf("mismatch in field annotations (-want, +got)\n:%s", diff)
	}
}

func TestMessageAnnotations(t *testing.T) {
	message := &api.Message{
		Name:          "TestMessage",
		Package:       "test.v1",
		ID:            ".test.v1.TestMessage",
		Documentation: "A test message.",
	}
	nested := &api.Message{
		Name:          "NestedMessage",
		Package:       "test.v1",
		ID:            ".test.v1.TestMessage.NestedMessage",
		Documentation: "A nested message.",
		Parent:        message,
	}
	message.Messages = []*api.Message{nested}

	model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{}, []*api.Service{})
	api.CrossReference(model)
	codec, err := newCodec("protobuf", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)
	want := &messageAnnotation{
		Name:              "TestMessage",
		ModuleName:        "test_message",
		QualifiedName:     "crate::model::TestMessage",
		RelativeName:      "TestMessage",
		NameInExamples:    "google_cloud_test_v1::model::TestMessage",
		PackageModuleName: "test::v1",
		SourceFQN:         "test.v1.TestMessage",
		DocLines:          []string{"/// A test message."},
		HasNestedTypes:    true,
		BasicFields:       []*api.Field{},
	}
	if diff := cmp.Diff(want, message.Codec); diff != "" {
		t.Errorf("mismatch in message annotations (-want, +got)\n:%s", diff)
	}

	want = &messageAnnotation{
		Name:              "NestedMessage",
		ModuleName:        "nested_message",
		QualifiedName:     "crate::model::test_message::NestedMessage",
		RelativeName:      "test_message::NestedMessage",
		NameInExamples:    "google_cloud_test_v1::model::test_message::NestedMessage",
		PackageModuleName: "test::v1",
		SourceFQN:         "test.v1.TestMessage.NestedMessage",
		DocLines:          []string{"/// A nested message."},
		HasNestedTypes:    false,
		BasicFields:       []*api.Field{},
	}
	if diff := cmp.Diff(want, nested.Codec); diff != "" {
		t.Errorf("mismatch in nested message annotations (-want, +got)\n:%s", diff)
	}
}

func TestPathInfoAnnotations(t *testing.T) {
	binding := func(verb string) *api.PathBinding {
		return &api.PathBinding{
			Verb: verb,
			PathTemplate: api.NewPathTemplate().
				WithLiteral("v1").
				WithLiteral("resource"),
		}
	}

	type TestCase struct {
		Bindings           []*api.PathBinding
		DefaultIdempotency string
	}
	testCases := []TestCase{
		{[]*api.PathBinding{}, "false"},
		{[]*api.PathBinding{binding("GET")}, "true"},
		{[]*api.PathBinding{binding("PUT")}, "true"},
		{[]*api.PathBinding{binding("DELETE")}, "true"},
		{[]*api.PathBinding{binding("POST")}, "false"},
		{[]*api.PathBinding{binding("PATCH")}, "false"},
		{[]*api.PathBinding{binding("GET"), binding("GET")}, "true"},
		{[]*api.PathBinding{binding("GET"), binding("POST")}, "false"},
		{[]*api.PathBinding{binding("POST"), binding("POST")}, "false"},
	}
	for _, testCase := range testCases {
		request := &api.Message{
			Name:    "Request",
			Package: "test.v1",
			ID:      ".test.v1.Request",
		}
		response := &api.Message{
			Name:    "Response",
			Package: "test.v1",
			ID:      ".test.v1.Response",
		}
		method := &api.Method{
			Name:         "GetResource",
			ID:           ".test.v1.Service.GetResource",
			InputTypeID:  ".test.v1.Request",
			OutputTypeID: ".test.v1.Response",
			PathInfo: &api.PathInfo{
				Bindings: testCase.Bindings,
			},
		}
		service := &api.Service{
			Name:    "ResourceService",
			ID:      ".test.v1.ResourceService",
			Package: "test.v1",
			Methods: []*api.Method{method},
		}

		model := api.NewTestAPI(
			[]*api.Message{request, response},
			[]*api.Enum{},
			[]*api.Service{service})
		api.CrossReference(model)
		codec, err := newCodec("protobuf", map[string]string{
			"include-grpc-only-methods": "true",
		})
		if err != nil {
			t.Fatal(err)
		}
		annotateModel(model, codec)

		pathInfoAnn := method.PathInfo.Codec.(*pathInfoAnnotation)
		if pathInfoAnn.IsIdempotent != testCase.DefaultIdempotency {
			t.Errorf("fail")
		}
	}
}

func TestPathBindingAnnotations(t *testing.T) {
	f_name := &api.Field{
		Name:     "name",
		JSONName: "name",
		ID:       ".test.Request.name",
		Typez:    api.STRING_TYPE,
	}

	f_project := &api.Field{
		Name:     "project",
		JSONName: "project",
		ID:       ".test.Request.project",
		Typez:    api.STRING_TYPE,
	}
	f_location := &api.Field{
		Name:     "location",
		JSONName: "location",
		ID:       ".test.Request.location",
		Typez:    api.STRING_TYPE,
	}
	f_id := &api.Field{
		Name:     "id",
		JSONName: "id",
		ID:       ".test.Request.id",
		Typez:    api.UINT64_TYPE,
	}
	f_optional := &api.Field{
		Name:     "optional",
		JSONName: "optional",
		ID:       ".test.Request.optional",
		Typez:    api.STRING_TYPE,
		Optional: true,
	}

	// A field also of type `Request`. We want to test nested path
	// parameters, and this saves us from having to define a new
	// `api.Message`, with all of its fields.
	f_child := &api.Field{
		Name:     "child",
		JSONName: "child",
		ID:       ".test.Request.child",
		Typez:    api.MESSAGE_TYPE,
		TypezID:  ".test.Request",
		Optional: true,
	}

	request := &api.Message{
		Name:    "Request",
		Package: "test",
		ID:      ".test.Request",
		Fields: []*api.Field{
			f_name,
			f_project,
			f_location,
			f_id,
			f_optional,
			f_child,
		},
	}
	response := &api.Message{
		Name:    "Response",
		Package: "test",
		ID:      ".test.Response",
	}

	b0 := &api.PathBinding{
		Verb: "POST",
		PathTemplate: api.NewPathTemplate().
			WithLiteral("v2").
			WithVariable(api.NewPathVariable("name").
				WithLiteral("projects").
				WithMatch().
				WithLiteral("locations").
				WithMatch()).
			WithVerb("create"),
		QueryParameters: map[string]bool{
			"id": true,
		},
	}
	want_b0 := &pathBindingAnnotation{
		PathFmt:     "/v2/{}:create",
		QueryParams: []*api.Field{f_id},
		Substitutions: []*bindingSubstitution{
			{
				FieldAccessor: "Some(&req).map(|m| &m.name).map(|s| s.as_str())",
				FieldName:     "name",
				Template:      []string{"projects", "*", "locations", "*"},
			},
		},
	}

	b1 := &api.PathBinding{
		Verb: "POST",
		PathTemplate: api.NewPathTemplate().
			WithLiteral("v1").
			WithLiteral("projects").
			WithVariableNamed("project").
			WithLiteral("locations").
			WithVariableNamed("location").
			WithLiteral("ids").
			WithVariableNamed("id").
			WithVerb("action"),
	}
	want_b1 := &pathBindingAnnotation{
		PathFmt: "/v1/projects/{}/locations/{}/ids/{}:action",
		Substitutions: []*bindingSubstitution{
			{
				FieldAccessor: "Some(&req).map(|m| &m.project).map(|s| s.as_str())",
				FieldName:     "project",
				Template:      []string{"*"},
			},
			{
				FieldAccessor: "Some(&req).map(|m| &m.location).map(|s| s.as_str())",
				FieldName:     "location",
				Template:      []string{"*"},
			},
			{
				FieldAccessor: "Some(&req).map(|m| &m.id)",
				FieldName:     "id",
				Template:      []string{"*"},
			},
		},
	}
	b2 := &api.PathBinding{
		Verb: "POST",
		PathTemplate: api.NewPathTemplate().
			WithLiteral("v1").
			WithLiteral("projects").
			WithVariableNamed("child", "project").
			WithLiteral("locations").
			WithVariableNamed("child", "location").
			WithLiteral("ids").
			WithVariableNamed("child", "id").
			WithVerb("actionOnChild"),
	}
	want_b2 := &pathBindingAnnotation{
		PathFmt: "/v1/projects/{}/locations/{}/ids/{}:actionOnChild",
		Substitutions: []*bindingSubstitution{
			{
				FieldAccessor: "Some(&req).and_then(|m| m.child.as_ref()).map(|m| &m.project).map(|s| s.as_str())",
				FieldName:     "child.project",
				Template:      []string{"*"},
			},
			{
				FieldAccessor: "Some(&req).and_then(|m| m.child.as_ref()).map(|m| &m.location).map(|s| s.as_str())",
				FieldName:     "child.location",
				Template:      []string{"*"},
			},
			{
				FieldAccessor: "Some(&req).and_then(|m| m.child.as_ref()).map(|m| &m.id)",
				FieldName:     "child.id",
				Template:      []string{"*"},
			},
		},
	}
	b3 := &api.PathBinding{
		Verb: "GET",
		PathTemplate: api.NewPathTemplate().
			WithLiteral("v2").
			WithLiteral("foos"),
		QueryParameters: map[string]bool{
			"name":     true,
			"optional": true,
			"child":    true,
		},
	}
	want_b3 := &pathBindingAnnotation{
		PathFmt:     "/v2/foos",
		QueryParams: []*api.Field{f_name, f_optional, f_child},
	}
	method := &api.Method{
		Name:         "DoFoo",
		ID:           ".test.Service.DoFoo",
		InputType:    request,
		InputTypeID:  ".test.Request",
		OutputTypeID: ".test.Response",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{b0, b1, b2, b3},
		},
	}
	service := &api.Service{
		Name:    "FooService",
		ID:      ".test.FooService",
		Package: "test",
		Methods: []*api.Method{method},
	}

	model := api.NewTestAPI(
		[]*api.Message{request, response},
		[]*api.Enum{},
		[]*api.Service{service})
	api.CrossReference(model)
	codec, err := newCodec("protobuf", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)

	if diff := cmp.Diff(want_b0, b0.Codec); diff != "" {
		t.Errorf("mismatch in path binding annotations (-want, +got)\n:%s", diff)
	}
	if diff := cmp.Diff(want_b1, b1.Codec); diff != "" {
		t.Errorf("mismatch in path binding annotations (-want, +got)\n:%s", diff)
	}
	if diff := cmp.Diff(want_b2, b2.Codec); diff != "" {
		t.Errorf("mismatch in path binding annotations (-want, +got)\n:%s", diff)
	}

	if diff := cmp.Diff(want_b3, b3.Codec); diff != "" {
		t.Errorf("mismatch in path binding annotations (-want, +got)\n:%s", diff)
	}
}

func TestPathBindingAnnotationsDetailedTracing(t *testing.T) {
	f_name := &api.Field{
		Name:     "name",
		JSONName: "name",
		ID:       ".test.Request.name",
		Typez:    api.STRING_TYPE,
	}
	request := &api.Message{
		Name:    "Request",
		Package: "test",
		ID:      ".test.Request",
		Fields:  []*api.Field{f_name},
	}
	response := &api.Message{
		Name:    "Response",
		Package: "test",
		ID:      ".test.Response",
	}
	binding := &api.PathBinding{
		Verb: "POST",
		PathTemplate: api.NewPathTemplate().
			WithLiteral("v2").
			WithVariable(api.NewPathVariable("name").
				WithLiteral("projects").
				WithMatch()).
			WithVerb("create"),
	}
	method := &api.Method{
		Name:         "DoFoo",
		ID:           ".test.Service.DoFoo",
		InputType:    request,
		InputTypeID:  ".test.Request",
		OutputTypeID: ".test.Response",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{binding},
		},
	}
	service := &api.Service{
		Name:    "FooService",
		ID:      ".test.FooService",
		Package: "test",
		Methods: []*api.Method{method},
	}
	model := api.NewTestAPI(
		[]*api.Message{request, response},
		[]*api.Enum{},
		[]*api.Service{service})
	api.CrossReference(model)
	codec, err := newCodec("protobuf", map[string]string{
		"detailed-tracing-attributes": "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)

	got := binding.Codec.(*pathBindingAnnotation)
	if !got.DetailedTracingAttributes {
		t.Errorf("pathBindingAnnotation.DetailedTracingAttributes = %v, want %v", got.DetailedTracingAttributes, true)
	}
}

func TestPathBindingAnnotationsStyle(t *testing.T) {
	for _, test := range []struct {
		FieldName     string
		WantFieldName string
		WantAccessor  string
	}{
		{"machine", "machine", "Some(&req).map(|m| &m.machine).map(|s| s.as_str())"},
		{"machineType", "machine_type", "Some(&req).map(|m| &m.machine_type).map(|s| s.as_str())"},
		{"machine_type", "machine_type", "Some(&req).map(|m| &m.machine_type).map(|s| s.as_str())"},
		{"type", "r#type", "Some(&req).map(|m| &m.r#type).map(|s| s.as_str())"},
	} {
		field := &api.Field{
			Name:     test.FieldName,
			JSONName: test.FieldName,
			ID:       fmt.Sprintf(".test.Request.%s", test.FieldName),
			Typez:    api.STRING_TYPE,
		}
		request := &api.Message{
			Name:    "Request",
			Package: "test",
			ID:      ".test.Request",
			Fields:  []*api.Field{field},
		}
		response := &api.Message{
			Name:    "Response",
			Package: "test",
			ID:      ".test.Response",
		}
		binding := &api.PathBinding{
			Verb: "GET",
			PathTemplate: api.NewPathTemplate().
				WithLiteral("v1").
				WithLiteral("machines").
				WithVariable(api.NewPathVariable(test.FieldName).
					WithMatch()).
				WithVerb("create"),
			QueryParameters: map[string]bool{},
		}
		wantBinding := &pathBindingAnnotation{
			PathFmt: "/v1/machines/{}:create",
			Substitutions: []*bindingSubstitution{
				{
					FieldAccessor: test.WantAccessor,
					FieldName:     test.WantFieldName,
					Template:      []string{"*"},
				},
			},
		}
		method := &api.Method{
			Name:         "Create",
			ID:           ".test.Service.Create",
			InputType:    request,
			InputTypeID:  ".test.Request",
			OutputTypeID: ".test.Response",
			PathInfo: &api.PathInfo{
				Bindings: []*api.PathBinding{binding},
			},
		}
		service := &api.Service{
			Name:    "Service",
			ID:      ".test.Service",
			Package: "test",
			Methods: []*api.Method{method},
		}
		model := api.NewTestAPI(
			[]*api.Message{request, response},
			[]*api.Enum{},
			[]*api.Service{service})
		api.CrossReference(model)
		codec, err := newCodec("protobuf", map[string]string{})
		if err != nil {
			t.Fatal(err)
		}
		annotateModel(model, codec)
		if diff := cmp.Diff(wantBinding, binding.Codec); diff != "" {
			t.Errorf("mismatch in path binding annotations (-want, +got)\n:%s", diff)
		}

	}
}

func TestPathBindingAnnotationsErrors(t *testing.T) {
	field := &api.Field{
		Name:     "field",
		JSONName: "field",
		ID:       ".test.Request.field",
		Typez:    api.STRING_TYPE,
	}
	request := &api.Message{
		Name:    "Request",
		Package: "test",
		ID:      ".test.Request",
		Fields:  []*api.Field{field},
	}
	method := &api.Method{
		Name:         "Create",
		ID:           ".test.Service.Create",
		InputType:    request,
		InputTypeID:  ".test.Request",
		OutputTypeID: ".test.Response",
	}
	got := makeAccessors([]string{"not-a-field-name"}, method)
	var want []string
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestPathTemplateGeneration(t *testing.T) {
	tests := []struct {
		name    string
		binding *pathBindingAnnotation
		want    string
	}{
		{
			name: "Simple Literal",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/things",
			},
			want: "/v1/things",
		},
		{
			name: "Single Variable",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/things/{}",
				Substitutions: []*bindingSubstitution{
					{FieldName: "thing_id"},
				},
			},
			want: "/v1/things/{thing_id}",
		},
		{
			name: "Multiple Variables",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/projects/{}/locations/{}",
				Substitutions: []*bindingSubstitution{
					{FieldName: "project"},
					{FieldName: "location"},
				},
			},
			want: "/v1/projects/{project}/locations/{location}",
		},
		{
			name: "Variable with Complex Segment Match",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/{}/databases",
				Substitutions: []*bindingSubstitution{
					{FieldName: "name"},
				},
			},
			want: "/v1/{name}/databases",
		},
		{
			name: "Variable Capturing Remaining Path",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/objects/{}",
				Substitutions: []*bindingSubstitution{
					{FieldName: "object"},
				},
			},
			want: "/v1/objects/{object}",
		},
		{
			name: "Top-Level Single Wildcard",
			binding: &pathBindingAnnotation{
				PathFmt: "/{}",
				Substitutions: []*bindingSubstitution{
					{FieldName: "field"},
				},
			},
			want: "/{field}",
		},
		{
			name: "Path with Custom Verb",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/things/{}:customVerb",
				Substitutions: []*bindingSubstitution{
					{FieldName: "thing_id"},
				},
			},
			want: "/v1/things/{thing_id}:customVerb",
		},
		{
			name: "Nested fields",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/projects/{}/locations/{}/ids/{}:actionOnChild",
				Substitutions: []*bindingSubstitution{
					{FieldName: "child.project"},
					{FieldName: "child.location"},
					{FieldName: "child.id"},
				},
			},
			want: "/v1/projects/{child.project}/locations/{child.location}/ids/{child.id}:actionOnChild",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.binding.PathTemplate(); got != tt.want {
				t.Errorf("PathTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBindingSubstitutionTemplates(t *testing.T) {
	b := bindingSubstitution{
		Template: []string{"projects", "*", "locations", "*", "**"},
	}

	got := b.TemplateAsString()
	want := "projects/*/locations/*/**"

	if want != got {
		t.Errorf("TemplateAsString() failed. want=%q, got=%q", want, got)
	}

	got = b.TemplateAsArray()
	want = `&[Segment::Literal("projects/"), Segment::SingleWildcard, Segment::Literal("/locations/"), Segment::SingleWildcard, Segment::TrailingMultiWildcard]`

	if want != got {
		t.Errorf("TemplateAsArray() failed. want=`%s`, got=`%s`", want, got)
	}
}

func TestInternalMessageOverrides(t *testing.T) {
	public := &api.Message{
		Name: "Public",
		ID:   ".test.Public",
	}
	private1 := &api.Message{
		Name: "Private1",
		ID:   ".test.Private1",
	}
	private2 := &api.Message{
		Name: "Private2",
		ID:   ".test.Private2",
	}
	model := api.NewTestAPI([]*api.Message{public, private1, private2},
		[]*api.Enum{},
		[]*api.Service{})
	codec, err := newCodec("protobuf", map[string]string{
		"internal-types": ".test.Private1,.test.Private2",
	})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)

	if public.Codec.(*messageAnnotation).Internal {
		t.Errorf("Public method should not be flagged as internal")
	}
	if !private1.Codec.(*messageAnnotation).Internal {
		t.Errorf("Private method should not be flagged as internal")
	}
	if !private2.Codec.(*messageAnnotation).Internal {
		t.Errorf("Private method should not be flagged as internal")
	}
}

func TestRoutingRequired(t *testing.T) {
	message := &api.Message{
		Name: "Message",
		ID:   ".test.Message",
	}
	method := &api.Method{
		Name:         "DoFoo",
		ID:           ".test.Service.DoFoo",
		InputTypeID:  ".test.Message",
		OutputTypeID: ".test.Message",
		PathInfo:     &api.PathInfo{},
	}
	service := &api.Service{
		Name:    "FooService",
		ID:      ".test.FooService",
		Package: "test",
		Methods: []*api.Method{method},
	}
	model := api.NewTestAPI([]*api.Message{message},
		[]*api.Enum{},
		[]*api.Service{service})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	codec, err := newCodec("protobuf", map[string]string{
		"include-grpc-only-methods": "true",
		"routing-required":          "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)

	if !method.Codec.(*methodAnnotation).RoutingRequired {
		t.Errorf("codec setting `routing-required` not respected")
	}
}

func TestGenerateSetterSamples(t *testing.T) {
	model := serviceAnnotationsModel()
	codec, err := newCodec("protobuf", map[string]string{
		"generate-setter-samples": "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)
	if !model.Codec.(*modelAnnotations).GenerateSetterSamples {
		t.Errorf("GenerateSetterSamples should be true")
	}
}

func TestAnnotateModelWithDetailedTracing(t *testing.T) {
	tests := []struct {
		name    string
		options map[string]string
		want    bool
	}{
		{
			name:    "DetailedTracingTrue",
			options: map[string]string{"detailed-tracing-attributes": "true"},
			want:    true,
		},
		{
			name:    "DetailedTracingFalse",
			options: map[string]string{"detailed-tracing-attributes": "false"},
			want:    false,
		},
		{
			name:    "DetailedTracingMissing",
			options: map[string]string{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
			codec, err := newCodec("protobuf", tt.options)
			if err != nil {
				t.Fatal(err)
			}
			got := annotateModel(model, codec)
			if got.DetailedTracingAttributes != tt.want {
				t.Errorf("annotateModel() DetailedTracingAttributes = %v, want %v", got.DetailedTracingAttributes, tt.want)
			}
		})
	}
}

func TestSetterSampleAnnotations(t *testing.T) {
	enum := &api.Enum{
		Name:    "TestEnum",
		ID:      ".test.v1.TestEnum",
		Package: "test.v1",
	}
	message := &api.Message{
		Name:    "TestMessage",
		ID:      ".test.v1.TestMessage",
		Package: "test.v1",
		Fields: []*api.Field{
			{
				Name:    "enum_field",
				ID:      ".test.v1.TestMessage.enum_field",
				Typez:   api.ENUM_TYPE,
				TypezID: ".test.v1.TestEnum",
			},
			{
				Name:    "message_field",
				ID:      ".test.v1.TestMessage.message_field",
				Typez:   api.MESSAGE_TYPE,
				TypezID: ".test.v1.TestMessage",
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{enum}, []*api.Service{})
	api.CrossReference(model)
	codec, err := newCodec("protobuf", map[string]string{
		"generate-setter-samples": "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)

	if message.Codec.(*messageAnnotation).NameInExamples != "google_cloud_test_v1::model::TestMessage" {
		t.Errorf("mismatch in message NameInExamples: got %q", message.Codec.(*messageAnnotation).NameInExamples)
	}
	if enum.Codec.(*enumAnnotation).NameInExamples != "google_cloud_test_v1::model::TestEnum" {
		t.Errorf("mismatch in enum NameInExamples: got %q", enum.Codec.(*enumAnnotation).NameInExamples)
	}

	enumField := message.Fields[0]
	if enumField.Parent != message {
		t.Errorf("mismatch in enum_field.Parent")
	}
	if enumField.EnumType != enum {
		t.Errorf("mismatch in enum_field.EnumType")
	}

	messageField := message.Fields[1]
	if messageField.Parent != message {
		t.Errorf("mismatch in message_field.Parent")
	}
	if messageField.MessageType != message {
		t.Errorf("mismatch in message_field.MessageType")
	}
}

func TestEnumAnnotationsValuesForExamples(t *testing.T) {
	v_good1 := &api.EnumValue{Name: "GOOD_1", Number: 1}
	v_good2 := &api.EnumValue{Name: "GOOD_2", Number: 2}
	v_good3 := &api.EnumValue{Name: "GOOD_3", Number: 3}
	v_good4 := &api.EnumValue{Name: "GOOD_4", Number: 4}
	v_bad_deprecated := &api.EnumValue{Name: "BAD_DEPRECATED", Number: 5, Deprecated: true}
	v_bad_default := &api.EnumValue{Name: "BAD_DEFAULT", Number: 0}

	testCases := []struct {
		name         string
		values       []*api.EnumValue
		wantExamples []*enumValueForExamples
	}{
		{
			name:   "more than 3 good values",
			values: []*api.EnumValue{v_good1, v_good2, v_good3, v_good4},
			wantExamples: []*enumValueForExamples{
				{EnumValue: v_good1, Index: 0},
				{EnumValue: v_good2, Index: 1},
				{EnumValue: v_good3, Index: 2},
			},
		},
		{
			name:   "less than 3 good values",
			values: []*api.EnumValue{v_good1, v_good2, v_bad_deprecated},
			wantExamples: []*enumValueForExamples{
				{EnumValue: v_good1, Index: 0},
				{EnumValue: v_good2, Index: 1},
			},
		},
		{
			name:   "no good values",
			values: []*api.EnumValue{v_bad_default, v_bad_deprecated},
			wantExamples: []*enumValueForExamples{
				{EnumValue: v_bad_default, Index: 0},
				{EnumValue: v_bad_deprecated, Index: 1},
			},
		},
		{
			name:         "no values",
			values:       []*api.EnumValue{},
			wantExamples: []*enumValueForExamples{},
		},
		{
			name:   "mixed good and bad values",
			values: []*api.EnumValue{v_bad_default, v_good1, v_bad_deprecated, v_good2},
			wantExamples: []*enumValueForExamples{
				{EnumValue: v_good1, Index: 0},
				{EnumValue: v_good2, Index: 1},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enum := &api.Enum{
				Name:    "TestEnum",
				ID:      ".test.v1.TestEnum",
				Package: "test.v1",
				Values:  tc.values,
			}
			model := api.NewTestAPI([]*api.Message{}, []*api.Enum{enum}, []*api.Service{})
			if err := api.CrossReference(model); err != nil {
				t.Fatalf("CrossReference() failed: %v", err)
			}
			codec, err := newCodec("protobuf", map[string]string{})
			if err != nil {
				t.Fatal(err)
			}
			annotateModel(model, codec)

			got := enum.Codec.(*enumAnnotation).ValuesForExamples
			if diff := cmp.Diff(tc.wantExamples, got, cmpopts.IgnoreFields(api.EnumValue{}, "Parent")); diff != "" {
				t.Errorf("mismatch in ValuesForExamples (-want, +got)\n:%s", diff)
			}
		})
	}
}

func TestOneOfExampleFieldSelection(t *testing.T) {
	deprecated := &api.Field{
		Name:       "deprecated_field",
		ID:         ".test.Message.deprecated_field",
		Typez:      api.STRING_TYPE,
		IsOneOf:    true,
		Deprecated: true,
	}
	map_field := &api.Field{
		Name:    "map_field",
		ID:      ".test.Message.map_field",
		Typez:   api.MESSAGE_TYPE,
		TypezID: ".test.$Map",
		IsOneOf: true,
		Map:     true,
	}
	repeated := &api.Field{
		Name:     "repeated_field",
		ID:       ".test.Message.repeated_field",
		Typez:    api.STRING_TYPE,
		Repeated: true,
		IsOneOf:  true,
	}
	scalar := &api.Field{
		Name:    "scalar_field",
		ID:      ".test.Message.scalar_field",
		Typez:   api.INT32_TYPE,
		IsOneOf: true,
	}
	message_field := &api.Field{
		Name:    "message_field",
		ID:      ".test.Message.message_field",
		Typez:   api.MESSAGE_TYPE,
		TypezID: ".test.OneMessage",
		IsOneOf: true,
	}
	another_message_field := &api.Field{
		Name:    "another_message_field",
		ID:      ".test.Message.another_message_field",
		Typez:   api.MESSAGE_TYPE,
		TypezID: ".test.AnotherMessage",
		IsOneOf: true,
	}

	testCases := []struct {
		name   string
		fields []*api.Field
		want   *api.Field
	}{
		{
			name:   "all types",
			fields: []*api.Field{deprecated, map_field, repeated, scalar, message_field},
			want:   scalar,
		},
		{
			name:   "no primitives",
			fields: []*api.Field{deprecated, map_field, repeated, message_field},
			want:   message_field,
		},
		{
			name:   "only scalars and messages",
			fields: []*api.Field{message_field, scalar, another_message_field},
			want:   scalar,
		},
		{
			name:   "no scalars",
			fields: []*api.Field{deprecated, map_field, repeated},
			want:   repeated,
		},
		{
			name:   "only map and deprecated",
			fields: []*api.Field{deprecated, map_field},
			want:   map_field,
		},
		{
			name:   "only deprecated",
			fields: []*api.Field{deprecated},
			want:   deprecated,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			group := &api.OneOf{
				Name:   "test_oneof",
				ID:     ".test.Message.test_oneof",
				Fields: tc.fields,
			}
			message := &api.Message{
				Name:    "Message",
				ID:      ".test.Message",
				Package: "test",
				Fields:  tc.fields,
				OneOfs:  []*api.OneOf{group},
			}
			oneMesage := &api.Message{
				Name:    "OneMessage",
				ID:      ".test.OneMessage",
				Package: "test",
			}
			anotherMessage := &api.Message{
				Name:    "AnotherMessage",
				ID:      ".test.AnotherMessage",
				Package: "test",
			}
			model := api.NewTestAPI([]*api.Message{message, oneMesage, anotherMessage}, []*api.Enum{}, []*api.Service{})
			api.CrossReference(model)
			codec := createRustCodec()
			annotateModel(model, codec)

			got := group.Codec.(*oneOfAnnotation).ExampleField
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch in ExampleField (-want, +got)\n:%s", diff)
			}
		})
	}
}
