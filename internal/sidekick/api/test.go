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

package api

import (
	"fmt"
	"slices"
	"strings"

	"github.com/iancoleman/strcase"
)

// NewTestAPI creates a new test API.
func NewTestAPI(messages []*Message, enums []*Enum, services []*Service) *API {
	model := &API{
		Name:           "Test",
		Messages:       messages,
		Enums:          enums,
		Services:       services,
		messageByID:    make(map[string]*Message),
		methodByID:     make(map[string]*Method),
		enumByID:       make(map[string]*Enum),
		serviceByID:    make(map[string]*Service),
		resourceByType: make(map[string]*Resource),
		mapByID:        make(map[string]*Map),
	}

	var indexMessage func(m *Message)
	indexMessage = func(m *Message) {
		model.messageByID[m.id] = m
		if m.resource != nil {
			model.resourceByType[m.resource.typez] = m.resource
		}
		for _, f := range m.fields {
			if f.mapType != nil {
				model.AddMap(f.mapType)
			}
		}
		for _, e := range m.enums {
			model.enumByID[e.id] = e
			e.parent = m
			for _, ev := range e.values {
				ev.parent = e
			}
		}
		for _, child := range m.messages {
			child.parent = m
			indexMessage(child)
		}
	}
	for _, m := range messages {
		model.PackageName = m.packagez
		indexMessage(m)
	}
	for _, e := range enums {
		model.PackageName = e.packagez
		model.enumByID[e.id] = e
	}
	for _, s := range services {
		model.PackageName = s.packagez
		model.serviceByID[s.id] = s
		for _, m := range s.methods {
			model.methodByID[m.id] = m
		}
	}
	for _, m := range messages {
		parentID := parentName(m.id)
		parent := model.messageByID[parentID]
		if parent != nil {
			m.parent = parent
			if !slices.Contains(parent.messages, m) {
				parent.messages = append(parent.messages, m)
			}
		}
	}
	for _, e := range enums {
		parent := model.messageByID[parentName(e.id)]
		if parent != nil {
			e.parent = parent
			if !slices.Contains(parent.enums, e) {
				parent.enums = append(parent.enums, e)
			}
		}
		for _, ev := range e.values {
			ev.parent = e
		}
	}

	model.LoadWellKnownTypes()
	return model
}

// WithName changes the name of an API instance.
func (a *API) WithName(name string) *API {
	a.Name = name
	return a
}

// WithTitle changes the title of an API instance.
func (a *API) WithTitle(title string) *API {
	a.Title = title
	return a
}

// WithDescription changes the description of an API instance.
func (a *API) WithDescription(description string) *API {
	a.Description = description
	return a
}

// WithQuickstartService sets the quickstart service on an API instance.
func (a *API) WithQuickstartService(service *Service) *API {
	a.QuickstartService = service
	return a
}

// WithPackageName changes the package name of an API instance.
func (a *API) WithPackageName(name string) *API {
	a.PackageName = name
	return a
}

// WithCsharpNamespace changes the CsharpNamespace of an API instance.
func (a *API) WithCsharpNamespace(name string) *API {
	a.CsharpNamespace = name
	return a
}

// WithPhpNamespace changes the PhpNamespace of an API instance.
func (a *API) WithPhpNamespace(name string) *API {
	a.PhpNamespace = name
	return a
}

// WithRubyPackage changes the RubyNamespace of an API instance.
func (a *API) WithRubyPackage(name string) *API {
	a.RubyPackage = name
	return a
}

// WithMaps adds maps to the API.
func (a *API) WithMaps(maps ...*Map) *API {
	for _, m := range maps {
		a.AddMap(m)
	}
	return a
}

// parentName returns the parent's name from a fully qualified identifier.
func parentName(id string) string {
	if lastIndex := strings.LastIndex(id, "."); lastIndex != -1 {
		return id[:lastIndex]
	}
	return "."
}

// NewTestMessage creates a message with defaults for testing.
// Default package is "test".
func NewTestMessage(name string) *Message {
	return NewMessageBuilder().WithName(name).Build().WithPackage("test")
}

// WithPackage sets the package for the message and updates its ID.
func (m *Message) WithPackage(pkg string) *Message {
	m.packagez = pkg
	m.id = fmt.Sprintf(".%s.%s", pkg, m.name)
	return m
}

