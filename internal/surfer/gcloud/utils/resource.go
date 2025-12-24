// Copyright 2025 Google LLC
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

package utils

import (
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/iancoleman/strcase"
)

// GetPluralFromSegments infers the plural name of a resource from its structured path segments.
// Per AIP-122, the plural is the literal segment before the final variable segment.
// Example: `.../instances/{instance}` -> "instances".
func GetPluralFromSegments(segments []api.PathSegment) string {
	if len(segments) >= 2 {
		lastSegment := segments[len(segments)-1]
		if lastSegment.Variable != nil {
			// The second to last segment should be the literal plural name
			secondLastSegment := segments[len(segments)-2]
			if secondLastSegment.Literal != nil {
				return *secondLastSegment.Literal
			}
		}
	}
	return ""
}

// GetSingularFromSegments infers the singular name of a resource from its structured path segments.
// The singular is the name of the final variable segment.
// Example: `.../instances/{instance}` -> "instance".
func GetSingularFromSegments(segments []api.PathSegment) string {
	if len(segments) > 0 {
		last := segments[len(segments)-1]
		if last.Variable != nil && len(last.Variable.FieldPath) > 0 {
			// Typically the variable name is the last component of the field path
			// e.g. for `name` binding it might be implied? No, httprule parser populates FieldPath.
			return last.Variable.FieldPath[len(last.Variable.FieldPath)-1]
		}
	}
	return ""
}

// GetCollectionPathFromSegments constructs the base gcloud collection path from a
// structured resource pattern, according to AIP-122 conventions.
// It joins the literal collection identifiers with dots.
// Example: `projects/{project}/locations/{location}/instances/{instance}` -> `projects.locations.instances`.
func GetCollectionPathFromSegments(segments []api.PathSegment) string {
	var collectionParts []string
	for i := 0; i < len(segments)-1; i++ {
		// A collection identifier is a literal segment followed by a variable segment.
		if segments[i].Literal != nil && segments[i+1].Variable != nil {
			collectionParts = append(collectionParts, *segments[i].Literal)
		}
	}
	return strings.Join(collectionParts, ".")
}

// IsPrimaryResource determines if a field represents the primary resource of a method.
func IsPrimaryResource(field *api.Field, method *api.Method) bool {
	if method.InputType == nil {
		return false
	}
	// For `Create` methods, the primary resource is identified by a field named
	// in the format "{resource}_id" (e.g., "instance_id").
	if strings.HasPrefix(method.Name, "Create") {
		resourceName, err := GetResourceName(method)
		if err == nil && field.Name == strcase.ToSnake(resourceName)+"_id" {
			return true
		}
	}
	// For `Get`, `Delete`, and `Update` methods, the primary resource is identified
	// by a field named "name", which holds the full resource name.
	if (strings.HasPrefix(method.Name, "Get") || strings.HasPrefix(method.Name, "Delete") || strings.HasPrefix(method.Name, "Update")) && field.Name == "name" {
		return true
	}
	return false
}

// GetResourceName extracts the name of the resource from a method's input message.
// For example, for `CreateInstanceRequest`, it would return "Instance".
func GetResourceName(method *api.Method) (string, error) {
	if method.InputType == nil {
		return "", fmt.Errorf("method input type is nil")
	}
	for _, f := range method.InputType.Fields {
		if msg := f.MessageType; msg != nil && msg.Resource != nil {
			return msg.Name, nil
		}
	}
	return "", fmt.Errorf("resource message not found in input type")
}

// GetResourceForMethod finds the `api.Resource` definition associated with a method.
// This is a crucial function for linking a method to the resource it operates on.
func GetResourceForMethod(method *api.Method, model *api.API) *api.Resource {
	if method.InputType == nil {
		return nil
	}

	// Strategy 1: For `Create` and `Update`, the request message usually contains
	// a field that *is* the resource message. This message is annotated with `(google.api.resource)`.
	for _, f := range method.InputType.Fields {
		if msg := f.MessageType; msg != nil && msg.Resource != nil {
			return msg.Resource
		}
	}

	// Strategy 2: For `Get`, `Delete`, and `List`, the request message has a `name`
	// or `parent` field with a `(google.api.resource_reference)`.
	var resourceType string
	for _, field := range method.InputType.Fields {
		if (field.Name == "name" || field.Name == "parent") && field.ResourceReference != nil {
			// For collection methods (like List), the reference is to the parent,
			// and the resource we care about is the `child_type`.
			if field.ResourceReference.ChildType != "" {
				resourceType = field.ResourceReference.ChildType
			} else {
				resourceType = field.ResourceReference.Type
			}
			break
		}
	}

	if resourceType == "" {
		return nil
	}

	// Use the API model's indexed maps for an efficient lookup.
	for _, r := range model.ResourceDefinitions {
		if r.Type == resourceType {
			return r
		}
	}

	// Also check resources defined on messages directly.
	for _, m := range model.Messages {
		if m.Resource != nil && m.Resource.Type == resourceType {
			return m.Resource
		}
	}

	return nil
}

// GetPluralResourceNameForMethod determines the plural name of a resource. It follows a clear
// hierarchy of truth: first, the explicit `plural` field in the resource
// definition, and second, inference from the resource pattern.
func GetPluralResourceNameForMethod(method *api.Method, model *api.API) string {
	resource := GetResourceForMethod(method, model)
	if resource != nil {
		// The `plural` field in the `(google.api.resource)` annotation is the
		// most authoritative source.
		if resource.Plural != "" {
			return resource.Plural
		}
		// If the `plural` field is not present, we fall back to inferring the
		// plural name from the resource's pattern string, as per AIP-122.
		if len(resource.Patterns) > 0 {
			return GetPluralFromSegments(resource.Patterns[0])
		}
	}
	return ""
}
