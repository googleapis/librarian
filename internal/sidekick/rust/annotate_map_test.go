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

	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestMapKeyAnnotations(t *testing.T) {
	for _, test := range []struct {
		wantSerdeAs string
		typez       api.Typez
	}{
		{"wkt::internal::I32", api.TypezInt32},
		{"wkt::internal::I32", api.TypezSfixed32},
		{"wkt::internal::I32", api.TypezSint32},
		{"wkt::internal::I64", api.TypezInt64},
		{"wkt::internal::I64", api.TypezSfixed64},
		{"wkt::internal::I64", api.TypezSint64},
		{"wkt::internal::U32", api.TypezUint32},
		{"wkt::internal::U32", api.TypezFixed32},
		{"wkt::internal::U64", api.TypezUint64},
		{"wkt::internal::U64", api.TypezFixed64},
		{"serde_with::DisplayFromStr", api.TypezBool},
	} {
		t.Run(test.wantSerdeAs, func(t *testing.T) {
			mapMessage := api.NewTestMessage("$map<unused, unused>").
				WithPackage("$").
				WithID("$map<unused, unused>").
				WithFields(
					api.NewTestField("key").
						WithType(test.typez),
					api.NewTestField("value").
						WithType(api.TypezString),
				).
				WithIsMap()

			field := api.NewTestField("field").
				WithMessageType(mapMessage)

			message := api.NewTestMessage("TestMessage").
				WithDocumentation("A test message.").
				WithFields(field)

			model := api.NewTestAPI([]*api.Message{message, mapMessage}, nil, nil)
			api.CrossReference(model)
			api.LabelRecursiveFields(model)
			codec, err := newCodec(libconfig.SpecProtobuf, map[string]string{})
			codec.packageMapping = map[string]*packagez{
				"test":            {name: "google-cloud-test"},
				"google.protobuf": {name: "wkt"},
				"$":               {name: "internal-detail"},
			}
			if err != nil {
				t.Fatal(err)
			}
			annotateModel(model, codec)

			got := field.Codec.(*fieldAnnotations).SerdeAs
			want := fmt.Sprintf("std::collections::HashMap<%s, serde_with::Same>", test.wantSerdeAs)
			if got != want {
				t.Errorf("mismatch for %s, want=%q, got=%q", test.wantSerdeAs, want, got)
			}
		})
	}
}

