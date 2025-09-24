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

func maybeMapField(model *api.API, message *api.Message, input *property) (*api.Field, error) {
	if input.Schema.Type != "object" || input.Schema.Format != "" {
		return nil, nil
	}
	if input.Schema.AdditionalProperties == nil {
		return nil, nil
	}

	if field := maybeMapOfObjectField(model, message, input); field != nil {
		return field, nil
	}
	if field, err := maybeMapOfEnumField(model, message, input); err != nil || field != nil {
		return field, err
	}
	return maybeMapOfPrimitiveField(model, message, input)
}

func maybeMapOfObjectField(model *api.API, message *api.Message, input *property) *api.Field {
	if input.Schema.AdditionalProperties.Ref == "" {
		return nil
	}
	valueTypezID := fmt.Sprintf(".%s.%s", model.PackageName, input.Schema.AdditionalProperties.Ref)
	typezID := insertMapType(model, api.MESSAGE_TYPE, valueTypezID)
	field := &api.Field{
		Name:          input.Name,
		JSONName:      input.Name,
		ID:            fmt.Sprintf("%s.%s", message.ID, input.Name),
		Documentation: input.Schema.Description,
		Typez:         api.MESSAGE_TYPE,
		TypezID:       typezID,
	}
	return field
}

func maybeMapOfEnumField(model *api.API, message *api.Message, input *property) (*api.Field, error) {
	if input.Schema.AdditionalProperties.Enums == nil {
		return nil, nil
	}
	if err := makeMessageEnum(model, message, input.Name, input.Schema.AdditionalProperties); err != nil {
		return nil, err
	}
	valueTypezID := fmt.Sprintf("%s.%s", message.ID, input.Name)
	typezID := insertMapType(model, api.ENUM_TYPE, valueTypezID)
	field := &api.Field{
		Name:          input.Name,
		JSONName:      input.Name,
		ID:            fmt.Sprintf("%s.%s", message.ID, input.Name),
		Documentation: input.Schema.Description,
		Typez:         api.MESSAGE_TYPE,
		TypezID:       typezID,
	}
	return field, nil
}

func maybeMapOfPrimitiveField(model *api.API, message *api.Message, input *property) (*api.Field, error) {
	valueTypez, valueTypezID, err := scalarType(model, message.ID, input.Name, input.Schema.AdditionalProperties)
	if err != nil {
		return nil, err
	}
	typezID := insertMapType(model, valueTypez, valueTypezID)
	field := &api.Field{
		Name:          input.Name,
		JSONName:      input.Name,
		ID:            fmt.Sprintf("%s.%s", message.ID, input.Name),
		Documentation: input.Schema.Description,
		Typez:         api.MESSAGE_TYPE,
		TypezID:       typezID,
	}
	return field, nil
}

func insertMapType(model *api.API, valueTypez api.Typez, valueTypezId string) string {
	id := fmt.Sprintf("$map<string, %s>", valueTypezId)
	if _, ok := model.State.MessageByID[id]; ok {
		return id
	}
	key := &api.Field{
		Name:    "key",
		ID:      fmt.Sprintf("%s.key", id),
		Typez:   api.STRING_TYPE,
		TypezID: "string",
	}
	value := &api.Field{
		Name:    "value",
		ID:      fmt.Sprintf("%s.value", id),
		Typez:   valueTypez,
		TypezID: valueTypezId,
	}
	message := &api.Message{
		Name:          id,
		ID:            id,
		Documentation: id,
		Package:       "$",
		IsMap:         true,
		Fields:        []*api.Field{key, value},
	}
	model.State.MessageByID[message.ID] = message
	return id
}
