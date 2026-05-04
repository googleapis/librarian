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

package rust

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func serviceAnnotationsModel() *api.API {
	request := &api.Message{
		Name:    "Request",
		Package: "test.v1",
		ID:      ".test.v1.Request",
	}
	response := &api.Message{
		Name:    "Response",
		Package: "test.v1",
		ID:      ".test.v1.Response",
		Fields: []*api.Field{
			{
				Name:    "field",
				ID:      ".test.v1.Response.field",
				Typez:   api.TypezEnum,
				TypezID: ".test.v1.UsedEnum",
			},
		},
	}
	method := &api.Method{
		Name:         "GetResource",
		ID:           ".test.v1.ResourceService.GetResource",
		InputType:    request,
		InputTypeID:  ".test.v1.Request",
		OutputTypeID: ".test.v1.Response",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb: "GET",
					PathTemplate: (&api.PathTemplate{}).
						WithLiteral("v1").
						WithLiteral("resource"),
				},
			},
		},
	}
	emptyMethod := &api.Method{
		Name:         "DeleteResource",
		ID:           ".test.v1.ResourceService.DeleteResource",
		InputType:    request,
		InputTypeID:  ".test.v1.Request",
		OutputTypeID: ".google.protobuf.Empty",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb: "DELETE",
					PathTemplate: (&api.PathTemplate{}).
						WithLiteral("v1").
						WithLiteral("resource"),
				},
			},
		},
		ReturnsEmpty: true,
	}
	noHttpMethod := &api.Method{
		Name:         "DoAThing",
		ID:           ".test.v1.ResourceService.DoAThing",
		InputTypeID:  ".test.v1.Request",
		OutputTypeID: ".test.v1.Response",
	}
	service := &api.Service{
		Name:    "ResourceService",
		ID:      ".test.v1.ResourceService",
		Package: "test.v1",
		Methods: []*api.Method{method, emptyMethod, noHttpMethod},
	}

	usedEnum := &api.Enum{
		Name:    "UsedEnum",
		ID:      ".test.v1.UsedEnum",
		Package: "test.v1",
	}
	extraEnum := &api.Enum{
		Name:    "ExtraEnum",
		ID:      ".test.v1.ExtraEnum",
		Package: "test.v1",
	}

	model := api.NewTestAPI(
		[]*api.Message{request, response},
		[]*api.Enum{usedEnum, extraEnum},
		[]*api.Service{service})
	api.CrossReference(model)
	return model
}

func TestPathInfoAnnotations(t *testing.T) {
	binding := func(verb string) *api.PathBinding {
		return &api.PathBinding{
			Verb: verb,
			PathTemplate: (&api.PathTemplate{}).
				WithLiteral("v1").
				WithLiteral("resource"),
		}
	}

	for _, test := range []struct {
		name               string
		Bindings           []*api.PathBinding
		DefaultIdempotency string
	}{
		{"empty", []*api.PathBinding{}, "false"},
		{"GET", []*api.PathBinding{binding("GET")}, "true"},
		{"PUT", []*api.PathBinding{binding("PUT")}, "true"},
		{"DELETE", []*api.PathBinding{binding("DELETE")}, "true"},
		{"POST", []*api.PathBinding{binding("POST")}, "false"},
		{"PATCH", []*api.PathBinding{binding("PATCH")}, "false"},
		{"GET_GET", []*api.PathBinding{binding("GET"), binding("GET")}, "true"},
		{"GET_POST", []*api.PathBinding{binding("GET"), binding("POST")}, "false"},
		{"POST_POST", []*api.PathBinding{binding("POST"), binding("POST")}, "false"},
	} {
		t.Run(test.name, func(t *testing.T) {
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
			method := &api.Method{
				Name:         "GetResource",
				ID:           ".test.v1.Service.GetResource",
				InputTypeID:  ".test.v1.Request",
				OutputTypeID: ".test.v1.Response",
				PathInfo: &api.PathInfo{
					Bindings: test.Bindings,
				},
			}
			service := &api.Service{
				Name:    "ResourceService",
				ID:      ".test.v1.ResourceService",
				Package: "test.v1",
				Methods: []*api.Method{method},
			}

			model := api.NewTestAPI(
				[]*api.Message{request, response},
				[]*api.Enum{},
				[]*api.Service{service})
			api.CrossReference(model)
			codec := newTestCodec(t, libconfig.SpecProtobuf, "test.v1", map[string]string{
				"include-grpc-only-methods": "true",
			})
			annotateModel(model, codec)

			pathInfoAnn := method.PathInfo.Codec.(*pathInfoAnnotation)
			if pathInfoAnn.IsIdempotent != test.DefaultIdempotency {
				t.Errorf("fail")
			}
		})
	}
}

