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
	"github.com/googleapis/librarian/internal/sidekick/internal/api"
)

func TestMakeMessageFields(t *testing.T) {
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
		},
	}
	got, err := makeMessageFields(".package.Message", input)
	if err != nil {
		t.Fatal(err)
	}
	want := []*api.Field{
		{
			Name:          "intField",
			JSONName:      "intField",
			Documentation: "The field description.",
			Typez:         api.INT32_TYPE,
			TypezID:       "int32",
		},
		{
			Name:          "longField",
			JSONName:      "longField",
			Documentation: "The field description.",
			Typez:         api.UINT64_TYPE,
			TypezID:       "uint64",
		},
	}
	less := func(a, b *api.Field) bool { return a.Name < b.Name }
	if diff := cmp.Diff(got, want, cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("message fields mismatch (-want, +got):\n%s", diff)
	}
}

func TestMakeMessageFieldsError(t *testing.T) {
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
	if got, err := makeMessageFields(".package.Message", input); err == nil {
		t.Errorf("expected error makeScalarField(), got=%v, Input=%v", got, input)
	}
}

func TestMakeScalarFieldError(t *testing.T) {
	input := &property{
		Name: "field",
		Schema: &schema{
			ID:          ".package.Message.field",
			Description: "The field description.",
			Type:        "--invalid--",
			Format:      "--unused--",
		},
	}
	if got, err := makeScalarField(".package.Message", input); err == nil {
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
		{"boolean", "", api.BOOL_TYPE, "bool"},
		{"integer", "int32", api.INT32_TYPE, "int32"},
		{"integer", "uint32", api.UINT32_TYPE, "uint32"},
		{"integer", "int64", api.INT64_TYPE, "int64"},
		{"integer", "uint64", api.UINT64_TYPE, "uint64"},
		{"number", "float", api.FLOAT_TYPE, "float"},
		{"number", "double", api.DOUBLE_TYPE, "double"},
		{"string", "", api.STRING_TYPE, "string"},
		{"string", "byte", api.BYTES_TYPE, "bytes"},
		{"string", "date", api.STRING_TYPE, "string"},
		{"string", "google-duration", api.MESSAGE_TYPE, ".google.protobuf.Duration"},
		{"string", "google-datetime", api.MESSAGE_TYPE, ".google.protobuf.Timestamp"},
		{"string", "date-time", api.MESSAGE_TYPE, ".google.protobuf.Timestamp"},
		{"string", "google-fieldmask", api.MESSAGE_TYPE, ".google.protobuf.FieldMask"},
		{"string", "int64", api.INT64_TYPE, "int64"},
		{"string", "uint64", api.UINT64_TYPE, "uint64"},
	} {
		input := &property{
			Name: "field",
			Schema: &schema{
				ID:          ".package.Message.field",
				Description: "The field description.",
				Type:        test.Type,
				Format:      test.Format,
			},
		}
		gotTypez, gotTypeID, err := scalarType(".package.Message", input)
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
	input := &property{
		Name: "field",
		Schema: &schema{
			ID:          ".package.Message.field",
			Description: "The field description.",
			Type:        "--invalid--",
			Format:      "--unused--",
		},
	}
	if gotTypez, gotTypeID, err := scalarType(".package.Message", input); err == nil {
		t.Errorf("expected error scalarType(), gotTypez=%d, gotTypezID=%q, Input=%v", gotTypez, gotTypeID, input)
	}
}

func TestScalarUnknownFormats(t *testing.T) {
	for _, test := range []struct {
		Type string
	}{
		{"integer"},
		{"number"},
		{"string"},
	} {
		input := &property{
			Name: "field",
			Schema: &schema{
				ID:          ".package.Message.field",
				Description: "The field description.",
				Type:        test.Type,
				Format:      "--invalid--",
			},
		}
		if gotTypez, gotTypeID, err := scalarType(".package.Message", input); err == nil {
			t.Errorf("expected error scalarType(), gotTypez=%d, gotTypezID=%q, Input=%v", gotTypez, gotTypeID, input)
		}
	}
}
