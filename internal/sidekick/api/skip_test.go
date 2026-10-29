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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestSkipMessages(t *testing.T) {
	m0 := NewTestMessage("Message0")
	m1 := NewTestMessage("Message1")
	m2 := NewTestMessage("Message2")
	model := NewTestAPI([]*Message{m0, m1, m2}, []*Enum{}, []*Service{})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		SkippedIDs: []string{".test.Message1"},
	})
	want := []*Message{m0, m2}

	if diff := cmp.Diff(want, model.Messages); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestSkipEnums(t *testing.T) {
	e0 := NewTestEnum("Enum0")
	e1 := NewTestEnum("Enum1")
	e2 := NewTestEnum("Enum2")
	model := NewTestAPI([]*Message{}, []*Enum{e0, e1, e2}, []*Service{})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		SkippedIDs: []string{".test.Enum1"},
	})

	want := []*Enum{e0, e2}

	if diff := cmp.Diff(want, model.Enums, allowUnexported()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestSkipNestedMessages(t *testing.T) {
	m0 := NewTestMessage("Message0")
	m1 := NewTestMessage("Message1")
	m2 := NewTestMessage("Message2").WithMessages(m0, m1)
	model := NewTestAPI([]*Message{m2}, []*Enum{}, []*Service{})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		SkippedIDs: []string{".test.Message2.Message1"},
	})
	want := []*Message{m0}
	if diff := cmp.Diff(want, m2.Messages, cmpopts.IgnoreFields(Message{}, "Messages")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestSkipNestedEnums(t *testing.T) {
	e0 := NewTestEnum("Enum0")
	e1 := NewTestEnum("Enum1")
	e2 := NewTestEnum("Enum2")
	m := NewTestMessage("Message").WithEnums(e0, e1, e2)
	model := NewTestAPI([]*Message{m}, []*Enum{}, []*Service{})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		SkippedIDs: []string{".test.Message.Enum1"},
	})

	want := []*Enum{e0, e2}
	if diff := cmp.Diff(want, m.Enums, allowUnexported(), cmpopts.IgnoreFields(Message{}, "Enums")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestSkipServices(t *testing.T) {
	s0 := NewTestService("Service0")
	s1 := NewTestService("Service1")
	s2 := NewTestService("Service2")
	model := NewTestAPI([]*Message{}, []*Enum{}, []*Service{s0, s1, s2})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		SkippedIDs: []string{".test.Service1"},
	})

	want := []*Service{s0, s2}

	if diff := cmp.Diff(want, model.Services, cmpopts.IgnoreFields(Service{}, "Model")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

var methodIgnoreFields = []string{
	"Model",
	"Service",
	"SourceService",
	"InputType",
	"OutputType",
	"InputTypeID",
	"OutputTypeID",
	"IsSimple",
	"IsLRO",
	"LongRunningResponseType",
	"LongRunningReturnsEmpty",
	"IsList",
	"IsStreaming",
	"IsAIPStandard",
	"IsAIPStandardGet",
	"IsAIPStandardDelete",
	"IsAIPStandardUndelete",
	"IsAIPStandardCreate",
	"IsAIPStandardUpdate",
	"IsAIPStandardList",
	"SampleInfo",
}

func TestSkipMethods(t *testing.T) {
	s0 := NewTestService("Service0")
	m0 := NewTestMethod("Method0")
	m1 := NewTestMethod("Method1")
	m2 := NewTestMethod("Method2")
	s1 := NewTestService("Service1").WithMethods(m0, m1, m2)
	s2 := NewTestService("Service2")
	model := NewTestAPI([]*Message{}, []*Enum{}, []*Service{s0, s1, s2})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		SkippedIDs: []string{".test.Service1.Method1"},
	})

	wantServices := []*Service{s0, s1, s2}
	if diff := cmp.Diff(wantServices, model.Services, allowUnexported(), cmpopts.IgnoreFields(Service{}, "Model")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantMethods := []*Method{m0, m2}
	if diff := cmp.Diff(wantMethods, s1.Methods, allowUnexported(), cmpopts.IgnoreFields(Method{}, methodIgnoreFields...)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestIncludeUnknownIdError(t *testing.T) {
	model := NewTestAPI([]*Message{}, []*Enum{}, []*Service{})
	err := SkipModelElements(model, ModelOverride{
		IncludedIDs: []string{".test.UnknownId"},
	})
	if err == nil {
		t.Fatal("SkipModelElements should error on unknown IDs")
	}

	msg := err.Error()
	if !strings.Contains(msg, ".test.UnknownId") {
		t.Errorf("SkipModelElements should report unknown IDs in its error message. message=`%s`", msg)
	}
}

func TestIncludeNestedEnums(t *testing.T) {
	e0 := NewTestEnum("Enum0")
	e1 := NewTestEnum("Enum1")
	e2 := NewTestEnum("Enum2")
	m := NewTestMessage("Message").WithEnums(e0, e1, e2)
	model := NewTestAPI([]*Message{m}, []*Enum{e0, e1, e2}, []*Service{})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		IncludedIDs: []string{".test.Message.Enum0"},
	})

	want := []*Enum{e0}
	if diff := cmp.Diff(want, m.Enums, allowUnexported(), cmpopts.IgnoreFields(Message{}, "Enums")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestIncludeNestedMessages(t *testing.T) {
	m0 := NewTestMessage("Message0")
	m1 := NewTestMessage("Message1")
	m2 := NewTestMessage("Message2").WithMessages(m0, m1)
	model := NewTestAPI([]*Message{m0, m1, m2}, []*Enum{}, []*Service{})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		IncludedIDs: []string{".test.Message2.Message0"},
	})
	want := []*Message{m0}
	if diff := cmp.Diff(want, m2.Messages, cmpopts.IgnoreFields(Message{}, "Messages")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestIncludeMethods(t *testing.T) {
	m := NewTestMessage("Empty")
	s0 := NewTestService("Service0")
	m0 := NewTestMethod("Method0").WithInput(m).WithOutput(m)
	m1 := NewTestMethod("Method1").WithInput(m).WithOutput(m)
	m2 := NewTestMethod("Method2").WithInput(m).WithOutput(m)
	s1 := NewTestService("Service1").WithMethods(m0, m1, m2)
	s2 := NewTestService("Service2")
	model := NewTestAPI([]*Message{m}, []*Enum{}, []*Service{s0, s1, s2})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		IncludedIDs: []string{".test.Service1.Method1", ".test.Service1.Method2"},
	})

	wantServices := []*Service{s1}
	if diff := cmp.Diff(wantServices, model.Services, allowUnexported(), cmpopts.IgnoreFields(Method{}, methodIgnoreFields...), cmpopts.IgnoreFields(Service{}, "Model", "QuickstartMethod")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantMethods := []*Method{m1, m2}
	if diff := cmp.Diff(wantMethods, s1.Methods, allowUnexported(), cmpopts.IgnoreFields(Method{}, methodIgnoreFields...)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestSkipMessageFields(t *testing.T) {
	f0 := NewTestField("field0").WithType(TypezString)
	f1 := NewTestField("field1").WithType(TypezString)
	f2 := NewTestField("field2").WithType(TypezInt32)
	m0 := NewTestMessage("Message0").WithFields(f0, f1, f2)
	model := NewTestAPI([]*Message{m0}, []*Enum{}, []*Service{})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		SkippedIDs: []string{".test.Message0.field1"},
	})
	want := []*Field{f0, f2}
	if diff := cmp.Diff(want, m0.Fields, cmpopts.IgnoreFields(Field{}, "Parent")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestSkipNestedMessageFields(t *testing.T) {
	f0 := NewTestField("field0").WithType(TypezString)
	f1 := NewTestField("field1").WithType(TypezString)
	inner := NewTestMessage("Inner").WithFields(f0, f1)
	outer := NewTestMessage("Outer").WithMessages(inner)
	model := NewTestAPI([]*Message{outer, inner}, []*Enum{}, []*Service{})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		SkippedIDs: []string{".test.Outer.Inner.field1"},
	})
	want := []*Field{f0}
	if diff := cmp.Diff(want, inner.Fields, cmpopts.IgnoreFields(Field{}, "Parent")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestSkipOneOfField(t *testing.T) {
	f0 := NewTestField("field0").WithType(TypezString)
	f1 := NewTestField("field1").WithType(TypezString)
	f2 := NewTestField("field2").WithType(TypezInt32)
	oneof := NewTestOneOf("choice").WithFields(f1, f2)
	m := NewTestMessage("Message").WithFields(f0).WithOneOfs(oneof)
	model := NewTestAPI([]*Message{m}, []*Enum{}, []*Service{})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		SkippedIDs: []string{".test.Message.field1"},
	})
	wantFields := []*Field{f0, f2}
	if diff := cmp.Diff(wantFields, m.Fields, cmpopts.IgnoreFields(Field{}, "Parent", "Group")); diff != "" {
		t.Errorf("mismatch in fields (-want +got):\n%s", diff)
	}
	wantOneOfFields := []*Field{f2}
	if diff := cmp.Diff(wantOneOfFields, oneof.Fields, cmpopts.IgnoreFields(Field{}, "Parent", "Group")); diff != "" {
		t.Errorf("mismatch in oneof fields (-want +got):\n%s", diff)
	}
	if oneof.ExampleField != f2 {
		t.Errorf("expected ExampleField to be %v, got %v", f2, oneof.ExampleField)
	}
}

func TestSkipAllOneOfFields(t *testing.T) {
	f1 := NewTestField("field1").WithType(TypezString)
	f2 := NewTestField("field2").WithType(TypezInt32)
	oneof := NewTestOneOf("choice").WithFields(f1, f2)
	m := NewTestMessage("Message").WithOneOfs(oneof)
	model := NewTestAPI([]*Message{m}, []*Enum{}, []*Service{})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		SkippedIDs: []string{".test.Message.field1", ".test.Message.field2"},
	})
	if len(m.Fields) != 0 {
		t.Errorf("expected no fields, got %v", m.Fields)
	}
	if len(m.OneOfs) != 0 {
		t.Errorf("expected empty oneof to be pruned, got %v", m.OneOfs)
	}
}

