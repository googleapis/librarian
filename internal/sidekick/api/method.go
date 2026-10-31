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

package api

import "slices"

// Method defines a RPC belonging to a Service.
type Method struct {
	// documentation is the documentation for the method.
	documentation string
	// name is the name of the attribute.
	name string
	// id is a unique identifier.
	id string
	// deprecated is true if the method is deprecated.
	deprecated bool
	// inputTypeID is the ID of the input type for the Method.
	inputTypeID string
	// inputType is the input to the Method.
	inputType *Message
	// outputTypeID is the ID of the output type for the Method.
	outputTypeID string
	// outputType is the output of the Method.
	outputType *Message
	// returnsEmpty is true if the method returns nothing.
	//
	// Protobuf uses the well-known type `google.protobuf.Empty` message to
	// represent this.
	//
	// OpenAPIv3 uses a missing content field:
	//   https://swagger.io/docs/specification/v3_0/describing-responses/#empty-response-body
	returnsEmpty bool
	// pathInfo contains information about the HTTP request.
	pathInfo *PathInfo
	// pagination holds the `page_token` field if the method conforms to the
	// standard defined by [AIP-4233](https://google.aip.dev/client-libraries/4233).
	pagination *Field
	// clientSideStreaming is true if the method supports client-side streaming.
	clientSideStreaming bool
	// serverSideStreaming is true if the method supports server-side streaming.
	serverSideStreaming bool
	// isLroPoller is true if the method is an LRO poller.
	isLroPoller bool
	// operationInfo contains information for methods returning long-running operations.
	operationInfo *OperationInfo
	// discoveryLro has a value if this is a discovery-style long-running operation.
	discoveryLro *DiscoveryLro
	// routing contains the routing annotations, if any.
	routing []*RoutingInfo
	// autoPopulated contains the auto-populated (request_id) field, if any, as defined in
	// [AIP-4235](https://google.aip.dev/client-libraries/4235)
	//
	// The field must be eligible for auto-population, and be listed in the
	// `google.api.MethodSettings.auto_populated_fields` entry in
	// `google.api.Publishing.method_settings` in the service config file.
	autoPopulated []*Field
	// apiVersion contains the interface-based-versioning version.
	//
	// If this is empty, then the method does not have a version annotation.
	apiVersion string
	// model is the model this method belongs to, Go templates use this field to
	// navigate the data structure.
	model *API
	// service is the service this method belongs to, Go templates use this field to
	// navigate the data structure.
	service *Service
	// `sourceService` is the original service this method belongs to. For most
	// methods `sourceService` and `service` are the same. For mixins, the
	// source service is the mixin, such as longrunning.Operations.
	sourceService *Service
	// `sourceServiceID` is the original service ID for this method.
	sourceServiceID string
	// isSimple is true if the method is not a streaming, pagination or LRO method.
	isSimple bool
	// isLRO is true if the method is a long-running operation.
	isLRO bool
	// longRunningResponseType is the response type of the long-running operation.
	longRunningResponseType *Message
	// longRunningReturnsEmpty is true if the long-running operation returns an empty response.
	longRunningReturnsEmpty bool
	// isList is true if the method is a list operation.
	isList bool
	// isStreaming is true if the method is client-side or server-side streaming.
	isStreaming bool
	// isAIPStandard is true if the method is one of the AIP standard methods.
	isAIPStandard bool
	// isAIPStandardGet is true if the method is an AIP standard get method.
	isAIPStandardGet bool
	// isAIPStandardDelete is true if the method is an AIP standard delete method.
	isAIPStandardDelete bool
	// isAIPStandardUndelete is true if the method is an AIP standard undelete method.
	isAIPStandardUndelete bool
	// isAIPStandardCreate is true if the method is an AIP standard create method.
	isAIPStandardCreate bool
	// isAIPStandardUpdate is true if the method is an AIP standard update method.
	isAIPStandardUpdate bool
	// isAIPStandardList is true if the method is an AIP standard list method.
	isAIPStandardList bool
	// sampleInfo may contain sample generation information for this method,
	// usually if it is an AIP conforming method.
	sampleInfo *SampleInfo
	// signatures defines alternative signatures (overloads) for the method.
	signatures []*MethodSignature
	// codec contains language specific annotations.
	codec any
}

// Documentation returns the documentation for the method.
func (m *Method) Documentation() string {
	return m.documentation
}

// Name returns the name of the attribute.
func (m *Method) Name() string {
	return m.name
}

// ID returns a unique identifier.
func (m *Method) ID() string {
	return m.id
}

// Deprecated returns true if the method is deprecated.
func (m *Method) Deprecated() bool {
	return m.deprecated
}

