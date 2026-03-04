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

package api

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// setupTestModel helper creates a minimal API model for testing resource identification.
func setupTestModel(serviceID string, pathTemplate *PathTemplate, fields []*Field) (*API, *PathBinding) {
	binding := &PathBinding{PathTemplate: pathTemplate}
	method := &Method{
		Name: "TestMethod",
		InputType: &Message{
			Fields: fields,
		},
		PathInfo: &PathInfo{
			Bindings: []*PathBinding{binding},
		},
	}
	service := &Service{
		ID:      serviceID,
		Methods: []*Method{method},
	}
	method.Service = service
	model := &API{
		Name:     "test-api",
		Services: []*Service{service},
	}
	method.Model = model
	service.Model = model
	return model, binding
}

func TestIdentifyTargetResources(t *testing.T) {
	for _, test := range []struct {
		name      string
		serviceID string
		path      *PathTemplate
		fields    []*Field
		want      *TargetResource
	}{
		{
			name:      "explicit: standard resource reference",
			serviceID: "any.service",
			path: NewPathTemplate().
				WithLiteral("projects").WithVariableNamed("project"),
			fields: []*Field{
				{
					Name:              "project",
					Typez:             STRING_TYPE,
					ResourceReference: &ResourceReference{Type: "cloudresourcemanager.googleapis.com/Project"},
				},
			},
			want: &TargetResource{
				FieldPaths: [][]string{{"project"}},
				Template:   ParseTemplateForTest("//test-api.googleapis.com/projects/{project}"),
			},
		},
		{
			name:      "explicit: multiple resource references",
			serviceID: "any.service",
			path: NewPathTemplate().
				WithLiteral("projects").WithVariableNamed("project").
				WithLiteral("locations").WithVariableNamed("location"),
			fields: []*Field{
				{
					Name:              "project",
					Typez:             STRING_TYPE,
					ResourceReference: &ResourceReference{Type: "cloudresourcemanager.googleapis.com/Project"},
				},
				{
					Name:              "location",
					Typez:             STRING_TYPE, // Often locations are string IDs
					ResourceReference: &ResourceReference{Type: "locations.googleapis.com/Location"},
				},
			},
			want: &TargetResource{
				FieldPaths: [][]string{{"project"}, {"location"}},
				Template:   ParseTemplateForTest("//test-api.googleapis.com/projects/{project}/locations/{location}"),
			},
		},
		{
			name:      "explicit: nested field reference",
			serviceID: "any.service",
			path: NewPathTemplate().
				WithLiteral("projects").WithVariableNamed("parent", "project"),
			fields: []*Field{
				{
					Name:  "parent",
					Typez: MESSAGE_TYPE,
					MessageType: &Message{
						Fields: []*Field{
							{
								Name:              "project",
								Typez:             STRING_TYPE,
								ResourceReference: &ResourceReference{Type: "cloudresourcemanager.googleapis.com/Project"},
							},
						},
					},
				},
			},
			want: &TargetResource{
				FieldPaths: [][]string{{"parent", "project"}},
				Template:   ParseTemplateForTest("//test-api.googleapis.com/projects/{parent.project}"),
			},
		},
		{
			name:      "explicit: complex variable pattern treated as full name",
			serviceID: "any.service",
			path: NewPathTemplate().
				WithLiteral("v1").
				WithVariable(NewPathVariable("name").WithLiteral("projects").WithMatch().WithLiteral("locations").WithMatch()),
			fields: []*Field{
				{
					Name:              "name",
					Typez:             STRING_TYPE,
					ResourceReference: &ResourceReference{Type: "example.googleapis.com/Resource"},
				},
			},
			want: &TargetResource{
				FieldPaths: [][]string{{"name"}},
				Template:   ParseTemplateForTest("//test-api.googleapis.com/v1/{name}"),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, binding := setupTestModel(test.serviceID, test.path, test.fields)
			IdentifyTargetResources(model)

			got := binding.TargetResource
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIdentifyTargetResources_NoMatch(t *testing.T) {
	for _, test := range []struct {
		name      string
		serviceID string
		path      *PathTemplate
		fields    []*Field
	}{
		{
			name:      "Explicit: missing reference returns nil",
			serviceID: "any.service",
			path: NewPathTemplate().
				WithLiteral("projects").WithVariableNamed("project"),
			fields: []*Field{
				{Name: "project", Typez: STRING_TYPE}, // No ResourceReference
			},
		},
		{
			name:      "Explicit: partial reference returns nil",
			serviceID: "any.service",
			path: NewPathTemplate().
				WithLiteral("projects").WithVariableNamed("project").
				WithLiteral("glossaries").WithVariableNamed("glossary"),
			fields: []*Field{
				{
					Name:              "project",
					Typez:             STRING_TYPE,
					ResourceReference: &ResourceReference{Type: "cloudresourcemanager.googleapis.com/Project"},
				},
				{
					Name:  "glossary",
					Typez: STRING_TYPE,
					// No ResourceReference on the second variable
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, binding := setupTestModel(test.serviceID, test.path, test.fields)
			IdentifyTargetResources(model)

			got := binding.TargetResource
			if got != nil {
				t.Errorf("IdentifyTargetResources() = %v, want nil", got)
			}
		})
	}
}

func TestIdentifyTargetResources_Heuristic(t *testing.T) {
	for _, test := range []struct {
		name      string
		serviceID string
		path      *PathTemplate
		fields    []*Field
		getPaths  []*PathTemplate // Path templates for Get/List methods to build vocabulary
		resources []*Resource
		want      *TargetResource
	}{

		{
			name:      "heuristic: standard infrastructure tokens",
			serviceID: ".google.cloud.compute.v1.Instances", // eligible
			path: NewPathTemplate().
				WithLiteral("projects").WithVariableNamed("project").
				WithLiteral("locations").WithVariableNamed("location"),
			fields: []*Field{
				{Name: "project", Typez: STRING_TYPE},
				{Name: "location", Typez: STRING_TYPE},
			},
			want: &TargetResource{
				FieldPaths: [][]string{{"project"}, {"location"}},
				Template:   ParseTemplateForTest("//test-api.googleapis.com/projects/{project}/locations/{location}"),
			},
		},
		{
			name:      "heuristic: learns custom tokens from GET methods",
			serviceID: ".google.cloud.compute.v1.Instances", // eligible
			path: NewPathTemplate().
				WithLiteral("projects").WithVariableNamed("project").
				WithLiteral("zones").WithVariableNamed("zone").
				WithLiteral("instances").WithVariableNamed("instance"),
			fields: []*Field{
				{Name: "project", Typez: STRING_TYPE},
				{Name: "zone", Typez: STRING_TYPE},
				{Name: "instance", Typez: STRING_TYPE},
			},
			getPaths: []*PathTemplate{
				NewPathTemplate().
					WithLiteral("projects").WithVariableNamed("project").
					WithLiteral("zones").WithVariableNamed("zone").
					WithLiteral("instances").WithVariableNamed("instance"),
			},
			want: &TargetResource{
				FieldPaths: [][]string{{"project"}, {"zone"}, {"instance"}},
				Template:   ParseTemplateForTest("//test-api.googleapis.com/projects/{project}/zones/{zone}/instances/{instance}"),
			},
		},

		{
			name:      "heuristic: not eligible service",
			serviceID: "any.service", // not eligible
			path: NewPathTemplate().
				WithLiteral("projects").WithVariableNamed("project"),
			fields: []*Field{
				{Name: "project", Typez: STRING_TYPE},
			},
			resources: nil,
			want:      nil,
		},

		{
			name:      "heuristic: skips if input field missing",
			serviceID: ".google.cloud.compute.v1.Instances",
			path: NewPathTemplate().
				WithLiteral("projects").WithVariableNamed("project"),
			fields: []*Field{}, // No fields
			want:   nil,
		},

		{
			name:      "heuristic: skips non-collection literal without 's'",
			serviceID: ".google.cloud.compute.v1.Instances",
			path: NewPathTemplate().
				WithLiteral("metadata").WithVariableNamed("data"), // does not match fallback
			fields: []*Field{
				{Name: "data", Typez: STRING_TYPE},
			},
			want: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, binding := setupTestModel(test.serviceID, test.path, test.fields)
			if test.resources != nil {
				model.ResourceDefinitions = test.resources
			}

			// Add Get methods to populate vocabulary
			for _, p := range test.getPaths {
				m := &Method{
					Name:          "GetSomething",
					IsAIPStandard: true,
					PathInfo: &PathInfo{
						Bindings: []*PathBinding{{PathTemplate: p}},
					},
				}
				model.Services[0].Methods = append(model.Services[0].Methods, m)
			}
			IdentifyTargetResources(model)

			got := binding.TargetResource
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
