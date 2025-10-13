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

package rust

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/internal/api"
)

func TestAnnotateMethodNames(t *testing.T) {
	model := annotateMethodModel(t)
	err := api.CrossReference(model)
	if err != nil {
		t.Fatal(err)
	}
	codec, err := newCodec("protobuf", map[string]string{
		"include-grpc-only-methods": "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = annotateModel(model, codec)

	for _, test := range []struct {
		MethodID string
		Want     *methodAnnotation
	}{
		{
			MethodID: ".test.v1.ResourceService.move",
			Want: &methodAnnotation{
				Name:                "r#move",
				BuilderName:         "Move",
				Body:                "None::<gaxi::http::NoBody>",
				ServiceNameToPascal: "ResourceService",
				ServiceNameToCamel:  "resourceService",
				ServiceNameToSnake:  "resource_service",
				ReturnType:          "crate::model::Response",
			},
		},
		{
			MethodID: ".test.v1.ResourceService.Delete",
			Want: &methodAnnotation{
				Name:                "delete",
				BuilderName:         "Delete",
				Body:                "None::<gaxi::http::NoBody>",
				ServiceNameToPascal: "ResourceService",
				ServiceNameToCamel:  "resourceService",
				ServiceNameToSnake:  "resource_service",
				ReturnType:          "()",
			},
		},
		{
			MethodID: ".test.v1.ResourceService.Self",
			Want: &methodAnnotation{
				Name:                "r#self",
				BuilderName:         "r#Self",
				Body:                "None::<gaxi::http::NoBody>",
				ServiceNameToPascal: "ResourceService",
				ServiceNameToCamel:  "resourceService",
				ServiceNameToSnake:  "resource_service",
				ReturnType:          "crate::model::Response",
			},
		},
	} {
		gotMethod, ok := model.State.MethodByID[test.MethodID]
		if !ok {
			t.Errorf("missing method %s", test.MethodID)
			continue
		}
		got := gotMethod.Codec.(*methodAnnotation)
		if diff := cmp.Diff(test.Want, got, cmpopts.IgnoreFields(methodAnnotation{}, "PathInfo", "SystemParameters")); diff != "" {
			t.Errorf("mismatch (-want, +got):\n%s", diff)
		}
	}
}

func TestAnnotateDiscoveryAnnotations(t *testing.T) {
	model := annotateMethodModel(t)
	err := api.CrossReference(model)
	if err != nil {
		t.Fatal(err)
	}
	codec, err := newCodec("protobuf", map[string]string{
		"include-grpc-only-methods": "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = annotateModel(model, codec)

	methodID := ".test.v1.ResourceService.Delete"
	gotMethod, ok := model.State.MethodByID[methodID]
	if !ok {
		t.Fatalf("missing method %s", methodID)
	}
	got := gotMethod.DiscoveryLro.Codec.(*discoveryLroAnnotations)
	want := &discoveryLroAnnotations{
		MethodName: "delete",
		ReturnType: "()",
		PollingPathParameters: []discoveryLroPathParameter{
			{Name: "project", SetterName: "project"},
			{Name: "zone", SetterName: "zone"},
			{Name: "r#type", SetterName: "type"},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func annotateMethodModel(t *testing.T) *api.API {
	t.Helper()
	request := &api.Message{
		Name:    "Request",
		Package: "test.v1",
		ID:      ".test.v1.Request",
	}
	response := &api.Message{
		Name:    "Response",
		Package: "test.v1",
		ID:      ".test.v1.Response",
	}
	methodMove := &api.Method{
		Name:         "move",
		ID:           ".test.v1.ResourceService.move",
		InputType:    request,
		InputTypeID:  ".test.v1.Request",
		OutputTypeID: ".test.v1.Response",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb:         "POST",
					PathTemplate: api.NewPathTemplate(),
				},
			},
		},
	}
	methodDelete := &api.Method{
		Name:         "Delete",
		ID:           ".test.v1.ResourceService.Delete",
		InputType:    request,
		InputTypeID:  ".test.v1.Request",
		OutputTypeID: ".google.protobuf.Empty",
		ReturnsEmpty: true,
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb: "DELETE",
					PathTemplate: api.NewPathTemplate().
						WithLiteral("projects").
						WithVariableNamed("project").
						WithLiteral("zones").
						WithVariableNamed("zone").
						// This is unlikely, but want to test variables that
						// are reserved words.
						WithLiteral("types").
						WithVariableNamed("type"),
				},
			},
		},
		DiscoveryLro: &api.DiscoveryLro{
			PollingPathParameters: []string{"project", "zone", "type"},
		},
	}
	methodSelf := &api.Method{
		Name:         "Self",
		ID:           ".test.v1.ResourceService.Self",
		InputType:    request,
		InputTypeID:  ".test.v1.Request",
		OutputTypeID: ".test.v1.Response",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb:         "GET",
					PathTemplate: api.NewPathTemplate(),
				},
			},
		},
	}
	service := &api.Service{
		Name:    "ResourceService",
		ID:      ".test.v1.ResourceService",
		Package: "test.v1",
		Methods: []*api.Method{methodMove, methodDelete, methodSelf},
	}

	model := api.NewTestAPI(
		[]*api.Message{request, response},
		[]*api.Enum{},
		[]*api.Service{service})
	api.CrossReference(model)
	return model
}
