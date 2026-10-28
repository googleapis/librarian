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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func serviceAnnotationsModel() *api.API {
	usedEnum := api.NewTestEnum("UsedEnum").WithPackage("test.v1")
	extraEnum := api.NewTestEnum("ExtraEnum").WithPackage("test.v1")

	request := api.NewTestMessage("Request").WithPackage("test.v1")
	response := api.NewTestMessage("Response").
		WithPackage("test.v1").
		WithFields(
			api.NewTestField("field").
				WithEnumType(usedEnum),
		)
	method := api.NewTestMethod("GetResource").
		WithInput(request).
		WithOutput(response).
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("v1").
			WithLiteral("resource"))
	emptyMsg := api.NewTestMessage("Empty").WithID(api.WktEmptyID)
	emptyMethod := api.NewTestMethod("DeleteResource").
		WithInput(request).
		WithOutput(emptyMsg).
		WithVerb("DELETE").
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("v1").
			WithLiteral("resource")).
		ReturnEmpty()

	noHttpMethod := api.NewTestMethod("DoAThing").
		WithInput(request).
		WithOutput(response).
		WithPathInfo(nil)

	service := api.NewTestService("ResourceService").
		WithPackage("test.v1").
		WithMethods(method, emptyMethod, noHttpMethod)

	model := api.NewTestAPI(
		[]*api.Message{request, response},
		[]*api.Enum{usedEnum, extraEnum},
		[]*api.Service{service})
	api.CrossReference(model)
	return model
}

