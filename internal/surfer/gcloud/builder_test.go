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

package gcloud

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestNewArguments(t *testing.T) {
	service := api.NewTestService("TestService")
	model := api.NewTestAPI([]*api.Message{}, nil, []*api.Service{service})

	for _, test := range []struct {
		name   string
		method *api.Method
		want   int
	}{
		{
			name: "Method with input fields",
			method: api.NewTestMethod("DoSomething").WithInput(
				api.NewTestMessage("Request").WithFields(
					api.NewTestField("field_one").WithType(api.STRING_TYPE),
					api.NewTestField("field_two").WithType(api.INT32_TYPE),
				),
			),
			want: 2,
		},
		{
			name: "Method with no InputType",
			method: &api.Method{
				Name:      "EmptyMethod",
				InputType: nil,
			},
			want: 0,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := newArguments(test.method, &Config{}, model, service)
			if err != nil {
				t.Fatalf("newArguments() unexpected error = %v", err)
			}
			if len(got.Params) != test.want {
				t.Errorf("newArguments() generated %d params, want %d", len(got.Params), test.want)
			}
		})
	}
}

func TestNewArguments_Error(t *testing.T) {
	service := api.NewTestService("TestService")
	model := api.NewTestAPI([]*api.Message{}, nil, []*api.Service{service})

	for _, test := range []struct {
		name    string
		method  *api.Method
		wantErr string
	}{
		{
			name: "Error mapping input fields",
			method: api.NewTestMethod("DoSomethingError").WithInput(
				api.NewTestMessage("Request").WithFields(
					api.NewTestField("bad_nested").WithType(api.MESSAGE_TYPE).WithMessageType(
						api.NewTestMessage("Bad").WithFields(
							api.NewTestField("bad_ref").WithResourceReference("unknown"),
						),
					),
				),
			),
			wantErr: "resource definition not found",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := newArguments(test.method, &Config{}, model, service)
			if err == nil {
				t.Fatalf("newArguments() expected error, got nil")
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("newArguments() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestNewParam(t *testing.T) {
	model := &api.API{
		ResourceDefinitions: []*api.Resource{
			{
				Type: "test.googleapis.com/Network",
				Patterns: []api.ResourcePattern{
					{*api.NewPathSegment().WithLiteral("projects"), *api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()), *api.NewPathSegment().WithLiteral("networks"), *api.NewPathSegment().WithVariable(api.NewPathVariable("network").WithMatch())},
				},
			},
		},
	}
	service := &api.Service{DefaultHost: "test.googleapis.com"}

	for _, test := range []struct {
		name      string
		field     *api.Field
		apiField  string
		method    *api.Method
		overrides *Config
		want      Param
	}{
		{
			name:     "String Field",
			field:    api.NewTestField("description").WithType(api.STRING_TYPE).WithBehavior(api.FIELD_BEHAVIOR_OPTIONAL),
			apiField: "description",
			method:   api.NewTestMethod("CreateInstance"),
			want: Param{
				ArgName:  "description",
				APIField: "description",
				Type:     "str",
				HelpText: "Value for the `description` field.",
				Required: false,
				Repeated: false,
			},
		},
		{
			name:     "Resource Reference Field",
			field:    api.NewTestField("network").WithResourceReference("test.googleapis.com/Network"),
			apiField: "network",
			method:   api.NewTestMethod("CreateInstance"),
			want: Param{
				ArgName:  "network",
				APIField: "network",
				HelpText: "Value for the `network` field.",
				ResourceSpec: &ResourceSpec{
					Name:       "network",
					PluralName: "networks",
					Collection: "test.projects.networks",
					Attributes: []Attribute{
						{AttributeName: "project", ParameterName: "projectsId", Help: "The project id of the {resource} resource.", Property: "core/project"},
						{AttributeName: "network", ParameterName: "networksId", Help: "The network id of the {resource} resource."},
					},
					DisableAutoCompleters: true,
				},
				ResourceMethodParams: map[string]string{"network": "{__relative_name__}"},
			},
		},
		{
			name: "Help Text Override",
			field: func() *api.Field {
				f := api.NewTestField("foo").WithType(api.STRING_TYPE)
				f.ID = "test.foo"
				return f
			}(),
			overrides: &Config{
				APIs: []API{
					{
						HelpText: &HelpTextRules{
							FieldRules: []*HelpTextRule{
								{Selector: "test.foo", HelpText: &HelpTextElement{Brief: "Override Foo"}},
							},
						},
					},
				},
			},
			apiField: "foo",
			method:   api.NewTestMethod("CreateInstance"),
			want: Param{
				ArgName:  "foo",
				APIField: "foo",
				Type:     "str",
				HelpText: "Override Foo",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			overrides := test.overrides
			if overrides == nil {
				overrides = &Config{}
			}
			got, err := newParam(test.field, test.apiField, overrides, model, service, test.method)
			if err != nil {
				t.Errorf("newParam(%s) unexpected error: %v", test.name, err)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("newParam() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestShouldSkipParam(t *testing.T) {
	for _, test := range []struct {
		name   string
		field  *api.Field
		method *api.Method
		want   bool
	}{
		{
			name:  "Primary Resource ID (Create)",
			field: api.NewTestField("thing_id").WithType(api.STRING_TYPE),
			method: api.NewTestMethod("CreateThing").WithVerb("POST").WithInput(
				api.NewTestMessage("CreateRequest").WithFields(
					api.NewTestField("thing").WithType(api.MESSAGE_TYPE).WithMessageType(
						api.NewTestMessage("Thing").WithFields(
							api.NewTestField("name").WithType(api.STRING_TYPE),
						).WithResource(api.NewTestResource("test.googleapis.com/Thing")),
					),
				),
			),
			want: false,
		},
		{
			name:  "Name Field (Primary)",
			field: api.NewTestField("name").WithType(api.STRING_TYPE).WithResourceReference("test.googleapis.com/Thing"),
			method: api.NewTestMethod("DeleteThing").WithVerb("DELETE").WithInput(
				api.NewTestMessage("DeleteRequest").WithFields(
					api.NewTestField("name").WithType(api.STRING_TYPE).WithResourceReference("test.googleapis.com/Thing"),
				),
			),
			want: false,
		},
		{
			name: "Parent Field (Primary in List)",
			field: func() *api.Field {
				f := api.NewTestField("parent").WithType(api.STRING_TYPE).WithResourceReference("test.googleapis.com/Parent")
				f.ResourceReference.ChildType = "test.googleapis.com/Thing"
				return f
			}(),
			method: func() *api.Method {
				m := api.NewTestMethod("ListThings").WithVerb("GET").WithInput(
					api.NewTestMessage("ListRequest").WithFields(
						api.NewTestField("parent").WithType(api.STRING_TYPE).WithResourceReference("test.googleapis.com/Parent"),
					),
				)
				m.InputType.Fields[0].ResourceReference.ChildType = "test.googleapis.com/Thing"
				return m
			}(),
			want: false,
		},
		{
			name:  "Parent Field (Skipped in Create)",
			field: api.NewTestField("parent").WithType(api.STRING_TYPE),
			method: api.NewTestMethod("CreateThing").WithVerb("POST").WithInput(
				api.NewTestMessage("CreateRequest").WithFields(
					api.NewTestField("thing").WithType(api.MESSAGE_TYPE).WithMessageType(
						api.NewTestMessage("Thing").WithResource(api.NewTestResource("test.googleapis.com/Thing")),
					),
				),
			),
			want: true,
		},
		{
			name:  "Update Mask",
			field: api.NewTestField("update_mask").WithType(api.MESSAGE_TYPE),
			method: api.NewTestMethod("UpdateThing").WithVerb("PATCH").WithInput(
				api.NewTestMessage("UpdateRequest").WithFields(
					api.NewTestField("update_mask").WithType(api.MESSAGE_TYPE),
				),
			),
			want: true,
		},
		{
			name:  "Page Size (List)",
			field: api.NewTestField("page_size"),
			method: func() *api.Method {
				m := api.NewTestMethod("ListThings").WithVerb("GET").WithOutput(
					api.NewTestMessage("ListResponse").WithFields(
						api.NewTestField("things").WithType(api.MESSAGE_TYPE).WithRepeated(),
						api.NewTestField("next_page_token").WithType(api.STRING_TYPE),
					),
				)
				m.OutputType.Pagination = &api.PaginationInfo{
					PageableItem: m.OutputType.Fields[0],
				}
				return m
			}(),
			want: true,
		},
		{
			name:   "Immutable Field (Update)",
			field:  api.NewTestField("immutable").WithBehavior(api.FIELD_BEHAVIOR_IMMUTABLE),
			method: api.NewTestMethod("UpdateThing").WithVerb("PATCH"),
			want:   true,
		},
		{
			name:   "Output Only Field",
			field:  api.NewTestField("output_only").WithBehavior(api.FIELD_BEHAVIOR_OUTPUT_ONLY),
			method: api.NewTestMethod("CreateThing").WithVerb("POST"),
			want:   true,
		},
		{
			name:   "Regular Field",
			field:  api.NewTestField("description"),
			method: api.NewTestMethod("CreateThing").WithVerb("POST"),
			want:   false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := shouldSkipParam(test.field, test.method)
			if got != test.want {
				t.Errorf("shouldSkipParam() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestNewOutputConfig(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *api.Method
		want   *OutputConfig
	}{
		{
			name: "standard list method",
			method: api.NewTestMethod("ListThings").WithVerb("GET").WithOutput(
				api.NewTestMessage("ListResponse").WithFields(
					api.NewTestField("things").WithType(api.MESSAGE_TYPE).WithRepeated().WithMessageType(
						api.NewTestMessage("Thing").WithFields(
							api.NewTestField("name").WithType(api.STRING_TYPE),
							api.NewTestField("description").WithType(api.STRING_TYPE),
						).WithResource(api.NewTestResource("test.googleapis.com/Thing")),
					),
				),
			),
			want: &OutputConfig{
				Format: "table(\nname,\ndescription)",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			test.method.OutputType.Pagination = &api.PaginationInfo{
				PageableItem: test.method.OutputType.Fields[0],
			}
			got := newOutputConfig(test.method, &api.API{})
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("newOutputConfig() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewOutputConfig_Error(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *api.Method
	}{
		{
			name:   "not a list method",
			method: api.NewTestMethod("CreateInstance"),
		},
		{
			name: "missing output type",
			method: &api.Method{
				Name: "ListInstances",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{Verb: "GET"}},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := newOutputConfig(test.method, &api.API{}); got != nil {
				t.Errorf("newOutputConfig() = %v, want nil", got)
			}
		})
	}
}

func TestNewPrimaryResourceParam(t *testing.T) {
	for _, test := range []struct {
		name         string
		field        *api.Field
		method       *api.Method
		resourceDefs []*api.Resource
		want         Param
	}{
		{
			name:  "Create Instance (Positional)",
			field: api.NewTestField("thing_id").WithType(api.STRING_TYPE),
			method: func() *api.Method {
				m := api.NewTestMethod("CreateThing").WithVerb("POST").WithInput(
					api.NewTestMessage("CreateRequest").WithFields(
						api.NewTestField("thing").WithType(api.MESSAGE_TYPE).WithMessageType(
							api.NewTestMessage("Thing").WithFields(
								api.NewTestField("name").WithType(api.STRING_TYPE),
							).WithResource(&api.Resource{
								Type:     "test.googleapis.com/Thing",
								Singular: "thing",
								Plural:   "things",
								Patterns: []api.ResourcePattern{
									{
										*api.NewPathSegment().WithLiteral("projects"),
										*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
										*api.NewPathSegment().WithLiteral("things"),
										*api.NewPathSegment().WithVariable(api.NewPathVariable("thing").WithMatch()),
									},
								},
							}),
						),
					),
				)
				return m
			}(),
			resourceDefs: []*api.Resource{
				{
					Type:     "test.googleapis.com/Thing",
					Singular: "thing",
					Plural:   "things",
					Patterns: []api.ResourcePattern{
						{
							*api.NewPathSegment().WithLiteral("projects"),
							*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
							*api.NewPathSegment().WithLiteral("things"),
							*api.NewPathSegment().WithVariable(api.NewPathVariable("thing").WithMatch()),
						},
					},
				},
			},
			want: Param{
				HelpText:          "The thing to create.",
				IsPositional:      true,
				IsPrimaryResource: true,
				Required:          true,
				RequestIDField:    "thingId",
				ResourceSpec: &ResourceSpec{
					Name:       "thing",
					PluralName: "things",
					Collection: "test.projects.things",
					Attributes: []Attribute{
						{
							ParameterName: "projectsId",
							AttributeName: "project",
							Help:          "The project id of the {resource} resource.",
							Property:      "core/project",
						},
						{
							ParameterName: "thingsId",
							AttributeName: "thing",
							Help:          "The thing id of the {resource} resource.",
						},
					},
				},
			},
		},
		{
			name: "List Instances (Not Positional, Parent)",
			field: func() *api.Field {
				f := api.NewTestField("parent").WithType(api.STRING_TYPE).WithResourceReference("test.googleapis.com/Thing")
				f.ResourceReference.ChildType = "test.googleapis.com/Thing"
				return f
			}(),
			method: func() *api.Method {
				m := api.NewTestMethod("ListThings").WithVerb("GET").WithInput(
					api.NewTestMessage("ListRequest").WithFields(
						api.NewTestField("parent").WithType(api.STRING_TYPE).WithResourceReference("test.googleapis.com/Thing"),
					),
				)
				m.InputType.Fields[0].ResourceReference.ChildType = "test.googleapis.com/Thing"
				return m
			}(),
			resourceDefs: []*api.Resource{
				{
					Type:     "test.googleapis.com/Thing",
					Singular: "thing",
					Plural:   "things",
					Patterns: []api.ResourcePattern{
						{
							*api.NewPathSegment().WithLiteral("projects"),
							*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
							*api.NewPathSegment().WithLiteral("things"),
							*api.NewPathSegment().WithVariable(api.NewPathVariable("thing").WithMatch()),
						},
					},
				},
			},
			want: Param{
				HelpText:          "The project and location for which to retrieve projects information.",
				IsPositional:      false,
				IsPrimaryResource: true,
				Required:          true,
				ResourceSpec: &ResourceSpec{
					Name:       "project",
					PluralName: "projects",
					Collection: "test.projects",
					Attributes: []Attribute{
						{
							ParameterName: "projectsId",
							AttributeName: "project",
							Help:          "The project id of the {resource} resource.",
							Property:      "core/project",
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			service := api.NewTestService("TestService").WithPackage("google.cloud.test.v1")
			service.DefaultHost = "test.googleapis.com"
			model := api.NewTestAPI([]*api.Message{}, nil, []*api.Service{service})
			model.ResourceDefinitions = test.resourceDefs

			test.method.Service = service
			test.method.Model = model

			got := newPrimaryResourceParam(test.field, test.method, model, &Config{}, service)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("newPrimaryResourceParam() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewRequest(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *api.Method
		want   *Request
	}{
		{
			name: "Standard Create",
			method: api.NewTestMethod("CreateThing").WithVerb("POST").WithPathTemplate(
				api.NewPathTemplate().WithLiteral("v1").WithVariable(api.NewPathVariable("parent").WithLiteral("projects").WithMatch()).WithLiteral("things"),
			),
			want: &Request{
				Collection: []string{"test.projects.things"},
			},
		},
		{
			name: "Custom Method with Verb",
			method: api.NewTestMethod("ImportData").WithVerb("POST").WithPathTemplate(
				api.NewPathTemplate().WithLiteral("v1").WithVariable(api.NewPathVariable("name").WithLiteral("projects").WithMatch()).WithVerb("importData"),
			),
			want: &Request{
				Collection: []string{"test.projects"},
				Method:     "importData",
			},
		},
		{
			name: "Custom Method without Verb (fallback to camelCase name)",
			method: api.NewTestMethod("ExportData").WithVerb("POST").WithPathTemplate(
				api.NewPathTemplate().WithLiteral("v1").WithVariable(api.NewPathVariable("name").WithLiteral("projects").WithMatch()),
			),
			want: &Request{
				Collection: []string{"test.projects"},
				Method:     "exportData",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			service := api.NewTestService("TestService").WithPackage("google.cloud.test.v1")
			service.DefaultHost = "test.googleapis.com"
			model := api.NewTestAPI([]*api.Message{}, nil, []*api.Service{service})
			test.method.Service = service

			got := newRequest(test.method, &Config{}, model, service)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("newRequest() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewAsync(t *testing.T) {
	service := api.NewTestService("TestService")

	for _, test := range []struct {
		name   string
		method *api.Method
		want   *Async
	}{
		{
			name: "Create returns Resource",
			method: func() *api.Method {
				m := api.NewTestMethod("CreateThing").WithVerb("POST").WithPathTemplate(
					api.NewPathTemplate().WithLiteral("v1").WithVariable(api.NewPathVariable("parent").WithLiteral("projects").WithMatch()).WithLiteral("things"),
				).WithInput(
					api.NewTestMessage("CreateRequest").WithFields(
						api.NewTestField("thing").WithType(api.MESSAGE_TYPE).WithMessageType(
							api.NewTestMessage("Thing").WithResource(api.NewTestResource("test.googleapis.com/Thing")),
						),
					),
				)
				m.OperationInfo = &api.OperationInfo{ResponseTypeID: "Thing"}
				return m
			}(),
			want: &Async{
				Collection:            []string{"test.projects.operations"},
				ExtractResourceResult: true,
			},
		},
		{
			name: "Delete returns Empty",
			method: func() *api.Method {
				m := api.NewTestMethod("DeleteThing").WithVerb("DELETE").WithPathTemplate(
					api.NewPathTemplate().WithLiteral("v1").WithVariable(api.NewPathVariable("name").WithLiteral("projects").WithMatch().WithLiteral("things").WithMatch()),
				)
				m.OperationInfo = &api.OperationInfo{ResponseTypeID: ".google.protobuf.Empty"}
				return m
			}(),
			want: &Async{
				Collection:            []string{"test.projects.operations"},
				ExtractResourceResult: false,
			},
		},
		{
			name: "Unrelated Response Type returns False",
			method: func() *api.Method {
				m := api.NewTestMethod("CreateThing").WithVerb("POST").WithPathTemplate(
					api.NewPathTemplate().WithLiteral("v1").WithVariable(api.NewPathVariable("parent").WithLiteral("projects").WithMatch()).WithLiteral("things"),
				).WithInput(
					api.NewTestMessage("CreateRequest").WithFields(
						api.NewTestField("thing").WithType(api.MESSAGE_TYPE).WithMessageType(
							api.NewTestMessage("Thing").WithResource(api.NewTestResource("test.googleapis.com/Thing")),
						),
					),
				)
				m.OperationInfo = &api.OperationInfo{ResponseTypeID: "UnrelatedType"}
				m.Service = service
				return m
			}(),
			want: &Async{
				Collection:            []string{"test.projects.operations"},
				ExtractResourceResult: false,
			},
		},
		{
			name: "Method Without Resource Returns Base Async",
			method: func() *api.Method {
				m := api.NewTestMethod("CustomMethod").WithVerb("POST").WithPathTemplate(
					api.NewPathTemplate().WithLiteral("v1").WithVariable(api.NewPathVariable("name").WithLiteral("projects").WithMatch()).WithLiteral("things").WithVerb("doAction"),
				)
				m.OperationInfo = &api.OperationInfo{ResponseTypeID: "ActionResponse"}
				m.Service = service
				return m
			}(),
			want: &Async{
				Collection:            []string{"test.projects.operations"},
				ExtractResourceResult: false,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			service := api.NewTestService("TestService").WithPackage("google.cloud.test.v1")
			service.DefaultHost = "test.googleapis.com"
			model := api.NewTestAPI([]*api.Message{}, nil, []*api.Service{service})
			test.method.Service = service

			got := newAsync(test.method, model, &Config{}, service)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("newAsync() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAddFlattenedParams(t *testing.T) {
	service := api.NewTestService("TestService")
	model := api.NewTestAPI([]*api.Message{}, nil, []*api.Service{service})

	createMethod := api.NewTestMethod("CreateThing").WithVerb("POST").WithInput(
		api.NewTestMessage("CreateRequest").WithFields(
			api.NewTestField("thing_id").WithType(api.STRING_TYPE),
			api.NewTestField("thing").WithType(api.MESSAGE_TYPE).WithMessageType(
				api.NewTestMessage("Thing").WithFields(
					api.NewTestField("name").WithType(api.STRING_TYPE),
				).WithResource(&api.Resource{
					Type:     "test.googleapis.com/Thing",
					Singular: "thing",
				}),
			),
		),
	)
	createMethod.Service = service
	createMethod.Model = model

	for _, test := range []struct {
		name    string
		field   *api.Field
		prefix  string
		want    []Param
		wantErr bool
	}{
		{
			name:   "Skips skipped fields",
			field:  api.NewTestField("parent"),
			prefix: "parent",
			want:   nil,
		},
		{
			name:   "Handles Primary Resource ID",
			field:  createMethod.InputType.Fields[0],
			prefix: "thingId",
			want: []Param{
				{
					ArgName:           "",
					APIField:          "",
					HelpText:          "The thing to create.",
					IsPositional:      true,
					IsPrimaryResource: true,
					Required:          true,
					RequestIDField:    "thingId",
				},
			},
		},
		{
			name: "Handles Nested Message",
			field: &api.Field{
				Name:     "subnetwork",
				JSONName: "subnetwork",
				Typez:    api.MESSAGE_TYPE,
				MessageType: &api.Message{
					Fields: []*api.Field{
						{
							Name:     "foo",
							JSONName: "foo",
							Typez:    api.STRING_TYPE,
						},
					},
				},
			},
			prefix: "networkConfig",
			want: []Param{
				{
					ArgName:  "foo",
					APIField: "networkConfig.foo",
					Type:     "str",
					HelpText: "Value for the `foo` field.",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			args := &Arguments{}
			err := addFlattenedParams(test.field, test.prefix, args, &Config{}, model, service, createMethod)
			if err != nil {
				t.Fatalf("addFlattenedParams() unexpected error = %v", err)
			}
			if diff := cmp.Diff(test.want, args.Params, cmpopts.IgnoreUnexported(Param{}), cmpopts.IgnoreFields(Param{}, "ResourceSpec")); diff != "" {
				t.Errorf("addFlattenedParams() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAddFlattenedParams_Error(t *testing.T) {
	service := api.NewTestService("TestService")
	model := api.NewTestAPI([]*api.Message{}, nil, []*api.Service{service})

	createMethod := api.NewTestMethod("CreateThing").WithVerb("POST").WithInput(
		api.NewTestMessage("CreateRequest").WithFields(
			api.NewTestField("thing_id").WithType(api.STRING_TYPE),
		),
	)

	for _, test := range []struct {
		name   string
		field  *api.Field
		prefix string
	}{
		{
			name: "Error mapping subfield",
			field: &api.Field{
				Name:     "bad_nested",
				JSONName: "badNested",
				Typez:    api.MESSAGE_TYPE,
				MessageType: &api.Message{
					Fields: []*api.Field{
						api.NewTestField("bad").WithResourceReference("unknown"),
					},
				},
			},
			prefix: "bad",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			args := &Arguments{}
			err := addFlattenedParams(test.field, test.prefix, args, &Config{}, model, service, createMethod)
			if err == nil {
				t.Fatalf("addFlattenedParams() expected error, got nil")
			}
		})
	}
}

func TestNewResourceReferenceSpec(t *testing.T) {
	service := api.NewTestService("TestService")
	service.DefaultHost = "test.googleapis.com"

	model := &api.API{
		ResourceDefinitions: []*api.Resource{
			{
				Type: "test.googleapis.com/OtherThing",
				Patterns: []api.ResourcePattern{
					{
						*api.NewPathSegment().WithLiteral("projects"),
						*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
						*api.NewPathSegment().WithLiteral("otherThings"),
						*api.NewPathSegment().WithVariable(api.NewPathVariable("other_thing").WithMatch()),
					},
				},
			},
		},
	}

	for _, test := range []struct {
		name  string
		field *api.Field
		want  *ResourceSpec
	}{
		{
			name:  "Handles valid resource reference",
			field: api.NewTestField("other_thing").WithResourceReference("test.googleapis.com/OtherThing"),
			want: &ResourceSpec{
				Name:                  "other_thing",
				PluralName:            "otherThings",
				Collection:            "test.projects.otherThings",
				DisableAutoCompleters: true,
				Attributes: []Attribute{
					{ParameterName: "projectsId", AttributeName: "project", Help: "The project id of the {resource} resource.", Property: "core/project"},
					{ParameterName: "otherThingsId", AttributeName: "other_thing", Help: "The other_thing id of the {resource} resource."},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := newResourceReferenceSpec(test.field, model, &Config{}, service)
			if err != nil {
				t.Fatalf("newResourceReferenceSpec() unexpected error = %v", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("newResourceReferenceSpec() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewResourceReferenceSpec_Error(t *testing.T) {
	service := api.NewTestService("TestService")

	for _, test := range []struct {
		name  string
		field *api.Field
	}{
		{
			name:  "Fails for missing resource definition",
			field: api.NewTestField("unknown").WithResourceReference("unknown.googleapis.com/Unknown"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := newResourceReferenceSpec(test.field, &api.API{}, &Config{}, service)
			if err == nil {
				t.Fatalf("newResourceReferenceSpec() expected error, got nil")
			}
		})
	}
}

func TestNewCollectionPath(t *testing.T) {
	service := &api.Service{
		DefaultHost: "test.googleapis.com",
	}

	stringPtr := func(s string) *string { return &s }

	for _, test := range []struct {
		name    string
		method  *api.Method
		isAsync bool
		want    []string
	}{
		{
			name: "Standard Regional Request",
			method: &api.Method{
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							PathTemplate: &api.PathTemplate{
								Segments: []api.PathSegment{
									{Literal: stringPtr("v1")},
									{Literal: stringPtr("projects")},
									{Variable: &api.PathVariable{FieldPath: []string{"project"}}},
									{Literal: stringPtr("locations")},
									{Variable: &api.PathVariable{FieldPath: []string{"location"}}},
									{Literal: stringPtr("instances")},
									{Variable: &api.PathVariable{FieldPath: []string{"instance"}}},
								},
							},
						},
					},
				},
			},
			isAsync: false,
			want:    []string{"test.projects.locations.instances"},
		},
		{
			name: "Standard Regional Async",
			method: &api.Method{
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							PathTemplate: &api.PathTemplate{
								Segments: []api.PathSegment{
									{Literal: stringPtr("v1")},
									{Literal: stringPtr("projects")},
									{Variable: &api.PathVariable{FieldPath: []string{"project"}}},
									{Literal: stringPtr("locations")},
									{Variable: &api.PathVariable{FieldPath: []string{"location"}}},
									{Literal: stringPtr("instances")},
									{Variable: &api.PathVariable{FieldPath: []string{"instance"}}},
								},
							},
						},
					},
				},
			},
			isAsync: true,
			want:    []string{"test.projects.locations.operations"},
		},
		{
			name: "Async without dots in path",
			method: &api.Method{
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							PathTemplate: &api.PathTemplate{
								Segments: []api.PathSegment{
									{Literal: stringPtr("v1")},
									{Literal: stringPtr("instances")},
								},
							},
						},
					},
				},
			},
			isAsync: true,
			want:    []string{"test.operations"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := newCollectionPath(test.method, service, test.isAsync)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("newCollectionPath() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindHelpTextRule(t *testing.T) {
	method := api.NewTestMethod("CreateInstance")
	method.ID = "google.cloud.test.v1.Service.CreateInstance"

	for _, test := range []struct {
		name      string
		overrides *Config
		want      *HelpTextRule
	}{
		{
			name:      "No APIs in config",
			overrides: &Config{},
			want:      nil,
		},
		{
			name: "Matching rule found",
			overrides: &Config{
				APIs: []API{
					{
						HelpText: &HelpTextRules{
							MethodRules: []*HelpTextRule{
								{
									Selector: "google.cloud.test.v1.Service.CreateInstance",
									HelpText: &HelpTextElement{
										Brief: "Override Brief",
									},
								},
							},
						},
					},
				},
			},
			want: &HelpTextRule{
				Selector: "google.cloud.test.v1.Service.CreateInstance",
				HelpText: &HelpTextElement{
					Brief: "Override Brief",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := findHelpTextRule(method, test.overrides)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("findHelpTextRule() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindFieldHelpTextRule(t *testing.T) {
	field := api.NewTestField("instance_id")
	field.ID = ".google.cloud.test.v1.Request.instance_id"

	for _, test := range []struct {
		name      string
		overrides *Config
		want      *HelpTextRule
	}{
		{
			name:      "No APIs in config",
			overrides: &Config{},
			want:      nil,
		},
		{
			name: "Matching rule found",
			overrides: &Config{
				APIs: []API{
					{
						HelpText: &HelpTextRules{
							FieldRules: []*HelpTextRule{
								{
									Selector: ".google.cloud.test.v1.Request.instance_id",
									HelpText: &HelpTextElement{
										Brief: "Override Field Brief",
									},
								},
							},
						},
					},
				},
			},
			want: &HelpTextRule{
				Selector: ".google.cloud.test.v1.Request.instance_id",
				HelpText: &HelpTextElement{
					Brief: "Override Field Brief",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := findFieldHelpTextRule(field, test.overrides)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("findFieldHelpTextRule() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAPIVersion(t *testing.T) {
	for _, test := range []struct {
		name      string
		overrides *Config
		want      string
	}{
		{
			name:      "No APIs in config",
			overrides: &Config{},
			want:      "",
		},
		{
			name: "API version found",
			overrides: &Config{
				APIs: []API{
					{APIVersion: "v2beta1"},
				},
			},
			want: "v2beta1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := apiVersion(test.overrides)
			if got != test.want {
				t.Errorf("apiVersion() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestNewCommand(t *testing.T) {
	for _, test := range []struct {
		name      string
		method    *api.Method
		overrides *Config
		want      *Command
	}{
		{
			name: "List Command",
			method: func() *api.Method {
				m := api.NewTestMethod("ListThings").
					WithVerb("GET").
					WithInput(api.NewTestMessage("ListThingsRequest").WithFields(
						api.NewTestField("parent").WithType(api.STRING_TYPE).WithResourceReference("test.googleapis.com/Parent"),
					)).
					WithOutput(api.NewTestMessage("ListThingsResponse").WithFields(
						api.NewTestField("things").WithType(api.MESSAGE_TYPE).WithRepeated().WithMessageType(
							api.NewTestMessage("Thing").WithFields(
								api.NewTestField("name").WithType(api.STRING_TYPE),
							),
						),
					)).
					WithPathTemplate(api.NewPathTemplate().
						WithLiteral("v1").
						WithVariable(api.NewPathVariable("parent").WithLiteral("projects").WithMatch()).
						WithLiteral("things"))
				m.OutputType.Pagination = &api.PaginationInfo{PageableItem: m.OutputType.Fields[0]}
				return m
			}(),
			overrides: &Config{
				APIs: []API{
					{RootIsHidden: false},
				},
			},
			want: &Command{
				Hidden:   false,
				Response: &Response{IDField: "name"},
				Output:   &OutputConfig{Format: "table(\nname)"},
			},
		},
		{
			name: "Update Command with Help Rule",
			method: func() *api.Method {
				m := api.NewTestMethod("UpdateThing").
					WithVerb("PATCH").
					WithInput(api.NewTestMessage("UpdateThingRequest").WithFields(
						api.NewTestField("thing").WithType(api.MESSAGE_TYPE).WithMessageType(
							api.NewTestMessage("Thing").WithFields(
								api.NewTestField("name").WithType(api.STRING_TYPE),
							).WithResource(api.NewTestResource("test.googleapis.com/Thing")),
						),
						api.NewTestField("update_mask").WithType(api.MESSAGE_TYPE),
					)).
					WithPathTemplate(api.NewPathTemplate().
						WithLiteral("v1").
						WithVariable(api.NewPathVariable("thing", "name").WithLiteral("projects").WithMatch().WithLiteral("things").WithMatch()))
				m.ID = "google.cloud.test.v1.Service.UpdateThing"
				return m
			}(),
			overrides: &Config{
				APIs: []API{
					{
						RootIsHidden: true,
						HelpText: &HelpTextRules{
							MethodRules: []*HelpTextRule{
								{
									Selector: "google.cloud.test.v1.Service.UpdateThing",
									HelpText: &HelpTextElement{Brief: "Updated Brief"},
								},
							},
						},
					},
				},
			},
			want: &Command{
				Hidden: true,
				Update: &UpdateConfig{ReadModifyUpdate: true},
			},
		},
		{
			name: "LRO Command",
			method: func() *api.Method {
				m := api.NewTestMethod("CreateThing").
					WithVerb("POST").
					WithInput(api.NewTestMessage("CreateRequest").WithFields(
						api.NewTestField("thing_id").WithType(api.STRING_TYPE),
					)).
					WithPathTemplate(api.NewPathTemplate().
						WithLiteral("v1").
						WithVariable(api.NewPathVariable("parent").WithLiteral("projects").WithMatch()).
						WithLiteral("things"))
				m.ID = "google.cloud.test.v1.Service.CreateThing"
				m.OperationInfo = &api.OperationInfo{ResponseTypeID: "Thing", MetadataTypeID: "Metadata"}
				return m
			}(),
			overrides: &Config{},
			want: &Command{
				Hidden: true,
				Async: &Async{
					Collection:            []string{"test.projects.operations"},
					ExtractResourceResult: false,
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			service := api.NewTestService("TestService").WithPackage("google.cloud.test.v1")
			service.DefaultHost = "test.googleapis.com"
			model := api.NewTestAPI([]*api.Message{}, nil, []*api.Service{service})
			test.method.Service = service
			test.method.Model = model

			got, err := NewCommand(test.method, test.overrides, model, service)
			if err != nil {
				t.Fatalf("NewCommand() unexpected error = %v", err)
			}

			// Compare the important pieces that are shaped uniquely by NewCommand
			// to avoid asserting on deeply nested auto-generated boilerplate like Arguments.
			opts := cmpopts.IgnoreFields(Command{}, "AutoGenerated", "ReleaseTracks", "Arguments", "Request", "HelpText")
			if diff := cmp.Diff(test.want, got, opts); diff != "" {
				t.Errorf("NewCommand() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewFormat(t *testing.T) {
	tests := []struct {
		name    string
		message *api.Message
		want    string
	}{
		{
			name: "Scalar and Repeated Fields",
			message: api.NewTestMessage("Thing").WithFields(
				api.NewTestField("name").WithType(api.STRING_TYPE),
				api.NewTestField("tags").WithType(api.STRING_TYPE).WithRepeated(),
				api.NewTestField("count").WithType(api.INT32_TYPE),
			),
			want: "table(\nname,\ntags.join(','),\ncount)",
		},
		{
			name: "Timestamp Field",
			message: func() *api.Message {
				f := api.NewTestField("create_time").WithType(api.MESSAGE_TYPE)
				f.JSONName = "createTime"
				f.TypezID = ".google.protobuf.Timestamp"
				f.MessageType = &api.Message{}
				return api.NewTestMessage("Timed").WithFields(f)
			}(),
			want: "table(\ncreateTime)",
		},
		{
			name: "Ignored Unsafe Field",
			message: api.NewTestMessage("Unsafe").WithFields(
				api.NewTestField("safe").WithType(api.STRING_TYPE),
				&api.Field{JSONName: "unsafe;injection", Typez: api.STRING_TYPE},
			),
			want: "table(\nsafe)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := newFormat(test.message)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("newFormat() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
