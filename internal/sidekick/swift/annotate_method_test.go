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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/api/apitest"
)

func TestAnnotateMethod(t *testing.T) {
	for _, test := range []struct {
		name       string
		methodFunc func() *api.Method
		wantFunc   func(keyField *api.Field) *methodAnnotations
	}{
		{
			name: "GET request",
			methodFunc: func() *api.Method {
				return api.NewTestMethod("GetOperation").
					WithVerb("GET").
					WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithLiteral("operations").Build()).
					WithDocumentation("Gets a thing.\n\nTest multiple comment lines.\n")
			},
			wantFunc: func(*api.Field) *methodAnnotations {
				return &methodAnnotations{
					Name:                       "getOperation",
					UnescapedName:              "getOperation",
					SeeAlsoLinks:               []string{"getOperation(request:)"},
					StorageControlSeeAlsoLinks: []string{"getOperation(request:)"},
					PathExpression:             "/v1/operations",
					DocLines:                   []string{"Gets a thing.", "", "Test multiple comment lines.", ""},
					HTTPMethod:                 "GET",
					HasBody:                    false,
					ReturnType:                 "Test.Response",
					ResponseEncoding:           "json;enum-encoding=int",
				}
			},
		},
		{
			name: "POST request with body field",
			methodFunc: func() *api.Method {
				return api.NewTestMethod("CreateKey").
					WithVerb("POST").
					WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithLiteral("keys").Build()).
					WithBodyFieldPath("key")
			},
			wantFunc: func(*api.Field) *methodAnnotations {
				return &methodAnnotations{
					Name:                       "createKey",
					UnescapedName:              "createKey",
					SeeAlsoLinks:               []string{"createKey(request:)"},
					StorageControlSeeAlsoLinks: []string{"createKey(request:)"},
					PathExpression:             "/v1/keys",
					HTTPMethod:                 "POST",
					HasBody:                    true,
					IsBodyWildcard:             false,
					BodyField:                  "key",
					ReturnType:                 "Test.Response",
					ResponseEncoding:           "json;enum-encoding=int",
				}
			},
		},
		{
			name: "POST request with wildcard body",
			methodFunc: func() *api.Method {
				return api.NewTestMethod("UploadData").
					WithVerb("POST").
					WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithLiteral("data").Build()).
					WithBodyFieldPath("*")
			},
			wantFunc: func(*api.Field) *methodAnnotations {
				return &methodAnnotations{
					Name:                       "uploadData",
					UnescapedName:              "uploadData",
					SeeAlsoLinks:               []string{"uploadData(request:)"},
					StorageControlSeeAlsoLinks: []string{"uploadData(request:)"},
					PathExpression:             "/v1/data",
					HTTPMethod:                 "POST",
					HasBody:                    true,
					IsBodyWildcard:             true,
					ReturnType:                 "Test.Response",
					ResponseEncoding:           "json;enum-encoding=int",
				}
			},
		},
		{
			name: "List request",
			methodFunc: func() *api.Method {
				return api.NewTestMethod("ListThings").
					WithVerb("GET").
					WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithLiteral("things").Build()).
					WithQueryParameters("key").
					WithDocumentation("Lists things.")
			},
			wantFunc: func(keyField *api.Field) *methodAnnotations {
				return &methodAnnotations{
					Name:                       "listThings",
					UnescapedName:              "listThings",
					SeeAlsoLinks:               []string{"listThings(request:)"},
					StorageControlSeeAlsoLinks: []string{"listThings(request:)"},
					PathExpression:             "/v1/things",
					DocLines:                   []string{"Lists things."},
					HTTPMethod:                 "GET",
					HasBody:                    false,
					QueryParams:                []*api.Field{keyField},
					ReturnType:                 "Test.Response",
					ResponseEncoding:           "json;enum-encoding=int",
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			keyField := api.NewTestField("key").WithType(api.TypezString)
			inputType := api.NewTestMessage("Request").WithFields(keyField)
			outputType := api.NewTestMessage("Response").
				WithFields(api.NewTestField("value").WithType(api.TypezString))

			method := test.methodFunc().WithInput(inputType).WithOutput(outputType)
			service := api.NewTestService("TestService").WithMethods(method)
			model := api.NewTestAPI([]*api.Message{inputType, outputType}, nil, []*api.Service{service})
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			got := method.Codec().(*methodAnnotations)
			want := test.wantFunc(keyField)
			if diff := cmp.Diff(want, got, apitest.AllowUnexported(), apitest.IgnoreFieldNames("parent", "codec"), cmpopts.IgnoreFields(methodAnnotations{}, "PathBindings", "HasMultipleBindings")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if !got.PlainRPC() {
				t.Errorf("got.PlainRPC() == true, want false\ngot=%+v", got)
			}
		})
	}
}

