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

package utils

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
			got := GetPluralFromSegments(test.segments)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("GetPluralFromSegments mismatch (-want +got):\n%s", diff)
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
			got := GetSingularFromSegments(test.segments)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("GetSingularFromSegments mismatch (-want +got):\n%s", diff)
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
			got := GetCollectionPathFromSegments(test.segments)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("GetCollectionPathFromSegments mismatch (-want +got):\n%s", diff)
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
			name:  "List Method - Not Primary Resource",
			field: &api.Field{Name: "parent"},
			method: &api.Method{
				Name: "ListInstances",
				InputType: &api.Message{
					Fields: []*api.Field{{Name: "parent"}},
				},
			},
			want: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := IsPrimaryResource(test.field, test.method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("IsPrimaryResource mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetResourceForMethod(t *testing.T) {
	instanceResource := &api.Resource{Type: "example.googleapis.com/Instance"}
	model := &api.API{
		ResourceDefinitions: []*api.Resource{
			instanceResource,
		},
	}

	for _, test := range []struct {
		name   string
		method *api.Method
		want   *api.Resource
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
			want: instanceResource,
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
			want: instanceResource,
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
			want: instanceResource,
		},
		{
			name: "Unknown Resource",
			method: &api.Method{
				Name: "Unknown",
				InputType: &api.Message{
					Fields: []*api.Field{{Name: "foo"}},
				},
			},
			want: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := GetResourceForMethod(test.method, model)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("GetResourceForMethod mismatch (-want +got):\n%s", diff)
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
	model := &api.API{
		ResourceDefinitions: []*api.Resource{
			instanceResource,
		},
	}

	for _, test := range []struct {
		name   string
		method *api.Method
		want   string
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
			want: "instances",
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
			want: "books", // Assuming we mock a Book resource with Plural="books" below
		},
	} {
		// Setup explicit plural for the second case
		if test.name == "Explicit Plural" {
			bookResource := &api.Resource{
				Type:   "example.googleapis.com/Book",
				Plural: "books",
			}
			model.ResourceDefinitions = append(model.ResourceDefinitions, bookResource)
		}

		t.Run(test.name, func(t *testing.T) {
			got := GetPluralResourceNameForMethod(test.method, model)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("GetPluralResourceNameForMethod mismatch (-want +got):\n%s", diff)
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
	model := &api.API{
		ResourceDefinitions: []*api.Resource{
			instanceResource,
		},
	}

	for _, test := range []struct {
		name   string
		method *api.Method
		want   string
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
			want: "instance",
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
			want: "book", // Assuming we mock a Book resource with Singular="book" below
		},
	} {
		// Setup explicit singular for the second case
		if test.name == "Explicit Singular" {
			bookResource := &api.Resource{
				Type:     "example.googleapis.com/Book",
				Singular: "book",
			}
			model.ResourceDefinitions = append(model.ResourceDefinitions, bookResource)
		}

		t.Run(test.name, func(t *testing.T) {
			got := GetSingularResourceNameForMethod(test.method, model)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("GetSingularResourceNameForMethod mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
