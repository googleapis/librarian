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
)

// IdentifyTargetResources populates the TargetResource field in PathBinding
// for all methods in the API.
//
// This is done in two passes:
//  1. Explicit Identification: Matches google.api.resource_reference annotations
//     with fields present in the PathTemplate.
//  2. Heuristic Identification: If enableHeuristics is true and the service is
//     allow-listed, uses path segment patterns to identify resources when annotations are missing.
func IdentifyTargetResources(model *API, enableHeuristics bool) error {
	// Build the set of known resource names for the heuristic.
	vocabulary := BuildHeuristicVocabulary(model)

	for _, service := range model.Services {
		for _, method := range service.Methods {
			if method.pathInfo == nil {
				continue
			}
			for _, binding := range method.pathInfo.Bindings() {
				if err := identifyTargetResourceForBinding(method, binding, vocabulary, enableHeuristics); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// identifyTargetResourceForBinding processes a single path binding to identify its target resource.
func identifyTargetResourceForBinding(method *Method, binding *PathBinding, vocabulary map[string]bool, enableHeuristics bool) error {
	if binding.pathTemplate == nil {
		return nil
	}

	// Uses path segment patterns to guess the resource.
	if enableHeuristics {
		target, err := identifyHeuristicTarget(method, binding, vocabulary)
		if err != nil {
			return err
		}
		if target != nil {
			binding.targetResource = target
			return nil
		}
	}

	target, err := identifyExplicitTarget(method, binding)
	if err != nil {
		return err
	}
	if target != nil {
		binding.targetResource = target
		return nil
	}
	return nil
}

func identifyHeuristicTarget(method *Method, binding *PathBinding, vocabulary map[string]bool) (*TargetResource, error) {

	tmpl := binding.pathTemplate
	if tmpl == nil {
		return nil, nil
	}

	// Iterate backwards over segments
	segments := tmpl.Segments()
	for i, seg := range slices.Backward(segments) {
		if seg.Variable() == nil || i == 0 || segments[i-1].Literal() == "" {
			continue
		}

		token := segments[i-1].Literal()
		if !vocabulary[token] && !isVersionString(token) {
			continue // continue scanning backward if not in vocabulary
		}

		// The default firstIndex is the current variable segment. If the preceding
		// literal is a known collection, we'll try to walk backwards to find the
		// beginning of a resource pattern chain.
		firstIndex := i
		if vocabulary[token] {
			// Walk backwards to find the start of the resource pattern chain
			firstIndex = i - 1
			for firstIndex > 0 {
				prevSeg := segments[firstIndex-1]

				// Case 1: The preceding segment is a standalone literal (e.g., "global")
				if prevSeg.Literal() != "" {
					if isVersionString(prevSeg.Literal()) {
						break // Stop at version strings (e.g., "v1")
					}
					firstIndex--
					continue
				}

				// Case 2: The preceding segment is a variable. It must be paired with a literal.
				if prevSeg.Variable() != nil {
					if firstIndex < 2 || segments[firstIndex-2].Literal() == "" {
						break // Variable near the beginning, cannot form a pair
					}

					prevLiteralVal := segments[firstIndex-2].Literal()

					// If the literal is a known collection, include the pair and continue
					if vocabulary[prevLiteralVal] {
						firstIndex -= 2
						continue
					}

					// If the literal is a version string, include the variable but stop here
					if isVersionString(prevLiteralVal) {
						firstIndex--
					}

					// The chain is broken, so stop scanning.
					break
				}
			}
		}

		if method.inputType == nil {
			return nil, fmt.Errorf("consistency error: method %q has no InputType", method.name)
		}

		// Verify the chain connects properly to the root of the path.
		// If earlier variables exist, this chain is just a trailing partial match and should be ignored.
		disconnected := false
		for k := 0; k < firstIndex; k++ {
			if segments[k].Variable() != nil {
				disconnected = true
				break
			}
		}
		if disconnected {
			continue
		}

		var fieldPaths [][]string
		targetSegments := segments[firstIndex : i+1]
		for _, s := range targetSegments {
			if s.Variable() != nil {
				fieldPath := s.Variable().FieldPath()
				_, err := findField(method.inputType, fieldPath)
				if err != nil {
					return nil, err
				}
				fieldPaths = append(fieldPaths, fieldPath)
			}
		}
		template, err := constructTemplate(method, targetSegments)
		if err != nil {
			return nil, err
		}
		return NewTargetResource(fieldPaths, template), nil
	}
	return nil, nil
}

func identifyExplicitTarget(method *Method, binding *PathBinding) (*TargetResource, error) {
	var fieldPaths [][]string
	if method.inputType == nil {
		return nil, fmt.Errorf("consistency error: method %q has no InputType", method.name)
	}

	var lastVarIndex = -1
	segments := binding.pathTemplate.Segments()
	// Collect field paths corresponding to variable segments in the path template
	for i, segment := range segments {
		if segment.Variable() == nil {
			continue
		}

		fieldPath := segment.Variable().FieldPath()
		field, err := findField(method.inputType, fieldPath)
		if err != nil {
			return nil, err
		}
		if field == nil {
			return nil, fmt.Errorf("consistency error: field %v not found in message %q", fieldPath, method.inputType.name)
		}
		if !field.IsResourceReference() {
			return nil, nil
		}
		fieldPaths = append(fieldPaths, fieldPath)
		lastVarIndex = i
	}

	if len(fieldPaths) == 0 {
		return nil, nil
	}
	template, err := constructTemplate(method, segments[:lastVarIndex+1])
	if err != nil {
		return nil, err
	}
	return NewTargetResource(fieldPaths, template), nil
}

// findField traverses the (nested) message structure to find a field by its field path.
func findField(msg *Message, path []string) (*Field, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("consistency error: empty field path in msg: %s", msg.id)
	}

	current := msg
	var field *Field

	for i, name := range path {
		idx := slices.IndexFunc(current.fields, func(f *Field) bool {
			return f.name == name
		})
		if idx == -1 {
			return nil, fmt.Errorf("consistency error: field %s not found in message %q", name, current.name)
		}
		field = current.fields[idx]

		if i < len(path)-1 {
			if field.messageType == nil {
				return nil, fmt.Errorf("consistency error: field %s in message %s has no MessageType", field.name, current.name)
			}
			current = field.messageType
		}
	}

	return field, nil
}

// constructTemplate reconstructs the canonical resource name template from path segments.
func constructTemplate(method *Method, segments []PathSegment) ([]PathSegment, error) {
	host, err := getServiceHost(method)
	if err != nil {
		return nil, err
	}

	var result []PathSegment
	h := "//" + host
	result = append(result, *NewPathSegmentBuilder().WithLiteral(h).Build())

	for _, seg := range segments {
		if seg.Literal() != "" {
			l := seg.Literal()
			if isVersionString(l) {
				continue
			}
			result = append(result, *NewPathSegmentBuilder().WithLiteral(l).Build())
		} else if seg.Variable() != nil {
			result = append(result, *NewPathSegmentBuilder().
				WithVariable(NewPathVariableBuilder(seg.Variable().FieldPath()...).Build()).
				Build())
		}
	}
	return result, nil
}

func getServiceHost(method *Method) (string, error) {
	if method.service != nil && method.service.DefaultHost != "" {
		return method.service.DefaultHost, nil
	}
	if method.model != nil && method.model.Name != "" {
		return method.model.Name + ".googleapis.com", nil
	}
	return "", fmt.Errorf("consistency error: no service host found for method %q", method.name)
}

// isVersionString checks if a string appears to be an API version segment,
// such as "v1" or "v1beta1".
func isVersionString(s string) bool {
	return len(s) >= 2 && s[0] == 'v' && s[1] >= '0' && s[1] <= '9'
}
