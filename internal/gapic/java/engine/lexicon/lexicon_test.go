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

package lexicon

import (
	"testing"
)

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestKeywords(t *testing.T) {
	keywords := []string{
		"abstract", "assert", "boolean", "break", "byte", "case", "catch", "char", "class", "const",
		"continue", "default", "do", "double", "else", "enum", "extends", "final", "finally", "float",
		"for", "goto", "if", "implements", "import", "instanceof", "int", "interface", "long", "native",
		"new", "package", "private", "protected", "public", "return", "short", "static", "strictfp",
		"super", "switch", "synchronized", "this", "throw", "throws", "transient", "try", "void",
		"volatile", "while", "record", "var", "yield", "sealed", "non-sealed", "permits",
	}
	for _, kw := range keywords {
		if !IsKeyword(kw) {
			t.Errorf("expected %q to be a keyword", kw)
		}
	}
	nonKeywords := []string{"hello", "foo", "Bar", "getFoo", "client_id", "myClass"}
	for _, nonKw := range nonKeywords {
		if IsKeyword(nonKw) {
			t.Errorf("expected %q not to be a keyword", nonKw)
		}
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestLiterals(t *testing.T) {
	literals := []string{"true", "false", "null"}
	for _, lit := range literals {
		if !IsLiteral(lit) {
			t.Errorf("expected %q to be a literal", lit)
		}
	}
	if IsLiteral("undefined") {
		t.Errorf("expected 'undefined' not to be a Java literal")
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestOperators(t *testing.T) {
	ops := []string{"=", "+", "-", "*", "/", "%", "==", "!=", "<", ">", "<=", ">=", "&&", "||", "!", "+=", "-=", "->"}
	for _, op := range ops {
		if !IsOperator(op) {
			t.Errorf("expected %q to be an operator", op)
		}
	}
	if IsOperator("~=") {
		t.Errorf("expected '~=' not to be a Java operator")
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestSeparators(t *testing.T) {
	seps := []string{"(", ")", "{", "}", "[", "]", ";", ",", ".", "...", "@", "::"}
	for _, sep := range seps {
		if !IsSeparator(sep) {
			t.Errorf("expected %q to be a separator", sep)
		}
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestValidIdentifier(t *testing.T) {
	valid := []string{"foo", "_bar", "$baz", "CamelCase", "snake_case", "x123", "a_b_c$1"}
	for _, v := range valid {
		if !IsValidIdentifier(v) {
			t.Errorf("expected %q to be a valid identifier", v)
		}
	}
	invalid := []string{"", "123foo", "class", "default", "foo-bar", "a+b", "package"}
	for _, inv := range invalid {
		if IsValidIdentifier(inv) {
			t.Errorf("expected %q to be invalid identifier", inv)
		}
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestEscapeKeyword(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"package", "package_"},
		{"default", "default_"},
		{"class", "class_"},
		{"regularIdentifier", "regularIdentifier"},
	}
	for _, tt := range tests {
		got := EscapeKeyword(tt.input)
		if got != tt.expected {
			t.Errorf("EscapeKeyword(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
