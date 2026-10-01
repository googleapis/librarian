// Copyright 2024 Google LLC
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

package parser

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/api/apitest"
	"github.com/googleapis/librarian/internal/sources"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/types/known/apipb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestProtobuf_Info(t *testing.T) {
	requireProtoc(t)
	sc := sample.ServiceConfig()
	got, err := makeAPIForProtobuf(sc, newTestCodeGeneratorRequest(t, "scalar.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	if got.Name != "secretmanager" {
		t.Errorf("want = %q; got = %q", "secretmanager", got.Name)
	}
	if got.Title != sc.Title {
		t.Errorf("want = %q; got = %q", sc.Title, got.Title)
	}
	if diff := cmp.Diff(sc.Documentation.Summary, got.Description); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestProtobuf_PartialInfo(t *testing.T) {
	requireProtoc(t)
	serviceConfig := &serviceconfig.Service{
		Name:  "secretmanager.googleapis.com",
		Title: "Secret Manager API",
	}

	got, err := makeAPIForProtobuf(serviceConfig, newTestCodeGeneratorRequest(t, "scalar.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	want := api.NewTestAPI(nil, nil, nil).
		WithName("secretmanager").
		WithPackageName("test").
		WithTitle("Secret Manager API").
		WithDescription("")
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(api.API{}, "Services", "Messages", "Enums"), cmpopts.IgnoreUnexported(api.API{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func newTestProtobufFakeMessage(fields ...*api.Field) *api.Message {
	m := api.NewTestMessage("Fake").
		WithPackage("test").
		WithID(".test.Fake").
		WithDocumentation("A test message.").
		WithFields(fields...)
	for _, f := range m.Fields {
		f.Parent = nil
	}
	return m
}

func TestProtobuf_Scalar(t *testing.T) {
	requireProtoc(t)
	test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "scalar.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	message := test.Message(".test.Fake")
	if message == nil {
		t.Fatalf("Cannot find message %s in API State", ".test.Fake")
	}
	apitest.CheckMessage(t, message, newTestProtobufFakeMessage(
		api.NewTestField("f_double").
			WithDocumentation("A singular field tag = 1").
			WithType(api.TypezDouble),
		api.NewTestField("f_float").
			WithDocumentation("A singular field tag = 2").
			WithType(api.TypezFloat),
		api.NewTestField("f_int64").
			WithDocumentation("A singular field tag = 3").
			WithType(api.TypezInt64),
		api.NewTestField("f_uint64").
			WithDocumentation("A singular field tag = 4").
			WithType(api.TypezUint64),
		api.NewTestField("f_int32").
			WithDocumentation("A singular field tag = 5").
			WithType(api.TypezInt32),
		api.NewTestField("f_fixed64").
			WithDocumentation("A singular field tag = 6").
			WithType(api.TypezFixed64),
		api.NewTestField("f_fixed32").
			WithDocumentation("A singular field tag = 7").
			WithType(api.TypezFixed32),
		api.NewTestField("f_bool").
			WithDocumentation("A singular field tag = 8").
			WithType(api.TypezBool),
		api.NewTestField("f_string").
			WithDocumentation("A singular field tag = 9").
			WithType(api.TypezString),
		api.NewTestField("f_bytes").
			WithDocumentation("A singular field tag = 12").
			WithType(api.TypezBytes),
		api.NewTestField("f_uint32").
			WithDocumentation("A singular field tag = 13").
			WithType(api.TypezUint32),
		api.NewTestField("f_sfixed32").
			WithDocumentation("A singular field tag = 15").
			WithType(api.TypezSfixed32),
		api.NewTestField("f_sfixed64").
			WithDocumentation("A singular field tag = 16").
			WithType(api.TypezSfixed64),
		api.NewTestField("f_sint32").
			WithDocumentation("A singular field tag = 17").
			WithType(api.TypezSint32),
		api.NewTestField("f_sint64").
			WithDocumentation("A singular field tag = 18").
			WithType(api.TypezSint64),
	))
}

func TestProtobuf_ScalarArray(t *testing.T) {
	requireProtoc(t)
	test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "scalar_array.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	message := test.Message(".test.Fake")
	if message == nil {
		t.Fatalf("Cannot find message %s in API State", ".test.Fake")
	}
	apitest.CheckMessage(t, message, newTestProtobufFakeMessage(
		api.NewTestField("f_double").
			WithDocumentation("A repeated field tag = 1").
			WithType(api.TypezDouble).
			WithRepeated(),
		api.NewTestField("f_int64").
			WithDocumentation("A repeated field tag = 3").
			WithType(api.TypezInt64).
			WithRepeated(),
		api.NewTestField("f_string").
			WithDocumentation("A repeated field tag = 9").
			WithType(api.TypezString).
			WithRepeated(),
		api.NewTestField("f_bytes").
			WithDocumentation("A repeated field tag = 12").
			WithType(api.TypezBytes).
			WithRepeated(),
	))
}

func TestProtobuf_ScalarOptional(t *testing.T) {
	requireProtoc(t)
	test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "scalar_optional.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	message := test.Message(".test.Fake")
	if message == nil {
		t.Fatalf("Cannot find message %s in API", "Fake")
	}
	apitest.CheckMessage(t, message, newTestProtobufFakeMessage(
		api.NewTestField("f_double").
			WithDocumentation("An optional field tag = 1").
			WithType(api.TypezDouble).
			WithOptional(),
		api.NewTestField("f_int64").
			WithDocumentation("An optional field tag = 3").
			WithType(api.TypezInt64).
			WithOptional(),
		api.NewTestField("f_string").
			WithDocumentation("An optional field tag = 9").
			WithType(api.TypezString).
			WithOptional(),
		api.NewTestField("f_bytes").
			WithDocumentation("An optional field tag = 12").
			WithType(api.TypezBytes).
			WithOptional(),
	))
}

func TestProtobuf_SkipExternalMessages(t *testing.T) {
	requireProtoc(t)
	test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "with_import.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	// Both `ImportedMessage` and `LocalMessage` should be in the index:
	if test.Message(".away.ImportedMessage") == nil {
		t.Fatalf("Cannot find message %s in API State", ".away.ImportedMessage")
	}
	message := test.Message(".test.LocalMessage")
	if message == nil {
		t.Fatalf("Cannot find message %s in API State", ".test.LocalMessage")
	}
	want := api.NewTestMessage("LocalMessage").
		WithPackage("test").
		WithDocumentation("This is a local message, it should be generated.").
		WithFields(
			api.NewTestField("payload").
				WithDocumentation("This field uses an imported message.").
				WithType(api.TypezMessage).
				WithTypezID(".away.ImportedMessage").
				WithOptional(),
			api.NewTestField("value").
				WithDocumentation("This field uses an imported enum.").
				WithType(api.TypezEnum).
				WithTypezID(".away.ImportedEnum"),
		)
	for _, f := range want.Fields {
		f.Parent = nil
	}
	apitest.CheckMessage(t, message, want)
	// Only `LocalMessage` should be found in the messages list:
	for _, msg := range test.Messages {
		if msg.ID == ".test.ImportedMessage" {
			t.Errorf("imported messages should not be in message list %v", msg)
		}
	}
}

func TestProtobuf_SkipExternaEnums(t *testing.T) {
	requireProtoc(t)
	test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "with_import.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	// Both `ImportedEnum` and `LocalEnum` should be in the index:
	if test.Enum(".away.ImportedEnum") == nil {
		t.Fatalf("Cannot find enum %s in API State", ".away.ImportedEnum")
	}
	enum := test.Enum(".test.LocalEnum")
	if enum == nil {
		t.Fatalf("Cannot find enum %s in API State", ".test.LocalEnum")
	}
	wantEnum := api.NewTestEnum("LocalEnum").
		WithPackage("test").
		WithDocumentation("This is a local enum, it should be generated.").
		WithValues(
			api.NewTestEnumValue("RED", 0),
			api.NewTestEnumValue("WHITE", 1),
			api.NewTestEnumValue("BLUE", 2),
		)
	for _, v := range wantEnum.Values {
		v.ID = ""
	}
	apitest.CheckEnum(t, *enum, *wantEnum)
	// Only `LocalMessage` should be found in the messages list:
	for _, msg := range test.Messages {
		if msg.ID == ".test.ImportedMessage" {
			t.Errorf("imported messages should not be in message list %v", msg)
		}
	}
}

func TestProtobuf_Comments(t *testing.T) {
	requireProtoc(t)
	test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "comments.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	message := test.Message(".test.Request")
	if message == nil {
		t.Fatalf("Cannot find message %s in API State", ".test.Request")
	}
	wantRequest := api.NewTestMessage("Request").
		WithPackage("test").
		WithDocumentation("A test message.\n\nWith even more of a description.\nMaybe in more than one line.\nAnd some markdown:\n- An item\n  - A nested item\n- Another item").
		WithFields(
			api.NewTestField("parent").
				WithDocumentation("A field.\n\nWith a longer description.").
				WithType(api.TypezString),
		)
	for _, f := range wantRequest.Fields {
		f.Parent = nil
	}
	apitest.CheckMessage(t, message, wantRequest)

	message = test.Message(".test.Response.Nested")
	if message == nil {
		t.Fatalf("Cannot find message %s in API State", ".test.Response.nested")
	}
	wantNested := api.NewTestMessage("Nested").
		WithPackage("test").
		WithID(".test.Response.Nested").
		WithDocumentation("A nested message.\n\n- Item 1\n  Item 1 continued").
		WithFields(
			api.NewTestField("path").
				WithDocumentation("Field in a nested message.\n\n* Bullet 1\n  Bullet 1 continued\n* Bullet 2\n  Bullet 2 continued").
				WithType(api.TypezString),
		)
	for _, f := range wantNested.Fields {
		f.Parent = nil
	}
	apitest.CheckMessage(t, message, wantNested)

	e := test.Enum(".test.Response.Status")
	if e == nil {
		t.Fatalf("Cannot find enum %s in API State", ".test.Response.Status")
	}
	wantStatus := api.NewTestEnum("Status").
		WithPackage("test").
		WithID(".test.Response.Status").
		WithDocumentation("Some enum.\n\nLine 1.\nLine 2.").
		WithValues(
			api.NewTestEnumValue("NOT_READY", 0).
				WithDocumentation("The first enum value description.\n\nValue Line 1.\nValue Line 2."),
			api.NewTestEnumValue("READY", 1).
				WithDocumentation("The second enum value description."),
		)
	for _, v := range wantStatus.Values {
		v.ID = ""
	}
	apitest.CheckEnum(t, *e, *wantStatus)

	service := test.Service(".test.Service")
	if service == nil {
		t.Fatalf("Cannot find service %s in API State", ".test.Service")
	}
	createMethod := api.NewTestMethod("Create").
		WithID(".test.Service.Create").
		WithDocumentation("Some RPC.\n\nIt does not do much.").
		WithVerb("POST").
		WithPathTemplate(
			(&api.PathTemplate{}).
				WithLiteral("v1").
				WithVariable(api.NewPathVariable("parent").
					WithLiteral("projects").
					WithMatch()).
				WithLiteral("foos"),
		).
		WithQueryParameters(map[string]bool{}).
		WithBodyFieldPath("*")
	createMethod.SourceServiceID = ".test.Service"
	createMethod.InputTypeID = ".test.Request"
	createMethod.OutputTypeID = ".test.Response"

	wantService := api.NewTestService("Service").
		WithPackage("test").
		WithDocumentation("A service.\n\nWith a longer service description.").
		WithDefaultHost("test.googleapis.com").
		WithMethods(createMethod)
	createMethod.Service = nil
	apitest.CheckService(t, service, wantService)
}

func TestProtobuf_UniqueEnumValues(t *testing.T) {
	requireProtoc(t)
	test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "enum_values.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	withAlias := test.Enum(".test.WithAlias")
	if withAlias == nil {
		t.Fatalf("Cannot find enum %s in API State", ".test.WithAlias")
	}
	fullList := []*api.EnumValue{
		api.NewTestEnumValue("X_UNSPECIFIED", 0).WithID(""),
		api.NewTestEnumValue("LONG_NAME_VALUE", 2).WithID(""),
		api.NewTestEnumValue("V2", 2).WithID(""),
		api.NewTestEnumValue("bad_style", 3).WithID(""),
		api.NewTestEnumValue("FOLLOWS_STYLE", 3).WithID(""),
	}

	uniqueList := []*api.EnumValue{
		api.NewTestEnumValue("X_UNSPECIFIED", 0).WithID(""),
		api.NewTestEnumValue("V2", 2).WithID(""),
		api.NewTestEnumValue("FOLLOWS_STYLE", 3).WithID(""),
	}

	less := func(a, b *api.EnumValue) bool { return a.Name < b.Name }
	if diff := cmp.Diff(fullList, withAlias.Values, cmpopts.SortSlices(less), cmpopts.IgnoreFields(api.EnumValue{}, "Parent")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(uniqueList, withAlias.UniqueNumberValues, cmpopts.SortSlices(less), cmpopts.IgnoreFields(api.EnumValue{}, "Parent")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestProtobuf_OneOfs(t *testing.T) {
	requireProtoc(t)
	test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "oneofs.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	message := test.Message(".test.Fake")
	if message == nil {
		t.Fatalf("Cannot find message %s in API State", ".test.Request")
	}
	choice := api.NewTestOneOf("choice").
		WithFields(
			api.NewTestField("field_one").
				WithDocumentation("A string choice").
				WithType(api.TypezString),
			api.NewTestField("field_two").
				WithDocumentation("An int choice").
				WithType(api.TypezInt64),
			api.NewTestField("field_five").
				WithDocumentation("A message choice").
				WithType(api.TypezMessage).
				WithTypezID(".test.Inner"),
		)
	for _, f := range choice.Fields {
		f.Group = nil
	}
	wantFake := api.NewTestMessage("Fake").
		WithPackage("test").
		WithDocumentation("A test message.").
		WithOneOfs(choice).
		WithFields(
			api.NewTestField("field_three").
				WithDocumentation("Optional is oneof in proto").
				WithType(api.TypezString).
				WithOptional(),
			api.NewTestField("field_four").
				WithDocumentation("A normal field").
				WithType(api.TypezInt32),
		)
	apitest.CheckMessage(t, message, wantFake)
}

func TestProtobuf_ObjectFields(t *testing.T) {
	requireProtoc(t)
	test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "object_fields.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	message := test.Message(".test.Fake")
	if message == nil {
		t.Fatalf("Cannot find message %s in API State", ".test.Fake")
	}
	wantObjectMessage := api.NewTestMessage("Fake").
		WithPackage("test").
		WithFields(
			api.NewTestField("singular_object").
				WithType(api.TypezMessage).
				WithTypezID(".test.Other").
				WithOptional(),
			api.NewTestField("repeated_object").
				WithType(api.TypezMessage).
				WithTypezID(".test.Other").
				WithRepeated(),
		)
	for _, f := range wantObjectMessage.Fields {
		f.Parent = nil
	}
	apitest.CheckMessage(t, message, wantObjectMessage)
}

func TestProtobuf_WellKnownTypeFields(t *testing.T) {
	requireProtoc(t)
	test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "wkt_fields.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	message := test.Message(".test.Fake")
	if message == nil {
		t.Fatalf("Cannot find message %s in API State", ".test.Fake")
	}
	want := api.NewTestMessage("Fake").
		WithPackage("test").
		WithFields(
			api.NewTestField("field_mask").
				WithType(api.TypezMessage).
				WithTypezID(".google.protobuf.FieldMask").
				WithOptional(),
			api.NewTestField("timestamp").
				WithType(api.TypezMessage).
				WithTypezID(".google.protobuf.Timestamp").
				WithOptional(),
			api.NewTestField("any").
				WithType(api.TypezMessage).
				WithTypezID(".google.protobuf.Any").
				WithOptional(),
			api.NewTestField("repeated_field_mask").
				WithType(api.TypezMessage).
				WithTypezID(".google.protobuf.FieldMask").
				WithRepeated(),
			api.NewTestField("repeated_timestamp").
				WithType(api.TypezMessage).
				WithTypezID(".google.protobuf.Timestamp").
				WithRepeated(),
			api.NewTestField("repeated_any").
				WithType(api.TypezMessage).
				WithTypezID(".google.protobuf.Any").
				WithRepeated(),
		)
	for _, f := range want.Fields {
		f.Parent = nil
	}
	apitest.CheckMessage(t, message, want)
}

func TestProtobuf_JsonName(t *testing.T) {
	requireProtoc(t)
	test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "json_name.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	message := test.Message(".test.Request")
	if message == nil {
		t.Fatalf("Cannot find message %s in API State", ".test.Request")
	}
	wantRequest := api.NewTestMessage("Request").
		WithPackage("test").
		WithDocumentation("A test message.").
		WithFields(
			api.NewTestField("parent").
				WithType(api.TypezString),
			api.NewTestField("public_key").
				WithJSONName("public_key").
				WithType(api.TypezString),
			api.NewTestField("read_time").
				WithType(api.TypezInt32),
		)
	for _, f := range wantRequest.Fields {
		f.Parent = nil
	}
	apitest.CheckMessage(t, message, wantRequest)
}

func TestProtobuf_MapFields(t *testing.T) {
	requireProtoc(t)
	test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "map_fields.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	message := test.Message(".test.Fake")
	if message == nil {
		t.Fatalf("Cannot find message %s in API State", ".test.Fake")
	}
	want := api.NewTestMessage("Fake").
		WithPackage("test").
		WithFields(
			api.NewTestField("singular_map").
				WithMap().
				WithType(api.TypezMessage).
				WithTypezID(".test.Fake.SingularMapEntry"),
			api.NewTestField("enum_value").
				WithMap().
				WithType(api.TypezMessage).
				WithTypezID(".test.Fake.EnumValueEntry"),
		)
	for _, f := range want.Fields {
		f.Parent = nil
	}
	apitest.CheckMessage(t, message, want)

	if diff := cmp.Diff([]*api.Message(nil), message.Messages); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	message = test.Message(".test.Fake.SingularMapEntry")
	if message == nil {
		t.Fatalf("Cannot find message %s in API State", ".test.Fake.SingularMapEntry")
	}
	wantSingularMapEntry := api.NewTestMessage("SingularMapEntry").
		WithPackage("test").
		WithID(".test.Fake.SingularMapEntry").
		WithFields(
			api.NewTestField("key").
				WithType(api.TypezString),
			api.NewTestField("value").
				WithType(api.TypezInt32),
		)
	wantSingularMapEntry.IsMap = true
	for _, f := range wantSingularMapEntry.Fields {
		f.Parent = nil
	}
	apitest.CheckMessage(t, message, wantSingularMapEntry)

	message = test.Message(".test.Fake.EnumValueEntry")
	if message == nil {
		t.Fatalf("Cannot find message %s in API State", ".test.Fake.EnumValueEntry")
	}
	wantEnumValueEntry := api.NewTestMessage("EnumValueEntry").
		WithPackage("test").
		WithID(".test.Fake.EnumValueEntry").
		WithFields(
			api.NewTestField("key").
				WithType(api.TypezString),
			api.NewTestField("value").
				WithType(api.TypezEnum).
				WithTypezID(".test.TestEnum"),
		)
	wantEnumValueEntry.IsMap = true
	for _, f := range wantEnumValueEntry.Fields {
		f.Parent = nil
	}
	apitest.CheckMessage(t, message, wantEnumValueEntry)
}

func TestProtobuf_Service(t *testing.T) {
	requireProtoc(t)
	test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "test_service.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	service := test.Service(".test.TestService")
	if service == nil {
		t.Fatalf("Cannot find service %s in API State", ".test.TestService")
	}
	getFoo := api.NewTestMethod("GetFoo").
		WithID(".test.TestService.GetFoo").
		WithDocumentation("Gets a Foo resource.").
		WithVerb("GET").
		WithPathTemplate(
			(&api.PathTemplate{}).
				WithLiteral("v1").
				WithVariable(api.NewPathVariable("name").
					WithLiteral("projects").
					WithMatch().
					WithLiteral("foos").
					WithMatch()),
		).
		WithQueryParameters(map[string]bool{})
	getFoo.SourceServiceID = ".test.TestService"
	getFoo.InputTypeID = ".test.GetFooRequest"
	getFoo.OutputTypeID = ".test.Foo"
	getFoo.PathInfo.BodyFieldPath = ""
	getFoo.Signatures = []*api.MethodSignature{{Names: []string{"name"}}}

	createFoo := api.NewTestMethod("CreateFoo").
		WithID(".test.TestService.CreateFoo").
		WithDocumentation("Creates a new Foo resource.").
		WithVerb("POST").
		WithPathTemplate(
			(&api.PathTemplate{}).
				WithLiteral("v1").
				WithVariable(api.NewPathVariable("parent").
					WithLiteral("projects").
					WithMatch()).
				WithLiteral("foos"),
		).
		WithQueryParameters(map[string]bool{"foo_id": true}).
		WithBodyFieldPath("foo")
	createFoo.SourceServiceID = ".test.TestService"
	createFoo.InputTypeID = ".test.CreateFooRequest"
	createFoo.OutputTypeID = ".test.Foo"
	createFoo.Signatures = []*api.MethodSignature{{Names: []string{"parent", "foo_id", "foo"}}}

	deleteFoo := api.NewTestMethod("DeleteFoo").
		WithID(".test.TestService.DeleteFoo").
		WithDocumentation("Deletes a Foo resource.").
		WithVerb("DELETE").
		WithPathTemplate(
			(&api.PathTemplate{}).
				WithLiteral("v1").
				WithVariable(api.NewPathVariable("name").
					WithLiteral("projects").
					WithMatch().
					WithLiteral("foos").
					WithMatch()),
		).
		WithQueryParameters(map[string]bool{}).
		ReturnEmpty()
	deleteFoo.SourceServiceID = ".test.TestService"
	deleteFoo.InputTypeID = ".test.DeleteFooRequest"
	deleteFoo.OutputTypeID = ".google.protobuf.Empty"

	uploadFoos := api.NewTestMethod("UploadFoos").
		WithID(".test.TestService.UploadFoos").
		WithDocumentation("A client-side streaming RPC.").
		WithClientSideStreaming()
	uploadFoos.SourceServiceID = ".test.TestService"
	uploadFoos.InputTypeID = ".test.CreateFooRequest"
	uploadFoos.OutputTypeID = ".test.Foo"
	uploadFoos.PathInfo = &api.PathInfo{}

	downloadFoos := api.NewTestMethod("DownloadFoos").
		WithID(".test.TestService.DownloadFoos").
		WithDocumentation("A server-side streaming RPC.").
		WithVerb("GET").
		WithPathTemplate(
			(&api.PathTemplate{}).
				WithLiteral("v1").
				WithVariable(api.NewPathVariable("name").
					WithLiteral("projects").
					WithMatch().
					WithLiteral("foos").
					WithMatch()).
				WithVerb("Download"),
		).
		WithQueryParameters(map[string]bool{}).
		WithServerSideStreaming()
	downloadFoos.SourceServiceID = ".test.TestService"
	downloadFoos.InputTypeID = ".test.GetFooRequest"
	downloadFoos.OutputTypeID = ".test.Foo"
	downloadFoos.PathInfo.BodyFieldPath = ""

	chatLike := api.NewTestMethod("ChatLike").
		WithID(".test.TestService.ChatLike").
		WithDocumentation("A bidi streaming RPC.").
		WithBidiStreaming()
	chatLike.SourceServiceID = ".test.TestService"
	chatLike.InputTypeID = ".test.Foo"
	chatLike.OutputTypeID = ".test.Foo"
	chatLike.PathInfo = &api.PathInfo{}

	wantService := api.NewTestService("TestService").
		WithPackage("test").
		WithDocumentation("A service to unit test the protobuf translator.").
		WithDefaultHost("test.googleapis.com").
		WithMethods(getFoo, createFoo, deleteFoo, uploadFoos, downloadFoos, chatLike)
	for _, m := range wantService.Methods {
		m.Service = nil
	}
	apitest.CheckService(t, service, wantService)
}

func TestProtobuf_QueryParameters(t *testing.T) {
	requireProtoc(t)
	test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "query_parameters.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	service := test.Service(".test.TestService")
	if service == nil {
		t.Fatalf("Cannot find service %s in API State", ".test.TestService")
	}
	createFoo := api.NewTestMethod("CreateFoo").
		WithID(".test.TestService.CreateFoo").
		WithDocumentation("Creates a new `Foo` resource. `Foo`s are containers for `Bar`s.\n\nShows how a `body: \"${field}\"` option works.").
		WithVerb("POST").
		WithPathTemplate(
			(&api.PathTemplate{}).
				WithLiteral("v1").
				WithVariable(api.NewPathVariable("parent").
					WithLiteral("projects").
					WithMatch()).
				WithLiteral("foos"),
		).
		WithQueryParameters(map[string]bool{"foo_id": true}).
		WithBodyFieldPath("bar")
	createFoo.SourceServiceID = ".test.TestService"
	createFoo.InputTypeID = ".test.CreateFooRequest"
	createFoo.OutputTypeID = ".test.Foo"
	createFoo.Signatures = []*api.MethodSignature{{Names: []string{"parent", "foo_id", "bar"}}}

	addBar := api.NewTestMethod("AddBar").
		WithID(".test.TestService.AddBar").
		WithDocumentation("Add a Bar resource.\n\nShows how a `body: \"*\"` option works.").
		WithVerb("POST").
		WithPathTemplate(
			(&api.PathTemplate{}).
				WithLiteral("v1").
				WithVariable(api.NewPathVariable("parent").
					WithLiteral("projects").
					WithMatch().
					WithLiteral("foos").
					WithMatch()).
				WithVerb("addFoo"),
		).
		WithQueryParameters(map[string]bool{}).
		WithBodyFieldPath("*")
	addBar.SourceServiceID = ".test.TestService"
	addBar.InputTypeID = ".test.AddBarRequest"
	addBar.OutputTypeID = ".test.Bar"
	addBar.Signatures = []*api.MethodSignature{{Names: []string{"parent", "payload"}}}

	wantService := api.NewTestService("TestService").
		WithPackage("test").
		WithDocumentation("A service to unit test the protobuf translator.").
		WithDefaultHost("test.googleapis.com").
		WithMethods(createFoo, addBar)
	for _, m := range wantService.Methods {
		m.Service = nil
	}
	apitest.CheckService(t, service, wantService)
}

func TestProtobuf_Enum(t *testing.T) {
	requireProtoc(t)
	test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "enum.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	e := test.Enum(".test.Code")
	if e == nil {
		t.Fatalf("Cannot find enum %s in API State", ".test.Code")
	}
	wantEnum := api.NewTestEnum("Code").
		WithPackage("test").
		WithDocumentation("An enum.").
		WithValues(
			api.NewTestEnumValue("OK", 0).
				WithDocumentation("Not an error; returned on success."),
			api.NewTestEnumValue("UNKNOWN", 1).
				WithDocumentation("Unknown error."),
		)
	for _, v := range wantEnum.Values {
		v.ID = ""
	}
	apitest.CheckEnum(t, *e, *wantEnum)
}

func TestProtobuf_TrimLeadingSpacesInDocumentation(t *testing.T) {
	input := ` In this example, in proto field could take one of the following values:

 * full_name for a violation in the full_name value
 * email_addresses[1].email for a violation in the email field of the
   first email_addresses message
 * email_addresses[3].type[2] for a violation in the second type
   value in the third email_addresses message.)`

	want := `In this example, in proto field could take one of the following values:

* full_name for a violation in the full_name value
* email_addresses[1].email for a violation in the email field of the
  first email_addresses message
* email_addresses[3].type[2] for a violation in the second type
  value in the third email_addresses message.)`

	got := trimLeadingSpacesInDocumentation(input)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func newTestPaginationMethod(name, inputType string, queryParams map[string]bool, pageTokenReq string) *api.Method {
	m := api.NewTestMethod(name).
		WithID(".test.TestService." + name).
		WithVerb("GET").
		WithPathTemplate(
			(&api.PathTemplate{}).
				WithLiteral("v1").
				WithVariable(api.NewPathVariable("parent").
					WithLiteral("projects").
					WithMatch()).
				WithLiteral("foos"),
		).
		WithQueryParameters(queryParams)
	m.SourceServiceID = ".test.TestService"
	m.InputTypeID = ".test." + inputType
	m.OutputTypeID = ".test.ListFooResponse"
	m.Signatures = []*api.MethodSignature{{Names: []string{"parent"}}}
	if pageTokenReq != "" {
		m.WithPagination(
			api.NewTestField("page_token").
				WithJSONName("pageToken").
				WithType(api.TypezString).
				WithBehavior(api.FieldBehaviorOptional),
		)
		m.Pagination.ID = fmt.Sprintf(".test.%s.page_token", pageTokenReq)
		m.IsList = false
	}
	return m
}

func TestProtobuf_Pagination(t *testing.T) {
	requireProtoc(t)
	test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "pagination.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	api.UpdateMethodPagination(nil, test)
	service := test.Service(".test.TestService")
	if service == nil {
		t.Fatalf("Cannot find service %s in API State", ".test.TestService")
	}
	missingNextPageToken := newTestPaginationMethod("ListFooMissingNextPageToken", "ListFooRequest", map[string]bool{"page_size": true, "page_token": true}, "")
	missingNextPageToken.OutputTypeID = ".test.ListFooMissingNextPageTokenResponse"

	missingRepeatedItem := newTestPaginationMethod("ListFooMissingRepeatedItemToken", "ListFooRequest", map[string]bool{"page_size": true, "page_token": true}, "")
	missingRepeatedItem.OutputTypeID = ".test.ListFooMissingRepeatedItemResponse"

	wantService := api.NewTestService("TestService").
		WithPackage("test").
		WithDefaultHost("test.googleapis.com").
		WithMethods(
			newTestPaginationMethod("ListFoo", "ListFooRequest", map[string]bool{"page_size": true, "page_token": true}, "ListFooRequest"),
			newTestPaginationMethod("ListFooWithMaxResultsInt32", "ListFooMaxResultsInt32Request", map[string]bool{"max_results": true, "page_token": true}, "ListFooMaxResultsInt32Request"),
			newTestPaginationMethod("ListFooWithMaxResultsUInt32", "ListFooMaxResultsUInt32Request", map[string]bool{"max_results": true, "page_token": true}, "ListFooMaxResultsUInt32Request"),
			newTestPaginationMethod("ListFooWithMaxResultsUInt32Value", "ListFooMaxResultsUInt32ValueRequest", map[string]bool{"max_results": true, "page_token": true}, "ListFooMaxResultsUInt32ValueRequest"),
			newTestPaginationMethod("ListFooWithMaxResultsInt32Value", "ListFooMaxResultsInt32ValueRequest", map[string]bool{"max_results": true, "page_token": true}, "ListFooMaxResultsInt32ValueRequest"),
			newTestPaginationMethod("ListFooWithMaxResultsIncorrectMessageType", "ListFooMaxResultIncorrectMessageTypeRequest", map[string]bool{"max_results": true, "page_token": true}, ""),
			missingNextPageToken,
			newTestPaginationMethod("ListFooMissingPageSize", "ListFooMissingPageSizeRequest", map[string]bool{"page_token": true}, ""),
			newTestPaginationMethod("ListFooMissingPageToken", "ListFooMissingPageTokenRequest", map[string]bool{"page_size": true}, ""),
			missingRepeatedItem,
		)
	for _, m := range wantService.Methods {
		m.Service = nil
	}
	apitest.CheckService(t, service, wantService)

	resp := test.Message(".test.ListFooResponse")
	if resp == nil {
		t.Errorf("missing message (ListFooResponse) in MessageByID index")
		return
	}
	nextPageToken := api.NewTestField("next_page_token").
		WithType(api.TypezString)
	foos := api.NewTestField("foos").
		WithType(api.TypezMessage).
		WithTypezID(".test.Foo").
		WithRepeated()
	wantResp := api.NewTestMessage("ListFooResponse").
		WithPackage("test").
		WithFields(
			api.NewTestField("total_size").
				WithType(api.TypezInt32),
		).
		WithPagination(nextPageToken, foos)
	for _, f := range wantResp.Fields {
		f.Parent = nil
	}
	apitest.CheckMessage(t, resp, wantResp)
}

func TestProtobuf_OperationInfo(t *testing.T) {
	requireProtoc(t)
	serviceConfig := &serviceconfig.Service{
		Name:  "test.googleapis.com",
		Title: "Test API",
		Documentation: &serviceconfig.Documentation{
			Summary:  "Used for testing generation.",
			Overview: "Test Overview",
			Rules: []*serviceconfig.DocumentationRule{
				{
					Selector:    "google.longrunning.Operations.GetOperation",
					Description: "Custom docs.",
				},
			},
		},
		Apis: []*apipb.Api{
			{
				Name: "google.longrunning.Operations",
			},
			{
				Name: "test.googleapis.com.TestService",
			},
		},
		Http: &annotations.Http{
			Rules: []*httpRule{
				{
					Selector: "google.longrunning.Operations.GetOperation",
					Pattern: &httpRuleGet{
						Get: "/v2/{name=operations/*}",
					},
					Body: "*",
				},
			},
		},
	}
	test, err := makeAPIForProtobuf(serviceConfig, newTestCodeGeneratorRequest(t, "test_operation_info.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	service := test.Service(".test.LroService")
	if service == nil {
		t.Fatalf("Cannot find service %s in API State", ".test.LroService")
	}
	createFoo := api.NewTestMethod("CreateFoo").
		WithID(".test.LroService.CreateFoo").
		WithDocumentation("Creates a new Foo resource.").
		WithVerb("POST").
		WithPathTemplate(
			(&api.PathTemplate{}).
				WithLiteral("v1").
				WithVariable(api.NewPathVariable("parent").
					WithLiteral("projects").
					WithMatch()).
				WithLiteral("foos"),
		).
		WithQueryParameters(map[string]bool{}).
		WithBodyFieldPath("foo").
		WithOperationInfo(&api.OperationInfo{
			MetadataTypeID: ".google.protobuf.Empty",
			ResponseTypeID: ".test.Foo",
		})
	createFoo.SourceServiceID = ".test.LroService"
	createFoo.InputTypeID = ".test.CreateFooRequest"
	createFoo.OutputTypeID = ".google.longrunning.Operation"
	createFoo.IsLRO = false

	createFooWithProgress := api.NewTestMethod("CreateFooWithProgress").
		WithID(".test.LroService.CreateFooWithProgress").
		WithDocumentation("Creates a new Foo resource.").
		WithVerb("POST").
		WithPathTemplate(
			(&api.PathTemplate{}).
				WithLiteral("v1").
				WithVariable(api.NewPathVariable("parent").
					WithLiteral("projects").
					WithMatch()).
				WithLiteral("foos"),
		).
		WithQueryParameters(map[string]bool{}).
		WithBodyFieldPath("foo").
		WithOperationInfo(&api.OperationInfo{
			MetadataTypeID: ".test.CreateMetadata",
			ResponseTypeID: ".test.Foo",
		})
	createFooWithProgress.SourceServiceID = ".test.LroService"
	createFooWithProgress.InputTypeID = ".test.CreateFooRequest"
	createFooWithProgress.OutputTypeID = ".google.longrunning.Operation"
	createFooWithProgress.IsLRO = false

	getOperation := api.NewTestMethod("GetOperation").
		WithID(".test.LroService.GetOperation").
		WithDocumentation("Custom docs.").
		WithVerb("GET").
		WithPathTemplate(
			(&api.PathTemplate{}).
				WithLiteral("v2").
				WithVariable(api.NewPathVariable("name").
					WithLiteral("operations").
					WithMatch()),
		).
		WithQueryParameters(map[string]bool{}).
		WithBodyFieldPath("*")
	getOperation.SourceServiceID = ".google.longrunning.Operations"
	getOperation.InputTypeID = ".google.longrunning.GetOperationRequest"
	getOperation.OutputTypeID = ".google.longrunning.Operation"
	getOperation.Signatures = []*api.MethodSignature{{Names: []string{"name"}}}

	wantService := api.NewTestService("LroService").
		WithPackage("test").
		WithDocumentation("A service to unit test the protobuf translator.").
		WithDefaultHost("test.googleapis.com").
		WithMethods(createFoo, createFooWithProgress, getOperation)
	for _, m := range wantService.Methods {
		m.Service = nil
	}
	apitest.CheckService(t, service, wantService)
}

func TestProtobuf_AutoPopulated(t *testing.T) {
	requireProtoc(t)
	serviceConfig := &serviceconfig.Service{
		Name:  "test.googleapis.com",
		Title: "Test API",
		Documentation: &serviceconfig.Documentation{
			Summary:  "Used for testing generation.",
			Overview: "Test Overview",
		},
		Apis: []*apipb.Api{
			{
				Name: "test.googleapis.com.TestService",
			},
		},
		Publishing: &annotations.Publishing{
			MethodSettings: []*annotations.MethodSettings{
				{
					Selector: "test.TestService.CreateFoo",
					AutoPopulatedFields: []string{
						"request_id",
						"request_id_optional",
						"request_id_with_field_behavior",
						// Intentionally add some fields that are not
						// auto-populated to test the other conditions.
						"not_request_id_bad_type",
						"not_request_id_required",
						"not_request_id_required_with_other_field_behavior",
						"not_request_id_missing_field_info",
						"not_request_id_missing_field_info_format",
						"not_request_id_bad_field_info_format",
					},
				},
			},
		},
	}
	test, err := makeAPIForProtobuf(serviceConfig, newTestCodeGeneratorRequest(t, "auto_populated.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	for _, service := range test.Services {
		if service.ID == ".google.longrunning.Operations" {
			t.Fatalf("Mixin %s should not be in list of services to generate", service.ID)
		}
	}
	message := test.Message(".test.CreateFooRequest")
	if message == nil {
		t.Fatalf("Cannot find message %s in API State", ".test.CreateFooRequest")
	}
	requestID := api.NewTestField("request_id").
		WithDocumentation("This is an auto-populated field. The remaining fields almost meet the\n" +
			"requirements to be auto-populated, but fail for the reasons implied by\n" +
			"their name.").
		WithType(api.TypezString).
		WithAutoPopulated()
	requestIDOptional := api.NewTestField("request_id_optional").
		WithType(api.TypezString).
		WithOptional().
		WithAutoPopulated()
	requestIDWithFieldBehavior := api.NewTestField("request_id_with_field_behavior").
		WithType(api.TypezString).
		WithAutoPopulated().
		WithBehavior(api.FieldBehaviorOptional, api.FieldBehaviorInputOnly)

	wantMessage := api.NewTestMessage("CreateFooRequest").
		WithPackage("test").
		WithDocumentation("A request to create a `Foo` resource.").
		WithFields(
			api.NewTestField("parent").
				WithDocumentation("Required. The resource name of the project.").
				WithType(api.TypezString).
				WithBehavior(api.FieldBehaviorRequired).
				WithResourceReference("cloudresourcemanager.googleapis.com/Project"),
			api.NewTestField("foo_id").
				WithDocumentation("Required. This must be unique within the project.").
				WithType(api.TypezString).
				WithBehavior(api.FieldBehaviorRequired),
			api.NewTestField("foo").
				WithDocumentation("Required. A [Foo][test.Foo] with initial field values.").
				WithType(api.TypezMessage).
				WithTypezID(".test.Foo").
				WithOptional().
				WithBehavior(api.FieldBehaviorRequired),
			requestID,
			requestIDOptional,
			requestIDWithFieldBehavior,
			api.NewTestField("not_request_id_bad_type").
				WithType(api.TypezBytes),
			api.NewTestField("not_request_id_required").
				WithType(api.TypezString).
				WithBehavior(api.FieldBehaviorRequired),
			api.NewTestField("not_request_id_required_with_other_field_behavior").
				WithType(api.TypezString).
				WithBehavior(api.FieldBehaviorInputOnly, api.FieldBehaviorRequired),
			api.NewTestField("not_request_id_missing_field_info").
				WithType(api.TypezString),
			api.NewTestField("not_request_id_missing_field_info_format").
				WithType(api.TypezString),
			api.NewTestField("not_request_id_bad_field_info_format").
				WithType(api.TypezString),
			api.NewTestField("not_request_id_missing_service_config").
				WithType(api.TypezString).
				WithAutoPopulated(),
		)
	for _, f := range wantMessage.Fields {
		f.Parent = nil
	}
	apitest.CheckMessage(t, message, wantMessage)

	method := test.Method(".test.TestService.CreateFoo")
	if method == nil {
		t.Fatalf("Cannot find method %s in API State", ".test.TestService.CreateFoo")
	}
	want := []*api.Field{requestID, requestIDOptional, requestIDWithFieldBehavior}
	if diff := cmp.Diff(want, method.AutoPopulated); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestProtobuf_Deprecated(t *testing.T) {
	requireProtoc(t)
	test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "deprecated.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	s := test.Service(".test.ServiceA")
	if s == nil {
		t.Fatalf("Cannot find %s in API State", ".test.ServiceA")
	}
	apitest.CheckService(t, s, api.NewTestService("ServiceA").
		WithPackage("test").
		WithDeprecated(true),
	)

	s = test.Service(".test.ServiceB")
	if s == nil {
		t.Fatalf("Cannot find %s in API State", ".test.ServiceB")
	}
	rpcA := api.NewTestMethod("RpcA").
		WithID(".test.ServiceB.RpcA").
		WithDeprecated(true)
	rpcA.InputTypeID = ".test.Request"
	rpcA.OutputTypeID = ".test.Response"
	rpcA.PathInfo = &api.PathInfo{}
	rpcA.SourceServiceID = ".test.ServiceB"

	wantServiceB := api.NewTestService("ServiceB").
		WithPackage("test").
		WithMethods(rpcA)
	rpcA.Service = nil
	apitest.CheckService(t, s, wantServiceB)

	m := test.Message(".test.Request")
	if m == nil {
		t.Fatalf("Cannot find %s in API State", ".test.Request")
	}
	wantRequest := api.NewTestMessage("Request").
		WithPackage("test").
		WithFields(
			api.NewTestField("name").
				WithType(api.TypezString),
			api.NewTestField("other").
				WithType(api.TypezString).
				WithDeprecated(true),
		)
	for _, f := range wantRequest.Fields {
		f.Parent = nil
	}
	apitest.CheckMessage(t, m, wantRequest)

	m = test.Message(".test.Response")
	if m == nil {
		t.Fatalf("Cannot find %s in API State", ".test.Response")
	}
	apitest.CheckMessage(t, m, api.NewTestMessage("Response").
		WithPackage("test").
		WithDeprecated(true),
	)

	e := test.Enum(".test.EnumA")
	if e == nil {
		t.Fatalf("Cannot find %s in API State", ".test.EnumA")
	}
	wantEnumA := api.NewTestEnum("EnumA").
		WithPackage("test").
		WithDeprecated(true).
		WithValues(
			api.NewTestEnumValue("ENUM_A_UNSPECIFIED", 0),
		)
	for _, v := range wantEnumA.Values {
		v.ID = ""
	}
	apitest.CheckEnum(t, *e, *wantEnumA)

	e = test.Enum(".test.EnumB")
	if e == nil {
		t.Fatalf("Cannot find %s in API State", ".test.EnumB")
	}
	wantEnumB := api.NewTestEnum("EnumB").
		WithPackage("test").
		WithValues(
			api.NewTestEnumValue("ENUM_B_UNSPECIFIED", 0),
			api.NewTestEnumValue("RED", 1).WithDeprecated(true),
			api.NewTestEnumValue("GREEN", 2),
			api.NewTestEnumValue("BLUE", 3),
		)
	for _, v := range wantEnumB.Values {
		v.ID = ""
	}
	apitest.CheckEnum(t, *e, *wantEnumB)
}

func TestProtobuf_ResourceAnnotations(t *testing.T) {
	requireProtoc(t)
	test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "resource_annotations.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}

	t.Run("API.ResourceDefinitions", func(t *testing.T) {
		// We expect 2 ResourceDefinitions: Shelf (file-level) and Book (message-level).
		if len(test.ResourceDefinitions) != 2 {
			t.Fatalf("Expected 2 ResourceDefinitions, got %d", len(test.ResourceDefinitions))
		}

		// Verify Shelf
		shelfResourceDef := api.NewTestResource("library.googleapis.com/Shelf").
			WithPatterns(
				api.ResourcePattern{
					*(&api.PathSegment{}).WithLiteral("publishers"),
					*(&api.PathSegment{}).WithVariable(api.NewPathVariable("publisher").WithMatch()),
					*(&api.PathSegment{}).WithLiteral("shelves"),
					*(&api.PathSegment{}).WithVariable(api.NewPathVariable("shelf").WithMatch()),
				},
			)
		// Find Shelf in the slice
		var foundShelf *api.Resource
		for _, r := range test.ResourceDefinitions {
			if r.Type == "library.googleapis.com/Shelf" {
				foundShelf = r
				break
			}
		}
		if foundShelf == nil {
			t.Fatalf("Expected ResourceDefinition for 'library.googleapis.com/Shelf' not found")
		}
		if diff := cmp.Diff(shelfResourceDef, foundShelf, cmpopts.IgnoreFields(api.Resource{}, "Self", "Codec", "Plural", "Singular")); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}

		// Verify Book
		bookResourceDef := api.NewTestResource("library.googleapis.com/Book").
			WithPatterns(
				api.ResourcePattern{
					*(&api.PathSegment{}).WithLiteral("publishers"),
					*(&api.PathSegment{}).WithVariable(api.NewPathVariable("publisher").WithMatch()),
					*(&api.PathSegment{}).WithLiteral("shelves"),
					*(&api.PathSegment{}).WithVariable(api.NewPathVariable("shelf").WithMatch()),
					*(&api.PathSegment{}).WithLiteral("books"),
					*(&api.PathSegment{}).WithVariable(api.NewPathVariable("book").WithMatch()),
				},
			).
			WithPlural("books").
			WithSingular("book")
		// Find Book in the slice
		var foundBook *api.Resource
		for _, r := range test.ResourceDefinitions {
			if r.Type == "library.googleapis.com/Book" {
				foundBook = r
				break
			}
		}
		if foundBook == nil {
			t.Fatalf("Expected ResourceDefinition for 'library.googleapis.com/Book' not found")
		}
		// Note: Book resource has 'Self' populated because it's a message resource.
		// Ignoring Self/Codec for comparison.
		if diff := cmp.Diff(bookResourceDef, foundBook, cmpopts.IgnoreFields(api.Resource{}, "Self", "Codec")); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("API.State.ResourceByType", func(t *testing.T) {
		if test.Resource("library.googleapis.com/Shelf") != nil {
			t.Errorf("Resource 'library.googleapis.com/Shelf' should not be in ResourceByType map")
		}

		bookResource := test.Resource("library.googleapis.com/Book")
		if bookResource == nil {
			t.Fatalf("Expected resource 'library.googleapis.com/Book' not found in ResourceByType map")
		}
		if bookResource.Type != "library.googleapis.com/Book" {
			t.Errorf("bookResource.Type = %q; want %q", bookResource.Type, "library.googleapis.com/Book")
		}
		if bookResource.Self.Name != "Book" {
			t.Errorf("bookResource.Self.Name = %q; want %q", bookResource.Self.Name, "Book")
		}
	})

	t.Run("Message.Resource", func(t *testing.T) {
		bookMessage := test.Message(".test.Book")
		if bookMessage == nil {
			t.Fatalf("Cannot find message %s in API State", ".test.Book")
		}

		// Check Resource separately to handle 'Self' cycle and ignore Codec
		wantBookResource := api.NewTestResource("library.googleapis.com/Book").
			WithPatterns(
				api.ResourcePattern{
					*(&api.PathSegment{}).WithLiteral("publishers"),
					*(&api.PathSegment{}).WithVariable(api.NewPathVariable("publisher").WithMatch()),
					*(&api.PathSegment{}).WithLiteral("shelves"),
					*(&api.PathSegment{}).WithVariable(api.NewPathVariable("shelf").WithMatch()),
					*(&api.PathSegment{}).WithLiteral("books"),
					*(&api.PathSegment{}).WithVariable(api.NewPathVariable("book").WithMatch()),
				},
			).
			WithPlural("books").
			WithSingular("book")

		if diff := cmp.Diff(wantBookResource, bookMessage.Resource, cmpopts.IgnoreFields(api.Resource{}, "Self", "Codec")); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}

		wantBook := api.NewTestMessage("Book").
			WithPackage("test").
			WithFields(
				api.NewTestField("name").
					WithType(api.TypezString),
			)
		for _, f := range wantBook.Fields {
			f.Parent = nil
		}
		apitest.CheckMessage(t, bookMessage, wantBook)
	})

	t.Run("CreateBookRequest", func(t *testing.T) {
		createBookRequest := test.Message(".test.CreateBookRequest")
		if createBookRequest == nil {
			t.Fatalf("Cannot find message %s in API State", ".test.CreateBookRequest")
		}

		wantCreate := api.NewTestMessage("CreateBookRequest").
			WithPackage("test").
			WithFields(
				api.NewTestField("parent").
					WithType(api.TypezString).
					WithResourceReference("library.googleapis.com/Shelf"),
				api.NewTestField("book_id").
					WithType(api.TypezString),
				api.NewTestField("book").
					WithType(api.TypezMessage).
					WithTypezID(".test.Book").
					WithOptional(),
			)
		for _, f := range wantCreate.Fields {
			f.Parent = nil
		}
		apitest.CheckMessage(t, createBookRequest, wantCreate)
	})

	t.Run("ListBooksRequest", func(t *testing.T) {
		listBooksRequest := test.Message(".test.ListBooksRequest")
		if listBooksRequest == nil {
			t.Fatalf("Cannot find message %s in API State", ".test.ListBooksRequest")
		}

		wantList := api.NewTestMessage("ListBooksRequest").
			WithPackage("test").
			WithFields(
				api.NewTestField("parent").
					WithType(api.TypezString).
					WithChildTypeReference("library.googleapis.com/Book"),
				api.NewTestField("page_size").
					WithType(api.TypezInt32),
				api.NewTestField("page_token").
					WithType(api.TypezString),
			)
		for _, f := range wantList.Fields {
			f.Parent = nil
		}
		apitest.CheckMessage(t, listBooksRequest, wantList)
	})

	t.Run("NoResourceMessage", func(t *testing.T) {
		msg := test.Message(".test.NoResourceMessage")
		if msg == nil {
			t.Fatalf("Cannot find message %s in API State", ".test.NoResourceMessage")
		}
		if msg.Resource != nil {
			t.Errorf("Expected NoResourceMessage to have nil Resource, got %v", msg.Resource)
		}
	})

	t.Run("NoReferenceMessage", func(t *testing.T) {
		msg := test.Message(".test.NoReferenceMessage")

		if msg == nil {
			t.Fatalf("Cannot find message %s in API State", ".test.NoReferenceMessage")
		}

		field := msg.Fields[0] // simple_field
		if field.IsResourceReference() {
			t.Errorf("Expected simple_field not to be ResourceReference, got %v", field.ResourceReference)
		}
	})
}

func TestProtobuf_ResourceCoverage(t *testing.T) {
	requireProtoc(t)

	t.Run("Deduplication", func(t *testing.T) {
		test, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "resource_coverage.proto"))
		if err != nil {
			t.Fatalf("Failed to make API for Protobuf %v", err)
		}

		// Verify only 1 resource exists and it is the message-level one ("book_message").
		// This confirms that message-level resources overwrite file-level resources with the same type.
		if len(test.ResourceDefinitions) != 1 {
			t.Fatalf("Expected 1 ResourceDefinition, got %d", len(test.ResourceDefinitions))
		}
		got := test.ResourceDefinitions[0]
		if got.Singular != "book_message" {
			t.Errorf("Expected singular 'book_message', got %q", got.Singular)
		}
		// Verify Self is set (only message resources have Self populated by processResourceAnnotation)
		if got.Self == nil {
			t.Errorf("Expected Resource.Self to be populated for message-level resource")
		}
	})

	t.Run("InvalidPattern", func(t *testing.T) {
		_, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "resource_invalid.proto"))
		if err == nil {
			t.Errorf("Expected error for invalid resource pattern, got nil")
		}
	})
}

func TestProtobuf_ParseBadFiles(t *testing.T) {
	requireProtoc(t)
	for _, cfg := range []*ModelConfig{
		{SpecificationSource: "-invalid-file-name-", ServiceConfig: secretManagerYamlFullPath},
		{SpecificationSource: protobufFile, ServiceConfig: "-invalid-file-name-"},
		{SpecificationSource: secretManagerYamlFullPath, ServiceConfig: secretManagerYamlFullPath},
		{DescriptorFiles: "dummy.desc", DescriptorFilesToGenerate: ""},
	} {
		if got, err := ParseProtobuf(cfg); err == nil {
			t.Fatalf("expected error with missing source file, got=%v", got)
		}
	}
}

func newTestCodeGeneratorRequest(t *testing.T, filename ...string) *pluginpb.CodeGeneratorRequest {
	t.Helper()
	src := &sources.SourceConfig{
		Sources: &sources.Sources{
			Googleapis:  "../../testdata/googleapis",
			ProtobufSrc: "testdata",
		},
		ActiveRoots: []string{"googleapis", "protobuf-src"},
		IncludeList: append([]string{}, filename...),
	}
	request, err := codeGeneratorRequestFromSource("testdata", src, nil)
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	return request
}

func TestParseResourcePatterns(t *testing.T) {
	t.Run("valid patterns", func(t *testing.T) {
		patterns := []string{
			"publishers/{publisher}/shelves/{shelf}",
			"projects/{project}",
		}
		want := []api.ResourcePattern{
			{
				*(&api.PathSegment{}).WithLiteral("publishers"),
				*(&api.PathSegment{}).WithVariable(api.NewPathVariable("publisher").WithMatch()),
				*(&api.PathSegment{}).WithLiteral("shelves"),
				*(&api.PathSegment{}).WithVariable(api.NewPathVariable("shelf").WithMatch()),
			},
			{
				*(&api.PathSegment{}).WithLiteral("projects"),
				*(&api.PathSegment{}).WithVariable(api.NewPathVariable("project").WithMatch()),
			},
		}
		got, err := parseResourcePatterns(patterns)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("invalid pattern", func(t *testing.T) {
		patterns := []string{"projects/{project=*}/**"}
		_, err := parseResourcePatterns(patterns)
		if err == nil {
			t.Fatal("parseResourcePatterns() expected an error, but got nil")
		}
		want := `failed to parse resource pattern "projects/{project=*}/**"`
		if !strings.Contains(err.Error(), want) {
			t.Errorf("parseResourcePatterns() returned error %q, want %q", err.Error(), want)
		}
	})
}

func TestParseProtobuf_Descriptors(t *testing.T) {
	requireProtoc(t)
	descFile := newTestDescriptorFile(t, "scalar.proto")
	defer os.Remove(descFile)

	cfg := &ModelConfig{
		DescriptorFiles:           descFile,
		DescriptorFilesToGenerate: "scalar.proto",
		ServiceConfig:             secretManagerYamlFullPath,
	}
	got, err := ParseProtobuf(cfg)
	if err != nil {
		t.Fatalf("ParseProtobuf failed: %v", err)
	}
	if got == nil {
		t.Fatalf("ParseProtobuf returned nil model")
	}
}

func TestParseProtobuf_ProtocConfig(t *testing.T) {
	cfg := &ModelConfig{
		SpecificationSource: "testdata",
		ServiceConfig:       secretManagerYamlFullPath,
		Protoc:              &config.Protoc{Version: "0.0.0-nonexistent"},
		Source: &sources.SourceConfig{
			Sources: &sources.Sources{
				Googleapis:  "../../testdata/googleapis",
				ProtobufSrc: "testdata",
			},
			ActiveRoots: []string{"googleapis", "protobuf-src"},
			IncludeList: []string{"scalar.proto"},
		},
	}
	if _, err := ParseProtobuf(cfg); err == nil {
		t.Fatal("ParseProtobuf with nonexistent protoc binary expected error, got nil")
	}
}

func newTestDescriptorFile(t *testing.T, filename string) string {
	t.Helper()

	tmpDir := t.TempDir()
	descFile := filepath.Join(tmpDir, "test.desc")

	cmd := exec.CommandContext(t.Context(), "protoc", "-o", descFile, "--include_imports",
		"-I", "testdata",
		"-I", "../../testdata/googleapis",
		filepath.Join("testdata", filename))

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to generate .desc file: %v", err)
	}

	return descFile
}

func requireProtoc(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("skipping test because protoc is not installed")
	}
}