func TestMapValueAnnotations(t *testing.T) {
	bytesValue := api.NewTestMessage("BytesValue").WithPackage("google.protobuf")
	uint64Value := api.NewTestMessage("UInt64Value").WithPackage("google.protobuf")
	testMessage := api.NewTestMessage("Message").WithPackage("test")
	for _, test := range []struct {
		spec        string
		typez       api.Typez
		valueMsg    *api.Message
		wantSerdeAs string
	}{
		{spec: libconfig.SpecProtobuf, typez: api.TypezString, wantSerdeAs: "serde_with::Same"},
		{spec: libconfig.SpecDiscovery, typez: api.TypezString, wantSerdeAs: "serde_with::Same"},
		{spec: libconfig.SpecProtobuf, typez: api.TypezBytes, wantSerdeAs: "serde_with::base64::Base64"},
		{spec: libconfig.SpecDiscovery, typez: api.TypezBytes, wantSerdeAs: "serde_with::base64::Base64<serde_with::base64::UrlSafe>"},
		{spec: libconfig.SpecProtobuf, valueMsg: bytesValue, wantSerdeAs: "serde_with::base64::Base64"},
		{spec: libconfig.SpecDiscovery, valueMsg: bytesValue, wantSerdeAs: "serde_with::base64::Base64<serde_with::base64::UrlSafe>"},

		{spec: libconfig.SpecProtobuf, typez: api.TypezBool, wantSerdeAs: "serde_with::Same"},
		{spec: libconfig.SpecProtobuf, typez: api.TypezInt32, wantSerdeAs: "wkt::internal::I32"},
		{spec: libconfig.SpecProtobuf, typez: api.TypezSfixed32, wantSerdeAs: "wkt::internal::I32"},
		{spec: libconfig.SpecProtobuf, typez: api.TypezSint32, wantSerdeAs: "wkt::internal::I32"},
		{spec: libconfig.SpecProtobuf, typez: api.TypezInt64, wantSerdeAs: "wkt::internal::I64"},
		{spec: libconfig.SpecProtobuf, typez: api.TypezSfixed64, wantSerdeAs: "wkt::internal::I64"},
		{spec: libconfig.SpecProtobuf, typez: api.TypezSint64, wantSerdeAs: "wkt::internal::I64"},
		{spec: libconfig.SpecProtobuf, typez: api.TypezUint32, wantSerdeAs: "wkt::internal::U32"},
		{spec: libconfig.SpecProtobuf, typez: api.TypezFixed32, wantSerdeAs: "wkt::internal::U32"},
		{spec: libconfig.SpecProtobuf, typez: api.TypezUint64, wantSerdeAs: "wkt::internal::U64"},
		{spec: libconfig.SpecProtobuf, typez: api.TypezFixed64, wantSerdeAs: "wkt::internal::U64"},

		{spec: libconfig.SpecProtobuf, valueMsg: uint64Value, wantSerdeAs: "wkt::internal::U64"},
		{spec: libconfig.SpecProtobuf, valueMsg: testMessage, wantSerdeAs: "serde_with::Same"},
	} {
		testName := fmt.Sprintf("%s_%v", test.spec, test.typez)
		if test.valueMsg != nil {
			testName = fmt.Sprintf("%s_message_%s", test.spec, test.valueMsg.ID)
		}
		t.Run(testName, func(t *testing.T) {
			valueField := api.NewTestField("value")
			if test.valueMsg != nil {
				valueField.WithMessageType(test.valueMsg)
			} else {
				valueField.WithType(test.typez)
			}
			mapMessage := api.NewTestMessage("$map<unused, unused>").
				WithPackage("$").
				WithID("$map<unused, unused>").
				WithFields(
					api.NewTestField("key").WithType(api.TypezInt32),
					valueField,
				).
				WithIsMap()

			field := api.NewTestField("field").WithMessageType(mapMessage)

			message := api.NewTestMessage("Message").
				WithPackage("test").
				WithDocumentation("A test message.").
				WithFields(field)

			model := api.NewTestAPI([]*api.Message{message, mapMessage}, nil, nil)
			api.CrossReference(model)
			api.LabelRecursiveFields(model)
			codec := newTestCodec(t, test.spec, "test", map[string]string{})
			annotateModel(model, codec)

			got := field.Codec.(*fieldAnnotations).SerdeAs
			want := fmt.Sprintf("std::collections::HashMap<wkt::internal::I32, %s>", test.wantSerdeAs)
			if got != want {
				t.Errorf("mismatch for %v, want=%q, got=%q", test, want, got)
			}
		})
	}
}

// A map without any SerdeAs mapping receives a special annotation.
func TestMapAnnotationsSameSame(t *testing.T) {
	mapMessage := api.NewTestMessage("$map<string, string>").
		WithPackage("$").
		WithID("$map<string, string>").
		WithFields(
			api.NewTestField("key").
				WithType(api.TypezString),
			api.NewTestField("value").
				WithType(api.TypezString),
		).
		WithIsMap()

	field := api.NewTestField("field").
		WithMessageType(mapMessage)

	message := api.NewTestMessage("Message").
		WithDocumentation("A test message.").
		WithFields(field)

	model := api.NewTestAPI([]*api.Message{message, mapMessage}, nil, nil)
	api.CrossReference(model)
	api.LabelRecursiveFields(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "test", map[string]string{})
	_, err := annotateModel(model, codec)
	if err != nil {
		t.Fatal(err)
	}

	got := field.Codec.(*fieldAnnotations).SerdeAs
	if got != "" {
		t.Errorf("mismatch for %v, got=%q", mapMessage, got)
	}
}