// WithID overrides the message's ID.
func (m *Message) WithID(id string) *Message {
	m.id = id
	return m
}

// WithFields adds fields to the message and updates their parent/ID.
func (m *Message) WithFields(fields ...*Field) *Message {
	for _, f := range fields {
		f.parent = m
		// If field ID is generic/default, re-scope it to the message.
		if strings.HasPrefix(f.id, ".test.") || f.id == "" {
			f.id = fmt.Sprintf("%s.%s", m.id, f.name)
		}
	}
	m.fields = append(m.fields, fields...)
	return m
}

// WithOneOfs adds oneofs to the message and updates their parent/ID.
func (m *Message) WithOneOfs(oneofs ...*OneOf) *Message {
	for _, o := range oneofs {
		// If oneof ID is generic/default, re-scope it to the message.
		if strings.HasPrefix(o.id, ".test.") || o.id == "" {
			o.id = fmt.Sprintf("%s.%s", m.id, o.name)
		}
		m.oneOfs = append(m.oneOfs, o)
		m.WithFields(o.fields...)
	}
	return m
}

func (m *Message) withParentRecursive(parent *Message) {
	m.parent = parent
	if m.parent == nil {
		return
	}
	m.packagez = parent.packagez
	if strings.HasPrefix(m.id, ".test.") || m.id == "" {
		m.id = fmt.Sprintf("%s.%s", parent.id, m.name)
	}
	for _, f := range m.fields {
		if strings.HasPrefix(f.id, ".test.") || f.id == "" {
			f.id = fmt.Sprintf("%s.%s", m.id, f.name)
		}
	}
	for _, o := range m.oneOfs {
		if strings.HasPrefix(o.id, ".test.") || o.id == "" {
			o.id = fmt.Sprintf("%s.%s", m.id, o.name)
		}
	}
	for _, c := range m.messages {
		c.withParentRecursive(m)
	}
	for _, e := range m.enums {
		_ = e.WithParent(m)
	}
}

// WithMessages adds nested messages to the message and updates their parent/ID.
func (m *Message) WithMessages(messages ...*Message) *Message {
	for _, child := range messages {
		child.withParentRecursive(m)
	}
	m.messages = append(m.messages, messages...)
	return m
}

// WithEnums adds enums to the message and updates their parent/ID.
func (m *Message) WithEnums(enums ...*Enum) *Message {
	for _, e := range enums {
		e.WithParent(m)
	}
	m.enums = append(m.enums, enums...)
	return m
}

// WithPagination items and page token fields for a pagination response.
func (m *Message) WithPagination(nextPageToken *Field, items *Field) *Message {
	if nextPageToken != nil && nextPageToken.parent != m {
		m.WithFields(nextPageToken)
	}
	if items != nil && items.parent != m {
		m.WithFields(items)
	}
	m.pagination = NewPaginationInfo(nextPageToken, items)
	return m
}

// WithResource sets the resource definition on the message.
func (m *Message) WithResource(resource *Resource) *Message {
	m.resource = resource
	if resource != nil {
		resource.self = m
	}
	return m
}

// WithDeprecated sets whether the message is deprecated.
func (m *Message) WithDeprecated(deprecated bool) *Message {
	m.deprecated = deprecated
	return m
}

// WithDocumentation sets the documentation for the message.
func (m *Message) WithDocumentation(doc string) *Message {
	m.documentation = doc
	return m
}

// WithParent sets the parent message directly without rewriting IDs.
func (m *Message) WithParent(parent *Message) *Message {
	m.parent = parent
	return m
}

// WithSyntheticRequest marks the message as a synthetic request.
func (m *Message) WithSyntheticRequest() *Message {
	m.syntheticRequest = true
	return m
}

// WithServicePlaceholder marks the message as a service placeholder.
func (m *Message) WithServicePlaceholder() *Message {
	m.servicePlaceholder = true
	return m
}

// WithSkippedFields sets the skipped fields on the message.
func (m *Message) WithSkippedFields(fields ...*Field) *Message {
	m.skippedFields = fields
	return m
}

// NewTestMap creates a synthetic Map with "key" and "value" fields.
func NewTestMap(name string, keyType, valueType Typez) *Map {
	return NewTestMapWithFields(
		name,
		NewTestField("key").WithType(keyType),
		NewTestField("value").WithType(valueType),
	)
}

