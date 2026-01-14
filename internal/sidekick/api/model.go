// Copyright 2024 Google LLC
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

// Package api defines the data model representing a parsed API surface.
package api

import (
	"fmt"
	"slices"
	"strings"
)

// Typez represent different field types that may be found in messages.
type Typez int

const (
	// These are the different field types as defined in
	// descriptorpb.FieldDescriptorProto_Type.

	UNDEFINED_TYPE Typez = iota // 0
	DOUBLE_TYPE                 // 1
	FLOAT_TYPE                  // 2
	INT64_TYPE                  // 3
	UINT64_TYPE                 // 4
	INT32_TYPE                  // 5
	FIXED64_TYPE                // 6
	FIXED32_TYPE                // 7
	BOOL_TYPE                   // 8
	STRING_TYPE                 // 9
	GROUP_TYPE                  // 10
	MESSAGE_TYPE                // 11
	BYTES_TYPE                  // 12
	UINT32_TYPE                 // 13
	ENUM_TYPE                   // 14
	SFIXED32_TYPE               // 15
	SFIXED64_TYPE               // 16
	SINT32_TYPE                 // 17
	SINT64_TYPE                 // 18
)

// FieldBehavior represents annotations for how the code generator handles a
// field.
//
// Regardless of the underlying data type and whether it is required or optional
// on the wire, some fields must be present for requests to succeed. Or may not
// be included in a request.
type FieldBehavior int

const (
	// FIELD_BEHAVIOR_UNSPECIFIED is the default, unspecified field behavior.
	FIELD_BEHAVIOR_UNSPECIFIED FieldBehavior = iota

	// FIELD_BEHAVIOR_OPTIONAL specifically denotes a field as optional.
	//
	// While Google Cloud uses proto3, where fields are either optional or have
	// a default value, this may be specified for emphasis.
	FIELD_BEHAVIOR_OPTIONAL

	// FIELD_BEHAVIOR_REQUIRED denotes a field as required.
	//
	// This indicates that the field **must** be provided as part of the request,
	// and failure to do so will cause an error (usually `INVALID_ARGUMENT`).
	//
	// Code generators may change the generated types to include this field as a
	// parameter necessary to construct the request.
	FIELD_BEHAVIOR_REQUIRED

	// FIELD_BEHAVIOR_OUTPUT_ONLY denotes a field as output only.
	//
	// Some messages (and their fields) are used in both requests and responses.
	// This indicates that the field is provided in responses, but including the
	// field in a request does nothing (the server *must* ignore it and
	// *must not* throw an error as a result of the field's presence).
	//
	// Code generators that use different builders for "the message as part of a
	// request" vs. "the standalone message" may omit this field in the former.
	FIELD_BEHAVIOR_OUTPUT_ONLY

	// FIELD_BEHAVIOR_INPUT_ONLY denotes a field as input only.
	//
	// This indicates that the field is provided in requests, and the
	// corresponding field is not included in output.
	FIELD_BEHAVIOR_INPUT_ONLY

	// FIELD_BEHAVIOR_IMMUTABLE denotes a field as immutable.
	//
	// This indicates that the field may be set once in a request to create a
	// resource, but may not be changed thereafter.
	FIELD_BEHAVIOR_IMMUTABLE

	// FIELD_BEHAVIOR_UNORDERED_LIST denotes that a (repeated) field is an unordered list.
	//
	// This indicates that the service may provide the elements of the list
	// in any arbitrary  order, rather than the order the user originally
	// provided. Additionally, the list's order may or may not be stable.
	FIELD_BEHAVIOR_UNORDERED_LIST

	// FIELD_BEHAVIOR_UNORDERED_NON_EMPTY_DEFAULT denotes that this field returns a non-empty default value if not set.
	//
	// This indicates that if the user provides the empty value in a request,
	// a non-empty value will be returned. The user will not be aware of what
	// non-empty value to expect.
	FIELD_BEHAVIOR_UNORDERED_NON_EMPTY_DEFAULT

	// FIELD_BEHAVIOR_IDENTIFIER denotes that the field in a resource (a message annotated with
	// google.api.resource) is used in the resource name to uniquely identify the
	// resource.
	//
	// For AIP-compliant APIs, this should only be applied to the
	// `name` field on the resource.
	//
	// This behavior should not be applied to references to other resources within
	// the message.
	//
	// The identifier field of resources often have different field behavior
	// depending on the request it is embedded in (e.g. for Create methods name
	// is optional and unused, while for Update methods it is required). Instead
	// of method-specific annotations, only `IDENTIFIER` is required.
	FIELD_BEHAVIOR_IDENTIFIER
)

