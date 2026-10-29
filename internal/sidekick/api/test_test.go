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
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/api/apitest"
)

func TestMethod_WithBindings_NilPathInfo(t *testing.T) {
	method := api.NewTestMethod("Test").WithPathInfo(nil)
	b1 := api.NewPathBindingBuilder().WithVerb("GET").WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").Build()).Build()
	b2 := api.NewPathBindingBuilder().WithVerb("POST").WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v2").Build()).Build()

	method.WithBindings(b1, b2)

	if method.PathInfo == nil {
		t.Fatal("expected PathInfo to be initialized, got nil")
	}
	if diff := cmp.Diff([]*api.PathBinding{b1, b2}, method.PathInfo.Bindings(), cmp.AllowUnexported(api.PathBinding{}, api.PathTemplate{}, api.PathSegment{}, api.PathVariable{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestMethod_WithBindings_NonNilPathInfo(t *testing.T) {
	method := api.NewTestMethod("Test")
	b1 := api.NewPathBindingBuilder().WithVerb("DELETE").WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").Build()).Build()

	method.WithBindings(b1)

	if diff := cmp.Diff([]*api.PathBinding{b1}, method.PathInfo.Bindings(), cmp.AllowUnexported(api.PathBinding{}, api.PathTemplate{}, api.PathSegment{}, api.PathVariable{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestMethod_WithBodyFieldPath_NilPathInfo(t *testing.T) {
	method := api.NewTestMethod("Test").WithPathInfo(nil)
	method.WithBodyFieldPath("data")

	if method.PathInfo == nil {
		t.Fatal("expected PathInfo to be initialized, got nil")
	}
	if method.PathInfo.BodyFieldPath() != "data" {
		t.Errorf("expected BodyFieldPath to be 'data', got %q", method.PathInfo.BodyFieldPath())
	}
}

func TestMethod_WithVerb_NilPathInfo(t *testing.T) {
	method := api.NewTestMethod("Test").WithPathInfo(nil)
	method.WithVerb("PUT")

	if method.PathInfo == nil || len(method.PathInfo.Bindings()) == 0 {
		t.Fatal("expected PathInfo and at least one binding to be initialized")
	}
	if method.PathInfo.Bindings()[0].Verb() != "PUT" {
		t.Errorf("expected Verb to be 'PUT', got %q", method.PathInfo.Bindings()[0].Verb())
	}
}

func TestMethod_WithPathTemplate_NilPathInfo(t *testing.T) {
	method := api.NewTestMethod("Test").WithPathInfo(nil)
	pt := api.NewPathTemplateBuilder().WithLiteral("v1").Build()
	method.WithPathTemplate(pt)

	if method.PathInfo == nil || len(method.PathInfo.Bindings()) == 0 {
		t.Fatal("expected PathInfo and at least one binding to be initialized")
	}
	if method.PathInfo.Bindings()[0].PathTemplate() != pt {
		t.Errorf("expected PathTemplate to match")
	}
}

func TestMethod_WithQueryParameters_NilPathInfo(t *testing.T) {
	method := api.NewTestMethod("Test").WithPathInfo(nil)
	method.WithQueryParameters("page_size")

	if method.PathInfo == nil || len(method.PathInfo.Bindings()) == 0 {
		t.Fatal("expected PathInfo and at least one binding to be initialized")
	}
	want := map[string]bool{"page_size": true}
	if diff := cmp.Diff(want, method.PathInfo.Bindings()[0].QueryParameters()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestMethod_WithDocumentation(t *testing.T) {
	method := api.NewTestMethod("Test").WithDocumentation("A test method.")
	if method.Documentation != "A test method." {
		t.Errorf("expected Documentation 'A test method.', got %q", method.Documentation)
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

func TestMessage_WithMessages_UpdatesOneOfIDs(t *testing.T) {
	oneof := api.NewTestOneOf("choice")
	child := api.NewTestMessage("Child").WithOneOfs(oneof)
	_ = api.NewTestMessage("Parent").WithMessages(child)

	if got, want := child.OneOfs[0].ID, ".test.Parent.Child.choice"; got != want {
		t.Errorf("child.OneOfs[0].ID = %q, want %q", got, want)
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

func TestNewTestMap(t *testing.T) {
	got := api.NewTestMap("TestMap", api.TypezString, api.TypezInt32)
	want := &api.Map{
		ID: ".test.TestMap",
		Key: &api.Field{
			Name:     "key",
			JSONName: "key",
			ID:       ".test.TestMap.key",
			Typez:    api.TypezString,
		},
		Value: &api.Field{
			Name:     "value",
			JSONName: "value",
			ID:       ".test.TestMap.value",
			Typez:    api.TypezInt32,
		},
	}
	apitest.CheckMap(t, got, want)
}

func TestNewTestMapWithFields(t *testing.T) {
	valMsg := api.NewTestMessage("ValueMsg")
	key := api.NewTestField("custom_key").WithType(api.TypezString)
	val := api.NewTestField("custom_val").WithMessageType(valMsg)
	got := api.NewTestMapWithFields(".custom.package.MyMap", key, val)
	want := &api.Map{
		ID: ".custom.package.MyMap",
		Key: &api.Field{
			Name:     "custom_key",
			JSONName: "customKey",
			ID:       ".custom.package.MyMap.custom_key",
			Typez:    api.TypezString,
		},
		Value: &api.Field{
			Name:        "custom_val",
			JSONName:    "customVal",
			ID:          ".custom.package.MyMap.custom_val",
			Typez:       api.TypezMessage,
			TypezID:     valMsg.ID,
			MessageType: valMsg,
		},
	}
	apitest.CheckMap(t, got, want)
}

func TestMap_WithID(t *testing.T) {
	m := api.NewTestMap("TestMap", api.TypezString, api.TypezInt32)
	m.WithID(".new.package.TestMap")
	if got, want := m.ID, ".new.package.TestMap"; got != want {
		t.Errorf("m.ID = %q, want %q", got, want)
	}
	if got, want := m.Key.ID, ".new.package.TestMap.key"; got != want {
		t.Errorf("m.Key.ID = %q, want %q", got, want)
	}
	if got, want := m.Value.ID, ".new.package.TestMap.value"; got != want {
		t.Errorf("m.Value.ID = %q, want %q", got, want)
	}
}

func TestField_WithMap(t *testing.T) {
	f := api.NewTestField("test_field").WithRepeated().WithOptional().WithMap()
	if got, want := f.Typez, api.TypezMap; got != want {
		t.Errorf("f.Typez = %v, want %v", got, want)
	}
	if !f.Map {
		t.Errorf("f.Map = %v, want true", f.Map)
	}
	if f.Repeated {
		t.Errorf("f.Repeated = %v, want false", f.Repeated)
	}
	if f.Optional {
		t.Errorf("f.Optional = %v, want false", f.Optional)
	}
}

func TestField_WithMapType(t *testing.T) {
	m := api.NewTestMap("TestMap", api.TypezString, api.TypezInt32)
	f := api.NewTestField("test_field").WithRepeated().WithOptional().WithMapType(m)
	if got := f.MapType; got != m {
		t.Errorf("f.MapType = %v, want %v", got, m)
	}
	if got, want := f.Typez, api.TypezMap; got != want {
		t.Errorf("f.Typez = %v, want %v", got, want)
	}
	if !f.Map {
		t.Errorf("f.Map = %v, want true", f.Map)
	}
	if f.Repeated {
		t.Errorf("f.Repeated = %v, want false", f.Repeated)
	}
	if f.Optional {
		t.Errorf("f.Optional = %v, want false", f.Optional)
	}
	if got, want := f.TypezID, m.ID; got != want {
		t.Errorf("f.TypezID = %q, want %q", got, want)
	}
}

func TestAPI_WithMaps(t *testing.T) {
	m1 := api.NewTestMap("Map1", api.TypezString, api.TypezInt32)
	m2 := api.NewTestMap("Map2", api.TypezString, api.TypezString)
	model := api.NewTestAPI(nil, nil, nil).WithMaps(m1, m2)

	if got := model.Map(m1.ID); got != m1 {
		t.Errorf("model.Map(%q) = %v, want %v", m1.ID, got, m1)
	}
	if got := model.Map(m2.ID); got != m2 {
		t.Errorf("model.Map(%q) = %v, want %v", m2.ID, got, m2)
	}
	var count int
	for range model.AllMaps() {
		count++
	}
	if count != 2 {
		t.Errorf("AllMaps count = %d, want 2", count)
	}
}

func TestAPI_FluentSetters(t *testing.T) {
	svc := api.NewTestService("TestService")
	model := api.NewTestAPI(nil, nil, []*api.Service{svc}).
		WithName("custom-name").
		WithTitle("Custom Title").
		WithDescription("Custom Description").
		WithQuickstartService(svc)

	if got, want := model.Name, "custom-name"; got != want {
		t.Errorf("model.Name = %q, want %q", got, want)
	}
	if got, want := model.Title, "Custom Title"; got != want {
		t.Errorf("model.Title = %q, want %q", got, want)
	}
	if got, want := model.Description, "Custom Description"; got != want {
		t.Errorf("model.Description = %q, want %q", got, want)
	}
	if got := model.QuickstartService; got != svc {
		t.Errorf("model.QuickstartService = %v, want %v", got, svc)
	}
}

func TestService_FluentSetters(t *testing.T) {
	svc := api.NewTestService("TestService").
		WithDefaultHost("test.googleapis.com").
		WithID(".custom.TestService")
	if got, want := svc.DefaultHost, "test.googleapis.com"; got != want {
		t.Errorf("svc.DefaultHost = %q, want %q", got, want)
	}
	if got, want := svc.ID, ".custom.TestService"; got != want {
		t.Errorf("svc.ID = %q, want %q", got, want)
	}
}

func TestMethod_FluentSetters(t *testing.T) {
	reqID := api.NewTestField("request_id").WithType(api.TypezString)
	routing := api.NewRoutingInfoBuilder().WithName("bucket").Build()
	m := api.NewTestMethod("Create").
		WithID(".custom.Service.Create").
		WithAPIVersion("v1_20260101").
		WithPathInfo(nil).
		WithRouting(routing).
		WithAutoPopulated(reqID)

	if got, want := m.ID, ".custom.Service.Create"; got != want {
		t.Errorf("m.ID = %q, want %q", got, want)
	}
	if got, want := m.APIVersion, "v1_20260101"; got != want {
		t.Errorf("m.APIVersion = %q, want %q", got, want)
	}
	if m.PathInfo != nil {
		t.Errorf("m.PathInfo = %v, want nil", m.PathInfo)
	}
	if len(m.Routing) != 1 || m.Routing[0] != routing {
		t.Errorf("m.Routing = %v, want [%v]", m.Routing, routing)
	}
	if len(m.AutoPopulated) != 1 || m.AutoPopulated[0] != reqID {
		t.Errorf("m.AutoPopulated = %v, want [%v]", m.AutoPopulated, reqID)
	}
	if !reqID.AutoPopulated {
		t.Errorf("reqID.AutoPopulated = false, want true")
	}
}

func TestField_FluentSetters(t *testing.T) {
	enum := api.NewTestEnum("State").WithPackage("test.v1")
	pattern := &api.ResourceNamePattern{Segments: []api.ResourceNameSegment{{Literal: "items"}}}
	f := api.NewTestField("state").
		WithEnumType(enum).
		WithAutoPopulated().
		WithResourceNamePattern(pattern)

	if f.EnumType != enum {
		t.Errorf("f.EnumType = %v, want %v", f.EnumType, enum)
	}
	if f.Typez != api.TypezEnum {
		t.Errorf("f.Typez = %v, want %v", f.Typez, api.TypezEnum)
	}
	if got, want := f.TypezID, enum.ID; got != want {
		t.Errorf("f.TypezID = %q, want %q", got, want)
	}
	if !f.AutoPopulated {
		t.Errorf("f.AutoPopulated = false, want true")
	}
	if f.ResourceNamePattern != pattern {
		t.Errorf("f.ResourceNamePattern = %v, want %v", f.ResourceNamePattern, pattern)
	}
}

func TestMethod_TypeIDSetters(t *testing.T) {
	m := api.NewTestMethod("DoWork").
		WithInputTypeID(".test.Request").
		WithOutputTypeID(".test.Response").
		WithSourceServiceID(".test.Service")

	if m.InputTypeID != ".test.Request" {
		t.Errorf("m.InputTypeID = %q, want %q", m.InputTypeID, ".test.Request")
	}
	if m.OutputTypeID != ".test.Response" {
		t.Errorf("m.OutputTypeID = %q, want %q", m.OutputTypeID, ".test.Response")
	}
	if m.SourceServiceID != ".test.Service" {
		t.Errorf("m.SourceServiceID = %q, want %q", m.SourceServiceID, ".test.Service")
	}
}

func TestResourceNamePattern_Builder(t *testing.T) {
	p := api.NewTestResourceNamePattern(
		api.ResourceNameSegment{Literal: "projects"},
		api.ResourceNameSegment{Variable: "project"},
	)
	if len(p.Segments) != 2 {
		t.Fatalf("len(p.Segments) = %d, want 2", len(p.Segments))
	}
	if p.Segments[0].Literal != "projects" || p.Segments[1].Variable != "project" {
		t.Errorf("unexpected segments: %+v", p.Segments)
	}
}

func TestSampleInfo_Builder(t *testing.T) {
	f := api.NewTestField("name")
	s := api.NewTestSampleInfo(f)
	if s.ResourceNameField != f {
		t.Errorf("s.ResourceNameField = %v, want %v", s.ResourceNameField, f)
	}
}

func TestPaginationInfo_Builder(t *testing.T) {
	token := api.NewTestField("next_page_token")
	items := api.NewTestField("items")
	p := api.NewTestPaginationInfo(token, items)

	if p.NextPageToken != token {
		t.Errorf("p.NextPageToken = %v, want %v", p.NextPageToken, token)
	}
	if p.PageableItem != items {
		t.Errorf("p.PageableItem = %v, want %v", p.PageableItem, items)
	}
}
