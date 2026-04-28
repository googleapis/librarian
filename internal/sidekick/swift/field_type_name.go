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

// fieldTypeName returns the Swift type name for a field.
//
// The implementation is pretty simple for primitive types. For message and enum fields it may get more
// difficult as the name may be in a separate package.
func (c *codec) fieldTypeName(field *api.Field) (string, error) {
	baseFieldType, err := c.baseFieldTypeName(field)
	if err != nil {
		return "", err
	}
	if field.Optional {
		return fmt.Sprintf("%s?", baseFieldType), nil
	}
	if field.Repeated {
		return fmt.Sprintf("[%s]", baseFieldType), nil
	}
	return baseFieldType, nil
}

// baseFieldTypeName returns the basic Swift type used for a field, excluding "optional" and "repeated" decorations.
func (c *codec) baseFieldTypeName(field *api.Field) (string, error) {
	switch field.Typez {
	case api.TypezMessage:
		m, err := lookupMessage(c.Model, field.TypezID)
		if err != nil {
			return "", err
		}
		if m.IsMap {
			return c.mapFieldTypeName(m)
		}
		return c.messageTypeName(m)
	case api.TypezEnum:
		e, err := lookupEnum(c.Model, field.TypezID)
		if err != nil {
			return "", err
		}
		return c.enumTypeName(e)
	default:
		return scalarFieldTypeName(field)
	}
}

func (c *codec) mapFieldTypeName(m *api.Message) (string, error) {
	var keyField, valueField *api.Field
	for _, f := range m.Fields {
		switch f.Name {
		case "key":
			keyField = f
		case "value":
			valueField = f
		}
	}
	if keyField == nil || valueField == nil {
		return "", fmt.Errorf("map message %q missing key or value field", m.ID)
	}
	keyType, err := c.baseFieldTypeName(keyField)
	if err != nil {
		return "", err
	}
	valueType, err := c.baseFieldTypeName(valueField)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("[%s: %s]", keyType, valueType), nil
}

func scalarFieldTypeName(field *api.Field) (string, error) {
	switch field.Typez {
	case api.TypezDouble:
		return "Double", nil
	case api.TypezFloat:
		return "Float", nil
	case api.TypezInt64:
		return "Int64", nil
	case api.TypezUint64:
		return "UInt64", nil
	case api.TypezInt32:
		return "Int32", nil
	case api.TypezFixed64:
		return "UInt64", nil
	case api.TypezFixed32:
		return "UInt32", nil
	case api.TypezBool:
		return "Bool", nil
	case api.TypezString:
		return "String", nil
	case api.TypezBytes:
		return "Data", nil
	case api.TypezUint32:
		return "UInt32", nil
	case api.TypezSfixed32:
		return "Int32", nil
	case api.TypezSfixed64:
		return "Int64", nil
	case api.TypezSint32:
		return "Int32", nil
	case api.TypezSint64:
		return "Int64", nil
	default:
		return "", fmt.Errorf("unexpected Typez (%s) for scalar field %q", field.Typez.String(), field.ID)
	}
}

func (c *codec) messageTypeName(m *api.Message) (string, error) {
	name := pascalCase(m.Name)
	if m.Parent == nil {
		prefix, err := c.externalTypePrefix(m.Package)
		if err != nil {
			return "", err
		}
		if prefix != "" {
			return fmt.Sprintf("%s.%s", prefix, name), nil
		}
		return name, nil
	}
	parent, err := c.messageTypeName(m.Parent)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s", parent, name), nil
}

func (c *codec) enumTypeName(e *api.Enum) (string, error) {
	name := pascalCase(e.Name)
	if e.Parent == nil {
		prefix, err := c.externalTypePrefix(e.Package)
		if err != nil {
			return "", err
		}
		if prefix != "" {
			return fmt.Sprintf("%s.%s", prefix, name), nil
		}
		return name, nil
	}
	parent, err := c.messageTypeName(e.Parent)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s", parent, name), nil
}

func (c *codec) externalTypePrefix(packageName string) (string, error) {
	if packageName == c.Model.PackageName {
		return "", nil
	}
	dep, ok := c.ApiPackages[packageName]
	if !ok {
		return "", fmt.Errorf("package %q not found in ApiPackages", packageName)
	}
	dep.Required = true
	return dep.Name, nil
}
