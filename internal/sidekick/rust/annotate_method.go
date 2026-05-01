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

package rust

import (
	"fmt"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

type methodAnnotation struct {
	Name                      string
	NameNoMangling            string
	BuilderName               string
	DocLines                  []string
	PathInfo                  *api.PathInfo
	Body                      string
	ServiceNameToPascal       string
	ServiceNameToCamel        string
	ServiceNameToSnake        string
	OperationInfo             *operationInfo
	SystemParameters          []systemParameter
	ReturnType                string
	HasVeneer                 bool
	Attributes                []string
	RoutingRequired           bool
	DetailedTracingAttributes bool
	InternalBuilders          bool
	HasResourceNameGeneration bool
	ResourceNameTemplateGrpc  string
	GrpcResourceNameArgs      []string
}

// HasGrpcResourceNameArgs returns true if the method has gRPC resource name arguments.
func (m *methodAnnotation) HasGrpcResourceNameArgs() bool {
	return len(m.GrpcResourceNameArgs) > 0
}

// HasBindings returns true if the method has path bindings.
func (m *methodAnnotation) HasBindings() bool {
	return m.PathInfo != nil && len(m.PathInfo.Bindings) > 0
}

// BuilderVisibility returns the visibility for client and request builders.
func (m *methodAnnotation) BuilderVisibility() string {
	if m.InternalBuilders {
		return "pub(crate)"
	}
	return "pub"
}

type pathInfoAnnotation struct {
	// Whether the request has a body or not
	HasBody bool

	// A list of possible request parameters
	//
	// This is only used for gRPC-based clients, where we must consider all
	// possible request parameters.
	//
	// https://google.aip.dev/client-libraries/4222
	//
	// Templates are ignored. We only care about the FieldName and FieldAccessor.
	UniqueParameters []*bindingSubstitution

	// Whether this is idempotent by default
	//
	// This is only used for gRPC-based clients.
	IsIdempotent string
}

type operationInfo struct {
	MetadataType     string
	ResponseType     string
	PackageNamespace string
}

// OnlyMetadataIsEmpty returns true if only the metadata is empty.
func (info *operationInfo) OnlyMetadataIsEmpty() bool {
	return info.MetadataType == "wkt::Empty" && info.ResponseType != "wkt::Empty"
}

// OnlyResponseIsEmpty returns true if only the response is empty.
func (info *operationInfo) OnlyResponseIsEmpty() bool {
	return info.MetadataType != "wkt::Empty" && info.ResponseType == "wkt::Empty"
}

// BothAreEmpty returns true if both the metadata and response are empty.
func (info *operationInfo) BothAreEmpty() bool {
	return info.MetadataType == "wkt::Empty" && info.ResponseType == "wkt::Empty"
}

// NoneAreEmpty returns true if neither the metadata nor the response are empty.
func (info *operationInfo) NoneAreEmpty() bool {
	return info.MetadataType != "wkt::Empty" && info.ResponseType != "wkt::Empty"
}

type discoveryLroAnnotations struct {
	MethodName            string
	ReturnType            string
	PollingPathParameters []discoveryLroPathParameter
}

type discoveryLroPathParameter struct {
	Name       string
	SetterName string
}

type routingVariantAnnotations struct {
	FieldAccessors   []string
	PrefixSegments   []string
	MatchingSegments []string
	SuffixSegments   []string
}

type bindingSubstitution struct {
	// Rust code to access the leaf field, given a `req`
	//
	// This field can be deeply nested. We need to capture code for the entire
	// chain. This accessor always returns an `Option<&T>`, even for fields
	// which are always present. This simplifies the mustache templates.
	//
	// The accessor should not
	// - copy any fields
	// - move any fields
	// - panic
	// - assume context i.e. use the try operator: `?`
	FieldAccessor string

	// The field name
	//
	// Nested fields are '.'-separated.
	//
	// e.g. "message_field.nested_field"
	FieldName string

	// The path template to match this substitution against
	//
	// e.g. ["projects", "*"]
	Template []string
}

// TemplateAsArray returns Rust code that yields an array of path segments.
//
// This array is supplied as an argument to `gaxi::path_parameter::try_match()`,
// and `gaxi::path_parameter::PathMismatchBuilder`.
//
// e.g.: `&[Segment::Literal("projects/"), Segment::SingleWildcard]`.
func (s *bindingSubstitution) TemplateAsArray() string {
	return "&[" + strings.Join(annotateSegments(s.Template), ", ") + "]"
}

// TemplateAsString returns the expected template, which can be used as a static string.
//
// e.g.: "projects/*".
func (s *bindingSubstitution) TemplateAsString() string {
	return strings.Join(s.Template, "/")
}

// VariableName returns the variable name to be used in the templates.
func (s *bindingSubstitution) VariableName() string {
	return fmt.Sprintf("var_%s", strings.ReplaceAll(s.FieldName, ".", "_"))
}

type pathBindingAnnotation struct {
	// The path format string for this binding
	//
	// e.g. "/v1/projects/{}/locations/{}"
	PathFmt string

	// The fields to be sent as query parameters for this binding
	QueryParams []*api.Field

	// The variables to be substituted into the path
	Substitutions []*bindingSubstitution

	// The codec is configured to generated detailed tracing attributes.
	DetailedTracingAttributes bool

	// Resource name generation fields, propagated from method scope.
	HasResourceNameGeneration bool
	ResourceNameTemplate      string
	ResourceNameArgs          []string
}

// HasResourceNameArgs returns true if the method has resource name arguments.
func (b *pathBindingAnnotation) HasResourceNameArgs() bool {
	return len(b.ResourceNameArgs) > 0
}

// QueryParamsCanFail returns true if we serialize certain query parameters, which can fail. The code we generate
// uses the try operator '?'. We need to run this code in a closure which
// returns a `Result<>`.
func (b *pathBindingAnnotation) QueryParamsCanFail() bool {
	for _, f := range b.QueryParams {
		if f.Typez == api.TypezMessage {
			return true
		}
	}
	return false
}

// HasVariablePath returns true if the path has a variable.
func (b *pathBindingAnnotation) HasVariablePath() bool {
	return len(b.Substitutions) != 0
}

// PathTemplate produces a path template suitable for instrumentation and logging.
// Variable parts are replaced with {field_name}.
func (b *pathBindingAnnotation) PathTemplate() string {
	if len(b.Substitutions) == 0 {
		return b.PathFmt
	}

	template := b.PathFmt
	for _, s := range b.Substitutions {
		// Construct the placeholder e.g., "{field_name}"
		placeholder := "{" + s.FieldName + "}"
		// Replace the first instance of "{}" with the field name placeholder
		template = strings.Replace(template, "{}", placeholder, 1)
	}
	return template
}

type sampleInfoAnnotation struct {
	// StringParameters is the set of parameters of type string that should be shown
	// on the sample method for any given RPC sample.
	StringParameters []string
	// FormatResourceName is true if the resource name format is known and shoulw be shown in the sample.
	FormatResourceName bool
}

// HasBindingSubstitutions returns true if the method has binding substitutions.
func (m *methodAnnotation) HasBindingSubstitutions() bool {
	for _, b := range m.PathInfo.Bindings {
		for _, s := range b.PathTemplate.Segments {
			if s.Variable != nil {
				return true
			}
		}
	}
	return false
}

func (c *codec) annotateMethod(m *api.Method) (*methodAnnotation, error) {
	if err := c.annotatePathInfo(m); err != nil {
		return nil, err
	}
	for _, routing := range m.Routing {
		for _, variant := range routing.Variants {
			fieldAccessors, err := c.annotateRoutingAccessors(variant, m)
			if err != nil {
				return nil, err
			}
			routingVariantAnnotations := &routingVariantAnnotations{
				FieldAccessors:   fieldAccessors,
				PrefixSegments:   annotateSegments(variant.Prefix.Segments),
				MatchingSegments: annotateSegments(variant.Matching.Segments),
				SuffixSegments:   annotateSegments(variant.Suffix.Segments),
			}
			variant.Codec = routingVariantAnnotations
		}
	}
	returnType, err := c.methodInOutTypeName(m.OutputTypeID, m.Model, m.Model.PackageName)
	if err != nil {
		return nil, err
	}
	if m.ReturnsEmpty {
		returnType = "()"
	}
	serviceName := c.ServiceName(m.Service)
	systemParameters := slices.Clone(c.systemParameters)
	if m.APIVersion != "" {
		systemParameters = append(systemParameters, systemParameter{
			Name:  "$apiVersion",
			Value: m.APIVersion,
		})
	}
	docLines, err := c.formatDocComments(m.Documentation, m.ID, m.Model, m.Service.Scopes())
	if err != nil {
		return nil, err
	}
	annotation := &methodAnnotation{
		Name:                      toSnake(m.Name),
		NameNoMangling:            toSnakeNoMangling(m.Name),
		BuilderName:               toPascal(m.Name),
		Body:                      bodyAccessor(m),
		DocLines:                  docLines,
		PathInfo:                  m.PathInfo,
		ServiceNameToPascal:       toPascal(serviceName),
		ServiceNameToCamel:        toCamel(serviceName),
		ServiceNameToSnake:        toSnake(serviceName),
		SystemParameters:          systemParameters,
		ReturnType:                returnType,
		HasVeneer:                 c.hasVeneer,
		RoutingRequired:           c.routingRequired,
		DetailedTracingAttributes: c.detailedTracingAttributes,
		InternalBuilders:          c.internalBuilders,
	}

	if err := c.annotateResourceNameGeneration(m, annotation); err != nil {
		return nil, err
	}
	if annotation.Name == "clone" {
		// Some methods look too similar to standard Rust traits. Clippy makes
		// a recommendation that is not applicable to generated code.
		annotation.Attributes = []string{"#[allow(clippy::should_implement_trait)]"}
	}
	if m.OperationInfo != nil {
		metadataType, err := c.methodInOutTypeName(m.OperationInfo.MetadataTypeID, m.Model, m.Model.PackageName)
		if err != nil {
			return nil, err
		}
		responseType, err := c.methodInOutTypeName(m.OperationInfo.ResponseTypeID, m.Model, m.Model.PackageName)
		if err != nil {
			return nil, err
		}
		m.OperationInfo.Codec = &operationInfo{
			MetadataType:     metadataType,
			ResponseType:     responseType,
			PackageNamespace: c.rootModuleName(m.Model),
		}
	}
	if m.DiscoveryLro != nil {
		lroAnnotation := &discoveryLroAnnotations{
			MethodName: annotation.Name,
			ReturnType: returnType,
		}
		for _, p := range m.DiscoveryLro.PollingPathParameters {
			a := discoveryLroPathParameter{
				Name:       toSnake(p),
				SetterName: toSnakeNoMangling(p),
			}
			lroAnnotation.PollingPathParameters = append(lroAnnotation.PollingPathParameters, a)
		}
		m.DiscoveryLro.Codec = lroAnnotation
	}
	m.Codec = annotation
	return annotation, nil
}

func (c *codec) annotateRoutingAccessors(variant *api.RoutingInfoVariant, m *api.Method) ([]string, error) {
	return makeAccessors(variant.FieldPath, m)
}

func makeAccessors(fields []string, m *api.Method) ([]string, error) {
	findField := func(name string, message *api.Message) *api.Field {
		for _, f := range message.Fields {
			if f.Name == name {
				return f
			}
		}
		return nil
	}
	var accessors []string
	message := m.InputType
	for _, name := range fields {
		field := findField(name, message)
		rustFieldName := toSnake(name)
		if field == nil {
			return nil, fmt.Errorf("invalid routing/path field (%q) for request message %s", rustFieldName, message.ID)
		}
		if field.Optional {
			accessors = append(accessors, fmt.Sprintf(".and_then(|m| m.%s.as_ref())", rustFieldName))
		} else {
			accessors = append(accessors, fmt.Sprintf(".map(|m| &m.%s)", rustFieldName))
		}
		if field.Typez == api.TypezString {
			accessors = append(accessors, ".map(|s| s.as_str())")
		}
		if field.Typez == api.TypezMessage {
			if fieldMessage := m.Model.Message(field.TypezID); fieldMessage != nil {
				message = fieldMessage
			}
		}
	}
	return accessors, nil
}

func annotateSegments(segments []string) []string {
	var ann []string
	// The model may have multiple consecutive literal segments. We use this
	// buffer to consolidate them into a single literal segment.
	literalBuffer := ""
	flushBuffer := func() {
		if literalBuffer != "" {
			ann = append(ann, fmt.Sprintf(`Segment::Literal("%s")`, literalBuffer))
		}
		literalBuffer = ""
	}
	for index, segment := range segments {
		switch segment {
		case api.MultiSegmentWildcard:
			flushBuffer()
			if len(segments) == 1 {
				ann = append(ann, "Segment::MultiWildcard")
			} else if len(segments) != index+1 {
				ann = append(ann, "Segment::MultiWildcard")
			} else {
				ann = append(ann, "Segment::TrailingMultiWildcard")
			}
		case api.SingleSegmentWildcard:
			if index != 0 {
				literalBuffer += "/"
			}
			flushBuffer()
			ann = append(ann, "Segment::SingleWildcard")
		default:
			if index != 0 {
				literalBuffer += "/"
			}
			literalBuffer += segment
		}
	}
	flushBuffer()
	return ann
}

func makeBindingSubstitution(v *api.PathVariable, m *api.Method) (*bindingSubstitution, error) {
	accessors, err := makeAccessors(v.FieldPath, m)
	if err != nil {
		return nil, err
	}
	fieldAccessor := "Some(&req)"
	for _, a := range accessors {
		fieldAccessor += a
	}
	var rustNames []string
	for _, n := range v.FieldPath {
		rustNames = append(rustNames, toSnakeNoMangling(n))
	}
	binding := &bindingSubstitution{
		FieldAccessor: fieldAccessor,
		FieldName:     strings.Join(rustNames, "."),
		Template:      v.Segments,
	}
	return binding, nil
}

func (c *codec) annotatePathBinding(b *api.PathBinding, m *api.Method) (*pathBindingAnnotation, error) {
	var subs []*bindingSubstitution
	for _, s := range b.PathTemplate.Segments {
		if s.Variable != nil {
			sub, err := makeBindingSubstitution(s.Variable, m)
			if err != nil {
				return nil, err
			}
			subs = append(subs, sub)
		}
	}
	binding := &pathBindingAnnotation{
		PathFmt:                   httpPathFmt(b.PathTemplate),
		QueryParams:               language.QueryParams(m, b),
		Substitutions:             subs,
		DetailedTracingAttributes: c.detailedTracingAttributes,
	}
	return binding, nil
}

func (c *codec) annotatePathInfo(m *api.Method) error {
	seen := make(map[string]bool)
	var uniqueParameters []*bindingSubstitution

	for _, b := range m.PathInfo.Bindings {
		ann, err := c.annotatePathBinding(b, m)
		if err != nil {
			return err
		}

		for _, s := range ann.Substitutions {
			if _, ok := seen[s.FieldName]; !ok {
				uniqueParameters = append(uniqueParameters, s)
				seen[s.FieldName] = true
			}
		}

		b.Codec = ann
	}

	m.PathInfo.Codec = &pathInfoAnnotation{
		HasBody:          m.PathInfo.BodyFieldPath != "",
		UniqueParameters: uniqueParameters,
		IsIdempotent:     isIdempotent(m.PathInfo),
	}
	return nil
}

func (c *codec) annotateSampleInfo(si *api.SampleInfo, m *api.Method) {
	var parameters []string
	var formatResourceName bool
	if rn := si.ResourceNameField; rn != nil {
		fieldAnn := rn.Codec.(*fieldAnnotations)
		if fieldAnn.FormattedResource != nil && len(fieldAnn.FormattedResource.FormatArgs) > 0 {
			parameters = fieldAnn.FormattedResource.FormatArgs
			formatResourceName = true
		} else {
			parameters = []string{fieldAnn.SetterName}
		}
	} else if m.IsAIPStandardUpdate {
		parameters = []string{"name"}
	} else {
		parameters = []string{}
	}

	ann := &sampleInfoAnnotation{
		StringParameters:   parameters,
		FormatResourceName: formatResourceName,
	}
	si.Codec = ann
}

func (c *codec) annotateResourceNameGeneration(m *api.Method, annotation *methodAnnotation) error {
	if !annotation.DetailedTracingAttributes {
		return nil
	}
	if m.PathInfo != nil {
		var firstBindingWithTargetResource *api.PathBinding
		for _, b := range m.PathInfo.Bindings {
			if b.TargetResource != nil {
				annotation.HasResourceNameGeneration = true
				firstBindingWithTargetResource = b
				break
			}
		}

		if annotation.HasResourceNameGeneration {
			tmpl, err := formatResourceNameTemplateFromPath(m, firstBindingWithTargetResource)
			if err != nil {
				return err
			}
			annotation.ResourceNameTemplateGrpc = tmpl

			var grpcArgs []string
			for _, path := range firstBindingWithTargetResource.TargetResource.FieldPaths {
				accessors, err := makeAccessors(path, m)
				if err != nil {
					return err
				}
				fieldAccessor := "Some(&req)"
				for _, a := range accessors {
					fieldAccessor += a
				}
				grpcArgs = append(grpcArgs, fieldAccessor)
			}
			annotation.GrpcResourceNameArgs = grpcArgs

			for _, b := range m.PathInfo.Bindings {
				bAnn, ok := b.Codec.(*pathBindingAnnotation)
				if !ok {
					continue
				}
				bAnn.HasResourceNameGeneration = true

				if b.TargetResource != nil {
					tmpl, err := formatResourceNameTemplateFromPath(m, b)
					if err != nil {
						return err
					}
					bAnn.ResourceNameTemplate = tmpl
					bAnn.ResourceNameArgs = formatResourceNameArgs(b.TargetResource.FieldPaths)
				} else {
					bAnn.ResourceNameTemplate = ""
					bAnn.ResourceNameArgs = nil
				}
			}
		}
	}
	return nil
}

func formatResourceNameTemplateFromPath(m *api.Method, b *api.PathBinding) (string, error) {
	if b.TargetResource == nil || len(b.TargetResource.Template) == 0 {
		return "", fmt.Errorf("missing target resource template for method %s", m.ID)
	}

	var sb strings.Builder
	for i, seg := range b.TargetResource.Template {
		if i > 0 {
			sb.WriteString("/")
		}
		if seg.Literal != nil {
			sb.WriteString(*seg.Literal)
		} else if seg.Variable != nil {
			sb.WriteString("{}")
		}
	}
	return sb.String(), nil
}

func formatResourceNameArgs(fieldPaths [][]string) []string {
	var args []string
	for _, path := range fieldPaths {
		var rustNames []string
		for _, p := range path {
			rustNames = append(rustNames, toSnakeNoMangling(p))
		}
		varName := fmt.Sprintf("var_%s", strings.Join(rustNames, "_"))
		args = append(args, varName)
	}
	return args
}

func isIdempotent(p *api.PathInfo) string {
	if len(p.Bindings) == 0 {
		return "false"
	}
	for _, b := range p.Bindings {
		if b.Verb == "POST" || b.Verb == "PATCH" {
			return "false"
		}
	}
	return "true"
}
