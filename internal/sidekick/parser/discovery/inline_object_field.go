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

func maybeInlineObjectField(model *api.APIBuilder, parentBuilder *api.MessageBuilder, name string, input *schema) (*api.Field, error) {
	return makeInlineObjectField(model, parentBuilder, name, input, input.Description, true, false)
}

func makeInlineObjectField(model *api.APIBuilder, parentBuilder *api.MessageBuilder, name string, input *schema, description string, optional, repeated bool) (*api.Field, error) {
	if input.Type != "object" || input.Properties == nil {
		return nil, nil
	}
	parent := parentBuilder.Build()
	id := fmt.Sprintf("%s.%s", parent.ID(), name)
	documentation := fmt.Sprintf("The message type for the [%s][%s] field.", name, id[1:])
	mb := api.NewMessageBuilder().
		WithName(name).
		WithID(id).
		WithPackage(parent.Package()).
		WithDocumentation(documentation).
		WithParent(parent)
	if err := makeMessageFields(model, mb, input); err != nil {
		return nil, err
	}
	message := mb.Build()
	parentBuilder.AddMessage(message)
	model.AddMessage(message)

	field := api.NewFieldBuilder().
		WithName(name).
		WithJSONName(name).
		WithID(id).
		WithDocumentation(description).
		WithDeprecated(input.Deprecated).
		WithOptional(optional).
		WithRepeated(repeated).
		WithTypez(api.TypezMessage).
		WithTypezID(id).
		Build()
	return field, nil
}
