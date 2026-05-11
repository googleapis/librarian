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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateMethod(t *testing.T) {
	keyField := &api.Field{Name: "key", ID: ".test.Request.key", Typez: api.TypezString}
	inputType := &api.Message{
		Name:   "Request",
		ID:     ".test.Request",
		Fields: []*api.Field{keyField},
	}
	keyField.Parent = inputType
	outputType := &api.Message{
		Name: "Response",
		ID:   ".test.Response",
		Fields: []*api.Field{
			{Name: "value", ID: ".test.Request.value", Typez: api.TypezString},
		},
	}
	outputType.Fields[0].Parent = outputType
	for _, test := range []struct {
		name   string
		method *api.Method
		want   *methodAnnotations
	}{
		{
			name: "GET request",
			method: &api.Method{
				Name:          "GetOperation",
				Documentation: "Gets a thing.\n\nTest multiple comment lines.\n",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb:         "GET",
							PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("operations"),
						},
					},
				},
			},
			want: &methodAnnotations{
				Name:           "getOperation",
				PathExpression: "/v1/operations",
				DocLines:       []string{"Gets a thing.", "", "Test multiple comment lines.", ""},
				HTTPMethod:     "GET",
				HasBody:        false,
			},
		},
		{
			name: "POST request with body field",
			method: &api.Method{
				Name: "CreateKey",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb:         "POST",
							PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("keys"),
						},
					},
					BodyFieldPath: "key",
				},
			},
			want: &methodAnnotations{
				Name:           "createKey",
				PathExpression: "/v1/keys",
				HTTPMethod:     "POST",
				HasBody:        true,
				IsBodyWildcard: false,
				BodyField:      "key",
			},
		},
		{
			name: "POST request with wildcard body",
			method: &api.Method{
				Name: "UploadData",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb:         "POST",
							PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("data"),
						},
					},
					BodyFieldPath: "*",
				},
			},
			want: &methodAnnotations{
				Name:           "uploadData",
				PathExpression: "/v1/data",
				HTTPMethod:     "POST",
				HasBody:        true,
				IsBodyWildcard: true,
			},
		},
		{
			name: "List request",
			method: &api.Method{
				Name:          "ListThings",
				Documentation: "Lists things.",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb:            "GET",
							PathTemplate:    (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("things"),
							QueryParameters: map[string]bool{"key": true},
						},
					},
				},
			},
			want: &methodAnnotations{
				Name:           "listThings",
				PathExpression: "/v1/things",
				DocLines:       []string{"Lists things."},
				HTTPMethod:     "GET",
				HasBody:        false,
				QueryParams:    []*api.Field{keyField},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.method.InputType = inputType
			test.method.InputTypeID = inputType.ID
			test.method.OutputType = outputType
			test.method.OutputTypeID = outputType.ID
			service := &api.Service{
				Name:    "TestService",
				ID:      ".test.TestService",
				Package: "test",
				Methods: []*api.Method{test.method},
			}
			model := api.NewTestAPI([]*api.Message{inputType, outputType}, nil, []*api.Service{service})
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			got := test.method.Codec.(*methodAnnotations)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
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
			inputType := &api.Message{
				Name: "Request",
				ID:   ".test.Request",
				Fields: []*api.Field{
					{Name: "key", ID: ".test.Request.key", Typez: api.TypezString},
				},
			}
			outputType := &api.Message{
				Name: "Response",
				ID:   ".test.Response",
				Fields: []*api.Field{
					{Name: "value", ID: ".test.Request.value", Typez: api.TypezString},
				},
			}
			method := &api.Method{
				Name:          test.methodName,
				Documentation: "Test documentation.",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{Verb: "GET", PathTemplate: &api.PathTemplate{}}},
				},
				InputTypeID:  inputType.ID,
				InputType:    inputType,
				OutputTypeID: outputType.ID,
				OutputType:   outputType,
			}
			service := &api.Service{
				Name:    "TestService",
				Methods: []*api.Method{method},
			}
			model := api.NewTestAPI([]*api.Message{inputType, outputType}, nil, []*api.Service{service})
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			codec := newTestCodec(t, model, nil)

			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			want := &methodAnnotations{
				Name:           test.wantName,
				DocLines:       []string{"Test documentation."},
				PathExpression: "/",
				HTTPMethod:     "GET",
			}

			if diff := cmp.Diff(want, method.Codec); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateMethod_WithExternalMessages(t *testing.T) {
	inputMessage := &api.Message{
		Name:    "InputMessage",
		Package: "google.cloud.external.v1",
		ID:      ".google.cloud.external.v1.InputMessage",
	}
	outputMessage := &api.Message{
		Name:    "OutputMessage",
		Package: "google.cloud.external.v1",
		ID:      ".google.cloud.external.v1.OutputMessage",
	}
	method := &api.Method{
		Name:         "TestMethod",
		InputType:    inputMessage,
		InputTypeID:  inputMessage.ID,
		OutputType:   outputMessage,
		OutputTypeID: outputMessage.ID,
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{{Verb: "POST", PathTemplate: &api.PathTemplate{}}},
		},
	}
	service := &api.Service{
		Name:    "TestService",
		Methods: []*api.Method{method},
	}
	model := api.NewTestAPI([]*api.Message{}, nil, []*api.Service{service})
	model.PackageName = "google.cloud.test.v1"
	model.AddMessage(inputMessage)
	model.AddMessage(outputMessage)
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	codec := newTestCodec(t, model, nil)

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
	pageSizeField := &api.Field{Name: "page_size", JSONName: "pageSize", Typez: api.TypezInt32}
	pageTokenField := &api.Field{Name: "page_token", JSONName: "pageToken", Typez: api.TypezString}
	inputType := &api.Message{
		Name:    "ListRequest",
		Package: "test",
		ID:      ".test.ListRequest",
		Fields:  []*api.Field{pageSizeField, pageTokenField},
	}
	pageSizeField.Parent = inputType
	pageTokenField.Parent = inputType

	itemField := &api.Field{Name: "items", JSONName: "items", Typez: api.TypezMessage, TypezID: ".test.Item", Repeated: true}
	nextPageTokenField := &api.Field{Name: "next_page_token", JSONName: "nextPageToken", Typez: api.TypezString}
	outputType := &api.Message{
		Name:    "ListResponse",
		Package: "test",
		ID:      ".test.ListResponse",
		Fields:  []*api.Field{itemField, nextPageTokenField},
		Pagination: &api.PaginationInfo{
			NextPageToken: nextPageTokenField,
			PageableItem:  itemField,
		},
	}
	itemField.Parent = outputType
	nextPageTokenField.Parent = outputType

	itemType := &api.Message{
		Name:    "Item",
		Package: "test",
		ID:      ".test.Item",
	}

	method := &api.Method{
		Name:         "ListItems",
		InputTypeID:  inputType.ID,
		InputType:    inputType,
		OutputTypeID: outputType.ID,
		OutputType:   outputType,
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{{Verb: "GET", PathTemplate: &api.PathTemplate{}}},
		},
		Pagination: pageTokenField,
	}

	service := &api.Service{
		Name:    "TestService",
		ID:      ".test.TestService",
		Package: "test",
		Methods: []*api.Method{method},
	}

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
	}
	if diff := cmp.Diff(wantMethod, gotMethod); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}

	// Verify request message annotations
	gotRequest := inputType.Codec.(*messageAnnotations)
	wantRequest := &messageAnnotations{
		Name:    "ListRequest",
		TypeURL: "type.googleapis.com/test.ListRequest",
	}
	if diff := cmp.Diff(wantRequest, gotRequest, cmpopts.IgnoreFields(messageAnnotations{}, "Model")); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}

	// Verify response message annotations
	gotResponse := outputType.Codec.(*messageAnnotations)
	wantResponse := &messageAnnotations{
		Name:                "ListResponse",
		TypeURL:             "type.googleapis.com/test.ListResponse",
		IsPaginatedResponse: true,
		PageableItemField:   "items",
		PageableItemType:    "Item",
		ImportsGax:          true,
	}
	if diff := cmp.Diff(wantResponse, gotResponse, cmpopts.IgnoreFields(messageAnnotations{}, "Model")); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}
