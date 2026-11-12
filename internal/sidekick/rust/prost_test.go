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
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestProstFieldType_ExternalAndSamePackageWkt(t *testing.T) {
	t.Run("external enum and message with prost-path", func(t *testing.T) {
		extEnum := api.NewTestEnum("DayOfWeek").WithPackage("google.type")
		extMsg := api.NewTestMessage("LatLng").WithPackage("google.type")
		wktTimestamp := api.NewTestMessage("Timestamp").WithPackage("google.protobuf")
		enumField := api.NewTestField("day").WithEnumType(extEnum)
		msgField := api.NewTestField("location").WithMessageType(extMsg)
		tsField := api.NewTestField("create_time").WithMessageType(wktTimestamp)
		message := api.NewTestMessage("Event").
			WithPackage("test.v1").
			WithFields(enumField, msgField, tsField)
		model := api.NewTestAPI([]*api.Message{message, extMsg, wktTimestamp}, []*api.Enum{extEnum}, nil).
			WithPackageName("test.v1")
		if err := api.CrossReference(model); err != nil {
			t.Fatal(err)
		}
		codec := newTestCodec(t, libconfig.SpecProtobuf, "google.type", map[string]string{
			"prost-path": "super::prost",
		})
		if _, err := annotateModel(model, codec); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff("super::prost::google::r#type::DayOfWeek", enumField.Codec().(*fieldAnnotations).ProstType); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff("super::prost::google::r#type::LatLng", msgField.Codec().(*fieldAnnotations).ProstType); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff("prost_types::Timestamp", tsField.Codec().(*fieldAnnotations).ProstType); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("same-package google.protobuf messages and enums", func(t *testing.T) {
		jsTypeEnum := api.NewTestEnum("JSType")
		fieldOptions := api.NewTestMessage("FieldOptions").
			WithPackage("google.protobuf").
			WithEnums(jsTypeEnum)
		nestedField := api.NewTestField("jstype").WithEnumType(jsTypeEnum)
		fieldOptions.WithFields(nestedField)

		descriptorProto := api.NewTestMessage("DescriptorProto").WithPackage("google.protobuf")
		msgField := api.NewTestField("nested_type").WithMessageType(descriptorProto).WithRepeated()
		optField := api.NewTestField("options").WithMessageType(fieldOptions).WithOptional()
		descriptorProto.WithFields(msgField, optField)

		model := api.NewTestAPI([]*api.Message{fieldOptions, descriptorProto}, nil, nil).
			WithPackageName("google.protobuf")
		if err := api.CrossReference(model); err != nil {
			t.Fatal(err)
		}
		codec := newTestCodec(t, libconfig.SpecProtobuf, "", nil)
		if _, err := annotateModel(model, codec); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff("field_options::JsType", nestedField.Codec().(*fieldAnnotations).ProstType); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff("DescriptorProto", msgField.Codec().(*fieldAnnotations).ProstType); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff("FieldOptions", optField.Codec().(*fieldAnnotations).ProstType); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}
