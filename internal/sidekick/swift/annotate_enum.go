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
	CopyrightYear     string
	BoilerPlate       []string
	Name              string
	DocLines          []string
	DefaultCaseName   string
	UnknownIntName    string
	UnknownStringName string
}

func (c *codec) annotateEnum(enum *api.Enum, model *modelAnnotations) error {
	// We need to find non-clashing names for the `unknownIntValue` and
	// `unknownStringvalue` cases. In practice, no enum uses those names, but
	// if one ever this, this will add enough trailing `_` to make the name
	// unique.
	type u struct{}
	caseNames := make(map[string]u)
	uniqueCaseName := func(seed string) string {
		_, ok := caseNames[seed]
		for ok {
			seed = seed + "_"
			_, ok = caseNames[seed]
		}
		return seed
	}

	existing := map[int32]*enumValueAnnotations{}
	var defaultCaseName string
	for _, ev := range enum.UniqueNumberValues {
		if err := c.annotateUniqueEnumValue(ev); err != nil {
			return err
		}
		ann := ev.Codec.(*enumValueAnnotations)
		if ann == nil {
			return fmt.Errorf("unknown annotation format for enum value: %s", ev.ID)
		}
		caseNames[ann.CaseName] = u{}
		existing[ev.Number] = ann
		if ev.Number == 0 {
			defaultCaseName = ann.CaseName
		}
	}
	// Fallback to first case if no 0 value found (should not happen in proto3)
	if defaultCaseName == "" {
		if len(enum.UniqueNumberValues) != 0 {
			ann := enum.UniqueNumberValues[0].Codec.(*enumValueAnnotations)
			if ann == nil {
				panic("mismatched annotation, previously checked, must be a bug")
			}
			defaultCaseName = ann.CaseName
		} else {
			return fmt.Errorf("cannot determine a default value for enum: %s", enum.ID)
		}
	}
	for _, ev := range enum.Values {
		if err := c.annotateEnumValue(ev, existing); err != nil {
			return err
		}
		existing[ev.Number] = ev.Codec.(*enumValueAnnotations)
	}

	docLines, err := c.formatDocumentation(enum.Documentation, enum.Scopes())
	if err != nil {
		return err
	}
	annotations := &enumAnnotations{
		CopyrightYear:     model.CopyrightYear,
		BoilerPlate:       model.BoilerPlate,
		Name:              pascalCase(enum.Name),
		DocLines:          docLines,
		DefaultCaseName:   defaultCaseName,
		UnknownIntName:    uniqueCaseName("unknownIntValue"),
		UnknownStringName: uniqueCaseName("unknownStringValue"),
	}

	enum.Codec = annotations
	return nil
}
