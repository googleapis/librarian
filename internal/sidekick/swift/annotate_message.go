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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

type messageAnnotations struct {
	Name     string
	DocLines []string
	Model    *modelAnnotations
}

func (codec *codec) annotateMessage(message *api.Message, model *modelAnnotations) (*api.Message, error) {
	if dep, ok := codec.ApiPackages[message.Package]; ok {
		dep.Required = true
	}
	if message.Codec != nil {
		return message, nil
	}
	docLines := codec.formatDocumentation(message.Documentation)
	annotations := &messageAnnotations{
		Name:     pascalCase(message.Name),
		DocLines: docLines,
		Model:    model,
	}

	message.Codec = annotations
	for _, oneof := range message.OneOfs {
		_ = codec.annotateOneOf(oneof)
	}
	for _, field := range message.Fields {
		if _, err := codec.annotateField(field); err != nil {
			return nil, err
		}
	}
	for _, nested := range message.Messages {
		if _, err := codec.annotateMessage(nested, model); err != nil {
			return nil, err
		}
	}
	for _, enum := range message.Enums {
		if _, err := codec.annotateEnum(enum, model); err != nil {
			return nil, err
		}
	}
	return message, nil
}
