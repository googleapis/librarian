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

func pathExpression(t *api.PathTemplate) string {
	count := 0
	var pathComponents []string
	for _, segment := range t.Segments {
		if segment.Literal != nil {
			pathComponents = append(pathComponents, *segment.Literal)
		} else if segment.Variable != nil {
			pathComponents = append(pathComponents, fmt.Sprintf(`\(pathVariable%d)`, count))
			count += 1
		}
	}
	return "/" + strings.Join(pathComponents, "/")
}

func (c *codec) pathVariables(message *api.Message, t *api.PathTemplate) ([]*pathVariable, error) {
	count := 0
	var variables []*pathVariable
	for _, segment := range t.Segments {
		if segment.Variable != nil {
			new, err := c.newPathVariable(message, segment.Variable, count)
			if err != nil {
				return nil, err
			}
			variables = append(variables, new)
			count += 1
		}
	}
	return variables, nil
}

func (c *codec) newPathVariable(message *api.Message, variable *api.PathVariable, count int) (*pathVariable, error) {
	test := ""
	name := fmt.Sprintf("pathVariable%d", count)
	var expression strings.Builder
	optional := false
	current := message
	for _, v := range variable.FieldPath {
		field, err := lookupField(current, v)
		if err != nil {
			return nil, err
		}
		fieldCodec, ok := field.Codec.(*fieldAnnotations)
		if !ok {
			return nil, fmt.Errorf("internal error: field %s in message %s does not have swift fieldAnnotations", field.Name, current.ID)
		}
		if optional && field.Optional {
			fmt.Fprintf(&expression, ".flatMap({ $0.%s })", fieldCodec.Name)
		} else if optional {
			fmt.Fprintf(&expression, ".map({ $0.%s })", fieldCodec.Name)
		} else if field.Optional {
			fmt.Fprintf(&expression, ".%s", fieldCodec.Name)
		} else {
			fmt.Fprintf(&expression, ".%s as %s?", fieldCodec.Name, fieldCodec.FieldType)
		}
		optional = field.Optional
		switch field.Typez {
		case api.TypezMessage:
			current, err = lookupMessage(c.Model, field.TypezID)
			if err != nil {
				return nil, err
			}
		case api.TypezString:
			test = fmt.Sprintf("!%s.isEmpty", name)
		case api.TypezBytes:
			return nil, fmt.Errorf("unsupported path parameter type %q, message=%q, path=%q", field.Typez.String(), message.ID, variable.FieldPath)
		default:
			test = ""
		}
	}
	new := &pathVariable{
		Name:       name,
		Expression: expression.String(),
		Test:       test,
	}
	return new, nil
}
