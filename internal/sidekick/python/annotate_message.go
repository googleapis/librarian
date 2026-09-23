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
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

// messageAnnotations decorates api.Message with Python-specific metadata.
type messageAnnotations struct {
	Model                            *modelAnnotations
	Message                          *api.Message
	Name                             string
	DocLines                         []string
	FirstDocLine                     string
	RemainingDocLines                []string
	HasDocLines                      bool
	HasMultiLineDoc                  bool
	Fields                           []*fieldAnnotations
	OneOfs                           []*oneofAnnotations
	NestedMessages                   []*messageAnnotations
	NestedEnums                      []*enumAnnotations
	HasNestedMessages                bool
	HasNestedEnums                   bool
	HasFields                        bool
	HasOneOfs                        bool
	HasOneOfNote                     bool
	HasOneOfLink                     bool
	HasRawPage                       bool
	HasExtendedOperationDoneProperty bool
	DoneStatusFieldName              string
}

func (c *codec) annotateMessage(message *api.Message, model *modelAnnotations) error {
	if message.Codec != nil {
		return nil
	}
	docLines := formatMessageDocLines(message.Documentation)
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
	ann := &messageAnnotations{
		Model:             model,
		Message:           message,
		Name:              pascalCase(message.Name),
		DocLines:          docLines,
		FirstDocLine:      firstLine,
		RemainingDocLines: remainingLines,
		HasDocLines:       len(docLines) > 0,
		HasMultiLineDoc:   len(docLines) > 1,
	}

	for _, field := range message.Fields {
		if err := c.annotateField(field, ann); err != nil {
			return err
		}
		if fAnn, ok := field.Codec.(*fieldAnnotations); ok {
			ann.Fields = append(ann.Fields, fAnn)
		}
	}

	for _, oneOf := range message.OneOfs {
		if err := c.annotateOneOf(oneOf, ann); err != nil {
			return err
		}
		if oAnn, ok := oneOf.Codec.(*oneofAnnotations); ok {
			ann.OneOfs = append(ann.OneOfs, oAnn)
		}
	}

	for _, nested := range message.Messages {
		if err := c.annotateMessage(nested, model); err != nil {
			return err
		}
		if nAnn, ok := nested.Codec.(*messageAnnotations); ok {
			ann.NestedMessages = append(ann.NestedMessages, nAnn)
		}
	}

	for _, enum := range message.Enums {
		if err := c.annotateEnum(enum, model); err != nil {
			return err
		}
		if eAnn, ok := enum.Codec.(*enumAnnotations); ok {
			ann.NestedEnums = append(ann.NestedEnums, eAnn)
		}
	}

	ann.HasNestedMessages = len(ann.NestedMessages) > 0
	ann.HasNestedEnums = len(ann.NestedEnums) > 0
	ann.HasFields = len(ann.Fields) > 0
	ann.HasOneOfs = len(ann.OneOfs) > 0

	for _, f := range ann.Fields {
		if f.Field.Name == "next_page_token" {
			ann.HasRawPage = true
		}
		if f.IsOneOf || f.DocIsOneOf {
			ann.HasOneOfLink = true
		}
	}

	for _, oneOf := range message.OneOfs {
		if len(oneOf.Fields) > 1 {
			ann.HasOneOfNote = true
			break
		}
	}
	if !ann.HasOneOfNote {
		for _, f := range message.Fields {
			if f.Group != nil && len(f.Group.Fields) > 1 {
				ann.HasOneOfNote = true
				break
			}
		}
	}

	if c.isComputeMessage(message) && ann.Name == "Operation" {
		for _, f := range ann.Fields {
			if f.Name == "status" {
				ann.HasExtendedOperationDoneProperty = true
				ann.DoneStatusFieldName = "status"
				break
			}
		}
	}

	message.Codec = ann
	return nil
}

func (c *codec) isComputeMessage(message *api.Message) bool {
	if c.Model != nil && strings.Contains(c.Model.PackageName, "compute") {
		return true
	}
	if c.Library != nil && strings.Contains(c.Library.Name, "compute") {
		return true
	}
	if message != nil && strings.Contains(message.Package, "compute") {
		return true
	}
	return false
}
