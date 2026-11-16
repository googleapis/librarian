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

func maybeMapField(model *api.API, mb *api.MessageBuilder, input *property) (*api.Field, error) {
	if input.Schema.Type != "object" || input.Schema.Format != "" {
		return nil, nil
	}
	if input.Schema.AdditionalProperties == nil {
		return nil, nil
	}

	if field := maybeMapOfObjectField(model, mb, input); field != nil {
		return field, nil
	}
	if field, err := maybeMapOfEnumField(model, mb, input); err != nil || field != nil {
		return field, err
	}
	return maybeMapOfPrimitiveField(model, mb, input)
}

func maybeMapOfObjectField(model *api.API, mb *api.MessageBuilder, input *property) *api.Field {
	if input.Schema.AdditionalProperties.Ref == "" {
		return nil
	}
	message := mb.Build()
	valueTypezID := fmt.Sprintf(".%s.%s", model.PackageName, input.Schema.AdditionalProperties.Ref)
	typezID := insertMapType(model, api.TypezMessage, valueTypezID)
	field := api.NewFieldBuilder().
		WithName(input.Name).
		WithJSONName(input.Name).
		WithID(fmt.Sprintf("%s.%s", message.ID(), input.Name)).
		WithDocumentation(input.Schema.Description).
		WithTypez(api.TypezMap).
		WithTypezID(typezID).
		WithDeprecated(input.Schema.Deprecated).
		WithMap(true).
		Build()
	return field
}

func maybeMapOfEnumField(model *api.API, mb *api.MessageBuilder, input *property) (*api.Field, error) {
	if input.Schema.AdditionalProperties.Enums == nil {
		return nil, nil
	}
	if err := makeMessageEnum(model, mb, input.Name, input.Schema.AdditionalProperties); err != nil {
		return nil, err
	}
	message := mb.Build()
	valueTypezID := fmt.Sprintf("%s.%s", message.ID(), input.Name)
	typezID := insertMapType(model, api.TypezEnum, valueTypezID)
	field := api.NewFieldBuilder().
		WithName(input.Name).
		WithJSONName(input.Name).
		WithID(fmt.Sprintf("%s.%s", message.ID(), input.Name)).
		WithDocumentation(input.Schema.Description).
		WithTypez(api.TypezMap).
		WithTypezID(typezID).
		WithDeprecated(input.Schema.Deprecated).
		WithMap(true).
		Build()
	return field, nil
}

func maybeMapOfPrimitiveField(model *api.API, mb *api.MessageBuilder, input *property) (*api.Field, error) {
	message := mb.Build()
	valueTypez, valueTypezID, err := scalarType(model, message.ID(), input.Name, input.Schema.AdditionalProperties)
	if err != nil {
		return nil, err
	}
	typezID := insertMapType(model, valueTypez, valueTypezID)
	field := api.NewFieldBuilder().
		WithName(input.Name).
		WithJSONName(input.Name).
		WithID(fmt.Sprintf("%s.%s", message.ID(), input.Name)).
		WithDocumentation(input.Schema.Description).
		WithTypez(api.TypezMap).
		WithTypezID(typezID).
		WithDeprecated(input.Schema.Deprecated).
		WithMap(true).
		Build()
	return field, nil
}

func insertMapType(model *api.API, valueTypez api.Typez, valueTypezID string) string {
	id := fmt.Sprintf("$map<string, %s>", valueTypezID)
	if model.Map(id) != nil {
		return id
	}
	key := api.NewFieldBuilder().
		WithName("key").
		WithID(fmt.Sprintf("%s.key", id)).
		WithTypez(api.TypezString).
		WithTypezID("string").
		Build()
	value := api.NewFieldBuilder().
		WithName("value").
		WithID(fmt.Sprintf("%s.value", id)).
		WithTypez(valueTypez).
		WithTypezID(valueTypezID).
		Build()
	mapEntity := api.NewMapBuilder().
		WithID(id).
		WithKey(key).
		WithValue(value).
		Build()
	model.AddMap(mapEntity)
	return id
}
