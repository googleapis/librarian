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

package api

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestCrossReferenceOneOfs(t *testing.T) {
	var fields1 []*Field
	for i := range 4 {
		name := fmt.Sprintf("field%d", i)
		fields1 = append(fields1, &Field{
			Name:    name,
			ID:      ".test.Message." + name,
			Typez:   STRING_TYPE,
			IsOneOf: true,
		})
	}
	fields1 = append(fields1, &Field{
		Name:    "basic_field",
		ID:      ".test.Message.basic_field",
		Typez:   STRING_TYPE,
		IsOneOf: true,
	})
	group0 := &OneOf{
		Name:   "group0",
		Fields: []*Field{fields1[0], fields1[1]},
	}
	group1 := &OneOf{
		Name:   "group1",
		Fields: []*Field{fields1[2], fields1[3]},
	}
	message1 := &Message{
		Name:   "Message1",
		ID:     ".test.Message1",
		Fields: fields1,
		OneOfs: []*OneOf{group0, group1},
	}
	var fields2 []*Field
	for i := range 2 {
		name := fmt.Sprintf("field%d", i+4)
		fields2 = append(fields2, &Field{
			Name:    name,
			ID:      ".test.Message." + name,
			Typez:   STRING_TYPE,
			IsOneOf: true,
		})
	}
	group2 := &OneOf{
		Name:   "group2",
		Fields: []*Field{fields2[0], fields2[1]},
	}
	message2 := &Message{
		Name:   "Message2",
		ID:     ".test.Message2",
		OneOfs: []*OneOf{group2},
	}
	model := NewTestAPI([]*Message{message1, message2}, []*Enum{}, []*Service{})
	if err := CrossReference(model); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		field  *Field
		oneof  *OneOf
		parent *Message
	}{
		{fields1[0], group0, message1},
		{fields1[1], group0, message1},
		{fields1[2], group1, message1},
		{fields1[3], group1, message1},
		{fields1[4], nil, message1},
		{fields2[0], group2, message2},
		{fields2[1], group2, message2},
	} {
		if test.field.Group != test.oneof {
			t.Errorf("mismatched group for %s, got=%v, want=%v", test.field.Name, test.field.Group, test.oneof)
		}
		if test.field.Parent != test.parent {
			t.Errorf("mismatched parent for %s, got=%v, want=%v", test.field.Name, test.field.Parent, test.parent)
		}
	}
}

func TestCrossReferenceFields(t *testing.T) {
	messageT := &Message{
		Name: "MessageT",
		ID:   ".test.MessageT",
	}
	fieldM := &Field{
		Name:    "message_field",
		ID:      ".test.Message.message_field",
		Typez:   MESSAGE_TYPE,
		TypezID: ".test.MessageT",
	}
	enumT := &Enum{
		Name: "EnumT",
		ID:   ".test.EnumT",
	}
	fieldE := &Field{
		Name:    "enum_field",
		ID:      ".test.Message.enum_field",
		Typez:   ENUM_TYPE,
		TypezID: ".test.EnumT",
	}
	message := &Message{
		Name:   "Message",
		ID:     ".test.Message",
		Fields: []*Field{fieldM, fieldE},
	}

	model := NewTestAPI([]*Message{messageT, message}, []*Enum{enumT}, []*Service{})
	if err := CrossReference(model); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		field  *Field
		parent *Message
	}{
		{fieldM, message},
		{fieldE, message},
	} {
		if test.field.Parent != test.parent {
			t.Errorf("mismatched parent for %s, got=%v, want=%v", test.field.Name, test.field.Parent, test.parent)
		}
	}
	if fieldM.MessageType != messageT {
		t.Errorf("mismatched message type for %s, got%v, want=%v", fieldM.Name, fieldM.MessageType, messageT)
	}
	if fieldE.EnumType != enumT {
		t.Errorf("mismatched enum type for %s, got%v, want=%v", fieldE.Name, fieldE.EnumType, enumT)
	}
}

