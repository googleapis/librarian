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
					Typez:   api.TypezString,
					TypezID: "unused",
				},
			},
		}
		field := &api.Field{
			Name:     "field",
			JSONName: "field",
			ID:       ".test.Message.field",
			Typez:    api.TypezMessage,
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
	}
}

func TestMapValueAnnotations(t *testing.T) {
	for _, test := range []struct {
		spec        string
		typez       api.Typez
		typezID     string
		wantSerdeAs string
	}{
		{libconfig.SpecProtobuf, api.TypezString, "unused", "serde_with::Same"},
		{libconfig.SpecDiscovery, api.TypezString, "unused", "serde_with::Same"},
		{libconfig.SpecProtobuf, api.TypezBytes, "unused", "serde_with::base64::Base64"},
		{libconfig.SpecDiscovery, api.TypezBytes, "unused", "serde_with::base64::Base64<serde_with::base64::UrlSafe>"},
		{libconfig.SpecProtobuf, api.TypezMessage, ".google.protobuf.BytesValue", "serde_with::base64::Base64"},
		{libconfig.SpecDiscovery, api.TypezMessage, ".google.protobuf.BytesValue", "serde_with::base64::Base64<serde_with::base64::UrlSafe>"},

		{libconfig.SpecProtobuf, api.TypezBool, "unused", "serde_with::Same"},
		{libconfig.SpecProtobuf, api.TypezInt32, "unused", "wkt::internal::I32"},
		{libconfig.SpecProtobuf, api.TypezSfixed32, "unused", "wkt::internal::I32"},
		{libconfig.SpecProtobuf, api.TypezSint32, "unused", "wkt::internal::I32"},
		{libconfig.SpecProtobuf, api.TypezInt64, "unused", "wkt::internal::I64"},
		{libconfig.SpecProtobuf, api.TypezSfixed64, "unused", "wkt::internal::I64"},
		{libconfig.SpecProtobuf, api.TypezSint64, "unused", "wkt::internal::I64"},
		{libconfig.SpecProtobuf, api.TypezUint32, "unused", "wkt::internal::U32"},
		{libconfig.SpecProtobuf, api.TypezFixed32, "unused", "wkt::internal::U32"},
		{libconfig.SpecProtobuf, api.TypezUint64, "unused", "wkt::internal::U64"},
		{libconfig.SpecProtobuf, api.TypezFixed64, "unused", "wkt::internal::U64"},

		{libconfig.SpecProtobuf, api.TypezMessage, ".google.protobuf.UInt64Value", "wkt::internal::U64"},
		{libconfig.SpecProtobuf, api.TypezMessage, ".test.Message", "serde_with::Same"},
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
					Typez:   api.TypezInt32,
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
			Typez:    api.TypezMessage,
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
		codec := newTestCodec(t, test.spec, "test", map[string]string{})
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
				Typez:   api.TypezString,
				TypezID: "unused",
			},
			{
				Name:  "value",
				ID:    "$map<string, string>.value",
				Typez: api.TypezString,
			},
		},
	}
	field := &api.Field{
		Name:     "field",
		JSONName: "field",
		ID:       ".test.Message.field",
		Typez:    api.TypezMessage,
		TypezID:  "$map<string, string>",
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
