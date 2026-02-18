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

func TestNewParam(t *testing.T) {
	// Helper to create a basic field
	makeField := func(name string, typez api.Typez) *api.Field {
		return &api.Field{
			Name:     name,
			JSONName: name, // simplify default
			Typez:    typez,
			Behavior: []api.FieldBehavior{api.FIELD_BEHAVIOR_OPTIONAL},
		}
	}

	for _, test := range []struct {
		name     string
		field    *api.Field
		apiField string
		method   *api.Method
		want     Param
		wantErr  bool
	}{
		{
			name:     "String Field",
			field:    makeField("description", api.STRING_TYPE),
			apiField: "description",
			method:   &api.Method{Name: "CreateInstance"},
			want: Param{
				ArgName:  "description",
				APIField: "description",
				Type:     "str", // String is default/empty
				HelpText: "Value for the `description` field.",
				Required: false,
				Repeated: false,
			},
		},
		{
			name:     "Long Field",
			field:    makeField("capacity_gib", api.INT64_TYPE),
			apiField: "capacityGib",
			method:   &api.Method{Name: "CreateInstance"},
			want: Param{
				ArgName:  "capacity-gib",
				APIField: "capacityGib",
				Type:     "long",
				HelpText: "Value for the `capacity-gib` field.",
				Required: false,
				Repeated: false,
			},
		},
		{
			name: "Repeated Field",
			field: &api.Field{
				Name:     "labels",
				JSONName: "labels",
				Typez:    api.STRING_TYPE,
				Repeated: true,
			},
			apiField: "labels",
			method:   &api.Method{Name: "CreateInstance"},
			want: Param{
				ArgName:  "labels",
				APIField: "labels",
				Type:     "str",
				HelpText: "Value for the `labels` field.",
				Required: false,
				Repeated: true,
			},
		},
		{
			name: "Required Field",
			field: &api.Field{
				Name:     "name",
				JSONName: "name",
				Typez:    api.STRING_TYPE,
				Behavior: []api.FieldBehavior{api.FIELD_BEHAVIOR_REQUIRED},
			},
			apiField: "name",
			method:   &api.Method{Name: "CreateInstance"},
			want: Param{
				ArgName:  "name",
				APIField: "name",
				Type:     "str",
				HelpText: "Value for the `name` field.",
				Required: true,
				Repeated: false,
			},
		},
		{
			name: "Clearable Map (Update)",
			field: &api.Field{
				Name:     "labels",
				JSONName: "labels",
				Typez:    api.STRING_TYPE,
				Map:      true,
			},
			apiField: "labels",
			method:   &api.Method{Name: "UpdateInstance"},
			want: Param{
				ArgName:   "labels",
				APIField:  "labels",
				HelpText:  "Value for the `labels` field.",
				Repeated:  true,
				Clearable: true,
				Spec: []ArgSpec{
					{APIField: "key"},
					{APIField: "value"},
				},
			},
		},
		{
			name: "Clearable Repeated Field (Update)",
			field: &api.Field{
				Name:     "access_points",
				JSONName: "accessPoints",
				Typez:    api.STRING_TYPE,
				Repeated: true,
			},
			apiField: "accessPoints",
			method:   &api.Method{Name: "UpdateInstance"},
			want: Param{
				ArgName:   "access-points",
				APIField:  "accessPoints",
				Type:      "str",
				HelpText:  "Value for the `access-points` field.",
				Repeated:  true,
				Clearable: true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := newParam(test.field, test.apiField, &Config{}, &api.API{}, &api.Service{}, test.method)
			if (err != nil) != test.wantErr {
				t.Errorf("newParam() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			// Ignore fields that are hard to mock or irrelevant for basic mapping test
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(Param{}, "ResourceSpec")); diff != "" {
				t.Errorf("newParam() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewOutputConfig(t *testing.T) {
	instanceMsg := &api.Message{
		Fields: []*api.Field{
			{Name: "name", JSONName: "name", Typez: api.STRING_TYPE},
			{Name: "create_time", JSONName: "createTime", Typez: api.MESSAGE_TYPE, TypezID: ".google.protobuf.Timestamp", MessageType: &api.Message{}},
			{Name: "state", JSONName: "state", Typez: api.ENUM_TYPE},
			{Name: "capacity_gib", JSONName: "capacityGib", Typez: api.INT64_TYPE},
			{Name: "access_points", JSONName: "accessPoints", Typez: api.STRING_TYPE, Repeated: true},
		},
	}

	listMethod := &api.Method{
		Name: "ListInstances",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{{Verb: "GET"}},
		},
		OutputType: &api.Message{
			Fields: []*api.Field{
				{
					Name:        "instances",
					Repeated:    true,
					MessageType: instanceMsg,
				},
			},
		},
	}

	for _, test := range []struct {
		name   string
		method *api.Method
		want   *OutputConfig
	}{
		{
			name:   "standard list method",
			method: listMethod,
			want: &OutputConfig{
				Format: "table(\nname,\ncreateTime,\nstate,\ncapacityGib,\naccessPoints.join(','))",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
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
			name: "not a list method",
			method: &api.Method{
				Name: "CreateInstance",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{Verb: "POST"}},
				},
			},
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
			name: "Complex Variable Request (Action)",
			method: &api.Method{
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							PathTemplate: &api.PathTemplate{
								Segments: []api.PathSegment{
									{Literal: stringPtr("v1")},
									{
										Variable: &api.PathVariable{
											FieldPath: []string{"name"},
											Segments:  []string{"projects", "*", "locations", "*", "instances", "*"},
										},
									},
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
			name: "Complex Variable Async (Action)",
			method: &api.Method{
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							PathTemplate: &api.PathTemplate{
								Segments: []api.PathSegment{
									{Literal: stringPtr("v1")},
									{
										Variable: &api.PathVariable{
											FieldPath: []string{"name"},
											Segments:  []string{"projects", "*", "locations", "*", "instances", "*"},
										},
									},
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
			name: "List Method Request (Collection Parent)",
			method: &api.Method{
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							PathTemplate: &api.PathTemplate{
								Segments: []api.PathSegment{
									{Literal: stringPtr("v1")},
									{
										Variable: &api.PathVariable{
											FieldPath: []string{"parent"},
											Segments:  []string{"projects", "*", "locations", "*"},
										},
									},
									{Literal: stringPtr("instances")},
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
			name: "Multitype Binding",
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
						{
							PathTemplate: &api.PathTemplate{
								Segments: []api.PathSegment{
									{Literal: stringPtr("v1")},
									{Literal: stringPtr("folders")},
									{Variable: &api.PathVariable{FieldPath: []string{"folder"}}},
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
			want:    []string{"test.folders.locations.instances", "test.projects.locations.instances"},
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := newCollectionPath(test.method, service, test.isAsync)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("newCollectionPath() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestShouldSkipParam(t *testing.T) {
	instanceResource := api.NewTestResource("example.googleapis.com/Instance").
		WithPatterns(api.ResourcePattern{*api.NewPathSegment().WithLiteral("instances")})

	instanceMsg := api.NewTestMessage("Instance").
		WithFields(api.NewTestField("name")).
		WithResource(instanceResource)

	// List Method
	listMethod := api.NewTestMethod("ListInstances").
		WithVerb("GET").
		WithInput(api.NewTestMessage("ListInstancesRequest").WithFields(
			api.NewTestField("parent"),
			api.NewTestField("page_size"),
			api.NewTestField("page_token"),
			api.NewTestField("filter"),
			api.NewTestField("order_by"),
		))

	// Create Method
	createMethod := api.NewTestMethod("CreateInstance").
		WithVerb("POST").
		WithInput(api.NewTestMessage("CreateInstanceRequest").WithFields(
			api.NewTestField("instance_id"),
			api.NewTestField("instance").WithMessageType(instanceMsg),
		))

	// Update Method
	updateMethod := api.NewTestMethod("UpdateInstance").
		WithVerb("PATCH").
		WithInput(api.NewTestMessage("UpdateInstanceRequest").WithFields(
			api.NewTestField("name"),
			api.NewTestField("update_mask"),
			api.NewTestField("description").WithBehavior(api.FIELD_BEHAVIOR_IMMUTABLE),
			api.NewTestField("create_time").WithBehavior(api.FIELD_BEHAVIOR_OUTPUT_ONLY),
			api.NewTestField("labels"),
		))

	tests := []struct {
		name   string
		method *api.Method
		field  *api.Field
		want   bool
	}{
		// List Method Tests
		{"List: parent should skip", listMethod, api.NewTestField("parent"), false}, // Primary Resource Arg
		{"List: page_size should skip", listMethod, api.NewTestField("page_size"), true},
		{"List: page_token should skip", listMethod, api.NewTestField("page_token"), true},
		{"List: filter should skip", listMethod, api.NewTestField("filter"), true},
		{"List: order_by should skip", listMethod, api.NewTestField("order_by"), true},
		{"List: other field should not skip", listMethod, api.NewTestField("other"), false},

		// Create Method Tests
		{"Create: instance_id (primary resource) should not skip", createMethod, createMethod.InputType.Fields[0], false},
		{"Create: instance (body) should not skip", createMethod, createMethod.InputType.Fields[1], false},

		// Update Method Tests
		{"Update: name should skip", updateMethod, updateMethod.InputType.Fields[0], false}, // Primary Resource Arg
		{"Update: update_mask should skip", updateMethod, updateMethod.InputType.Fields[1], true},
		{"Update: immutable field should skip", updateMethod, updateMethod.InputType.Fields[2], true},
		{"Update: output only field should skip", updateMethod, updateMethod.InputType.Fields[3], true},
		{"Update: regular field should not skip", updateMethod, updateMethod.InputType.Fields[4], false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSkipParam(tt.field, tt.method)
			if got != tt.want {
				t.Errorf("shouldSkipParam(%q, %q) = %v, want %v", tt.field.Name, tt.method.Name, got, tt.want)
			}
		})
	}
}
