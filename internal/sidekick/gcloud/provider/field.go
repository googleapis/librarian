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

package provider

import (
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/iancoleman/strcase"
)

// CleanDocumentation removes common prefixes like "Required. ", "Identifier. ", etc from a help text string in any order.
func CleanDocumentation(s string) string {
	for {
		original := s
		s = strings.TrimPrefix(s, "Required. ")
		s = strings.TrimPrefix(s, "Identifier. ")
		s = strings.TrimPrefix(s, "Optional. ")
		if s == original {
			break
		}
	}
	return s
}

// GetGcloudType maps an API field type to the corresponding gcloud argparse type.
func GetGcloudType(t api.Typez) string {
	switch t {
	case api.TypezDouble, api.TypezFloat:
		return "float"
	case api.TypezInt64, api.TypezUint64, api.TypezFixed64, api.TypezSfixed64, api.TypezSint64:
		return "long"
	case api.TypezInt32, api.TypezFixed32, api.TypezUint32, api.TypezSfixed32, api.TypezSint32:
		return "int"
	case api.TypezBool:
		return "bool"
	case api.TypezString, api.TypezEnum:
		return "str"
	case api.TypezBytes:
		return "bytes"
	case api.TypezMessage, api.TypezGroup:
		return "arg_object"
	default:
		panic(fmt.Sprintf("unsupported API type: %v", t))
	}
}

// IsSafeName checks if a name contains only safe characters (alphanumeric, underscores, dots).
// This prevents injection vulnerabilities when generating code or templates.
func IsSafeName(name string) bool {
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}

// GetFieldHelpText returns the help text for a field, using overrides or documentation if available.
func GetFieldHelpText(overrides *Config, field *api.Field) string {
	if rule := FindFieldHelpTextRule(overrides, field.ID); rule != nil {
		return rule.HelpText.Brief
	}
	if field.Documentation != "" {
		return CleanDocumentation(strings.TrimSpace(field.Documentation))
	}
	return fmt.Sprintf("Value for the `%s` field.", strcase.ToKebab(field.Name))
}