// NewTestMapWithFields creates a synthetic Map with custom key and value fields.
func NewTestMapWithFields(name string, key, value *Field) *Map {
	id := name
	if !strings.HasPrefix(name, ".") && !strings.HasPrefix(name, "$") {
		id = fmt.Sprintf(".test.%s", name)
	}
	if key != nil && (strings.HasPrefix(key.id, ".test.") || key.id == "") {
		key.id = fmt.Sprintf("%s.%s", id, key.name)
	}
	if value != nil && (strings.HasPrefix(value.id, ".test.") || value.id == "") {
		value.id = fmt.Sprintf("%s.%s", id, value.name)
	}
	return NewMapBuilder().WithID(id).WithKey(key).WithValue(value).Build()
}

// WithID overrides the map's ID.
func (m *Map) WithID(id string) *Map {
	oldID := m.id
	m.id = id
	if m.key != nil && (strings.HasPrefix(m.key.id, oldID+".") || strings.HasPrefix(m.key.id, ".test.") || m.key.id == "") {
		m.key.id = fmt.Sprintf("%s.%s", id, m.key.name)
	}
	if m.value != nil && (strings.HasPrefix(m.value.id, oldID+".") || strings.HasPrefix(m.value.id, ".test.") || m.value.id == "") {
		m.value.id = fmt.Sprintf("%s.%s", id, m.value.name)
	}
	return m
}

// NewTestService creates a service with defaults for testing.
// Default package is "test".
func NewTestService(name string) *Service {
	return NewServiceBuilder().WithName(name).Build().WithPackage("test")
}

// WithPackage sets the package for the service and updates its ID.
func (s *Service) WithPackage(pkg string) *Service {
	s.packagez = pkg
	s.id = fmt.Sprintf(".%s.%s", pkg, s.name)
	return s
}

// WithName overrides the service's Name.
func (s *Service) WithName(name string) *Service {
	s.name = name
	return s
}

// WithID overrides the service's ID.
func (s *Service) WithID(id string) *Service {
	s.id = id
	return s
}

// WithMethods adds methods to the service and updates their ID/Service.
func (s *Service) WithMethods(methods ...*Method) *Service {
	for _, m := range methods {
		m.service = s
		// If method ID is generic/default, re-scope it to the service.
		if strings.HasPrefix(m.id, ".test.") || m.id == "" {
			m.id = fmt.Sprintf("%s.%s", s.id, m.name)
		}
	}
	s.methods = append(s.methods, methods...)
	return s
}

// WithDeprecated sets whether the service is deprecated.
func (s *Service) WithDeprecated(deprecated bool) *Service {
	s.deprecated = deprecated
	return s
}

// WithDocumentation sets the documentation for the service.
func (s *Service) WithDocumentation(doc string) *Service {
	s.documentation = doc
	return s
}

// WithDefaultHost sets the default host for the service.
func (s *Service) WithDefaultHost(host string) *Service {
	s.defaultHost = host
	return s
}

// WithQuickstartMethod sets the quickstart method for the service.
func (s *Service) WithQuickstartMethod(method *Method) *Service {
	s.quickstartMethod = method
	return s
}

// WithModel sets the API model for the service.
func (s *Service) WithModel(model *API) *Service {
	s.model = model
	return s
}

// NewTestMethod creates a method with defaults for testing.
// Default package is "test" (implies ID .test.Name).
func NewTestMethod(name string) *Method {
	return NewMethodBuilder().
		WithName(name).
		WithID(fmt.Sprintf(".%s.%s", "test", name)).
		WithPathInfo(NewPathInfoBuilder().WithBindings(NewPathBindingBuilder().Build()).Build()).
		Build()
}

// WithID overrides the method's ID.
func (m *Method) WithID(id string) *Method {
	m.id = id
	return m
}

// WithAPIVersion sets the APIVersion for the method.
func (m *Method) WithAPIVersion(version string) *Method {
	m.apiVersion = version
	return m
}

// WithPathInfo sets the PathInfo for the method.
func (m *Method) WithPathInfo(pathInfo *PathInfo) *Method {
	m.pathInfo = pathInfo
	return m
}

// WithRouting adds routing info to the method.
func (m *Method) WithRouting(routing ...*RoutingInfo) *Method {
	m.routing = append(m.routing, routing...)
	return m
}

