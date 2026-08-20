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
		if segment.Literal != "" {
			pathComponents = append(pathComponents, segment.Literal)
		} else if segment.Variable != nil {
			pathComponents = append(pathComponents, fmt.Sprintf(`\(pathVariable%d)`, count))
			count += 1
		}
	}
	path := "/" + strings.Join(pathComponents, "/")
	if t.Verb != "" {
		path += ":" + t.Verb
	}
	return path
}

func (c *codec) pathVariables(message *api.Message, t *api.PathTemplate) ([]*pathVariable, error) {
	count := 0
	var variables []*pathVariable
	for _, segment := range t.Segments {
		if segment.Variable != nil {
			pathVar, err := c.newPathVariable(message, segment.Variable, count)
			if err != nil {
				return nil, err
			}
			variables = append(variables, pathVar)
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
		expr, err := c.fieldPathParameterExpression(optional, field)
		if err != nil {
			return nil, err
		}
		expression.WriteString(expr)
		optional = field.Optional
		switch field.Typez {
		case api.TypezMessage:
			if !field.Optional {
				// Panics are the right way to deal with bugs in other parts of the code.
				panic(fmt.Sprintf("invalid state: field %s in message %s has message type but is not optional", field.Name, current.ID))
			}
			current, err = lookupMessage(c.Model, field.TypezID)
			if err != nil {
				return nil, err
			}
		case api.TypezString:
			test = fmt.Sprintf("!%s.isEmpty", name)
		case api.TypezBytes:
			return nil, fmt.Errorf("unsupported path parameter type %q, message=%q, path=%q", field.Typez.String(), message.ID, strings.Join(variable.FieldPath, "."))
		default:
			test = ""
		}
	}
	pathVar := &pathVariable{
		Name:       name,
		Expression: expression.String(),
		Test:       test,
		FieldPath:  strings.Join(variable.FieldPath, "."),
	}
	return pathVar, nil
}

func (*codec) fieldPathParameterExpression(optional bool, field *api.Field) (string, error) {
	if field.IsOneOf {
		return "", fmt.Errorf("unsupported path parameter: field %s", field.ID)
	}
	fieldCodec, ok := field.Codec.(*fieldAnnotations)
	if !ok {
		return "", fmt.Errorf("internal error: field %s does not have swift fieldAnnotations", field.ID)
	}
	if optional && field.Optional {
		return fmt.Sprintf(".flatMap({ $0.%s })", fieldCodec.Name), nil
	}
	if optional {
		return fmt.Sprintf(".map({ $0.%s })", fieldCodec.Name), nil
	}
	if field.Optional {
		return fmt.Sprintf(".%s", fieldCodec.Name), nil
	}
	return fmt.Sprintf(".%s as %s?", fieldCodec.Name, fieldCodec.FieldType), nil
}

// Describes a parameter used for gRPC x-goog-request-params routing metadata.
type routingParam struct {
	RoutingKey string
	Variants   []*routingParamVariant
}

// Represents a single candidate field path and pattern rule for extracting a routing
// parameter value.
type routingParamVariant struct {
	FieldAccessor    string
	PrefixSegments   []string
	MatchingSegments []string
	SuffixSegments   []string
	Last             bool
}

// Builds a Swift field access expression with optional chaining from a field path slice
// (e.g., ["table", "parent"] -> "request.table?.parent").
//
// Returns an empty string if fieldPath is empty; callers are expected to filter out
// empty field paths as AIP-4222 routing parameters must extract from a specific field.
func formatFieldAccessor(fieldPath []string) string {
	if len(fieldPath) == 0 {
		return ""
	}
	var parts []string
	for _, p := range fieldPath {
		parts = append(parts, camelCase(p))
	}
	return "request." + strings.Join(parts, "?.")
}

// Converts a list of path segments into Swift _RoutingMatcher token expressions
// (e.g. .literal("projects/"), .singleWildcard, .multiWildcard).
//
// isPrefix ensures a trailing slash delimiter is appended before the matching segment.
// isSuffix ensures leading/inter-segment slashes are preserved after the matching segment.
func annotateSegments(segments []string, isPrefix bool, isSuffix bool) []string {
	if len(segments) == 0 {
		return nil
	}
	var ann []string
	literalBuffer := ""
	flushBuffer := func() {
		if literalBuffer != "" {
			ann = append(ann, fmt.Sprintf(`.literal(%q)`, literalBuffer))
		}
		literalBuffer = ""
	}
	for index, segment := range segments {
		switch segment {
		case api.MultiSegmentWildcard:
			flushBuffer()
			if len(segments) == 1 && !isSuffix {
				ann = append(ann, ".multiWildcard")
			} else if len(segments) != index+1 {
				ann = append(ann, ".multiWildcard")
			} else {
				ann = append(ann, ".trailingMultiWildcard")
			}
		case api.SingleSegmentWildcard:
			if index != 0 || isSuffix {
				literalBuffer += "/"
			}
			flushBuffer()
			ann = append(ann, ".singleWildcard")
		default:
			if index != 0 || isSuffix {
				literalBuffer += "/"
			}
			literalBuffer += segment
		}
	}
	if isPrefix && len(segments) > 0 {
		literalBuffer += "/"
	}
	flushBuffer()
	return ann
}

// Extracts routing parameters from explicit google.api.routing annotations per AIP-4222.
func (c *codec) routingParamsFromRouting(routingInfos []*api.RoutingInfo) []*routingParam {
	var params []*routingParam
	for _, info := range routingInfos {
		if info.Name == "" && len(info.Variants) > 0 && len(info.Variants[0].FieldPath) == 0 {
			// Explicit empty routing annotation disables routing headers per AIP-4222.
			continue
		}
		routingKey := info.Name

		var variants []*routingParamVariant
		for _, variant := range info.Variants {
			if len(variant.FieldPath) == 0 {
				continue
			}
			if routingKey == "" {
				routingKey = strings.Join(variant.FieldPath, ".")
			}
			variants = append(variants, &routingParamVariant{
				FieldAccessor:    formatFieldAccessor(variant.FieldPath),
				PrefixSegments:   annotateSegments(variant.Prefix.Segments, true, false),
				MatchingSegments: annotateSegments(variant.Matching.Segments, false, false),
				SuffixSegments:   annotateSegments(variant.Suffix.Segments, false, true),
			})
		}

		if len(variants) > 0 {
			for i, v := range variants {
				v.Last = (i == len(variants)-1)
			}
			params = append(params, &routingParam{
				RoutingKey: routingKey,
				Variants:   variants,
			})
		}
	}
	return params
}

// Extracts fallback routing parameters from a google.api.http path template when explicit
// google.api.routing annotations are absent per AIP-4222.
func (c *codec) routingParamsFromPathTemplate(t *api.PathTemplate) []*routingParam {
	var params []*routingParam
	for _, segment := range t.Segments {
		if segment.Variable != nil {
			fieldPath := segment.Variable.FieldPath
			if len(fieldPath) == 0 {
				continue
			}
			params = append(params, &routingParam{
				RoutingKey: strings.Join(fieldPath, "."),
				Variants: []*routingParamVariant{
					{
						FieldAccessor:    formatFieldAccessor(fieldPath),
						PrefixSegments:   nil,
						MatchingSegments: []string{".multiWildcard"},
						SuffixSegments:   nil,
						Last:             true,
					},
				},
			})
		}
	}
	return params
}
