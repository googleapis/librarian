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

// oneofAnnotations decorates api.OneOf with Python-specific metadata.
type oneofAnnotations struct {
	Message  *messageAnnotations
	OneOf    *api.OneOf
	Name     string
	DocLines []string
	Fields   []*fieldAnnotations
}

func (c *codec) annotateOneOf(oneOf *api.OneOf, message *messageAnnotations) error {
	docLines := formatDocLines(oneOf.Documentation)
	ann := &oneofAnnotations{
		Message:  message,
		OneOf:    oneOf,
		Name:     pythonIdentifier(snakeCase(oneOf.Name)),
		DocLines: docLines,
	}

	for _, field := range oneOf.Fields {
		if fAnn, ok := field.Codec.(*fieldAnnotations); ok {
			ann.Fields = append(ann.Fields, fAnn)
		}
	}

	oneOf.Codec = ann
	return nil
}
