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

// enumAnnotations decorates api.Enum with Python-specific metadata.
type enumAnnotations struct {
	Model             *modelAnnotations
	Enum              *api.Enum
	Name              string
	DocLines          []string
	FirstDocLine      string
	RemainingDocLines []string
	HasDocLines       bool
	HasMultiLineDoc   bool
	Values            []*enumValueAnnotations
	HasValues         bool
}

func (c *codec) annotateEnum(enum *api.Enum, model *modelAnnotations) error {
	docLines := formatMessageDocLines(enum.Documentation)
	var (
		firstLine      string
		remainingLines []string
	)
	if len(docLines) > 0 {
		firstLine = docLines[0]
		if len(docLines) > 1 {
			remainingLines = docLines[1:]
		}
	}
	ann := &enumAnnotations{
		Model:             model,
		Enum:              enum,
		Name:              pascalCase(enum.Name),
		DocLines:          docLines,
		FirstDocLine:      firstLine,
		RemainingDocLines: remainingLines,
		HasDocLines:       len(docLines) > 0,
		HasMultiLineDoc:   len(docLines) > 1,
	}

	for _, ev := range enum.Values {
		if err := c.annotateEnumValue(ev, ann); err != nil {
			return err
		}
		if evAnn, ok := ev.Codec.(*enumValueAnnotations); ok {
			ann.Values = append(ann.Values, evAnn)
		}
	}
	ann.HasValues = len(ann.Values) > 0

	enum.Codec = ann
	return nil
}
