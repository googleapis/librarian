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

package gcloud

import (
	"regexp"
	"strings"
)

// Regular expressions used for case conversion.
var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

// toSnakeCase converts a camelCase or PascalCase string to snake_case.
// This is a common utility for converting API field names to a more
// command-line friendly format.
// For example, "apiFieldName" becomes "api_field_name".
func toSnakeCase(str string) string {
	// This first replacement handles cases like "APIFoo" -> "API_Foo".
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	// This second replacement handles cases like "FooBar" -> "Foo_Bar".
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// ToKebabCase converts a camelCase or PascalCase string to kebab-case.
// This is the standard format for gcloud command-line flags.
// For example, "apiFieldName" becomes "api-field-name".
func ToKebabCase(str string) string {
	// First, convert the string to snake_case.
	snake := toSnakeCase(str)
	// Then, replace all underscores with hyphens.
	return strings.ReplaceAll(snake, "_", "-")
}
