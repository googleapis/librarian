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

// MethodBuilder builds a [Method] instance.
type MethodBuilder struct {
	m Method
}

// NewMethodBuilder creates a new [MethodBuilder].
func NewMethodBuilder() *MethodBuilder {
	return &MethodBuilder{}
}

// WithDocumentation sets the documentation for the method.
func (b *MethodBuilder) WithDocumentation(doc string) *MethodBuilder {
	b.m.documentation = doc
	return b
}

// WithName sets the name of the method.
func (b *MethodBuilder) WithName(name string) *MethodBuilder {
	b.m.name = name
	return b
}

// WithID sets the unique identifier for the method.
func (b *MethodBuilder) WithID(id string) *MethodBuilder {
	b.m.id = id
	return b
}

// WithDeprecated sets whether the method is deprecated.
func (b *MethodBuilder) WithDeprecated(deprecated bool) *MethodBuilder {
	b.m.deprecated = deprecated
	return b
}

// WithInputTypeID sets the input type ID for the method.
func (b *MethodBuilder) WithInputTypeID(inputTypeID string) *MethodBuilder {
	b.m.inputTypeID = inputTypeID
	return b
}

// WithInputType sets the input type for the method.
func (b *MethodBuilder) WithInputType(inputType *Message) *MethodBuilder {
	b.m.inputType = inputType
	return b
}

// WithOutputTypeID sets the output type ID for the method.
func (b *MethodBuilder) WithOutputTypeID(outputTypeID string) *MethodBuilder {
	b.m.outputTypeID = outputTypeID
	return b
}

// WithOutputType sets the output type for the method.
func (b *MethodBuilder) WithOutputType(outputType *Message) *MethodBuilder {
	b.m.outputType = outputType
	return b
}

// WithReturnsEmpty sets whether the method returns nothing.
func (b *MethodBuilder) WithReturnsEmpty(returnsEmpty bool) *MethodBuilder {
	b.m.returnsEmpty = returnsEmpty
	return b
}

// WithPathInfo sets the HTTP request path info for the method.
func (b *MethodBuilder) WithPathInfo(pathInfo *PathInfo) *MethodBuilder {
	b.m.pathInfo = pathInfo
	return b
}

// WithPagination sets the pagination page_token field for the method.
func (b *MethodBuilder) WithPagination(pagination *Field) *MethodBuilder {
	b.m.pagination = pagination
	return b
}

// WithClientSideStreaming sets whether the method supports client-side streaming.
func (b *MethodBuilder) WithClientSideStreaming(clientSideStreaming bool) *MethodBuilder {
	b.m.clientSideStreaming = clientSideStreaming
	return b
}

// WithServerSideStreaming sets whether the method supports server-side streaming.
func (b *MethodBuilder) WithServerSideStreaming(serverSideStreaming bool) *MethodBuilder {
	b.m.serverSideStreaming = serverSideStreaming
	return b
}

// WithIsLroPoller sets whether the method is an LRO poller.
func (b *MethodBuilder) WithIsLroPoller(isLroPoller bool) *MethodBuilder {
	b.m.isLroPoller = isLroPoller
	return b
}

// WithOperationInfo sets the long-running operation info for the method.
func (b *MethodBuilder) WithOperationInfo(operationInfo *OperationInfo) *MethodBuilder {
	b.m.operationInfo = operationInfo
	return b
}

// WithDiscoveryLro sets the discovery LRO info for the method.
func (b *MethodBuilder) WithDiscoveryLro(discoveryLro *DiscoveryLro) *MethodBuilder {
	b.m.discoveryLro = discoveryLro
	return b
}

// WithRouting sets the routing annotations for the method.
func (b *MethodBuilder) WithRouting(routing []*RoutingInfo) *MethodBuilder {
	b.m.routing = routing
	return b
}

// WithAutoPopulated sets the auto-populated fields for the method.
func (b *MethodBuilder) WithAutoPopulated(autoPopulated []*Field) *MethodBuilder {
	b.m.autoPopulated = autoPopulated
	return b
}

