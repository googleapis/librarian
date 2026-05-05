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
	"bytes"
	"log"
	"os"
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
				*(&api.PathSegment{}).WithLiteral("projects"),
				*(&api.PathSegment{}).WithVariable(api.NewPathVariable("project").WithMatch()),
				*(&api.PathSegment{}).WithLiteral("locations"),
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
				*(&api.PathSegment{}).WithLiteral("projects"),
			},
			want: nil,
		},
		{
			name: "Invalid Pattern (Ends in Literal)",
			segments: []api.PathSegment{
				*(&api.PathSegment{}).WithLiteral("projects"),
				*(&api.PathSegment{}).WithVariable(api.NewPathVariable("project").WithMatch()),
				*(&api.PathSegment{}).WithLiteral("locations"),
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
				*(&api.PathSegment{}).WithLiteral("projects"),
				*(&api.PathSegment{}).WithVariable(api.NewPathVariable("project").WithMatch()),
				*(&api.PathSegment{}).WithLiteral("locations"),
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
				*(&api.PathSegment{}).WithLiteral("v1"),
				*(&api.PathSegment{}).WithVariable(api.NewPathVariable("name").WithLiteral("projects").WithMatch().WithLiteral("locations").WithMatch().WithLiteral("instances").WithMatch()),
			},
			want: "projects.locations.instances",
		},
		{
			name: "Trailing Literal (List)",
			segments: []api.PathSegment{
				*(&api.PathSegment{}).WithLiteral("v1"),
				*(&api.PathSegment{}).WithVariable(api.NewPathVariable("name").WithLiteral("projects").WithMatch().WithLiteral("locations").WithMatch()),
				*(&api.PathSegment{}).WithLiteral("instances"),
			},
			want: "projects.locations.instances",
		},
		{
			name: "No Version",
			segments: []api.PathSegment{
				*(&api.PathSegment{}).WithLiteral("projects"),
				*(&api.PathSegment{}).WithVariable(api.NewPathVariable("project")),
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

func TestIsPrimaryResourceField(t *testing.T) {
	for _, test := range []struct {
		name   string
		field  *api.Field
		method *api.Method
		want   bool
	}{
		{
			name:  "Create Method - Primary Resource Parent",
			field: &api.Field{Name: "parent"},
			method: &api.Method{
				Name:      "CreateInstance",
				InputType: &api.Message{},
			},
			want: true,
		},
		{
			name:  "Get Method - Primary Resource Name",
			field: &api.Field{Name: "name"},
			method: &api.Method{
				Name:      "GetInstance",
				InputType: &api.Message{},
			},
			want: true,
		},
		{
			name:  "Delete Method - Primary Resource Name",
			field: &api.Field{Name: "name"},
			method: &api.Method{
				Name:      "DeleteInstance",
				InputType: &api.Message{},
			},
			want: true,
		},
		{
			name:  "Update Method - Primary Resource Name",
			field: &api.Field{Name: "name"},
			method: &api.Method{
				Name:      "UpdateInstance",
				InputType: &api.Message{},
			},
			want: true,
		},
		{
			name:  "List Method - Primary Resource Parent",
			field: &api.Field{Name: "parent"},
			method: &api.Method{
				Name:      "ListInstances",
				InputType: &api.Message{},
			},
			want: true,
		},
		{
			name:  "List Operations Method - Primary Resource Name",
			field: &api.Field{Name: "name"},
			method: &api.Method{
				Name:            "ListOperations",
				SourceServiceID: ".google.longrunning.Operations",
				InputType:       &api.Message{},
			},
			want: true,
		},
		{
			name:  "Non-Primary Field",
			field: &api.Field{Name: "display_name"},
			method: &api.Method{
				Name:      "GetInstance",
				InputType: &api.Message{},
			},
			want: false,
		},
		{
			name:  "Nil InputType",
			field: &api.Field{Name: "name"},
			method: &api.Method{
				Name: "GetInstance",
			},
			want: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := IsPrimaryResourceField(test.field, test.method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsResourceIdField(t *testing.T) {
	mockModel := &api.API{
		ResourceDefinitions: []*api.Resource{
			{
				Type: "example.googleapis.com/Instance",
			},
		},
	}

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
			name:  "Create Method - Not Resource ID",
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
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := IsResourceIdField(test.field, test.method, mockModel)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetResourceForMethod(t *testing.T) {
	instanceResource := &api.Resource{Type: "example.googleapis.com/Instance"}

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
			name: "GetOperation Method - Pre-defined Resource",
			method: &api.Method{
				Name:            GetOperation,
				SourceServiceID: ".google.longrunning.Operations",
				InputType:       &api.Message{},
			},
			resourceDefs: []*api.Resource{
				{
					Type:     operationResourceType,
					Singular: "operation",
					Plural:   "operations",
				},
			},
			want: &api.Resource{
				Type:     operationResourceType,
				Singular: "operation",
				Plural:   "operations",
			},
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
			got := GetResourceNameFromType(test.typeStr)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindNameField(t *testing.T) {
	nameField := api.NewTestField("name")
	otherField := api.NewTestField("other")

	for _, test := range []struct {
		name     string
		resource *api.Resource
		want     *api.Field
	}{
		{
			name: "HasNameField",
			resource: &api.Resource{
				Self: &api.Message{
					Fields: []*api.Field{otherField, nameField},
				},
			},
			want: nameField,
		},
		{
			name: "NoNameField",
			resource: &api.Resource{
				Self: &api.Message{
					Fields: []*api.Field{otherField},
				},
			},
			want: nil,
		},
		{
			name: "NilSelf",
			resource: &api.Resource{
				Self: nil,
			},
			want: nil,
		},
		{
			name:     "NilResource",
			resource: nil,
			want:     nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := FindNameField(test.resource)
			if got != test.want {
				t.Errorf("FindNameField() = %v, want %v", got, test.want)
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
			segments = append(segments, *(&api.PathSegment{}).WithVariable(api.NewPathVariable(name).WithMatch()))
		} else {
			segments = append(segments, *(&api.PathSegment{}).WithLiteral(part))
		}
	}
	return segments
}

func TestGetLiteralSegments(t *testing.T) {
	for _, test := range []struct {
		name     string
		segments []api.PathSegment
		want     []string
	}{
		{
			name:     "Standard",
			segments: parseResourcePattern("projects/{project}/locations/{location}/instances/{instance}"),
			want:     []string{"projects", "locations", "instances"},
		},
		{
			name:     "Version Filtered",
			segments: parseResourcePattern("v1/projects/{project}"),
			want:     []string{"projects"},
		},
		{
			name: "With Wildcards",
			segments: []api.PathSegment{
				*(&api.PathSegment{}).WithLiteral("projects"),
				*(&api.PathSegment{}).WithVariable(api.NewPathVariable("name").WithLiteral("foo").WithMatch().WithLiteral("bar")),
			},
			want: []string{"projects", "foo", "bar"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := GetLiteralSegments(test.segments)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetResourceForPath(t *testing.T) {
	instanceResource := &api.Resource{
		Type: "example.googleapis.com/Instance",
		Patterns: []api.ResourcePattern{
			parseResourcePattern("projects/{project}/locations/{location}/instances/{instance}"),
		},
	}
	model := &api.API{
		ResourceDefinitions: []*api.Resource{instanceResource},
	}

	for _, test := range []struct {
		name string
		path []string
		want *api.Resource
	}{
		{
			name: "Match First Pattern",
			path: []string{"projects", "locations", "instances"},
			want: instanceResource,
		},
		{
			name: "No Match",
			path: []string{"projects", "locations", "unknown"},
			want: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := GetResourceForPath(model, test.path)
			if got != test.want {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestGetResourceTypeName(t *testing.T) {
	instanceResource := &api.Resource{
		Type:     "example.googleapis.com/Instance",
		Singular: "instance_custom",
		Patterns: []api.ResourcePattern{
			parseResourcePattern("projects/{project}/locations/{location}/instances/{instance}"),
		},
	}
	noAnnoResource := &api.Resource{
		Type: "example.googleapis.com/NoAnnotation",
		Patterns: []api.ResourcePattern{
			parseResourcePattern("projects/{project}/locations/{location}/noAnnotations/{noAnnotation}"),
		},
	}
	model := &api.API{
		ResourceDefinitions: []*api.Resource{instanceResource, noAnnoResource},
	}

	for _, test := range []struct {
		name string
		path []string
		want string
	}{
		{
			name: "Match With Annotations",
			path: []string{"projects", "locations", "instances"},
			want: "instance_custom",
		},
		{
			name: "Match Without Annotations",
			path: []string{"projects", "locations", "noAnnotations"},
			want: "NoAnnotation",
		},
		{
			name: "No Match",
			path: []string{"projects", "locations", "unknown"},
			want: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := GetResourceTypeName(model, test.path)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestGetPluralResourceTypeName(t *testing.T) {
	instanceResource := &api.Resource{
		Type:   "example.googleapis.com/Instance",
		Plural: "instances_custom",
		Patterns: []api.ResourcePattern{
			parseResourcePattern("projects/{project}/locations/{location}/instances/{instance}"),
		},
	}
	model := &api.API{
		ResourceDefinitions: []*api.Resource{instanceResource},
	}

	for _, test := range []struct {
		name string
		path []string
		want string
	}{
		{
			name: "Match With Annotations",
			path: []string{"projects", "locations", "instances"},
			want: "instances_custom",
		},
		{
			name: "No Match",
			path: []string{"projects", "locations", "unknown"},
			want: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := GetPluralResourceTypeName(model, test.path)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func Test_getAllResources(t *testing.T) {
	fileResource := &api.Resource{Type: "example.googleapis.com/File"}
	messageResource := &api.Resource{Type: "example.googleapis.com/Message"}

	model := &api.API{
		ResourceDefinitions: []*api.Resource{fileResource, messageResource},
		Services: []*api.Service{
			{
				Methods: []*api.Method{
					{
						Name:            GetOperation,
						SourceServiceID: ".google.longrunning.Operations",
						PathInfo: &api.PathInfo{
							Bindings: []*api.PathBinding{
								{
									PathTemplate: &api.PathTemplate{
										Segments: []api.PathSegment{
											*(&api.PathSegment{}).WithLiteral("v1"),
											*(&api.PathSegment{}).WithLiteral("operations"),
											*(&api.PathSegment{}).WithVariable(api.NewPathVariable("operation")),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	got := getAllResources(model)

	if len(got) != 3 {
		t.Errorf("getAllResources() returned %d resources, want 3", len(got))
	}

	expectedTypes := map[string]bool{
		"example.googleapis.com/File":          true,
		"example.googleapis.com/Message":       true,
		"longrunning.googleapis.com/Operation": true,
	}

	for _, r := range got {
		if !expectedTypes[r.Type] {
			t.Errorf("Unexpected resource type: %s", r.Type)
		}
	}
}

func Test_getAllResources_Warning(t *testing.T) {
	// Capture log output.
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	fileResource := &api.Resource{Type: "example.googleapis.com/File"}

	// An invalid path variable with a leading wildcard in GetOperation (will trigger inference error).
	model := &api.API{
		ResourceDefinitions: []*api.Resource{fileResource},
		Services: []*api.Service{
			{
				Methods: []*api.Method{
					{
						Name:            GetOperation,
						SourceServiceID: ".google.longrunning.Operations",
						PathInfo: &api.PathInfo{
							Bindings: []*api.PathBinding{
								{
									PathTemplate: &api.PathTemplate{
										Segments: []api.PathSegment{
											*(&api.PathSegment{}).WithLiteral("v1"),
											*(&api.PathSegment{}).WithVariable(
												api.NewPathVariable("name").
													WithMatch().
													WithLiteral("locations").WithMatch(),
											),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	got := getAllResources(model)

	// 1. Verify it gracefully skipped the operations resource (length should be 1, just the fileResource).
	if len(got) != 1 {
		t.Errorf("getAllResources() returned %d resources, want 1 (graceful skip)", len(got))
	}

	// 2. Verify the warning message was logged to stderr.
	logMsg := buf.String()
	if !strings.Contains(logMsg, "failed to infer operations resource") {
		t.Errorf("Expected warning log, got: %q", logMsg)
	}
}

func TestSingular(t *testing.T) {
	tests := []struct {
		name string
		word string
		want string
	}{
		{
			name: "Standard projects mapping",
			word: "projects",
			want: "project",
		},
		{
			name: "Already singular word stays singular",
			word: "project",
			want: "project",
		},
		{
			name: "Plural ies mapping",
			word: "policies",
			want: "policy",
		},
		{
			name: "Plural sses mapping",
			word: "addresses",
			want: "address",
		},
		{
			name: "Plural databases mapping",
			word: "databases",
			want: "database",
		},
		{
			name: "Plural xes mapping",
			word: "indexes",
			want: "index",
		},
		{
			name: "Plural ches mapping",
			word: "branches",
			want: "branch",
		},
		{
			name: "Plural shes mapping",
			word: "meshes",
			want: "mesh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := singular(tt.word)
			if got != tt.want {
				t.Errorf("singular(%q) = %q, want %q", tt.word, got, tt.want)
			}
		})
	}
}

func Test_resourceFromType(t *testing.T) {
	r1 := &api.Resource{Type: "example.googleapis.com/Instance"}
	r2 := &api.Resource{Type: "example.googleapis.com/Network"}
	model := &api.API{
		ResourceDefinitions: []*api.Resource{r1, r2},
	}

	tests := []struct {
		name         string
		resourceType string
		want         *api.Resource
	}{
		{
			name:         "Find First",
			resourceType: "example.googleapis.com/Instance",
			want:         r1,
		},
		{
			name:         "Find Second",
			resourceType: "example.googleapis.com/Network",
			want:         r2,
		},
		{
			name:         "Empty Type Returns Nil",
			resourceType: "",
			want:         nil,
		},
		{
			name:         "Unknown Type Returns Nil",
			resourceType: "example.googleapis.com/Unknown",
			want:         nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := resourceFromType(model, tt.resourceType)
			if got != tt.want {
				t.Errorf("resourceFromType() = %v, want %v", got, tt.want)
			}
		})
	}
}
