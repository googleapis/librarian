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

	"github.com/googleapis/librarian/internal/sidekick/api"
)

// fieldTypeName returns the Swift type name for a field.
//
// The implementation is pretty simple for primitive types. For message and enum fields it may get more
// difficult as the name may be in a separate package.
func (c *codec) fieldTypeName(field *api.Field) (string, error) {
	switch field.Typez {
	case api.MESSAGE_TYPE:
		m, err := lookupMessage(c.Model, field.TypezID)
		if err != nil {
			return "", err
		}
		if m.IsMap {
			return "", fmt.Errorf("TODO(#5060) - map fields are not supported: %s", field.ID)
		}
		return "", fmt.Errorf("TODO(#5060) - message fields are not supported: %s", field.ID)
	case api.ENUM_TYPE:
		return "", fmt.Errorf("TODO(#5060) - enum fields are not supported: %s", field.ID)
	default:
		return scalarFieldTypeName(field)
	}
}

func scalarFieldTypeName(field *api.Field) (string, error) {
	switch field.Typez {
	case api.DOUBLE_TYPE:
		return "Double", nil
	case api.FLOAT_TYPE:
		return "Float", nil
	case api.INT64_TYPE:
		return "Int64", nil
	case api.UINT64_TYPE:
		return "UInt64", nil
	case api.INT32_TYPE:
		return "Int32", nil
	case api.FIXED64_TYPE:
		return "UInt64", nil
	case api.FIXED32_TYPE:
		return "UInt32", nil
	case api.BOOL_TYPE:
		return "Bool", nil
	case api.STRING_TYPE:
		return "String", nil
	case api.BYTES_TYPE:
		return "Data", nil
	case api.UINT32_TYPE:
		return "UInt32", nil
	case api.SFIXED32_TYPE:
		return "Int32", nil
	case api.SFIXED64_TYPE:
		return "Int64", nil
	case api.SINT32_TYPE:
		return "Int32", nil
	case api.SINT64_TYPE:
		return "Int64", nil
	default:
		return "", fmt.Errorf("unexpected Typez (%s) for scalar field %q", field.Typez.String(), field.ID)
	}
}
