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

// Package gcloud generates a self-contained Go CLI module whose command tree
// mirrors the gcloud command surface for a parsed API model. Generated code
// uses github.com/urfave/cli/v3 to implement the command tree.
package gcloud

import "strings"

// goReservedWords lists Go keywords and predeclared identifiers that should
// not be used as local variable names in generated code. See
// https://go.dev/ref/spec#Keywords.
var goReservedWords = map[string]bool{
	"break":       true,
	"case":        true,
	"chan":        true,
	"const":       true,
	"continue":    true,
	"default":     true,
	"defer":       true,
	"else":        true,
	"fallthrough": true,
	"for":         true,
	"func":        true,
	"go":          true,
	"goto":        true,
	"if":          true,
	"import":      true,
	"interface":   true,
	"map":         true,
	"package":     true,
	"range":       true,
	"return":      true,
	"select":      true,
	"struct":      true,
	"switch":      true,
	"type":        true,
	"var":         true,
	"any":         true,
	"bool":        true,
	"byte":        true,
	"error":       true,
	"false":       true,
	"float32":     true,
	"float64":     true,
	"int":         true,
	"int8":        true,
	"int16":       true,
	"int32":       true,
	"int64":       true,
	"new":         true,
	"nil":         true,
	"rune":        true,
	"string":      true,
	"true":        true,
	"uint":        true,
	"uint8":       true,
	"uint16":      true,
	"uint32":      true,
	"uint64":      true,
	"uintptr":     true,
}

// kebabToLowerCamel converts a kebab-case identifier to lowerCamelCase. If the
// resulting name collides with a Go reserved word, the name is suffixed with
// an underscore.
func kebabToLowerCamel(name string) string {
	parts := strings.Split(name, "-")
	var sb strings.Builder
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == 0 {
			sb.WriteString(strings.ToLower(p))
			continue
		}
		sb.WriteString(strings.ToUpper(p[:1]))
		sb.WriteString(strings.ToLower(p[1:]))
	}
	out := sb.String()
	if goReservedWords[out] {
		return out + "_"
	}
	return out
}

// kebabToUpperCamel converts a kebab-case identifier to UpperCamelCase.
func kebabToUpperCamel(name string) string {
	parts := strings.Split(name, "-")
	var sb strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		sb.WriteString(strings.ToUpper(p[:1]))
		sb.WriteString(strings.ToLower(p[1:]))
	}
	return sb.String()
}

// snakeToKebab converts a snake_case identifier to kebab-case.
func snakeToKebab(name string) string {
	return strings.ReplaceAll(name, "_", "-")
}
