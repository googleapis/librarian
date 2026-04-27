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

	"github.com/googleapis/librarian/internal/sidekick/api"
)

func makeMessageFields(model *api.API, message *api.Message, schema *schema) error {
	for _, input := range schema.Properties {
		field, err := makeField(model, message, input)
		if err != nil {
			return err
		}
		message.Fields = append(message.Fields, field)
	}
	return nil
}

func makeField(model *api.API, message *api.Message, input *property) (*api.Field, error) {
	if input.Schema.Type == "array" {
		return makeArrayField(model, message, input)
	}
	if field, err := maybeMapField(model, message, input); err != nil || field != nil {
		return field, err
	}
	if field, err := maybeInlineObjectField(model, message, input.Name, input.Schema); err != nil || field != nil {
		return field, err
	}
	return makeScalarField(model, message, input.Name, input.Schema)
}

func makeArrayField(model *api.API, message *api.Message, input *property) (*api.Field, error) {
	field, err := maybeInlineObjectField(model, message, input.Name, input.Schema.ItemSchema)
	if err != nil {
		return nil, err
	}
	if field != nil {
		field.Documentation = input.Schema.Description
		field.Repeated = true
		field.Optional = false
		return field, nil
	}
	field, err = makeScalarField(model, message, input.Name, input.Schema.ItemSchema)
	if err != nil {
		return nil, err
	}
	field.Documentation = input.Schema.Description
	field.Repeated = true
	field.Optional = false
	return field, nil
}

func makeScalarField(model *api.API, message *api.Message, name string, schema *schema) (*api.Field, error) {
	if err := makeMessageEnum(model, message, name, schema); err != nil {
		return nil, err
	}
	typez, typezID, err := scalarType(model, message.ID, name, schema)
	if err != nil {
		return nil, err
	}
	return &api.Field{
		Name:          name,
		JSONName:      name, // Discovery doc field names are always camelCase
		ID:            fmt.Sprintf("%s.%s", message.ID, name),
		Documentation: schema.Description,
		Typez:         typez,
		TypezID:       typezID,
		Deprecated:    schema.Deprecated,
		Optional:      true,
	}, nil
}

func scalarType(model *api.API, messageID, name string, input *schema) (api.Typez, string, error) {
	if input.Type == "" && input.Ref != "" {
		typezID := fmt.Sprintf(".%s.%s", model.PackageName, input.Ref)
		return api.TypezMessage, typezID, nil
	}
	switch input.Type {
	case "boolean":
		return api.TypezBool, "bool", nil
	case "integer":
		return scalarTypeForIntegerFormats(messageID, name, input)
	case "number":
		return scalarTypeForNumberFormats(messageID, name, input)
	case "string":
		return scalarTypeForStringFormats(messageID, name, input)
	case "any":
		return scalarTypeForAny(messageID, name, input)
	case "object":
		return scalarTypeForObject(messageID, name, input)
	}
	return unknownFormat("scalar", messageID, name, input)
}

func scalarTypeForIntegerFormats(messageID, name string, input *schema) (api.Typez, string, error) {
	switch input.Format {
	case "int32":
		return api.TypezInt32, "int32", nil
	case "uint32":
		return api.TypezUint32, "uint32", nil
	case "int64":
		return api.TypezInt64, "int64", nil
	case "uint64":
		return api.TypezUint64, "uint64", nil
	}
	return unknownFormat("integer", messageID, name, input)
}

func scalarTypeForNumberFormats(messageID, name string, input *schema) (api.Typez, string, error) {
	switch input.Format {
	case "float":
		return api.TypezFloat, "float", nil
	case "double":
		return api.TypezDouble, "double", nil
	}
	return unknownFormat("number", messageID, name, input)
}

func scalarTypeForStringFormats(messageID, name string, input *schema) (api.Typez, string, error) {
	if input.Enums != nil {
		return api.TypezEnum, fmt.Sprintf("%s.%s", messageID, name), nil
	}
	switch input.Format {
	case "":
		return api.TypezString, "string", nil
	case "byte":
		return api.TypezBytes, "bytes", nil
	case "date":
		return api.TypezString, "string", nil
	case "google-duration":
		return api.TypezMessage, ".google.protobuf.Duration", nil
	case "date-time", "google-datetime":
		return api.TypezMessage, ".google.protobuf.Timestamp", nil
	case "google-fieldmask":
		return api.TypezMessage, ".google.protobuf.FieldMask", nil
	case "int64":
		return api.TypezInt64, "int64", nil
	case "uint64":
		return api.TypezUint64, "uint64", nil
	}
	return unknownFormat("string", messageID, name, input)
}

func scalarTypeForAny(messageID, name string, input *schema) (api.Typez, string, error) {
	switch input.Format {
	case "google.protobuf.Value":
		return api.TypezMessage, ".google.protobuf.Value", nil
	}
	return unknownFormat("any", messageID, name, input)
}

func scalarTypeForObject(messageID, name string, input *schema) (api.Typez, string, error) {
	switch input.Format {
	case "google.protobuf.Struct":
		return api.TypezMessage, ".google.protobuf.Struct", nil
	case "google.protobuf.Any":
		return api.TypezMessage, ".google.protobuf.Any", nil
	}
	return unknownFormat("object", messageID, name, input)
}

func unknownFormat(baseType, messageID, name string, input *schema) (api.Typez, string, error) {
	return 0, "", fmt.Errorf("unknown %s format (%s) for field %s.%s", baseType, input.Format, messageID, name)
}
