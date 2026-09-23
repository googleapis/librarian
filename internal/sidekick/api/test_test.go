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

package api_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestMethod_WithBindings_NilPathInfo(t *testing.T) {
	method := &api.Method{Name: "Test"}
	b1 := api.NewTestPathBinding("GET", (&api.PathTemplate{}).WithLiteral("v1"))
	b2 := api.NewTestPathBinding("POST", (&api.PathTemplate{}).WithLiteral("v2"))

	method.WithBindings(b1, b2)

	if method.PathInfo == nil {
		t.Fatal("expected PathInfo to be initialized, got nil")
	}
	if diff := cmp.Diff([]*api.PathBinding{b1, b2}, method.PathInfo.Bindings, cmpopts.IgnoreUnexported(api.PathTemplate{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestMethod_WithBindings_NonNilPathInfo(t *testing.T) {
	method := api.NewTestMethod("Test")
	b1 := api.NewTestPathBinding("DELETE", (&api.PathTemplate{}).WithLiteral("v1"))

	method.WithBindings(b1)

	if diff := cmp.Diff([]*api.PathBinding{b1}, method.PathInfo.Bindings, cmpopts.IgnoreUnexported(api.PathTemplate{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestMethod_WithBodyFieldPath_NilPathInfo(t *testing.T) {
	method := &api.Method{Name: "Test"}
	method.WithBodyFieldPath("data")

	if method.PathInfo == nil {
		t.Fatal("expected PathInfo to be initialized, got nil")
	}
	if method.PathInfo.BodyFieldPath != "data" {
		t.Errorf("expected BodyFieldPath to be 'data', got %q", method.PathInfo.BodyFieldPath)
	}
}

func TestMethod_WithVerb_NilPathInfo(t *testing.T) {
	method := &api.Method{Name: "Test"}
	method.WithVerb("PUT")

	if method.PathInfo == nil || len(method.PathInfo.Bindings) == 0 {
		t.Fatal("expected PathInfo and at least one binding to be initialized")
	}
	if method.PathInfo.Bindings[0].Verb != "PUT" {
		t.Errorf("expected Verb to be 'PUT', got %q", method.PathInfo.Bindings[0].Verb)
	}
}

func TestMethod_WithPathTemplate_NilPathInfo(t *testing.T) {
	method := &api.Method{Name: "Test"}
	pt := (&api.PathTemplate{}).WithLiteral("v1")
	method.WithPathTemplate(pt)

	if method.PathInfo == nil || len(method.PathInfo.Bindings) == 0 {
		t.Fatal("expected PathInfo and at least one binding to be initialized")
	}
	if method.PathInfo.Bindings[0].PathTemplate != pt {
		t.Errorf("expected PathTemplate to match")
	}
}

func TestMethod_WithQueryParameters_NilPathInfo(t *testing.T) {
	method := &api.Method{Name: "Test"}
	params := map[string]bool{"page_size": true}
	method.WithQueryParameters(params)

	if method.PathInfo == nil || len(method.PathInfo.Bindings) == 0 {
		t.Fatal("expected PathInfo and at least one binding to be initialized")
	}
	if diff := cmp.Diff(params, method.PathInfo.Bindings[0].QueryParameters); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestMethod_WithDocumentation(t *testing.T) {
	method := api.NewTestMethod("Test").WithDocumentation("A test method.")
	if method.Documentation != "A test method." {
		t.Errorf("expected Documentation 'A test method.', got %q", method.Documentation)
	}
}

func TestPathBinding_Fluent(t *testing.T) {
	pt := (&api.PathTemplate{}).WithLiteral("v1")
	b := api.NewTestPathBinding("GET", pt).
		WithVerb("POST").
		WithQueryParameters(map[string]bool{"filter": true})

	if b.Verb != "POST" {
		t.Errorf("expected Verb 'POST', got %q", b.Verb)
	}
	if b.PathTemplate != pt {
		t.Errorf("expected PathTemplate to match")
	}
	if !b.QueryParameters["filter"] {
		t.Errorf("expected filter query parameter to be true")
	}
}

func TestField_WithJSONName(t *testing.T) {
	f := api.NewTestField("foo_bar").WithJSONName("customName")
	if f.JSONName != "customName" {
		t.Errorf("expected JSONName 'customName', got %q", f.JSONName)
	}
}

func TestMessage_WithDocumentation(t *testing.T) {
	m := api.NewTestMessage("Test").WithDocumentation("A test message.")
	if m.Documentation != "A test message." {
		t.Errorf("expected Documentation 'A test message.', got %q", m.Documentation)
	}
}

func TestMessage_WithIsMap(t *testing.T) {
	m := api.NewTestMessage("TestEntry").WithIsMap()
	if !m.IsMap {
		t.Errorf("expected IsMap to be true")
	}
}

func TestOneOf_WithDocumentation(t *testing.T) {
	o := api.NewTestOneOf("choice").WithDocumentation("A test oneof.")
	if o.Documentation != "A test oneof." {
		t.Errorf("expected Documentation 'A test oneof.', got %q", o.Documentation)
	}
}

func TestField_WithRecursive(t *testing.T) {
	f := api.NewTestField("child").WithRecursive()
	if !f.Recursive {
		t.Errorf("expected Recursive to be true")
	}
}

func TestNewTestAPI_IndexesNestedMessages(t *testing.T) {
	child := api.NewTestMessage("Child")
	parent := api.NewTestMessage("Parent").WithMessages(child)
	model := api.NewTestAPI([]*api.Message{parent}, nil, nil)

	if got := model.Message(child.ID); got != child {
		t.Errorf("model.Message(%q) = %v, want %v", child.ID, got, child)
	}
	if child.Parent != parent {
		t.Errorf("child.Parent = %v, want %v", child.Parent, parent)
	}
}

func TestNewTestAPI_IndexesNestedEnums(t *testing.T) {
	val := api.NewTestEnumValue("VAL", 0)
	nestedEnum := api.NewTestEnum("NestedEnum").WithValues(val)
	parent := api.NewTestMessage("Parent")
	// Bypassing WithEnums() deliberately leaves Parent and Value.Parent unwired,
	// verifying that NewTestAPI properly initializes them during indexing.
	parent.Enums = []*api.Enum{nestedEnum}

	model := api.NewTestAPI([]*api.Message{parent}, nil, nil)

	if got := model.Enum(nestedEnum.ID); got != nestedEnum {
		t.Errorf("model.Enum(%q) = %v, want %v", nestedEnum.ID, got, nestedEnum)
	}
	if nestedEnum.Parent != parent {
		t.Errorf("nestedEnum.Parent = %v, want %v", nestedEnum.Parent, parent)
	}
	if val.Parent != nestedEnum {
		t.Errorf("val.Parent = %v, want %v", val.Parent, nestedEnum)
	}
}

func TestNewTestAPI_DeduplicatesNestedEnums(t *testing.T) {
	enum := api.NewTestEnum("NestedEnum")
	parent := api.NewTestMessage("Parent").WithEnums(enum)

	model := api.NewTestAPI([]*api.Message{parent}, []*api.Enum{enum}, nil)

	if got := len(parent.Enums); got != 1 {
		t.Errorf("len(parent.Enums) = %d, want 1", got)
	}
	if got := model.Enum(enum.ID); got != enum {
		t.Errorf("model.Enum(%q) = %v, want %v", enum.ID, got, enum)
	}
}

func TestNewTestAPI_DeduplicatesNestedMessages(t *testing.T) {
	child := api.NewTestMessage("Child")
	parent := api.NewTestMessage("Parent").WithMessages(child)

	model := api.NewTestAPI([]*api.Message{parent, child}, nil, nil)

	if got := len(parent.Messages); got != 1 {
		t.Errorf("len(parent.Messages) = %d, want 1", got)
	}
	if got := model.Message(child.ID); got != child {
		t.Errorf("model.Message(%q) = %v, want %v", child.ID, got, child)
	}
}

func TestService_WithDocumentation(t *testing.T) {
	const doc = "Test service documentation."
	s := api.NewTestService("TestService").WithDocumentation(doc)
	if s.Documentation != doc {
		t.Errorf("s.Documentation = %q, want %q", s.Documentation, doc)
	}
}

func TestMethod_ReturnEmpty(t *testing.T) {
	m := api.NewTestMethod("Delete").ReturnEmpty()
	if got := m.ReturnsEmpty; !got {
		t.Errorf("m.ReturnsEmpty = %v, want true", got)
	}
}
