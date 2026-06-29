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

type fieldTypeNames struct {
	Full  string
	Base  string
	Key   string
	Value string
}

func (c *codec) fieldTypeName(field *api.Field) (*fieldTypeNames, error) {
	names, err := c.fieldTypeBase(field)
	if err != nil {
		return nil, err
	}
	if field.Optional {
		names.Full = fmt.Sprintf("%s?", names.Base)
		return names, nil
	}
	if field.Repeated {
		names.Full = fmt.Sprintf("[%s]", names.Base)
		return names, nil
	}
	names.Full = names.Base
	return names, nil
}

func (c *codec) fieldTypeBase(field *api.Field) (*fieldTypeNames, error) {
	switch field.Typez {
	case api.TypezMessage:
		m, err := lookupMessage(c.Model, field.TypezID)
		if err != nil {
			return nil, err
		}
		if m.IsMap {
			return c.mapFieldTypeNames(m)
		}
		base, err := c.messageTypeName(m)
		if err != nil {
			return nil, err
		}
		return &fieldTypeNames{Base: base}, nil
	case api.TypezEnum:
		e, err := lookupEnum(c.Model, field.TypezID)
		if err != nil {
			return nil, err
		}
		base, err := c.enumTypeName(e)
		if err != nil {
			return nil, err
		}
		return &fieldTypeNames{Base: base}, nil
	default:
		base, err := scalarFieldTypeName(field)
		if err != nil {
			return nil, err
		}
		return &fieldTypeNames{Base: base}, nil
	}
}

func (c *codec) mapFieldTypeNames(m *api.Message) (*fieldTypeNames, error) {
	keyType, valueType, err := c.mapFieldTypeComponents(m)
	if err != nil {
		return nil, err
	}
	base := fmt.Sprintf("[%s: %s]", keyType, valueType)
	return &fieldTypeNames{
		Base:  base,
		Key:   keyType,
		Value: valueType,
	}, nil
}

func (c *codec) mapFieldTypeComponents(m *api.Message) (string, string, error) {
	kv, err := decomposeMap(m)
	if err != nil {
		return "", "", err
	}
	key, err := c.fieldTypeBase(kv.Key)
	if err != nil {
		return "", "", err
	}
	value, err := c.fieldTypeBase(kv.Value)
	if err != nil {
		return "", "", err
	}
	return key.Base, value.Base, nil
}

func scalarFieldTypeName(field *api.Field) (string, error) {
	switch field.Typez {
	case api.TypezDouble:
		return "Swift.Double", nil
	case api.TypezFloat:
		return "Swift.Float", nil
	case api.TypezInt64:
		return "Swift.Int64", nil
	case api.TypezUint64:
		return "Swift.UInt64", nil
	case api.TypezInt32:
		return "Swift.Int32", nil
	case api.TypezFixed64:
		return "Swift.UInt64", nil
	case api.TypezFixed32:
		return "Swift.UInt32", nil
	case api.TypezBool:
		return "Swift.Bool", nil
	case api.TypezString:
		return "Swift.String", nil
	case api.TypezBytes:
		return "Foundation.Data", nil
	case api.TypezUint32:
		return "Swift.UInt32", nil
	case api.TypezSfixed32:
		return "Swift.Int32", nil
	case api.TypezSfixed64:
		return "Swift.Int64", nil
	case api.TypezSint32:
		return "Swift.Int32", nil
	case api.TypezSint64:
		return "Swift.Int64", nil
	default:
		return "", fmt.Errorf("unexpected Typez (%s) for scalar field %q", field.Typez.String(), field.ID)
	}
}

func (c *codec) messageTypeName(m *api.Message) (string, error) {
	name := pascalCase(m.Name)
	if m.ServicePlaceholder {
		name = pascalCase(m.Name + "Client")
	}
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

func (c *codec) fullyQualifiedMessageTypeName(m *api.Message) (string, error) {
	name := pascalCase(m.Name)
	if m.Parent == nil {
		if m.Package == "" {
			// there is no package, so return the bare type name
			return name, nil
		}
		if m.Package == c.Model.PackageName {
			// this is the current package
			return fmt.Sprintf("%s.%s", c.PackageName, name), nil
		}
		dep, ok := c.ApiPackages[m.Package]
		if !ok {
			return "", fmt.Errorf("package %q not found in ApiPackages", m.Package)
		}
		return fmt.Sprintf("%s.%s", dep.Name, name), nil
	}
	parent, err := c.fullyQualifiedMessageTypeName(m.Parent)
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
	return dep.Name, nil
}
