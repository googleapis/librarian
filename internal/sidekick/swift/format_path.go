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
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

func pathExpression(t *api.PathTemplate) string {
	count := 0
	var pathComponents []string
	for _, segment := range t.Segments() {
		if segment.Literal() != "" {
			pathComponents = append(pathComponents, segment.Literal())
		} else if segment.Variable() != nil {
			pathComponents = append(pathComponents, fmt.Sprintf(`\(pathVariable%d)`, count))
			count += 1
		}
	}
	path := "/" + strings.Join(pathComponents, "/")
	if t.Verb() != "" {
		path += ":" + t.Verb()
	}
	return path
}

func (c *codec) pathVariables(message *api.Message, t *api.PathTemplate) ([]*pathVariable, error) {
	count := 0
	var variables []*pathVariable
	for _, segment := range t.Segments() {
		if segment.Variable() != nil {
			pathVar, err := c.newPathVariable(message, segment.Variable(), count)
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
	var lastField *api.Field
	var jsonNames []string
	fieldPath := variable.FieldPath()
	for _, v := range fieldPath {
		field, err := lookupField(current, v)
		if err != nil {
			return nil, err
		}
		lastField = field
		jsonNames = append(jsonNames, jsonFieldName(field))
		expr, err := c.fieldPathParameterExpression(optional, field)
		if err != nil {
			return nil, err
		}
		expression.WriteString(expr)
		optional = field.Optional()
		switch field.Typez() {
		case api.TypezMessage:
			if !field.Optional() {
				// Panics are the right way to deal with bugs in other parts of the code.
				panic(fmt.Sprintf("invalid state: field %s in message %s has message type but is not optional", field.Name(), current.ID()))
			}
			current, err = lookupMessage(c.Model, field.TypezID())
			if err != nil {
				return nil, err
			}
		case api.TypezString:
			test = fmt.Sprintf("!%s.isEmpty", name)
		case api.TypezBytes:
			return nil, fmt.Errorf("unsupported path parameter type %q, message=%q, path=%q", field.Typez().String(), message.ID(), strings.Join(fieldPath, "."))
		default:
			test = ""
		}
	}
	var matchingSegments []string
	var templateString string
	if lastField != nil && lastField.Typez() == api.TypezString {
		segments := variable.Segments()
		matchingSegments = annotateSegments(segments, false, false)
		if len(segments) > 0 {
			templateString = strings.Join(segments, "/")
		}
	}
	pathVar := &pathVariable{
		Name:             name,
		Expression:       expression.String(),
		Test:             test,
		FieldPath:        strings.Join(fieldPath, "."),
		JSONFieldPath:    strings.Join(jsonNames, "."),
		MatchingSegments: matchingSegments,
		TemplateString:   templateString,
	}
	return pathVar, nil
}

// jsonFieldName returns the name of the field in the ProtoJSON encoding.
//
// The generated `CodingKeys` use this name. Some models (mostly in tests) leave `JSONName` unset,
// in that case the Swift property name is also the ProtoJSON name.
func jsonFieldName(field *api.Field) string {
	if field.JSONName() != "" {
		return field.JSONName()
	}
	return camelCase(field.Name())
}

func (*codec) fieldPathParameterExpression(optional bool, field *api.Field) (string, error) {
	if field.IsOneOf() {
		return "", fmt.Errorf("unsupported path parameter: field %s", field.ID())
	}
	fieldCodec, ok := field.Codec().(*fieldAnnotations)
	if !ok {
		return "", fmt.Errorf("internal error: field %s does not have swift fieldAnnotations", field.ID())
	}
	if optional && field.Optional() {
		return fmt.Sprintf(".flatMap({ $0.%s })", fieldCodec.Name), nil
	}
	if optional {
		return fmt.Sprintf(".map({ $0.%s })", fieldCodec.Name), nil
	}
	if field.Optional() {
		return fmt.Sprintf(".%s", fieldCodec.Name), nil
	}
	return fmt.Sprintf(".%s as %s?", fieldCodec.Name, fieldCodec.FieldType), nil
}

// Describes a parameter used for gRPC x-goog-request-params routing metadata.
type routingParam struct {
	RoutingKey string
	Variants   []*routingParamVariant
}

// routingFieldStep describes a single hop along a routing parameter field path
// starting from `request`.
type routingFieldStep struct {
	PropertyName  string
	ChainOptional bool
}

// Represents a single candidate field path and pattern rule for extracting a routing
// parameter value.
type routingParamVariant struct {
	FieldSteps       []*routingFieldStep
	ConvertToString  bool
	IsEnum           bool
	IsStringEnum     bool
	IsOptional       bool
	NeedsParens      bool
	RequiredPackages []string
	PrefixSegments   []string
	MatchingSegments []string
	SuffixSegments   []string
	Last             bool
}

func isScalarWrapperWKT(typezID string) bool {
	switch typezID {
	case api.WktStringValueID,
		api.WktInt64ValueID,
		api.WktUInt64ValueID,
		api.WktInt32ValueID,
		api.WktUInt32ValueID,
		api.WktFloatValueID,
		api.WktDoubleValueID,
		api.WktBoolValueID:
		return true
	default:
		return false
	}
}

// resolveRoutingFieldPath resolves a routing parameter field path against message
// and describes how to extract and convert the leaf value into a Swift String?.
func (c *codec) resolveRoutingFieldPath(message *api.Message, fieldPath []string) (*routingParamVariant, error) {
	if len(fieldPath) == 0 {
		return nil, nil
	}
	if message == nil {
		return nil, fmt.Errorf("missing input message for routing field path %q", strings.Join(fieldPath, "."))
	}
	var modelAnn *modelAnnotations
	if c.Model != nil {
		modelAnn, _ = c.Model.Codec.(*modelAnnotations)
	}
	current := message
	var steps []*routingFieldStep
	var requiredPackages []string
	addRequiredPackage := func(pkg string) {
		if pkg == "" || (c.Model != nil && pkg == c.Model.PackageName) {
			return
		}
		if !slices.Contains(requiredPackages, pkg) {
			requiredPackages = append(requiredPackages, pkg)
		}
	}
	isOptional := false
	tailOptional := false
	inOptionalChain := false
	convertToString := false
	isEnum := false
	isStringEnum := false

	for i, part := range fieldPath {
		field, err := lookupField(current, part)
		if err != nil {
			return nil, err
		}
		if !field.Singular() {
			return nil, fmt.Errorf("unsupported non-singular routing field %q in message %q", field.Name(), current.ID())
		}
		if field.IsOneOf() {
			return nil, fmt.Errorf("unsupported oneof routing field %q in message %q", field.Name(), current.ID())
		}
		if field.Typez() == api.TypezBytes || field.TypezID() == api.WktBytesValueID || field.TypezID() == api.WktNullValueID || isScalarWrapperWKT(field.TypezID()) {
			return nil, fmt.Errorf("unsupported routing parameter type %q, message=%q, path=%q", unsupportedRoutingTypeName(field), message.ID(), strings.Join(fieldPath, "."))
		}
		fieldAnn, err := c.ensureRoutingFieldAnnotated(field, modelAnn)
		if err != nil {
			return nil, err
		}
		if fieldAnn.Recursive {
			return nil, fmt.Errorf("unsupported recursive routing field %q in message %q", field.Name(), current.ID())
		}
		addRequiredPackage(fieldAnn.PackageName)

		chainOptional := tailOptional
		if chainOptional {
			inOptionalChain = true
		}
		steps = append(steps, &routingFieldStep{
			PropertyName:  fieldAnn.Name,
			ChainOptional: chainOptional,
		})
		tailOptional = field.Optional()
		isOptional = isOptional || field.Optional()

		if i < len(fieldPath)-1 {
			if field.Typez() != api.TypezMessage {
				return nil, fmt.Errorf("intermediate routing field %q in message %q is not a message", field.Name(), current.ID())
			}
			nextMsg, err := c.lookupRoutingNextMessage(field)
			if err != nil {
				return nil, err
			}
			current = nextMsg
			continue
		}

		convertToString, isEnum, isStringEnum, err = c.classifyRoutingLeaf(field, message, fieldPath)
		if err != nil {
			return nil, err
		}
	}

	return &routingParamVariant{
		FieldSteps:       steps,
		ConvertToString:  convertToString,
		IsEnum:           isEnum,
		IsStringEnum:     isStringEnum,
		IsOptional:       isOptional,
		NeedsParens:      inOptionalChain,
		RequiredPackages: requiredPackages,
	}, nil
}

// ensureRoutingFieldAnnotated ensures only the traversed field has Swift
// annotations, without annotating unrelated sibling fields on external
// intermediate messages.
func (c *codec) ensureRoutingFieldAnnotated(field *api.Field, modelAnn *modelAnnotations) (*fieldAnnotations, error) {
	fieldAnn, ok := field.Codec().(*fieldAnnotations)
	if !ok {
		if c.Model == nil {
			return nil, fmt.Errorf("internal error: field %s does not have swift fieldAnnotations", field.ID())
		}
		var err error
		fieldAnn, err = c.annotateField(field, modelAnn)
		if err != nil {
			return nil, err
		}
	}
	return fieldAnn, nil
}

func (c *codec) lookupRoutingNextMessage(field *api.Field) (*api.Message, error) {
	if field.MessageType() != nil {
		return field.MessageType(), nil
	}
	if c.Model != nil {
		return lookupMessage(c.Model, field.TypezID())
	}
	return nil, fmt.Errorf("unable to lookup message %q", field.TypezID())
}

func (c *codec) classifyRoutingLeaf(field *api.Field, message *api.Message, fieldPath []string) (convertToString, isEnum, isStringEnum bool, err error) {
	switch field.Typez() {
	case api.TypezString:
		return false, false, false, nil
	case api.TypezEnum:
		return false, true, c.UseStringEnums, nil
	case api.TypezInt32, api.TypezInt64, api.TypezUint32, api.TypezUint64,
		api.TypezSint32, api.TypezSint64, api.TypezFixed32, api.TypezFixed64,
		api.TypezSfixed32, api.TypezSfixed64, api.TypezFloat, api.TypezDouble, api.TypezBool:
		return true, false, false, nil
	default:
		return false, false, false, fmt.Errorf("unsupported routing parameter type %q, message=%q, path=%q", unsupportedRoutingTypeName(field), message.ID(), strings.Join(fieldPath, "."))
	}
}

func unsupportedRoutingTypeName(field *api.Field) string {
	if field.TypezID() == api.WktBytesValueID || field.TypezID() == api.WktNullValueID || isScalarWrapperWKT(field.TypezID()) {
		return field.TypezID()
	}
	return field.Typez().String()
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
func (c *codec) routingParamsFromRouting(message *api.Message, routingInfos []*api.RoutingInfo) ([]*routingParam, error) {
	var params []*routingParam
	for _, info := range routingInfos {
		routingKey := info.Name()

		var variants []*routingParamVariant
		for _, variant := range info.Variants() {
			fieldPath := variant.FieldPath()
			if len(fieldPath) == 0 {
				continue
			}
			if routingKey == "" {
				routingKey = strings.Join(fieldPath, ".")
			}
			v, err := c.resolveRoutingFieldPath(message, fieldPath)
			if err != nil {
				return nil, err
			}
			v.PrefixSegments = annotateSegments(variant.Prefix().Segments(), true, false)
			v.MatchingSegments = annotateSegments(variant.Matching().Segments(), false, false)
			v.SuffixSegments = annotateSegments(variant.Suffix().Segments(), false, true)
			variants = append(variants, v)
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
	return params, nil
}

// Extracts fallback routing parameters from a google.api.http path template when explicit
// google.api.routing annotations are absent per AIP-4222.
func (c *codec) routingParamsFromPathTemplate(message *api.Message, t *api.PathTemplate) ([]*routingParam, error) {
	if t == nil {
		return nil, nil
	}
	var params []*routingParam
	for _, segment := range t.Segments() {
		if segment.Variable() != nil {
			fieldPath := segment.Variable().FieldPath()
			if len(fieldPath) == 0 {
				continue
			}
			v, err := c.resolveRoutingFieldPath(message, fieldPath)
			if err != nil {
				return nil, err
			}
			v.MatchingSegments = []string{".multiWildcard"}
			v.Last = true
			params = append(params, &routingParam{
				RoutingKey: strings.Join(fieldPath, "."),
				Variants:   []*routingParamVariant{v},
			})
		}
	}
	return params, nil
}

// routingParamsFromBindings extracts fallback routing parameters across all
// google.api.http bindings in declaration order, deduplicating repeated routing
// keys per AIP-4222.
func (c *codec) routingParamsFromBindings(message *api.Message, bindings []*api.PathBinding) ([]*routingParam, error) {
	var params []*routingParam
	seen := make(map[string]bool)
	for _, binding := range bindings {
		if binding == nil {
			continue
		}
		bindingParams, err := c.routingParamsFromPathTemplate(message, binding.PathTemplate())
		if err != nil {
			return nil, err
		}
		for _, param := range bindingParams {
			if seen[param.RoutingKey] {
				continue
			}
			seen[param.RoutingKey] = true
			params = append(params, param)
		}
	}
	return params, nil
}
