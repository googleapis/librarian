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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

type fieldAnnotations struct {
	// Name is the name of the field in the generated `struct`.
	//
	// The naming convention in Swift is to use camelCase, same as OpenAPI and discovery doc. However, most of the
	// Google Cloud services use Protobuf where the convention is `snake_case`.
	Name string

	// FieldType is name type of the field in the generated `struct`.
	//
	// This includes the optional (`T?`), repeated (`[T]`), and map (`[K: V]`) decorators.
	FieldType string

	// BaseFieldType is `FieldType` without optional/repeated decorations.
	//
	// This is used in the mustache templates, which sometimes need to refer to the underlying type.
	BaseFieldType string

	// DocLines is the field documentation broken by lines with any filtering / corrections for Swift.
	DocLines []string

	// OneOfPropertyName is the name of the oneof property containing this field.
	//
	// This is empty for fields that are not part of a oneof group.
	OneOfPropertyName string
}

func (c *codec) annotateField(field *api.Field) error {
	fieldType, err := c.fieldTypeName(field)
	if err != nil {
		return err
	}
	baseFieldType, err := c.baseFieldTypeName(field)
	if err != nil {
		return err
	}
	annotations := &fieldAnnotations{
		Name:          camelCase(field.Name),
		FieldType:     fieldType,
		BaseFieldType: baseFieldType,
		DocLines:      c.formatDocumentation(field.Documentation),
	}
	if field.IsOneOf && field.Group != nil {
		if oneofAnn, ok := field.Group.Codec.(*oneOfAnnotations); ok {
			annotations.OneOfPropertyName = oneofAnn.PropertyName
		}
	}
	field.Codec = annotations
	return nil
}
