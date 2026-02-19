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
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestIdentifyTargetResources(t *testing.T) {
	lit := func(s string) PathSegment {
		return PathSegment{Literal: &s}
	}
	var_seg := func(fieldPath ...string) PathSegment {
		return PathSegment{Variable: &PathVariable{FieldPath: fieldPath}}
	}

	for _, test := range []struct {
		name      string
		serviceID string
		path      []PathSegment
		fields    []*Field
		want      *TargetResource
	}{
		{
			name:      "explicit: standard resource reference",
			serviceID: "any.service",
			path: []PathSegment{
				lit("projects"), var_seg("project"),
			},
			fields: []*Field{
				{
					Name:              "project",
					Typez:             STRING_TYPE,
					ResourceReference: &ResourceReference{Type: "cloudresourcemanager.googleapis.com/Project"},
				},
			},
			want: &TargetResource{
				FieldPaths: [][]string{{"project"}},
			},
		},
		{
			name:      "explicit: multiple resource references",
			serviceID: "any.service",
			path: []PathSegment{
				lit("projects"), var_seg("project"),
				lit("locations"), var_seg("location"),
			},
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
			},
		},
		{
			name:      "explicit: nested field reference",
			serviceID: "any.service",
			path: []PathSegment{
				lit("projects"), var_seg("parent", "project"),
			},
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
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			pathTemplate := &PathTemplate{Segments: test.path}
			binding := &PathBinding{PathTemplate: pathTemplate}
			method := &Method{
				Name: "TestMethod",
				InputType: &Message{
					Fields: test.fields,
				},
				PathInfo: &PathInfo{
					Bindings: []*PathBinding{binding},
				},
			}
			service := &Service{
				ID:      test.serviceID,
				Methods: []*Method{method},
			}
			model := &API{
				Services: []*Service{service},
			}

			IdentifyTargetResources(model)

			got := binding.TargetResource
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(TargetResource{}, "PathTemplate")); diff != "" {
				t.Errorf("IdentifyTargetResources() mismatch (-want +got):\n%s", diff)
			}
			// Check if PathTemplate reference is correct (should be the same object)
			if got != nil && got.PathTemplate != pathTemplate {
				t.Errorf("PathTemplate reference mismatch")
			}
		})
	}
}

func TestIdentifyTargetResources_NoMatch(t *testing.T) {
	lit := func(s string) PathSegment {
		return PathSegment{Literal: &s}
	}
	var_seg := func(fieldPath ...string) PathSegment {
		return PathSegment{Variable: &PathVariable{FieldPath: fieldPath}}
	}

	for _, test := range []struct {
		name      string
		serviceID string
		path      []PathSegment
		fields    []*Field
	}{
		{
			name:      "Explicit: missing reference returns nil",
			serviceID: "any.service",
			path: []PathSegment{
				lit("projects"), var_seg("project"),
			},
			fields: []*Field{
				{Name: "project", Typez: STRING_TYPE}, // No ResourceReference
			},
		},
		{
			name:      "Explicit: partial reference returns nil",
			serviceID: "any.service",
			path: []PathSegment{
				lit("projects"), var_seg("project"),
				lit("glossaries"), var_seg("glossary"),
			},
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
			pathTemplate := &PathTemplate{Segments: test.path}
			binding := &PathBinding{PathTemplate: pathTemplate}
			method := &Method{
				Name: "TestMethod",
				InputType: &Message{
					Fields: test.fields,
				},
				PathInfo: &PathInfo{
					Bindings: []*PathBinding{binding},
				},
			}
			service := &Service{
				ID:      test.serviceID,
				Methods: []*Method{method},
			}
			model := &API{
				Services: []*Service{service},
			}

			IdentifyTargetResources(model)

			got := binding.TargetResource
			if got != nil {
				t.Errorf("IdentifyTargetResources() = %v, want nil", got)
			}
		})
	}
}
