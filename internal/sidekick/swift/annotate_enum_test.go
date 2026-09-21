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
	"github.com/googleapis/librarian/internal/config"
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
				api.NewTestEnumValue("COLOR_UNSPECIFIED", 0),
				api.NewTestEnumValue("COLOR_RED", 1),
			},
			want: &enumAnnotations{
				Name:               "Color",
				FullyQualifiedName: "Color",
				DocLines:           []string{"A color enum.", "With two lines."},
				DefaultCaseName:    "unspecified",
				UnknownIntName:     "unknownIntValue",
				UnknownStringName:  "unknownStringValue",
				ProtoTypeName:      "Test_Color",
				ModulePath:         "",
			},
		},
		{
			name:          "escaped name",
			enumName:      "Protocol",
			documentation: "An enum named Protocol.",
			values: []*api.EnumValue{
				api.NewTestEnumValue("PROTOCOL_UNSPECIFIED", 0),
			},
			want: &enumAnnotations{
				Name:               "Protocol_",
				FullyQualifiedName: "Protocol_",
				DocLines:           []string{"An enum named Protocol."},
				DefaultCaseName:    "unspecified",
				UnknownIntName:     "unknownIntValue",
				UnknownStringName:  "unknownStringValue",
				ProtoTypeName:      "Test_Protocol_",
				ModulePath:         "",
			},
		},
		{
			name:          "duplicate unknown",
			enumName:      "Weird",
			documentation: "An enum named Weird.",
			values: []*api.EnumValue{
				api.NewTestEnumValue("WEIRD_UNSPECIFIED", 0),
				api.NewTestEnumValue("UNKNOWN_INT_VALUE", 1),
				api.NewTestEnumValue("UNKNOWN_STRING_VALUE", 2),
			},
			want: &enumAnnotations{
				Name:               "Weird",
				FullyQualifiedName: "Weird",
				DocLines:           []string{"An enum named Weird."},
				DefaultCaseName:    "unspecified",
				UnknownIntName:     "unknownIntValue_",
				UnknownStringName:  "unknownStringValue_",
				ProtoTypeName:      "Test_Weird",
				ModulePath:         "",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			enum := api.NewTestEnum(test.enumName).
				WithDocumentation(test.documentation).
				WithValues(test.values...)
			model := api.NewTestAPI(nil, []*api.Enum{enum}, nil)
			codec := newTestCodec(t, model, nil)
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
	enum := api.NewTestEnum("Empty")
	model := api.NewTestAPI(nil, []*api.Enum{enum}, nil)
	codec := newTestCodec(t, model, nil)

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
	enum := api.NewTestEnum("Color").
		WithValues(
			api.NewTestEnumValue("COLOR_UNSPECIFIED", 0),
			api.NewTestEnumValue("COLOR_RED", 1),
		)
	model := api.NewTestAPI(nil, []*api.Enum{enum}, nil)
	codec, err := newCodec(model, &config.Library{}, &config.SwiftModule{ModulePath: "TestProtos"}, ".")
	if err != nil {
		t.Fatal(err)
	}
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	want := &enumAnnotations{
		Name:               "Color",
		FullyQualifiedName: "Color",
		DefaultCaseName:    "unspecified",
		UnknownIntName:     "unknownIntValue",
		UnknownStringName:  "unknownStringValue",
		ModulePath:         "TestProtos",
		ProtoTypeName:      "TestProtos.Test_Color",
	}
	if diff := cmp.Diff(want, enum.Codec, cmpopts.IgnoreFields(enumAnnotations{}, "Model")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateEnum_NestedModulePath(t *testing.T) {
	parent := api.NewTestMessage("OuterMessage")
	enum := api.NewTestEnum("InnerEnum").
		WithParent(parent).
		WithValues(
			api.NewTestEnumValue("INNER_ENUM_UNSPECIFIED", 0),
			api.NewTestEnumValue("INNER_ENUM_VALUE_A", 1),
		)
	model := api.NewTestAPI([]*api.Message{parent}, []*api.Enum{enum}, nil)
	codec, err := newCodec(model, &config.Library{}, &config.SwiftModule{ModulePath: "TestProtos"}, ".")
	if err != nil {
		t.Fatal(err)
	}
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	want := &enumAnnotations{
		Name:               "InnerEnum",
		FullyQualifiedName: "OuterMessage.InnerEnum",
		DefaultCaseName:    "unspecified",
		UnknownIntName:     "unknownIntValue",
		UnknownStringName:  "unknownStringValue",
		ModulePath:         "TestProtos",
		ProtoTypeName:      "TestProtos.Test_OuterMessage.InnerEnum",
	}
	if diff := cmp.Diff(want, enum.Codec, cmpopts.IgnoreFields(enumAnnotations{}, "Model")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
