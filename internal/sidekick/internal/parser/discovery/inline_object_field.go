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

func maybeInlineObjectField(model *api.API, parent *api.Message, name string, input *schema) (*api.Field, error) {
	if input.Type != "object" || input.Properties == nil {
		return nil, nil
	}
	id := fmt.Sprintf("%s.%s", parent.ID, name)
	documentation := fmt.Sprintf("The message type for the [%s][%s] field.", name, id[1:])
	message := &api.Message{
		Name:          name,
		ID:            id,
		Package:       parent.Package,
		Documentation: documentation,
		Parent:        parent,
	}
	if err := makeMessageFields(model, message, input); err != nil {
		return nil, err
	}
	parent.Messages = append(parent.Messages, message)
	model.State.MessageByID[id] = message

	field := &api.Field{
		Name:          name,
		JSONName:      name,
		ID:            id,
		Documentation: input.Description,
		Deprecated:    input.Deprecated,
		Optional:      true,
		Typez:         api.MESSAGE_TYPE,
		TypezID:       id,
	}
	return field, nil
}