// WithAutoPopulated adds auto-populated fields to the method and marks them as auto-populated.
func (m *Method) WithAutoPopulated(fields ...*Field) *Method {
	for _, f := range fields {
		f.autoPopulated = true
	}
	m.autoPopulated = append(m.autoPopulated, fields...)
	return m
}

// WithDocumentation sets the documentation for the method.
func (m *Method) WithDocumentation(doc string) *Method {
	m.documentation = doc
	return m
}

func (m *Method) ensureFirstBinding() *PathBinding {
	if m.pathInfo == nil {
		m.pathInfo = &PathInfo{}
	}
	if len(m.pathInfo.bindings) == 0 {
		m.pathInfo.bindings = append(m.pathInfo.bindings, &PathBinding{})
	}
	return m.pathInfo.bindings[0]
}

// WithVerb sets the HTTP verb for the first binding.
func (m *Method) WithVerb(verb string) *Method {
	m.ensureFirstBinding().verb = verb
	return m
}

// WithInput sets the input type message for the method.
// It sets both InputType and InputTypeID.
func (m *Method) WithInput(msg *Message) *Method {
	m.inputType = msg
	if msg != nil {
		m.inputTypeID = msg.id
	}
	return m
}

// WithOutput sets the output type message for the method.
// It sets both OutputType and OutputTypeID.
func (m *Method) WithOutput(msg *Message) *Method {
	m.outputType = msg
	if msg != nil {
		m.outputTypeID = msg.id
	}
	return m
}

// WithInputTypeID sets the input type ID for the method.
func (m *Method) WithInputTypeID(id string) *Method {
	m.inputTypeID = id
	return m
}

// WithOutputTypeID sets the output type ID for the method.
func (m *Method) WithOutputTypeID(id string) *Method {
	m.outputTypeID = id
	return m
}

// WithPathTemplate sets the path template for the first binding.
func (m *Method) WithPathTemplate(pt *PathTemplate) *Method {
	m.ensureFirstBinding().pathTemplate = pt
	return m
}

// WithBindings sets the HTTP bindings for the method.
// It initializes PathInfo if it is nil.
func (m *Method) WithBindings(bindings ...*PathBinding) *Method {
	if m.pathInfo == nil {
		m.pathInfo = &PathInfo{}
	}
	m.pathInfo.bindings = bindings
	return m
}

// WithBodyFieldPath sets the body field path for the method.
// It initializes PathInfo if it is nil.
func (m *Method) WithBodyFieldPath(path string) *Method {
	if m.pathInfo == nil {
		m.pathInfo = &PathInfo{}
	}
	m.pathInfo.bodyFieldPath = path
	return m
}

// WithQueryParameters sets the query parameters on the first binding of the method.
// It initializes PathInfo if it is nil, and creates a binding if none exists.
func (m *Method) WithQueryParameters(params ...string) *Method {
	b := m.ensureFirstBinding()
	b.queryParameters = make(map[string]bool, len(params))
	for _, p := range params {
		b.queryParameters[p] = true
	}
	return m
}

// WithSourceMethod defines mixin methods.
func (m *Method) WithSourceMethod(source *Method) *Method {
	m.WithOutput(source.outputType).WithInput(source.inputType)
	m.sourceService = source.service
	if m.sourceService != nil {
		m.sourceServiceID = m.sourceService.id
	}
	m.pathInfo = source.pathInfo
	return m
}

// WithSourceServiceID sets the source service ID for the method.
func (m *Method) WithSourceServiceID(id string) *Method {
	m.sourceServiceID = id
	return m
}

// WithPagination sets the page token field for the request.
func (m *Method) WithPagination(pageToken *Field) *Method {
	m.isList = true
	m.pagination = pageToken
	return m
}

// WithOperationInfo sets the LRO information and flag.
func (m *Method) WithOperationInfo(info *OperationInfo) *Method {
	m.isLRO = true
	m.operationInfo = info
	return m
}

// WithBidiStreaming sets the method as bidirectional streaming.
func (m *Method) WithBidiStreaming() *Method {
	m.clientSideStreaming = true
	m.serverSideStreaming = true
	return m
}