// API represents and API surface.
type API struct {
	// Name of the API (e.g. secretmanager).
	Name string
	// Name of the package name in the source specification format. For Protobuf
	// this may be `google.cloud.secretmanager.v1`.
	PackageName string
	// The API Title (e.g. "Secret Manager API" or "Cloud Spanner API").
	Title string
	// The API Description.
	Description string
	// The API Revision. In discovery-based services this is the "revision"
	// attribute.
	Revision string
	// Services are a collection of services that make up the API.
	Services []*Service
	// Messages are a collection of messages used to process request and
	// responses in the API.
	Messages []*Message
	// Enums
	Enums []*Enum
	// State contains helpful information that can be used when generating
	// clients.
	State *APIState
	// ResourceDefinitions contains the data from the `google.api.resource_definition` annotation.
	ResourceDefinitions []*Resource
	// Language specific annotations.
	Codec any
}

// HasMessages returns true if the API contains messages (most do).
//
// This is useful in the mustache templates to skip code that only makes sense
// when per-message code follows.
func (api *API) HasMessages() bool {
	return len(api.Messages) != 0
}

// ModelCodec returns the Codec field with an alternative name.
//
// In some mustache templates we want to access the annotations for the
// enclosing model. In mustache you can get a field from an enclosing context
// *if* the name is unique.
func (a *API) ModelCodec() any {
	return a.Codec
}

// APIState contains helpful information that can be used when generating
// clients.
type APIState struct {
	// ServiceByID returns a service that is associated with the API.
	ServiceByID map[string]*Service
	// MethodByID returns a method that is associated with the API.
	MethodByID map[string]*Method
	// MessageByID returns a message that is associated with the API.
	MessageByID map[string]*Message
	// EnumByID returns a message that is associated with the API.
	EnumByID map[string]*Enum
	// ResourceByType returns a resource that is associated with the API.
	ResourceByType map[string]*Resource
}

// Service represents a service in an API.
type Service struct {
	// Documentation for the service.
	Documentation string
	// Name of the attribute.
	Name string
	// ID is a unique identifier.
	ID string
	// Some source specifications allow marking services as deprecated.
	Deprecated bool
	// Methods associated with the Service.
	Methods []*Method
	// DefaultHost fragment of a URL.
	DefaultHost string
	// The Protobuf package this service belongs to.
	Package string

	// The model this service belongs to, mustache templates use this field to
	// navigate the data structure.
	Model *API
	// Language specific annotations.
	Codec any
}

// Method defines a RPC belonging to a Service.
type Method struct {
	// Documentation is the documentation for the method.
	Documentation string
	// Name is the name of the attribute.
	Name string
	// ID is a unique identifier.
	ID string
	// Deprecated is true if the method is deprecated.
	Deprecated bool
	// InputTypeID is the ID of the input type for the Method.
	InputTypeID string
	// InputType is the input to the Method.
	InputType *Message
	// OutputTypeID is the ID of the output type for the Method.
	OutputTypeID string
	// OutputType is the output of the Method.
	OutputType *Message
	// ReturnsEmpty is true if the method returns nothing.
	//
	// Protobuf uses the well-known type `google.protobuf.Empty` message to
	// represent this.
	//
	// OpenAPIv3 uses a missing content field:
	//   https://swagger.io/docs/specification/v3_0/describing-responses/#empty-response-body
	ReturnsEmpty bool
	// PathInfo contains information about the HTTP request.
	PathInfo *PathInfo
	// Pagination holds the `page_token` field if the method conforms to the
	// standard defined by [AIP-4233](https://google.aip.dev/client-libraries/4233).
	Pagination *Field
	// ClientSideStreaming is true if the method supports client-side streaming.
	ClientSideStreaming bool
	// ServerSideStreaming is true if the method supports server-side streaming.
	ServerSideStreaming bool
	// OperationInfo contains information for methods returning long-running operations.
	OperationInfo *OperationInfo
	// DiscoveryLro has a value if this is a discovery-style long-running operation.
	DiscoveryLro *DiscoveryLro
	// Routing contains the routing annotations, if any.
	Routing []*RoutingInfo
	// AutoPopulated contains the auto-populated (request_id) field, if any, as defined in
	// [AIP-4235](https://google.aip.dev/client-libraries/4235)
	//
	// The field must be eligible for auto-population, and be listed in the
	// `google.api.MethodSettings.auto_populated_fields` entry in
	// `google.api.Publishing.method_settings` in the service config file.
	AutoPopulated []*Field
	// Model is the model this method belongs to, mustache templates use this field to
	// navigate the data structure.
	Model *API
	// Service is the service this method belongs to, mustache templates use this field to
	// navigate the data structure.
	Service *Service
	// `SourceService` is the original service this method belongs to. For most
	// methods `SourceService` and `Service` are the same. For mixins, the
	// source service is the mixin, such as longrunning.Operations.
	SourceService *Service
	// `SourceServiceID` is the original service ID for this method.
	SourceServiceID string
	// Codec contains language specific annotations.
	Codec any
}

