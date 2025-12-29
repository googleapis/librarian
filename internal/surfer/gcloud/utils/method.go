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

	"github.com/iancoleman/strcase"
)

// GetVerb maps an API method name to a standard gcloud command verb.
func GetVerb(methodName string) (string, error) {
	if methodName == "" {
		return "", fmt.Errorf("method name cannot be empty")
	}
	switch {
	case IsGet(methodName):
		return "describe", nil
	case IsList(methodName):
		return "list", nil
	case IsCreate(methodName):
		return "create", nil
	case IsUpdate(methodName):
		return "update", nil
	case IsDelete(methodName):
		return "delete", nil
	default:
		// For non-standard methods, we just use the snake_case version of the method name.
		return strcase.ToSnake(methodName), nil
	}
}

// TODO(https://github.com/googleapis/librarian/issues/3362): implement a robust AIP-compliant method idetification.
// IsCreate determines if the method is a Create method.
func IsCreate(methodName string) bool {
	return strings.HasPrefix(methodName, "Create")
}

// IsGet determines if the method is a Get method.
func IsGet(methodName string) bool {
	return strings.HasPrefix(methodName, "Get")
}

// IsList determines if the method is a List method.
func IsList(methodName string) bool {
	return strings.HasPrefix(methodName, "List")
}

// IsUpdate determines if the method is a Update method.
func IsUpdate(methodName string) bool {
	return strings.HasPrefix(methodName, "Update")
}

// IsDelete determines if the method is a Delete method.
func IsDelete(methodName string) bool {
	return strings.HasPrefix(methodName, "Delete")
}