// WithServerSideStreaming sets the method as a server-side streaming
// method. This method will not have client side streaming.
func (m *Method) WithServerSideStreaming() *Method {
	m.clientSideStreaming = false
	m.serverSideStreaming = true
	return m
}

// WithClientSideStreaming sets the method as a client-side streaming
// method. This method will not have server-side streaming.
func (m *Method) WithClientSideStreaming() *Method {
	m.clientSideStreaming = true
	m.serverSideStreaming = false
	return m
}

// WithDiscoveryLro sets the discovery LRO information.
func (m *Method) WithDiscoveryLro(info *DiscoveryLro) *Method {
	m.discoveryLro = info
	return m
}

// WithSignatures adds method signatures.
//
// A method signature typically maps to an overloads with a subset of
// the request fields.
func (m *Method) WithSignatures(signatures ...*MethodSignature) *Method {
	if m.inputType == nil {
		panic(fmt.Sprintf("missing InputType for method: %s", m.name))
	}
	for _, s := range signatures {
		s.method = m
		for _, name := range s.names {
			for _, f := range m.inputType.fields {
				if name != f.name {
					continue
				}
				s.fields = append(s.fields, f)
				break
			}
		}
		m.signatures = append(m.signatures, s)
	}
	return m
}

// WithRawSignatures sets the method signatures directly without resolving fields.
func (m *Method) WithRawSignatures(signatures ...*MethodSignature) *Method {
	if m.signatures == nil {
		m.signatures = []*MethodSignature{}
	}
	m.signatures = append(m.signatures, signatures...)
	return m
}

// WithDeprecated sets whether the method is deprecated.
func (m *Method) WithDeprecated(deprecated bool) *Method {
	m.deprecated = deprecated
	return m
}

// WithSampleInfo sets the sample info for the method.
func (m *Method) WithSampleInfo(sampleInfo *SampleInfo) *Method {
	m.sampleInfo = sampleInfo
	return m
}

// WithService sets the service for the method.
func (m *Method) WithService(service *Service) *Method {
	m.service = service
	return m
}

// WithModel sets the API model for the method.
func (m *Method) WithModel(model *API) *Method {
	m.model = model
	return m
}

// WithIsLroPoller marks the method as an LRO poller.
func (m *Method) WithIsLroPoller() *Method {
	m.isLroPoller = true
	return m
}

// WithIsSimple sets whether the method is simple.
func (m *Method) WithIsSimple(isSimple bool) *Method {
	m.isSimple = isSimple
	return m
}

// WithIsLRO sets whether the method is an LRO.
func (m *Method) WithIsLRO(isLRO bool) *Method {
	m.isLRO = isLRO
	return m
}

// WithIsList sets whether the method is a list method.
func (m *Method) WithIsList(isList bool) *Method {
	m.isList = isList
	return m
}

// WithIsStreaming sets whether the method is streaming.
func (m *Method) WithIsStreaming(isStreaming bool) *Method {
	m.isStreaming = isStreaming
	return m
}

// WithIsAIPStandard sets whether the method is an AIP standard method.
func (m *Method) WithIsAIPStandard(isStandard bool) *Method {
	m.isAIPStandard = isStandard
	return m
}

// WithIsAIPStandardGet sets whether the method is an AIP standard get method.
func (m *Method) WithIsAIPStandardGet(isGet bool) *Method {
	m.isAIPStandardGet = isGet
	return m
}

// WithIsAIPStandardDelete sets whether the method is an AIP standard delete method.
func (m *Method) WithIsAIPStandardDelete(isDelete bool) *Method {
	m.isAIPStandardDelete = isDelete
	return m
}

// WithIsAIPStandardUndelete sets whether the method is an AIP standard undelete method.
func (m *Method) WithIsAIPStandardUndelete(isUndelete bool) *Method {
	m.isAIPStandardUndelete = isUndelete
	return m
}

// WithIsAIPStandardCreate sets whether the method is an AIP standard create method.
func (m *Method) WithIsAIPStandardCreate(isCreate bool) *Method {
	m.isAIPStandardCreate = isCreate
	return m
}

// WithIsAIPStandardUpdate sets whether the method is an AIP standard update method.
func (m *Method) WithIsAIPStandardUpdate(isUpdate bool) *Method {
	m.isAIPStandardUpdate = isUpdate
	return m
}