// RoutingCombos returns all combinations of routing parameters.
//
// The routing info is stored as a map from the key to a list of the variants.
// e.g.:
//
// ```
//
//	{
//	  a: [va1, va2, va3],
//	  b: [vb1, vb2]
//	  c: [vc1]
//	}
//
// ```
//
// We reorganize each kv pair into a list of pairs. e.g.:
//
// ```
// [
//
//	[(a, va1), (a, va2), (a, va3)],
//	[(b, vb1), (b, vb2)],
//	[(c, vc1)],
//
// ]
// ```
//
// Then we take a Cartesian product of that list to find all the combinations.
// e.g.:
//
// ```
// [
//
//	[(a, va1), (b, vb1), (c, vc1)],
//	[(a, va1), (b, vb2), (c, vc1)],
//	[(a, va2), (b, vb1), (c, vc1)],
//	[(a, va2), (b, vb2), (c, vc1)],
//	[(a, va3), (b, vb1), (c, vc1)],
//	[(a, va3), (b, vb2), (c, vc1)],
//
// ]
// ```.
func (m *Method) RoutingCombos() []*RoutingInfoCombo {
	combos := []*RoutingInfoCombo{
		{},
	}
	for _, info := range m.Routing {
		next := []*RoutingInfoCombo{}
		for _, c := range combos {
			for _, v := range info.Variants {
				next = append(next, &RoutingInfoCombo{
					Items: append(c.Items, &RoutingInfoComboItem{
						Name:    info.Name,
						Variant: v,
					}),
				})
			}
		}
		combos = next
	}
	return combos
}

// RoutingInfoCombo represents a single combination of routing parameters.
type RoutingInfoCombo struct {
	Items []*RoutingInfoComboItem
}

// RoutingInfoComboItem represents a single item in a RoutingInfoCombo.
type RoutingInfoComboItem struct {
	Name    string
	Variant *RoutingInfoVariant
}

// HasRouting returns true if the method has routing information.
func (m *Method) HasRouting() bool {
	return len(m.Routing) != 0
}

// HasAutoPopulatedFields returns true if the method has auto-populated fields.
func (m *Method) HasAutoPopulatedFields() bool {
	return len(m.AutoPopulated) != 0
}

// IsSimple returns true if the method is not a streaming, pagination or LRO method.
// IsSimple simplifies writing mustache templates, mostly for samples.
func (m *Method) IsSimple() bool {
	return m.Pagination == nil &&
		!m.ClientSideStreaming && !m.ServerSideStreaming &&
		m.OperationInfo == nil && m.DiscoveryLro == nil
}

// IsAIPStandard returns true if the method is one of the AIP standard methods.
// IsAIPStandard simplifies writing mustache templates, mostly for samples.
func (m *Method) IsAIPStandard() bool {
	return m.AIPStandardGetInfo() != nil ||
		m.AIPStandardDeleteInfo() != nil ||
		m.AIPStandardUndeleteInfo() != nil
}

// AIPStandardGetInfo contains information relevant to get operations
// that are like those defined by AIP-131.
type AIPStandardGetInfo struct {
	// ResourceNameRequestField is the field in the method input that contains the resource name
	// of the resource that the get operation should fetch.
	ResourceNameRequestField *Field
}

// AIPStandardGetInfo returns information relevant to a get operation that is like
// a get operation as defined by AIP-131, if the method is such an operation.
func (m *Method) AIPStandardGetInfo() *AIPStandardGetInfo {
	// A get operation is always a simple operation that returns a resource.
	if !m.IsSimple() || m.InputType == nil || m.ReturnsEmpty || m.OutputType == nil || m.OutputType.Resource == nil {
		return nil
	}

	// Standard get methods for resource "Foo" should be named "GetFoo".
	maybeSingular, found := strings.CutPrefix(strings.ToLower(m.Name), "get")
	if !found || maybeSingular == "" {
		return nil
	}
	// The request name should be "GetFooRequest".
	if strings.ToLower(m.InputType.Name) != fmt.Sprintf("get%srequest", maybeSingular) {
		return nil
	}

	// If the resource has a singular name, it must match.
	if m.OutputType.Resource.Singular != "" &&
		strings.ToLower(m.OutputType.Resource.Singular) != maybeSingular {
		return nil
	}

	// The request needs to have a field for the resource name of the resource to obtain.
	// We prioritize the matches as follows:
	// 1. The field explicitly references the output resource type.
	// 2. The field has a wildcard reference type "*" and is named "name".
	var bestField *Field
	for _, f := range m.InputType.Fields {
		if f.IsResourceReference() {
			if f.ResourceReference.Type == m.OutputType.Resource.Type {
				bestField = f
				break
			}
			if f.ResourceReference.Type == "*" && f.Name == "name" {
				bestField = f
			}
		}
	}

	if bestField == nil {
		return nil
	}

	return &AIPStandardGetInfo{
		ResourceNameRequestField: bestField,
	}
}

