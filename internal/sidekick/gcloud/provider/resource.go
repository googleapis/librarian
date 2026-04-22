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

// GetParentFromSegments extracts the pattern segments for the parent resource.
// It assumes the standard resource pattern structure where the last two segments
// are the literal plural noun and the variable singular noun of the child resource.
// Example: `projects/.../locations/{location}/instances/{instance}` -> `projects/.../locations/{location}`.
func GetParentFromSegments(segments []api.PathSegment) []api.PathSegment {
	if len(segments) < 2 {
		return nil
	}
	// We verify that the last segment is a variable and the second to last is a literal,
	// consistent with standard AIP-122 patterns.
	if segments[len(segments)-1].Variable != nil && segments[len(segments)-2].Literal != nil {
		return segments[:len(segments)-2]
	}
	return nil
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

// IsPrimaryResourceField determines if a field represents the primary resource of a method.
func IsPrimaryResourceField(field *api.Field, method *api.Method) bool {
	if method.InputType == nil {
		return false
	}

	// If it's a collection method, field name is parent, return true.
	if IsCollectionMethod(method) && field.Name == "parent" {
		return true
	}

	// If it is a resource method, field name "name", return true.
	if IsResourceMethod(method) && field.Name == "name" {
		return true
	}

	return false
}

// IsResourceIdField determines if a field represents the resource ID in a create method.
func IsResourceIdField(field *api.Field, method *api.Method, model *api.API) bool {
	if !IsCreate(method) {
		return false
	}
	resource := GetResourceForMethod(method, model)
	if resource == nil {
		return false
	}
	resName := GetResourceNameFromType(resource.Type)
	return resName != "" && field.Name == strcase.ToSnake(resName)+"_id"
}

// GetResourceNameFromType returns the singular form of the resource noun from a resource type string.
// According to AIP-123, the format of a resource type is {Service Name}/{Type}, where
// {Type} is the singular form of the resource noun.
func GetResourceNameFromType(typeStr string) string {
	parts := strings.Split(typeStr, "/")
	return parts[len(parts)-1]
}

// FindNameField returns the field named "name" from the resource's message definition, if present.
func FindNameField(resource *api.Resource) *api.Field {
	if resource == nil || resource.Self == nil {
		return nil
	}
	for _, f := range resource.Self.Fields {
		if f.Name == "name" {
			return f
		}
	}
	return nil
}

// GetResourceForMethod finds the `api.Resource` definition associated with a method.
// This is a crucial function for linking a method to the resource it operates on.
func GetResourceForMethod(method *api.Method, model *api.API) *api.Resource {
	if method == nil || method.InputType == nil || model == nil {
		return nil
	}

	// Strategy 1: For Create (AIP-133) and Update (AIP-134), the request message
	// usually contains a field that *is* the resource message.
	for _, f := range method.InputType.Fields {
		if f.MessageType != nil && f.MessageType.Resource != nil {
			return f.MessageType.Resource
		}
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

// isSingletonResource determines whether a resource is a singleton.
// It checks if any of the resource's canonical patterns end in a literal segment
// or contain two adjacent literal segments.
func isSingletonResource(resource *api.Resource) bool {
	if resource == nil {
		return false
	}

	for _, pattern := range resource.Patterns {
		if len(pattern) == 0 {
			continue
		}

		// A pattern ending in a literal is a singleton (e.g., projects/{project}/singletonConfig).
		if pattern[len(pattern)-1].Literal != nil {
			return true
		}

		// Adjacent literals anywhere in the pattern signify a singleton (e.g., .../literal1/literal2/...).
		for i := 0; i < len(pattern)-1; i++ {
			if pattern[i].Literal != nil && pattern[i+1].Literal != nil {
				return true
			}
		}
	}

	return false
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

// ExtractPathFromSegments extracts the dot-separated collection path from path segments.
// It handles:
// 1. Skipping API version prefixes (e.g., v1).
// 2. Extracting internal structure from complex variables (e.g., {name=projects/*/locations/*}).
// 3. Including all literal segments (e.g., instances in .../instances).
func ExtractPathFromSegments(segments []api.PathSegment) string {
	var parts []string
	for i, seg := range segments {
		if seg.Literal != nil {
			val := *seg.Literal
			// Heuristic: Skip API version at the start.
			if i == 0 && len(val) >= 2 && val[0] == 'v' && val[1] >= '0' && val[1] <= '9' {
				continue
			}
			parts = append(parts, val)
		} else if seg.Variable != nil && len(seg.Variable.Segments) > 1 {
			internal := ExtractCollectionFromStrings(seg.Variable.Segments)
			if internal != "" {
				parts = append(parts, internal)
			}
		}
	}
	return strings.Join(parts, ".")
}

// ExtractCollectionFromStrings constructs a collection path from a list of string segments
// (literals and wildcards), following AIP-122 conventions (literal followed by variable/wildcard).
func ExtractCollectionFromStrings(parts []string) string {
	var sb strings.Builder
	var prev string

	for _, curr := range parts {
		switch curr {
		case "*", "**":
			if prev != "" {
				if sb.Len() > 0 {
					sb.WriteByte('.')
				}
				sb.WriteString(prev)
				prev = ""
			}
		default:
			prev = curr
		}
	}
	return sb.String()
}

// GetLiteralSegments extracts the literal path segments from a list of PathSegments,
// filtering out version suffixes like `V1`, `V2`, etc. and wildcard segments `*`, `**`.
func GetLiteralSegments(raw []api.PathSegment) []string {
	var literals []string
	for _, seg := range raw {
		if seg.Literal != nil {
			literals = append(literals, *seg.Literal)
		} else if seg.Variable != nil {
			for _, vSeg := range seg.Variable.Segments {
				if vSeg != "*" && vSeg != "**" {
					literals = append(literals, vSeg)
				}
			}
		}
	}

	var filtered []string
	for _, lit := range literals {
		if len(lit) > 1 && lit[0] == 'v' && lit[1] >= '0' && lit[1] <= '9' {
			continue
		}
		filtered = append(filtered, lit)
	}
	return filtered
}

// GetSingularResourceNameForPrefix looks up the singular resource name for a given list of URL prefix literals.
func GetSingularResourceNameForPrefix(model *api.API, prefix []string) string {
	target := strings.Join(prefix, ".")
	for _, res := range model.ResourceDefinitions {
		if len(res.Patterns) == 0 {
			continue
		}
		segments := GetLiteralSegments(res.Patterns[0])
		if strings.Join(segments, ".") == target {
			if res.Singular != "" {
				return res.Singular
			}
			return GetSingularFromSegments(res.Patterns[0])
		}
	}
	return ""
}
