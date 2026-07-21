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

// Resource is a fundamental building block of an API, representing an
// individually-named entity (a "noun").
//
// Resources are typically organized into a hierarchy, where each node is either a simple resource or a
// collection of resources.
// This definition is based on AIP-121 (https://google.aip.dev/121).
type Resource struct {
	// Type identifies the kind of resource (e.g., "cloudresourcemanager.googleapis.com/Project").
	// This string is globally unique and identifies the type of resource across Google Cloud.
	Type string
	// Pattern is a list of resource patterns, where each pattern is a sequence of path segments.
	// This defines the structure of the resource's unique identifier.
	Patterns []ResourcePattern
	// Plural is the plural form of the resource name.
	// For example, for a "Book" resource, Plural would be "books".
	Plural string
	// Singular is the singular form of the resource name.
	// For example, for a "Book" resource, Singular would be "book".
	Singular string
	// Self points to the Message that defines this resource.
	// This creates a back-reference for navigating the API model,
	// allowing a Resource definition to access its originating Message structure.
	Self *Message
	// Language specific annotations.
	Codec any
}

// ResourcePattern is a sequence of path segments that defines the structure of a resource's unique identifier.
//
// Given a resource name pattern like `projects/{project}/locations/{region}/secret/{secret}` this
// will be:
//
//	[]PathSegment{
//	  {Literal: "projects"},
//	  {Variable: "project"},
//	  {Literal: "locations"},
//	  {Variable: "region"},
//	  {Literal: "secrets"},
//	  {Variable: "secret"},
//	}
type ResourcePattern []PathSegment

// ResourceNameSegment is a segment of a resource name pattern.
//
// This should be a union type, either `Literal` or `Variable` are set, but not both.
type ResourceNameSegment struct {
	// Literal is the literal part of the segment.
	//
	// When formatting a resource name this should be used, well, literally.
	Literal string

	// Variable is the name of the variable part of the segment.
	//
	// When formatting a resource name, this names a variable used to populate the resource name.
	Variable string
}

// ResourceNamePattern describes the structure of a resource name.
type ResourceNamePattern struct {
	Segments []ResourceNameSegment
}

// ResourceReference describes a field's relationship to another resource type.
// It acts as a foreign key, indicating that the field's value identifies an instance of another resource.
// This relationship is established via the `google.api.resource_reference` annotation in Protobuf.
type ResourceReference struct {
	// Type is the unique identifier of the referenced resource's kind (e.g., "library.googleapis.com/Shelf").
	// This string matches the `Type` field in the corresponding `Resource` definition.
	Type string
	// ChildType is the unique identifier of a *child* resource's kind.
	// This is used when a field references a parent resource (e.g., "Shelf"), but the context
	// implies interaction with a specific child type (e.g., "Book" within that shelf).
	ChildType string
	// Language specific annotations.
	Codec any
}

// TargetResource contains the results of the resource name identification.
// It provides the sequences of fields used by language-specific generators to inject tracing attributes.
type TargetResource struct {
	// FieldPaths is a list of field name sequences that, when joined, form a resource name.
	// For example, [["project"], ["zone"], ["instance"]] identifies a multi-part resource.
	FieldPaths [][]string

	// Template is the canonical HTTP path template for the resource, derived from the PathBinding's PathTemplate by removing the API version prefix.
	// For example, if the PathTemplate is "//compute.googleapis.com/projects/{project}/zones/{zone}", the Template will be a []PathSegment containing:
	// - a Literal segment for "//compute.googleapis.com"
	// - a Literal segment for "projects"
	// - a Variable segment with FieldPath ["project"]
	// - a Literal segment for "zones"
	// - a Variable segment with FieldPath ["zone"]
	Template []PathSegment
}
