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
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/iancoleman/strcase"
)

// MethodType enumerates the structural intent of an API method.
type MethodType int

const (
	// MethodTypeUnknown implies an unparseable method pattern.
	MethodTypeUnknown MethodType = iota
	// MethodTypeGet represents standard GET/READ requests.
	MethodTypeGet
	// MethodTypeList represents standard LIST metadata requests.
	MethodTypeList
	// MethodTypeCreate represents standard POST/CREATE requests.
	MethodTypeCreate
	// MethodTypeUpdate represents standard PATCH/UPDATE requests.
	MethodTypeUpdate
	// MethodTypeDelete represents standard DELETE requests.
	MethodTypeDelete
	// MethodTypeCustom represents custom verbs acting on resources.
	MethodTypeCustom
)

// MethodAdapter wraps sidekick's api.Method to encapsulate its evaluation logic.
type MethodAdapter struct {
	Method *api.Method
}

// Type returns the standard AIP method type of the underlying method.
func (a *MethodAdapter) Type() MethodType {
	verb := a.getHTTPVerb()
	name := a.Method.Name

	if a.Method.IsAIPStandardGet || (strings.HasPrefix(name, "Get") && (verb == "" || verb == "GET")) {
		return MethodTypeGet
	}
	if strings.HasPrefix(name, "List") && (verb == "" || verb == "GET") {
		return MethodTypeList
	}
	if strings.HasPrefix(name, "Create") && (verb == "" || verb == "POST") {
		return MethodTypeCreate
	}
	if strings.HasPrefix(name, "Update") && (verb == "" || verb == "PATCH" || verb == "PUT") {
		return MethodTypeUpdate
	}
	if a.Method.IsAIPStandardDelete || (strings.HasPrefix(name, "Delete") && (verb == "" || verb == "DELETE")) {
		return MethodTypeDelete
	}
	return MethodTypeCustom
}

// GetCommandName maps an API method to a standard gcloud command name (in snake_case).
// This name is typically used for the command's file name.
func (a *MethodAdapter) GetCommandName() (string, error) {
	if a.Method == nil {
		return "", fmt.Errorf("method cannot be nil")
	}
	switch a.Type() {
	case MethodTypeGet:
		return "describe", nil
	case MethodTypeList:
		return "list", nil
	case MethodTypeCreate:
		return "create", nil
	case MethodTypeUpdate:
		return "update", nil
	case MethodTypeDelete:
		return "delete", nil
	default:
		// For custom methods (AIP-136), we try to extract the custom verb from the HTTP path.
		// The custom verb is the part after the colon (e.g., .../instances/*:exportData).
		if a.Method.PathInfo != nil && len(a.Method.PathInfo.Bindings) > 0 {
			binding := a.Method.PathInfo.Bindings[0]
			if binding.PathTemplate != nil && binding.PathTemplate.Verb != nil {
				return strcase.ToSnake(*binding.PathTemplate.Verb), nil
			}
		}
		// Fallback: use the method name converted to snake_case.
		return strcase.ToSnake(a.Method.Name), nil
	}
}

// IsStandardMethod determines if the method is one of the standard AIP methods
// (Get, List, Create, Update, Delete).
func (a *MethodAdapter) IsStandardMethod() bool {
	return a.Type() != MethodTypeCustom
}

// getHTTPVerb returns the HTTP verb from the primary binding, or an empty string if not available.
func (a *MethodAdapter) getHTTPVerb() string {
	if a.Method.PathInfo != nil && len(a.Method.PathInfo.Bindings) > 0 {
		return a.Method.PathInfo.Bindings[0].Verb
	}
	return ""
}

// isResourceMethod determines if the method operates on a specific resource instance.
// This includes standard Get, Update, Delete methods, and custom methods where the
// HTTP path ends with a variable segment (e.g. `.../instances/{instance}`).
func (a *MethodAdapter) isResourceMethod() bool {
	switch a.Type() {
	case MethodTypeGet, MethodTypeUpdate, MethodTypeDelete:
		return true
	case MethodTypeCreate, MethodTypeList:
		return false
	default:
		// Fallback for custom methods
		if a.Method.PathInfo == nil || len(a.Method.PathInfo.Bindings) == 0 {
			return false
		}
		template := a.Method.PathInfo.Bindings[0].PathTemplate
		if template == nil || len(template.Segments) == 0 {
			return false
		}
		lastSegment := template.Segments[len(template.Segments)-1]
		// If the path ends with a variable, it's a resource method.
		return lastSegment.Variable != nil
	}
}