func TestPathBindingAnnotations(t *testing.T) {
	f_name := &api.Field{
		Name:     "name",
		JSONName: "name",
		ID:       ".test.Request.name",
		Typez:    api.TypezString,
	}

	f_project := &api.Field{
		Name:     "project",
		JSONName: "project",
		ID:       ".test.Request.project",
		Typez:    api.TypezString,
	}
	f_location := &api.Field{
		Name:     "location",
		JSONName: "location",
		ID:       ".test.Request.location",
		Typez:    api.TypezString,
	}
	f_id := &api.Field{
		Name:     "id",
		JSONName: "id",
		ID:       ".test.Request.id",
		Typez:    api.TypezUint64,
	}
	f_optional := &api.Field{
		Name:     "optional",
		JSONName: "optional",
		ID:       ".test.Request.optional",
		Typez:    api.TypezString,
		Optional: true,
	}

	// A field also of type `Request`. We want to test nested path
	// parameters, and this saves us from having to define a new
	// `api.Message`, with all of its fields.
	f_child := &api.Field{
		Name:     "child",
		JSONName: "child",
		ID:       ".test.Request.child",
		Typez:    api.TypezMessage,
		TypezID:  ".test.Request",
		Optional: true,
	}

	request := &api.Message{
		Name:    "Request",
		Package: "test",
		ID:      ".test.Request",
		Fields: []*api.Field{
			f_name,
			f_project,
			f_location,
			f_id,
			f_optional,
			f_child,
		},
	}
	response := &api.Message{
		Name:    "Response",
		Package: "test",
		ID:      ".test.Response",
	}

	b0 := &api.PathBinding{
		Verb: "POST",
		PathTemplate: (&api.PathTemplate{}).
			WithLiteral("v2").
			WithVariable(api.NewPathVariable("name").
				WithLiteral("projects").
				WithMatch().
				WithLiteral("locations").
				WithMatch()).
			WithVerb("create"),
		QueryParameters: map[string]bool{
			"id": true,
		},
	}
	want_b0 := &pathBindingAnnotation{
		PathFmt:     "/v2/{}:create",
		QueryParams: []*api.Field{f_id},
		Substitutions: []*bindingSubstitution{
			{
				FieldAccessor: "Some(&req).map(|m| &m.name).map(|s| s.as_str())",
				FieldName:     "name",
				Template:      []string{"projects", "*", "locations", "*"},
			},
		},
	}

	b1 := &api.PathBinding{
		Verb: "POST",
		PathTemplate: (&api.PathTemplate{}).
			WithLiteral("v1").
			WithLiteral("projects").
			WithVariableNamed("project").
			WithLiteral("locations").
			WithVariableNamed("location").
			WithLiteral("ids").
			WithVariableNamed("id").
			WithVerb("action"),
	}
	want_b1 := &pathBindingAnnotation{
		PathFmt: "/v1/projects/{}/locations/{}/ids/{}:action",
		Substitutions: []*bindingSubstitution{
			{
				FieldAccessor: "Some(&req).map(|m| &m.project).map(|s| s.as_str())",
				FieldName:     "project",
				Template:      []string{"*"},
			},
			{
				FieldAccessor: "Some(&req).map(|m| &m.location).map(|s| s.as_str())",
				FieldName:     "location",
				Template:      []string{"*"},
			},
			{
				FieldAccessor: "Some(&req).map(|m| &m.id)",
				FieldName:     "id",
				Template:      []string{"*"},
			},
		},
	}
	b2 := &api.PathBinding{
		Verb: "POST",
		PathTemplate: (&api.PathTemplate{}).
			WithLiteral("v1").
			WithLiteral("projects").
			WithVariableNamed("child", "project").
			WithLiteral("locations").
			WithVariableNamed("child", "location").
			WithLiteral("ids").
			WithVariableNamed("child", "id").
			WithVerb("actionOnChild"),
	}
	want_b2 := &pathBindingAnnotation{
		PathFmt: "/v1/projects/{}/locations/{}/ids/{}:actionOnChild",
		Substitutions: []*bindingSubstitution{
			{
				FieldAccessor: "Some(&req).and_then(|m| m.child.as_ref()).map(|m| &m.project).map(|s| s.as_str())",
				FieldName:     "child.project",
				Template:      []string{"*"},
			},
			{
				FieldAccessor: "Some(&req).and_then(|m| m.child.as_ref()).map(|m| &m.location).map(|s| s.as_str())",
				FieldName:     "child.location",
				Template:      []string{"*"},
			},
			{
				FieldAccessor: "Some(&req).and_then(|m| m.child.as_ref()).map(|m| &m.id)",
				FieldName:     "child.id",
				Template:      []string{"*"},
			},
		},
	}
	b3 := &api.PathBinding{
		Verb: "GET",
		PathTemplate: (&api.PathTemplate{}).
			WithLiteral("v2").
			WithLiteral("foos"),
		QueryParameters: map[string]bool{
			"name":     true,
			"optional": true,
			"child":    true,
		},
	}
	want_b3 := &pathBindingAnnotation{
		PathFmt:     "/v2/foos",
		QueryParams: []*api.Field{f_name, f_optional, f_child},
	}
	method := &api.Method{
		Name:         "DoFoo",
		ID:           ".test.Service.DoFoo",
		InputType:    request,
		InputTypeID:  ".test.Request",
		OutputTypeID: ".test.Response",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{b0, b1, b2, b3},
		},
	}
	service := &api.Service{
		Name:    "FooService",
		ID:      ".test.FooService",
		Package: "test",
		Methods: []*api.Method{method},
	}

	model := api.NewTestAPI(
		[]*api.Message{request, response},
		[]*api.Enum{},
		[]*api.Service{service})
	api.CrossReference(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{})
	annotateModel(model, codec)

	if diff := cmp.Diff(want_b0, b0.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want_b1, b1.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want_b2, b2.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(want_b3, b3.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPathBindingAnnotationsDetailedTracing(t *testing.T) {
	f_name := &api.Field{
		Name:     "name",
		JSONName: "name",
		ID:       ".test.Request.name",
		Typez:    api.TypezString,
	}
	request := &api.Message{
		Name:    "Request",
		Package: "test",
		ID:      ".test.Request",
		Fields:  []*api.Field{f_name},
	}
	response := &api.Message{
		Name:    "Response",
		Package: "test",
		ID:      ".test.Response",
	}
	binding := &api.PathBinding{
		Verb: "POST",
		PathTemplate: (&api.PathTemplate{}).
			WithLiteral("v2").
			WithVariable(api.NewPathVariable("name").
				WithLiteral("projects").
				WithMatch()).
			WithVerb("create"),
	}
	method := &api.Method{
		Name:         "DoFoo",
		ID:           ".test.Service.DoFoo",
		InputType:    request,
		InputTypeID:  ".test.Request",
		OutputTypeID: ".test.Response",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{binding},
		},
	}
	service := &api.Service{
		Name:    "FooService",
		ID:      ".test.FooService",
		Package: "test",
		Methods: []*api.Method{method},
	}
	model := api.NewTestAPI(
		[]*api.Message{request, response},
		[]*api.Enum{},
		[]*api.Service{service})
	api.CrossReference(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"detailed-tracing-attributes": "true",
	})
	annotateModel(model, codec)

	got := binding.Codec.(*pathBindingAnnotation)
	if !got.DetailedTracingAttributes {
		t.Errorf("pathBindingAnnotation.DetailedTracingAttributes = %v, want %v", got.DetailedTracingAttributes, true)
	}
}

