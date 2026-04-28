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
			path: (&PathTemplate{}).
				WithLiteral("projects").WithVariableNamed("project"),
			fields: []*Field{
				{
					Name:              "project",
					Typez:             TypezString,
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
			path: (&PathTemplate{}).
				WithLiteral("projects").WithVariableNamed("project").
				WithLiteral("locations").WithVariableNamed("location"),
			fields: []*Field{
				{
					Name:              "project",
					Typez:             TypezString,
					ResourceReference: &ResourceReference{Type: "cloudresourcemanager.googleapis.com/Project"},
				},
				{
					Name:              "location",
					Typez:             TypezString, // Often locations are string IDs
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
			path: (&PathTemplate{}).
				WithLiteral("projects").WithVariableNamed("parent", "project"),
			fields: []*Field{
				{
					Name:  "parent",
					Typez: TypezMessage,
					MessageType: &Message{
						Fields: []*Field{
							{
								Name:              "project",
								Typez:             TypezString,
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
			path: (&PathTemplate{}).
				WithLiteral("v1").
				WithVariable(NewPathVariable("name").WithLiteral("projects").WithMatch().WithLiteral("locations").WithMatch()),
			fields: []*Field{
				{
					Name:              "name",
					Typez:             TypezString,
					ResourceReference: &ResourceReference{Type: "example.googleapis.com/Resource"},
				},
			},
			want: &TargetResource{
				FieldPaths: [][]string{{"name"}},
				Template:   ParseTemplateForTest("//test-api.googleapis.com/{name}"),
			},
		},
		{
			name:      "explicit: crop trailing literals (API Keys keyString case)",
			serviceID: "apikeys.googleapis.com",
			path: (&PathTemplate{}).
				WithLiteral("v2").
				WithVariable(NewPathVariable("name").WithLiteral("projects").WithMatch().WithLiteral("locations").WithMatch().WithLiteral("keys").WithMatch()).
				WithLiteral("keyString"),
			fields: []*Field{
				{
					Name:              "name",
					Typez:             TypezString,
					ResourceReference: &ResourceReference{Type: "apikeys.googleapis.com/Key"},
				},
			},
			want: &TargetResource{
				FieldPaths: [][]string{{"name"}},
				Template:   ParseTemplateForTest("//test-api.googleapis.com/{name}"),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, binding := setupTestModel(test.serviceID, test.path, test.fields)
			IdentifyTargetResources(model, true)

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
			path: (&PathTemplate{}).
				WithLiteral("projects").WithVariableNamed("project"),
			fields: []*Field{
				{Name: "project", Typez: TypezString}, // No ResourceReference
			},
		},
		{
			name:      "Explicit: partial reference returns nil",
			serviceID: "any.service",
			path: (&PathTemplate{}).
				WithLiteral("projects").WithVariableNamed("project").
				WithLiteral("glossaries").WithVariableNamed("glossary"),
			fields: []*Field{
				{
					Name:              "project",
					Typez:             TypezString,
					ResourceReference: &ResourceReference{Type: "cloudresourcemanager.googleapis.com/Project"},
				},
				{
					Name:  "glossary",
					Typez: TypezString,
					// No ResourceReference on the second variable
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, binding := setupTestModel(test.serviceID, test.path, test.fields)
			IdentifyTargetResources(model, false)

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
			path: (&PathTemplate{}).
				WithLiteral("projects").WithVariableNamed("project").
				WithLiteral("locations").WithVariableNamed("location"),
			fields: []*Field{
				{Name: "project", Typez: TypezString},
				{Name: "location", Typez: TypezString},
			},
			want: &TargetResource{
				FieldPaths: [][]string{{"project"}, {"location"}},
				Template:   ParseTemplateForTest("//test-api.googleapis.com/projects/{project}/locations/{location}"),
			},
		},
		{
			name:      "heuristic: learns custom tokens from GET methods",
			serviceID: ".google.cloud.compute.v1.Instances", // eligible
			path: (&PathTemplate{}).
				WithLiteral("projects").WithVariableNamed("project").
				WithLiteral("zones").WithVariableNamed("zone").
				WithLiteral("instances").WithVariableNamed("instance"),
			fields: []*Field{
				{Name: "project", Typez: TypezString},
				{Name: "zone", Typez: TypezString},
				{Name: "instance", Typez: TypezString},
			},
			getPaths: []*PathTemplate{
				(&PathTemplate{}).
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
			name:      "heuristic: paths with standalone literals without variables (e.g. global)",
			serviceID: ".google.cloud.compute.v1.BackendServices",
			path: (&PathTemplate{}).
				WithLiteral("projects").WithVariableNamed("project").
				WithLiteral("global").
				WithLiteral("backendServices").WithVariableNamed("backend_service"),
			fields: []*Field{
				{Name: "project", Typez: TypezString},
				{Name: "backend_service", Typez: TypezString},
			},
			getPaths: []*PathTemplate{
				(&PathTemplate{}).
					WithLiteral("projects").WithVariableNamed("project").
					WithLiteral("global").
					WithLiteral("backendServices").WithVariableNamed("backend_service"),
			},
			want: &TargetResource{
				FieldPaths: [][]string{{"project"}, {"backend_service"}},
				Template:   ParseTemplateForTest("//test-api.googleapis.com/projects/{project}/global/backendServices/{backend_service}"),
			},
		},
		{
			name:      "heuristic: path with non-variable standalone literal",
			serviceID: ".google.cloud.example.v1.Service",
			path: (&PathTemplate{}).
				WithLiteral("v1").
				WithLiteral("projects").
				WithLiteral("xyz").
				WithLiteral("global").
				WithLiteral("foos").WithVariableNamed("foo"),
			fields: []*Field{
				{Name: "foo", Typez: TypezString},
			},
			getPaths: []*PathTemplate{
				(&PathTemplate{}).
					WithLiteral("v1").
					WithLiteral("projects").
					WithLiteral("xyz").
					WithLiteral("global").
					WithLiteral("foos").WithVariableNamed("foo"),
			},
			want: &TargetResource{
				FieldPaths: [][]string{{"foo"}},
				Template:   ParseTemplateForTest("//test-api.googleapis.com/projects/xyz/global/foos/{foo}"),
			},
		},
		{
			name:      "heuristic: paths with un-grouped variable after version string",
			serviceID: ".google.cloud.compute.v1.Instances",
			path: (&PathTemplate{}).
				WithLiteral("v1").WithVariableNamed("resource").
				WithLiteral("children").WithVariableNamed("child"),
			fields: []*Field{
				{Name: "resource", Typez: TypezString},
				{Name: "child", Typez: TypezString},
			},
			getPaths: []*PathTemplate{
				(&PathTemplate{}).
					WithLiteral("v1").WithVariableNamed("resource").
					WithLiteral("children").WithVariableNamed("child"),
			},
			want: &TargetResource{
				FieldPaths: [][]string{{"resource"}, {"child"}},
				Template:   ParseTemplateForTest("//test-api.googleapis.com/{resource}/children/{child}"),
			},
		},
		{
			name:      "heuristic: valid compute path with version string",
			serviceID: ".google.cloud.compute.v1.Instances",
			path: (&PathTemplate{}).
				WithLiteral("v1").
				WithLiteral("projects").WithVariableNamed("project").
				WithLiteral("locations").WithVariableNamed("location"),
			fields: []*Field{
				{Name: "project", Typez: TypezString},
				{Name: "location", Typez: TypezString},
			},
			want: &TargetResource{
				FieldPaths: [][]string{{"project"}, {"location"}},
				Template:   ParseTemplateForTest("//test-api.googleapis.com/projects/{project}/locations/{location}"),
			},
		},
		{
			name:      "heuristic: stops at trailing action",
			serviceID: ".google.cloud.compute.v1.Instances",
			path: (&PathTemplate{}).
				WithLiteral("projects").WithVariableNamed("project").
				WithLiteral("zones").WithVariableNamed("zone").
				WithLiteral("instances").WithVariableNamed("instance").
				WithLiteral("start").WithVariableNamed("action_id"), // "start" is not a collection
			fields: []*Field{
				{Name: "project", Typez: TypezString},
				{Name: "zone", Typez: TypezString},
				{Name: "instance", Typez: TypezString},
				{Name: "action_id", Typez: TypezString},
			},
			getPaths: []*PathTemplate{
				(&PathTemplate{}).
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
			name:      "heuristic: stops at unknown segment",
			serviceID: ".google.cloud.compute.v1.Instances",
			path: (&PathTemplate{}).
				WithLiteral("projects").WithVariableNamed("project").
				WithLiteral("unknown").WithVariableNamed("other").
				WithLiteral("instances").WithVariableNamed("instance"),
			fields: []*Field{
				{Name: "project", Typez: TypezString},
				{Name: "other", Typez: TypezString},
				{Name: "instance", Typez: TypezString},
			},
			getPaths: []*PathTemplate{
				(&PathTemplate{}).
					WithLiteral("projects").WithVariableNamed("project").
					WithLiteral("zones").WithVariableNamed("zone").
					WithLiteral("instances").WithVariableNamed("instance"),
			},
			want: &TargetResource{
				FieldPaths: [][]string{{"project"}},
				Template:   ParseTemplateForTest("//test-api.googleapis.com/projects/{project}"),
			},
		},
		{
			name:      "heuristic: skips if input field missing",
			serviceID: ".google.cloud.compute.v1.Instances",
			path: (&PathTemplate{}).
				WithLiteral("projects").WithVariableNamed("project"),
			fields: []*Field{}, // No fields
			want:   nil,
		},
		{
			name:      "heuristic: skips non-collection literal without 's'",
			serviceID: ".google.cloud.compute.v1.Instances",
			path: (&PathTemplate{}).
				WithLiteral("metadata").WithVariableNamed("data"), // does not match fallback
			fields: []*Field{
				{Name: "data", Typez: TypezString},
			},
			want: nil,
		},
		{
			name:      "heuristic: known custom resource after standalone literal",
			serviceID: ".google.cloud.compute.v1.CrossSiteNetworks",
			path: (&PathTemplate{}).
				WithLiteral("projects").WithVariableNamed("project").
				WithLiteral("global").
				WithLiteral("crossSiteNetworks").WithVariableNamed("cross_site_network"),
			fields: []*Field{
				{Name: "project", Typez: TypezString},
				{Name: "cross_site_network", Typez: TypezString},
			},
			getPaths: []*PathTemplate{
				(&PathTemplate{}).
					WithLiteral("projects").WithVariableNamed("project").
					WithLiteral("global").
					WithLiteral("crossSiteNetworks").WithVariableNamed("cross_site_network"),
			},
			want: &TargetResource{
				FieldPaths: [][]string{{"project"}, {"cross_site_network"}},
				Template:   ParseTemplateForTest("//test-api.googleapis.com/projects/{project}/global/crossSiteNetworks/{cross_site_network}"),
			},
		},
		{
			name:      "heuristic: unknown custom resource falls back to parent",
			serviceID: ".google.cloud.compute.v1.CrossSiteNetworks",
			path: (&PathTemplate{}).
				WithLiteral("projects").WithVariableNamed("project").
				WithLiteral("global").
				WithLiteral("crossSiteNetworks").WithVariableNamed("cross_site_network"),
			fields: []*Field{
				{Name: "project", Typez: TypezString},
				{Name: "cross_site_network", Typez: TypezString},
			},
			getPaths: []*PathTemplate{
				(&PathTemplate{}).
					WithLiteral("projects").WithVariableNamed("project"),
			},
			want: &TargetResource{
				FieldPaths: [][]string{{"project"}},
				Template:   ParseTemplateForTest("//test-api.googleapis.com/projects/{project}"),
			},
		},
		{
			name:      "heuristic: multiple standalone literals before known resource",
			serviceID: ".google.cloud.compute.v1.FirewallPolicies",
			path: (&PathTemplate{}).
				WithLiteral("locations").
				WithLiteral("global").
				WithLiteral("firewallPolicies").WithVariableNamed("resource"),
			fields: []*Field{
				{Name: "resource", Typez: TypezString},
			},
			getPaths: []*PathTemplate{
				(&PathTemplate{}).
					WithLiteral("locations").
					WithLiteral("global").
					WithLiteral("firewallPolicies").WithVariableNamed("resource"),
			},
			want: &TargetResource{
				FieldPaths: [][]string{{"resource"}},
				Template:   ParseTemplateForTest("//test-api.googleapis.com/locations/global/firewallPolicies/{resource}"),
			},
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
					Service: model.Services[0],
				}
				model.Services[0].Methods = append(model.Services[0].Methods, m)
			}
			IdentifyTargetResources(model, true)

			got := binding.TargetResource
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIdentifyTargetResources_HeuristicsDisabled(t *testing.T) {
	// A setup that would normally match the heuristics
	serviceID := ".google.cloud.compute.v1.Instances"
	path := (&PathTemplate{}).
		WithLiteral("projects").WithVariableNamed("project").
		WithLiteral("locations").WithVariableNamed("location")
	fields := []*Field{
		{Name: "project", Typez: TypezString},
		{Name: "location", Typez: TypezString},
	}

	model, binding := setupTestModel(serviceID, path, fields)

	// Explicitly disable heuristics
	IdentifyTargetResources(model, false)

	// Since heuristics are disabled, it should not find the target resource
	if binding.TargetResource != nil {
		t.Errorf("IdentifyTargetResources(model, false) populated TargetResource %v, want nil", binding.TargetResource)
	}
}
