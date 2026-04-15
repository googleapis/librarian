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

	"github.com/googleapis/librarian/internal/sidekick/api"
)

type enumAnnotations struct {
	CopyrightYear   string
	BoilerPlate     []string
	Name            string
	DocLines        []string
	DefaultCaseName string
}

func (codec *codec) annotateEnum(enum *api.Enum, model *modelAnnotations) error {
	existing := map[int32]*enumValueAnnotations{}
	var defaultCaseName string
	for _, ev := range enum.UniqueNumberValues {
		codec.annotateUniqueEnumValue(ev)
		existing[ev.Number] = ev.Codec.(*enumValueAnnotations)
		if ev.Number == 0 {
			defaultCaseName = ev.Codec.(*enumValueAnnotations).CaseName
		}
	}
	// Fallback to first case if no 0 value found (should not happen in proto3)
	if defaultCaseName == "" {
		if len(enum.UniqueNumberValues) != 0 {
			defaultCaseName = enum.UniqueNumberValues[0].Codec.(*enumValueAnnotations).CaseName
		} else {
			return fmt.Errorf("cannot determine a default value for enum: %s", enum.ID)
		}
	}
	for _, ev := range enum.Values {
		if err := codec.annotateEnumValue(ev, existing); err != nil {
			return err
		}
		existing[ev.Number] = ev.Codec.(*enumValueAnnotations)
	}

	docLines := codec.formatDocumentation(enum.Documentation)
	annotations := &enumAnnotations{
		CopyrightYear:   model.CopyrightYear,
		BoilerPlate:     model.BoilerPlate,
		Name:            pascalCase(enum.Name),
		DocLines:        docLines,
		DefaultCaseName: defaultCaseName,
	}

	enum.Codec = annotations
	return nil
}
