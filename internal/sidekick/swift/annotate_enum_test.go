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

package swift

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateEnum(t *testing.T) {
	for _, test := range []struct {
		name          string
		enumName      string
		documentation string
		values        []*api.EnumValue
		want          *enumAnnotations
	}{
		{
			name:          "basic enum",
			enumName:      "Color",
			documentation: "A color enum.\nWith two lines.",
			values: []*api.EnumValue{
				{Name: "COLOR_UNSPECIFIED", Number: 0},
				{Name: "COLOR_RED", Number: 1},
			},
			want: &enumAnnotations{
				Name:              "Color",
				DocLines:          []string{"A color enum.", "With two lines."},
				DefaultCaseName:   "unspecified",
				UnknownIntName:    "unknownIntValue",
				UnknownStringName: "unknownStringValue",
			},
		},
		{
			name:          "escaped name",
			enumName:      "Protocol",
			documentation: "An enum named Protocol.",
			values: []*api.EnumValue{
				{Name: "PROTOCOL_UNSPECIFIED", Number: 0},
			},
			want: &enumAnnotations{
				Name:              "Protocol_",
				DocLines:          []string{"An enum named Protocol."},
				DefaultCaseName:   "unspecified",
				UnknownIntName:    "unknownIntValue",
				UnknownStringName: "unknownStringValue",
			},
		},
		{
			name:          "duplicate unknown",
			enumName:      "Weird",
			documentation: "An enum named Weird.",
			values: []*api.EnumValue{
				{Name: "WEIRD_UNSPECIFIED", Number: 0},
				{Name: "UNKNOWN_INT_VALUE", Number: 1},
				{Name: "UNKNOWN_STRING_VALUE", Number: 2},
			},
			want: &enumAnnotations{
				Name:              "Weird",
				DocLines:          []string{"An enum named Weird."},
				DefaultCaseName:   "unspecified",
				UnknownIntName:    "unknownIntValue_",
				UnknownStringName: "unknownStringValue_",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			enum := &api.Enum{
				Name:               test.enumName,
				Documentation:      test.documentation,
				ID:                 ".test." + test.enumName,
				Package:            "test",
				Values:             test.values,
				UniqueNumberValues: test.values,
			}
			for _, ev := range enum.Values {
				ev.Parent = enum
			}
			model := api.NewTestAPI([]*api.Message{}, []*api.Enum{enum}, []*api.Service{})
			codec := newTestCodec(t, model, map[string]string{})
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, enum.Codec, cmpopts.IgnoreFields(enumAnnotations{}, "Model")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateEnum_Error(t *testing.T) {
	enum := &api.Enum{
		Name:    "Empty",
		ID:      ".test.Empty",
		Package: "test",
	}
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{enum}, []*api.Service{})
	codec := newTestCodec(t, model, map[string]string{})

	err := codec.annotateModel()
	if err == nil {
		t.Errorf("annotateModel() expected error for enum with no values, got nil")
	}
}

func TestAnnotateEnum_Gating(t *testing.T) {
	model := makeGatedTestModel()
	codec := newTestCodec(t, model, nil)
	codec.PerServiceTraits = true

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		enumName       string
		wantExpression string
	}{
		{"Shared enum used by both services", "SharedEnum", "Service1 || Service2"},
		{"Enum used by Service1 only", "Service1Enum", "Service1"},
		{"Enum used by Service2 only", "Service2Enum", "Service2"},
		{"Enum used by neither service", "UnusedEnum", "Service1 && Service2"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var enum *api.Enum
			for e := range model.AllEnums() {
				if e.Name == test.enumName {
					enum = e
					break
				}
			}
			if enum == nil {
				t.Fatalf("enum %s not found", test.enumName)
			}
			ann, ok := enum.Codec.(*enumAnnotations)
			if !ok {
				t.Fatalf("expected enum.Codec to be *enumAnnotations, got %T", enum.Codec)
			}

			if diff := cmp.Diff(test.wantExpression, ann.GateExpression()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}

			if !ann.IsGated() {
				t.Error("expected IsGated() to be true")
			}
		})
	}
}

func TestAnnotateEnum_ModulePath(t *testing.T) {
	enum := &api.Enum{
		Name:    "Color",
		ID:      ".test.Color",
		Package: "test",
		Values: []*api.EnumValue{
			{Name: "COLOR_UNSPECIFIED", Number: 0},
			{Name: "COLOR_RED", Number: 1},
		},
	}
	enum.UniqueNumberValues = enum.Values
	for _, ev := range enum.Values {
		ev.Parent = enum
	}
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{enum}, []*api.Service{})
	codec := newTestCodec(t, model, map[string]string{
		"module-path": "TestProtos",
	})
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	ann, ok := enum.Codec.(*enumAnnotations)
	if !ok {
		t.Fatalf("expected enum.Codec to be *enumAnnotations, got %T", enum.Codec)
	}

	if ann.ModulePath != "TestProtos" {
		t.Errorf("ann.ModulePath = %q, want %q", ann.ModulePath, "TestProtos")
	}

	wantProtoTypeName := "TestProtos.Test_Color"
	if ann.ProtoTypeName != wantProtoTypeName {
		t.Errorf("ann.ProtoTypeName = %q, want %q", ann.ProtoTypeName, wantProtoTypeName)
	}
}

func TestAnnotateEnum_NestedModulePath(t *testing.T) {
	parent := &api.Message{
		Name:    "OuterMessage",
		ID:      ".test.OuterMessage",
		Package: "test",
	}
	enum := &api.Enum{
		Name:    "InnerEnum",
		ID:      ".test.OuterMessage.InnerEnum",
		Package: "test",
		Values: []*api.EnumValue{
			{Name: "INNER_ENUM_UNSPECIFIED", Number: 0},
			{Name: "INNER_ENUM_VALUE_A", Number: 1},
		},
		Parent: parent,
	}
	enum.UniqueNumberValues = enum.Values
	for _, ev := range enum.Values {
		ev.Parent = enum
	}
	model := api.NewTestAPI([]*api.Message{parent}, []*api.Enum{enum}, []*api.Service{})
	codec := newTestCodec(t, model, map[string]string{
		"module-path": "TestProtos",
	})
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	ann, ok := enum.Codec.(*enumAnnotations)
	if !ok {
		t.Fatalf("expected enum.Codec to be *enumAnnotations, got %T", enum.Codec)
	}

	if ann.ModulePath != "TestProtos" {
		t.Errorf("ann.ModulePath = %q, want %q", ann.ModulePath, "TestProtos")
	}

	wantProtoTypeName := "TestProtos.Test_OuterMessage.InnerEnum"
	if ann.ProtoTypeName != wantProtoTypeName {
		t.Errorf("ann.ProtoTypeName = %q, want %q", ann.ProtoTypeName, wantProtoTypeName)
	}
}