// AIPStandardDeleteInfo contains information relevant to delete operations
// that are like those defined by AIP-135.
type AIPStandardDeleteInfo struct {
	// ResourceNameRequestField is the field in the method input that contains the resource name
	// of the resource that the delete operation should delete.
	ResourceNameRequestField *Field
}

// AIPStandardDeleteInfo returns information relevant to a delete operation that is like
// a delete operation as defined by AIP-135, if the method is such an operation.
func (m *Method) AIPStandardDeleteInfo() *AIPStandardDeleteInfo {
	// A delete operation is either simple or LRO.
	if !m.IsSimple() && m.OperationInfo == nil {
		return nil
	}

	// Standard delete methods for resource "Foo" should be named "DeleteFoo".
	maybeSingular, found := strings.CutPrefix(strings.ToLower(m.Name), "delete")
	if !found || maybeSingular == "" {
		return nil
	}
	// The request name should be "DeleteFooRequest".
	if m.InputType == nil ||
		strings.ToLower(m.InputType.Name) != fmt.Sprintf("delete%srequest", maybeSingular) {
		return nil
	}

	// Find the field in the request that is a resource reference of a resource,
	// chosen as follows:
	//
	// We prioritize the matches as follows:
	// 1. The field name is "name" and the resource singular name matches maybeSingular.
	// 2. The field name is "name" and the resource singular name is empty.
	// 3. The resource singular name matches maybeSingular.
	//
	// Special case: If the resource reference type is "*", we treat it as a match
	// if the field name is "name".
	bestField := findBestResourceField(m.InputType.Fields, maybeSingular, m.Model.State.ResourceByType)
	if bestField == nil {
		return nil
	}

	return &AIPStandardDeleteInfo{
		ResourceNameRequestField: bestField,
	}
}

// AIPStandardUndeleteInfo contains information relevant to an undelete operation
// as implied by AIP-135.
type AIPStandardUndeleteInfo struct {
	// ResourceNameRequestField is the field in the method input that contains the resource name
	// of the resource that the undelete operation should restore.
	ResourceNameRequestField *Field
}

// AIPStandardUndeleteInfo returns information relevant to an undelete operation that is like
// an undelete operation as implied by AIP-135, if the method is such an operation.
func (m *Method) AIPStandardUndeleteInfo() *AIPStandardUndeleteInfo {
	// An undelete operation is either simple or LRO.
	if !m.IsSimple() && m.OperationInfo == nil {
		return nil
	}

	// Standard undelete methods for resource "Foo" should be named "UndeleteFoo".
	maybeSingular, found := strings.CutPrefix(strings.ToLower(m.Name), "undelete")
	if !found || maybeSingular == "" {
		return nil
	}
	// The request name should be "UndeleteFooRequest".
	if m.InputType == nil ||
		strings.ToLower(m.InputType.Name) != fmt.Sprintf("undelete%srequest", maybeSingular) {
		return nil
	}

	// Find the field in the request that is a resource reference of a resource,
	// chosen as follows:
	//
	// We prioritize the matches as follows:
	// 1. The field name is "name" and the resource singular name matches maybeSingular.
	// 2. The field name is "name" and the resource singular name is empty.
	// 3. The resource singular name matches maybeSingular.
	//
	// Special case: If the resource reference type is "*", we treat it as a match
	// if the field name is "name".
	bestField := findBestResourceField(m.InputType.Fields, maybeSingular, m.Model.State.ResourceByType)
	if bestField == nil {
		return nil
	}

	return &AIPStandardUndeleteInfo{
		ResourceNameRequestField: bestField,
	}
}

func findBestResourceField(fields []*Field, maybeSingular string, resourceMap map[string]*Resource) *Field {
	var bestField *Field
	for _, f := range fields {
		if f.ResourceReference == nil {
			continue
		}

		if f.ResourceReference.Type == "*" {
			if f.Name == "name" {
				bestField = f
				break
			}
			continue
		}

		resource, ok := resourceMap[f.ResourceReference.Type]
		if !ok {
			continue
		}

		fieldSingular := strings.ToLower(resource.Singular)

		if f.Name == "name" &&
			(fieldSingular == "" || fieldSingular == maybeSingular) {
			bestField = f
			// If we already found a field named "name" and it matches 1. or 2.
			// we won't find a better field.
			break
		}

		if fieldSingular == maybeSingular {
			bestField = f
		}
	}
	return bestField
}

// PathInfo contains normalized request path information.
type PathInfo struct {
	// The list of bindings, including the top-level binding.
	Bindings []*PathBinding
	// Body is the name of the field that should be used as the body of the
	// request.
	//
	// This is a string that may be "*" which indicates that the entire request
	// should be used as the body.
	//
	// If this is empty then the body is not used.
	BodyFieldPath string
	// Language specific annotations.
	Codec any
}

