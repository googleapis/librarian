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

const (
	// GetOperation is the RPC method name for getting an operation's status.
	GetOperation = "GetOperation"
	// CancelOperation is the RPC method name for cancelling an active operation.
	CancelOperation = "CancelOperation"
	// DeleteOperation is the RPC method name for deleting a completed operation.
	DeleteOperation = "DeleteOperation"
	// ListOperations is the RPC method name for listing operations.
	ListOperations = "ListOperations"
)

// IsOperationsServiceMethod determines if the method belongs to the long-running operations service.
func IsOperationsServiceMethod(m *api.Method) bool {
	return m.SourceServiceID == ".google.longrunning.Operations"
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

// inferOperationResource creates a synthetic resource for operations based on the method's path.
func inferOperationResource(m *api.Method) (*api.Resource, error) {
	if m.PathInfo == nil || len(m.PathInfo.Bindings) == 0 {
		return nil, nil
	}

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

	if len(patterns) == 0 {
		return nil, nil
	}

	return &api.Resource{
		Type:     operationResourceType,
		Singular: "operation",
		Plural:   "operations",
		Patterns: patterns,
	}, nil
}

func expandBinding(binding *api.PathBinding) ([]api.PathSegment, error) {
	if binding == nil || binding.PathTemplate == nil {
		return nil, nil
	}

	var expandedSegments []api.PathSegment
	for _, seg := range binding.PathTemplate.Segments {
		if seg.Literal != nil {
			lit := *seg.Literal
			// Skip version segments like "v1", "v2", etc.
			if len(lit) > 1 && lit[0] == 'v' && lit[1] >= '0' && lit[1] <= '9' {
				continue
			}
			expandedSegments = append(expandedSegments, seg)
		} else if seg.Variable != nil && len(seg.Variable.Segments) > 1 {
			expanded, err := expandNestedVariable(seg.Variable)
			if err != nil {
				return nil, err
			}
			expandedSegments = append(expandedSegments, expanded...)
		} else {
			expandedSegments = append(expandedSegments, seg)
		}
	}
	return expandedSegments, nil
}

// expandNestedVariable expands a complex nested path variable (e.g., {name=projects/*/locations/*})
// into alternating literal and wildcard path segments.
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
