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

// fieldAnnotations decorates api.Field with Python-specific metadata.
type fieldAnnotations struct {
	Message    *messageAnnotations
	Field      *api.Field
	Name       string
	DocLines   []string
	TypeName   string
	IsRepeated bool
	IsMap      bool
}

func (c *codec) annotateField(field *api.Field, message *messageAnnotations) error {
	docLines := formatDocLines(field.Documentation)
	ann := &fieldAnnotations{
		Message:    message,
		Field:      field,
		Name:       pythonIdentifier(snakeCase(field.Name)),
		DocLines:   docLines,
		TypeName:   field.TypezID,
		IsRepeated: field.Repeated,
		IsMap:      field.Map,
	}
	field.Codec = ann
	return nil
}
