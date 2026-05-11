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
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

const (
	typeURLPrefix = "type.googleapis.com/"
)

type messageAnnotations struct {
	Name                string
	DocLines            []string
	Model               *modelAnnotations
	TypeURL             string
	CustomSerialization bool

	IsPaginatedResponse bool
	PageableItemField   string
	PageableItemType    string
	ImportsGax          bool
}

func (c *codec) annotateMessage(message *api.Message, model *modelAnnotations) error {
	if dep, ok := c.ApiPackages[message.Package]; ok {
		dep.Required = true
	}
	if message.Codec != nil {
		return nil
	}
	docLines, err := c.formatDocumentation(message.Documentation, message.Scopes())
	if err != nil {
		return err
	}
	annotations := &messageAnnotations{
		Name:                pascalCase(message.Name),
		DocLines:            docLines,
		Model:               model,
		TypeURL:             typeURLPrefix + strings.TrimPrefix(message.ID, "."),
		CustomSerialization: len(message.OneOfs) > 0,
	}

	message.Codec = annotations
	for _, oneof := range message.OneOfs {
		if err := c.annotateOneOf(oneof); err != nil {
			return err
		}
	}
	for _, field := range message.Fields {
		if err := c.annotateField(field); err != nil {
			return err
		}
		if fieldCodec, ok := field.Codec.(*fieldAnnotations); ok && fieldCodec.Name != field.JSONName {
			annotations.CustomSerialization = true
		}
	}

	annotations.IsPaginatedResponse = message.Pagination != nil
	annotations.ImportsGax = annotations.IsPaginatedResponse
	if message.Pagination != nil {
		itemField := message.Pagination.PageableItem
		itemFieldCodec, ok := itemField.Codec.(*fieldAnnotations)
		if !ok {
			return fmt.Errorf("internal error: pageable item field %q is not annotated", itemField.Name)
		}
		annotations.PageableItemField = itemFieldCodec.Name
		annotations.PageableItemType = itemFieldCodec.BaseFieldType
	}

	for _, nested := range message.Messages {
		if err := c.annotateMessage(nested, model); err != nil {
			return err
		}
	}
	for _, enum := range message.Enums {
		if err := c.annotateEnum(enum, model); err != nil {
			return err
		}
	}
	return nil
}
