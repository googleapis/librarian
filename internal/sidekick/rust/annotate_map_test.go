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

	"cloud.google.com/go/librarian/internal/sidekick/api"
)

func TestMapKeyAnnotations(t *testing.T) {
	for _, test := range []struct {
		wantSerdeAs string
		typez       api.Typez
	}{
		{"wkt::internal::I32", api.INT32_TYPE},
		{"wkt::internal::I32", api.SFIXED32_TYPE},
		{"wkt::internal::I32", api.SINT32_TYPE},
		{"wkt::internal::I64", api.INT64_TYPE},
		{"wkt::internal::I64", api.SFIXED64_TYPE},
		{"wkt::internal::I64", api.SINT64_TYPE},
		{"wkt::internal::U32", api.UINT32_TYPE},
		{"wkt::internal::U32", api.FIXED32_TYPE},
		{"wkt::internal::U64", api.UINT64_TYPE},
		{"wkt::internal::U64", api.FIXED64_TYPE},
		{"serde_with::DisplayFromStr", api.BOOL_TYPE},
	} {
		mapMessage := &api.Message{
			Name:    "$map<unused, unused>",
			ID:      "$map<unused, unused>",
			Package: "$",
			IsMap:   true,
			Fields: []*api.Field{
				{
					Name:    "key",
					ID:      "$map<unused, unused>.key",
					Typez:   test.typez,
					TypezID: "unused",
				},
				{
					Name:    "value",
					ID:      "$map<unused, unused>.value",
					Typez:   api.STRING_TYPE,
					TypezID: "unused",
				},
			},
		}
		field := &api.Field{
			Name:     "field",
			JSONName: "field",
			ID:       ".test.Message.field",
			Typez:    api.MESSAGE_TYPE,
			TypezID:  "$map<unused, unused>",
		}
		message := &api.Message{
			Name:          "TestMessage",
			Package:       "test",
			ID:            ".test.TestMessage",
			Documentation: "A test message.",
			Fields:        []*api.Field{field},
		}
		model := api.NewTestAPI([]*api.Message{message, mapMessage}, []*api.Enum{}, []*api.Service{})
		api.CrossReference(model)
		api.LabelRecursiveFields(model)
		codec, err := newCodec("protobuf", map[string]string{})
		if err != nil {
			t.Fatal(err)
		}
		annotateModel(model, codec)

		got := field.Codec.(*fieldAnnotations).SerdeAs
		want := fmt.Sprintf("std::collections::HashMap<%s, serde_with::Same>", test.wantSerdeAs)
		if got != want {
			t.Errorf("mismatch for %s, want=%q, got=%q", test.wantSerdeAs, want, got)
		}
	}
}

