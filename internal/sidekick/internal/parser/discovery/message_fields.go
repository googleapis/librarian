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

package discovery

import (
	"fmt"

	"github.com/googleapis/librarian/internal/sidekick/internal/api"
)

func makeMessageFields(messageID string, schema *schema) ([]*api.Field, error) {
	var fields []*api.Field
	for _, input := range schema.Properties {
		field, err := makeField(messageID, input)
		if err != nil {
			return nil, err
		}
		if field == nil {
			continue
		}
		fields = append(fields, field)
	}
	return fields, nil
}

func makeField(messageID string, input *property) (*api.Field, error) {
	switch input.Schema.Type {
	case "":
		return nil, nil
	case "array":
		return nil, nil
	case "object":
		return nil, nil
	default:
		return makeScalarField(messageID, input)
	}
}

func makeScalarField(messageID string, input *property) (*api.Field, error) {
	typez, typezID, err := scalarType(messageID, input)
	if err != nil {
		return nil, err
	}
	return &api.Field{
		Name:          input.Name,
		JSONName:      input.Name, // OpenAPI field names are always camelCase
		Documentation: input.Schema.Description,
		Typez:         typez,
		TypezID:       typezID,
		// TODO(#1850) - deprecated fields?
		// TODO(#1850) - optional fields?
	}, nil
}

func scalarType(messageID string, input *property) (api.Typez, string, error) {
	switch input.Schema.Type {
	// TODO(#1850) - handle "any", "object":
	case "boolean":
		return api.BOOL_TYPE, "bool", nil
	case "integer":
		return scalarTypeForIntegerFormats(messageID, input)
	case "number":
		return scalarTypeForNumberFormats(messageID, input)
	case "string":
		return scalarTypeForStringFormats(messageID, input)
	}
	return 0, "", fmt.Errorf("unknown scalar type for field %s.%s: %v", messageID, input.Name, input.Schema.Type)
}

func scalarTypeForIntegerFormats(messageID string, input *property) (api.Typez, string, error) {
	switch input.Schema.Format {
	case "int32":
		return api.INT32_TYPE, "int32", nil
	case "uint32":
		return api.UINT32_TYPE, "uint32", nil
	case "int64":
		return api.INT64_TYPE, "int64", nil
	case "uint64":
		return api.UINT64_TYPE, "uint64", nil
	}
	return unknownFormat("integer", messageID, input)
}

func scalarTypeForNumberFormats(messageID string, input *property) (api.Typez, string, error) {
	switch input.Schema.Format {
	case "float":
		return api.FLOAT_TYPE, "float", nil
	case "double":
		return api.DOUBLE_TYPE, "double", nil
	}
	return unknownFormat("number", messageID, input)
}

func scalarTypeForStringFormats(messageID string, input *property) (api.Typez, string, error) {
	switch input.Schema.Format {
	case "":
		return api.STRING_TYPE, "string", nil
	case "byte":
		return api.BYTES_TYPE, "bytes", nil
	case "date":
		return api.STRING_TYPE, "string", nil
	case "google-duration":
		return api.MESSAGE_TYPE, ".google.protobuf.Duration", nil
	case "date-time", "google-datetime":
		return api.MESSAGE_TYPE, ".google.protobuf.Timestamp", nil
	case "google-fieldmask":
		return api.MESSAGE_TYPE, ".google.protobuf.FieldMask", nil
	case "int64":
		return api.INT64_TYPE, "int64", nil
	case "uint64":
		return api.UINT64_TYPE, "uint64", nil
	}
	return unknownFormat("string", messageID, input)
}

func unknownFormat(baseType, messageID string, input *property) (api.Typez, string, error) {
	return 0, "", fmt.Errorf("unknown %s format (%s) for field %s.%s", baseType, input.Schema.Format, messageID, input.Name)
}