func TestCrossReferenceMethod(t *testing.T) {
	request := &Message{
		Name: "Request",
		ID:   ".test.Request",
	}
	response := &Message{
		Name: "Response",
		ID:   ".test.Response",
	}
	method := &Method{
		Name:         "GetResource",
		ID:           ".test.Service.GetResource",
		InputTypeID:  ".test.Request",
		OutputTypeID: ".test.Response",
	}
	mixinMethod := &Method{
		Name:            "GetOperation",
		ID:              ".test.Service.GetOperation",
		SourceServiceID: ".google.longrunning.Operations",
		InputTypeID:     ".test.Request",
		OutputTypeID:    ".test.Response",
	}
	service := &Service{
		Name:    "Service",
		ID:      ".test.Service",
		Methods: []*Method{method, mixinMethod},
	}
	mixinService := &Service{
		Name:    "Operations",
		ID:      ".google.longrunning.Operations",
		Methods: []*Method{},
	}

	model := NewTestAPI([]*Message{request, response}, []*Enum{}, []*Service{service, mixinService})
	if err := CrossReference(model); err != nil {
		t.Fatal(err)
	}
	if method.InputType != request {
		t.Errorf("mismatched input type, got=%v, want=%v", method.InputType, request)
	}
	if method.OutputType != response {
		t.Errorf("mismatched output type, got=%v, want=%v", method.OutputType, response)
	}
}

func TestCrossReferenceService(t *testing.T) {
	service := &Service{
		Name: "Service",
		ID:   ".test.Service",
	}
	mixin := &Service{
		Name: "Mixin",
		ID:   ".external.Mixin",
	}

	model := NewTestAPI([]*Message{}, []*Enum{}, []*Service{service})
	model.State.ServiceByID[mixin.ID] = mixin
	if err := CrossReference(model); err != nil {
		t.Fatal(err)
	}
	if service.Model != model {
		t.Errorf("mismatched model, got=%v, want=%v", service.Model, model)
	}
	if mixin.Model != model {
		t.Errorf("mismatched model, got=%v, want=%v", mixin.Model, model)
	}
}

func TestEnrichSamplesEnumValues(t *testing.T) {
	v_good1 := &EnumValue{Name: "GOOD_1", Number: 1}
	v_good2 := &EnumValue{Name: "GOOD_2", Number: 2}
	v_good3 := &EnumValue{Name: "GOOD_3", Number: 3}
	v_good4 := &EnumValue{Name: "GOOD_4", Number: 4}
	v_bad_deprecated := &EnumValue{Name: "BAD_DEPRECATED", Number: 5, Deprecated: true}
	v_bad_default := &EnumValue{Name: "BAD_DEFAULT", Number: 0}

	testCases := []struct {
		name         string
		values       []*EnumValue
		wantExamples []*SampleValue
	}{
		{
			name:   "more than 3 good values",
			values: []*EnumValue{v_good1, v_good2, v_good3, v_good4},
			wantExamples: []*SampleValue{
				{EnumValue: v_good1, Index: 0},
				{EnumValue: v_good2, Index: 1},
				{EnumValue: v_good3, Index: 2},
			},
		},
		{
			name:   "less than 3 good values",
			values: []*EnumValue{v_good1, v_good2, v_bad_deprecated},
			wantExamples: []*SampleValue{
				{EnumValue: v_good1, Index: 0},
				{EnumValue: v_good2, Index: 1},
			},
		},
		{
			name:   "no good values",
			values: []*EnumValue{v_bad_default, v_bad_deprecated},
			wantExamples: []*SampleValue{
				{EnumValue: v_bad_default, Index: 0},
				{EnumValue: v_bad_deprecated, Index: 1},
			},
		},
		{
			name:         "no values",
			values:       []*EnumValue{},
			wantExamples: []*SampleValue{},
		},
		{
			name:   "mixed good and bad values",
			values: []*EnumValue{v_bad_default, v_good1, v_bad_deprecated, v_good2},
			wantExamples: []*SampleValue{
				{EnumValue: v_good1, Index: 0},
				{EnumValue: v_good2, Index: 1},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enum := &Enum{
				Name:    "TestEnum",
				ID:      ".test.v1.TestEnum",
				Package: "test.v1",
				Values:  tc.values,
			}
			model := NewTestAPI([]*Message{}, []*Enum{enum}, []*Service{})
			if err := CrossReference(model); err != nil {
				t.Fatalf("CrossReference() failed: %v", err)
			}

			got := enum.ValuesForExamples
			if diff := cmp.Diff(tc.wantExamples, got, cmpopts.IgnoreFields(EnumValue{}, "Parent")); diff != "" {
				t.Errorf("mismatch in ValuesForExamples (-want, +got)\n:%s", diff)
			}
		})
	}
}