func TestPathInfoAnnotations(t *testing.T) {
	binding := func(verb string) *api.PathBinding {
		return api.NewTestPathBinding(verb, (&api.PathTemplate{}).
			WithLiteral("v1").
			WithLiteral("resource"))
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
			request := api.NewTestMessage("Request").WithPackage("test.v1")
			response := api.NewTestMessage("Response").WithPackage("test.v1")
			method := api.NewTestMethod("GetResource").
				WithInput(request).
				WithOutput(response).
				WithBindings(test.Bindings...)
			service := api.NewTestService("ResourceService").
				WithPackage("test.v1").
				WithMethods(method)

			model := api.NewTestAPI(
				[]*api.Message{request, response},
				nil,
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
	request := api.NewTestMessage("Request")

	fName := api.NewTestField("name").WithType(api.TypezString)
	fProject := api.NewTestField("project").WithType(api.TypezString)
	fLocation := api.NewTestField("location").WithType(api.TypezString)
	fID := api.NewTestField("id").WithType(api.TypezUint64)
	fOptional := api.NewTestField("optional").WithType(api.TypezString).WithOptional()

	// A field also of type `Request`. We want to test nested path
	// parameters, and this saves us from having to define a new
	// `api.Message`, with all of its fields.
	fChild := api.NewTestField("child").
		WithMessageType(request).
		WithOptional()

	fOneof := api.NewTestField("oneofField").WithType(api.TypezString)
	oneof := api.NewTestOneOf("oneofField").WithFields(fOneof)

	request.WithFields(
		fName,
		fProject,
		fLocation,
		fID,
		fOptional,
		fChild,
	).WithOneOfs(oneof)
	response := api.NewTestMessage("Response")

	b0 := api.NewTestPathBinding("POST", (&api.PathTemplate{}).
		WithLiteral("v2").
		WithVariable(api.NewPathVariable("name").
			WithLiteral("projects").
			WithMatch().
			WithLiteral("locations").
			WithMatch()).
		WithVerb("create")).
		WithQueryParameters(map[string]bool{
			"id": true,
		})
	wantB0 := &pathBindingAnnotation{
		PathFmt:     "/v2/{}:create",
		QueryParams: []*api.Field{fID},
		Substitutions: []*bindingSubstitution{
			{
				FieldAccessor:  "Some(&req).map(|m| &m.name).map(|s| s.as_str())",
				PathExtraction: &PathExtraction{FieldTake: "Some(&mut req).map(|m| std::mem::take(&mut m.name))"},
				FieldName:      "name",
				Template:       []string{"projects", "*", "locations", "*"},
			},
		},
	}

	b1 := api.NewTestPathBinding("POST", (&api.PathTemplate{}).
		WithLiteral("v1").
		WithLiteral("projects").
		WithVariableNamed("project").
		WithLiteral("locations").
		WithVariableNamed("location").
		WithLiteral("ids").
		WithVariableNamed("id").
		WithVerb("action"))
	wantB1 := &pathBindingAnnotation{
		PathFmt: "/v1/projects/{}/locations/{}/ids/{}:action",
		Substitutions: []*bindingSubstitution{
			{
				FieldAccessor:  "Some(&req).map(|m| &m.project).map(|s| s.as_str())",
				PathExtraction: &PathExtraction{FieldTake: "Some(&mut req).map(|m| std::mem::take(&mut m.project))"},
				FieldName:      "project",
				Template:       []string{"*"},
			},
			{
				FieldAccessor:  "Some(&req).map(|m| &m.location).map(|s| s.as_str())",
				PathExtraction: &PathExtraction{FieldTake: "Some(&mut req).map(|m| std::mem::take(&mut m.location))"},
				FieldName:      "location",
				Template:       []string{"*"},
			},
			{
				FieldAccessor:  "Some(&req).map(|m| &m.id)",
				PathExtraction: &PathExtraction{FieldTake: "Some(&mut req).map(|m| std::mem::take(&mut m.id))"},
				FieldName:      "id",
				Template:       []string{"*"},
			},
		},
	}
	b2 := api.NewTestPathBinding("POST", (&api.PathTemplate{}).
		WithLiteral("v1").
		WithLiteral("projects").
		WithVariableNamed("child", "project").
		WithLiteral("locations").
		WithVariableNamed("child", "location").
		WithLiteral("ids").
		WithVariableNamed("child", "id").
		WithVerb("actionOnChild"))
	wantB2 := &pathBindingAnnotation{
		PathFmt: "/v1/projects/{}/locations/{}/ids/{}:actionOnChild",
		Substitutions: []*bindingSubstitution{
			{
				FieldAccessor:  "Some(&req).and_then(|m| m.child.as_ref()).map(|m| &m.project).map(|s| s.as_str())",
				PathExtraction: &PathExtraction{FieldTake: "Some(&mut req).and_then(|m| m.child.as_mut()).map(|m| std::mem::take(&mut m.project))"},
				FieldName:      "child.project",
				Template:       []string{"*"},
			},
			{
				FieldAccessor:  "Some(&req).and_then(|m| m.child.as_ref()).map(|m| &m.location).map(|s| s.as_str())",
				PathExtraction: &PathExtraction{FieldTake: "Some(&mut req).and_then(|m| m.child.as_mut()).map(|m| std::mem::take(&mut m.location))"},
				FieldName:      "child.location",
				Template:       []string{"*"},
			},
			{
				FieldAccessor:  "Some(&req).and_then(|m| m.child.as_ref()).map(|m| &m.id)",
				PathExtraction: &PathExtraction{FieldTake: "Some(&mut req).and_then(|m| m.child.as_mut()).map(|m| std::mem::take(&mut m.id))"},
				FieldName:      "child.id",
				Template:       []string{"*"},
			},
		},
	}
	b3 := api.NewTestPathBinding("GET", (&api.PathTemplate{}).
		WithLiteral("v2").
		WithLiteral("foos")).
		WithQueryParameters(map[string]bool{
			"name":     true,
			"optional": true,
			"child":    true,
		})
	wantB3 := &pathBindingAnnotation{
		PathFmt:     "/v2/foos",
		QueryParams: []*api.Field{fName, fOptional, fChild},
	}
	b4 := api.NewTestPathBinding("POST", (&api.PathTemplate{}).
		WithLiteral("v1").
		WithLiteral("projects").
		WithVariableNamed("oneofField").
		WithVerb("actionOnOneof"))
	wantB4 := &pathBindingAnnotation{
		PathFmt: "/v1/projects/{}:actionOnOneof",
		Substitutions: []*bindingSubstitution{
			{
				FieldAccessor: "Some(&req).and_then(|m| m.oneof_field()).map(|s| s.as_str())",
				FieldName:     "oneof_field",
				Template:      []string{"*"},
			},
		},
	}
	method := api.NewTestMethod("DoFoo").
		WithInput(request).
		WithOutput(response).
		WithBindings(b0, b1, b2, b3).
		WithBodyFieldPath("*")
	methodBar := api.NewTestMethod("DoBar").
		WithInput(request).
		WithOutput(response).
		WithBindings(b4).
		WithBodyFieldPath("")
	service := api.NewTestService("FooService").
		WithMethods(method, methodBar)

	model := api.NewTestAPI(
		[]*api.Message{request, response},
		nil,
		[]*api.Service{service})
	api.CrossReference(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{})
	annotateModel(model, codec)

	if diff := cmp.Diff(wantB0, b0.Codec, cmpopts.IgnoreUnexported(pathBindingAnnotation{}, bindingSubstitution{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantB1, b1.Codec, cmpopts.IgnoreUnexported(pathBindingAnnotation{}, bindingSubstitution{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantB2, b2.Codec, cmpopts.IgnoreUnexported(pathBindingAnnotation{}, bindingSubstitution{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(wantB3, b3.Codec, cmpopts.IgnoreUnexported(pathBindingAnnotation{}, bindingSubstitution{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(wantB4, b4.Codec, cmpopts.IgnoreUnexported(pathBindingAnnotation{}, bindingSubstitution{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPathBindingAnnotationsDetailedTracing(t *testing.T) {
	fName := api.NewTestField("name").WithType(api.TypezString)
	request := api.NewTestMessage("Request").WithFields(fName)
	response := api.NewTestMessage("Response")
	binding := api.NewTestPathBinding("POST", (&api.PathTemplate{}).
		WithLiteral("v2").
		WithVariable(api.NewPathVariable("name").
			WithLiteral("projects").
			WithMatch()).
		WithVerb("create"))
	method := api.NewTestMethod("DoFoo").
		WithInput(request).
		WithOutput(response).
		WithBindings(binding)
	service := api.NewTestService("FooService").WithMethods(method)
	model := api.NewTestAPI(
		[]*api.Message{request, response},
		nil,
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
		WantClear     string
	}{
		{"machine", "machine", "Some(&req).map(|m| &m.machine).map(|s| s.as_str())", "Some(&mut req).map(|m| std::mem::take(&mut m.machine))"},
		{"machineType", "machine_type", "Some(&req).map(|m| &m.machine_type).map(|s| s.as_str())", "Some(&mut req).map(|m| std::mem::take(&mut m.machine_type))"},
		{"machine_type", "machine_type", "Some(&req).map(|m| &m.machine_type).map(|s| s.as_str())", "Some(&mut req).map(|m| std::mem::take(&mut m.machine_type))"},
		{"type", "type", "Some(&req).map(|m| &m.r#type).map(|s| s.as_str())", "Some(&mut req).map(|m| std::mem::take(&mut m.r#type))"},
	} {
		field := api.NewTestField(test.FieldName).
			WithType(api.TypezString).
			WithJSONName(test.FieldName)
		request := api.NewTestMessage("Request").WithFields(field)
		response := api.NewTestMessage("Response")
		binding := api.NewTestPathBinding("GET", (&api.PathTemplate{}).
			WithLiteral("v1").
			WithLiteral("machines").
			WithVariable(api.NewPathVariable(test.FieldName).
				WithMatch()).
			WithVerb("create")).
			WithQueryParameters(map[string]bool{})
		wantBinding := &pathBindingAnnotation{
			PathFmt: "/v1/machines/{}:create",
			Substitutions: []*bindingSubstitution{
				{
					FieldAccessor:  test.WantAccessor,
					FieldName:      test.WantFieldName,
					PathExtraction: &PathExtraction{FieldTake: test.WantClear},
					Template:       []string{"*"},
				},
			},
		}
		method := api.NewTestMethod("Create").
			WithInput(request).
			WithOutput(response).
			WithBindings(binding).
			WithBodyFieldPath("*")
		service := api.NewTestService("Service").WithMethods(method)
		model := api.NewTestAPI(
			[]*api.Message{request, response},
			nil,
			[]*api.Service{service})
		api.CrossReference(model)
		codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{})
		annotateModel(model, codec)
		if diff := cmp.Diff(wantBinding, binding.Codec, cmpopts.IgnoreUnexported(pathBindingAnnotation{}, bindingSubstitution{})); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}

	}
}

func TestPathBindingAnnotationsErrors(t *testing.T) {
	field := api.NewTestField("field").WithType(api.TypezString)
	request := api.NewTestMessage("Request").WithFields(field)
	method := api.NewTestMethod("Create").WithInput(request)
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
