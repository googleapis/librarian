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

package discovery

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/api/apitest"
)

func TestMapFields(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "package"
	input := &schema{
		Properties: []*property{
			{
				Name: "labels",
				Schema: &schema{
					Description: "Lots of messages have labels.",
					Deprecated:  true,
					Type:        "object",
					AdditionalProperties: &schema{
						Type: "string",
					},
				},
			},
		},
	}
	message := &api.Message{ID: ".package.Message"}
	if err := makeMessageFields(model, message, input); err != nil {
		t.Fatal(err)
	}

	wantMessage := &api.Message{
		ID: ".package.Message",
		Fields: []*api.Field{
			{
				Name:          "labels",
				JSONName:      "labels",
				ID:            ".package.Message.labels",
				Documentation: "Lots of messages have labels.",
				Deprecated:    true,
				Typez:         api.TypezMessage,
				TypezID:       "$map<string, string>",
				Map:           true,
			},
		},
	}
	apitest.CheckMessage(t, message, wantMessage)

	wantMap := &api.Message{
		IsMap:         true,
		Name:          "$map<string, string>",
		ID:            "$map<string, string>",
		Documentation: "$map<string, string>",
		Package:       "$",
		Fields: []*api.Field{
			{
				Name:    "key",
				ID:      "$map<string, string>.key",
				Typez:   api.TypezString,
				TypezID: "string",
			},
			{
				Name:    "value",
				ID:      "$map<string, string>.value",
				Typez:   api.TypezString,
				TypezID: "string",
			},
		},
	}
	gotMap, ok := model.State.MessageByID[wantMap.ID]
	if !ok {
		t.Fatalf("missing map message %s", wantMap.ID)
	}
	apitest.CheckMessage(t, gotMap, wantMap)
}

func TestMapFieldWithObjectValues(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "package"
	input := &schema{
		Properties: []*property{
			{
				Name: "objectMapField",
				Schema: &schema{
					Description: "The description for objectMapField.",
					Deprecated:  true,
					Type:        "object",
					AdditionalProperties: &schema{
						Type: "object",
						Ref:  "SomeOtherMessage",
					},
				},
			},
		},
	}
	message := &api.Message{ID: ".package.Message"}
	if err := makeMessageFields(model, message, input); err != nil {
		t.Fatal(err)
	}

	wantMessage := &api.Message{
		ID: ".package.Message",
		Fields: []*api.Field{
			{
				Name:          "objectMapField",
				JSONName:      "objectMapField",
				ID:            ".package.Message.objectMapField",
				Documentation: "The description for objectMapField.",
				Deprecated:    true,
				Typez:         api.TypezMessage,
				TypezID:       "$map<string, .package.SomeOtherMessage>",
				Map:           true,
			},
		},
	}
	apitest.CheckMessage(t, message, wantMessage)

	wantMap := &api.Message{
		IsMap:         true,
		Name:          "$map<string, .package.SomeOtherMessage>",
		ID:            "$map<string, .package.SomeOtherMessage>",
		Documentation: "$map<string, .package.SomeOtherMessage>",
		Package:       "$",
		Fields: []*api.Field{
			{
				Name:    "key",
				ID:      "$map<string, .package.SomeOtherMessage>.key",
				Typez:   api.TypezString,
				TypezID: "string",
			},
			{
				Name:    "value",
				ID:      "$map<string, .package.SomeOtherMessage>.value",
				Typez:   api.TypezMessage,
				TypezID: ".package.SomeOtherMessage",
			},
		},
	}
	gotMap, ok := model.State.MessageByID[wantMap.ID]
	if !ok {
		t.Fatalf("missing map message %s", wantMap.ID)
	}
	apitest.CheckMessage(t, gotMap, wantMap)
}

func TestMapFieldWithEnumValues(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "package"
	input := &schema{
		Properties: []*property{
			{
				Name: "enumMapField",
				Schema: &schema{
					Description: "The description for enumMapField.",
					Type:        "object",
					Deprecated:  true,
					AdditionalProperties: &schema{
						Type: "string",
						Enums: []string{
							"ACTIVE",
							"PROVISIONING",
						},
						EnumDescriptions: []string{
							"The description for the ACTIVE state.",
							"The description for the PROVISIONING state.",
						},
					},
				},
			},
		},
	}
	message := &api.Message{ID: ".package.Message"}
	if err := makeMessageFields(model, message, input); err != nil {
		t.Fatal(err)
	}

	wantMessage := &api.Message{
		ID: ".package.Message",
		Fields: []*api.Field{
			{
				Name:          "enumMapField",
				JSONName:      "enumMapField",
				ID:            ".package.Message.enumMapField",
				Documentation: "The description for enumMapField.",
				Deprecated:    true,
				Typez:         api.TypezMessage,
				TypezID:       "$map<string, .package.Message.enumMapField>",
				Map:           true,
			},
		},
	}
	apitest.CheckMessage(t, message, wantMessage)

	wantMap := &api.Message{
		IsMap:         true,
		Name:          "$map<string, .package.Message.enumMapField>",
		ID:            "$map<string, .package.Message.enumMapField>",
		Documentation: "$map<string, .package.Message.enumMapField>",
		Package:       "$",
		Fields: []*api.Field{
			{
				Name:    "key",
				ID:      "$map<string, .package.Message.enumMapField>.key",
				Typez:   api.TypezString,
				TypezID: "string",
			},
			{
				Name:    "value",
				ID:      "$map<string, .package.Message.enumMapField>.value",
				Typez:   api.TypezEnum,
				TypezID: ".package.Message.enumMapField",
			},
		},
	}
	gotMap, ok := model.State.MessageByID[wantMap.ID]
	if !ok {
		t.Fatalf("missing map message %s", wantMap.ID)
	}
	apitest.CheckMessage(t, gotMap, wantMap)

	wantEnum := &api.Enum{
		Name:          "enumMapField",
		ID:            ".package.Message.enumMapField",
		Documentation: "The enumerated type for the [enumMapField][package.Message.enumMapField] field.",
		Values: []*api.EnumValue{
			{
				Name:          "ACTIVE",
				ID:            ".package.Message.enumMapField.ACTIVE",
				Number:        0,
				Documentation: "The description for the ACTIVE state.",
			},
			{
				Name:          "PROVISIONING",
				ID:            ".package.Message.enumMapField.PROVISIONING",
				Number:        1,
				Documentation: "The description for the PROVISIONING state.",
			},
		},
	}
	wantEnum.UniqueNumberValues = wantEnum.Values
	gotEnum := model.Enum(wantEnum.ID)
	if gotEnum == nil {
		t.Fatalf("missing enum %s", wantEnum.ID)
	}
	apitest.CheckEnum(t, *gotEnum, *wantEnum)
}

