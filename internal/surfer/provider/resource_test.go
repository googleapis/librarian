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

package provider

import (
	"strings"
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
			name:     "Standard",
			segments: parseResourcePattern("projects/{project}/locations/{location}/instances/{instance}"),
			want:     "instances",
		},
		{
			name:     "Short",
			segments: parseResourcePattern("shelves/{shelf}"),
			want:     "shelves",
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
			got := GetPluralFromSegments(test.segments)
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
			name:     "Standard",
			segments: parseResourcePattern("projects/{project}/locations/{location}/instances/{instance}"),
			want:     parseResourcePattern("projects/{project}/locations/{location}"),
		},
		{
			name:     "Root",
			segments: parseResourcePattern("projects/{project}"),
			want:     []api.PathSegment{},
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
			got := GetParentFromSegments(test.segments)
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
			name:     "Standard",
			segments: parseResourcePattern("projects/{project}/locations/{location}/instances/{instance}"),
			want:     "instance",
		},
		{
			name:     "Short",
			segments: parseResourcePattern("shelves/{shelf}"),
			want:     "shelf",
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
			got := GetSingularFromSegments(test.segments)
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
			name:     "Standard",
			segments: parseResourcePattern("projects/{project}/locations/{location}/instances/{instance}"),
			want:     "projects.locations.instances",
		},
		{
			name:     "Short",
			segments: parseResourcePattern("shelves/{shelf}"),
			want:     "shelves",
		},
		{
			name:     "Root",
			segments: parseResourcePattern("projects/{project}"),
			want:     "projects",
		},
		{
			name:     "Mixed",
			segments: parseResourcePattern("organizations/{organization}/locations/{location}/clusters/{cluster}"),
			want:     "organizations.locations.clusters",
		},
		{
			name:     "Global",
			segments: parseResourcePattern("projects/{project}/global/networks/{network}"),
			want:     "projects.networks",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := GetCollectionPathFromSegments(test.segments)
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
			name:     "Standard Regional",
			segments: parseResourcePattern("v1/projects/{project}/locations/{location}/instances/{instance}"),
			want:     "projects.locations.instances",
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
			got := ExtractPathFromSegments(test.segments)
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
			got := IsPrimaryResource(test.field, test.method)
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
						api.NewTestField("name").WithResourceReference("example.googleapis.com/Instance"),
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
						api.NewTestField("parent").WithChildTypeReference("example.googleapis.com/Instance"),
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
						api.NewTestField("name").WithResourceReference("example.googleapis.com/Other"),
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
			got := GetResourceForMethod(test.method, model)
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
			parseResourcePattern("instances/{instance}"),
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
						api.NewTestField("parent").WithChildTypeReference("example.googleapis.com/Instance"),
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
						api.NewTestField("parent").WithChildTypeReference("example.googleapis.com/Book"),
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
						api.NewTestField("parent").WithChildTypeReference("example.googleapis.com/Unknown"),
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
			got := GetPluralResourceNameForMethod(test.method, model)
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
			parseResourcePattern("instances/{instance}"),
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
						api.NewTestField("parent").WithChildTypeReference("example.googleapis.com/Instance"),
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
						api.NewTestField("parent").WithChildTypeReference("example.googleapis.com/Book"),
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
						api.NewTestField("parent").WithChildTypeReference("example.googleapis.com/Unknown"),
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
			got := GetSingularResourceNameForMethod(test.method, model)
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

// parseResourcePattern converts a resource pattern string into a
// []api.PathSegment slice for testing. It handles AIP resource patterns
// (e.g., "projects/{project}/locations/{location}"). Variables
// automatically get a single-segment wildcard match.
func parseResourcePattern(pattern string) []api.PathSegment {
	var segments []api.PathSegment
	for part := range strings.SplitSeq(pattern, "/") {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			name := part[1 : len(part)-1]
			segments = append(segments, *api.NewPathSegment().WithVariable(api.NewPathVariable(name).WithMatch()))
		} else {
			segments = append(segments, *api.NewPathSegment().WithLiteral(part))
		}
	}
	return segments
}
