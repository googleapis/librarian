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

func makeMessageEnum(model *api.API, message *api.Message, name string, schema *schema) error {
	if schema.Enums == nil {
		return nil
	}
	if len(schema.Enums) != len(schema.EnumDescriptions) {
		return fmt.Errorf("mismatched enum value list vs. enum value descriptions list")
	}
	id := fmt.Sprintf("%s.%s", message.ID, name)
	enum := &api.Enum{
		Name:          name,
		ID:            id,
		Package:       message.Package,
		Documentation: fmt.Sprintf("The enumerated type for the [%s][%s] field.", name, id[1:]),
		Parent:        message,
	}
	for number, name := range schema.Enums {
		value := &api.EnumValue{
			Name:          name,
			Number:        int32(number),
			ID:            fmt.Sprintf("%s.%s", enum.ID, name),
			Documentation: schema.EnumDescriptions[number],
			Parent:        enum,
		}
		enum.Values = append(enum.Values, value)
		enum.UniqueNumberValues = append(enum.UniqueNumberValues, value)
	}
	model.State.EnumByID[enum.ID] = enum
	message.Enums = append(message.Enums, enum)
	return nil
}