// PathBinding is a binding of a path to a method.
type PathBinding struct {
	// HTTP Verb.
	//
	// This is one of:
	// - GET
	// - POST
	// - PUT
	// - DELETE
	// - PATCH
	Verb string
	// The path broken by components.
	PathTemplate *PathTemplate
	// Query parameter fields.
	QueryParameters map[string]bool
	// Language specific annotations.
	Codec any
}

// OperationInfo contains normalized long running operation info.
type OperationInfo struct {
	// The metadata type. If there is no metadata, this is set to
	// `.google.protobuf.Empty`.
	MetadataTypeID string
	// The result type. This is the expected type when the long running
	// operation completes successfully.
	ResponseTypeID string
	// The method.
	Method *Method
	// Language specific annotations.
	Codec any
}

// DiscoveryLro contains old-style long-running operation descriptors.
type DiscoveryLro struct {
	// The path parameters required by the polling operation.
	PollingPathParameters []string
	// Language specific annotations.
	Codec any
}

// RoutingInfo contains normalized routing info.
//
// The routing information format is documented in:
//
// https://google.aip.dev/client-libraries/4222
//
// At a high level, it consists of a field name (from the request) that is used
// to match a certain path template. If the value of the field matches the
// template, the matching portion is added to `x-goog-request-params`.
//
// An empty `Name` field is used as the special marker to cover this case in
// AIP-4222:
//
//	An empty google.api.routing annotation is acceptable. It means that no
//	routing headers should be generated for the RPC, when they otherwise
//	would be e.g. implicitly from the google.api.http annotation.
type RoutingInfo struct {
	// The name in `x-goog-request-params`.
	Name string
	// Group the possible variants for the given name.
	//
	// The variants are parsed into the reverse order of definition. AIP-4222
	// declares:
	//
	//   In cases when multiple routing parameters have the same resource ID
	//   path segment name, thus referencing the same header key, the
	//   "last one wins" rule is used to determine which value to send.
	//
	// Reversing the order allows us to implement "the first match wins". That
	// is easier and more efficient in most languages.
	Variants []*RoutingInfoVariant
}

// RoutingInfoVariant represents the routing information stripped of its name.
type RoutingInfoVariant struct {
	// The sequence of field names accessed to get the routing information.
	FieldPath []string
	// A path template that must match the beginning of the field value.
	Prefix RoutingPathSpec
	// A path template that, if matching, is used in the `x-goog-request-params`.
	Matching RoutingPathSpec
	// A path template that must match the end of the field value.
	Suffix RoutingPathSpec
	// Language specific information
	Codec any
}

// FieldName returns the field path as a string.
func (v *RoutingInfoVariant) FieldName() string {
	return strings.Join(v.FieldPath, ".")
}

// TemplateAsString returns the template as a string.
func (v *RoutingInfoVariant) TemplateAsString() string {
	var full []string
	full = append(full, v.Prefix.Segments...)
	full = append(full, v.Matching.Segments...)
	full = append(full, v.Suffix.Segments...)
	return strings.Join(full, "/")
}

// RoutingPathSpec is a specification for a routing path.
type RoutingPathSpec struct {
	// A sequence of matching segments.
	//
	// A template like `projects/*/location/*/**` maps to
	// `["projects", "*", "locations", "*", "**"]`.
	Segments []string
}

const (
	// SingleSegmentWildcard is a special routing path segment which indicates
	// "match anything that does not include a `/`".
	SingleSegmentWildcard = "*"

	// MultiSegmentWildcard is a special routing path segment which indicates
	// "match anything including `/`".
	MultiSegmentWildcard = "**"
)

// PathTemplate is a template for a path.
type PathTemplate struct {
	Segments []PathSegment
	Verb     *string
}

// FlatPath returns a simplified representation of the path template as a string.
//
// In the context of discovery LROs it is useful to get the path template as a
// simplified string, such as "compute/v1/projects/{project}/zones/{zone}/instances".
// The path can be matched against LRO prefixes and then mapped to the correct
// poller RPC.
func (template *PathTemplate) FlatPath() string {
	var buffer strings.Builder
	sep := ""
	for _, segment := range template.Segments {
		buffer.WriteString(sep)
		if segment.Literal != nil {
			buffer.WriteString(*segment.Literal)
		} else if segment.Variable != nil {
			buffer.WriteString(fmt.Sprintf("{%s}", strings.Join(segment.Variable.FieldPath, ".")))
		}
		sep = "/"
	}
	return buffer.String()
}

// PathSegment is a segment of a path.
type PathSegment struct {
	Literal  *string
	Variable *PathVariable
}