func TestSkipOneOfDirectly(t *testing.T) {
	f0 := NewTestField("field0").WithType(TypezString)
	f1 := NewTestField("field1").WithType(TypezString)
	f2 := NewTestField("field2").WithType(TypezInt32)
	oneof := NewTestOneOf("choice").WithFields(f1, f2)
	m := NewTestMessage("Message").WithFields(f0).WithOneOfs(oneof)
	model := NewTestAPI([]*Message{m}, []*Enum{}, []*Service{})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		SkippedIDs: []string{".test.Message.choice"},
	})
	wantFields := []*Field{f0}
	if diff := cmp.Diff(wantFields, m.Fields, cmpopts.IgnoreFields(Field{}, "Parent", "Group")); diff != "" {
		t.Errorf("mismatch in fields (-want +got):\n%s", diff)
	}
	if len(m.OneOfs) != 0 {
		t.Errorf("expected oneof to be removed, got %v", m.OneOfs)
	}
}

func TestSkipFieldAffectsMethodSignature(t *testing.T) {
	fName := NewTestField("name").WithType(TypezString)
	fTarget := NewTestField("target").WithType(TypezString)
	fSkip := NewTestField("skip_me").WithType(TypezString)
	req := NewTestMessage("Req").WithFields(fName, fTarget, fSkip)
	resp := NewTestMessage("Resp")
	sigKeep := NewTestMethodSignature("name", "target").WithFields(fName, fTarget)
	sigDrop := NewTestMethodSignature("name", "skip_me").WithFields(fName, fSkip)
	method := NewTestMethod("DoWork").WithInput(req).WithOutput(resp).WithSignatures(sigKeep, sigDrop)
	service := NewTestService("Service").WithMethods(method)
	model := NewTestAPI([]*Message{req, resp}, []*Enum{}, []*Service{service})
	CrossReference(model)
	err := SkipModelElements(model, ModelOverride{SkippedIDs: []string{".test.Req.skip_me"}})
	if err == nil {
		t.Errorf("removing a signature should result in an error, got %+v", method)
	}
}

