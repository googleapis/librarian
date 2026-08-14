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
	m0 := &Message{
		Name:    "Message0",
		Package: "test",
		ID:      ".test.Message0",
	}
	m1 := &Message{
		Name:    "Message1",
		Package: "test",
		ID:      ".test.Message1",
	}
	m2 := &Message{
		Name:    "Message2",
		Package: "test",
		ID:      ".test.Message2",
	}
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
	e0 := &Enum{
		Name:    "Enum0",
		Package: "test",
		ID:      ".test.Enum0",
	}
	e1 := &Enum{
		Name:    "Enum1",
		Package: "test",
		ID:      ".test.Enum1",
	}
	e2 := &Enum{
		Name:    "Enum2",
		Package: "test",
		ID:      ".test.Enum2",
	}
	model := NewTestAPI([]*Message{}, []*Enum{e0, e1, e2}, []*Service{})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		SkippedIDs: []string{".test.Enum1"},
	})

	want := []*Enum{e0, e2}

	if diff := cmp.Diff(want, model.Enums); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestSkipNestedMessages(t *testing.T) {
	m0 := &Message{
		Name:    "Message0",
		Package: "test",
		ID:      ".test.Message2.Message0",
	}
	m1 := &Message{
		Name:    "Message1",
		Package: "test",
		ID:      ".test.Message2.Message1",
	}
	m2 := &Message{
		Name:     "Message2",
		Package:  "test",
		ID:       ".test.Message2",
		Messages: []*Message{m0, m1},
	}
	model := NewTestAPI([]*Message{m2}, []*Enum{}, []*Service{})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		SkippedIDs: []string{".test.Message2.Message1"},
	})
	want := []*Message{m0}
	if diff := cmp.Diff(want, m2.Messages); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestSkipNestedEnums(t *testing.T) {
	e0 := &Enum{
		Name:    "Enum0",
		Package: "test",
		ID:      ".test.Message.Enum0",
	}
	e1 := &Enum{
		Name:    "Enum1",
		Package: "test",
		ID:      ".test.Message.Enum1",
	}
	e2 := &Enum{
		Name:    "Enum2",
		Package: "test",
		ID:      ".test.Message.Enum2",
	}
	m := &Message{
		Name:    "Message",
		Package: "test",
		ID:      ".test.Message",
		Enums:   []*Enum{e0, e1, e2},
	}
	model := NewTestAPI([]*Message{m}, []*Enum{}, []*Service{})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		SkippedIDs: []string{".test.Message.Enum1"},
	})

	want := []*Enum{e0, e2}
	if diff := cmp.Diff(want, m.Enums); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestSkipServices(t *testing.T) {
	s0 := &Service{
		Name:    "Service0",
		Package: "test",
		ID:      ".test.Service0",
	}
	s1 := &Service{
		Name:    "Service1",
		Package: "test",
		ID:      ".test.Service1",
	}
	s2 := &Service{
		Name:    "Service2",
		Package: "test",
		ID:      ".test.Service2",
	}
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

func TestSkipMethods(t *testing.T) {
	s0 := &Service{
		Name:    "Service0",
		Package: "test",
		ID:      ".test.Service0",
	}
	s1 := &Service{
		Name:    "Service1",
		Package: "test",
		ID:      ".test.Service1",
		Methods: []*Method{
			{
				Name: "Method0",
				ID:   ".test.Service1.Method0",
			},
			{
				Name: "Method1",
				ID:   ".test.Service1.Method1",
			},
			{
				Name: "Method2",
				ID:   ".test.Service1.Method2",
			},
		},
	}
	s2 := &Service{
		Name:    "Service2",
		Package: "test",
		ID:      ".test.Service2",
	}
	model := NewTestAPI([]*Message{}, []*Enum{}, []*Service{s0, s1, s2})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		SkippedIDs: []string{".test.Service1.Method1"},
	})

	wantServices := []*Service{s0, s1, s2}
	if diff := cmp.Diff(wantServices, model.Services, cmpopts.IgnoreFields(Service{}, "Model")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantMethods := []*Method{
		{
			Name: "Method0",
			ID:   ".test.Service1.Method0",
		},
		{
			Name: "Method2",
			ID:   ".test.Service1.Method2",
		},
	}
	if diff := cmp.Diff(wantMethods, s1.Methods); diff != "" {
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
	e0 := &Enum{
		Name:    "Enum0",
		Package: "test",
		ID:      ".test.Message.Enum0",
	}
	e1 := &Enum{
		Name:    "Enum1",
		Package: "test",
		ID:      ".test.Message.Enum1",
	}
	e2 := &Enum{
		Name:    "Enum2",
		Package: "test",
		ID:      ".test.Message.Enum2",
	}
	m := &Message{
		Name:    "Message",
		Package: "test",
		ID:      ".test.Message",
	}
	model := NewTestAPI([]*Message{m}, []*Enum{e0, e1, e2}, []*Service{})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		IncludedIDs: []string{".test.Message.Enum0"},
	})

	want := []*Enum{e0}
	if diff := cmp.Diff(want, m.Enums, cmpopts.IgnoreFields(Message{}, "Enums")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestIncludeNestedMessages(t *testing.T) {
	m0 := &Message{
		Name:    "Message0",
		Package: "test",
		ID:      ".test.Message2.Message0",
	}
	m1 := &Message{
		Name:    "Message1",
		Package: "test",
		ID:      ".test.Message2.Message1",
	}
	m2 := &Message{
		Name:    "Message2",
		Package: "test",
		ID:      ".test.Message2",
	}
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

func TestIncludeMethods(t *testing.T) {
	m := &Message{
		Name: "Empty",
		ID:   ".test.Empty",
	}
	s0 := &Service{
		Name:    "Service0",
		Package: "test",
		ID:      ".test.Service0",
	}
	s1 := &Service{
		Name:    "Service1",
		Package: "test",
		ID:      ".test.Service1",
		Methods: []*Method{
			{
				Name:         "Method0",
				ID:           ".test.Service1.Method0",
				InputTypeID:  ".test.Empty",
				OutputTypeID: ".test.Empty",
			},
			{
				Name:         "Method1",
				ID:           ".test.Service1.Method1",
				InputTypeID:  ".test.Empty",
				OutputTypeID: ".test.Empty",
			},
			{
				Name:         "Method2",
				ID:           ".test.Service1.Method2",
				InputTypeID:  ".test.Empty",
				OutputTypeID: ".test.Empty",
			},
		},
	}
	s2 := &Service{
		Name:    "Service2",
		Package: "test",
		ID:      ".test.Service2",
	}
	model := NewTestAPI([]*Message{m}, []*Enum{}, []*Service{s0, s1, s2})
	CrossReference(model)
	SkipModelElements(model, ModelOverride{
		IncludedIDs: []string{".test.Service1.Method1", ".test.Service1.Method2"},
	})

	wantServices := []*Service{s1}
	if diff := cmp.Diff(wantServices, model.Services, cmpopts.IgnoreFields(Method{}, methodIgnoreFields...), cmpopts.IgnoreFields(Service{}, "Model", "QuickstartMethod")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantMethods := []*Method{
		{
			Name: "Method1",
			ID:   ".test.Service1.Method1",
		},
		{
			Name: "Method2",
			ID:   ".test.Service1.Method2",
		},
	}
	if diff := cmp.Diff(wantMethods, s1.Methods, cmpopts.IgnoreFields(Method{}, methodIgnoreFields...)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestSkipMessageFields(t *testing.T) {
	f0 := &Field{Name: "field0", ID: ".test.Message0.field0", Typez: TypezString}
	f1 := &Field{Name: "field1", ID: ".test.Message0.field1", Typez: TypezString}
	f2 := &Field{Name: "field2", ID: ".test.Message0.field2", Typez: TypezInt32}
	m0 := &Message{
		Name:    "Message0",
		Package: "test",
		ID:      ".test.Message0",
		Fields:  []*Field{f0, f1, f2},
	}
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
	f0 := &Field{Name: "field0", ID: ".test.Outer.Inner.field0", Typez: TypezString}
	f1 := &Field{Name: "field1", ID: ".test.Outer.Inner.field1", Typez: TypezString}
	inner := &Message{
		Name:    "Inner",
		Package: "test",
		ID:      ".test.Outer.Inner",
		Fields:  []*Field{f0, f1},
	}
	outer := &Message{
		Name:     "Outer",
		Package:  "test",
		ID:       ".test.Outer",
		Messages: []*Message{inner},
	}
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
	f0 := &Field{Name: "field0", ID: ".test.Message.field0", Typez: TypezString}
	f1 := &Field{Name: "field1", ID: ".test.Message.field1", Typez: TypezString, IsOneOf: true}
	f2 := &Field{Name: "field2", ID: ".test.Message.field2", Typez: TypezInt32, IsOneOf: true}
	oneof := &OneOf{
		Name:   "choice",
		ID:     ".test.Message.choice",
		Fields: []*Field{f1, f2},
	}
	m := &Message{
		Name:    "Message",
		Package: "test",
		ID:      ".test.Message",
		Fields:  []*Field{f0, f1, f2},
		OneOfs:  []*OneOf{oneof},
	}
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
	f1 := &Field{Name: "field1", ID: ".test.Message.field1", Typez: TypezString, IsOneOf: true}
	f2 := &Field{Name: "field2", ID: ".test.Message.field2", Typez: TypezInt32, IsOneOf: true}
	oneof := &OneOf{
		Name:   "choice",
		ID:     ".test.Message.choice",
		Fields: []*Field{f1, f2},
	}
	m := &Message{
		Name:    "Message",
		Package: "test",
		ID:      ".test.Message",
		Fields:  []*Field{f1, f2},
		OneOfs:  []*OneOf{oneof},
	}
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
	f0 := &Field{Name: "field0", ID: ".test.Message.field0", Typez: TypezString}
	f1 := &Field{Name: "field1", ID: ".test.Message.field1", Typez: TypezString, IsOneOf: true}
	f2 := &Field{Name: "field2", ID: ".test.Message.field2", Typez: TypezInt32, IsOneOf: true}
	oneof := &OneOf{
		Name:   "choice",
		ID:     ".test.Message.choice",
		Fields: []*Field{f1, f2},
	}
	m := &Message{
		Name:    "Message",
		Package: "test",
		ID:      ".test.Message",
		Fields:  []*Field{f0, f1, f2},
		OneOfs:  []*OneOf{oneof},
	}
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
	fName := &Field{Name: "name", ID: ".test.Req.name", Typez: TypezString}
	fTarget := &Field{Name: "target", ID: ".test.Req.target", Typez: TypezString}
	fSkip := &Field{Name: "skip_me", ID: ".test.Req.skip_me", Typez: TypezString}
	req := &Message{
		Name:    "Req",
		Package: "test",
		ID:      ".test.Req",
		Fields:  []*Field{fName, fTarget, fSkip},
	}
	resp := &Message{
		Name:    "Resp",
		Package: "test",
		ID:      ".test.Resp",
	}
	sigKeep := &MethodSignature{
		Names:  []string{"name", "target"},
		Fields: []*Field{fName, fTarget},
	}
	sigDrop := &MethodSignature{
		Names:  []string{"name", "skip_me"},
		Fields: []*Field{fName, fSkip},
	}
	method := &Method{
		Name:         "DoWork",
		ID:           ".test.Service.DoWork",
		InputTypeID:  req.ID,
		OutputTypeID: resp.ID,
		Signatures:   []*MethodSignature{sigKeep, sigDrop},
	}
	service := &Service{
		Name:    "Service",
		Package: "test",
		ID:      ".test.Service",
		Methods: []*Method{method},
	}
	model := NewTestAPI([]*Message{req, resp}, []*Enum{}, []*Service{service})
	CrossReference(model)
	err := SkipModelElements(model, ModelOverride{SkippedIDs: []string{".test.Req.skip_me"}})
	if err == nil {
		t.Errorf("removing a signature should result in an error, got %+v", method)
	}
}

func TestSkipFieldAffectsPagination(t *testing.T) {
	pageSizeField := &Field{Name: "page_size", JSONName: "pageSize", ID: ".test.ListReq.page_size", Typez: TypezInt32}
	pageTokenField := &Field{Name: "page_token", JSONName: "pageToken", ID: ".test.ListReq.page_token", Typez: TypezString}
	nextPageTokenField := &Field{Name: "next_page_token", JSONName: "nextPageToken", ID: ".test.ListResp.next_page_token", Typez: TypezString}
	itemsField := &Field{Name: "items", JSONName: "items", ID: ".test.ListResp.items", Typez: TypezMessage, TypezID: ".test.Item", Repeated: true}

	req := &Message{
		Name:    "ListReq",
		Package: "test",
		ID:      ".test.ListReq",
		Fields:  []*Field{pageSizeField, pageTokenField},
	}
	resp := &Message{
		Name:    "ListResp",
		Package: "test",
		ID:      ".test.ListResp",
		Fields:  []*Field{nextPageTokenField, itemsField},
	}
	item := &Message{
		Name:    "Item",
		Package: "test",
		ID:      ".test.Item",
	}
	method := &Method{
		Name:         "ListItems",
		ID:           ".test.Service.ListItems",
		InputTypeID:  req.ID,
		OutputTypeID: resp.ID,
	}
	service := &Service{
		Name:    "Service",
		Package: "test",
		ID:      ".test.Service",
		Methods: []*Method{method},
	}
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
	f0 := &Field{Name: "field0", ID: ".test.Message0.field0", Typez: TypezString}
	f1 := &Field{Name: "field1", ID: ".test.Message0.field1", Typez: TypezString}
	m0 := &Message{
		Name:    "Message0",
		Package: "test",
		ID:      ".test.Message0",
		Fields:  []*Field{f0, f1},
	}
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
	timestamp := &Message{
		Name:    "Timestamp",
		Package: "google.protobuf",
		ID:      ".google.protobuf.Timestamp",
	}
	m0 := &Message{
		Name:    "Message0",
		Package: "test",
		ID:      ".test.Message0",
		Fields: []*Field{
			{Name: "field0", ID: ".test.Message0.field0", Typez: TypezString},
			{Name: "field1", ID: ".test.Message0.field1", Typez: TypezString},
		},
	}
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