// WithIsAIPStandardList sets whether the method is an AIP standard list method.
func (m *Method) WithIsAIPStandardList(isList bool) *Method {
	m.isAIPStandardList = isList
	return m
}

// ReturnEmpty marks the method as returning an empty response.
func (m *Method) ReturnEmpty() *Method {
	m.returnsEmpty = true
	return m
}

// NewTestOneOf creates a OneOf with defaults for testing.
func NewTestOneOf(name string) *OneOf {
	return NewOneOfBuilder().
		WithName(name).
		WithID(fmt.Sprintf(".test.%s", name)).
		Build()
}

// WithFields adds fields to a oneof group.
//
// When the group is included in a message, all its fields are included into the
// message too.
func (o *OneOf) WithFields(fields ...*Field) *OneOf {
	for _, f := range fields {
		f.isOneOf = true
		f.group = o
		o.fields = append(o.fields, f)
	}
	return o
}

// WithDocumentation sets the documentation for the oneof group.
func (o *OneOf) WithDocumentation(doc string) *OneOf {
	o.documentation = doc
	return o
}

// WithID overrides the oneof's ID.
func (o *OneOf) WithID(id string) *OneOf {
	o.id = id
	return o
}

// NewTestField creates a field with defaults for testing.
// JSONName is automatically camelCased.
func NewTestField(name string) *Field {
	return NewFieldBuilder().
		WithName(name).
		WithJSONName(strcase.ToLowerCamel(name)).
		WithID(fmt.Sprintf(".test.%s", name)).
		Build()
}

// WithID overrides the field's ID.
func (f *Field) WithID(id string) *Field {
	f.id = id
	return f
}

// WithOneOf marks the field as belonging to a oneof.
func (f *Field) WithOneOf() *Field {
	f.isOneOf = true
	return f
}

// WithType sets the type of the field.
func (f *Field) WithType(t Typez) *Field {
	f.typez = t
	return f
}

// WithRepeated marks the field as repeated.
func (f *Field) WithRepeated() *Field {
	f.repeated = true
	f.optional = false
	f.mapz = false
	return f
}

// WithOptional marks the field as optional.
func (f *Field) WithOptional() *Field {
	f.optional = true
	f.repeated = false
	f.mapz = false
	return f
}

// WithProto2Optional marks the field as a proto2 optional field.
func (f *Field) WithProto2Optional() *Field {
	f.proto2Optional = true
	return f
}

// WithMap marks the field as a map.
func (f *Field) WithMap() *Field {
	f.typez = TypezMap
	f.mapz = true
	f.repeated = false
	f.optional = false
	return f
}

// WithMapType sets the field's map type.
// It sets MapType, Typez=TypezMap, Map=true, and TypezID.
func (f *Field) WithMapType(m *Map) *Field {
	f.mapType = m
	f.typez = TypezMap
	f.mapz = true
	f.repeated = false
	f.optional = false
	if m != nil {
		f.typezID = m.id
	}
	return f
}

// WithRecursive marks the field as recursive.
func (f *Field) WithRecursive() *Field {
	f.recursive = true
	return f
}

// WithBehavior adds behavior(s) to the field.
func (f *Field) WithBehavior(behaviors ...FieldBehavior) *Field {
	f.behavior = append(f.behavior, behaviors...)
	return f
}

// WithMessageType sets the field's message type.
// It sets MessageType, Typez=TypezMessage, and TypezID.
func (f *Field) WithMessageType(msg *Message) *Field {
	f.messageType = msg
	f.typez = TypezMessage
	if msg != nil {
		f.typezID = msg.id
	}
	return f
}

// WithEnumType sets the field's enum type.
// It sets EnumType, Typez=TypezEnum, and TypezID.
func (f *Field) WithEnumType(enum *Enum) *Field {
	f.enumType = enum
	f.typez = TypezEnum
	if enum != nil {
		f.typezID = enum.id
	}
	return f
}

// WithAutoPopulated marks the field as auto-populated.
func (f *Field) WithAutoPopulated() *Field {
	f.autoPopulated = true
	return f
}

// WithResourceNamePattern sets the resource name pattern on the field.
func (f *Field) WithResourceNamePattern(pattern *ResourceNamePattern) *Field {
	f.resourceNamePattern = pattern
	return f
}

// WithTypezID sets the field's TypezID.
func (f *Field) WithTypezID(id string) *Field {
	f.typezID = id
	return f
}

