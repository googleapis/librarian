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

type MethodType int

const (
	MethodTypeUnknown MethodType = iota
	MethodTypeGet
	MethodTypeList
	MethodTypeCreate
	MethodTypeUpdate
	MethodTypeDelete
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
