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
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

func (c *codec) linkDefinition(link string, scopes []string) (string, error) {
	for _, s := range scopes {
		localId := fmt.Sprintf(".%s.%s", s, link)
		result, err := c.tryDocLinkWithId(localId)
		if err != nil {
			return "", err
		}
		if result != "" {
			return result, nil
		}
	}
	localId := fmt.Sprintf(".%s", link)
	return c.tryDocLinkWithId(localId)
}

// docLink returns the documentation link for a symbol.
func (c *codec) docLink(packageName, name string) string {
	if packageName != c.Model.PackageName {
		// TODO(#5072) - we have not implemented cross-reference links to external packages.
		return fmt.Sprintf("https://www.google.com/search?q=Swift+%s+%s", packageName, name)
	}
	swiftyName := strings.ReplaceAll(name, ".", "/")
	return fmt.Sprintf("<doc:%s>", swiftyName)
}

func (c *codec) tryDocLinkWithId(id string) (string, error) {
	if m := c.Model.Message(id); m != nil {
		name, err := c.messageTypeName(m)
		if err != nil {
			return "", err
		}
		return c.docLink(m.Package, name), nil
	}
	if e := c.Model.Enum(id); e != nil {
		name, err := c.enumTypeName(e)
		if err != nil {
			return "", err
		}
		return c.docLink(e.Package, name), nil
	}
	if me := c.Model.Method(id); me != nil {
		return c.methodDocLink(me)
	}
	if s := c.Model.Service(id); s != nil {
		return c.serviceDocLink(s), nil
	}
	if rdLink, err := c.tryFieldDocLink(id); err != nil || rdLink != "" {
		return rdLink, err
	}
	if rdLink, err := c.tryEnumValueDocLink(id); err != nil || rdLink != "" {
		return rdLink, err
	}
	return "", nil
}

func (c *codec) tryFieldDocLink(id string) (string, error) {
	idx := strings.LastIndex(id, ".")
	if idx == -1 {
		return "", nil
	}
	messageId := id[0:idx]
	fieldName := id[idx+1:]
	m := c.Model.Message(messageId)
	if m == nil {
		return "", nil
	}
	for _, f := range m.Fields {
		if f.Name == fieldName {
			p, err := c.messageTypeName(m)
			if err != nil {
				return "", err
			}
			return c.docLink(m.Package, fmt.Sprintf("%s/%s", p, camelCase(f.Name))), nil
		}
	}
	for _, o := range m.OneOfs {
		if o.Name == fieldName {
			p, err := c.messageTypeName(m)
			if err != nil {
				return "", err
			}
			return c.docLink(m.Package, fmt.Sprintf("%s/%s", p, camelCase(o.Name))), nil
		}
	}
	return "", nil
}

func (c *codec) tryEnumValueDocLink(id string) (string, error) {
	idx := strings.LastIndex(id, ".")
	if idx == -1 {
		return "", nil
	}
	enumId := id[0:idx]
	valueName := id[idx+1:]
	e := c.Model.Enum(enumId)
	if e == nil {
		return "", nil
	}
	for _, v := range e.Values {
		if v.Name == valueName {
			p, err := c.enumTypeName(e)
			if err != nil {
				return "", err
			}
			return c.docLink(e.Package, fmt.Sprintf("%s/%s", p, enumValueCaseName(v))), nil
		}
	}
	return "", nil
}

func (c *codec) methodDocLink(m *api.Method) (string, error) {
	idx := strings.LastIndex(m.ID, ".")
	if idx == -1 {
		return "", nil
	}
	serviceId := m.ID[0:idx]
	s := c.Model.Service(serviceId)
	if s == nil {
		return "", nil
	}
	return c.docLink(s.Package, fmt.Sprintf("%s/%s(request:)", pascalCase(s.Name), camelCase(m.Name))), nil
}

func (c *codec) serviceDocLink(s *api.Service) string {
	return c.docLink(s.Package, pascalCase(s.Name))
}
