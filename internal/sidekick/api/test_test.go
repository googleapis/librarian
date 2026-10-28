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

	if method.PathInfo() == nil {
		t.Fatal("expected PathInfo to be initialized, got nil")
	}
	if diff := cmp.Diff([]*api.PathBinding{b1, b2}, method.PathInfo().Bindings(), cmp.AllowUnexported(api.PathBinding{}, api.PathTemplate{}, api.PathSegment{}, api.PathVariable{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestMethod_WithBindings_NonNilPathInfo(t *testing.T) {
	method := api.NewTestMethod("Test")
	b1 := api.NewPathBindingBuilder().WithVerb("DELETE").WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").Build()).Build()

	method.WithBindings(b1)

	if diff := cmp.Diff([]*api.PathBinding{b1}, method.PathInfo().Bindings(), cmp.AllowUnexported(api.PathBinding{}, api.PathTemplate{}, api.PathSegment{}, api.PathVariable{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestMethod_WithBodyFieldPath_NilPathInfo(t *testing.T) {
	method := api.NewTestMethod("Test").WithPathInfo(nil)
	method.WithBodyFieldPath("data")

	if method.PathInfo() == nil {
		t.Fatal("expected PathInfo to be initialized, got nil")
	}
	if method.PathInfo().BodyFieldPath() != "data" {
		t.Errorf("expected BodyFieldPath to be 'data', got %q", method.PathInfo().BodyFieldPath())
	}
}

func TestMethod_WithVerb_NilPathInfo(t *testing.T) {
	method := api.NewTestMethod("Test").WithPathInfo(nil)
	method.WithVerb("PUT")

	if method.PathInfo() == nil || len(method.PathInfo().Bindings()) == 0 {
		t.Fatal("expected PathInfo and at least one binding to be initialized")
	}
	if method.PathInfo().Bindings()[0].Verb() != "PUT" {
		t.Errorf("expected Verb to be 'PUT', got %q", method.PathInfo().Bindings()[0].Verb())
	}
}

func TestMethod_WithPathTemplate_NilPathInfo(t *testing.T) {
	method := api.NewTestMethod("Test").WithPathInfo(nil)
	pt := api.NewPathTemplateBuilder().WithLiteral("v1").Build()
	method.WithPathTemplate(pt)

	if method.PathInfo() == nil || len(method.PathInfo().Bindings()) == 0 {
		t.Fatal("expected PathInfo and at least one binding to be initialized")
	}
	if method.PathInfo().Bindings()[0].PathTemplate() != pt {
		t.Errorf("expected PathTemplate to match")
	}
}

func TestMethod_WithQueryParameters_NilPathInfo(t *testing.T) {
	method := api.NewTestMethod("Test").WithPathInfo(nil)
	method.WithQueryParameters("page_size")

	if method.PathInfo() == nil || len(method.PathInfo().Bindings()) == 0 {
		t.Fatal("expected PathInfo and at least one binding to be initialized")
	}
	want := map[string]bool{"page_size": true}
	if diff := cmp.Diff(want, method.PathInfo().Bindings()[0].QueryParameters()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestMethod_WithDocumentation(t *testing.T) {
	method := api.NewTestMethod("Test").WithDocumentation("A test method.")
	if method.Documentation() != "A test method." {
		t.Errorf("expected Documentation 'A test method.', got %q", method.Documentation())
	}
}

func TestPathBinding_Fluent(t *testing.T) {
	pt := api.NewPathTemplateBuilder().WithLiteral("v1").Build()
	b := api.NewPathBindingBuilder().
		WithVerb("GET").
		WithPathTemplate(pt).
		WithVerb("POST").
		WithQueryParameters("filter").
		WithQueryParameter("extra").
		Build()
	b.SetAnnotations("my-codec")

	if b.Verb() != "POST" {
		t.Errorf("expected Verb 'POST', got %q", b.Verb())
	}
	if b.PathTemplate() != pt {
		t.Errorf("expected PathTemplate to match")
	}
	if !b.QueryParameters()["filter"] || !b.QueryParameters()["extra"] {
		t.Errorf("expected filter and extra query parameters to be true, got %v", b.QueryParameters())
	}
	if b.Codec() != "my-codec" {
		t.Errorf("expected Codec 'my-codec', got %v", b.Codec())
	}
}

func TestField_WithJSONName(t *testing.T) {
	f := api.NewTestField("foo_bar").WithJSONName("customName")
	if f.JSONName() != "customName" {
		t.Errorf("expected JSONName 'customName', got %q", f.JSONName())
	}
}

func TestMessage_WithDocumentation(t *testing.T) {
	m := api.NewTestMessage("Test").WithDocumentation("A test message.")
	if m.Documentation() != "A test message." {
		t.Errorf("expected Documentation 'A test message.', got %q", m.Documentation())
	}
}

func TestOneOf_WithDocumentation(t *testing.T) {
	o := api.NewTestOneOf("choice").WithDocumentation("A test oneof.")
	if o.Documentation() != "A test oneof." {
		t.Errorf("expected Documentation 'A test oneof.', got %q", o.Documentation())
	}
}

func TestField_WithRecursive(t *testing.T) {
	f := api.NewTestField("child").WithRecursive()
	if !f.Recursive() {
		t.Errorf("expected Recursive to be true")
	}
}

func TestNewTestAPI_IndexesNestedMessages(t *testing.T) {
	child := api.NewTestMessage("Child")
	parent := api.NewTestMessage("Parent").WithMessages(child)
	model := api.NewTestAPI([]*api.Message{parent}, nil, nil)

	if got := model.Message(child.ID()); got != child {
		t.Errorf("model.Message(%q) = %v, want %v", child.ID(), got, child)
	}
	if child.Parent() != parent {
		t.Errorf("child.Parent() = %v, want %v", child.Parent(), parent)
	}
}

func TestNewTestAPI_IndexesNestedEnums(t *testing.T) {
	val := api.NewTestEnumValue("VAL", 0)
	nestedEnum := api.NewTestEnum("NestedEnum").WithValues(val)
	// Bypassing WithEnums() deliberately leaves Parent and Value.Parent unwired,
	// verifying that NewTestAPI properly initializes them during indexing.
	parent := api.NewMessageBuilderFrom(api.NewTestMessage("Parent")).AddEnum(nestedEnum).Build()

	model := api.NewTestAPI([]*api.Message{parent}, nil, nil)

	if got := model.Enum(nestedEnum.ID()); got != nestedEnum {
		t.Errorf("model.Enum(%q) = %v, want %v", nestedEnum.ID(), got, nestedEnum)
	}
	if nestedEnum.Parent() != parent {
		t.Errorf("nestedEnum.Parent() = %v, want %v", nestedEnum.Parent(), parent)
	}
	if val.Parent() != nestedEnum {
		t.Errorf("val.Parent() = %v, want %v", val.Parent(), nestedEnum)
	}
}

func TestNewTestAPI_DeduplicatesNestedEnums(t *testing.T) {
	enum := api.NewTestEnum("NestedEnum")
	parent := api.NewTestMessage("Parent").WithEnums(enum)

	model := api.NewTestAPI([]*api.Message{parent}, []*api.Enum{enum}, nil)

	if got := len(parent.Enums()); got != 1 {
		t.Errorf("len(parent.Enums()) = %d, want 1", got)
	}
	if got := model.Enum(enum.ID()); got != enum {
		t.Errorf("model.Enum(%q) = %v, want %v", enum.ID(), got, enum)
	}
}

func TestNewTestAPI_DeduplicatesNestedMessages(t *testing.T) {
	child := api.NewTestMessage("Child")
	parent := api.NewTestMessage("Parent").WithMessages(child)

	model := api.NewTestAPI([]*api.Message{parent, child}, nil, nil)

	if got := len(parent.Messages()); got != 1 {
		t.Errorf("len(parent.Messages()) = %d, want 1", got)
	}
	if got := model.Message(child.ID()); got != child {
		t.Errorf("model.Message(%q) = %v, want %v", child.ID(), got, child)
	}
}

func TestMessage_WithMessages_UpdatesOneOfIDs(t *testing.T) {
	oneof := api.NewTestOneOf("choice")
	child := api.NewTestMessage("Child").WithOneOfs(oneof)
	_ = api.NewTestMessage("Parent").WithMessages(child)

	if got, want := child.OneOfs()[0].ID(), ".test.Parent.Child.choice"; got != want {
		t.Errorf("child.OneOfs()[0].ID() = %q, want %q", got, want)
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
	if got := m.ReturnsEmpty(); !got {
		t.Errorf("m.ReturnsEmpty() = %v, want true", got)
	}
}

func TestNewTestMap(t *testing.T) {
	got := api.NewTestMap("TestMap", api.TypezString, api.TypezInt32)
	want := api.NewMapBuilder().
		WithID(".test.TestMap").
		WithKey(api.NewFieldBuilder().
			WithName("key").
			WithJSONName("key").
			WithID(".test.TestMap.key").
			WithTypez(api.TypezString).
			Build()).
		WithValue(api.NewFieldBuilder().
			WithName("value").
			WithJSONName("value").
			WithID(".test.TestMap.value").
			WithTypez(api.TypezInt32).
			Build()).
		Build()
	apitest.CheckMap(t, got, want)
}

func TestNewTestMapWithFields(t *testing.T) {
	valMsg := api.NewTestMessage("ValueMsg")
	key := api.NewTestField("custom_key").WithType(api.TypezString)
	val := api.NewTestField("custom_val").WithMessageType(valMsg)
	got := api.NewTestMapWithFields(".custom.package.MyMap", key, val)
	want := api.NewMapBuilder().
		WithID(".custom.package.MyMap").
		WithKey(api.NewFieldBuilder().
			WithName("custom_key").
			WithJSONName("customKey").
			WithID(".custom.package.MyMap.custom_key").
			WithTypez(api.TypezString).
			Build()).
		WithValue(api.NewFieldBuilder().
			WithName("custom_val").
			WithJSONName("customVal").
			WithID(".custom.package.MyMap.custom_val").
			WithTypez(api.TypezMessage).
			WithTypezID(valMsg.ID()).
			WithMessageType(valMsg).
			Build()).
		Build()
	apitest.CheckMap(t, got, want)
}

func TestMap_WithID(t *testing.T) {
	m := api.NewTestMap("TestMap", api.TypezString, api.TypezInt32)
	m.WithID(".new.package.TestMap")
	if got, want := m.ID(), ".new.package.TestMap"; got != want {
		t.Errorf("m.ID() = %q, want %q", got, want)
	}
	if got, want := m.Key().ID(), ".new.package.TestMap.key"; got != want {
		t.Errorf("m.Key().ID() = %q, want %q", got, want)
	}
	if got, want := m.Value().ID(), ".new.package.TestMap.value"; got != want {
		t.Errorf("m.Value().ID() = %q, want %q", got, want)
	}
}

func TestMapBuilderFrom(t *testing.T) {
	orig := api.NewTestMap("TestMap", api.TypezString, api.TypezInt32)
	copied := api.NewMapBuilderFrom(orig).Build()
	apitest.CheckMap(t, copied, orig)

	fromNil := api.NewMapBuilderFrom(nil).Build()
	if fromNil.ID() != "" || fromNil.Key() != nil || fromNil.Value() != nil {
		t.Errorf("NewMapBuilderFrom(nil) = %+v, want empty Map", fromNil)
	}
}

func TestField_WithMap(t *testing.T) {
	f := api.NewTestField("test_field").WithRepeated().WithOptional().WithMap()
	if got, want := f.Typez(), api.TypezMap; got != want {
		t.Errorf("f.Typez() = %v, want %v", got, want)
	}
	if !f.Map() {
		t.Errorf("f.Map() = %v, want true", f.Map())
	}
	if f.Repeated() {
		t.Errorf("f.Repeated() = %v, want false", f.Repeated())
	}
	if f.Optional() {
		t.Errorf("f.Optional() = %v, want false", f.Optional())
	}
}

func TestField_WithMapType(t *testing.T) {
	m := api.NewTestMap("TestMap", api.TypezString, api.TypezInt32)
	f := api.NewTestField("test_field").WithRepeated().WithOptional().WithMapType(m)
	if got := f.MapType(); got != m {
		t.Errorf("f.MapType() = %v, want %v", got, m)
	}
	if got, want := f.Typez(), api.TypezMap; got != want {
		t.Errorf("f.Typez() = %v, want %v", got, want)
	}
	if !f.Map() {
		t.Errorf("f.Map() = %v, want true", f.Map())
	}
	if f.Repeated() {
		t.Errorf("f.Repeated() = %v, want false", f.Repeated())
	}
	if f.Optional() {
		t.Errorf("f.Optional() = %v, want false", f.Optional())
	}
	if got, want := f.TypezID(), m.ID(); got != want {
		t.Errorf("f.TypezID() = %q, want %q", got, want)
	}
}

func TestAPI_WithMaps(t *testing.T) {
	m1 := api.NewTestMap("Map1", api.TypezString, api.TypezInt32)
	m2 := api.NewTestMap("Map2", api.TypezString, api.TypezString)
	model := api.NewTestAPI(nil, nil, nil).WithMaps(m1, m2)

	if got := model.Map(m1.ID()); got != m1 {
		t.Errorf("model.Map(%q) = %v, want %v", m1.ID(), got, m1)
	}
	if got := model.Map(m2.ID()); got != m2 {
		t.Errorf("model.Map(%q) = %v, want %v", m2.ID(), got, m2)
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

	if got, want := m.ID(), ".custom.Service.Create"; got != want {
		t.Errorf("m.ID() = %q, want %q", got, want)
	}
	if got, want := m.APIVersion(), "v1_20260101"; got != want {
		t.Errorf("m.APIVersion() = %q, want %q", got, want)
	}
	if m.PathInfo() != nil {
		t.Errorf("m.PathInfo() = %v, want nil", m.PathInfo())
	}
	if len(m.Routing()) != 1 || m.Routing()[0] != routing {
		t.Errorf("m.Routing() = %v, want [%v]", m.Routing(), routing)
	}
	if len(m.AutoPopulated()) != 1 || m.AutoPopulated()[0] != reqID {
		t.Errorf("m.AutoPopulated() = %v, want [%v]", m.AutoPopulated(), reqID)
	}
	if !reqID.AutoPopulated() {
		t.Errorf("reqID.AutoPopulated() = false, want true")
	}
}

func TestField_FluentSetters(t *testing.T) {
	enum := api.NewTestEnum("State").WithPackage("test.v1")
	pattern := api.NewTestResourceNamePattern(api.NewResourceNameSegment("items", ""))
	f := api.NewTestField("state").
		WithEnumType(enum).
		WithAutoPopulated().
		WithResourceNamePattern(pattern)

	if f.EnumType() != enum {
		t.Errorf("f.EnumType() = %v, want %v", f.EnumType(), enum)
	}
	if f.Typez() != api.TypezEnum {
		t.Errorf("f.Typez() = %v, want %v", f.Typez(), api.TypezEnum)
	}
	if got, want := f.TypezID(), enum.ID(); got != want {
		t.Errorf("f.TypezID() = %q, want %q", got, want)
	}
	if !f.AutoPopulated() {
		t.Errorf("f.AutoPopulated() = false, want true")
	}
	if f.ResourceNamePattern() != pattern {
		t.Errorf("f.ResourceNamePattern() = %v, want %v", f.ResourceNamePattern(), pattern)
	}
}

func TestMethod_TypeIDSetters(t *testing.T) {
	m := api.NewTestMethod("DoWork").
		WithInputTypeID(".test.Request").
		WithOutputTypeID(".test.Response").
		WithSourceServiceID(".test.Service")

	if m.InputTypeID() != ".test.Request" {
		t.Errorf("m.InputTypeID() = %q, want %q", m.InputTypeID(), ".test.Request")
	}
	if m.OutputTypeID() != ".test.Response" {
		t.Errorf("m.OutputTypeID() = %q, want %q", m.OutputTypeID(), ".test.Response")
	}
	if m.SourceServiceID() != ".test.Service" {
		t.Errorf("m.SourceServiceID() = %q, want %q", m.SourceServiceID(), ".test.Service")
	}
}

func TestPathBinding_WithTargetResource(t *testing.T) {
	seg := *api.NewPathSegmentBuilder().WithLiteral("items").Build()
	tr := api.NewTargetResource([][]string{{"parent"}}, []api.PathSegment{seg})
	b := api.NewPathBindingBuilder().
		WithVerb("GET").
		WithQueryParameter("only").
		WithQueryParameterMap(map[string]bool{"mapKey": true}).
		WithTargetResource(tr).
		Build()
	if b.TargetResource() != tr {
		t.Errorf("b.TargetResource() = %v, want %v", b.TargetResource(), tr)
	}
	if diff := cmp.Diff([][]string{{"parent"}}, tr.FieldPaths()); diff != "" {
		t.Errorf("mismatch in FieldPaths() (-want +got):\n%s", diff)
	}
	if len(tr.Template()) != 1 || tr.Template()[0].Literal() != "items" {
		t.Errorf("unexpected Template(): %v", tr.Template())
	}
	if !b.QueryParameters()["mapKey"] {
		t.Errorf("expected mapKey query parameter to be true")
	}
}

func TestMethodSignature_Builders(t *testing.T) {
	f1 := api.NewTestField("name")
	f2 := api.NewTestField("parent")
	sig := api.NewTestMethodSignature("name", "parent").
		WithFields(f1).
		WithFields(f2)

	if diff := cmp.Diff([]string{"name", "parent"}, sig.Names()); diff != "" {
		t.Errorf("mismatch in Names (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]*api.Field{f1, f2}, sig.Fields(), cmp.AllowUnexported(api.Field{})); diff != "" {
		t.Errorf("mismatch in Fields (-want +got):\n%s", diff)
	}
	if sig.Method() != nil {
		t.Errorf("sig.Method() = %v, want nil", sig.Method())
	}

	built := api.NewMethodSignatureBuilder().
		WithNames("name").
		WithFields(f1).
		Build()
	if diff := cmp.Diff([]string{"name"}, built.Names()); diff != "" {
		t.Errorf("mismatch in built.Names (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]*api.Field{f1}, built.Fields(), cmp.AllowUnexported(api.Field{})); diff != "" {
		t.Errorf("mismatch in built.Fields (-want +got):\n%s", diff)
	}
}

func TestResourceNamePattern_Builder(t *testing.T) {
	p := api.NewTestResourceNamePattern(
		api.NewResourceNameSegment("projects", ""),
		api.NewResourceNameSegment("", "project"),
	)
	if len(p.Segments()) != 2 {
		t.Fatalf("len(p.Segments()) = %d, want 2", len(p.Segments()))
	}
	if p.Segments()[0].Literal() != "projects" || p.Segments()[1].Variable() != "project" {
		t.Errorf("unexpected segments: %+v", p.Segments())
	}

	rr := api.NewResourceReferenceBuilder().
		WithType("test.googleapis.com/Book").
		WithChildType("test.googleapis.com/Page").
		Build()
	rr.SetAnnotations("rr-codec")
	if rr.Type() != "test.googleapis.com/Book" {
		t.Errorf("rr.Type() = %q, want test.googleapis.com/Book", rr.Type())
	}
	if rr.ChildType() != "test.googleapis.com/Page" {
		t.Errorf("rr.ChildType() = %q, want test.googleapis.com/Page", rr.ChildType())
	}
	if rr.Codec() != "rr-codec" {
		t.Errorf("rr.Codec() = %v, want rr-codec", rr.Codec())
	}
}

func TestRoutingInfo_Builders(t *testing.T) {
	v := api.NewTestRoutingInfoVariant("name", "id")
	r := api.NewTestRoutingInfo("routing", v)

	if r.Name() != "routing" {
		t.Errorf("r.Name() = %q, want 'routing'", r.Name())
	}
	if len(r.Variants()) != 1 || r.Variants()[0] != v {
		t.Errorf("r.Variants() = %v, want [%v]", r.Variants(), v)
	}
	if diff := cmp.Diff([]string{"name", "id"}, v.FieldPath()); diff != "" {
		t.Errorf("mismatch in FieldPath (-want +got):\n%s", diff)
	}
}

func TestOperationInfo_Builder(t *testing.T) {
	op := api.NewTestOperationInfo(".test.Response", ".test.Metadata")
	if op.ResponseTypeID() != ".test.Response" {
		t.Errorf("op.ResponseTypeID() = %q, want .test.Response", op.ResponseTypeID())
	}
	if op.MetadataTypeID() != ".test.Metadata" {
		t.Errorf("op.MetadataTypeID() = %q, want .test.Metadata", op.MetadataTypeID())
	}
	op.SetAnnotations("op-codec")
	if op.Codec() != "op-codec" {
		t.Errorf("op.Codec() = %v, want op-codec", op.Codec())
	}
}

func TestDiscoveryLro_Builder(t *testing.T) {
	lro := api.NewTestDiscoveryLro("project", "zone")
	if diff := cmp.Diff([]string{"project", "zone"}, lro.PollingPathParameters()); diff != "" {
		t.Errorf("mismatch in PollingPathParameters (-want +got):\n%s", diff)
	}
	lro.SetAnnotations("lro-codec")
	if lro.Codec() != "lro-codec" {
		t.Errorf("lro.Codec() = %v, want lro-codec", lro.Codec())
	}
}

func TestSampleInfo_Builder(t *testing.T) {
	f := api.NewTestField("name")
	msgField := api.NewTestField("book")
	idField := api.NewTestField("book_id")
	maskField := api.NewTestField("update_mask")
	s := api.NewSampleInfoBuilder().
		WithResourceNameField(f).
		WithMessageField(msgField).
		WithResourceIDField(idField).
		WithUpdateMaskField(maskField).
		WithIsRequestResourceName(true).
		WithIsMessageResourceName(true).
		Build()
	if s.ResourceNameField() != f {
		t.Errorf("s.ResourceNameField() = %v, want %v", s.ResourceNameField(), f)
	}
	if s.MessageField() != msgField {
		t.Errorf("s.MessageField() = %v, want %v", s.MessageField(), msgField)
	}
	if s.ResourceIDField() != idField {
		t.Errorf("s.ResourceIDField() = %v, want %v", s.ResourceIDField(), idField)
	}
	if s.UpdateMaskField() != maskField {
		t.Errorf("s.UpdateMaskField() = %v, want %v", s.UpdateMaskField(), maskField)
	}
	if !s.IsRequestResourceName() {
		t.Errorf("s.IsRequestResourceName() = false, want true")
	}
	if !s.IsMessageResourceName() {
		t.Errorf("s.IsMessageResourceName() = false, want true")
	}

	testSample := api.NewTestSampleInfo(f)
	if testSample.ResourceNameField() != f {
		t.Errorf("testSample.ResourceNameField() = %v, want %v", testSample.ResourceNameField(), f)
	}
}

func TestMethod_Builder(t *testing.T) {
	inMsg := api.NewTestMessage("Req")
	outMsg := api.NewTestMessage("Resp")
	pi := api.NewPathInfoBuilder().WithBodyFieldPath("*").Build()
	pag := api.NewTestField("page_token")
	op := api.NewTestOperationInfo(".test.LroResp", ".test.LroMeta")
	dlro := api.NewTestDiscoveryLro("project")
	routing := api.NewTestRoutingInfo("name")
	autoPop := api.NewTestField("request_id")
	sig := api.NewTestMethodSignature("name")
	sample := api.NewSampleInfoBuilder().WithResourceNameField(pag).Build()
	svc := api.NewTestService("Svc")
	srcSvc := api.NewTestService("SrcSvc")

	m := api.NewMethodBuilder().
		WithName("Create").
		WithID(".test.Svc.Create").
		WithDocumentation("Create docs").
		WithDeprecated(true).
		WithInputTypeID(".test.Req").
		WithOutputTypeID(".test.Resp").
		WithInputType(inMsg).
		WithOutputType(outMsg).
		WithReturnsEmpty(true).
		WithPathInfo(pi).
		WithPagination(pag).
		WithClientSideStreaming(true).
		WithServerSideStreaming(true).
		WithOperationInfo(op).
		WithDiscoveryLro(dlro).
		WithIsLroPoller(true).
		WithRouting([]*api.RoutingInfo{routing}).
		WithAutoPopulated([]*api.Field{autoPop}).
		WithSignatures([]*api.MethodSignature{sig}).
		WithService(svc).
		WithSourceServiceID(".test.SrcSvc").
		WithSourceService(srcSvc).
		WithAPIVersion("v1").
		WithSampleInfo(sample).
		Build()
	m.SetAnnotations("method-codec")

	if m.Name() != "Create" {
		t.Errorf("m.Name() = %q, want Create", m.Name())
	}
	if m.ID() != ".test.Svc.Create" {
		t.Errorf("m.ID() = %q, want .test.Svc.Create", m.ID())
	}
	if m.Documentation() != "Create docs" {
		t.Errorf("m.Documentation() = %q, want Create docs", m.Documentation())
	}
	if !m.Deprecated() {
		t.Errorf("m.Deprecated() = false, want true")
	}
	if m.InputTypeID() != ".test.Req" {
		t.Errorf("m.InputTypeID() = %q, want .test.Req", m.InputTypeID())
	}
	if m.OutputTypeID() != ".test.Resp" {
		t.Errorf("m.OutputTypeID() = %q, want .test.Resp", m.OutputTypeID())
	}
	if m.InputType() != inMsg {
		t.Errorf("m.InputType() = %v, want %v", m.InputType(), inMsg)
	}
	if m.OutputType() != outMsg {
		t.Errorf("m.OutputType() = %v, want %v", m.OutputType(), outMsg)
	}
	if !m.ReturnsEmpty() {
		t.Errorf("m.ReturnsEmpty() = false, want true")
	}
	if m.PathInfo() != pi {
		t.Errorf("m.PathInfo() = %v, want %v", m.PathInfo(), pi)
	}
	if m.Pagination() != pag {
		t.Errorf("m.Pagination() = %v, want %v", m.Pagination(), pag)
	}
	if !m.ClientSideStreaming() {
		t.Errorf("m.ClientSideStreaming() = false, want true")
	}
	if !m.ServerSideStreaming() {
		t.Errorf("m.ServerSideStreaming() = false, want true")
	}
	if m.OperationInfo() != op {
		t.Errorf("m.OperationInfo() = %v, want %v", m.OperationInfo(), op)
	}
	if m.DiscoveryLro() != dlro {
		t.Errorf("m.DiscoveryLro() = %v, want %v", m.DiscoveryLro(), dlro)
	}
	if !m.IsLroPoller() {
		t.Errorf("m.IsLroPoller() = false, want true")
	}
	if len(m.Routing()) != 1 || m.Routing()[0] != routing {
		t.Errorf("m.Routing() = %v, want [%v]", m.Routing(), routing)
	}
	if len(m.AutoPopulated()) != 1 || m.AutoPopulated()[0] != autoPop {
		t.Errorf("m.AutoPopulated() = %v, want [%v]", m.AutoPopulated(), autoPop)
	}
	if len(m.Signatures()) != 1 || m.Signatures()[0] != sig {
		t.Errorf("m.Signatures() = %v, want [%v]", m.Signatures(), sig)
	}
	if m.Service() != svc {
		t.Errorf("m.Service() = %v, want %v", m.Service(), svc)
	}
	if m.SourceServiceID() != ".test.SrcSvc" {
		t.Errorf("m.SourceServiceID() = %q, want .test.SrcSvc", m.SourceServiceID())
	}
	if m.SourceService() != srcSvc {
		t.Errorf("m.SourceService() = %v, want %v", m.SourceService(), srcSvc)
	}
	if m.APIVersion() != "v1" {
		t.Errorf("m.APIVersion() = %q, want v1", m.APIVersion())
	}
	if m.SampleInfo() != sample {
		t.Errorf("m.SampleInfo() = %v, want %v", m.SampleInfo(), sample)
	}
	if m.Codec() != "method-codec" {
		t.Errorf("m.Codec() = %v, want method-codec", m.Codec())
	}
}

func TestPaginationInfo_Builder(t *testing.T) {
	token := api.NewTestField("next_page_token")
	items := api.NewTestField("items")
	p := api.NewTestPaginationInfo(token, items)

	if p.NextPageToken() != token {
		t.Errorf("p.NextPageToken() = %v, want %v", p.NextPageToken(), token)
	}
	if p.PageableItem() != items {
		t.Errorf("p.PageableItem() = %v, want %v", p.PageableItem(), items)
	}
}

func TestPathSegment_Builder(t *testing.T) {
	v := api.NewPathVariableBuilder().WithFieldPath("name").WithSegments("items", "*").Build()
	s1 := api.NewPathSegmentBuilder().WithLiteral("v1").Build()
	s2 := api.NewPathSegmentBuilder().WithVariable(v).Build()

	if got := s1.Literal(); got != "v1" {
		t.Errorf("s1.Literal() = %q, want %q", got, "v1")
	}
	if got := s1.Variable(); got != nil {
		t.Errorf("s1.Variable() = %v, want nil", got)
	}
	if got := s2.Literal(); got != "" {
		t.Errorf("s2.Literal() = %q, want empty", got)
	}
	if got := s2.Variable(); got != v {
		t.Errorf("s2.Variable() = %v, want %v", got, v)
	}
}

func TestPathTemplate_Builder(t *testing.T) {
	v1 := api.NewPathVariableBuilder().WithFieldPath("project").WithSegments("*").Build()
	s1 := api.NewPathSegmentBuilder().WithLiteral("projects").Build()
	pt := api.NewPathTemplateBuilder().
		WithSegments(*s1).
		WithSegment(*api.NewPathSegmentBuilder().WithVariable(v1).Build()).
		WithLiteral("zones").
		WithVariable(api.NewPathVariableBuilder().WithFieldPath("zone").WithSegments("*").Build()).
		WithVariableNamed("instance", "info").
		WithVerb("start").
		Build()

	if got := pt.Verb(); got != "start" {
		t.Errorf("pt.Verb() = %q, want %q", got, "start")
	}
	if got := len(pt.Segments()); got != 5 {
		t.Fatalf("len(pt.Segments()) = %d, want 5", got)
	}
	if got := pt.Segments()[0].Literal(); got != "projects" {
		t.Errorf("Segments()[0].Literal() = %q, want %q", got, "projects")
	}
	if got := pt.Segments()[1].Variable(); got != v1 {
		t.Errorf("Segments()[1].Variable() = %v, want %v", got, v1)
	}
	if got := pt.Segments()[2].Literal(); got != "zones" {
		t.Errorf("Segments()[2].Literal() = %q, want %q", got, "zones")
	}
	if diff := cmp.Diff([]string{"zone"}, pt.Segments()[3].Variable().FieldPath()); diff != "" {
		t.Errorf("mismatch in Segments()[3].Variable().FieldPath() (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"instance", "info"}, pt.Segments()[4].Variable().FieldPath()); diff != "" {
		t.Errorf("mismatch in Segments()[4].Variable().FieldPath() (-want +got):\n%s", diff)
	}
}

func TestPathInfo_Builder(t *testing.T) {
	b1 := api.NewPathBindingBuilder().WithVerb("GET").Build()
	b2 := api.NewPathBindingBuilder().WithVerb("POST").Build()
	pi := api.NewPathInfoBuilder().
		WithBindings(b1).
		WithBinding(b2).
		WithBodyFieldPath("body").
		Build()
	pi.SetAnnotations("pi-codec")

	if len(pi.Bindings()) != 2 || pi.Bindings()[0] != b1 || pi.Bindings()[1] != b2 {
		t.Errorf("unexpected Bindings(): %v", pi.Bindings())
	}
	if pi.BodyFieldPath() != "body" {
		t.Errorf("pi.BodyFieldPath() = %q, want %q", pi.BodyFieldPath(), "body")
	}
	if pi.Codec() != "pi-codec" {
		t.Errorf("pi.Codec() = %v, want %q", pi.Codec(), "pi-codec")
	}
}

func TestResource_Builder(t *testing.T) {
	msg := api.NewTestMessage("Book")
	pat := api.ResourcePattern{
		*api.NewPathSegmentBuilder().WithLiteral("books").Build(),
	}
	res := api.NewResourceBuilder().
		WithType("library.googleapis.com/Book").
		WithPatterns(pat).
		WithSingular("book").
		WithPlural("books").
		WithSelf(msg).
		Build()
	res.SetAnnotations("res-codec")

	if res.Type() != "library.googleapis.com/Book" {
		t.Errorf("res.Type() = %q, want %q", res.Type(), "library.googleapis.com/Book")
	}
	if len(res.Patterns()) != 1 {
		t.Fatalf("len(res.Patterns()) = %d, want 1", len(res.Patterns()))
	}
	if res.Singular() != "book" {
		t.Errorf("res.Singular() = %q, want %q", res.Singular(), "book")
	}
	if res.Plural() != "books" {
		t.Errorf("res.Plural() = %q, want %q", res.Plural(), "books")
	}
	if res.Self() != msg {
		t.Errorf("res.Self() = %v, want %v", res.Self(), msg)
	}
	if res.Codec() != "res-codec" {
		t.Errorf("res.Codec() = %v, want %q", res.Codec(), "res-codec")
	}
}

func TestEnumValue_Builder(t *testing.T) {
	enum := api.NewTestEnum("Status")
	ev := api.NewEnumValueBuilder().
		WithName("ACTIVE").
		WithID(".test.Status.ACTIVE").
		WithDocumentation("Active status").
		WithDeprecated(true).
		WithNumber(1).
		WithParent(enum).
		Build()
	ev.SetAnnotations("ev-codec")

	if ev.Name() != "ACTIVE" {
		t.Errorf("ev.Name() = %q, want %q", ev.Name(), "ACTIVE")
	}
	if ev.ID() != ".test.Status.ACTIVE" {
		t.Errorf("ev.ID() = %q, want %q", ev.ID(), ".test.Status.ACTIVE")
	}
	if ev.Documentation() != "Active status" {
		t.Errorf("ev.Documentation() = %q, want %q", ev.Documentation(), "Active status")
	}
	if !ev.Deprecated() {
		t.Errorf("ev.Deprecated() = false, want true")
	}
	if ev.Number() != 1 {
		t.Errorf("ev.Number() = %d, want 1", ev.Number())
	}
	if ev.Parent() != enum {
		t.Errorf("ev.Parent() = %v, want %v", ev.Parent(), enum)
	}
	if ev.Codec() != "ev-codec" {
		t.Errorf("ev.Codec() = %v, want %q", ev.Codec(), "ev-codec")
	}

	sv := api.NewSampleValue(ev, 2)
	if sv.EnumValue() != ev {
		t.Errorf("sv.EnumValue() = %v, want %v", sv.EnumValue(), ev)
	}
	if sv.Index() != 2 {
		t.Errorf("sv.Index() = %d, want 2", sv.Index())
	}
}

func TestEnum_Builder(t *testing.T) {
	parent := api.NewTestMessage("Parent")
	ev := api.NewEnumValueBuilder().
		WithName("ACTIVE").
		WithID(".test.Parent.Status.ACTIVE").
		WithNumber(1).
		Build()
	sv := api.NewSampleValue(ev, 0)
	enum := api.NewEnumBuilder().
		WithName("Status").
		WithID(".test.Parent.Status").
		WithDocumentation("Status enum").
		WithDeprecated(true).
		WithValues(ev).
		WithUniqueNumberValues(ev).
		WithValuesForExamples(sv).
		WithParent(parent).
		WithPackage("test").
		Build()
	enum.SetAnnotations("enum-codec")

	if enum.Name() != "Status" {
		t.Errorf("enum.Name() = %q, want %q", enum.Name(), "Status")
	}
	if enum.ID() != ".test.Parent.Status" {
		t.Errorf("enum.ID() = %q, want %q", enum.ID(), ".test.Parent.Status")
	}
	if enum.Documentation() != "Status enum" {
		t.Errorf("enum.Documentation() = %q, want %q", enum.Documentation(), "Status enum")
	}
	if !enum.Deprecated() {
		t.Errorf("enum.Deprecated() = false, want true")
	}
	if len(enum.Values()) != 1 || enum.Values()[0] != ev {
		t.Errorf("enum.Values() = %v, want [%v]", enum.Values(), ev)
	}
	if ev.Parent() != enum {
		t.Errorf("ev.Parent() = %v, want %v", ev.Parent(), enum)
	}
	if len(enum.UniqueNumberValues()) != 1 || enum.UniqueNumberValues()[0] != ev {
		t.Errorf("enum.UniqueNumberValues() = %v, want [%v]", enum.UniqueNumberValues(), ev)
	}
	if len(enum.ValuesForExamples()) != 1 || enum.ValuesForExamples()[0] != sv {
		t.Errorf("enum.ValuesForExamples() = %v, want [%v]", enum.ValuesForExamples(), sv)
	}
	if enum.Parent() != parent {
		t.Errorf("enum.Parent() = %v, want %v", enum.Parent(), parent)
	}
	if enum.Package() != "test" {
		t.Errorf("enum.Package() = %q, want %q", enum.Package(), "test")
	}
	if enum.Codec() != "enum-codec" {
		t.Errorf("enum.Codec() = %v, want %q", enum.Codec(), "enum-codec")
	}
}

func TestField_Builder(t *testing.T) {
	parent := api.NewTestMessage("Parent")
	msgType := api.NewTestMessage("Child")
	enumType := api.NewTestEnum("Status")
	group := api.NewTestOneOf("choice")
	rr := api.NewResourceReferenceBuilder().WithType("test.googleapis.com/Book").Build()
	rnp := api.NewTestResourceNamePattern(api.NewResourceNameSegment("books", "book"))

	f := api.NewFieldBuilder().
		WithName("my_field").
		WithID(".test.Parent.my_field").
		WithDocumentation("Field docs").
		WithJSONName("myField").
		WithDeprecated(true).
		WithTypez(api.TypezMessage).
		WithTypezID(".test.Child").
		WithOptional(true).
		WithProto2Optional(true).
		WithRepeated(true).
		WithMap(true).
		WithIsOneOf(true).
		WithAutoPopulated(true).
		WithSkipProtoConversion(true).
		WithRecursive(true).
		WithBehavior(api.FieldBehaviorRequired, api.FieldBehaviorOutputOnly).
		WithGroup(group).
		WithMessageType(msgType).
		WithEnumType(enumType).
		WithParent(parent).
		WithResourceReference(rr).
		WithResourceNamePattern(rnp).
		Build()
	f.SetAnnotations("field-codec")

	if f.Name() != "my_field" {
		t.Errorf("f.Name() = %q, want %q", f.Name(), "my_field")
	}
	if f.ID() != ".test.Parent.my_field" {
		t.Errorf("f.ID() = %q, want %q", f.ID(), ".test.Parent.my_field")
	}
	if f.Documentation() != "Field docs" {
		t.Errorf("f.Documentation() = %q, want %q", f.Documentation(), "Field docs")
	}
	if f.JSONName() != "myField" {
		t.Errorf("f.JSONName() = %q, want %q", f.JSONName(), "myField")
	}
	if !f.Deprecated() {
		t.Errorf("f.Deprecated() = false, want true")
	}
	if f.Typez() != api.TypezMessage {
		t.Errorf("f.Typez() = %v, want %v", f.Typez(), api.TypezMessage)
	}
	if f.TypezID() != ".test.Child" {
		t.Errorf("f.TypezID() = %q, want %q", f.TypezID(), ".test.Child")
	}
	if !f.Optional() {
		t.Errorf("f.Optional() = false, want true")
	}
	if !f.Proto2Optional() {
		t.Errorf("f.Proto2Optional() = false, want true")
	}
	if !f.Repeated() {
		t.Errorf("f.Repeated() = false, want true")
	}
	if !f.Map() {
		t.Errorf("f.Map() = false, want true")
	}
	if !f.IsOneOf() {
		t.Errorf("f.IsOneOf() = false, want true")
	}
	if !f.AutoPopulated() {
		t.Errorf("f.AutoPopulated() = false, want true")
	}
	if !f.SkipProtoConversion() {
		t.Errorf("f.SkipProtoConversion() = false, want true")
	}
	if !f.Recursive() {
		t.Errorf("f.Recursive() = false, want true")
	}
	if len(f.Behavior()) != 2 {
		t.Errorf("f.Behavior() = %v, want 2 elements", f.Behavior())
	}
	if f.Group() != group {
		t.Errorf("f.Group() = %v, want %v", f.Group(), group)
	}
	if f.MessageType() != msgType {
		t.Errorf("f.MessageType() = %v, want %v", f.MessageType(), msgType)
	}
	if f.EnumType() != enumType {
		t.Errorf("f.EnumType() = %v, want %v", f.EnumType(), enumType)
	}
	if f.Parent() != parent {
		t.Errorf("f.Parent() = %v, want %v", f.Parent(), parent)
	}
	if f.ResourceReference() != rr {
		t.Errorf("f.ResourceReference() = %v, want %v", f.ResourceReference(), rr)
	}
	if f.ResourceNamePattern() != rnp {
		t.Errorf("f.ResourceNamePattern() = %v, want %v", f.ResourceNamePattern(), rnp)
	}
	if f.Codec() != "field-codec" {
		t.Errorf("f.Codec() = %v, want %q", f.Codec(), "field-codec")
	}

	cloned := api.NewFieldBuilderFrom(f).WithID(".synthetic" + f.ID()).Build()
	if cloned.ID() != ".synthetic.test.Parent.my_field" {
		t.Errorf("cloned.ID() = %q, want %q", cloned.ID(), ".synthetic.test.Parent.my_field")
	}
	if cloned.Name() != f.Name() {
		t.Errorf("cloned.Name() = %q, want %q", cloned.Name(), f.Name())
	}

	f2 := api.NewFieldBuilder().WithName("other").WithID(".test.Msg.other").Build()
	if api.MarkSkipProtoConversionIfAllowed(f2, ".test.Msg.other", []string{".test.Msg.unrelated"}) {
		t.Errorf("MarkSkipProtoConversionIfAllowed returned true for unrelated field")
	}
	if !api.MarkSkipProtoConversionIfAllowed(f2, ".test.Msg.other", []string{"test.Msg.other"}) {
		t.Errorf("MarkSkipProtoConversionIfAllowed returned false for matching field")
	}
	if !f2.SkipProtoConversion() {
		t.Errorf("f2.SkipProtoConversion() = false, want true")
	}
}

func TestMessage_Builder(t *testing.T) {
	parent := api.NewTestMessage("Parent")
	f1 := api.NewFieldBuilder().WithName("f1").Build()
	f2 := api.NewFieldBuilder().WithName("f2").Build()
	fSkipped := api.NewFieldBuilder().WithName("skipped").Build()
	e1 := api.NewEnumBuilder().WithName("E1").Build()
	e2 := api.NewEnumBuilder().WithName("E2").Build()
	m1 := api.NewMessageBuilder().WithName("M1").Build()
	m2 := api.NewMessageBuilder().WithName("M2").Build()
	o1 := api.NewOneOfBuilder().WithName("o1").Build()
	o2 := api.NewOneOfBuilder().WithName("o2").Build()
	res := api.NewResourceBuilder().WithType("test.googleapis.com/Book").Build()
	pag := api.NewPaginationInfo(f1, f2)

	msg := api.NewMessageBuilder().
		WithName("Book").
		WithID(".test.Parent.Book").
		WithDocumentation("Book docs").
		WithDeprecated(true).
		WithFields(f1).
		AddField(f2).
		WithSkippedFields(fSkipped).
		WithEnums(e1).
		AddEnum(e2).
		WithMessages(m1).
		AddMessage(m2).
		WithOneOfs(o1).
		AddOneOf(o2).
		WithParent(parent).
		WithPackage("test").
		WithSyntheticRequest(true).
		WithServicePlaceholder(true).
		WithPagination(pag).
		WithResource(res).
		Build()
	msg.SetAnnotations("msg-codec")

	if msg.Name() != "Book" {
		t.Errorf("msg.Name() = %q, want Book", msg.Name())
	}
	if msg.ID() != ".test.Parent.Book" {
		t.Errorf("msg.ID() = %q, want .test.Parent.Book", msg.ID())
	}
	if msg.Documentation() != "Book docs" {
		t.Errorf("msg.Documentation() = %q, want Book docs", msg.Documentation())
	}
	if !msg.Deprecated() {
		t.Errorf("msg.Deprecated() = false, want true")
	}
	if len(msg.Fields()) != 2 || msg.Fields()[0] != f1 || msg.Fields()[1] != f2 {
		t.Errorf("msg.Fields() = %v, want [f1, f2]", msg.Fields())
	}
	if len(msg.SkippedFields()) != 1 || msg.SkippedFields()[0] != fSkipped {
		t.Errorf("msg.SkippedFields() = %v, want [fSkipped]", msg.SkippedFields())
	}
	if len(msg.Enums()) != 2 || msg.Enums()[0] != e1 || msg.Enums()[1] != e2 {
		t.Errorf("msg.Enums() = %v, want [e1, e2]", msg.Enums())
	}
	if len(msg.Messages()) != 2 || msg.Messages()[0] != m1 || msg.Messages()[1] != m2 {
		t.Errorf("msg.Messages() = %v, want [m1, m2]", msg.Messages())
	}
	if len(msg.OneOfs()) != 2 || msg.OneOfs()[0] != o1 || msg.OneOfs()[1] != o2 {
		t.Errorf("msg.OneOfs() = %v, want [o1, o2]", msg.OneOfs())
	}
	if msg.Parent() != parent {
		t.Errorf("msg.Parent() = %v, want %v", msg.Parent(), parent)
	}
	if msg.Package() != "test" {
		t.Errorf("msg.Package() = %q, want test", msg.Package())
	}
	if !msg.SyntheticRequest() {
		t.Errorf("msg.SyntheticRequest() = false, want true")
	}
	if !msg.ServicePlaceholder() {
		t.Errorf("msg.ServicePlaceholder() = false, want true")
	}
	if msg.Pagination() != pag {
		t.Errorf("msg.Pagination() = %v, want %v", msg.Pagination(), pag)
	}
	if msg.Resource() != res {
		t.Errorf("msg.Resource() = %v, want %v", msg.Resource(), res)
	}
	if msg.Codec() != "msg-codec" {
		t.Errorf("msg.Codec() = %v, want msg-codec", msg.Codec())
	}

	cloned := api.NewMessageBuilderFrom(msg).WithName("ClonedBook").Build()
	if cloned.Name() != "ClonedBook" {
		t.Errorf("cloned.Name() = %q, want ClonedBook", cloned.Name())
	}
	if cloned.ID() != msg.ID() {
		t.Errorf("cloned.ID() = %q, want %q", cloned.ID(), msg.ID())
	}
}
