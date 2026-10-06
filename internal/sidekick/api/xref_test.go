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
		fields1 = append(fields1, NewTestField(name).WithType(TypezString))
	}
	fields1 = append(fields1, NewTestField("basic_field").WithType(TypezString))
	group0 := NewTestOneOf("group0").WithFields(fields1[0], fields1[1])
	group1 := NewTestOneOf("group1").WithFields(fields1[2], fields1[3])
	message1 := NewTestMessage("Message1").
		WithFields(fields1...).
		WithOneOfs(group0, group1)

	var fields2 []*Field
	for i := range 2 {
		name := fmt.Sprintf("field%d", i+4)
		fields2 = append(fields2, NewTestField(name).WithType(TypezString))
	}
	group2 := NewTestOneOf("group2").WithFields(fields2[0], fields2[1])
	message2 := NewTestMessage("Message2").WithOneOfs(group2)

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
	messageT := NewTestMessage("MessageT")
	fieldM := NewTestField("message_field").
		WithType(TypezMessage).
		WithTypezID(".test.MessageT")
	enumT := NewTestEnum("EnumT")
	fieldE := NewTestField("enum_field").
		WithType(TypezEnum).
		WithTypezID(".test.EnumT")
	message := NewTestMessage("Message").WithFields(fieldM, fieldE)

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
	request := NewTestMessage("Request")
	response := NewTestMessage("Response")
	method := NewTestMethod("GetResource").
		WithInputTypeID(".test.Request").
		WithOutputTypeID(".test.Response")
	mixinMethod := NewTestMethod("GetOperation").
		WithSourceServiceID(".google.longrunning.Operations").
		WithInputTypeID(".test.Request").
		WithOutputTypeID(".test.Response")
	service := NewTestService("Service").WithMethods(method, mixinMethod)
	mixinService := NewTestService("Operations").WithID(".google.longrunning.Operations")

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
	service := NewTestService("Service")
	mixin := NewTestService("Mixin").WithID(".external.Mixin")

	model := NewTestAPI([]*Message{}, []*Enum{}, []*Service{service})
	model.AddService(mixin)
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
