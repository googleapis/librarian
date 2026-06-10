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

	// KeyType is the key's Swift type for maps and empty otherwise.
	KeyType string

	// ValueType is the value's Swift type for maps and empty otherwise.
	ValueType string

	// PackageName is the name of the package defining the type of this field.
	PackageName string

	// DocLines is the field documentation broken by lines with any filtering / corrections for Swift.
	DocLines []string

	// OneOfChecker is the name of a function to set the oneof containing this field.
	//
	// This is empty for fields that are not part of a oneof group.
	OneOfChecker string

	// Recursive is true if the field is a recursive reference to another message.
	Recursive bool
}

// IsStringKeyed returns true if the field is a map field and the key is a
// string type.
func (a *fieldAnnotations) IsStringKeyed() bool {
	return a.KeyType == "Swift.String"
}

func (c *codec) annotateField(field *api.Field) error {
	parts, err := c.fieldTypeName(field)
	if err != nil {
		return err
	}
	docLines, err := c.formatDocumentation(field.Documentation, field.Scopes())
	if err != nil {
		return err
	}
	packageName, err := c.fieldPackage(field)
	if err != nil {
		return err
	}

	annotations := &fieldAnnotations{
		Name:          camelCase(field.Name),
		FieldType:     parts.Full,
		BaseFieldType: parts.Base,
		KeyType:       parts.Key,
		ValueType:     parts.Value,
		PackageName:   packageName,
		DocLines:      docLines,
	}
	// Swift value types (structs) cannot contain recursive references directly because their
	// size must be known at compile time. To break the cycle, we wrap the reference in a box type
	// when the following conditions are met:
	// 1. field.Recursive: The field is part of a recursive reference cycle.
	// 2. field.Singular(): Repeated fields ([T]) and Maps ([K: V]) store their elements dynamically
	//    on the heap, so they do not cause compile-time infinite struct size issues.
	// 3. !field.IsOneOf: Oneof fields are nested inside a Swift enum which handles recursive boxing
	//    automatically using the native indirect case mechanism.
	if field.Recursive && field.Singular() && !field.IsOneOf {
		annotations.Recursive = true
		annotations.BaseFieldType = fmt.Sprintf("%s.Recursive<%s>", wellKnownSwiftPackage, parts.Base)
		annotations.FieldType = annotations.BaseFieldType + "?"
	}
	if field.IsOneOf && field.Group != nil {
		if oneofAnn, ok := field.Group.Codec.(*oneOfAnnotations); ok {
			annotations.OneOfChecker = oneofAnn.Checker
		}
	}
	field.Codec = annotations
	return nil
}

func (c *codec) fieldPackage(field *api.Field) (string, error) {
	switch field.Typez {
	case api.TypezMessage:
		m, err := lookupMessage(c.Model, field.TypezID)
		if err != nil {
			return "", err
		}
		if !m.IsMap {
			return m.Package, nil
		}
		fields, err := decomposeMap(m)
		if err != nil {
			return "", err
		}
		mf := fields.Value
		switch mf.Typez {
		case api.TypezMessage:
			vm, err := lookupMessage(c.Model, mf.TypezID)
			if err != nil {
				return "", err
			}
			return vm.Package, nil
		case api.TypezEnum:
			ve, err := lookupEnum(c.Model, mf.TypezID)
			if err != nil {
				return "", err
			}
			return ve.Package, nil
		}
	case api.TypezEnum:
		e, err := lookupEnum(c.Model, field.TypezID)
		if err != nil {
			return "", err
		}
		return e.Package, nil
	}
	return "", nil
}