func TestPathBindingAnnotationsStyle(t *testing.T) {
	for _, test := range []struct {
		FieldName     string
		WantFieldName string
		WantAccessor  string
	}{
		{"machine", "machine", "Some(&req).map(|m| &m.machine).map(|s| s.as_str())"},
		{"machineType", "machine_type", "Some(&req).map(|m| &m.machine_type).map(|s| s.as_str())"},
		{"machine_type", "machine_type", "Some(&req).map(|m| &m.machine_type).map(|s| s.as_str())"},
		{"type", "type", "Some(&req).map(|m| &m.r#type).map(|s| s.as_str())"},
	} {
		field := &api.Field{
			Name:     test.FieldName,
			JSONName: test.FieldName,
			ID:       fmt.Sprintf(".test.Request.%s", test.FieldName),
			Typez:    api.TypezString,
		}
		request := &api.Message{
			Name:    "Request",
			Package: "test",
			ID:      ".test.Request",
			Fields:  []*api.Field{field},
		}
		response := &api.Message{
			Name:    "Response",
			Package: "test",
			ID:      ".test.Response",
		}
		binding := &api.PathBinding{
			Verb: "GET",
			PathTemplate: (&api.PathTemplate{}).
				WithLiteral("v1").
				WithLiteral("machines").
				WithVariable(api.NewPathVariable(test.FieldName).
					WithMatch()).
				WithVerb("create"),
			QueryParameters: map[string]bool{},
		}
		wantBinding := &pathBindingAnnotation{
			PathFmt: "/v1/machines/{}:create",
			Substitutions: []*bindingSubstitution{
				{
					FieldAccessor: test.WantAccessor,
					FieldName:     test.WantFieldName,
					Template:      []string{"*"},
				},
			},
		}
		method := &api.Method{
			Name:         "Create",
			ID:           ".test.Service.Create",
			InputType:    request,
			InputTypeID:  ".test.Request",
			OutputTypeID: ".test.Response",
			PathInfo: &api.PathInfo{
				Bindings: []*api.PathBinding{binding},
			},
		}
		service := &api.Service{
			Name:    "Service",
			ID:      ".test.Service",
			Package: "test",
			Methods: []*api.Method{method},
		}
		model := api.NewTestAPI(
			[]*api.Message{request, response},
			[]*api.Enum{},
			[]*api.Service{service})
		api.CrossReference(model)
		codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{})
		annotateModel(model, codec)
		if diff := cmp.Diff(wantBinding, binding.Codec); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}

	}
}