func TestSkipFieldAffectsPagination(t *testing.T) {
	pageSizeField := NewTestField("page_size").WithJSONName("pageSize").WithType(TypezInt32)
	pageTokenField := NewTestField("page_token").WithJSONName("pageToken").WithType(TypezString)
	nextPageTokenField := NewTestField("next_page_token").WithJSONName("nextPageToken").WithType(TypezString)
	itemsField := NewTestField("items").WithJSONName("items").WithType(TypezMessage).WithTypezID(".test.Item").WithRepeated()

	req := NewTestMessage("ListReq").WithFields(pageSizeField, pageTokenField)
	resp := NewTestMessage("ListResp").WithFields(nextPageTokenField, itemsField)
	item := NewTestMessage("Item")
	method := NewTestMethod("ListItems").WithInput(req).WithOutput(resp)
	service := NewTestService("Service").WithMethods(method)
	model := NewTestAPI([]*Message{req, resp, item}, []*Enum{}, []*Service{service})
	UpdateMethodPagination(nil, model)
	CrossReference(model)
	if method.Pagination == nil || resp.Pagination == nil {
		t.Fatalf("expected pagination to be set before skip")
	}

	err := SkipModelElements(model, ModelOverride{
		SkippedIDs: []string{".test.ListReq.page_token"},
	})
	if err == nil {
		t.Errorf("removing a pagination field should result in errors: got = %+v", method)
	}
}

func TestIncludeKeepsAllMessageFields(t *testing.T) {
	f0 := NewTestField("field0").WithType(TypezString)
	f1 := NewTestField("field1").WithType(TypezString)
	m0 := NewTestMessage("Message0").WithFields(f0, f1)
	model := NewTestAPI([]*Message{m0}, []*Enum{}, []*Service{})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		IncludedIDs: []string{".test.Message0"},
	})
	want := []*Field{f0, f1}
	if diff := cmp.Diff(want, m0.Fields, cmpopts.IgnoreFields(Field{}, "Parent")); diff != "" {
		t.Errorf("mismatch in fields (-want +got):\n%s", diff)
	}
}

func TestSkipPreservesExternalAndDependencyMessagesInSymbolTable(t *testing.T) {
	timestamp := NewTestMessage("Timestamp").WithPackage("google.protobuf")
	f0 := NewTestField("field0").WithType(TypezString)
	f1 := NewTestField("field1").WithType(TypezString)
	m0 := NewTestMessage("Message0").WithFields(f0, f1)
	model := NewTestAPI([]*Message{m0}, []*Enum{}, []*Service{})
	model.AddMessage(timestamp)
	CrossReference(model)

	if err := SkipModelElements(model, ModelOverride{
		SkippedIDs: []string{".test.Message0.field1"},
	}); err != nil {
		t.Fatal(err)
	}

	if got := model.Message(".google.protobuf.Timestamp"); got == nil {
		t.Errorf("expected .google.protobuf.Timestamp to remain in symbol table after skipping an unrelated field")
	}
}