// InputTypeID returns the ID of the input type for the Method.
func (m *Method) InputTypeID() string {
	return m.inputTypeID
}

// InputType returns the input to the Method.
func (m *Method) InputType() *Message {
	return m.inputType
}

// OutputTypeID returns the ID of the output type for the Method.
func (m *Method) OutputTypeID() string {
	return m.outputTypeID
}

// OutputType returns the output of the Method.
func (m *Method) OutputType() *Message {
	return m.outputType
}

// ReturnsEmpty returns true if the method returns nothing.
func (m *Method) ReturnsEmpty() bool {
	return m.returnsEmpty
}

// PathInfo returns information about the HTTP request.
func (m *Method) PathInfo() *PathInfo {
	return m.pathInfo
}

// Pagination returns the `page_token` field if the method conforms to AIP-4233.
func (m *Method) Pagination() *Field {
	return m.pagination
}

// ClientSideStreaming returns true if the method supports client-side streaming.
func (m *Method) ClientSideStreaming() bool {
	return m.clientSideStreaming
}

// ServerSideStreaming returns true if the method supports server-side streaming.
func (m *Method) ServerSideStreaming() bool {
	return m.serverSideStreaming
}

// IsLroPoller returns true if the method is an LRO poller.
func (m *Method) IsLroPoller() bool {
	return m.isLroPoller
}

// OperationInfo returns information for methods returning long-running operations.
func (m *Method) OperationInfo() *OperationInfo {
	return m.operationInfo
}

// DiscoveryLro returns a value if this is a discovery-style long-running operation.
func (m *Method) DiscoveryLro() *DiscoveryLro {
	return m.discoveryLro
}

// Routing returns the routing annotations, if any.
func (m *Method) Routing() []*RoutingInfo {
	return slices.Clone(m.routing)
}

// AutoPopulated returns the auto-populated (request_id) fields, if any.
func (m *Method) AutoPopulated() []*Field {
	return slices.Clone(m.autoPopulated)
}

// APIVersion returns the interface-based-versioning version.
func (m *Method) APIVersion() string {
	return m.apiVersion
}

// Model returns the model this method belongs to.
func (m *Method) Model() *API {
	return m.model
}

// Service returns the service this method belongs to.
func (m *Method) Service() *Service {
	return m.service
}

// SourceService returns the original service this method belongs to.
func (m *Method) SourceService() *Service {
	return m.sourceService
}

// SourceServiceID returns the original service ID for this method.
func (m *Method) SourceServiceID() string {
	return m.sourceServiceID
}

// IsSimple returns true if the method is not a streaming, pagination or LRO method.
func (m *Method) IsSimple() bool {
	return m.isSimple
}

// IsLRO returns true if the method is a long-running operation.
func (m *Method) IsLRO() bool {
	return m.isLRO
}

// LongRunningResponseType returns the response type of the long-running operation.
func (m *Method) LongRunningResponseType() *Message {
	return m.longRunningResponseType
}

// LongRunningReturnsEmpty returns true if the long-running operation returns an empty response.
func (m *Method) LongRunningReturnsEmpty() bool {
	return m.longRunningReturnsEmpty
}

// IsList returns true if the method is a list operation.
func (m *Method) IsList() bool {
	return m.isList
}

// IsStreaming returns true if the method is client-side or server-side streaming.
func (m *Method) IsStreaming() bool {
	return m.isStreaming
}

// IsAIPStandard returns true if the method is one of the AIP standard methods.
func (m *Method) IsAIPStandard() bool {
	return m.isAIPStandard
}

// IsAIPStandardGet returns true if the method is an AIP standard get method.
func (m *Method) IsAIPStandardGet() bool {
	return m.isAIPStandardGet
}

// IsAIPStandardDelete returns true if the method is an AIP standard delete method.
func (m *Method) IsAIPStandardDelete() bool {
	return m.isAIPStandardDelete
}

// IsAIPStandardUndelete returns true if the method is an AIP standard undelete method.
func (m *Method) IsAIPStandardUndelete() bool {
	return m.isAIPStandardUndelete
}

// IsAIPStandardCreate returns true if the method is an AIP standard create method.
func (m *Method) IsAIPStandardCreate() bool {
	return m.isAIPStandardCreate
}

// IsAIPStandardUpdate returns true if the method is an AIP standard update method.
func (m *Method) IsAIPStandardUpdate() bool {
	return m.isAIPStandardUpdate
}

// IsAIPStandardList returns true if the method is an AIP standard list method.
func (m *Method) IsAIPStandardList() bool {
	return m.isAIPStandardList
}

