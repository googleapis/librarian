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

func makeMessageEnum(model *api.APIBuilder, mb *api.MessageBuilder, name string, schema *schema) error {
	if schema.Enums == nil {
		return nil
	}
	if len(schema.Enums) != len(schema.EnumDescriptions) {
		return fmt.Errorf("mismatched enum value list vs. enum value descriptions list")
	}
	if len(schema.EnumDeprecated) != 0 && len(schema.Enums) != len(schema.EnumDeprecated) {
		// The list of deprecated enums is omitted in some cases.
		return fmt.Errorf("mismatched enum value list vs. enum deprecated values list")
	}
	message := mb.Build()
	id := fmt.Sprintf("%s.%s", message.ID(), name)
	var values []*api.EnumValue
	for number, name := range schema.Enums {
		deprecated := false
		if len(schema.EnumDeprecated) != 0 {
			deprecated = schema.EnumDeprecated[number]
		}
		value := api.NewEnumValueBuilder().
			WithName(name).
			WithNumber(int32(number)).
			WithID(fmt.Sprintf("%s.%s", id, name)).
			WithDocumentation(schema.EnumDescriptions[number]).
			WithDeprecated(deprecated).
			Build()
		values = append(values, value)
	}
	enum := api.NewEnumBuilder().
		WithName(name).
		WithID(id).
		WithPackage(message.Package()).
		WithDocumentation(fmt.Sprintf("The enumerated type for the [%s][%s] field.", name, id[1:])).
		WithDeprecated(schema.Deprecated).
		WithParent(message).
		WithValues(values...).
		WithUniqueNumberValues(values...).
		Build()
	model.AddEnum(enum)
	mb.AddEnum(enum)
	return nil
}
