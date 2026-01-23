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

// GetCommandName maps an API method to a standard gcloud command name (in snake_case).
// This name is typically used for the command's file name.
func GetCommandName(method *api.Method) (string, error) {
	if method == nil {
		return "", fmt.Errorf("method cannot be nil")
	}
	switch {
	case IsGet(method):
		return "describe", nil
	case IsList(method):
		return "list", nil
	case IsCreate(method):
		return "create", nil
	case IsUpdate(method):
		return "update", nil
	case IsDelete(method):
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

// IsCreate determines if the method is a standard Create method (AIP-133).
func IsCreate(m *api.Method) bool {
	if !strings.HasPrefix(m.Name, "Create") {
		return false
	}
	if verb := getHTTPVerb(m); verb != "" {
		return verb == "POST"
	}
	return true
}

// IsGet determines if the method is a standard Get method (AIP-131).
func IsGet(m *api.Method) bool {
	// Use sidekick's robust AIP check if available.
	if m.AIPStandardGetInfo() != nil {
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

// IsList determines if the method is a standard List method (AIP-132).
func IsList(m *api.Method) bool {
	if !strings.HasPrefix(m.Name, "List") {
		return false
	}
	if verb := getHTTPVerb(m); verb != "" {
		return verb == "GET"
	}
	return true
}

// IsUpdate determines if the method is a standard Update method (AIP-134).
func IsUpdate(m *api.Method) bool {
	if !strings.HasPrefix(m.Name, "Update") {
		return false
	}
	if verb := getHTTPVerb(m); verb != "" {
		return verb == "PATCH" || verb == "PUT"
	}
	return true
}

// IsDelete determines if the method is a standard Delete method (AIP-135).
func IsDelete(m *api.Method) bool {
	// Use sidekick's robust AIP check if available.
	if m.AIPStandardDeleteInfo() != nil {
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

// IsStandardMethod determines if the method is one of the standard AIP methods
// (Get, List, Create, Update, Delete).
func IsStandardMethod(m *api.Method) bool {
	return IsGet(m) || IsList(m) || IsCreate(m) || IsUpdate(m) || IsDelete(m)
}

// IsCustomMethod determines if the method is a custom method (i.e., not a standard AIP method).
func IsCustomMethod(m *api.Method) bool {
	return !IsStandardMethod(m)
}

// getHTTPVerb returns the HTTP verb from the primary binding, or an empty string if not available.
func getHTTPVerb(m *api.Method) string {
	if m.PathInfo != nil && len(m.PathInfo.Bindings) > 0 {
		return m.PathInfo.Bindings[0].Verb
	}
	return ""
}

// IsResourceMethod determines if the method operates on a specific resource instance.
// This includes standard Get, Update, Delete methods, and custom methods where the
// HTTP path ends with a variable segment (e.g. `.../instances/{instance}`).
func IsResourceMethod(m *api.Method) bool {
	switch {
	case IsGet(m), IsUpdate(m), IsDelete(m):
		return true
	case IsCreate(m), IsList(m):
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

// IsCollectionMethod determines if the method operates on a collection of resources.
// This includes standard List and Create methods, and custom methods where the
// HTTP path ends with a literal segment (e.g. `.../instances`).
func IsCollectionMethod(m *api.Method) bool {
	switch {
	case IsList(m), IsCreate(m):
		return true
	case IsGet(m), IsUpdate(m), IsDelete(m):
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
