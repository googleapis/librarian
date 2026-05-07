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
	"fmt"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

const operationResourceType = "longrunning.googleapis.com/Operation"
const locationResourceType = "locations.googleapis.com/Location"

const (
	// GetOperation is the RPC method name for getting an operation's status.
	GetOperation = "GetOperation"
	// CancelOperation is the RPC method name for cancelling an active operation.
	CancelOperation = "CancelOperation"
	// DeleteOperation is the RPC method name for deleting a completed operation.
	DeleteOperation = "DeleteOperation"
	// ListOperations is the RPC method name for listing operations.
	ListOperations = "ListOperations"
	// GetLocation is the RPC method name for getting a location's details.
	GetLocation = "GetLocation"
	// ListLocations is the RPC method name for listing locations.
	ListLocations = "ListLocations"
)

// IsOperationsServiceMethod determines if the method belongs to the long-running operations service.
func IsOperationsServiceMethod(m *api.Method) bool {
	return m.SourceServiceID == ".google.longrunning.Operations"
}

// IsLocationsServiceMethod determines if the method belongs to the locations service.
func IsLocationsServiceMethod(m *api.Method) bool {
	return m.SourceServiceID == ".google.cloud.location.Locations"
}

// OperationMethodDocumentation returns the fallback help text for operations methods.
// Current documentation is stripped from the proto in the LRO service.
func OperationMethodDocumentation(methodName string) string {
	switch methodName {
	case GetOperation:
		return "The name of the operation resource."
	case CancelOperation:
		return "The name of the operation resource to be cancelled."
	case DeleteOperation:
		return "The name of the operation resource to be deleted."
	case ListOperations:
		return "The name of the operation's parent resource."
	}
	return ""
}

// LocationMethodDocumentation returns the fallback help text for locations methods.
// Current documentation is stripped from the proto in the Locations service.
func LocationMethodDocumentation(methodName string) string {
	switch methodName {
	case GetLocation:
		return "The name of the location resource."
	case ListLocations:
		return "The name of the location's parent resource."
	}
	return ""
}

// inferOperationResource creates a synthetic resource for operations based on the method's path.
func inferOperationResource(m *api.Method) (*api.Resource, error) {
	patterns, err := resourcePatterns(m)
	if err != nil {
		return nil, err
	}
	if len(patterns) == 0 {
		return nil, fmt.Errorf("operations mixin method %s has no HTTP path bindings", m.ID)
	}

	return &api.Resource{
		Type:     operationResourceType,
		Singular: "operation",
		Plural:   "operations",
		Patterns: patterns,
	}, nil
}

// inferLocationResource creates a synthetic resource for locations based on the method's path.
func inferLocationResource(m *api.Method) (*api.Resource, error) {
	patterns, err := resourcePatterns(m)
	if err != nil {
		return nil, err
	}
	if len(patterns) == 0 {
		return nil, fmt.Errorf("locations mixin method %s has no HTTP path bindings", m.ID)
	}

	return &api.Resource{
		Type:     locationResourceType,
		Singular: "location",
		Plural:   "locations",
		Patterns: patterns,
	}, nil
}

// resourcePatterns parses a method's HTTP bindings to construct resource patterns.
// For each method pattern like /v1/{name=projects/*/locations/*/operations/*}, create a corresponding
// resource pattern like /projects/{project}/locations/{location}/operations/{operation}.
func resourcePatterns(m *api.Method) ([]api.ResourcePattern, error) {
	var patterns []api.ResourcePattern
	for _, binding := range m.PathInfo.Bindings {
		expanded, err := expandBinding(binding)
		if err != nil {
			return nil, err
		}
		if len(expanded) > 0 {
			patterns = append(patterns, expanded)
		}
	}
	return patterns, nil
}

// TODO(https://github.com/googleapis/librarian/issues/4946): Simplify or abstract
// path segment models in the API representation rather than performing complex
// synthetic path segment expansion here.
// expandBinding flattens and expands the path segments in a method's HTTP PathBinding.
//
// Standard HTTP bindings often contain version numbers and complex nested resource paths.
// For example: "/v1/{name=projects/*/locations/*/operations/*}"
//
// This function cleans and expands it by:
//  1. Skipping API version prefixes (e.g., "v1", "v1beta1") to align with gcloud's base-level paths.
//  2. Expanding complex, multi-level nested variables (e.g., "{name=projects/*/locations/*}")
//     into alternating literal and wildcard variables (e.g., "projects", "{project}", "locations", "{location}").
func expandBinding(binding *api.PathBinding) ([]api.PathSegment, error) {
	var expandedSegments []api.PathSegment
	for _, seg := range binding.PathTemplate.Segments {
		if seg.Literal != nil {
			lit := *seg.Literal
			// Skip version prefix segments (e.g. "v1", "v2", "v1beta1")
			if len(lit) > 1 && lit[0] == 'v' && lit[1] >= '0' && lit[1] <= '9' {
				continue
			}
			// Literal segments represent static collection nouns or resource verbs (e.g., "projects", "locations").
			expandedSegments = append(expandedSegments, seg)
		} else if seg.Variable != nil && len(seg.Variable.Segments) > 1 {
			// Nested path variable detected (e.g., {name=projects/*/locations/*}).
			// Expand it to full alternating literal & wildcard segments.
			expanded, err := expandNestedVariable(seg.Variable)
			if err != nil {
				return nil, err
			}
			expandedSegments = append(expandedSegments, expanded...)
		} else {
			// Simple variable (e.g., {project} or {instance}) — keep it as is.
			expandedSegments = append(expandedSegments, seg)
		}
	}
	return expandedSegments, nil
}

// expandNestedVariable expands a complex nested path variable (e.g., {name=projects/*/locations/*})
// into alternating literal and wildcard path segments by replacing wildcards with singularized variable segments.
func expandNestedVariable(v *api.PathVariable) ([]api.PathSegment, error) {
	var expanded []api.PathSegment
	for i, part := range v.Segments {
		if part == "*" || part == "**" {
			if i == 0 {
				return nil, fmt.Errorf("invalid operations path segment: wildcard cannot be the first segment in a variable")
			}
			plural := v.Segments[i-1]
			singular := singular(plural)

			expanded = append(expanded, api.PathSegment{
				Variable: &api.PathVariable{
					FieldPath: []string{singular},
					Segments:  []string{"*"},
				},
			})
		} else {
			lit := part
			expanded = append(expanded, api.PathSegment{
				Literal: &lit,
			})
		}
	}
	return expanded, nil
}