// isCollectionMethod determines if the method operates on a collection of resources.
// This includes standard List and Create methods, and custom methods where the
// HTTP path ends with a literal segment (e.g. `.../instances`).
func (a *MethodAdapter) isCollectionMethod() bool {
	switch a.Type() {
	case MethodTypeList, MethodTypeCreate:
		return true
	case MethodTypeGet, MethodTypeUpdate, MethodTypeDelete:
		return false
	default:
		// Fallback for custom methods
		if a.Method.PathInfo == nil || len(a.Method.PathInfo.Bindings) == 0 {
			return false
		}
		template := a.Method.PathInfo.Bindings[0].PathTemplate
		if template == nil || len(template.Segments) == 0 {
			return false
		}
		lastSegment := template.Segments[len(template.Segments)-1]
		// If the path ends with a literal, it's a collection method.
		return lastSegment.Literal != nil
	}
}

// FindResourceMessage identifies the primary resource message within a List response.
// Per AIP-132, this is usually the repeated field in the response message.
func FindResourceMessage(outputType *api.Message) *api.Message {
	if outputType == nil {
		return nil
	}
	for _, f := range outputType.Fields {
		if f.Repeated && f.MessageType != nil {
			return f.MessageType
		}
	}
	return nil
}

// IsPrimaryResource determines if a field represents the primary resource of a method.
func (a *MethodAdapter) IsPrimaryResource(field *api.Field) bool {
	if a.Method.InputType == nil {
		return false
	}
	// For `Create` methods, the primary resource is identified by a field named
	// in the format "{resource}_id" (e.g., "instance_id").
	if a.Type() == MethodTypeCreate {
		resource, err := a.getResource()
		if err == nil {
			name := getResourceNameFromType(resource.Type)
			// Convert CamelCase resource name (e.g., "Instance") to snake_case
			// to match the proto field naming convention (e.g., "instance_id").
			if name != "" && field.Name == strcase.ToSnake(name)+"_id" {
				return true
			}
		}
	}

	// For collection-based methods (List and custom collection methods),
	// the primary resource scope is identified by the "parent" field.
	// Note: Create is collection-based but uses the new resource ID (e.g. "instance_id")
	// as the primary positional argument, so "parent" is not the primary resource arg.
	if a.isCollectionMethod() && a.Type() != MethodTypeCreate && field.Name == "parent" {
		return true
	}

	// For resource-based methods (Get, Delete, Update, and custom resource methods),
	// the primary resource is identified by the "name" field.
	if a.isResourceMethod() && field.Name == "name" {
		return true
	}

	return false
}

// getResource extracts the resource definition from a method's input message if it exists.
func (a *MethodAdapter) getResource() (*api.Resource, error) {
	for _, f := range a.Method.InputType.Fields {
		if f.MessageType != nil && f.MessageType.Resource != nil {
			return f.MessageType.Resource, nil
		}
	}
	return nil, fmt.Errorf("resource message not found in input type for method %q", a.Method.Name)
}

// GetResource finds the `api.Resource` definition associated with a method.
// This is a crucial function for linking a method to the resource it operates on.
func (a *MethodAdapter) GetResource(model *api.API) *api.Resource {
	if a.Method.InputType == nil {
		return nil
	}

	// Strategy 1: For Create (AIP-133) and Update (AIP-134), the request message
	// usually contains a field that *is* the resource message.
	if resource, err := a.getResource(); err == nil {
		return resource
	}

	// Strategy 2: For Get (AIP-131), Delete (AIP-135), and List (AIP-132), the
	// request message has a `name` or `parent` field with a `(google.api.resource_reference)`.
	var resourceType string
	for _, field := range a.Method.InputType.Fields {
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

// GetPluralResourceName determines the plural name of a resource. It follows a clear
// hierarchy of truth: first, the explicit `plural` field in the resource
// definition, and second, inference from the resource pattern.
func (a *MethodAdapter) GetPluralResourceName(model *api.API) string {
	resource := a.GetResource(model)
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

// GetSingularResourceName determines the singular name of a resource. It follows a clear
// hierarchy of truth: first, the explicit `singular` field in the resource
// definition, and second, inference from the resource pattern.
func (a *MethodAdapter) GetSingularResourceName(model *api.API) string {
	resource := a.GetResource(model)
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
