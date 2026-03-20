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

// getResourceNameFromType extracts the singular resource name from a resource type string.
// According to AIP-123, the format of a resource type is {Service Name}/{Type}, where
// {Type} is the singular form of the resource noun.
func getResourceNameFromType(typeStr string) string {
	parts := strings.Split(typeStr, "/")
	return parts[len(parts)-1]
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
			internal := extractCollectionFromStrings(seg.Variable.Segments)
			if internal != "" {
				parts = append(parts, internal)
			}
		}
	}
	return strings.Join(parts, ".")
}

// extractCollectionFromStrings constructs a collection path from a list of string segments
// (literals and wildcards), following AIP-122 conventions (literal followed by variable/wildcard).
func extractCollectionFromStrings(parts []string) string {
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