func TestAnnotateMethod_OmittedBodyFields(t *testing.T) {
	for _, test := range []struct {
		name                  string
		bodyFieldPath         string
		bindings              []*api.PathBinding
		want                  [][]string
		wantHasOmitted        bool
		wantOmittedExpression []string
	}{
		{
			name:          "wildcard body with multiple bindings",
			bodyFieldPath: "*",
			bindings: []*api.PathBinding{
				api.NewPathBindingBuilder().WithVerb("POST").WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("parent").Build()).Build(),
				api.NewPathBindingBuilder().WithVerb("POST").WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("secret", "name").Build()).Build(),
			},
			want:                  [][]string{{"parent"}, {"secret.name"}},
			wantHasOmitted:        true,
			wantOmittedExpression: []string{`["parent"]`, `["secret.name"]`},
		},
		{
			name:          "wildcard body without path variables",
			bodyFieldPath: "*",
			bindings: []*api.PathBinding{
				api.NewPathBindingBuilder().WithVerb("POST").WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithLiteral("secrets").Build()).Build(),
			},
			want:                  [][]string{nil},
			wantHasOmitted:        false,
			wantOmittedExpression: []string{`[]`},
		},
		{
			name:          "named body field",
			bodyFieldPath: "secret",
			bindings: []*api.PathBinding{
				api.NewPathBindingBuilder().WithVerb("POST").WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("parent").Build()).Build(),
			},
			want:                  [][]string{nil},
			wantHasOmitted:        false,
			wantOmittedExpression: []string{`[]`},
		},
		{
			name:          "no body",
			bodyFieldPath: "",
			bindings: []*api.PathBinding{
				api.NewPathBindingBuilder().WithVerb("GET").WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("parent").Build()).Build(),
			},
			want:                  [][]string{nil},
			wantHasOmitted:        false,
			wantOmittedExpression: []string{`[]`},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			secretMessage := api.NewTestMessage("Secret").
				WithFields(api.NewTestField("name").WithType(api.TypezString))
			inputType := api.NewTestMessage("Request").
				WithFields(
					api.NewTestField("parent").WithType(api.TypezString),
					api.NewTestField("display_name").WithType(api.TypezString),
					api.NewTestField("secret").WithMessageType(secretMessage).WithOptional(),
				)
			outputType := api.NewTestMessage("Response").
				WithFields(api.NewTestField("value").WithType(api.TypezString))

			method := api.NewTestMethod("Mutate").
				WithInput(inputType).
				WithOutput(outputType).
				WithBindings(test.bindings...).
				WithBodyFieldPath(test.bodyFieldPath)
			service := api.NewTestService("TestService").WithMethods(method)
			model := api.NewTestAPI([]*api.Message{inputType, outputType, secretMessage}, nil, []*api.Service{service})
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			got := method.Codec().(*methodAnnotations)
			var gotOmitted [][]string
			var gotExpressions []string
			for _, binding := range got.PathBindings {
				gotOmitted = append(gotOmitted, binding.OmittedBodyFields)
				gotExpressions = append(gotExpressions, binding.OmittedBodyFieldsExpression())
			}
			if diff := cmp.Diff(test.want, gotOmitted); diff != "" {
				t.Errorf("mismatch in OmittedBodyFields (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantOmittedExpression, gotExpressions); diff != "" {
				t.Errorf("mismatch in OmittedBodyFieldsExpression (-want +got):\n%s", diff)
			}
			if got.HasOmittedBodyFields() != test.wantHasOmitted {
				t.Errorf("got.HasOmittedBodyFields() == %v, want %v", got.HasOmittedBodyFields(), test.wantHasOmitted)
			}
		})
	}
}

