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

package python

import (
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// messageAnnotations decorates api.Message with Python-specific metadata.
type messageAnnotations struct {
	Model    *modelAnnotations
	Message  *api.Message
	Name     string
	DocLines []string
	Fields   []*fieldAnnotations
	OneOfs   []*oneofAnnotations
}

func (c *codec) annotateMessage(message *api.Message, model *modelAnnotations) error {
	if message.Codec != nil {
		return nil
	}
	docLines := formatDocLines(message.Documentation)
	ann := &messageAnnotations{
		Model:    model,
		Message:  message,
		Name:     pascalCase(message.Name),
		DocLines: docLines,
	}

	for _, field := range message.Fields {
		if err := c.annotateField(field, ann); err != nil {
			return err
		}
		if fAnn, ok := field.Codec.(*fieldAnnotations); ok {
			ann.Fields = append(ann.Fields, fAnn)
		}
	}

	for _, oneOf := range message.OneOfs {
		if err := c.annotateOneOf(oneOf, ann); err != nil {
			return err
		}
		if oAnn, ok := oneOf.Codec.(*oneofAnnotations); ok {
			ann.OneOfs = append(ann.OneOfs, oAnn)
		}
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

	message.Codec = ann
	return nil
}
