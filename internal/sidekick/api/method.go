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
	// IsLroPoller is true if the method is an LRO poller.
	IsLroPoller bool
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
	// APIVersion contains the interface-based-versioning version.
	//
	// If this is empty, then the method does not have a version annotation.
	APIVersion string
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
	// IsSimple is true if the method is not a streaming, pagination or LRO method.
	IsSimple bool
	// IsLRO is true if the method is a long-running operation.
	IsLRO bool
	// LongRunningResponseType is the response type of the long-running operation.
	LongRunningResponseType *Message
	// LongRunningReturnsEmpty is true if the long-running operation returns an empty response.
	LongRunningReturnsEmpty bool
	// IsList is true if the method is a list operation.
	IsList bool
	// IsStreaming is true if the method is client-side or server-side streaming.
	IsStreaming bool
	// IsAIPStandard is true if the method is one of the AIP standard methods.
	IsAIPStandard bool
	// IsAIPStandardGet is true if the method is an AIP standard get method.
	IsAIPStandardGet bool
	// IsAIPStandardDelete is true if the method is an AIP standard delete method.
	IsAIPStandardDelete bool
	// IsAIPStandardUndelete is true if the method is an AIP standard undelete method.
	IsAIPStandardUndelete bool
	// IsAIPStandardCreate is true if the method is an AIP standard create method.
	IsAIPStandardCreate bool
	// IsAIPStandardUpdate is true if the method is an AIP standard update method.
	IsAIPStandardUpdate bool
	// IsAIPStandardList is true if the method is an AIP standard list method.
	IsAIPStandardList bool
	// SampleInfo may contain sample generation information for this method,
	// usually if it is an AIP conforming metho.
	SampleInfo *SampleInfo
	// Signatures defines alternative signatures (overloads) for the method.
	Signatures []*MethodSignature
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

// SampleInfo contains sample generation information for a single method,
// usually if it is an AIP conforming method.
type SampleInfo struct {
	// ResourceNameField is the field containing the resource name or parent resource name.
	ResourceNameField *Field
	// ResourceIDField is the field containing the resource ID, usually present in Create methods.
	ResourceIDField *Field
	// MessageField is the field containing the message body to be created or updated.
	MessageField *Field
	// UpdateMaskField is the field containing the update mask, present in Update methods.
	UpdateMaskField *Field
	// IsRequestResourceName is true if the main resource name associated to this sample,
	// i.e. the name of the resource manipulated by the RPC, is a field of the request
	// object itself, e.g. for Delete or Get operations.
	IsRequestResourceName bool
	// IsMessageResourceName is true if the main resource name associated to this sample,
	// i.e. the name of the resource manipulated by the RPC, is a field of the message body of
	// the request object, e.g. for Update operations.
	IsMessageResourceName bool
	// Codec is a placeholder to put language specific annotations.
	Codec any
}
