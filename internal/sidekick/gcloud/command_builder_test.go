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
	"github.com/googleapis/librarian/internal/sidekick/gcloud/provider"
)

func TestOutputFormat(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *api.Method
		want   string
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
			want: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			test.method.OutputType.Pagination = &api.PaginationInfo{
				PageableItem: test.method.OutputType.Fields[0],
			}
			got := newCommandBuilder(test.method, nil, nil, nil).outputFormat()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("outputFormat() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestOutputFormat_Error(t *testing.T) {
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
			if got := newCommandBuilder(test.method, nil, nil, nil).outputFormat(); got != "" {
				t.Errorf("outputFormat() = %v, want empty string", got)
			}
		})
	}
}

func TestRequestMethod(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *api.Method
		want   string
	}{
		{
			name: "Standard Create",
			method: api.NewTestMethod("CreateThing").WithVerb("POST").WithPathTemplate(
				api.NewPathTemplate().WithLiteral("v1").WithVariable(api.NewPathVariable("parent").WithLiteral("projects").WithMatch()).WithLiteral("things"),
			),
			want: "",
		},
		{
			name: "Custom Method with Verb",
			method: api.NewTestMethod("ImportData").WithVerb("POST").WithPathTemplate(
				api.NewPathTemplate().WithLiteral("v1").WithVariable(api.NewPathVariable("name").WithLiteral("projects").WithMatch()).WithVerb("importData"),
			),
			want: "importData",
		},
		{
			name: "Custom Method without Verb (fallback to camelCase name)",
			method: api.NewTestMethod("ExportData").WithVerb("POST").WithPathTemplate(
				api.NewPathTemplate().WithLiteral("v1").WithVariable(api.NewPathVariable("name").WithLiteral("projects").WithMatch()),
			),
			want: "exportData",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			service := api.NewTestService("TestService").WithPackage("google.cloud.test.v1")
			service.DefaultHost = "test.googleapis.com"
			test.method.Service = service

			got := newCommandBuilder(test.method, &provider.Config{}, nil, service).requestMethod()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("requestMethod() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCommandName(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *api.Method
		want   string
	}{
		{
			name: "Standard Create",
			method: func() *api.Method {
				m := api.NewTestMethod("CreateThing").WithVerb("POST")
				m.Service = api.NewTestService("TestService").WithPackage("google.cloud.test.v1")
				m.Service.DefaultHost = "test.googleapis.com"
				return m
			}(),
			want: "create",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := newCommandBuilder(test.method, nil, nil, test.method.Service).name()
			if got != test.want {
				t.Errorf("name() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestAsync(t *testing.T) {
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

			got := newCommandBuilder(test.method, nil, model, service).async()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("async() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCollectionPath(t *testing.T) {
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
			got := newCommandBuilder(test.method, nil, nil, service).collectionPath(test.isAsync)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("collectionPath() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewCommand(t *testing.T) {
	for _, test := range []struct {
		name      string
		method    *api.Method
		overrides *provider.Config
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
			overrides: &provider.Config{
				APIs: []provider.API{
					{RootIsHidden: false},
				},
			},
			want: &Command{
				Name:            "list",
				Hidden:          false,
				ResponseIDField: "name",
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
			overrides: &provider.Config{
				APIs: []provider.API{
					{
						RootIsHidden: true,
						HelpText: &provider.HelpTextRules{
							MethodRules: []*provider.HelpTextRule{
								{
									Selector: "google.cloud.test.v1.Service.UpdateThing",
									HelpText: &provider.HelpTextElement{Brief: "Updated Brief"},
								},
							},
						},
					},
				},
			},
			want: &Command{
				Name:                 "update",
				Hidden:               true,
				ReadModifyUpdate:     true,
				StarUpdateMask:       true,
				DisableAutoFieldMask: true,
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
			overrides: &provider.Config{},
			want: &Command{
				Name:   "create",
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
			model.ResourceDefinitions = []*api.Resource{
				{
					Type:     "test.googleapis.com/Parent",
					Singular: "parent",
				},
			}
			test.method.Service = service
			test.method.Model = model

			got, err := newCommandBuilder(test.method, test.overrides, model, service).build()
			if err != nil {
				t.Fatalf("newCommandBuilder().build() unexpected error: %v", err)
			}

			opts := cmpopts.IgnoreFields(Command{}, "Arguments", "APIVersion", "Collection", "Method", "HelpText")
			if diff := cmp.Diff(test.want, got, opts); diff != "" {
				t.Errorf("NewCommand() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTableFormat(t *testing.T) {
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
			got := tableFormat(test.message)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("tableFormat() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCommandBuilderNewArguments(t *testing.T) {
	for _, test := range []struct {
		name          string
		method        *api.Method
		service       *api.Service
		model         *api.API
		wantAPIFields []string
		wantErr       bool
	}{
		{
			name: "Method with no InputType",
			method: &api.Method{
				Name:      "EmptyMethod",
				InputType: nil,
			},
			service:       api.NewTestService("TestService"),
			model:         api.NewTestAPI([]*api.Message{}, nil, []*api.Service{api.NewTestService("TestService")}),
			wantAPIFields: []string{},
		},
		{
			name: "Identify name field",
			method: func() *api.Method {
				m := api.NewTestMethod("GetResource").WithVerb("GET").WithInput(
					api.NewTestMessage("GetRequest").WithFields(
						api.NewTestField("name").WithType(api.STRING_TYPE),
					),
				)
				m.PathInfo = &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							PathTemplate: api.NewPathTemplate().WithLiteral("v1").WithVariable(api.NewPathVariable("name").WithMatch()),
						},
					},
				}
				return m
			}(),
			service: api.NewTestService("TestService"),
			model: func() *api.API {
				s := api.NewTestService("TestService")
				s.DefaultHost = "test.googleapis.com"
				m := api.NewTestAPI([]*api.Message{}, nil, []*api.Service{s})
				m.ResourceDefinitions = []*api.Resource{
					{
						Type:     "test.googleapis.com/Resource",
						Singular: "resource",
						Plural:   "resources",
					},
				}
				return m
			}(),
			wantAPIFields: []string{""},
		},
		{
			name: "Identify parent field",
			method: func() *api.Method {
				f := api.NewTestField("parent").WithType(api.STRING_TYPE)
				f.ResourceReference = &api.ResourceReference{Type: "test.googleapis.com/Resource"}

				m := api.NewTestMethod("ListResources").WithVerb("GET").WithInput(
					api.NewTestMessage("ListRequest").WithFields(f),
				)
				m.PathInfo = &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							PathTemplate: api.NewPathTemplate().WithLiteral("v1").WithVariable(api.NewPathVariable("parent").WithMatch()),
						},
					},
				}
				return m
			}(),
			service: api.NewTestService("TestService"),
			model: func() *api.API {
				s := api.NewTestService("TestService")
				s.DefaultHost = "test.googleapis.com"
				m := api.NewTestAPI([]*api.Message{}, nil, []*api.Service{s})
				m.ResourceDefinitions = []*api.Resource{
					{
						Type:     "test.googleapis.com/Resource",
						Singular: "resource",
						Plural:   "resources",
					},
				}
				return m
			}(),
			wantAPIFields: []string{""},
		},
		{
			name: "Flatten fields with body *",
			method: func() *api.Method {
				m := api.NewTestMethod("ArchiveResource").WithVerb("POST").WithInput(
					api.NewTestMessage("ArchiveRequest").WithFields(
						api.NewTestField("root_field").WithType(api.STRING_TYPE),
						api.NewTestField("resource").WithType(api.MESSAGE_TYPE).WithMessageType(
							api.NewTestMessage("Resource").WithFields(
								api.NewTestField("name").WithType(api.STRING_TYPE),
								api.NewTestField("field1").WithType(api.STRING_TYPE),
							),
						),
					),
				)
				m.PathInfo = &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							PathTemplate: api.NewPathTemplate().WithLiteral("v1").WithVariable(api.NewPathVariable("resource.name").WithMatch()),
						},
					},
					BodyFieldPath: "*",
				}
				return m
			}(),
			service:       api.NewTestService("TestService"),
			model:         api.NewTestAPI([]*api.Message{}, nil, []*api.Service{api.NewTestService("TestService")}),
			wantAPIFields: []string{"", "rootField", "resource.field1"},
		},
		{
			name: "Identify both parent and resource_id",
			method: func() *api.Method {
				m := api.NewTestMethod("CreateResource").WithVerb("POST").WithInput(
					api.NewTestMessage("CreateRequest").WithFields(
						api.NewTestField("parent").WithType(api.STRING_TYPE),
						api.NewTestField("resource_id").WithType(api.STRING_TYPE),
						api.NewTestField("top_level_string").WithType(api.STRING_TYPE),
						api.NewTestField("resource").WithType(api.MESSAGE_TYPE).WithMessageType(
							api.NewTestMessage("Resource").WithFields(
								api.NewTestField("name").WithType(api.STRING_TYPE),
								api.NewTestField("inner_string").WithType(api.STRING_TYPE),
							).WithResource(api.NewTestResource("test.googleapis.com/Resource")),
						),
					),
				)
				m.PathInfo = &api.PathInfo{
					BodyFieldPath: "resource",
				}
				return m
			}(),
			service: api.NewTestService("TestService"),
			model: func() *api.API {
				s := api.NewTestService("TestService")
				s.DefaultHost = "test.googleapis.com"
				m := api.NewTestAPI([]*api.Message{}, nil, []*api.Service{s})
				m.ResourceDefinitions = []*api.Resource{
					{
						Type:     "test.googleapis.com/Resource",
						Singular: "resource",
						Plural:   "resources",
					},
				}
				return m
			}(),
			wantAPIFields: []string{"", "topLevelString", "resource.innerString"},
		},
		{
			name: "Map suppression with body *",
			method: func() *api.Method {
				f := api.NewTestField("labels").WithType(api.MESSAGE_TYPE)
				f.Map = true

				m := api.NewTestMethod("ArchiveResource").WithVerb("POST").WithInput(
					api.NewTestMessage("ArchiveRequest").WithFields(
						api.NewTestField("resource").WithType(api.MESSAGE_TYPE).WithMessageType(
							api.NewTestMessage("Resource").WithFields(
								api.NewTestField("name").WithType(api.STRING_TYPE),
								f,
							),
						),
					),
				)
				m.PathInfo = &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							PathTemplate: api.NewPathTemplate().WithLiteral("v1").WithVariable(api.NewPathVariable("resource.name").WithMatch()),
						},
					},
					BodyFieldPath: "*",
				}
				return m
			}(),
			service:       api.NewTestService("TestService"),
			model:         api.NewTestAPI([]*api.Message{}, nil, []*api.Service{api.NewTestService("TestService")}),
			wantAPIFields: []string{"", "resource.labels"},
		},
		{
			name: "Output-only exclusion with body *",
			method: func() *api.Method {
				f := api.NewTestField("create_time").WithType(api.STRING_TYPE)
				f.Behavior = []api.FieldBehavior{api.FIELD_BEHAVIOR_OUTPUT_ONLY}

				m := api.NewTestMethod("ArchiveResource").WithVerb("POST").WithInput(
					api.NewTestMessage("ArchiveRequest").WithFields(
						api.NewTestField("resource").WithType(api.MESSAGE_TYPE).WithMessageType(
							api.NewTestMessage("Resource").WithFields(
								api.NewTestField("name").WithType(api.STRING_TYPE),
								f,
							),
						),
					),
				)
				m.PathInfo = &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							PathTemplate: api.NewPathTemplate().WithLiteral("v1").WithVariable(api.NewPathVariable("resource.name").WithMatch()),
						},
					},
					BodyFieldPath: "*",
				}
				return m
			}(),
			service:       api.NewTestService("TestService"),
			model:         api.NewTestAPI([]*api.Message{}, nil, []*api.Service{api.NewTestService("TestService")}),
			wantAPIFields: []string{""},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := &commandBuilder{
				method:    test.method,
				service:   test.service,
				model:     test.model,
				overrides: &provider.Config{},
			}
			args, err := b.newArguments()
			if (err != nil) != test.wantErr {
				t.Fatalf("newArguments() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr {
				return
			}

			var gotAPIFields []string
			for _, arg := range args {
				gotAPIFields = append(gotAPIFields, arg.APIField)
			}
			if diff := cmp.Diff(test.wantAPIFields, gotAPIFields, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("api_fields mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCommandBuilderNewArgumentsDuplicatesError(t *testing.T) {
	m := api.NewTestMethod("CustomMethod").WithVerb("POST").WithInput(
		api.NewTestMessage("CustomRequest").WithFields(
			api.NewTestField("name").WithType(api.STRING_TYPE), // Top level name
			api.NewTestField("resource").WithType(api.MESSAGE_TYPE).WithMessageType(
				api.NewTestMessage("Resource").WithFields(
					api.NewTestField("name").WithType(api.STRING_TYPE), // Inner name
				),
			),
		),
	)
	m.PathInfo = &api.PathInfo{
		Bindings: []*api.PathBinding{
			{
				PathTemplate: api.NewPathTemplate().WithLiteral("v1").WithVariable(api.NewPathVariable("name").WithMatch()),
			},
		},
		BodyFieldPath: "*",
	}

	b := &commandBuilder{
		method:    m,
		service:   api.NewTestService("TestService"),
		model:     api.NewTestAPI([]*api.Message{}, nil, []*api.Service{api.NewTestService("TestService")}),
		overrides: &provider.Config{},
	}

	_, err := b.newArguments()
	if err == nil {
		t.Errorf("newArguments() expected error for duplicate resource fields, got nil")
	}
}

func TestCommandBuilderNewArgumentsResourceError(t *testing.T) {
	service := api.NewTestService("TestService")
	model := api.NewTestAPI([]*api.Message{}, nil, []*api.Service{service})

	badMethod := api.NewTestMethod("DoSomethingError").WithInput(
		api.NewTestMessage("Request").WithFields(
			api.NewTestField("bad_nested").WithType(api.MESSAGE_TYPE).WithMessageType(
				api.NewTestMessage("Bad").WithFields(
					api.NewTestField("bad_ref").WithResourceReference("unknown"),
				),
			),
		),
	)
	badMethod.PathInfo.BodyFieldPath = "bad_nested"

	b := &commandBuilder{
		method:    badMethod,
		service:   service,
		model:     model,
		overrides: &provider.Config{},
	}

	_, err := b.newArguments()
	if err == nil {
		t.Fatalf("newArguments() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "resource definition not found") {
		t.Errorf("newArguments() error = %v, want error containing %q", err, "resource definition not found")
	}
}
