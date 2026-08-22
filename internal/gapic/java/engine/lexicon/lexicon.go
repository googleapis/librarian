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

// Package lexicon provides Java syntax lexicon checks (keywords, literals, operators, identifiers).
package lexicon

import "unicode"

var javaKeywords = map[string]bool{
	"abstract":     true,
	"assert":       true,
	"boolean":      true,
	"break":        true,
	"byte":         true,
	"case":         true,
	"catch":        true,
	"char":         true,
	"class":        true,
	"const":        true,
	"continue":     true,
	"default":      true,
	"do":           true,
	"double":       true,
	"else":         true,
	"enum":         true,
	"extends":      true,
	"final":        true,
	"finally":      true,
	"float":        true,
	"for":          true,
	"goto":         true,
	"if":           true,
	"implements":   true,
	"import":       true,
	"instanceof":   true,
	"int":          true,
	"interface":    true,
	"long":         true,
	"native":       true,
	"new":          true,
	"package":      true,
	"private":      true,
	"protected":    true,
	"public":       true,
	"return":       true,
	"short":        true,
	"static":       true,
	"strictfp":     true,
	"super":        true,
	"switch":       true,
	"synchronized": true,
	"this":         true,
	"throw":        true,
	"throws":       true,
	"transient":    true,
	"try":          true,
	"void":         true,
	"volatile":     true,
	"while":        true,
	// Contextual / literal keywords
	"record":     true,
	"sealed":     true,
	"non-sealed": true,
	"permits":    true,
	"yield":      true,
	"var":        true,
	"true":       true,
	"false":      true,
	"null":       true,
}

var javaLiterals = map[string]bool{
	"true":  true,
	"false": true,
	"null":  true,
}

var javaOperators = map[string]bool{
	"=":    true,
	">":    true,
	"<":    true,
	"!":    true,
	"~":    true,
	"?":    true,
	":":    true,
	"->":   true,
	"==":   true,
	">=":   true,
	"<=":   true,
	"!=":   true,
	"&&":   true,
	"||":   true,
	"++":   true,
	"--":   true,
	"+":    true,
	"-":    true,
	"*":    true,
	"/":    true,
	"&":    true,
	"|":    true,
	"^":    true,
	"%":    true,
	"<<":   true,
	">>":   true,
	">>>":  true,
	"+=":   true,
	"-=":   true,
	"*=":   true,
	"/=":   true,
	"&=":   true,
	"|=":   true,
	"^=":   true,
	"%=":   true,
	"<<=":  true,
	">>=":  true,
	">>>=": true,
}

var javaSeparators = map[string]bool{
	"(":   true,
	")":   true,
	"{":   true,
	"}":   true,
	"[":   true,
	"]":   true,
	";":   true,
	",":   true,
	".":   true,
	"...": true,
	"@":   true,
	"::":  true,
}

// IsKeyword returns true if s is a reserved Java keyword or reserved identifier.
func IsKeyword(s string) bool {
	return javaKeywords[s]
}

// IsLiteral returns true if s is a boolean or null literal.
func IsLiteral(s string) bool {
	return javaLiterals[s]
}

// IsOperator returns true if s is a Java operator symbol.
func IsOperator(s string) bool {
	return javaOperators[s]
}

// IsSeparator returns true if s is a Java separator symbol.
func IsSeparator(s string) bool {
	return javaSeparators[s]
}

// IsValidIdentifier returns true if s is a valid Java identifier.
func IsValidIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	if IsKeyword(s) {
		return false
	}
	runes := []rune(s)
	if !unicode.IsLetter(runes[0]) && runes[0] != '_' && runes[0] != '$' {
		return false
	}
	for _, r := range runes[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '$' {
			return false
		}
	}
	return true
}

// EscapeKeyword appends an underscore if s is a reserved keyword in Java.
func EscapeKeyword(s string) string {
	if IsKeyword(s) {
		return s + "_"
	}
	return s
}