// SampleInfo returns sample generation information for this method.
func (m *Method) SampleInfo() *SampleInfo {
	return m.sampleInfo
}

// Signatures returns alternative signatures (overloads) for the method.
func (m *Method) Signatures() []*MethodSignature {
	return slices.Clone(m.signatures)
}

// Codec returns language specific annotations.
func (m *Method) Codec() any {
	return m.codec
}

// SetAnnotations sets the language-specific annotations on the method.
func (m *Method) SetAnnotations(codec any) {
	m.codec = codec
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
	for _, info := range m.routing {
		next := []*RoutingInfoCombo{}
		for _, c := range combos {
			for _, v := range info.Variants() {
				next = append(next, &RoutingInfoCombo{
					items: append(c.items, &RoutingInfoComboItem{
						name:    info.Name(),
						variant: v,
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
	items []*RoutingInfoComboItem
}

// NewRoutingInfoCombo creates a new RoutingInfoCombo with the given items.
func NewRoutingInfoCombo(items ...*RoutingInfoComboItem) *RoutingInfoCombo {
	return &RoutingInfoCombo{items: items}
}

// Items returns the items in the routing info combination.
func (c *RoutingInfoCombo) Items() []*RoutingInfoComboItem {
	return slices.Clone(c.items)
}

// RoutingInfoComboItem represents a single item in a RoutingInfoCombo.
type RoutingInfoComboItem struct {
	name    string
	variant *RoutingInfoVariant
}

// NewRoutingInfoComboItem creates a new RoutingInfoComboItem with the given name and variant.
func NewRoutingInfoComboItem(name string, variant *RoutingInfoVariant) *RoutingInfoComboItem {
	return &RoutingInfoComboItem{name: name, variant: variant}
}

// Name returns the routing parameter name.
func (i *RoutingInfoComboItem) Name() string {
	return i.name
}

// Variant returns the routing parameter variant.
func (i *RoutingInfoComboItem) Variant() *RoutingInfoVariant {
	return i.variant
}

// HasRouting returns true if the method has routing information.
func (m *Method) HasRouting() bool {
	return len(m.routing) != 0
}

// HasAutoPopulatedFields returns true if the method has auto-populated fields.
func (m *Method) HasAutoPopulatedFields() bool {
	return len(m.autoPopulated) != 0
}

// IsBidiStreaming returns true if the method is a bidirectional streaming RPC.
func (m *Method) IsBidiStreaming() bool {
	return m.clientSideStreaming && m.serverSideStreaming
}

// SampleInfo contains sample generation information for a single method,
// usually if it is an AIP conforming method.
type SampleInfo struct {
	// resourceNameField is the field containing the resource name or parent resource name.
	resourceNameField *Field
	// resourceIDField is the field containing the resource ID, usually present in Create methods.
	resourceIDField *Field
	// messageField is the field containing the message body to be created or updated.
	messageField *Field
	// updateMaskField is the field containing the update mask, present in Update methods.
	updateMaskField *Field
	// isRequestResourceName is true if the main resource name associated to this sample,
	// i.e. the name of the resource manipulated by the RPC, is a field of the request
	// object itself, e.g. for Delete or Get operations.
	isRequestResourceName bool
	// isMessageResourceName is true if the main resource name associated to this sample,
	// i.e. the name of the resource manipulated by the RPC, is a field of the message body of
	// the request object, e.g. for Update operations.
	isMessageResourceName bool
	// codec is a placeholder to put language specific annotations.
	codec any
}

// ResourceNameField returns the field containing the resource name or parent resource name.
func (s *SampleInfo) ResourceNameField() *Field {
	return s.resourceNameField
}

// ResourceIDField returns the field containing the resource ID.
func (s *SampleInfo) ResourceIDField() *Field {
	return s.resourceIDField
}

// MessageField returns the field containing the message body to be created or updated.
func (s *SampleInfo) MessageField() *Field {
	return s.messageField
}

// UpdateMaskField returns the field containing the update mask.
func (s *SampleInfo) UpdateMaskField() *Field {
	return s.updateMaskField
}

// IsRequestResourceName returns true if the main resource name is a field of the request object itself.
func (s *SampleInfo) IsRequestResourceName() bool {
	return s.isRequestResourceName
}

// IsMessageResourceName returns true if the main resource name is a field of the message body.
func (s *SampleInfo) IsMessageResourceName() bool {
	return s.isMessageResourceName
}

// Codec returns language specific annotations.
func (s *SampleInfo) Codec() any {
	return s.codec
}

// SetAnnotations sets the language-specific annotations on the SampleInfo.
func (s *SampleInfo) SetAnnotations(codec any) {
	s.codec = codec
}