// PathVariable is a variable in a path.
type PathVariable struct {
	FieldPath []string
	Segments  []string
	// Allow characters defined as `reserved` by RFC-6570 1.5 to pass through without
	// percent encoding. See RFC-6570 1.2 for examples.
	AllowReserved bool
}

// PathMatch is a single wildcard match in a path.
type PathMatch struct{}

// PathMatchRecursive is a recursive wildcard match in a path.
type PathMatchRecursive struct{}

// NewPathTemplate creates a new PathTemplate.
func NewPathTemplate() *PathTemplate {
	return &PathTemplate{}
}

// NewPathVariable creates a new path variable.
func NewPathVariable(fields ...string) *PathVariable {
	return &PathVariable{FieldPath: fields}
}

// WithLiteral adds a literal to the path template.
func (p *PathTemplate) WithLiteral(l string) *PathTemplate {
	p.Segments = append(p.Segments, PathSegment{Literal: &l})
	return p
}

// WithVariable adds a variable to the path template.
func (p *PathTemplate) WithVariable(v *PathVariable) *PathTemplate {
	p.Segments = append(p.Segments, PathSegment{Variable: v})
	return p
}

// WithVariableNamed adds a variable with the given name to the path template.
func (p *PathTemplate) WithVariableNamed(fields ...string) *PathTemplate {
	v := PathVariable{FieldPath: fields}
	p.Segments = append(p.Segments, PathSegment{Variable: v.WithMatch()})
	return p
}

// WithVerb adds a verb to the path template.
func (p *PathTemplate) WithVerb(v string) *PathTemplate {
	p.Verb = &v
	return p
}

// WithLiteral adds a literal to the path variable.
func (v *PathVariable) WithLiteral(l string) *PathVariable {
	v.Segments = append(v.Segments, l)
	return v
}

// WithMatchRecursive adds a recursive match to the path variable.
func (v *PathVariable) WithMatchRecursive() *PathVariable {
	v.Segments = append(v.Segments, MultiSegmentWildcard)
	return v
}

// WithMatch adds a match to the path variable.
func (v *PathVariable) WithMatch() *PathVariable {
	v.Segments = append(v.Segments, SingleSegmentWildcard)
	return v
}

// WithAllowReserved marks the variable as allowing reserved characters to remain unescaped.
func (v *PathVariable) WithAllowReserved() *PathVariable {
	v.AllowReserved = true
	return v
}

// NewPathSegment creates a new path segment.
func NewPathSegment() *PathSegment {
	return &PathSegment{}
}

// WithLiteral adds a literal to the path segment.
func (s *PathSegment) WithLiteral(l string) *PathSegment {
	s.Literal = &l
	return s
}

// WithVariable adds a variable to the path segment.
func (s *PathSegment) WithVariable(v *PathVariable) *PathSegment {
	s.Variable = v
	return s
}

// Message defines a message used in request/response handling.
type Message struct {
	// Documentation for the message.
	Documentation string
	// Name of the attribute.
	Name string
	// ID is a unique identifier.
	ID string
	// Some source specifications allow marking messages as deprecated.
	Deprecated bool
	// Fields associated with the Message.
	Fields []*Field
	// If true, this is a synthetic request message.
	//
	// These messages are created by sidekick when parsing Discovery docs and
	// OpenAPI specifications. The query and request parameters for each method
	// are grouped into a synthetic message.
	SyntheticRequest bool
	// If true, this message is a placeholder / doppelganger for a `api.Service`.
	//
	// These messages are created by sidekick when parsing Discovery docs and
	// OpenAPI specifications. All the synthetic messages for a service need to
	// be grouped under a unique namespace to avoid clashes with similar
	// synthetic messages in other services. Sidekick creates a placeholder
	// message that represents "the service".
	//
	// That is, `service1` and `service2` may both have a synthetic `getRequest`
	// message, with different attributes. We need these to be different
	// messages, with different names. So we create a different parent message
	// for each.
	ServicePlaceholder bool
	// Enums associated with the Message.
	Enums []*Enum
	// Messages associated with the Message. In protobuf these are referred to as
	// nested messages.
	Messages []*Message
	// OneOfs associated with the Message.
	OneOfs []*OneOf
	// Parent returns the ancestor of this message, if any.
	Parent *Message
	// The Protobuf package this message belongs to.
	Package string
	IsMap   bool
	// Indicates that this Message is returned by a standard
	// List RPC and conforms to [AIP-4233](https://google.aip.dev/client-libraries/4233).
	Pagination *PaginationInfo
	// Resource contains the data from the `google.api.resource` annotation.
	Resource *Resource
	// Language specific annotations.
	Codec any
}

// HasFields returns true if the message has fields.
func (m *Message) HasFields() bool {
	return len(m.Fields) != 0
}

