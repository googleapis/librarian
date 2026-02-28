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

package gcloud

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGetPluralFromSegments(t *testing.T) {
	for _, test := range []struct {
		name     string
		segments []api.PathSegment
		want     string
	}{
		{
			name: "Standard",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("projects"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
				*api.NewPathSegment().WithLiteral("locations"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("location").WithMatch()),
				*api.NewPathSegment().WithLiteral("instances"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("instance").WithMatch()),
			},
			want: "instances",
		},
		{
			name: "Short",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("shelves"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("shelf").WithMatch()),
			},
			want: "shelves",
		},
		{
			name: "No Variable End",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("projects"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
				*api.NewPathSegment().WithLiteral("locations"),
			},
			want: "",
		},
		{
			name:     "Empty",
			segments: nil,
			want:     "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := getPluralFromSegments(test.segments)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetParentFromSegments(t *testing.T) {
	for _, test := range []struct {
		name     string
		segments []api.PathSegment
		want     []api.PathSegment
	}{
		{
			name: "Standard",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("projects"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
				*api.NewPathSegment().WithLiteral("locations"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("location").WithMatch()),
				*api.NewPathSegment().WithLiteral("instances"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("instance").WithMatch()),
			},
			want: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("projects"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
				*api.NewPathSegment().WithLiteral("locations"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("location").WithMatch()),
			},
		},
		{
			name: "Root",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("projects"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
			},
			want: []api.PathSegment{},
		},
		{
			name: "Too Short",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("projects"),
			},
			want: nil,
		},
		{
			name: "Invalid Pattern (Ends in Literal)",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("projects"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
				*api.NewPathSegment().WithLiteral("locations"),
			},
			want: nil,
		},
		{
			name:     "Empty",
			segments: nil,
			want:     nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := getParentFromSegments(test.segments)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetSingularFromSegments(t *testing.T) {
	for _, test := range []struct {
		name     string
		segments []api.PathSegment
		want     string
	}{
		{
			name: "Standard",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("projects"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
				*api.NewPathSegment().WithLiteral("locations"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("location").WithMatch()),
				*api.NewPathSegment().WithLiteral("instances"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("instance").WithMatch()),
			},
			want: "instance",
		},
		{
			name: "Short",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("shelves"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("shelf").WithMatch()),
			},
			want: "shelf",
		},
		{
			name: "No Variable End",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("projects"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
				*api.NewPathSegment().WithLiteral("locations"),
			},
			want: "",
		},
		{
			name:     "Empty",
			segments: nil,
			want:     "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := getSingularFromSegments(test.segments)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetCollectionPathFromSegments(t *testing.T) {
	for _, test := range []struct {
		name     string
		segments []api.PathSegment
		want     string
	}{
		{
			name: "Standard",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("projects"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
				*api.NewPathSegment().WithLiteral("locations"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("location").WithMatch()),
				*api.NewPathSegment().WithLiteral("instances"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("instance").WithMatch()),
			},
			want: "projects.locations.instances",
		},
		{
			name: "Short",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("shelves"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("shelf").WithMatch()),
			},
			want: "shelves",
		},
		{
			name: "Root",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("projects"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
			},
			want: "projects",
		},
		{
			name: "Mixed",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("organizations"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("organization").WithMatch()),
				*api.NewPathSegment().WithLiteral("locations"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("location").WithMatch()),
				*api.NewPathSegment().WithLiteral("clusters"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("cluster").WithMatch()),
			},
			want: "organizations.locations.clusters",
		},
		{
			name: "Global",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("projects"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
				*api.NewPathSegment().WithLiteral("global"),
				*api.NewPathSegment().WithLiteral("networks"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("network").WithMatch()),
			},
			want: "projects.networks",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := getCollectionPathFromSegments(test.segments)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractPathFromSegments(t *testing.T) {
	for _, test := range []struct {
		name     string
		segments []api.PathSegment
		want     string
	}{
		{
			name: "Standard Regional",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("v1"),
				*api.NewPathSegment().WithLiteral("projects"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
				*api.NewPathSegment().WithLiteral("locations"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("location").WithMatch()),
				*api.NewPathSegment().WithLiteral("instances"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("instance").WithMatch()),
			},
			want: "projects.locations.instances",
		},
		{
			name: "Complex Variable",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("v1"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("name").WithLiteral("projects").WithMatch().WithLiteral("locations").WithMatch().WithLiteral("instances").WithMatch()),
			},
			want: "projects.locations.instances",
		},
		{
			name: "Trailing Literal (List)",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("v1"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("name").WithLiteral("projects").WithMatch().WithLiteral("locations").WithMatch()),
				*api.NewPathSegment().WithLiteral("instances"),
			},
			want: "projects.locations.instances",
		},
		{
			name: "No Version",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("projects"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("project")),
			},
			want: "projects",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := extractPathFromSegments(test.segments)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsPrimaryResource(t *testing.T) {
	for _, test := range []struct {
		name   string
		field  *api.Field
		method *api.Method
		want   bool
	}{
		{
			name:  "Create Method - Primary Resource ID",
			field: &api.Field{Name: "instance_id"},
			method: &api.Method{
				Name: "CreateInstance",
				InputType: &api.Message{
					Fields: []*api.Field{
						{
							MessageType: &api.Message{
								Name: "Instance",
								Resource: &api.Resource{
									Type: "example.googleapis.com/Instance",
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name:  "Create Method - Not Primary Resource",
			field: &api.Field{Name: "parent"},
			method: &api.Method{
				Name: "CreateInstance",
				InputType: &api.Message{
					Fields: []*api.Field{
						{
							MessageType: &api.Message{
								Name: "Instance",
								Resource: &api.Resource{
									Type: "example.googleapis.com/Instance",
								},
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name:  "Get Method - Primary Resource Name",
			field: &api.Field{Name: "name"},
			method: &api.Method{
				Name: "GetInstance",
				InputType: &api.Message{
					Fields: []*api.Field{{Name: "name"}},
				},
			},
			want: true,
		},
		{
			name:  "Delete Method - Primary Resource Name",
			field: &api.Field{Name: "name"},
			method: &api.Method{
				Name: "DeleteInstance",
				InputType: &api.Message{
					Fields: []*api.Field{{Name: "name"}},
				},
			},
			want: true,
		},
		{
			name:  "Update Method - Primary Resource Name",
			field: &api.Field{Name: "name"},
			method: &api.Method{
				Name: "UpdateInstance",
				InputType: &api.Message{
					Fields: []*api.Field{{Name: "name"}},
				},
			},
			want: true,
		},
		{
			name:  "List Method - Primary Resource",
			field: &api.Field{Name: "parent"},
			method: &api.Method{
				Name: "ListInstances",
				InputType: &api.Message{
					Fields: []*api.Field{{Name: "parent"}},
				},
			},
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := isPrimaryResource(test.field, test.method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetResourceForMethod(t *testing.T) {
	instanceResource := &api.Resource{Type: "example.googleapis.com/Instance"}
	otherResource := &api.Resource{Type: "example.googleapis.com/Other"}

	for _, test := range []struct {
		name         string
		method       *api.Method
		resourceDefs []*api.Resource
		messages     []*api.Message
		want         *api.Resource
	}{
		{
			name: "Create Method - Resource in Message",
			method: &api.Method{
				Name: "CreateInstance",
				InputType: &api.Message{
					Fields: []*api.Field{
						{
							MessageType: &api.Message{
								Name:     "Instance",
								Resource: instanceResource,
							},
						},
					},
				},
			},
			resourceDefs: []*api.Resource{instanceResource},
			want:         instanceResource,
		},
		{
			name: "Get Method - Resource Reference",
			method: &api.Method{
				Name: "GetInstance",
				InputType: &api.Message{
					Fields: []*api.Field{
						{
							Name: "name",
							ResourceReference: &api.ResourceReference{
								Type: "example.googleapis.com/Instance",
							},
						},
					},
				},
			},
			resourceDefs: []*api.Resource{instanceResource},
			want:         instanceResource,
		},
		{
			name: "List Method - Child Type Reference",
			method: &api.Method{
				Name: "ListInstances",
				InputType: &api.Message{
					Fields: []*api.Field{
						{
							Name: "parent",
							ResourceReference: &api.ResourceReference{
								ChildType: "example.googleapis.com/Instance",
							},
						},
					},
				},
			},
			resourceDefs: []*api.Resource{instanceResource},
			want:         instanceResource,
		},
		{
			name: "Unknown Resource",
			method: &api.Method{
				Name: "Unknown",
				InputType: &api.Message{
					Fields: []*api.Field{{Name: "foo"}},
				},
			},
			resourceDefs: []*api.Resource{instanceResource},
			want:         nil,
		},
		{
			name: "Nil InputType",
			method: &api.Method{
				Name:      "NoInput",
				InputType: nil,
			},
			resourceDefs: []*api.Resource{instanceResource},
			want:         nil,
		},
		{
			name: "Resource on Message Directly",
			method: &api.Method{
				Name: "GetOther",
				InputType: &api.Message{
					Fields: []*api.Field{
						{
							Name: "name",
							ResourceReference: &api.ResourceReference{
								Type: "example.googleapis.com/Other",
							},
						},
					},
				},
			},
			messages: []*api.Message{
				{
					Name:     "OtherMessage",
					Resource: otherResource,
				},
			},
			want: otherResource,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			model := &api.API{
				ResourceDefinitions: test.resourceDefs,
				Messages:            test.messages,
			}
			got := getResourceForMethod(test.method, model)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetPluralResourceNameForMethod(t *testing.T) {
	instanceResource := &api.Resource{
		Type: "example.googleapis.com/Instance",
		Patterns: []api.ResourcePattern{
			{
				*api.NewPathSegment().WithLiteral("instances"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("instance").WithMatch()),
			},
		},
	}

	for _, test := range []struct {
		name         string
		method       *api.Method
		resourceDefs []*api.Resource
		want         string
	}{
		{
			name: "Inferred from Pattern",
			method: &api.Method{
				Name: "ListInstances",
				InputType: &api.Message{
					Fields: []*api.Field{
						{
							Name: "parent",
							ResourceReference: &api.ResourceReference{
								ChildType: "example.googleapis.com/Instance",
							},
						},
					},
				},
			},
			resourceDefs: []*api.Resource{instanceResource},
			want:         "instances",
		},
		{
			name: "Explicit Plural",
			method: &api.Method{
				Name: "ListBooks",
				InputType: &api.Message{
					Fields: []*api.Field{
						{
							Name: "parent",
							ResourceReference: &api.ResourceReference{
								ChildType: "example.googleapis.com/Book",
							},
						},
					},
				},
			},
			resourceDefs: []*api.Resource{
				instanceResource,
				{
					Type:   "example.googleapis.com/Book",
					Plural: "books",
				},
			},
			want: "books",
		},
		{
			name: "Resource Not Found",
			method: &api.Method{
				Name: "ListUnknown",
				InputType: &api.Message{
					Fields: []*api.Field{
						{
							Name: "parent",
							ResourceReference: &api.ResourceReference{
								ChildType: "example.googleapis.com/Unknown",
							},
						},
					},
				},
			},
			resourceDefs: []*api.Resource{instanceResource},
			want:         "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			model := &api.API{
				ResourceDefinitions: test.resourceDefs,
			}
			got := pluralResourceName(getResourceForMethod(test.method, model))
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetSingularResourceNameForMethod(t *testing.T) {
	instanceResource := &api.Resource{
		Type: "example.googleapis.com/Instance",
		Patterns: []api.ResourcePattern{
			{
				*api.NewPathSegment().WithLiteral("instances"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("instance").WithMatch()),
			},
		},
	}

	for _, test := range []struct {
		name         string
		method       *api.Method
		resourceDefs []*api.Resource
		want         string
	}{
		{
			name: "Inferred from Pattern",
			method: &api.Method{
				Name: "ListInstances",
				InputType: &api.Message{
					Fields: []*api.Field{
						{
							Name: "parent",
							ResourceReference: &api.ResourceReference{
								ChildType: "example.googleapis.com/Instance",
							},
						},
					},
				},
			},
			resourceDefs: []*api.Resource{instanceResource},
			want:         "instance",
		},
		{
			name: "Explicit Singular",
			method: &api.Method{
				Name: "ListBooks",
				InputType: &api.Message{
					Fields: []*api.Field{
						{
							Name: "parent",
							ResourceReference: &api.ResourceReference{
								ChildType: "example.googleapis.com/Book",
							},
						},
					},
				},
			},
			resourceDefs: []*api.Resource{
				instanceResource,
				{
					Type:     "example.googleapis.com/Book",
					Singular: "book",
				},
			},
			want: "book",
		},
		{
			name: "Resource Not Found",
			method: &api.Method{
				Name: "ListUnknown",
				InputType: &api.Message{
					Fields: []*api.Field{
						{
							Name: "parent",
							ResourceReference: &api.ResourceReference{
								ChildType: "example.googleapis.com/Unknown",
							},
						},
					},
				},
			},
			resourceDefs: []*api.Resource{instanceResource},
			want:         "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			model := &api.API{
				ResourceDefinitions: test.resourceDefs,
			}
			got := singularResourceName(getResourceForMethod(test.method, model))
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetResourceNameFromType(t *testing.T) {
	for _, test := range []struct {
		name    string
		typeStr string
		want    string
	}{
		{"Standard", "example.googleapis.com/Instance", "Instance"},
		{"No Slash", "Instance", "Instance"},
		{"Empty", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := getResourceNameFromType(test.typeStr)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
