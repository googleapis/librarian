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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
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
					WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("operations")).
					WithDocumentation("Gets a thing.\n\nTest multiple comment lines.\n")
			},
			wantFunc: func(*api.Field) *methodAnnotations {
				return &methodAnnotations{
					Name:             "getOperation",
					PathExpression:   "/v1/operations",
					DocLines:         []string{"Gets a thing.", "", "Test multiple comment lines.", ""},
					HTTPMethod:       "GET",
					HasBody:          false,
					ReturnType:       "Test.Response",
					ResponseEncoding: "json;enum-encoding=int",
				}
			},
		},
		{
			name: "POST request with body field",
			methodFunc: func() *api.Method {
				return api.NewTestMethod("CreateKey").
					WithVerb("POST").
					WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("keys")).
					WithBodyFieldPath("key")
			},
			wantFunc: func(*api.Field) *methodAnnotations {
				return &methodAnnotations{
					Name:             "createKey",
					PathExpression:   "/v1/keys",
					HTTPMethod:       "POST",
					HasBody:          true,
					IsBodyWildcard:   false,
					BodyField:        "key",
					ReturnType:       "Test.Response",
					ResponseEncoding: "json;enum-encoding=int",
				}
			},
		},
		{
			name: "POST request with wildcard body",
			methodFunc: func() *api.Method {
				return api.NewTestMethod("UploadData").
					WithVerb("POST").
					WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("data")).
					WithBodyFieldPath("*")
			},
			wantFunc: func(*api.Field) *methodAnnotations {
				return &methodAnnotations{
					Name:             "uploadData",
					PathExpression:   "/v1/data",
					HTTPMethod:       "POST",
					HasBody:          true,
					IsBodyWildcard:   true,
					ReturnType:       "Test.Response",
					ResponseEncoding: "json;enum-encoding=int",
				}
			},
		},
		{
			name: "List request",
			methodFunc: func() *api.Method {
				return api.NewTestMethod("ListThings").
					WithVerb("GET").
					WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("things")).
					WithQueryParameters(map[string]bool{"key": true}).
					WithDocumentation("Lists things.")
			},
			wantFunc: func(keyField *api.Field) *methodAnnotations {
				return &methodAnnotations{
					Name:             "listThings",
					PathExpression:   "/v1/things",
					DocLines:         []string{"Lists things."},
					HTTPMethod:       "GET",
					HasBody:          false,
					QueryParams:      []*api.Field{keyField},
					ReturnType:       "Test.Response",
					ResponseEncoding: "json;enum-encoding=int",
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
			got := method.Codec.(*methodAnnotations)
			want := test.wantFunc(keyField)
			if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(methodAnnotations{}, "PathBindings", "HasMultipleBindings")); diff != "" {
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
				api.NewTestPathBinding("POST", (&api.PathTemplate{}).WithLiteral("v1").WithVariableNamed("parent")),
				api.NewTestPathBinding("POST", (&api.PathTemplate{}).WithLiteral("v1").WithVariableNamed("secret", "name")),
			},
			want:                  [][]string{{"parent"}, {"secret.name"}},
			wantHasOmitted:        true,
			wantOmittedExpression: []string{`["parent"]`, `["secret.name"]`},
		},
		{
			name:          "wildcard body without path variables",
			bodyFieldPath: "*",
			bindings: []*api.PathBinding{
				api.NewTestPathBinding("POST", (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("secrets")),
			},
			want:                  [][]string{nil},
			wantHasOmitted:        false,
			wantOmittedExpression: []string{`[]`},
		},
		{
			name:          "named body field",
			bodyFieldPath: "secret",
			bindings: []*api.PathBinding{
				api.NewTestPathBinding("POST", (&api.PathTemplate{}).WithLiteral("v1").WithVariableNamed("parent")),
			},
			want:                  [][]string{nil},
			wantHasOmitted:        false,
			wantOmittedExpression: []string{`[]`},
		},
		{
			name:          "no body",
			bodyFieldPath: "",
			bindings: []*api.PathBinding{
				api.NewTestPathBinding("GET", (&api.PathTemplate{}).WithLiteral("v1").WithVariableNamed("parent")),
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
			got := method.Codec.(*methodAnnotations)
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
				WithVerb("GET").WithPathTemplate(&api.PathTemplate{}).
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

			want := &methodAnnotations{
				Name:             test.wantName,
				DocLines:         []string{"Test documentation."},
				PathExpression:   "/",
				HTTPMethod:       "GET",
				ReturnType:       "Test.Response",
				ResponseEncoding: "json;enum-encoding=int",
			}

			if diff := cmp.Diff(want, method.Codec, cmpopts.IgnoreFields(methodAnnotations{}, "PathBindings", "HasMultipleBindings")); diff != "" {
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
		WithPathTemplate(&api.PathTemplate{})
	service := api.NewTestService("TestService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{}, nil, []*api.Service{service})
	model.PackageName = "google.cloud.test.v1"
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

	if inputMessage.Codec == nil {
		t.Error("expected input message to be annotated")
	}
	if outputMessage.Codec == nil {
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
		WithPathTemplate(&api.PathTemplate{}).
		WithPagination(pageTokenField)

	service := api.NewTestService("TestService").WithMethods(method)

	model := api.NewTestAPI([]*api.Message{inputType, outputType, itemType}, nil, []*api.Service{service})
	model.PackageName = "test"
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	codec := newTestCodec(t, model, nil)

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	// Verify method annotations
	gotMethod := method.Codec.(*methodAnnotations)
	wantMethod := &methodAnnotations{
		Name:           "listItems",
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
	gotRequest := inputType.Codec.(*messageAnnotations)
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
	gotResponse := outputType.Codec.(*messageAnnotations)
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
		WithPathTemplate(&api.PathTemplate{}).
		WithOperationInfo(&api.OperationInfo{
			ResponseTypeID: lroResponseType.ID,
			MetadataTypeID: lroMetadataType.ID,
		})

	service := api.NewTestService("TestService").WithMethods(method)

	model := api.NewTestAPI([]*api.Message{inputType, outputType, lroResponseType, lroMetadataType}, nil, []*api.Service{service})
	model.PackageName = "test"
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	codec := newTestCodec(t, model, nil)

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	gotMethod := method.Codec.(*methodAnnotations)
	wantMethod := &methodAnnotations{
		Name:           "lroMethod",
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
		WithPathTemplate(&api.PathTemplate{}).
		WithOperationInfo(&api.OperationInfo{
			ResponseTypeID: ".google.protobuf.Empty",
			MetadataTypeID: lroMetadataType.ID,
		})

	service := api.NewTestService("TestService").WithMethods(method)

	model := api.NewTestAPI([]*api.Message{inputType, outputType, lroMetadataType}, nil, []*api.Service{service})
	model.PackageName = "test"
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	codec := newTestCodec(t, model, nil)

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	gotMethod := method.Codec.(*methodAnnotations)
	wantMethod := &methodAnnotations{
		Name:           "lroMethod",
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
			api.NewTestPathBinding("GET", (&api.PathTemplate{}).
				WithLiteral("v1").
				WithVariableNamed("name")),
			api.NewTestPathBinding("POST", (&api.PathTemplate{}).
				WithLiteral("v1").
				WithVariableNamed("project").
				WithLiteral("resources")),
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
	got := method.Codec.(*methodAnnotations)
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