func TestMapScalarTypes(t *testing.T) {
	for _, test := range []struct {
		Type       string
		Format     string
		WantTypez  api.Typez
		WantTypeID string
	}{
		{"boolean", "", api.TypezBool, "bool"},
		{"integer", "int32", api.TypezInt32, "int32"},
		{"integer", "uint32", api.TypezUint32, "uint32"},
		{"integer", "int64", api.TypezInt64, "int64"},
		{"integer", "uint64", api.TypezUint64, "uint64"},
		{"number", "float", api.TypezFloat, "float"},
		{"number", "double", api.TypezDouble, "double"},
		{"string", "", api.TypezString, "string"},
		{"string", "byte", api.TypezBytes, "bytes"},
		{"string", "date", api.TypezString, "string"},
		{"string", "google-duration", api.TypezMessage, ".google.protobuf.Duration"},
		{"string", "google-datetime", api.TypezMessage, ".google.protobuf.Timestamp"},
		{"string", "date-time", api.TypezMessage, ".google.protobuf.Timestamp"},
		{"string", "google-fieldmask", api.TypezMessage, ".google.protobuf.FieldMask"},
		{"string", "int64", api.TypezInt64, "int64"},
		{"string", "uint64", api.TypezUint64, "uint64"},
		{"any", "google.protobuf.Value", api.TypezMessage, ".google.protobuf.Value"},
		{"object", "google.protobuf.Struct", api.TypezMessage, ".google.protobuf.Struct"},
		{"object", "google.protobuf.Any", api.TypezMessage, ".google.protobuf.Any"},
	} {
		model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
		input := &schema{
			Properties: []*property{
				{
					Name: "mapField",
					Schema: &schema{
						Description: "The description for mapField.",
						Type:        "object",
						AdditionalProperties: &schema{
							Type:   test.Type,
							Format: test.Format,
						},
					},
				},
			},
		}
		message := &api.Message{ID: ".package.Message"}
		if err := makeMessageFields(model, message, input); err != nil {
			t.Error(err)
			continue
		}
		wantFields := []*api.Field{
			{
				Name:          "mapField",
				JSONName:      "mapField",
				ID:            ".package.Message.mapField",
				Documentation: "The description for mapField.",
				Typez:         api.TypezMessage,
				Map:           true,
			},
		}
		if diff := cmp.Diff(wantFields, message.Fields, cmpopts.IgnoreFields(api.Field{}, "TypezID")); diff != "" {
			t.Errorf("mismatch (-want, +got):\n%s", diff)
			continue
		}
		mapMessage, ok := model.State.MessageByID[message.Fields[0].TypezID]
		if !ok {
			t.Errorf("missing map message %s", message.Fields[0].TypezID)
		}
		if len(mapMessage.Fields) != 2 {
			t.Errorf("expected exactly two fields, got=%v", mapMessage.Fields)
			continue
		}
		got := mapMessage.Fields[1]
		want := &api.Field{
			Name:    "value",
			ID:      fmt.Sprintf("%s.value", mapMessage.ID),
			Typez:   test.WantTypez,
			TypezID: test.WantTypeID,
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch on value field (-want, +got):\n%s", diff)
		}
	}
}

func TestMapFieldEnumError(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "package"
	input := &schema{
		Properties: []*property{
			{
				Name: "badMapField",
				Schema: &schema{
					Description: "The networking tier.",
					Type:        "object",
					AdditionalProperties: &schema{
						Enums:            []string{"VALUE", "MISSING_DESCRIPTION"},
						EnumDescriptions: []string{"value"},
					},
				},
			},
		},
	}
	message := &api.Message{ID: ".package.Message"}
	if err := makeMessageFields(model, message, input); err == nil {
		t.Errorf("expected error in map with invalid enum, got=%v", message)
	}
}

func TestMapFieldScalarError(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "package"
	input := &schema{
		Properties: []*property{
			{
				Name: "badMapField",
				Schema: &schema{
					Description: "The networking tier.",
					Type:        "object",
					AdditionalProperties: &schema{
						Type:   "string",
						Format: "--invalid--",
					},
				},
			},
		},
	}
	message := &api.Message{ID: ".package.Message"}
	if err := makeMessageFields(model, message, input); err == nil {
		t.Errorf("expected error in map with invalid value format, got=%v", message)
	}
}
