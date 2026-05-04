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

package rust

import (
	"github.com/googleapis/librarian/internal/sidekick/api"
)

type enumValueAnnotation struct {
	Name              string
	VariantName       string
	EnumType          string
	DocLines          []string
	SerializeAsString bool
}

func (c *codec) annotateEnumValue(ev *api.EnumValue, model *api.API, full bool) error {
	annotations := &enumValueAnnotation{
		Name:              enumValueName(ev),
		EnumType:          enumName(ev.Parent),
		VariantName:       enumValueVariantName(ev),
		SerializeAsString: c.serializeEnumsAsStrings,
	}
	ev.Codec = annotations

	if !full {
		// We have basic annotations, we are done.
		return nil
	}
	lines, err := c.formatDocComments(ev.Documentation, ev.ID, model, ev.Scopes())
	if err != nil {
		return err
	}
	annotations.DocLines = lines
	return nil
}