// WithJSONName sets the JSON name of the field.
func (f *Field) WithJSONName(name string) *Field {
	f.jsonName = name
	return f
}

// WithSkipProtoConversion marks the field as skipping proto conversion.
func (f *Field) WithSkipProtoConversion() *Field {
	f.skipProtoConversion = true
	return f
}

// WithResourceReference sets the resource reference on a field.
func (f *Field) WithResourceReference(refType string) *Field {
	f.resourceReference = NewResourceReferenceBuilder().WithType(refType).Build()
	return f
}

// WithChildTypeReference sets the child type resource reference on a field.
func (f *Field) WithChildTypeReference(childType string) *Field {
	f.resourceReference = NewResourceReferenceBuilder().WithChildType(childType).Build()
	return f
}

// WithDeprecated sets whether the field is deprecated.
func (f *Field) WithDeprecated(deprecated bool) *Field {
	f.deprecated = deprecated
	return f
}

// WithDocumentation sets the documentation of the field.
func (f *Field) WithDocumentation(documentation string) *Field {
	f.documentation = documentation
	return f
}

// WithParent sets the parent message for the field.
func (f *Field) WithParent(parent *Message) *Field {
	f.parent = parent
	return f
}

// WithGroup sets the OneOf group for the field.
func (f *Field) WithGroup(group *OneOf) *Field {
	f.group = group
	return f
}

// NewTestResource creates a resource with defaults.
func NewTestResource(typez string) *Resource {
	return NewResourceBuilder().WithType(typez).Build()
}

// WithPatterns adds patterns to the resource.
func (r *Resource) WithPatterns(patterns ...ResourcePattern) *Resource {
	r.patterns = append(r.patterns, patterns...)
	return r
}

// WithSingular sets the singular name of the resource.
func (r *Resource) WithSingular(name string) *Resource {
	r.singular = name
	return r
}

// WithPlural sets the plural name of the resource.
func (r *Resource) WithPlural(name string) *Resource {
	r.plural = name
	return r
}

// WithSelf sets the Message that defines this resource.
func (r *Resource) WithSelf(self *Message) *Resource {
	r.self = self
	return r
}

// NewTestEnum creates an Enum with defaults for testing.
// Default package is "test".
func NewTestEnum(name string) *Enum {
	return NewEnumBuilder().WithName(name).Build().WithPackage("test")
}

// WithPackage sets the package for the enum and updates its ID.
func (e *Enum) WithPackage(pkg string) *Enum {
	e.packagez = pkg
	e.id = fmt.Sprintf(".%s.%s", pkg, e.name)
	return e
}

// WithID overrides the enum's ID.
func (e *Enum) WithID(id string) *Enum {
	e.id = id
	return e
}

// WithDocumentation sets the documentation for the enum.
func (e *Enum) WithDocumentation(doc string) *Enum {
	e.documentation = doc
	return e
}

// WithDeprecated sets whether the enum is deprecated.
func (e *Enum) WithDeprecated(deprecated bool) *Enum {
	e.deprecated = deprecated
	return e
}

// WithParent sets the parent message for the enum and updates its ID.
func (e *Enum) WithParent(parent *Message) *Enum {
	e.parent = parent
	if parent != nil {
		e.packagez = parent.packagez
		if strings.HasPrefix(e.id, ".test.") || e.id == "" {
			e.id = fmt.Sprintf("%s.%s", parent.id, e.name)
		}
		for _, v := range e.values {
			v.parent = e
			if strings.HasPrefix(v.id, ".test.") || v.id == "" {
				v.id = fmt.Sprintf("%s.%s", e.id, v.name)
			}
		}
	}
	return e
}

// WithValues adds values to the enum, setting their Parent, ID, and populating UniqueNumberValues.
func (e *Enum) WithValues(values ...*EnumValue) *Enum {
	for _, v := range values {
		v.parent = e
		if strings.HasPrefix(v.id, ".test.") || v.id == "" {
			v.id = fmt.Sprintf("%s.%s", e.id, v.name)
		}
	}
	e.values = append(e.values, values...)
	seen := make(map[int32]bool)
	for _, v := range e.uniqueNumberValues {
		seen[v.number] = true
	}
	for _, v := range values {
		if !seen[v.number] {
			e.uniqueNumberValues = append(e.uniqueNumberValues, v)
			seen[v.number] = true
		}
	}
	return e
}

