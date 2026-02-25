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
	"fmt"
	"slices"
	"strings"
)

// IdentifyTargetResources populates the TargetResource field in PathBinding
// for all methods in the API.
//
// This is done in two passes:
//  1. Explicit Identification: Matches google.api.resource_reference annotations
//     with fields present in the PathTemplate.
//  2. Heuristic Identification: For allow-listed services, uses path segment
//     patterns to identify resources when annotations are missing.
func IdentifyTargetResources(model *API) error {
	// Build the set of known resource names for the heuristic.
	vocabulary := BuildHeuristicVocabulary(model)

	for _, service := range model.Services {
		for _, method := range service.Methods {
			if method.PathInfo == nil {
				continue
			}
			for _, binding := range method.PathInfo.Bindings {
				if err := identifyTargetResourceForBinding(method, binding, vocabulary); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// identifyTargetResourceForBinding processes a single path binding to identify its target resource.
func identifyTargetResourceForBinding(method *Method, binding *PathBinding, vocabulary map[string]bool) error {
	if binding.PathTemplate == nil {
		return nil
	}

	// Priority 1: Explicit Identification
	// Matches google.api.resource_reference annotations.
	target, err := identifyExplicitTarget(method, binding)
	if err != nil {
		return err
	}
	if target != nil {
		binding.TargetResource = target
		return nil
	}

	// Priority 2: Heuristic Identification
	// Uses path segment patterns to guess the resource.
	target, err = identifyHeuristicTarget(method, binding, vocabulary)
	if err != nil {
		return err
	}
	if target != nil {
		binding.TargetResource = target
		return nil
	}
	return nil
}

func identifyHeuristicTarget(method *Method, binding *PathBinding, vocabulary map[string]bool) (*TargetResource, error) {
	if !IsHeuristicEligible(method.Service.ID) {
		return nil, nil
	}

	var fieldPaths [][]string
	var lastIndex int
	segments := binding.PathTemplate.Segments

	// Iterate starting from the second segment, looking for (literal, variable) pairs.
	for i := 1; i < len(segments); i++ {
		curr := segments[i]
		prev := segments[i-1]

		if curr.Variable == nil || prev.Literal == nil {
			continue
		}

		// Check if the preceding literal is a valid collection identifier.
		if !isCollectionIdentifier(*prev.Literal, vocabulary) {
			// Once a path segment breaks the chain of valid (collection, ID) pairs,
			// the resource identifier stops.
			break
		}

		fieldPath := curr.Variable.FieldPath
		// Verify the field exists in the input message.
		_, err := findField(method.InputType, fieldPath)
		if err != nil {
			return nil, err
		}
		fieldPaths = append(fieldPaths, fieldPath)
		lastIndex = i + 1
	}

	if len(fieldPaths) == 0 {
		return nil, nil
	}

	template, err := constructTemplate(method, segments[:lastIndex])
	if err != nil {
		return nil, err
	}

	return &TargetResource{
		FieldPaths: fieldPaths,
		Template:   template,
	}, nil
}

func identifyExplicitTarget(method *Method, binding *PathBinding) (*TargetResource, error) {
	var fieldPaths [][]string
	if method.InputType == nil {
		return nil, fmt.Errorf("consistency error: method %q has no InputType", method.Name)
	}

	// Collect field paths corresponding to variable segments in the path template
	for _, segment := range binding.PathTemplate.Segments {
		if segment.Variable == nil {
			continue
		}

		fieldPath := segment.Variable.FieldPath
		field, err := findField(method.InputType, fieldPath)
		if err != nil {
			return nil, err
		}
		if field == nil {
			return nil, fmt.Errorf("consistency error: field %v not found in message %q", fieldPath, method.InputType.Name)
		}
		if !field.IsResourceReference() {
			return nil, nil
		}
		fieldPaths = append(fieldPaths, fieldPath)
	}

	if len(fieldPaths) == 0 {
		return nil, nil
	}
	template, err := constructTemplate(method, binding.PathTemplate.Segments)
	if err != nil {
		return nil, err
	}
	return &TargetResource{
		FieldPaths: fieldPaths,
		Template:   template,
	}, nil
}

// findField traverses the (nested) message structure to find a field by its field path.
func findField(msg *Message, path []string) (*Field, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("consistency error: empty field path in msg: %s", msg.ID)
	}

	current := msg
	var field *Field

	for i, name := range path {
		idx := slices.IndexFunc(current.Fields, func(f *Field) bool {
			return f.Name == name
		})
		if idx == -1 {
			return nil, fmt.Errorf("consistency error: field %s not found in message %q", name, current.Name)
		}
		field = current.Fields[idx]

		if i < len(path)-1 {
			if field.MessageType == nil {
				return nil, fmt.Errorf("consistency error: field %s in message %s has no MessageType", field.Name, current.Name)
			}
			current = field.MessageType
		}
	}

	return field, nil
}

// constructTemplate reconstructs the canonical resource name template from path segments.
func constructTemplate(method *Method, segments []PathSegment) (string, error) {
	host, err := getServiceHost(method)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("//")
	sb.WriteString(host)

	for _, seg := range segments {
		sb.WriteString("/")
		if seg.Literal != nil {
			sb.WriteString(*seg.Literal)
		} else if seg.Variable != nil {
			// Use simple field path form {field}, ignoring internal patterns, handling both exploded paths and full name variables correctly.
			fmt.Fprintf(&sb, "{%s}", strings.Join(seg.Variable.FieldPath, "."))
		}
	}
	return sb.String(), nil
}

func getServiceHost(method *Method) (string, error) {
	if method.Service != nil && method.Service.DefaultHost != "" {
		return method.Service.DefaultHost, nil
	}
	if method.Model != nil && method.Model.Name != "" {
		return method.Model.Name + ".googleapis.com", nil
	}
	return "", fmt.Errorf("consistency error: no service host found for method %q", method.Name)
}