// WithAPIVersion sets the APIVersion for the method.
func (b *MethodBuilder) WithAPIVersion(apiVersion string) *MethodBuilder {
	b.m.apiVersion = apiVersion
	return b
}

// WithService sets the service this method belongs to.
func (b *MethodBuilder) WithService(service *Service) *MethodBuilder {
	b.m.service = service
	return b
}

// WithSourceService sets the original service this method belongs to.
func (b *MethodBuilder) WithSourceService(sourceService *Service) *MethodBuilder {
	b.m.sourceService = sourceService
	return b
}

// WithSourceServiceID sets the original service ID for this method.
func (b *MethodBuilder) WithSourceServiceID(sourceServiceID string) *MethodBuilder {
	b.m.sourceServiceID = sourceServiceID
	return b
}

// WithSignatures sets the method signatures for the method.
func (b *MethodBuilder) WithSignatures(signatures []*MethodSignature) *MethodBuilder {
	b.m.signatures = signatures
	return b
}

// WithSampleInfo sets the sample generation info for the method.
func (b *MethodBuilder) WithSampleInfo(sampleInfo *SampleInfo) *MethodBuilder {
	b.m.sampleInfo = sampleInfo
	return b
}

// WithModel sets the API model this method belongs to.
func (b *MethodBuilder) WithModel(model *API) *MethodBuilder {
	b.m.model = model
	return b
}

// WithIsSimple sets whether the method is simple.
func (b *MethodBuilder) WithIsSimple(isSimple bool) *MethodBuilder {
	b.m.isSimple = isSimple
	return b
}

// WithIsLRO sets whether the method is a long-running operation.
func (b *MethodBuilder) WithIsLRO(isLRO bool) *MethodBuilder {
	b.m.isLRO = isLRO
	return b
}

// WithIsList sets whether the method is a list operation.
func (b *MethodBuilder) WithIsList(isList bool) *MethodBuilder {
	b.m.isList = isList
	return b
}

// WithIsStreaming sets whether the method is streaming.
func (b *MethodBuilder) WithIsStreaming(isStreaming bool) *MethodBuilder {
	b.m.isStreaming = isStreaming
	return b
}

// WithIsAIPStandard sets whether the method is an AIP standard method.
func (b *MethodBuilder) WithIsAIPStandard(isAIPStandard bool) *MethodBuilder {
	b.m.isAIPStandard = isAIPStandard
	return b
}

// WithIsAIPStandardGet sets whether the method is an AIP standard get method.
func (b *MethodBuilder) WithIsAIPStandardGet(isGet bool) *MethodBuilder {
	b.m.isAIPStandardGet = isGet
	return b
}

// WithIsAIPStandardDelete sets whether the method is an AIP standard delete method.
func (b *MethodBuilder) WithIsAIPStandardDelete(isDelete bool) *MethodBuilder {
	b.m.isAIPStandardDelete = isDelete
	return b
}

// WithIsAIPStandardUndelete sets whether the method is an AIP standard undelete method.
func (b *MethodBuilder) WithIsAIPStandardUndelete(isUndelete bool) *MethodBuilder {
	b.m.isAIPStandardUndelete = isUndelete
	return b
}

// WithIsAIPStandardCreate sets whether the method is an AIP standard create method.
func (b *MethodBuilder) WithIsAIPStandardCreate(isCreate bool) *MethodBuilder {
	b.m.isAIPStandardCreate = isCreate
	return b
}

// WithIsAIPStandardUpdate sets whether the method is an AIP standard update method.
func (b *MethodBuilder) WithIsAIPStandardUpdate(isUpdate bool) *MethodBuilder {
	b.m.isAIPStandardUpdate = isUpdate
	return b
}

// WithIsAIPStandardList sets whether the method is an AIP standard list method.
func (b *MethodBuilder) WithIsAIPStandardList(isList bool) *MethodBuilder {
	b.m.isAIPStandardList = isList
	return b
}

// Build returns the constructed [Method].
func (b *MethodBuilder) Build() *Method {
	m := b.m
	return &m
}
