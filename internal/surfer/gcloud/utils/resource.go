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
	if len(segments) < 2 {
		return ""
	}
	lastSegment := segments[len(segments)-1]
	if lastSegment.Variable == nil {
		return ""
	}
	// The second to last segment should be the literal plural name
	secondLastSegment := segments[len(segments)-2]
	if secondLastSegment.Literal == nil {
		return ""
	}
	return *secondLastSegment.Literal
}

// GetSingularFromSegments infers the singular name of a resource from its structured path segments.
// According to AIP-123, the last segment of a resource pattern MUST be a variable representing
// the resource ID, and its name MUST be the singular form of the resource noun.
// Example: `.../instances/{instance}` -> "instance".
func GetSingularFromSegments(segments []api.PathSegment) string {
	if len(segments) == 0 {
		return ""
	}
	last := segments[len(segments)-1]
	if last.Variable == nil || len(last.Variable.FieldPath) == 0 {
		return ""
	}
	// Per AIP-123, the last variable name is the singular form of the resource noun.
	return last.Variable.FieldPath[len(last.Variable.FieldPath)-1]
}

// GetCollectionPathFromSegments constructs the base gcloud collection path from a
// structured resource pattern, according to AIP-122 conventions.
// It joins the literal collection identifiers with dots.
// Example: `projects/{project}/locations/{location}/instances/{instance}` -> `projects.locations.instances`.
func GetCollectionPathFromSegments(segments []api.PathSegment) string {
	var collectionParts []string
	for i := 0; i < len(segments)-1; i++ {
		// A collection identifier is a literal segment followed by a variable segment.
		if segments[i].Literal == nil || segments[i+1].Variable == nil {
			continue
		}
		collectionParts = append(collectionParts, *segments[i].Literal)
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
	if IsCreate(method) {
		resource, err := getResourceFromMethod(method)
		if err == nil {
			name := getResourceNameFromType(resource.Type)
			// TODO(https://github.com/googleapis/librarian/issues/3361): Verify that this case transformation
			// is consistent with gcloud conventions and doesn't introduce traceability issues.
			if name != "" && field.Name == strcase.ToSnake(name)+"_id" {
				return true
			}
		}
	}
	// For `Get`, `Delete`, and `Update` methods, the primary resource is identified
	// by a field named "name", which holds the full resource name.
	if (IsGet(method) || IsDelete(method) || IsUpdate(method)) && field.Name == "name" {
		return true
	}
	return false
}

// getResourceNameFromType extracts the singular resource name from a resource type string.
// According to AIP-123, the format of a resource type is {Service Name}/{Type}, where
// {Type} is the singular form of the resource noun.
func getResourceNameFromType(typeStr string) string {
	parts := strings.Split(typeStr, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// getResourceFromMethod extracts the resource definition from a method's input message if it exists.
func getResourceFromMethod(method *api.Method) (*api.Resource, error) {
	if method.InputType == nil {
		return nil, fmt.Errorf("method %q does not have an input type", method.Name)
	}
	for _, f := range method.InputType.Fields {
		if f.MessageType != nil && f.MessageType.Resource != nil {
			return f.MessageType.Resource, nil
		}
	}
	return nil, fmt.Errorf("resource message not found in input type for method %q", method.Name)
}

// GetResourceForMethod finds the `api.Resource` definition associated with a method.
// This is a crucial function for linking a method to the resource it operates on.
func GetResourceForMethod(method *api.Method, model *api.API) *api.Resource {
	if method.InputType == nil {
		return nil
	}

	// Strategy 1: For Create (AIP-133) and Update (AIP-134), the request message
	// usually contains a field that *is* the resource message.
	if resource, err := getResourceFromMethod(method); err == nil {
		return resource
	}

	// Strategy 2: For Get (AIP-131), Delete (AIP-135), and List (AIP-132), the
	// request message has a `name` or `parent` field with a `(google.api.resource_reference)`.
	var resourceType string
	for _, field := range method.InputType.Fields {
		if (field.Name == "name" || field.Name == "parent") && field.ResourceReference != nil {
			// AIP-132 (List): The "parent" field refers to the parent collection, but the
			// annotation's `child_type` field (if present) points to the resource being listed.
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

	// TODO(https://github.com/googleapis/librarian/issues/3363): Avoid this lookup by linking the ResourceReference
	// to the Resource definition during model creation or post-processing.

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

// GetSingularResourceNameForMethod determines the singular name of a resource. It follows a clear
// hierarchy of truth: first, the explicit `singular` field in the resource
// definition, and second, inference from the resource pattern.
func GetSingularResourceNameForMethod(method *api.Method, model *api.API) string {
	resource := GetResourceForMethod(method, model)
	if resource != nil {
		if resource.Singular != "" {
			return resource.Singular
		}
		if len(resource.Patterns) > 0 {
			return GetSingularFromSegments(resource.Patterns[0])
		}
	}
	return ""
}
