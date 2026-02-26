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

package gcloud

import (
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/iancoleman/strcase"
)

// getCommandName maps an API method to a standard gcloud command name (in snake_case).
// This name is typically used for the command's file name.
func getCommandName(method *api.Method) (string, error) {
	if method == nil {
		return "", fmt.Errorf("method cannot be nil")
	}
	switch {
	case isGet(method):
		return "describe", nil
	case isList(method):
		return "list", nil
	case isCreate(method):
		return "create", nil
	case isUpdate(method):
		return "update", nil
	case isDelete(method):
		return "delete", nil
	default:
		// For custom methods (AIP-136), we try to extract the custom verb from the HTTP path.
		// The custom verb is the part after the colon (e.g., .../instances/*:exportData).
		if method.PathInfo != nil && len(method.PathInfo.Bindings) > 0 {
			binding := method.PathInfo.Bindings[0]
			if binding.PathTemplate != nil && binding.PathTemplate.Verb != nil {
				return strcase.ToSnake(*binding.PathTemplate.Verb), nil
			}
		}
		// Fallback: use the method name converted to snake_case.
		return strcase.ToSnake(method.Name), nil
	}
}

// isCreate determines if the method is a standard Create method (AIP-133).
func isCreate(m *api.Method) bool {
	if !strings.HasPrefix(m.Name, "Create") {
		return false
	}
	if verb := getHTTPVerb(m); verb != "" {
		return verb == "POST"
	}
	return true
}

// isGet determines if the method is a standard Get method (AIP-131).
func isGet(m *api.Method) bool {
	// Use sidekick's robust AIP check if available.
	if m.IsAIPStandardGet {
		return true
	}
	// Fallback heuristic
	if !strings.HasPrefix(m.Name, "Get") {
		return false
	}
	if verb := getHTTPVerb(m); verb != "" {
		return verb == "GET"
	}
	return true
}

// isList determines if the method is a standard List method (AIP-132).
func isList(m *api.Method) bool {
	if !strings.HasPrefix(m.Name, "List") {
		return false
	}
	if verb := getHTTPVerb(m); verb != "" {
		return verb == "GET"
	}
	return true
}

// isUpdate determines if the method is a standard Update method (AIP-134).
func isUpdate(m *api.Method) bool {
	if !strings.HasPrefix(m.Name, "Update") {
		return false
	}
	if verb := getHTTPVerb(m); verb != "" {
		return verb == "PATCH" || verb == "PUT"
	}
	return true
}

// isDelete determines if the method is a standard Delete method (AIP-135).
func isDelete(m *api.Method) bool {
	// Use sidekick's robust AIP check if available.
	if m.IsAIPStandardDelete {
		return true
	}
	// Fallback heuristic
	if !strings.HasPrefix(m.Name, "Delete") {
		return false
	}
	if verb := getHTTPVerb(m); verb != "" {
		return verb == "DELETE"
	}
	return true
}

// isStandardMethod determines if the method is one of the standard AIP methods
// (Get, List, Create, Update, Delete).
func isStandardMethod(m *api.Method) bool {
	return isGet(m) || isList(m) || isCreate(m) || isUpdate(m) || isDelete(m)
}

// getHTTPVerb returns the HTTP verb from the primary binding, or an empty string if not available.
func getHTTPVerb(m *api.Method) string {
	if m.PathInfo != nil && len(m.PathInfo.Bindings) > 0 {
		return m.PathInfo.Bindings[0].Verb
	}
	return ""
}

// isResourceMethod determines if the method operates on a specific resource instance.
// This includes standard Get, Update, Delete methods, and custom methods where the
// HTTP path ends with a variable segment (e.g. `.../instances/{instance}`).
func isResourceMethod(m *api.Method) bool {
	switch {
	case isGet(m), isUpdate(m), isDelete(m):
		return true
	case isCreate(m), isList(m):
		return false
	default:
		// Fallback for custom methods
		if m.PathInfo == nil || len(m.PathInfo.Bindings) == 0 {
			return false
		}
		template := m.PathInfo.Bindings[0].PathTemplate
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
func isCollectionMethod(m *api.Method) bool {
	switch {
	case isList(m), isCreate(m):
		return true
	case isGet(m), isUpdate(m), isDelete(m):
		return false
	default:
		// Fallback for custom methods
		if m.PathInfo == nil || len(m.PathInfo.Bindings) == 0 {
			return false
		}
		template := m.PathInfo.Bindings[0].PathTemplate
		if template == nil || len(template.Segments) == 0 {
			return false
		}
		lastSegment := template.Segments[len(template.Segments)-1]
		// If the path ends with a literal, it's a collection method.
		return lastSegment.Literal != nil
	}
}

// findResourceMessage identifies the primary resource message within a List response.
// Per AIP-132, this is usually the repeated field in the response message.
func findResourceMessage(outputType *api.Message) *api.Message {
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
