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

package python

import (
	"strings"

	"github.com/iancoleman/strcase"
)

// pythonKeywords contains keywords reserved in Python 3.
var pythonKeywords = map[string]bool{
	"False":    true,
	"None":     true,
	"True":     true,
	"and":      true,
	"as":       true,
	"assert":   true,
	"async":    true,
	"await":    true,
	"break":    true,
	"class":    true,
	"continue": true,
	"def":      true,
	"del":      true,
	"elif":     true,
	"else":     true,
	"except":   true,
	"finally":  true,
	"for":      true,
	"from":     true,
	"global":   true,
	"if":       true,
	"import":   true,
	"in":       true,
	"is":       true,
	"lambda":   true,
	"nonlocal": true,
	"not":      true,
	"or":       true,
	"pass":     true,
	"raise":    true,
	"return":   true,
	"try":      true,
	"while":    true,
	"with":     true,
	"yield":    true,
}

// snakeCase converts a string to snake_case.
func snakeCase(s string) string {
	if strings.ToLower(s) == s && !strings.ContainsRune(s, '-') {
		return s
	}
	return strcase.ToSnake(s)
}

// pascalCase converts a string to PascalCase.
func pascalCase(s string) string {
	return strcase.ToCamel(s)
}

// pythonIdentifier escapes keywords with a trailing underscore.
func pythonIdentifier(s string) string {
	if pythonKeywords[s] {
		return s + "_"
	}
	return s
}

// formatDocLines splits documentation into lines, trimming trailing whitespace.
func formatDocLines(doc string) []string {
	if doc == "" {
		return nil
	}
	lines := strings.Split(doc, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		result = append(result, strings.TrimRight(line, " \t\r"))
	}
	return result
}