func TestMapValueAnnotations(t *testing.T) {
	for _, test := range []struct {
		spec        string
		typez       api.Typez
		typezID     string
		wantSerdeAs string
	}{
		{"protobuf", api.STRING_TYPE, "unused", "serde_with::Same"},
		{"disco", api.STRING_TYPE, "unused", "serde_with::Same"},
		{"protobuf", api.BYTES_TYPE, "unused", "serde_with::base64::Base64"},
		{"disco", api.BYTES_TYPE, "unused", "serde_with::base64::Base64<serde_with::base64::UrlSafe>"},
		{"protobuf", api.MESSAGE_TYPE, ".google.protobuf.BytesValue", "serde_with::base64::Base64"},
		{"disco", api.MESSAGE_TYPE, ".google.protobuf.BytesValue", "serde_with::base64::Base64<serde_with::base64::UrlSafe>"},

		{"protobuf", api.BOOL_TYPE, "unused", "serde_with::Same"},
		{"protobuf", api.INT32_TYPE, "unused", "wkt::internal::I32"},
		{"protobuf", api.SFIXED32_TYPE, "unused", "wkt::internal::I32"},
		{"protobuf", api.SINT32_TYPE, "unused", "wkt::internal::I32"},
		{"protobuf", api.INT64_TYPE, "unused", "wkt::internal::I64"},
		{"protobuf", api.SFIXED64_TYPE, "unused", "wkt::internal::I64"},
		{"protobuf", api.SINT64_TYPE, "unused", "wkt::internal::I64"},
		{"protobuf", api.UINT32_TYPE, "unused", "wkt::internal::U32"},
		{"protobuf", api.FIXED32_TYPE, "unused", "wkt::internal::U32"},
		{"protobuf", api.UINT64_TYPE, "unused", "wkt::internal::U64"},
		{"protobuf", api.FIXED64_TYPE, "unused", "wkt::internal::U64"},

		{"protobuf", api.MESSAGE_TYPE, ".google.protobuf.UInt64Value", "wkt::internal::U64"},
		{"protobuf", api.MESSAGE_TYPE, ".test.Message", "serde_with::Same"},
	} {
		mapMessage := &api.Message{
			Name:    "$map<unused, unused>",
			ID:      "$map<unused, unused>",
			Package: "$",
			IsMap:   true,
			Fields: []*api.Field{
				{
					Name:    "key",
					ID:      "$map<unused, unused>.key",
					Typez:   api.INT32_TYPE,
					TypezID: "unused",
				},
				{
					Name:    "value",
					ID:      "$map<unused, unused>.value",
					Typez:   test.typez,
					TypezID: test.typezID,
				},
			},
		}
		field := &api.Field{
			Name:     "field",
			JSONName: "field",
			ID:       ".test.Message.field",
			Typez:    api.MESSAGE_TYPE,
			TypezID:  "$map<unused, unused>",
		}
		message := &api.Message{
			Name:          "Message",
			Package:       "test",
			ID:            ".test.Message",
			Documentation: "A test message.",
			Fields:        []*api.Field{field},
		}
		model := api.NewTestAPI([]*api.Message{message, mapMessage}, []*api.Enum{}, []*api.Service{})
		api.CrossReference(model)
		api.LabelRecursiveFields(model)
		codec, err := newCodec(test.spec, map[string]string{})
		if err != nil {
			t.Fatal(err)
		}
		annotateModel(model, codec)

		got := field.Codec.(*fieldAnnotations).SerdeAs
		want := fmt.Sprintf("std::collections::HashMap<wkt::internal::I32, %s>", test.wantSerdeAs)
		if got != want {
			t.Errorf("mismatch for %v, want=%q, got=%q", test, want, got)
		}
	}
}

// A map without any SerdeAs mapping receives a special annotation.
func TestMapAnnotationsSameSame(t *testing.T) {
	mapMessage := &api.Message{
		Name:    "$map<string, string>",
		ID:      "$map<string, string>",
		Package: "$",
		IsMap:   true,
		Fields: []*api.Field{
			{
				Name:    "key",
				ID:      "$map<string, string>.key",
				Typez:   api.STRING_TYPE,
				TypezID: "unused",
			},
			{
				Name:  "value",
				ID:    "$map<string, string>.value",
				Typez: api.STRING_TYPE,
			},
		},
	}
	field := &api.Field{
		Name:     "field",
		JSONName: "field",
		ID:       ".test.Message.field",
		Typez:    api.MESSAGE_TYPE,
		TypezID:  "$map<unused, unused>",
	}
	message := &api.Message{
		Name:          "Message",
		Package:       "test",
		ID:            ".test.Message",
		Documentation: "A test message.",
		Fields:        []*api.Field{field},
	}
	model := api.NewTestAPI([]*api.Message{message, mapMessage}, []*api.Enum{}, []*api.Service{})
	api.CrossReference(model)
	api.LabelRecursiveFields(model)
	codec, err := newCodec("protobuf", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	annotateModel(model, codec)

	got := field.Codec.(*fieldAnnotations).SerdeAs
	if got != "" {
		t.Errorf("mismatch for %v, got=%q", mapMessage, got)
	}
}
