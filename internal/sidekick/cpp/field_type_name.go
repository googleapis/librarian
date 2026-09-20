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

package cpp

import (
	"fmt"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

var cppKeywords = map[string]bool{
	"alignas":          true,
	"alignof":          true,
	"and":              true,
	"and_eq":           true,
	"asm":              true,
	"auto":             true,
	"bitand":           true,
	"bitor":            true,
	"bool":             true,
	"break":            true,
	"case":             true,
	"catch":            true,
	"char":             true,
	"char8_t":          true,
	"char16_t":         true,
	"char32_t":         true,
	"class":            true,
	"compl":            true,
	"concept":          true,
	"const":            true,
	"consteval":        true,
	"constexpr":        true,
	"constinit":        true,
	"const_cast":       true,
	"continue":         true,
	"co_await":         true,
	"co_return":        true,
	"co_yield":         true,
	"decltype":         true,
	"default":          true,
	"delete":           true,
	"do":               true,
	"double":           true,
	"dynamic_cast":     true,
	"else":             true,
	"enum":             true,
	"explicit":         true,
	"export":           true,
	"extern":           true,
	"false":            true,
	"float":            true,
	"for":              true,
	"friend":           true,
	"goto":             true,
	"if":               true,
	"inline":           true,
	"int":              true,
	"long":             true,
	"mutable":          true,
	"namespace":        true,
	"new":              true,
	"noexcept":         true,
	"not":              true,
	"not_eq":           true,
	"nullptr":          true,
	"operator":         true,
	"or":               true,
	"or_eq":            true,
	"private":          true,
	"protected":        true,
	"public":           true,
	"reflexpr":         true,
	"register":         true,
	"reinterpret_cast": true,
	"requires":         true,
	"return":           true,
	"short":            true,
	"signed":           true,
	"sizeof":           true,
	"static":           true,
	"static_assert":    true,
	"static_cast":      true,
	"struct":           true,
	"switch":           true,
	"synchronized":     true,
	"template":         true,
	"this":             true,
	"thread_local":     true,
	"throw":            true,
	"true":             true,
	"try":              true,
	"typedef":          true,
	"typeid":           true,
	"typename":         true,
	"union":            true,
	"unsigned":         true,
	"using":            true,
	"virtual":          true,
	"void":             true,
	"volatile":         true,
	"wchar_t":          true,
	"while":            true,
	"xor":              true,
	"xor_eq":           true,
}

// CppParamName converts a protobuf field name to a C++ parameter name,
// appending an underscore if the name is a C++ keyword.
func CppParamName(name string) string {
	if cppKeywords[name] {
		return name + "_"
	}
	return name
}

// CppTypeToString converts an api.Field to its C++ type representation.
func CppTypeToString(field *api.Field) string {
	if field == nil {
		return ""
	}
	if field.Map {
		var keyType, valType string
		if field.MessageType != nil && len(field.MessageType.Fields) >= 2 {
			keyType = CppTypeToString(field.MessageType.Fields[0])
			valType = CppTypeToString(field.MessageType.Fields[1])
		}
		if keyType != "" && valType != "" {
			return fmt.Sprintf("std::map<%s, %s>", keyType, valType)
		}
	}
	if field.Repeated {
		return fmt.Sprintf("std::vector<%s>", cppBaseTypeToString(field))
	}
	return cppBaseTypeToString(field)
}

func cppBaseTypeToString(field *api.Field) string {
	switch field.Typez {
	case api.TypezInt32, api.TypezSint32, api.TypezSfixed32:
		return "std::int32_t"
	case api.TypezInt64, api.TypezSint64, api.TypezSfixed64:
		return "std::int64_t"
	case api.TypezUint32, api.TypezFixed32:
		return "std::uint32_t"
	case api.TypezUint64, api.TypezFixed64:
		return "std::uint64_t"
	case api.TypezDouble:
		return "double"
	case api.TypezFloat:
		return "float"
	case api.TypezBool:
		return "bool"
	case api.TypezString, api.TypezBytes:
		return "std::string"
	case api.TypezEnum, api.TypezMessage:
		return ProtoNameToCppName(field.TypezID)
	default:
		return ""
	}
}

// CppParamTypeToString converts an api.Field to its C++ parameter type representation (value vs const&).
func CppParamTypeToString(field *api.Field) string {
	if field == nil {
		return ""
	}
	if field.Map {
		return CppTypeToString(field) + " const&"
	}
	if field.Repeated {
		return CppTypeToString(field) + " const&"
	}
	if field.Typez == api.TypezMessage || field.Typez == api.TypezString || field.Typez == api.TypezBytes {
		return CppTypeToString(field) + " const&"
	}
	return CppTypeToString(field)
}
