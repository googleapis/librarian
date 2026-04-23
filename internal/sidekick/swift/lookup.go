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
	"slices"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

// lookupMessage finds a message in the model by its fully-qualified ID.
func lookupMessage(model *api.API, id string) (*api.Message, error) {
	m := model.Message(id)
	if m == nil {
		return nil, fmt.Errorf("unable to lookup message %q", id)
	}
	return m, nil
}

// lookupEnum finds an enum in the model by its fully-qualified ID.
func lookupEnum(model *api.API, id string) (*api.Enum, error) {
	e := model.Enum(id)
	if e == nil {
		return nil, fmt.Errorf("unable to lookup enum %q", id)
	}
	return e, nil
}

// lookupField finds a field in a message.
func lookupField(message *api.Message, name string) (*api.Field, error) {
	idx := slices.IndexFunc(message.Fields, func(f *api.Field) bool {
		return f.Name == name
	})
	if idx == -1 {
		return nil, fmt.Errorf("consistency error: field %s not found in message %q", name, message.ID)
	}
	return message.Fields[idx], nil
}
