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

// IdentifyTargetResources populates the TargetResource field in PathBinding
// for all methods in the API.
//
// This is done in two passes:
//  1. Explicit Identification: Matches google.api.resource_reference annotations
//     with fields present in the PathTemplate.
//  2. Heuristic Identification: For allow-listed services, uses path segment
//     patterns to identify resources when annotations are missing.
func IdentifyTargetResources(model *API) {
	for _, service := range model.Services {
		for _, method := range service.Methods {
			if method.PathInfo == nil {
				continue
			}
			for _, binding := range method.PathInfo.Bindings {
				identifyTargetResourceForBinding(method, binding)
			}
		}
	}
}

// identifyTargetResourceForBinding processes a single path binding to identify its target resource.
func identifyTargetResourceForBinding(method *Method, binding *PathBinding) {
	if binding.PathTemplate == nil {
		return
	}

	// Priority 1: Explicit Identification
	// Matches google.api.resource_reference annotations.
	if target := identifyExplicitTarget(method, binding); target != nil {
		binding.TargetResource = target
		return
	}

	// Priority 2: Heuristic Identification
	// Uses path segment patterns to guess the resource.
	// TODO(#4100): Implement IdentifyTargetResources for allow-listed services using heuristic path segment patterns.
}

func identifyExplicitTarget(method *Method, binding *PathBinding) *TargetResource {
	var fieldPaths [][]string
	if method.InputType == nil {
		return nil
	}

	// Collect variable segments from the path template
	for _, segment := range binding.PathTemplate.Segments {
		if segment.Variable == nil {
			continue
		}

		fieldPath := segment.Variable.FieldPath
		field := findField(method.InputType, fieldPath)

		if field == nil {
			return nil
		}

		if !field.IsResourceReference() {
			return nil
		}

		fieldPaths = append(fieldPaths, fieldPath)
	}

	if len(fieldPaths) == 0 {
		return nil
	}

	return &TargetResource{
		FieldPaths:   fieldPaths,
		PathTemplate: binding.PathTemplate,
	}
}

// findField traverses the (nested) message structure to find a field by its field path.
func findField(msg *Message, path []string) *Field {
	if len(path) == 0 {
		return nil
	}

	current := msg
	var field *Field

	findInFields := func(fields []*Field, name string) *Field {
		for _, f := range fields {
			if f.Name == name {
				return f
			}
		}
		return nil
	}

	for i, name := range path {
		field = findInFields(current.Fields, name)
		if field == nil {
			return nil
		}

		if i < len(path)-1 {
			if field.MessageType == nil {
				return nil
			}
			current = field.MessageType
		}
	}

	return field
}
