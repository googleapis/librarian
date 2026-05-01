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

// keywords lists Go keywords and predeclared identifiers that should
// not be used as local variable names in generated code. See
// https://go.dev/ref/spec#Keywords and
// https://go.dev/ref/spec#Predeclared_identifiers.
var keywords = map[string]bool{
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
	"append":      true,
	"bool":        true,
	"byte":        true,
	"cap":         true,
	"clear":       true,
	"close":       true,
	"comparable":  true,
	"complex":     true,
	"complex128":  true,
	"complex64":   true,
	"copy":        true,
	"delete":      true,
	"error":       true,
	"false":       true,
	"float32":     true,
	"float64":     true,
	"imag":        true,
	"int":         true,
	"int8":        true,
	"int16":       true,
	"int32":       true,
	"int64":       true,
	"iota":        true,
	"len":         true,
	"make":        true,
	"max":         true,
	"min":         true,
	"new":         true,
	"nil":         true,
	"panic":       true,
	"print":       true,
	"println":     true,
	"real":        true,
	"recover":     true,
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

// escapeKeyword escapes a string if it is a keyword.
func escapeKeyword(s string) string {
	if _, ok := keywords[s]; ok {
		return s + "_"
	}
	return s
}
