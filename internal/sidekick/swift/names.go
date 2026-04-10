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

package swift

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/iancoleman/strcase"
)

// keywords defines the list of identifiers that cannot be used as Swift identifiers.
//
// These keywords were sourced, circa 2026-04-03, from [Keywords and Punctuation].
// At that time, 6.2 was the latest Swift version.
//
// [Keywords and Punctuation]: https://docs.swift.org/swift-book/documentation/the-swift-programming-language/lexicalstructure/#Keywords-and-Punctuation
var keywords = map[string]bool{
	// Keywords used in declarations:
	"associatedtype":  true,
	"borrowing":       true,
	"class":           true,
	"consuming":       true,
	"deinit":          true,
	"enum":            true,
	"extension":       true,
	"fileprivate":     true,
	"func":            true,
	"import":          true,
	"init":            true,
	"inout":           true,
	"internal":        true,
	"let":             true,
	"nonisolated":     true,
	"open":            true,
	"operator":        true,
	"precedencegroup": true,
	"private":         true,
	"protocol":        true,
	"public":          true,
	"rethrows":        true,
	"static":          true,
	"struct":          true,
	"subscript":       true,
	"typealias":       true,
	"var":             true,
	// Keywords used in statements:
	"break":       true,
	"case":        true,
	"catch":       true,
	"continue":    true,
	"default":     true,
	"defer":       true,
	"do":          true,
	"else":        true,
	"fallthrough": true,
	"for":         true,
	"guard":       true,
	"if":          true,
	"in":          true,
	"repeat":      true,
	"return":      true,
	"switch":      true,
	"throw":       true,
	"where":       true,
	"while":       true,
	// Keywords used in expressions and types:
	"Any":   true,
	"as":    true,
	"await": true,
	// "catch":    true,
	"false": true,
	"is":    true,
	"nil":   true,
	// "rethrows": true,
	"self":  true,
	"Self":  true,
	"super": true,
	// "throw":    true,
	"throws": true,
	"true":   true,
	"try":    true,
	// Keywords used in patterns:
	"_": true,
	// These are not used in Protobuf, Discovery Docs, or OpenAPI, but it is less confusing if they are listed:
	"#available":      true,
	"#colorLiteral":   true,
	"#else":           true,
	"#elseif":         true,
	"#endif":          true,
	"#fileLiteral":    true,
	"#if":             true,
	"#imageLiteral":   true,
	"#keyPath":        true,
	"#selector":       true,
	"#sourceLocation": true,
	"#unavailable":    true,
	// Keywords reserved in particular contexts:
	"associativity": true,
	"async":         true,
	"convenience":   true,
	"didSet":        true,
	"dynamic":       true,
	"final":         true,
	"get":           true,
	"indirect":      true,
	"infix":         true,
	"lazy":          true,
	"left":          true,
	"mutating":      true,
	"none":          true,
	"nonmutating":   true,
	"optional":      true,
	"override":      true,
	"package":       true,
	"postfix":       true,
	"precedence":    true,
	"prefix":        true,
	"Protocol":      true,
	"required":      true,
	"right":         true,
	"set":           true,
	"some":          true,
	"Type":          true,
	"unowned":       true,
	"weak":          true,
	"willSet":       true,
}

// escapeKeyword escapes a string if it is a keyword.
func escapeKeyword(s string) string {
	// In Swift we can use backtick escaping for most keywords except `Type`, `Protocol`, and `self`:
	//   https://docs.swift.org/swift-book/documentation/the-swift-programming-language/types/#Metatype-Type
	// In an expression like `Foo.Type` that is *always* the metatype of `Foo` and not the nested type `Type` even if `Foo` has such a nested type.
	if s == "Protocol" || s == "Type" || s == "self" {
		return fmt.Sprintf("%s_", s)
	}
	if keywords[s] {
		return fmt.Sprintf("`%s`", s)
	}
	return s
}

// camelCase converts an identifier to camelCase (note the leading lowercase) and, if needed, escapes it.
//
// This function is used for field names and method names, where the Swift style is `camelCase`.
func camelCase(s string) string {
	return escapeKeyword(strcase.ToLowerCamel(s))
}

// pascalCase converts an identifier to PascalCase (note the leading uppercase) and, if needed, escapes it.
//
// This function is used for services, messages, and enums, where the Swift style is `PascalCase`.
func pascalCase(s string) string {
	// In Swift, it is conventional to preserve ALL CAPS names:
	//     https://www.swift.org/documentation/api-design-guidelines/#conventions
	if strings.ToUpper(s) == s {
		return escapeKeyword(s)
	}
	// Symbols that are already `PascalCase` should need no mapping. This works
	// better than calling `strcase.ToCamel()` in cases like `IAMPolicy`, which
	// would be converted to `Iampolicy`. We are trusting that the original
	// name in API definition chose to keep the acronym for a reason.
	if unicode.IsUpper(rune(s[0])) && !strings.ContainsRune(s, '_') {
		return escapeKeyword(s)
	}
	return escapeKeyword(strcase.ToCamel(s))
}

// enumValueCaseName returns the name of the Swift enumeration case for a given enumeration value.
func enumValueCaseName(e *api.EnumValue) string {
	prefix := strcase.ToScreamingSnake(e.Parent.Name) + "_"
	trimmed := strings.TrimPrefix(e.Name, prefix)
	if strings.HasPrefix(e.Name, prefix) && strings.IndexFunc(trimmed, unicode.IsLetter) == 0 {
		return camelCase(trimmed)
	}
	trimNumbers := regexp.MustCompile(`_([0-9])`)
	prefix = trimNumbers.ReplaceAllString(prefix, `$1`)
	trimmed = strings.TrimPrefix(e.Name, prefix)
	if strings.HasPrefix(e.Name, prefix) && strings.IndexFunc(trimmed, unicode.IsLetter) == 0 {
		return camelCase(trimmed)
	}
	return camelCase(e.Name)
}
