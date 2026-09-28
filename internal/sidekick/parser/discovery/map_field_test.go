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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/api/apitest"
)

func newTestDiscoveryMap(id string, valueTypez api.Typez, valueTypezID string) *api.Message {
	m := api.NewTestMessage(id).
		WithIsMap().
		WithPackage("$").
		WithID(id).
		WithDocumentation(id).
		WithFields(
			api.NewTestField("key").WithType(api.TypezString).WithTypezID("string").WithJSONName(""),
			api.NewTestField("value").WithType(valueTypez).WithTypezID(valueTypezID).WithJSONName(""),
		)
	for _, f := range m.Fields {
		f.Parent = nil
	}
	return m
}

func newTestDiscoveryMessage(fields ...*api.Field) *api.Message {
	m := api.NewTestMessage("Message").
		WithPackage("package").
		WithFields(fields...)
	for _, f := range m.Fields {
		f.Parent = nil
	}
	return m
}

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
	message := api.NewTestMessage("Message").WithPackage("package")
	if err := makeMessageFields(model, message, input); err != nil {
		t.Fatal(err)
	}

	wantMessage := newTestDiscoveryMessage(
		api.NewTestField("labels").
			WithDocumentation("Lots of messages have labels.").
			WithDeprecated(true).
			WithType(api.TypezMessage).
			WithTypezID("$map<string, string>").
			WithMap(),
	)
	apitest.CheckMessage(t, message, wantMessage)

	wantMap := newTestDiscoveryMap("$map<string, string>", api.TypezString, "string")
	gotMap := model.Message(wantMap.ID)
	if gotMap == nil {
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
	message := api.NewTestMessage("Message").WithPackage("package")
	if err := makeMessageFields(model, message, input); err != nil {
		t.Fatal(err)
	}

	wantMessage := newTestDiscoveryMessage(
		api.NewTestField("objectMapField").
			WithDocumentation("The description for objectMapField.").
			WithDeprecated(true).
			WithType(api.TypezMessage).
			WithTypezID("$map<string, .package.SomeOtherMessage>").
			WithMap(),
	)
	apitest.CheckMessage(t, message, wantMessage)

	wantMap := newTestDiscoveryMap("$map<string, .package.SomeOtherMessage>", api.TypezMessage, ".package.SomeOtherMessage")
	gotMap := model.Message(wantMap.ID)
	if gotMap == nil {
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
	message := api.NewTestMessage("Message").WithPackage("package")
	if err := makeMessageFields(model, message, input); err != nil {
		t.Fatal(err)
	}

	wantMessage := newTestDiscoveryMessage(
		api.NewTestField("enumMapField").
			WithDocumentation("The description for enumMapField.").
			WithDeprecated(true).
			WithType(api.TypezMessage).
			WithTypezID("$map<string, .package.Message.enumMapField>").
			WithMap(),
	)
	apitest.CheckMessage(t, message, wantMessage)

	wantMap := newTestDiscoveryMap("$map<string, .package.Message.enumMapField>", api.TypezEnum, ".package.Message.enumMapField")
	gotMap := model.Message(wantMap.ID)
	if gotMap == nil {
		t.Fatalf("missing map message %s", wantMap.ID)
	}
	apitest.CheckMessage(t, gotMap, wantMap)

	wantEnum := api.NewTestEnum("enumMapField").
		WithPackage("package").
		WithID(".package.Message.enumMapField").
		WithDocumentation("The enumerated type for the [enumMapField][package.Message.enumMapField] field.").
		WithValues(
			api.NewTestEnumValue("ACTIVE", 0).
				WithID(".package.Message.enumMapField.ACTIVE").
				WithDocumentation("The description for the ACTIVE state."),
			api.NewTestEnumValue("PROVISIONING", 1).
				WithID(".package.Message.enumMapField.PROVISIONING").
				WithDocumentation("The description for the PROVISIONING state."),
		)
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
		message := api.NewTestMessage("Message").WithPackage("package")
		if err := makeMessageFields(model, message, input); err != nil {
			t.Error(err)
			continue
		}
		wantFields := []*api.Field{
			api.NewTestMessage("Message").
				WithPackage("package").
				WithFields(
					api.NewTestField("mapField").
						WithDocumentation("The description for mapField.").
						WithType(api.TypezMessage).
						WithMap(),
				).Fields[0],
		}
		if diff := cmp.Diff(wantFields, message.Fields, cmpopts.IgnoreFields(api.Field{}, "TypezID", "Parent")); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
			continue
		}
		mapMessage := model.Message(message.Fields[0].TypezID)
		if mapMessage == nil {
			t.Errorf("missing map message %s", message.Fields[0].TypezID)
			continue
		}
		if len(mapMessage.Fields) != 2 {
			t.Errorf("expected exactly two fields, got=%v", mapMessage.Fields)
			continue
		}
		got := mapMessage.Fields[1]
		want := api.NewTestMessage(mapMessage.ID).
			WithID(mapMessage.ID).
			WithFields(
				api.NewTestField("key").WithJSONName(""),
				api.NewTestField("value").
					WithType(test.WantTypez).
					WithTypezID(test.WantTypeID).
					WithJSONName(""),
			).Fields[1]
		if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(api.Field{}, "Parent")); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
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
	message := api.NewTestMessage("Message").WithPackage("package")
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
	message := api.NewTestMessage("Message").WithPackage("package")
	if err := makeMessageFields(model, message, input); err == nil {
		t.Errorf("expected error in map with invalid value format, got=%v", message)
	}
}