func TestAnnotateMethod_EscapedName(t *testing.T) {
	for _, test := range []struct {
		name       string
		methodName string
		wantName   string
	}{
		{"escaped func", "Func", "`func`"},
		{"escaped self", "Self", "self_"},
		{"escaped default", "Default", "`default`"},
	} {
		t.Run(test.name, func(t *testing.T) {
			inputType := api.NewTestMessage("Request").
				WithFields(api.NewTestField("key").WithType(api.TypezString))
			outputType := api.NewTestMessage("Response").
				WithFields(api.NewTestField("value").WithType(api.TypezString))
			method := api.NewTestMethod(test.methodName).
				WithInput(inputType).
				WithOutput(outputType).
				WithVerb("GET").WithPathTemplate(api.NewPathTemplateBuilder().Build()).
				WithDocumentation("Test documentation.")
			service := api.NewTestService("TestService").WithMethods(method)
			model := api.NewTestAPI([]*api.Message{inputType, outputType}, nil, []*api.Service{service})
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			codec := newTestCodec(t, model, nil)

			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			unescaped := strings.ReplaceAll(test.wantName, "`", "")
			want := &methodAnnotations{
				Name:                       test.wantName,
				UnescapedName:              unescaped,
				SeeAlsoLinks:               []string{unescaped + "(request:)"},
				StorageControlSeeAlsoLinks: []string{unescaped + "(request:)"},
				DocLines:                   []string{"Test documentation."},
				PathExpression:             "/",
				HTTPMethod:                 "GET",
				ReturnType:                 "Test.Response",
				ResponseEncoding:           "json;enum-encoding=int",
			}

			if diff := cmp.Diff(want, method.Codec(), cmpopts.IgnoreFields(methodAnnotations{}, "PathBindings", "HasMultipleBindings")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateMethod_WithExternalMessages(t *testing.T) {
	inputMessage := api.NewTestMessage("InputMessage").
		WithPackage("google.cloud.external.v1")
	outputMessage := api.NewTestMessage("OutputMessage").
		WithPackage("google.cloud.external.v1")
	method := api.NewTestMethod("TestMethod").
		WithInput(inputMessage).
		WithOutput(outputMessage).
		WithVerb("POST").
		WithPathTemplate(api.NewPathTemplateBuilder().Build())
	service := api.NewTestService("TestService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{}, nil, []*api.Service{service}).
		WithPackageName("google.cloud.test.v1")
	model.AddMessage(inputMessage)
	model.AddMessage(outputMessage)
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	codec := newTestCodec(t, model, nil)
	codec.withExtraDependencies(t, []config.SwiftDependency{
		{
			ApiPackage: "google.cloud.external.v1",
			Name:       "GoogleCloudExternalV1",
		},
	})

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	if inputMessage.Codec() == nil {
		t.Error("expected input message to be annotated")
	}
	if outputMessage.Codec() == nil {
		t.Error("expected output message to be annotated")
	}
}

func TestAnnotateMethod_Pagination(t *testing.T) {
	pageSizeField := api.NewTestField("page_size").WithType(api.TypezInt32)
	pageTokenField := api.NewTestField("page_token").WithType(api.TypezString)
	inputType := api.NewTestMessage("ListRequest").
		WithFields(pageSizeField, pageTokenField)

	itemType := api.NewTestMessage("Item")
	itemField := api.NewTestField("items").
		WithMessageType(itemType).
		WithRepeated()
	nextPageTokenField := api.NewTestField("next_page_token").WithType(api.TypezString)
	outputType := api.NewTestMessage("ListResponse").
		WithFields(itemField, nextPageTokenField).
		WithPagination(nextPageTokenField, itemField)

	method := api.NewTestMethod("ListItems").
		WithInput(inputType).
		WithOutput(outputType).
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().Build()).
		WithPagination(pageTokenField)

	service := api.NewTestService("TestService").WithMethods(method)

	model := api.NewTestAPI([]*api.Message{inputType, outputType, itemType}, nil, []*api.Service{service}).
		WithPackageName("test")
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	codec := newTestCodec(t, model, nil)

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	// Verify method annotations
	gotMethod := method.Codec().(*methodAnnotations)
	wantMethod := &methodAnnotations{
		Name:          "listItems",
		UnescapedName: "listItems",
		SeeAlsoLinks: []string{
			"listItems(request:)",
			"listItemsByItems(request:)",
			"listItemsByItems(request:options:)",
		},
		StorageControlSeeAlsoLinks: []string{
			"listItems(request:)",
			"listItemsByItems(request:)",
			"listItemsByItems(request:options:)",
		},
		PathExpression: "/",
		HTTPMethod:     "GET",
		Pagination: &paginationAnnotations{
			ItemType: "Item",
		},
		ReturnType:       "Test.ListResponse",
		ResponseEncoding: "json;enum-encoding=int",
	}
	if diff := cmp.Diff(wantMethod, gotMethod, cmpopts.IgnoreFields(methodAnnotations{}, "PathBindings", "HasMultipleBindings")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if gotMethod.PlainRPC() {
		t.Errorf("gotMethod.PlainRPC() == false, want true\ngotMethod=%+v", gotMethod)
	}

	// Verify request message annotations
	gotRequest := inputType.Codec().(*messageAnnotations)
	wantRequest := &messageAnnotations{
		Name:              "ListRequest",
		TypeURL:           "type.googleapis.com/test.ListRequest",
		SampleField:       "pageSize",
		ParameterTypeName: "ListRequest",
	}
	if diff := cmp.Diff(wantRequest, gotRequest, cmpopts.IgnoreFields(messageAnnotations{}, "Model", "DependsOn", "ProtoTypeName", "ModulePath")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantRequestImports := []*dependencyImport{{Module: "GoogleWKT"}}
	if diff := cmp.Diff(wantRequestImports, gotRequest.MessageImports()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify response message annotations
	gotResponse := outputType.Codec().(*messageAnnotations)
	wantResponse := &messageAnnotations{
		Name:                "ListResponse",
		TypeURL:             "type.googleapis.com/test.ListResponse",
		IsPaginatedResponse: true,
		PageableItemField:   "items",
		PageableItemType:    "Item",
		SampleField:         "items",
		ParameterTypeName:   "ListResponse",
	}
	if diff := cmp.Diff(wantResponse, gotResponse, cmpopts.IgnoreFields(messageAnnotations{}, "Model", "DependsOn", "ProtoTypeName", "ModulePath")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	// Response type is a paginated response which depends on gax
	wantResponseImports := []*dependencyImport{{Module: "GoogleGax"}, {Module: "GoogleWKT"}}
	if diff := cmp.Diff(wantResponseImports, gotResponse.MessageImports()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateMethod_LRO(t *testing.T) {
	inputType := api.NewTestMessage("Request")
	outputType := api.NewTestMessage("Operation")
	lroResponseType := api.NewTestMessage("LroResponse")
	lroMetadataType := api.NewTestMessage("LroMetadata")

	method := api.NewTestMethod("LroMethod").
		WithInput(inputType).
		WithOutput(outputType).
		WithVerb("POST").
		WithPathTemplate(api.NewPathTemplateBuilder().Build()).
		WithOperationInfo(api.NewTestOperationInfo(lroResponseType.ID(), lroMetadataType.ID()))

	service := api.NewTestService("TestService").WithMethods(method)

	model := api.NewTestAPI([]*api.Message{inputType, outputType, lroResponseType, lroMetadataType}, nil, []*api.Service{service}).
		WithPackageName("test")
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	codec := newTestCodec(t, model, nil)

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	gotMethod := method.Codec().(*methodAnnotations)
	wantMethod := &methodAnnotations{
		Name:          "lroMethod",
		UnescapedName: "lroMethod",
		SeeAlsoLinks: []string{
			"lroMethod(request:)",
			"lroMethodPollingUntilDone(request:)",
			"lroMethodPollingUntilDone(request:options:)",
		},
		PollingSeeAlsoLinks: []string{
			"lroMethod(request:)",
			"lroMethod(request:options:)",
			"lroMethodPollingUntilDone(request:)",
		},
		StorageControlSeeAlsoLinks: []string{
			"lroMethod(request:)",
			"lroMethodPollingUntilDone(request:)",
			"lroMethodPollingUntilDone(request:options:)",
		},
		StorageControlPollingSeeAlsoLinks: []string{
			"lroMethod(request:)",
			"lroMethod(request:options:)",
			"lroMethodPollingUntilDone(request:)",
		},
		PathExpression: "/",
		HTTPMethod:     "POST",
		LRO: &lroAnnotations{
			ReturnType:      "LroResponse",
			MetadataType:    "LroMetadata",
			ResponseIsEmpty: false,
		},
		ReturnType:       "Test.Operation",
		ResponseEncoding: "json;enum-encoding=int",
	}
	if diff := cmp.Diff(wantMethod, gotMethod, cmpopts.IgnoreFields(methodAnnotations{}, "PathBindings", "HasMultipleBindings")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if gotMethod.PlainRPC() {
		t.Errorf("gotMethod.PlainRPC() == false, want true\ngotMethod=%+v", gotMethod)
	}
}

func TestAnnotateMethod_LRO_Empty(t *testing.T) {
	inputType := api.NewTestMessage("Request")
	outputType := api.NewTestMessage("Operation")
	lroMetadataType := api.NewTestMessage("LroMetadata")

	method := api.NewTestMethod("LroMethod").
		WithInput(inputType).
		WithOutput(outputType).
		WithVerb("POST").
		WithPathTemplate(api.NewPathTemplateBuilder().Build()).
		WithOperationInfo(api.NewTestOperationInfo(".google.protobuf.Empty", lroMetadataType.ID()))

	service := api.NewTestService("TestService").WithMethods(method)

	model := api.NewTestAPI([]*api.Message{inputType, outputType, lroMetadataType}, nil, []*api.Service{service}).
		WithPackageName("test")
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	codec := newTestCodec(t, model, nil)

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	gotMethod := method.Codec().(*methodAnnotations)
	wantMethod := &methodAnnotations{
		Name:          "lroMethod",
		UnescapedName: "lroMethod",
		SeeAlsoLinks: []string{
			"lroMethod(request:)",
			"lroMethodPollingUntilDone(request:)",
			"lroMethodPollingUntilDone(request:options:)",
		},
		PollingSeeAlsoLinks: []string{
			"lroMethod(request:)",
			"lroMethod(request:options:)",
			"lroMethodPollingUntilDone(request:)",
		},
		StorageControlSeeAlsoLinks: []string{
			"lroMethod(request:)",
			"lroMethodPollingUntilDone(request:)",
			"lroMethodPollingUntilDone(request:options:)",
		},
		StorageControlPollingSeeAlsoLinks: []string{
			"lroMethod(request:)",
			"lroMethod(request:options:)",
			"lroMethodPollingUntilDone(request:)",
		},
		PathExpression: "/",
		HTTPMethod:     "POST",
		LRO: &lroAnnotations{
			ReturnType:      "Swift.Void",
			MetadataType:    "LroMetadata",
			ResponseIsEmpty: true,
		},
		ReturnType:       "Test.Operation",
		ResponseEncoding: "json;enum-encoding=int",
	}
	if diff := cmp.Diff(wantMethod, gotMethod, cmpopts.IgnoreFields(methodAnnotations{}, "PathBindings", "HasMultipleBindings")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if gotMethod.PlainRPC() {
		t.Errorf("gotMethod.PlainRPC() == false, want true\ngotMethod=%+v", gotMethod)
	}
}

func TestAnnotateMethod_MultipleBindings(t *testing.T) {
	inputType := api.NewTestMessage("Request").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
			api.NewTestField("project").WithType(api.TypezString),
		)
	outputType := api.NewTestMessage("Response")

	method := api.NewTestMethod("GetResource").
		WithInput(inputType).
		WithOutput(outputType).
		WithBindings(
			api.NewPathBindingBuilder().
				WithVerb("GET").
				WithPathTemplate(api.NewPathTemplateBuilder().
					WithLiteral("v1").
					WithVariableNamed("name").
					Build()).
				Build(),
			api.NewPathBindingBuilder().
				WithVerb("POST").
				WithPathTemplate(api.NewPathTemplateBuilder().
					WithLiteral("v1").
					WithVariableNamed("project").
					WithLiteral("resources").
					Build()).
				Build(),
		)
	service := api.NewTestService("TestService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{inputType, outputType}, nil, []*api.Service{service})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	codec := newTestCodec(t, model, nil)
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}
	got := method.Codec().(*methodAnnotations)
	if !got.HasMultipleBindings {
		t.Errorf("got.HasMultipleBindings = false, want true")
	}
	if len(got.PathBindings) != 2 {
		t.Fatalf("len(got.PathBindings) = %d, want 2", len(got.PathBindings))
	}
	if got.HTTPMethod != "GET" || got.PathExpression != "/v1/\\(pathVariable0)" {
		t.Errorf("primary binding mismatch: got method %s, path %s", got.HTTPMethod, got.PathExpression)
	}
	b0 := got.PathBindings[0]
	if b0.HTTPMethod != "GET" || b0.PathExpression != "/v1/\\(pathVariable0)" {
		t.Errorf("b0 mismatch: got method %s, path %s", b0.HTTPMethod, b0.PathExpression)
	}
	if len(b0.PathVariables) != 1 || b0.PathVariables[0].FieldPath != "name" {
		t.Errorf("b0.PathVariables mismatch: %+v", b0.PathVariables)
	}
	b1 := got.PathBindings[1]
	if b1.HTTPMethod != "POST" || b1.PathExpression != "/v1/\\(pathVariable0)/resources" {
		t.Errorf("b1 mismatch: got method %s, path %s", b1.HTTPMethod, b1.PathExpression)
	}
	if len(b1.PathVariables) != 1 || b1.PathVariables[0].FieldPath != "project" {
		t.Errorf("b1.PathVariables mismatch: %+v", b1.PathVariables)
	}
}

// methodDiagnoseFlags is the `Diagnose*` subset of methodAnnotations.
//
// The flags form a matrix that is far easier to read on its own than inside a
// full methodAnnotations literal.
type methodDiagnoseFlags struct {
	Types      bool
	Fields     bool
	StubTypes  bool
	StubFields bool
	Snippet    bool
}

func getMethodDiagnoseFlags(t *testing.T, method *api.Method) methodDiagnoseFlags {
	t.Helper()
	ann, ok := method.Codec().(*methodAnnotations)
	if !ok {
		t.Fatalf("method %q is not annotated", method.Name())
	}
	return methodDiagnoseFlags{
		Types:      ann.DiagnoseTypes,
		Fields:     ann.DiagnoseFields,
		StubTypes:  ann.DiagnoseStubTypes,
		StubFields: ann.DiagnoseStubFields,
		Snippet:    ann.DiagnoseSnippet,
	}
}

func TestAnnotateMethod_Diagnose(t *testing.T) {
	for _, test := range []struct {
		name               string
		requestDeprecated  bool
		responseDeprecated bool
		fieldDeprecated    bool
		methodDeprecated   bool
		serviceDeprecated  bool
		want               methodDiagnoseFlags
	}{
		{
			name: "nothing-deprecated",
			want: methodDiagnoseFlags{},
		},
		{
			name:              "deprecated-request-type",
			requestDeprecated: true,
			want: methodDiagnoseFlags{
				Types: true, Fields: true,
				StubTypes: true, StubFields: true,
				Snippet: true,
			},
		},
		{
			name:               "deprecated-response-type",
			responseDeprecated: true,
			want: methodDiagnoseFlags{
				Types: true, Fields: true,
				StubTypes: true, StubFields: true,
				Snippet: true,
			},
		},
		{
			// The types are live, so only the declarations that name the
			// field itself need the attribute.
			name:            "deprecated-request-field",
			fieldDeprecated: true,
			want: methodDiagnoseFlags{
				Fields: true, StubFields: true, Snippet: true,
			},
		},
		{
			// The public declarations carry `@available(*, deprecated)`, the
			// internal ones do not, and nothing else here is deprecated.
			name:             "deprecated-method-only",
			methodDeprecated: true,
			want: methodDiagnoseFlags{
				Snippet: true,
			},
		},
		{
			name:              "deprecated-method-and-request-type",
			methodDeprecated:  true,
			requestDeprecated: true,
			want: methodDiagnoseFlags{
				StubTypes: true, StubFields: true, Snippet: true,
			},
		},
		{
			name:              "deprecated-service-and-request-type",
			serviceDeprecated: true,
			requestDeprecated: true,
			want: methodDiagnoseFlags{
				StubTypes: true, StubFields: true, Snippet: true,
			},
		},
		{
			// The transport names the field, the public overloads do not,
			// because they are already deprecated.
			name:             "deprecated-method-and-request-field",
			methodDeprecated: true,
			fieldDeprecated:  true,
			want: methodDiagnoseFlags{
				StubFields: true, Snippet: true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			inputType := api.NewTestMessage("Request").
				WithDeprecated(test.requestDeprecated).
				WithFields(api.NewTestField("name").
					WithType(api.TypezString).
					WithDeprecated(test.fieldDeprecated))
			outputType := api.NewTestMessage("Response").
				WithDeprecated(test.responseDeprecated)

			method := api.NewTestMethod("SimpleMethod").
				WithInput(inputType).
				WithOutput(outputType).
				WithVerb("POST").
				WithPathTemplate(api.NewPathTemplateBuilder().Build()).
				WithDeprecated(test.methodDeprecated)
			service := api.NewTestService("TestService").
				WithDeprecated(test.serviceDeprecated).
				WithMethods(method)

			model := api.NewTestAPI([]*api.Message{inputType, outputType}, nil, []*api.Service{service}).
				WithPackageName("test")
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			got := getMethodDiagnoseFlags(t, method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestAnnotateMethod_DiagnosePaginationItemType covers the pagination item
// type.
//
// The pagination signatures return `any AsyncSequence<ItemType, ...>`. The item
// type is a field type of the response message, so it is neither the input nor
// the output type and `diagnoseMethodTypes` alone cannot see it.
func TestAnnotateMethod_DiagnosePaginationItemType(t *testing.T) {
	for _, test := range []struct {
		name               string
		itemTypeDeprecated bool
		want               methodDiagnoseFlags
	}{
		{
			name:               "deprecated-item-type",
			itemTypeDeprecated: true,
			want: methodDiagnoseFlags{
				Types: true, Fields: true,
				StubTypes: true, StubFields: true,
				Snippet: true,
			},
		},
		{
			name: "not-deprecated",
			want: methodDiagnoseFlags{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			pageTokenField := api.NewTestField("page_token").WithType(api.TypezString)
			inputType := api.NewTestMessage("ListRequest").WithFields(pageTokenField)

			itemType := api.NewTestMessage("Item").WithDeprecated(test.itemTypeDeprecated)
			itemField := api.NewTestField("items").WithMessageType(itemType).WithRepeated()
			nextPageTokenField := api.NewTestField("next_page_token").WithType(api.TypezString)
			outputType := api.NewTestMessage("ListResponse").
				WithFields(itemField, nextPageTokenField).
				WithPagination(nextPageTokenField, itemField)

			method := api.NewTestMethod("ListItems").
				WithInput(inputType).
				WithOutput(outputType).
				WithVerb("GET").
				WithPathTemplate(api.NewPathTemplateBuilder().Build()).
				WithPagination(pageTokenField)
			service := api.NewTestService("TestService").WithMethods(method)

			model := api.NewTestAPI(
				[]*api.Message{inputType, outputType, itemType}, nil, []*api.Service{service}).
				WithPackageName("test")
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			got := getMethodDiagnoseFlags(t, method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestAnnotateMethod_DiagnoseLROResponseType covers the LRO response type.
//
// The LRO signatures return `ReturnType` (or Void if empty). For
// a real LRO the output type is `google.longrunning.Operation`; the type the
// caller sees is resolved from `OperationInfo.ResponseTypeID`, so
// `diagnoseMethodTypes` alone cannot see it.
func TestAnnotateMethod_DiagnoseLROResponseType(t *testing.T) {
	for _, test := range []struct {
		name                  string
		lroResponseDeprecated bool
		lroMetadataDeprecated bool
		want                  methodDiagnoseFlags
	}{
		{
			name:                  "deprecated-lro-response",
			lroResponseDeprecated: true,
			want: methodDiagnoseFlags{
				Types: true, Fields: true,
				StubTypes: true, StubFields: true,
				Snippet: true,
			},
		},
		{
			// No template names the metadata type.
			name:                  "deprecated-lro-metadata",
			lroMetadataDeprecated: true,
			want:                  methodDiagnoseFlags{},
		},
		{
			name: "not-deprecated",
			want: methodDiagnoseFlags{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			inputType := api.NewTestMessage("Request")
			outputType := api.NewTestMessage("Operation")
			lroResponseType := api.NewTestMessage("LroResponse").
				WithDeprecated(test.lroResponseDeprecated)
			lroMetadataType := api.NewTestMessage("LroMetadata").
				WithDeprecated(test.lroMetadataDeprecated)

			method := api.NewTestMethod("LroMethod").
				WithInput(inputType).
				WithOutput(outputType).
				WithVerb("POST").
				WithPathTemplate(api.NewPathTemplateBuilder().Build()).
				WithOperationInfo(api.NewTestOperationInfo(lroResponseType.ID(), lroMetadataType.ID()))
			service := api.NewTestService("TestService").WithMethods(method)

			model := api.NewTestAPI(
				[]*api.Message{inputType, outputType, lroResponseType, lroMetadataType},
				nil, []*api.Service{service}).
				WithPackageName("test")
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			got := getMethodDiagnoseFlags(t, method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateMethod_AutoPopulated(t *testing.T) {
	for _, test := range []struct {
		name                 string
		verb                 string
		autoPopulated        bool
		wantHasAutoPopulated bool
		wantIdempotent       bool
	}{
		{
			name:                 "post-without-auto-populated",
			verb:                 "POST",
			autoPopulated:        false,
			wantHasAutoPopulated: false,
			wantIdempotent:       false,
		},
		{
			name:                 "post-with-auto-populated",
			verb:                 "POST",
			autoPopulated:        true,
			wantHasAutoPopulated: true,
			wantIdempotent:       true,
		},
		{
			name:                 "get-with-auto-populated",
			verb:                 "GET",
			autoPopulated:        true,
			wantHasAutoPopulated: true,
			wantIdempotent:       true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			reqIDField := api.NewTestField("request_id").WithType(api.TypezString)
			if test.autoPopulated {
				reqIDField.WithAutoPopulated()
			}
			inputType := api.NewTestMessage("CreateRequest").WithFields(reqIDField)
			outputType := api.NewTestMessage("CreateResponse")
			method := api.NewTestMethod("CreateResource").
				WithInput(inputType).
				WithOutput(outputType).
				WithVerb(test.verb).
				WithPathTemplate(api.NewPathTemplateBuilder().Build())
			if test.autoPopulated {
				method.WithAutoPopulated(reqIDField)
			}
			service := api.NewTestService("TestService").WithMethods(method)
			model := api.NewTestAPI([]*api.Message{inputType, outputType}, nil, []*api.Service{service}).
				WithPackageName("test")
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			ann, ok := method.Codec().(*methodAnnotations)
			if !ok {
				t.Fatalf("method.Codec() is not *methodAnnotations")
			}
			if ann.HasAutoPopulatedFields != test.wantHasAutoPopulated {
				t.Errorf("ann.HasAutoPopulatedFields = %v, want %v", ann.HasAutoPopulatedFields, test.wantHasAutoPopulated)
			}
			if ann.Idempotent() != test.wantIdempotent {
				t.Errorf("ann.Idempotent() = %v, want %v", ann.Idempotent(), test.wantIdempotent)
			}
		})
	}
}

func TestAnnotateMethodIdempotencyHook(t *testing.T) {
	for _, test := range []struct {
		name                   string
		library                *config.Library
		wantHasIdempotencyHook bool
		wantIdempotencyHook    string
	}{
		{
			name: "with idempotency hook configured",
			library: &config.Library{
				Swift: &config.SwiftPackage{
					IdempotencyHook: "resolveIdempotency",
				},
			},
			wantHasIdempotencyHook: true,
			wantIdempotencyHook:    "resolveIdempotency",
		},
		{
			name:                   "without idempotency hook configured",
			library:                nil,
			wantHasIdempotencyHook: false,
			wantIdempotencyHook:    "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			inputType := api.NewTestMessage("Request")
			outputType := api.NewTestMessage("Response")
			method := api.NewTestMethod("DeleteObject").
				WithInput(inputType).
				WithOutput(outputType).
				WithVerb("POST").
				WithPathTemplate(api.NewPathTemplateBuilder().Build())
			service := api.NewTestService("Storage").WithMethods(method)
			model := api.NewTestAPI([]*api.Message{inputType, outputType}, nil, []*api.Service{service}).
				WithPackageName("test")
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			codec := newTestCodec(t, model, test.library)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			ann, ok := method.Codec().(*methodAnnotations)
			if !ok {
				t.Fatalf("expected method.Codec() to be *methodAnnotations, got %T", method.Codec())
			}
			if ann.HasIdempotencyHook != test.wantHasIdempotencyHook {
				t.Errorf("got HasIdempotencyHook = %v, want %v", ann.HasIdempotencyHook, test.wantHasIdempotencyHook)
			}
			if ann.IdempotencyHook != test.wantIdempotencyHook {
				t.Errorf("got IdempotencyHook = %q, want %q", ann.IdempotencyHook, test.wantIdempotencyHook)
			}
		})
	}
}

func TestAnnotateMethod_HTTPBody(t *testing.T) {
	httpBodyMsg := api.NewTestMessage("HttpBody").
		WithPackage("google.api").
		WithFields(
			api.NewTestField("content_type").WithType(api.TypezString),
			api.NewTestField("data").WithType(api.TypezBytes),
		)
	wrapperReq := api.NewTestMessage("RawPredictRequest").
		WithFields(
			api.NewTestField("endpoint").WithType(api.TypezString),
			api.NewTestField("http_body").WithMessageType(httpBodyMsg).WithOptional(),
		)
	regularMsg := api.NewTestMessage("RegularMessage").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
		)

	for _, test := range []struct {
		name          string
		input         *api.Message
		output        *api.Message
		bodyFieldPath string
		want          *methodAnnotations
	}{
		{
			name:          "wrapper request with http_body field and HttpBody response",
			input:         wrapperReq,
			output:        httpBodyMsg,
			bodyFieldPath: "http_body",
			want: &methodAnnotations{
				Name:                       "testMethod",
				UnescapedName:              "testMethod",
				SeeAlsoLinks:               []string{"testMethod(request:)"},
				StorageControlSeeAlsoLinks: []string{"testMethod(request:)"},
				PathExpression:             "/v1/test",
				HTTPMethod:                 "POST",
				HasBody:                    true,
				BodyField:                  "httpBody",
				IsHTTPBodyRequest:          true,
				IsHTTPBodyResponse:         true,
				ReturnType:                 "GoogleApi.HttpBody",
				ResponseEncoding:           "json;enum-encoding=int",
			},
		},
		{
			name:          "direct HttpBody request with wildcard body and regular response",
			input:         httpBodyMsg,
			output:        regularMsg,
			bodyFieldPath: "*",
			want: &methodAnnotations{
				Name:                       "testMethod",
				UnescapedName:              "testMethod",
				SeeAlsoLinks:               []string{"testMethod(request:)"},
				StorageControlSeeAlsoLinks: []string{"testMethod(request:)"},
				PathExpression:             "/v1/test",
				HTTPMethod:                 "POST",
				HasBody:                    true,
				IsBodyWildcard:             true,
				IsHTTPBodyRequest:          true,
				IsHTTPBodyResponse:         false,
				ReturnType:                 "Test.RegularMessage",
				ResponseEncoding:           "json;enum-encoding=int",
			},
		},
		{
			name:          "regular request and HttpBody response",
			input:         regularMsg,
			output:        httpBodyMsg,
			bodyFieldPath: "",
			want: &methodAnnotations{
				Name:                       "testMethod",
				UnescapedName:              "testMethod",
				SeeAlsoLinks:               []string{"testMethod(request:)"},
				StorageControlSeeAlsoLinks: []string{"testMethod(request:)"},
				PathExpression:             "/v1/test",
				HTTPMethod:                 "POST",
				HasBody:                    false,
				IsHTTPBodyRequest:          false,
				IsHTTPBodyResponse:         true,
				ReturnType:                 "GoogleApi.HttpBody",
				ResponseEncoding:           "json;enum-encoding=int",
			},
		},
		{
			name:          "regular request with wildcard body and regular response",
			input:         regularMsg,
			output:        regularMsg,
			bodyFieldPath: "*",
			want: &methodAnnotations{
				Name:                       "testMethod",
				UnescapedName:              "testMethod",
				SeeAlsoLinks:               []string{"testMethod(request:)"},
				StorageControlSeeAlsoLinks: []string{"testMethod(request:)"},
				PathExpression:             "/v1/test",
				HTTPMethod:                 "POST",
				HasBody:                    true,
				IsBodyWildcard:             true,
				IsHTTPBodyRequest:          false,
				IsHTTPBodyResponse:         false,
				ReturnType:                 "Test.RegularMessage",
				ResponseEncoding:           "json;enum-encoding=int",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			method := api.NewTestMethod("TestMethod").
				WithInput(test.input).
				WithOutput(test.output).
				WithVerb("POST").
				WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithLiteral("test").Build()).
				WithBodyFieldPath(test.bodyFieldPath)
			service := api.NewTestService("TestService").WithMethods(method)
			model := api.NewTestAPI([]*api.Message{wrapperReq, regularMsg}, nil, []*api.Service{service}).
				WithPackageName("test")
			model.AddMessage(httpBodyMsg)
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			codec := newTestCodec(t, model, nil)
			codec.withExtraDependencies(t, []config.SwiftDependency{
				{
					ApiPackage: "google.api",
					Name:       "GoogleApi",
				},
			})
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			got := method.Codec().(*methodAnnotations)
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(methodAnnotations{}, "PathBindings", "HasMultipleBindings")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateMethod_ExplicitEmptyRoutingSuppressesHTTPFallback(t *testing.T) {
	inputType := api.NewTestMessage("Request").
		WithPackage("test.v1").
		WithFields(
			api.NewTestField("table_name").WithType(api.TypezString),
		)
	outputType := api.NewTestMessage("Response").WithPackage("test.v1")

	explicitEmpty := api.NewTestMethod("ExplicitEmpty").
		WithInput(inputType).
		WithOutput(outputType).
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("table_name").Build()).
		WithRouting()

	implicitOnly := api.NewTestMethod("ImplicitOnly").
		WithInput(inputType).
		WithOutput(outputType).
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("table_name").Build())

	service := api.NewTestService("TestService").
		WithPackage("test.v1").
		WithMethods(explicitEmpty, implicitOnly)
	model := api.NewTestAPI([]*api.Message{inputType, outputType}, nil, []*api.Service{service}).
		WithPackageName("test.v1")
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	codec, err := newCodec(
		model,
		&config.Library{Swift: swiftConfig(t, nil)},
		&config.SwiftModule{ModuleType: "grpc-client", ModulePath: "TestProtos"},
		t.TempDir(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	gotExplicit := explicitEmpty.Codec().(*methodAnnotations)
	if gotExplicit.HasRoutingParams() {
		t.Errorf("ExplicitEmpty HasRoutingParams() = true, want false; got %v", gotExplicit.RoutingParams)
	}
	if len(gotExplicit.RoutingParams) != 0 {
		t.Errorf("ExplicitEmpty RoutingParams = %v, want empty", gotExplicit.RoutingParams)
	}

	gotImplicit := implicitOnly.Codec().(*methodAnnotations)
	if !gotImplicit.HasRoutingParams() {
		t.Errorf("ImplicitOnly HasRoutingParams() = false, want true")
	}
	if len(gotImplicit.RoutingParams) != 1 || gotImplicit.RoutingParams[0].RoutingKey != "table_name" {
		t.Errorf("ImplicitOnly RoutingParams = %v, want table_name", gotImplicit.RoutingParams)
	}
}

func TestAnnotateMethod_RoutingAdditionalBindings(t *testing.T) {
	inputType := api.NewTestMessage("BillingRequest").
		WithPackage("google.cloud.billing.v1").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
			api.NewTestField("parent").WithType(api.TypezString),
			api.NewTestField("destination_parent").WithType(api.TypezString),
		)
	outputType := api.NewTestMessage("BillingResponse").
		WithPackage("google.cloud.billing.v1")

	listAccounts := api.NewTestMethod("ListBillingAccounts").
		WithInput(inputType).
		WithOutput(outputType).
		WithBindings(
			newTestPathBinding("GET", api.NewPathTemplateBuilder().WithLiteral("v1").WithLiteral("billingAccounts").Build()),
			newTestPathBinding("GET", api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("parent").WithLiteral("billingAccounts").Build()),
		)

	moveAccount := api.NewTestMethod("MoveBillingAccount").
		WithInput(inputType).
		WithOutput(outputType).
		WithBindings(
			newTestPathBinding("POST", api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("name").WithVerb("move").Build()),
			newTestPathBinding("GET", api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("destination_parent").WithVariableNamed("name").WithVerb("move").Build()),
		)

	explicitOverride := api.NewTestMethod("ExplicitOverride").
		WithInput(inputType).
		WithOutput(outputType).
		WithBindings(
			newTestPathBinding("GET", api.NewPathTemplateBuilder().WithLiteral("v1").WithLiteral("billingAccounts").Build()),
			newTestPathBinding("GET", api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("parent").WithLiteral("billingAccounts").Build()),
		).
		WithRouting(api.NewTestRoutingInfo("bucket", api.NewTestRoutingInfoVariant("destination_parent")))

	explicitEmpty := api.NewTestMethod("ExplicitEmpty").
		WithInput(inputType).
		WithOutput(outputType).
		WithBindings(
			newTestPathBinding("GET", api.NewPathTemplateBuilder().WithLiteral("v1").WithLiteral("billingAccounts").Build()),
			newTestPathBinding("GET", api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("parent").WithLiteral("billingAccounts").Build()),
		).
		WithRouting()

	service := api.NewTestService("CloudBilling").
		WithPackage("google.cloud.billing.v1").
		WithMethods(listAccounts, moveAccount, explicitOverride, explicitEmpty)
	model := api.NewTestAPI([]*api.Message{inputType, outputType}, nil, []*api.Service{service}).
		WithPackageName("google.cloud.billing.v1")
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	codec, err := newCodec(
		model,
		&config.Library{Swift: swiftConfig(t, nil)},
		&config.SwiftModule{ModuleType: "grpc-client", ModulePath: "TestProtos"},
		t.TempDir(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name   string
		method *api.Method
		want   []*routingParam
	}{
		{
			name:   "literal primary with additional binding variable",
			method: listAccounts,
			want:   []*routingParam{newTestMultiWildcardRoutingParam("parent", []*routingFieldStep{{PropertyName: "parent"}}, false)},
		},
		{
			name:   "overlapping and secondary variables preserve order",
			method: moveAccount,
			want: []*routingParam{
				newTestMultiWildcardRoutingParam("name", []*routingFieldStep{{PropertyName: "name"}}, false),
				newTestMultiWildcardRoutingParam("destination_parent", []*routingFieldStep{{PropertyName: "destinationParent"}}, false),
			},
		},
		{
			name:   "explicit routing overrides additional bindings",
			method: explicitOverride,
			want: []*routingParam{
				{
					RoutingKey: "bucket",
					Variants: []*routingParamVariant{
						{
							FieldSteps: []*routingFieldStep{{PropertyName: "destinationParent"}},
							Last:       true,
						},
					},
				},
			},
		},
		{
			name:   "explicit empty routing suppresses additional bindings",
			method: explicitEmpty,
			want:   nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.method.Codec().(*methodAnnotations)
			if diff := cmp.Diff(test.want, got.RoutingParams); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if got.HasRoutingParams() != (len(test.want) > 0) {
				t.Errorf("HasRoutingParams() = %v, want %v", got.HasRoutingParams(), len(test.want) > 0)
			}
		})
	}
}

func TestAnnotateMethod_SeeAlsoLinks(t *testing.T) {
	nameField := api.NewTestField("name").WithType(api.TypezString)
	typeField := api.NewTestField("type").WithType(api.TypezString)
	pageTokenField := api.NewTestField("page_token").WithType(api.TypezString)
	inputType := api.NewTestMessage("Request").
		WithPackage("test").
		WithFields(nameField, typeField, pageTokenField)
	responseType := api.NewTestMessage("Response").WithPackage("test")
	itemType := api.NewTestMessage("Item")
	itemsField := api.NewTestField("items").WithMessageType(itemType).WithRepeated()
	nextPageTokenField := api.NewTestField("next_page_token").WithType(api.TypezString)
	paginatedResponse := api.NewTestMessage("ListResponse").
		WithFields(itemsField, nextPageTokenField).
		WithPagination(nextPageTokenField, itemsField)
	operationType := api.NewTestMessage("Operation").WithPackage("test")
	lroResponse := api.NewTestMessage("LroResponse").WithPackage("test")
	lroMetadata := api.NewTestMessage("LroMetadata").WithPackage("test")

	plainMethod := api.NewTestMethod("Get").
		WithInput(inputType).
		WithOutput(responseType).
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").Build()).
		WithSignatures(api.NewTestMethodSignature("name", "type"))

	paginatedMethod := api.NewTestMethod("ListThings").
		WithInput(inputType).
		WithOutput(paginatedResponse).
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").Build()).
		WithPagination(pageTokenField).
		WithSignatures(api.NewTestMethodSignature("name"))

	lroMethod := api.NewTestMethod("CreateThing").
		WithInput(inputType).
		WithOutput(operationType).
		WithVerb("POST").
		WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").Build()).
		WithOperationInfo(api.NewTestOperationInfo(lroResponse.ID(), lroMetadata.ID())).
		WithSignatures(api.NewTestMethodSignature("name"))

	pollerMethod := api.NewTestMethod("GetOperation").
		WithInput(inputType).
		WithOutput(operationType).
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").Build()).
		WithIsLroPoller()

	service := api.NewTestService("TestService").
		WithPackage("test").
		WithMethods(plainMethod, paginatedMethod, lroMethod, pollerMethod)
	model := api.NewTestAPI(
		[]*api.Message{inputType, responseType, itemType, paginatedResponse, operationType, lroResponse, lroMetadata},
		nil,
		[]*api.Service{service},
	).WithPackageName("test")
	codec := newTestCodec(t, model, nil)
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name            string
		method          *api.Method
		wantSeeAlso     []string
		wantPolling     []string
		wantStorage     []string
		wantStoragePoll []string
	}{
		{
			name:        "plain with keyword method and field",
			method:      plainMethod,
			wantSeeAlso: []string{"get(name:type:)", "get(request:)"},
			wantStorage: []string{"get(request:)"},
		},
		{
			name:        "pagination with signature",
			method:      paginatedMethod,
			wantSeeAlso: []string{"listThings(request:)", "listThingsByItems(name:)", "listThingsByItems(request:)", "listThingsByItems(request:options:)"},
			wantStorage: []string{"listThings(request:)", "listThingsByItems(request:)", "listThingsByItems(request:options:)"},
		},
		{
			name:            "lro with signature",
			method:          lroMethod,
			wantSeeAlso:     []string{"createThing(request:)", "createThingPollingUntilDone(name:)", "createThingPollingUntilDone(request:)", "createThingPollingUntilDone(request:options:)"},
			wantPolling:     []string{"createThing(request:)", "createThing(request:options:)", "createThingPollingUntilDone(name:)", "createThingPollingUntilDone(request:)"},
			wantStorage:     []string{"createThing(request:)", "createThingPollingUntilDone(request:)", "createThingPollingUntilDone(request:options:)"},
			wantStoragePoll: []string{"createThing(request:)", "createThing(request:options:)", "createThingPollingUntilDone(request:)"},
		},
		{
			name:   "lro poller omitted",
			method: pollerMethod,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.method.Codec().(*methodAnnotations)
			if diff := cmp.Diff(test.wantSeeAlso, got.SeeAlsoLinks); diff != "" {
				t.Errorf("SeeAlsoLinks mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantPolling, got.PollingSeeAlsoLinks); diff != "" {
				t.Errorf("PollingSeeAlsoLinks mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantStorage, got.StorageControlSeeAlsoLinks); diff != "" {
				t.Errorf("StorageControlSeeAlsoLinks mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantStoragePoll, got.StorageControlPollingSeeAlsoLinks); diff != "" {
				t.Errorf("StorageControlPollingSeeAlsoLinks mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