// PaginationInfo contains information related to pagination aka [AIP-4233](https://google.aip.dev/client-libraries/4233).
type PaginationInfo struct {
	// The field that gives us the next page token.
	NextPageToken *Field
	// PageableItem is the field to be paginated over.
	PageableItem *Field
}

// Enum defines a message used in request/response handling.
type Enum struct {
	// Documentation for the message.
	Documentation string
	// Name of the attribute.
	Name string
	// ID is a unique identifier.
	ID string
	// Some source specifications allow marking enums as deprecated.
	Deprecated bool
	// Values associated with the Enum.
	Values []*EnumValue
	// The unique integer values, some enums have multiple aliases for the
	// same number (e.g. `enum X { a = 0, b = 0, c = 1 }`).
	UniqueNumberValues []*EnumValue
	// Parent returns the ancestor of this node, if any.
	Parent *Message
	// The Protobuf package this enum belongs to.
	Package string
	// Language specific annotations.
	Codec any
}

// EnumValue defines a value in an Enum.
type EnumValue struct {
	// Documentation for the message.
	Documentation string
	// Name of the attribute.
	Name string
	// ID is a unique identifier.
	ID string
	// Some source specifications allow marking enum values as deprecated.
	Deprecated bool
	// Number of the attribute.
	Number int32
	// Parent returns the ancestor of this node, if any.
	Parent *Enum
	// Language specific annotations.
	Codec any
}

// Field defines a field in a Message.
type Field struct {
	// Documentation for the field.
	Documentation string
	// Name of the attribute.
	Name string
	// ID is a unique identifier.
	ID string
	// Typez is the datatype of the field.
	Typez Typez
	// TypezID is the ID of the type the field refers to. This value is populated
	// for message-like types only.
	TypezID string
	// JSONName is the name of the field as it appears in JSON. Useful for
	// serializing to JSON.
	JSONName string
	// Optional indicates that the field is marked as optional in proto3.
	Optional bool

	// For a given field, at most one of `Repeated` or `Map` is true.
	//
	// Using booleans (as opposed to an enum) makes it easier to write mustache
	// templates.
	//
	// Repeated is true if the field is a repeated field.
	Repeated bool
	// Map is true if the field is a map.
	Map bool
	// Some source specifications allow marking fields as deprecated.
	Deprecated bool
	// IsOneOf is true if the field is related to a one-of and not
	// a proto3 optional field.
	IsOneOf bool
	// Some fields have a type that refers (sometimes indirectly) to the
	// containing message. That triggers slightly different code generation for
	// some languages.
	Recursive bool
	// AutoPopulated is true if the field is eligible to be auto-populated,
	// per the requirements in AIP-4235.
	//
	// That is:
	// - It has Typez == STRING_TYPE
	// - For Protobuf, does not have the `google.api.field_behavior = REQUIRED` annotation
	// - For Protobuf, has the `google.api.field_info.format = UUID4` annotation
	// - For OpenAPI, it is an optional field
	// - For OpenAPI, it has format == "uuid"
	AutoPopulated bool
	// FieldBehavior indicates how the field behaves in requests and responses.
	//
	// For example, that a field is required in requests, or given as output
	// but ignored as input.
	Behavior []FieldBehavior
	// For fields that are part of a OneOf, the group of fields that makes the
	// OneOf.
	Group *OneOf
	// The message that contains this field.
	Parent *Message
	// The message type for this field, can be nil.
	MessageType *Message
	// The enum type for this field, can be nil.
	EnumType *Enum
	// ResourceReference contains the data from the `google.api.resource_reference`
	// annotation.
	ResourceReference *ResourceReference
	// Codec is a placeholder to put language specific annotations.
	Codec any
}

// FieldParent returns the Parent field with an alternative name.
//
// In some mustache templates we want to access the parent for the
// enclosing field. In mustache you can get a field from an enclosing context
// *if* the name is unique.
func (f *Field) FieldParent() *Message {
	return f.Parent
}

// FieldCodec returns the Codec field with an alternative name.
//
// In some mustache templates we want to access the codec for the
// enclosing field. In mustache you can get a field from an enclosing context
// *if* the name is unique.
func (f *Field) FieldCodec() any {
	return f.Codec
}

// DocumentAsRequired returns true if the field should be documented as required.
func (field *Field) DocumentAsRequired() bool {
	return slices.Contains(field.Behavior, FIELD_BEHAVIOR_REQUIRED)
}

// Singular returns true if the field is not a map or a repeated field.
func (f *Field) Singular() bool {
	return !f.Map && !f.Repeated
}

// NameEqualJSONName returns true if the field's name is the same as its JSON name.
func (f *Field) NameEqualJSONName() bool {
	return f.JSONName == f.Name
}

// IsString returns true if the primitive type of a field is `STRING_TYPE`.
//
// This is useful for mustache templates that differ only
// in the broad category of field type involved.
func (f *Field) IsString() bool {
	return f.Typez == STRING_TYPE
}