func TestEnrichSamplesOneOfExampleField(t *testing.T) {
	deprecated := &Field{
		Name:       "deprecated_field",
		ID:         ".test.Message.deprecated_field",
		Typez:      STRING_TYPE,
		IsOneOf:    true,
		Deprecated: true,
	}
	mapMessage := &Message{
		Name:  "$map<string, string>",
		ID:    "$map<string, string>",
		IsMap: true,
		Fields: []*Field{
			{Name: "key", ID: "$map<string, string>.key", Typez: STRING_TYPE},
			{Name: "value", ID: "$map<string, string>.value", Typez: STRING_TYPE},
		},
	}
	mapField := &Field{
		Name:    "map_field",
		ID:      ".test.Message.map_field",
		Typez:   MESSAGE_TYPE,
		TypezID: "$map<string, string>",
		IsOneOf: true,
		Map:     true,
	}
	repeated := &Field{
		Name:     "repeated_field",
		ID:       ".test.Message.repeated_field",
		Typez:    STRING_TYPE,
		Repeated: true,
		IsOneOf:  true,
	}
	scalar := &Field{
		Name:    "scalar_field",
		ID:      ".test.Message.scalar_field",
		Typez:   INT32_TYPE,
		IsOneOf: true,
	}
	messageField := &Field{
		Name:    "message_field",
		ID:      ".test.Message.message_field",
		Typez:   MESSAGE_TYPE,
		TypezID: ".test.OneMessage",
		IsOneOf: true,
	}
	anotherMessageField := &Field{
		Name:    "another_message_field",
		ID:      ".test.Message.another_message_field",
		Typez:   MESSAGE_TYPE,
		TypezID: ".test.AnotherMessage",
		IsOneOf: true,
	}

	testCases := []struct {
		name   string
		fields []*Field
		want   *Field
	}{
		{
			name:   "all types",
			fields: []*Field{deprecated, mapField, repeated, scalar, messageField},
			want:   scalar,
		},
		{
			name:   "no primitives",
			fields: []*Field{deprecated, mapField, repeated, messageField},
			want:   messageField,
		},
		{
			name:   "only scalars and messages",
			fields: []*Field{messageField, scalar, anotherMessageField},
			want:   scalar,
		},
		{
			name:   "no scalars",
			fields: []*Field{deprecated, mapField, repeated},
			want:   repeated,
		},
		{
			name:   "only map and deprecated",
			fields: []*Field{deprecated, mapField},
			want:   mapField,
		},
		{
			name:   "only deprecated",
			fields: []*Field{deprecated},
			want:   deprecated,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			group := &OneOf{
				Name:   "test_oneof",
				ID:     ".test.Message.test_oneof",
				Fields: tc.fields,
			}
			message := &Message{
				Name:    "Message",
				ID:      ".test.Message",
				Package: "test",
				Fields:  tc.fields,
				OneOfs:  []*OneOf{group},
			}
			oneMesage := &Message{
				Name:    "OneMessage",
				ID:      ".test.OneMessage",
				Package: "test",
			}
			anotherMessage := &Message{
				Name:    "AnotherMessage",
				ID:      ".test.AnotherMessage",
				Package: "test",
			}
			model := NewTestAPI([]*Message{message, oneMesage, anotherMessage, mapMessage}, []*Enum{}, []*Service{})
			if err := CrossReference(model); err != nil {
				t.Fatal(err)
			}

			got := group.ExampleField
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch in ExampleField (-want, +got)\n:%s", diff)
			}
		})
	}
}
