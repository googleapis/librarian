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
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestNewParam(t *testing.T) {
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
			name:     "Long Field",
			field:    api.NewTestField("capacity_gib").WithType(api.INT64_TYPE).WithBehavior(api.FIELD_BEHAVIOR_OPTIONAL),
			apiField: "capacityGib",
			method:   api.NewTestMethod("CreateInstance"),
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
			name:     "Repeated Field",
			field:    api.NewTestField("labels").WithType(api.STRING_TYPE).WithRepeated(),
			apiField: "labels",
			method:   api.NewTestMethod("CreateInstance"),
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
			name:     "Required Field",
			field:    api.NewTestField("name").WithType(api.STRING_TYPE).WithBehavior(api.FIELD_BEHAVIOR_REQUIRED),
			apiField: "name",
			method:   api.NewTestMethod("CreateInstance"),
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
			name:     "Clearable Map (Update)",
			field:    api.NewTestField("labels").WithType(api.STRING_TYPE).WithMap(),
			apiField: "labels",
			method:   api.NewTestMethod("UpdateInstance").WithVerb("PATCH"),
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
			name:     "Clearable Repeated Field (Update)",
			field:    api.NewTestField("access_points").WithType(api.STRING_TYPE).WithRepeated(),
			apiField: "accessPoints",
			method:   api.NewTestMethod("UpdateInstance").WithVerb("PATCH"),
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
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(Param{}, "ResourceSpec")); diff != "" {
				t.Errorf("newParam() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestShouldSkipParam(t *testing.T) {
	createMethod := sample.MethodCreate()
	listMethod := sample.MethodListSecretVersions()
	updateMethod := sample.MethodUpdate()

	for _, test := range []struct {
		name   string
		field  *api.Field
		method *api.Method
		want   bool
	}{
		{
			name:   "Primary Resource ID (Create)",
			field:  api.NewTestField("secret_id"),
			method: createMethod,
			want:   false,
		},
		{
			name:   "Parent Field",
			field:  api.NewTestField("parent"),
			method: createMethod,
			want:   true,
		},
		{
			name:   "Name Field (Not Primary)",
			field:  api.NewTestField("name"),
			method: createMethod,
			want:   true,
		},
		{
			name:   "Update Mask",
			field:  api.NewTestField("update_mask"),
			method: updateMethod,
			want:   true,
		},
		{
			name:   "Page Size (List)",
			field:  api.NewTestField("page_size"),
			method: listMethod,
			want:   true,
		},
		{
			name:   "Page Token (List)",
			field:  api.NewTestField("page_token"),
			method: listMethod,
			want:   true,
		},
		{
			name:   "Filter (List)",
			field:  api.NewTestField("filter"),
			method: listMethod,
			want:   true,
		},
		{
			name:   "Order By (List)",
			field:  api.NewTestField("order_by"),
			method: listMethod,
			want:   true,
		},
		{
			name:   "Output Only Field",
			field:  api.NewTestField("output_only").WithBehavior(api.FIELD_BEHAVIOR_OUTPUT_ONLY),
			method: createMethod,
			want:   true,
		},
		{
			name:   "Immutable Field (Update)",
			field:  api.NewTestField("immutable").WithBehavior(api.FIELD_BEHAVIOR_IMMUTABLE),
			method: updateMethod,
			want:   true,
		},
		{
			name:   "Immutable Field (Create - Not Skipped)",
			field:  api.NewTestField("immutable").WithBehavior(api.FIELD_BEHAVIOR_IMMUTABLE),
			method: createMethod,
			want:   false,
		},
		{
			name:   "Regular Field",
			field:  api.NewTestField("description"),
			method: createMethod,
			want:   false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
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
			name:   "standard list method",
			method: sample.MethodListSecretVersions(),
			want: &OutputConfig{
				Format: "table(\nname,\nstate)",
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
			name:   "not a list method",
			method: sample.MethodCreate(),
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