func TestPathBindingAnnotationsErrors(t *testing.T) {
	field := &api.Field{
		Name:     "field",
		JSONName: "field",
		ID:       ".test.Request.field",
		Typez:    api.TypezString,
	}
	request := &api.Message{
		Name:    "Request",
		Package: "test",
		ID:      ".test.Request",
		Fields:  []*api.Field{field},
	}
	method := &api.Method{
		Name:         "Create",
		ID:           ".test.Service.Create",
		InputType:    request,
		InputTypeID:  ".test.Request",
		OutputTypeID: ".test.Response",
	}
	if got, err := makeAccessors([]string{"not-a-field-name"}, method); err == nil {
		t.Errorf("expected an error in makeAccessors() for an invalid field name, got=%v", got)
	}
}

func TestPathTemplateGeneration(t *testing.T) {
	for _, test := range []struct {
		name    string
		binding *pathBindingAnnotation
		want    string
	}{
		{
			name: "Simple Literal",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/things",
			},
			want: "/v1/things",
		},
		{
			name: "Single Variable",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/things/{}",
				Substitutions: []*bindingSubstitution{
					{FieldName: "thing_id"},
				},
			},
			want: "/v1/things/{thing_id}",
		},
		{
			name: "Multiple Variables",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/projects/{}/locations/{}",
				Substitutions: []*bindingSubstitution{
					{FieldName: "project"},
					{FieldName: "location"},
				},
			},
			want: "/v1/projects/{project}/locations/{location}",
		},
		{
			name: "Variable with Complex Segment Match",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/{}/databases",
				Substitutions: []*bindingSubstitution{
					{FieldName: "name"},
				},
			},
			want: "/v1/{name}/databases",
		},
		{
			name: "Variable Capturing Remaining Path",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/objects/{}",
				Substitutions: []*bindingSubstitution{
					{FieldName: "object"},
				},
			},
			want: "/v1/objects/{object}",
		},
		{
			name: "Top-Level Single Wildcard",
			binding: &pathBindingAnnotation{
				PathFmt: "/{}",
				Substitutions: []*bindingSubstitution{
					{FieldName: "field"},
				},
			},
			want: "/{field}",
		},
		{
			name: "Path with Custom Verb",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/things/{}:customVerb",
				Substitutions: []*bindingSubstitution{
					{FieldName: "thing_id"},
				},
			},
			want: "/v1/things/{thing_id}:customVerb",
		},
		{
			name: "Nested fields",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/projects/{}/locations/{}/ids/{}:actionOnChild",
				Substitutions: []*bindingSubstitution{
					{FieldName: "child.project"},
					{FieldName: "child.location"},
					{FieldName: "child.id"},
				},
			},
			want: "/v1/projects/{child.project}/locations/{child.location}/ids/{child.id}:actionOnChild",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.binding.PathTemplate(); got != test.want {
				t.Errorf("PathTemplate() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestBindingSubstitutionTemplates(t *testing.T) {
	b := bindingSubstitution{
		Template: []string{"projects", "*", "locations", "*", "**"},
	}

	got := b.TemplateAsString()
	want := "projects/*/locations/*/**"

	if want != got {
		t.Errorf("TemplateAsString() failed. want=%q, got=%q", want, got)
	}

	got = b.TemplateAsArray()
	want = `&[Segment::Literal("projects/"), Segment::SingleWildcard, Segment::Literal("/locations/"), Segment::SingleWildcard, Segment::TrailingMultiWildcard]`

	if want != got {
		t.Errorf("TemplateAsArray() failed. want=`%s`, got=`%s`", want, got)
	}
}
