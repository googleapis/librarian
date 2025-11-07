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