// IsBytes returns true if the primitive type of a field is `BYTES_TYPE`.
//
// This is useful for mustache templates that differ only
// in the broad category of field type involved.
func (f *Field) IsBytes() bool {
	return f.Typez == BYTES_TYPE
}

// IsBool returns true if the primitive type of a field is `BOOL_TYPE`.
//
// This is useful for mustache templates that differ only
// in the broad category of field type involved.
func (f *Field) IsBool() bool {
	return f.Typez == BOOL_TYPE
}

// IsLikeInt returns true if the primitive type of a field is one of the
// integer types.
//
// This is useful for mustache templates that differ only
// in the broad category of field type involved.
func (f *Field) IsLikeInt() bool {
	switch f.Typez {
	case INT32_TYPE, INT64_TYPE, SINT32_TYPE, SINT64_TYPE:
		return true
	case SFIXED32_TYPE, SFIXED64_TYPE:
		return true
	default:
		return false
	}
}

// IsLikeUInt returns true if the primitive type of a field is one of the
// unsigned integer types.
//
// This is useful for mustache templates that differ only
// in the broad category of field type involved.
func (f *Field) IsLikeUInt() bool {
	switch f.Typez {
	case UINT32_TYPE, UINT64_TYPE, FIXED32_TYPE, FIXED64_TYPE:
		return true
	default:
		return false
	}
}

// IsLikeFloat returns true if the primitive type of a field is a float or
// double.
//
// This is useful for mustache templates that differ only
// in the broad category of field type involved.
func (f *Field) IsLikeFloat() bool {
	return f.Typez == DOUBLE_TYPE || f.Typez == FLOAT_TYPE
}

// IsEnum returns true if the primitive type of a field is `ENUM_TYPE`.
//
// This is useful for mustache templates that differ only
// in the broad category of field type involved.
func (f *Field) IsEnum() bool {
	return f.Typez == ENUM_TYPE
}

// IsObject returns true if the primitive type of a field is `OBJECT_TYPE`.
//
// This is useful for mustache templates that differ only
// in the broad category of field type involved.
//
// The templates *should* first check if the field is singular, as all maps are
// also objects.
func (f *Field) IsObject() bool {
	return f.Typez == MESSAGE_TYPE
}

// Pair is a key-value pair.
type Pair struct {
	// Key of the pair.
	Key string
	// Value of the pair.
	Value string
}

// OneOf is a group of fields that are mutually exclusive. Notably, proto3 optional
// fields are all their own one-of.
type OneOf struct {
	// Name of the attribute.
	Name string
	// ID is a unique identifier.
	ID string
	// Documentation for the field.
	Documentation string
	// Fields associated with the one-of.
	Fields []*Field
	// Codec is a placeholder to put language specific annotations.
	Codec any
}

// Resource is a fundamental building block of an API, representing an
// individually-named entity (a "noun").
//
// Resources are typically organized into a hierarchy, where each node is either a simple resource or a
// collection of resources.
// This definition is based on AIP-121 (https://google.aip.dev/121).
type Resource struct {
	// Type identifies the kind of resource (e.g., "cloudresourcemanager.googleapis.com/Project").
	// This string is globally unique and identifies the type of resource across Google Cloud.
	Type string
	// Pattern is a list of resource patterns, where each pattern is a sequence of path segments.
	// This defines the structure of the resource's unique identifier.
	Patterns []ResourcePattern
	// Plural is the plural form of the resource name.
	// For example, for a "Book" resource, Plural would be "books".
	Plural string
	// Singular is the singular form of the resource name.
	// For example, for a "Book" resource, Singular would be "book".
	Singular string
	// Self points to the Message that defines this resource.
	// This creates a back-reference for navigating the API model,
	// allowing a Resource definition to access its originating Message structure.
	Self *Message
	// Language specific annotations.
	Codec any
}

// ResourcePattern is a sequence of path segments that defines the structure of a resource's unique identifier.
type ResourcePattern []PathSegment

// ResourceReference describes a field's relationship to another resource type.
// It acts as a foreign key, indicating that the field's value identifies an instance of another resource.
// This relationship is established via the `google.api.resource_reference` annotation in Protobuf.
type ResourceReference struct {
	// Type is the unique identifier of the referenced resource's kind (e.g., "library.googleapis.com/Shelf").
	// This string matches the `Type` field in the corresponding `Resource` definition.
	Type string
	// ChildType is the unique identifier of a *child* resource's kind.
	// This is used when a field references a parent resource (e.g., "Shelf"), but the context
	// implies interaction with a specific child type (e.g., "Book" within that shelf).
	ChildType string
	// Language specific annotations.
	Codec any
}

// IsResourceReference returns true if the field is annotated with google.api.resource_reference.
func (f *Field) IsResourceReference() bool {
	return f.ResourceReference != nil
}
