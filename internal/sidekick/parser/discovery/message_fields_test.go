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
)

func TestMakeMessageFields(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "package"
	input := &schema{
		Properties: []*property{
			{
				Name: "longField",
				Schema: &schema{
					ID:          ".package.Message.longField",
					Description: "The field description.",
					Type:        "string",
					Format:      "uint64",
				},
			},
			{
				Name: "intField",
				Schema: &schema{
					ID:          ".package.Message.intField",
					Description: "The field description.",
					Type:        "integer",
					Format:      "int32",
				},
			},
			{
				Name: "deprecatedField",
				Schema: &schema{
					ID:          ".package.Message.deprecatedField",
					Description: "The field description.",
					Type:        "integer",
					Format:      "uint32",
					Deprecated:  true,
				},
			},
			{
				Name: "arrayFieldString",
				Schema: &schema{
					ID:          ".package.Message.arrayFieldString",
					Description: "The field description.",
					Type:        "array",
					ItemSchema: &schema{
						Type: "string",
					},
				},
			},
			{
				Name: "arrayFieldObject",
				Schema: &schema{
					ID:          ".package.Message.arrayFieldObject",
					Description: "The field description.",
					Type:        "array",
					ItemSchema: &schema{
						Ref: "AnotherMessage",
					},
				},
			},
		},
	}
	message := &api.Message{ID: ".package.Message"}
	err := makeMessageFields(model, message, input)
	if err != nil {
		t.Fatal(err)
	}
	want := []*api.Field{
		{
			Name:          "deprecatedField",
			JSONName:      "deprecatedField",
			ID:            ".package.Message.deprecatedField",
			Documentation: "The field description.",
			Typez:         api.TypezUint32,
			TypezID:       "uint32",
			Deprecated:    true,
			Optional:      true,
		},
		{
			Name:          "intField",
			JSONName:      "intField",
			ID:            ".package.Message.intField",
			Documentation: "The field description.",
			Typez:         api.TypezInt32,
			TypezID:       "int32",
			Optional:      true,
		},
		{
			Name:          "longField",
			JSONName:      "longField",
			ID:            ".package.Message.longField",
			Documentation: "The field description.",
			Typez:         api.TypezUint64,
			TypezID:       "uint64",
			Optional:      true,
		},
		{
			Name:          "arrayFieldString",
			JSONName:      "arrayFieldString",
			ID:            ".package.Message.arrayFieldString",
			Documentation: "The field description.",
			Typez:         api.TypezString,
			TypezID:       "string",
			Repeated:      true,
		},
		{
			Name:          "arrayFieldObject",
			JSONName:      "arrayFieldObject",
			ID:            ".package.Message.arrayFieldObject",
			Documentation: "The field description.",
			Typez:         api.TypezMessage,
			TypezID:       ".package.AnotherMessage",
			Repeated:      true,
		},
	}
	less := func(a, b *api.Field) bool { return a.Name < b.Name }
	if diff := cmp.Diff(want, message.Fields, cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestMakeMessageFieldsError(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	input := &schema{
		Properties: []*property{
			{
				Name: "field",
				Schema: &schema{
					ID:          ".package.Message.field",
					Description: "The field description.",
					Type:        "--invalid--",
					Format:      "--unused--",
				},
			},
		},
	}
	message := &api.Message{ID: ".package.Message"}
	if err := makeMessageFields(model, message, input); err == nil {
		t.Errorf("expected error makeScalarField(), got=%v, Input=%v", message, input)
	}
}

func TestMakeArrayFieldError(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	input := &property{
		Name: "field",
		Schema: &schema{
			Type: "array",
			ItemSchema: &schema{
				ID:          ".package.Message.field",
				Description: "The field description.",
				Type:        "--invalid--",
				Format:      "--unused--",
			},
		},
	}
	message := &api.Message{ID: ".package.Message"}
	if got, err := makeArrayField(model, message, input); err == nil {
		t.Errorf("expected error makeScalarField(), got=%v, Input=%v", got, input)
	}
}

func TestMakeScalarFieldError(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	input := &property{
		Name: "field",
		Schema: &schema{
			ID:          ".package.Message.field",
			Description: "The field description.",
			Type:        "--invalid--",
			Format:      "--unused--",
		},
	}
	message := &api.Message{ID: ".package.Message"}
	if got, err := makeScalarField(model, message, input.Name, input.Schema); err == nil {
		t.Errorf("expected error makeScalarField(), got=%v, Input=%v", got, input)
	}
}

func TestScalarTypes(t *testing.T) {
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
			ID:          ".package.Message.field",
			Description: "The field description.",
			Type:        test.Type,
			Format:      test.Format,
		}
		gotTypez, gotTypeID, err := scalarType(model, ".package.Message", "field", input)
		if err != nil {
			t.Errorf("error in scalarType(), Type=%q, Format=%q: %v", test.Type, test.Format, err)
		}
		if gotTypez != test.WantTypez {
			t.Errorf("mismatched scalarType() Typez, want=%d, got=%d with Type=%q, Format=%q",
				test.WantTypez, gotTypez, test.Type, test.Format)
		}
		if gotTypeID != test.WantTypeID {
			t.Errorf("mismatched scalarType() TypeID, want=%q, got=%q with Type=%q, Format=%q",
				test.WantTypeID, gotTypeID, test.Type, test.Format)
		}
	}
}

func TestScalarUnknownType(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	input := &schema{
		ID:          ".package.Message.field",
		Description: "The field description.",
		Type:        "--invalid--",
		Format:      "--unused--",
	}
	if gotTypez, gotTypeID, err := scalarType(model, ".package.Message", "field", input); err == nil {
		t.Errorf("expected error scalarType(), gotTypez=%d, gotTypezID=%q, Input=%v", gotTypez, gotTypeID, input)
	}
}

func TestScalarUnknownFormats(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	for _, test := range []struct {
		Type string
	}{
		{"integer"},
		{"number"},
		{"string"},
		{"any"},
		{"object"},
	} {
		input := &schema{
			ID:          ".package.Message.field",
			Description: "The field description.",
			Type:        test.Type,
			Format:      "--invalid--",
		}
		if gotTypez, gotTypeID, err := scalarType(model, ".package.Message", "field", input); err == nil {
			t.Errorf("expected error scalarType(), gotTypez=%d, gotTypezID=%q, Input=%v", gotTypez, gotTypeID, input)
		}
	}
}
