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
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateMethodNames(t *testing.T) {
	model := annotateMethodModel(t)
	err := api.CrossReference(model)
	if err != nil {
		t.Fatal(err)
	}
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"include-grpc-only-methods": "true",
	})
	_, err = annotateModel(model, codec)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		MethodID string
		Want     *methodAnnotation
	}{
		{
			MethodID: ".test.v1.ResourceService.move",
			Want: &methodAnnotation{
				Name:                "r#move",
				NameNoMangling:      "move",
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
				NameNoMangling:      "delete",
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
				NameNoMangling:      "self",
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
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"include-grpc-only-methods": "true",
	})
	_, err = annotateModel(model, codec)
	if err != nil {
		t.Fatal(err)
	}

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

func TestAnnotateMethodAPIVersion(t *testing.T) {
	model := annotateMethodModel(t)
	err := api.CrossReference(model)
	if err != nil {
		t.Fatal(err)
	}

	// Inject an APIVersion to the existing model.
	methodID := ".test.v1.ResourceService.Delete"
	gotMethod, ok := model.State.MethodByID[methodID]
	if !ok {
		t.Fatalf("missing method %s", methodID)
	}
	gotMethod.APIVersion = "v1_20260205"

	codec := newTestCodec(t, libconfig.SpecDiscovery, "", map[string]string{})
	_, err = annotateModel(model, codec)
	if err != nil {
		t.Fatal(err)
	}

	got := gotMethod.Codec.(*methodAnnotation)
	want := []systemParameter{
		{Name: "$alt", Value: "json"},
		{Name: "$apiVersion", Value: "v1_20260205"},
	}
	if diff := cmp.Diff(want, got.SystemParameters); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestAnnotateMethodInternalBuilders(t *testing.T) {
	model := annotateMethodModel(t)
	err := api.CrossReference(model)
	if err != nil {
		t.Fatal(err)
	}

	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"internal-builders": "true",
	})
	_, err = annotateModel(model, codec)
	if err != nil {
		t.Fatal(err)
	}

	methodID := ".test.v1.ResourceService.Delete"
	gotMethod, ok := model.State.MethodByID[methodID]
	if !ok {
		t.Fatalf("missing method %s", methodID)
	}
	got := gotMethod.Codec.(*methodAnnotation)
	if !got.InternalBuilders {
		t.Errorf("expected InternalBuilders to be true for method %s", methodID)
	}
	if got.BuilderVisibility() != "pub(crate)" {
		t.Errorf("mismatch in BuilderVisibility, want=pub(crate), got=%s", got.BuilderVisibility())
	}
}

func annotateMethodModel(t *testing.T) *api.API {
	t.Helper()
	request := &api.Message{
		Name:    "Request",
		Package: "test.v1",
		ID:      ".test.v1.Request",
		Fields: []*api.Field{
			{Name: "project", ID: ".test.v1.Request.project", Typez: api.STRING_TYPE},
			{Name: "zone", ID: ".test.v1.Request.zone", Typez: api.STRING_TYPE},
			{Name: "type", ID: ".test.v1.Request.type", Typez: api.STRING_TYPE},
		},
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

func TestAnnotateMethodResourceNameTemplate(t *testing.T) {
	model := annotateMethodModel(t)
	err := api.CrossReference(model)
	if err != nil {
		t.Fatal(err)
	}

	// Helper to inject TargetResource
	injectTargetResource := func(methodID string, template string, fields [][]string) {
		m, ok := model.State.MethodByID[methodID]
		if !ok {
			t.Fatalf("missing method %s", methodID)
		}
		if m.PathInfo != nil && len(m.PathInfo.Bindings) > 0 {
			m.PathInfo.Bindings[0].PathTemplate = api.NewPathTemplate().
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("zones").
				WithVariableNamed("zone").
				WithLiteral("types").
				WithVariableNamed("type")
			m.PathInfo.Bindings[0].TargetResource = &api.TargetResource{
				Template:   template,
				FieldPaths: fields,
			}
		}
	}

	// Setup: Inject resource for the "move" method
	injectTargetResource(".test.v1.ResourceService.move", "//Test.googleapis.com/projects/{project}/zones/{zone}/types/{type}", [][]string{
		{"project"},
		{"zone"},
		{"type"},
	})

	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{})
	_, err = annotateModel(model, codec)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		id   string
		want *methodAnnotation
	}{
		{
			name: "WithTargetResource",
			id:   ".test.v1.ResourceService.move",
			want: &methodAnnotation{
				ResourceNameTemplate: "//Test.googleapis.com/projects/{}/zones/{}/types/{}",
				ResourceNameArgs: []string{
					"Some(&req).map(|m| &m.project).map(|s| s.as_str()).unwrap_or(\"\")",
					"Some(&req).map(|m| &m.zone).map(|s| s.as_str()).unwrap_or(\"\")",
					"Some(&req).map(|m| &m.r#type).map(|s| s.as_str()).unwrap_or(\"\")",
				},
				HasResourceNameGeneration: true,
			},
		},
		{
			name: "WithoutTargetResource",
			id:   ".test.v1.ResourceService.Delete",
			want: &methodAnnotation{
				HasResourceNameGeneration: false,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, ok := model.State.MethodByID[tc.id]
			if !ok {
				t.Fatalf("missing method %s", tc.id)
			}
			got := m.Codec.(*methodAnnotation)
			if diff := cmp.Diff(tc.want, got, cmpopts.IgnoreFields(methodAnnotation{},
				"Name", "NameNoMangling", "BuilderName", "Body", "DocLines",
				"ServiceNameToPascal", "ServiceNameToCamel", "ServiceNameToSnake",
				"SystemParameters", "ReturnType", "PathInfo", "Attributes",
				"RoutingRequired", "DetailedTracingAttributes", "ResourceNameFields",
				"HasResourceNameFields", "InternalBuilders")); diff != "" {
				t.Errorf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}