// WithUniqueNumberValues overrides UniqueNumberValues on the enum.
func (e *Enum) WithUniqueNumberValues(values ...*EnumValue) *Enum {
	e.uniqueNumberValues = values
	return e
}

// NewTestEnumValue creates an EnumValue with defaults for testing.
func NewTestEnumValue(name string, number int32) *EnumValue {
	return NewEnumValueBuilder().
		WithName(name).
		WithNumber(number).
		WithID(fmt.Sprintf(".test.%s", name)).
		Build()
}

// WithID overrides the enum value's ID.
func (ev *EnumValue) WithID(id string) *EnumValue {
	ev.id = id
	return ev
}

// WithDocumentation sets the documentation for the enum value.
func (ev *EnumValue) WithDocumentation(doc string) *EnumValue {
	ev.documentation = doc
	return ev
}

// WithDeprecated sets whether the enum value is deprecated.
func (ev *EnumValue) WithDeprecated(deprecated bool) *EnumValue {
	ev.deprecated = deprecated
	return ev
}

// ParseTemplateForTest converts a string literal into a []PathSegment slice for testing purposes.
func ParseTemplateForTest(template string) []PathSegment {
	var segments []PathSegment
	trimmed, hasDoubleSlash := strings.CutPrefix(template, "//")
	parts := strings.Split(trimmed, "/")

	host := parts[0]
	if hasDoubleSlash {
		host = "//" + parts[0]
	}
	segments = append(segments, *NewPathSegmentBuilder().WithLiteral(host).Build())

	for _, part := range parts[1:] {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			fieldPath := strings.Split(part[1:len(part)-1], ".")
			segments = append(segments, *NewPathSegmentBuilder().WithVariable(NewPathVariableBuilder(fieldPath...).Build()).Build())
		} else {
			segments = append(segments, *NewPathSegmentBuilder().WithLiteral(part).Build())
		}
	}
	return segments
}

// NewTestMethodSignature creates a MethodSignature with parameter names.
func NewTestMethodSignature(names ...string) *MethodSignature {
	return NewMethodSignatureBuilder().WithNames(names...).Build()
}

// WithFields appends the resolved fields to the signature.
func (s *MethodSignature) WithFields(fields ...*Field) *MethodSignature {
	s.fields = append(s.fields, fields...)
	return s
}

// NewTestResourceNamePattern creates a ResourceNamePattern with segments.
func NewTestResourceNamePattern(segments ...ResourceNameSegment) *ResourceNamePattern {
	return NewResourceNamePattern(segments...)
}

// NewTestRoutingInfo creates a RoutingInfo struct with variants.
func NewTestRoutingInfo(name string, variants ...*RoutingInfoVariant) *RoutingInfo {
	return NewRoutingInfoBuilder().WithName(name).WithVariants(variants...).Build()
}

// NewTestRoutingInfoVariant creates a RoutingInfoVariant with a field path.
func NewTestRoutingInfoVariant(fieldPath ...string) *RoutingInfoVariant {
	return NewRoutingInfoVariantBuilder().WithFieldPath(fieldPath...).Build()
}

// NewTestOperationInfo creates an OperationInfo with response and metadata type IDs.
func NewTestOperationInfo(responseTypeID, metadataTypeID string) *OperationInfo {
	return NewOperationInfoBuilder().
		WithResponseTypeID(responseTypeID).
		WithMetadataTypeID(metadataTypeID).
		Build()
}

// NewTestDiscoveryLro creates a DiscoveryLro with polling path parameters.
func NewTestDiscoveryLro(pollingParams ...string) *DiscoveryLro {
	return NewDiscoveryLroBuilder().
		WithPollingPathParameters(pollingParams...).
		Build()
}

// NewTestSampleInfo creates a SampleInfo initialized with a resource name field.
func NewTestSampleInfo(resourceNameField *Field) *SampleInfo {
	return NewSampleInfoBuilder().WithResourceNameField(resourceNameField).Build()
}

// NewTestPaginationInfo creates a PaginationInfo for pagination responses.
func NewTestPaginationInfo(nextPageToken, pageableItem *Field) *PaginationInfo {
	return NewPaginationInfo(nextPageToken, pageableItem)
}
