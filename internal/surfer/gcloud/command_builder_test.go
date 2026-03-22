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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestNewOutputFormat(t *testing.T) {
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
			want: "table(\nname,\ndescription)",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			test.method.OutputType.Pagination = &api.PaginationInfo{
				PageableItem: test.method.OutputType.Fields[0],
			}
			got := NewCommandBuilder(test.method, nil, nil, nil).newOutputFormat()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("newOutputFormat() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewOutputFormat_Error(t *testing.T) {
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
			if got := NewCommandBuilder(test.method, nil, nil, nil).newOutputFormat(); got != "" {
				t.Errorf("newOutputFormat() = %v, want empty string", got)
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
			test.method.Service = service

			got := NewCommandBuilder(test.method, &Config{}, nil, service).newRequest()
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

			got := NewCommandBuilder(test.method, nil, model, service).newAsync()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("newAsync() mismatch (-want +got):\n%s", diff)
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
			got := NewCommandBuilder(test.method, nil, nil, service).newCollectionPath(test.isAsync)
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
				Hidden:          false,
				ResponseIDField: "name",
				OutputFormat:    "table(\nname)",
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
				Hidden:           true,
				ReadModifyUpdate: true,
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

			got, err := NewCommandBuilder(test.method, test.overrides, model, service).Build()
			if err != nil {
				t.Fatalf("NewCommand() unexpected error = %v", err)
			}

			// Compare the important pieces that are shaped uniquely by NewCommand
			// to avoid asserting on deeply nested auto-generated boilerplate like Arguments.
			opts := cmpopts.IgnoreFields(Command{}, "ReleaseTracks", "Arguments", "Request", "HelpText")
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
