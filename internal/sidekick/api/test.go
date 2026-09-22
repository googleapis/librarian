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
	}

	var indexMessage func(m *Message)
	indexMessage = func(m *Message) {
		model.messageByID[m.ID] = m
		if m.Resource != nil {
			model.resourceByType[m.Resource.Type] = m.Resource
		}
		for _, e := range m.Enums {
			model.enumByID[e.ID] = e
			e.Parent = m
			for _, ev := range e.Values {
				ev.Parent = e
			}
		}
		for _, child := range m.Messages {
			child.Parent = m
			indexMessage(child)
		}
	}
	for _, m := range messages {
		model.PackageName = m.Package
		indexMessage(m)
	}
	for _, e := range enums {
		model.PackageName = e.Package
		model.enumByID[e.ID] = e
	}
	for _, s := range services {
		model.PackageName = s.Package
		model.serviceByID[s.ID] = s
		for _, m := range s.Methods {
			model.methodByID[m.ID] = m
		}
	}
	for _, m := range messages {
		parentID := parentName(m.ID)
		parent := model.messageByID[parentID]
		if parent != nil {
			m.Parent = parent
			if !slices.Contains(parent.Messages, m) {
				parent.Messages = append(parent.Messages, m)
			}
		}
	}
	for _, e := range enums {
		parent := model.messageByID[parentName(e.ID)]
		if parent != nil {
			e.Parent = parent
			if !slices.Contains(parent.Enums, e) {
				parent.Enums = append(parent.Enums, e)
			}
		}
		for _, ev := range e.Values {
			ev.Parent = e
		}
	}

	model.LoadWellKnownTypes()
	return model
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
	return (&Message{Name: name}).WithPackage("test")
}

// WithPackage sets the package for the message and updates its ID.
func (m *Message) WithPackage(pkg string) *Message {
	m.Package = pkg
	m.ID = fmt.Sprintf(".%s.%s", pkg, m.Name)
	return m
}

// WithID overrides the message's ID.
func (m *Message) WithID(id string) *Message {
	m.ID = id
	return m
}

// WithFields adds fields to the message and updates their parent/ID.
func (m *Message) WithFields(fields ...*Field) *Message {
	for _, f := range fields {
		f.Parent = m
		// If field ID is generic/default, re-scope it to the message.
		if strings.HasPrefix(f.ID, ".test.") || f.ID == "" {
			f.ID = fmt.Sprintf("%s.%s", m.ID, f.Name)
		}
	}
	m.Fields = append(m.Fields, fields...)
	return m
}

// WithOneOfs adds oneofs to the message and updates their parent/ID.
func (m *Message) WithOneOfs(oneofs ...*OneOf) *Message {
	for _, o := range oneofs {
		// If oneof ID is generic/default, re-scope it to the message.
		if strings.HasPrefix(o.ID, ".test.") || o.ID == "" {
			o.ID = fmt.Sprintf("%s.%s", m.ID, o.Name)
		}
		m.OneOfs = append(m.OneOfs, o)
		m.WithFields(o.Fields...)
	}
	return m
}

func (m *Message) withParentRecursive(parent *Message) {
	m.Parent = parent
	if m.Parent == nil {
		return
	}
	m.Package = parent.Package
	if strings.HasPrefix(m.ID, ".test.") || m.ID == "" {
		m.ID = fmt.Sprintf("%s.%s", parent.ID, m.Name)
	}
	for _, c := range m.Messages {
		c.withParentRecursive(m)
	}
	for _, e := range m.Enums {
		_ = e.WithParent(m)
	}
}

// WithMessages adds nested messages to the message and updates their parent/ID.
func (m *Message) WithMessages(messages ...*Message) *Message {
	for _, child := range messages {
		child.withParentRecursive(m)
	}
	m.Messages = append(m.Messages, messages...)
	return m
}

// WithEnums adds enums to the message and updates their parent/ID.
func (m *Message) WithEnums(enums ...*Enum) *Message {
	for _, e := range enums {
		e.WithParent(m)
	}
	m.Enums = append(m.Enums, enums...)
	return m
}

// WithPagination items and page token fields for a pagination response.
func (m *Message) WithPagination(nextPageToken *Field, items *Field) *Message {
	if nextPageToken.Parent != m {
		m.WithFields(nextPageToken)
	}
	if items.Parent != m {
		m.WithFields(items)
	}
	m.Pagination = &PaginationInfo{
		NextPageToken: nextPageToken,
		PageableItem:  items,
	}
	return m
}

// WithResource sets the resource definition on the message.
func (m *Message) WithResource(resource *Resource) *Message {
	m.Resource = resource
	if resource != nil {
		resource.Self = m
	}
	return m
}

// WithDeprecated sets whether the message is deprecated.
func (m *Message) WithDeprecated(deprecated bool) *Message {
	m.Deprecated = deprecated
	return m
}

// WithDocumentation sets the documentation for the message.
func (m *Message) WithDocumentation(doc string) *Message {
	m.Documentation = doc
	return m
}

// WithIsMap sets whether the message represents a map entry.
func (m *Message) WithIsMap() *Message {
	m.IsMap = true
	return m
}

// NewTestService creates a service with defaults for testing.
// Default package is "test".
func NewTestService(name string) *Service {
	return (&Service{Name: name}).WithPackage("test")
}

// WithPackage sets the package for the service and updates its ID.
func (s *Service) WithPackage(pkg string) *Service {
	s.Package = pkg
	s.ID = fmt.Sprintf(".%s.%s", pkg, s.Name)
	return s
}

// WithMethods adds methods to the service and updates their ID/Service.
func (s *Service) WithMethods(methods ...*Method) *Service {
	for _, m := range methods {
		m.Service = s
		// If method ID is generic/default, re-scope it to the service.
		if strings.HasPrefix(m.ID, ".test.") || m.ID == "" {
			m.ID = fmt.Sprintf("%s.%s", s.ID, m.Name)
		}
	}
	s.Methods = append(s.Methods, methods...)
	return s
}

// WithDeprecated sets whether the service is deprecated.
func (s *Service) WithDeprecated(deprecated bool) *Service {
	s.Deprecated = deprecated
	return s
}

// WithDocumentation sets the documentation for the service.
func (s *Service) WithDocumentation(doc string) *Service {
	s.Documentation = doc
	return s
}

// NewTestMethod creates a method with defaults for testing.
// Default package is "test" (implies ID .test.Name).
func NewTestMethod(name string) *Method {
	return &Method{
		Name: name,
		ID:   fmt.Sprintf(".test.%s", name),
		PathInfo: &PathInfo{
			Bindings: []*PathBinding{{}},
		},
	}
}

// WithDocumentation sets the documentation for the method.
func (m *Method) WithDocumentation(doc string) *Method {
	m.Documentation = doc
	return m
}

func (m *Method) ensureFirstBinding() *PathBinding {
	if m.PathInfo == nil {
		m.PathInfo = &PathInfo{}
	}
	if len(m.PathInfo.Bindings) == 0 {
		m.PathInfo.Bindings = append(m.PathInfo.Bindings, &PathBinding{})
	}
	return m.PathInfo.Bindings[0]
}

// WithVerb sets the HTTP verb for the first binding.
func (m *Method) WithVerb(verb string) *Method {
	m.ensureFirstBinding().Verb = verb
	return m
}

// WithInput sets the input type message for the method.
// It sets both InputType and InputTypeID.
func (m *Method) WithInput(msg *Message) *Method {
	m.InputType = msg
	if msg != nil {
		m.InputTypeID = msg.ID
	}
	return m
}

// WithOutput sets the output type message for the method.
// It sets both OutputType and OutputTypeID.
func (m *Method) WithOutput(msg *Message) *Method {
	m.OutputType = msg
	if msg != nil {
		m.OutputTypeID = msg.ID
	}
	return m
}

// WithPathTemplate sets the path template for the first binding.
func (m *Method) WithPathTemplate(pt *PathTemplate) *Method {
	m.ensureFirstBinding().PathTemplate = pt
	return m
}

// WithBindings sets the HTTP bindings for the method.
// It initializes PathInfo if it is nil.
func (m *Method) WithBindings(bindings ...*PathBinding) *Method {
	if m.PathInfo == nil {
		m.PathInfo = &PathInfo{}
	}
	m.PathInfo.Bindings = bindings
	return m
}

// WithBodyFieldPath sets the body field path for the method.
// It initializes PathInfo if it is nil.
func (m *Method) WithBodyFieldPath(path string) *Method {
	if m.PathInfo == nil {
		m.PathInfo = &PathInfo{}
	}
	m.PathInfo.BodyFieldPath = path
	return m
}

// WithQueryParameters sets the query parameters on the first binding of the method.
// It initializes PathInfo if it is nil, and creates a binding if none exists.
func (m *Method) WithQueryParameters(params map[string]bool) *Method {
	m.ensureFirstBinding().QueryParameters = params
	return m
}

// WithSourceMethod defines mixin methods.
func (m *Method) WithSourceMethod(source *Method) *Method {
	m.WithOutput(source.OutputType).WithInput(source.InputType)
	m.SourceService = source.Service
	if m.SourceService != nil {
		m.SourceServiceID = m.SourceService.ID
	}
	m.PathInfo = source.PathInfo
	return m
}

// WithPagination sets the page token field for the request.
func (m *Method) WithPagination(pageToken *Field) *Method {
	m.IsList = true
	m.Pagination = pageToken
	return m
}

// WithOperationInfo sets the LRO information and flag.
func (m *Method) WithOperationInfo(info *OperationInfo) *Method {
	m.IsLRO = true
	m.OperationInfo = info
	return m
}

// WithBidiStreaming sets the method as bidirectional streaming.
func (m *Method) WithBidiStreaming() *Method {
	m.ClientSideStreaming = true
	m.ServerSideStreaming = true
	return m
}

// WithServerSideStreaming sets the method as a server-side streaming
// method. This method will not have client side streaming.
func (m *Method) WithServerSideStreaming() *Method {
	m.ClientSideStreaming = false
	m.ServerSideStreaming = true
	return m
}

// WithClientSideStreaming sets the method as a client-side streaming
// method. This method will not have server-side streaming.
func (m *Method) WithClientSideStreaming() *Method {
	m.ClientSideStreaming = true
	m.ServerSideStreaming = false
	return m
}

// WithDiscoveryLro sets the discovery LRO information.
func (m *Method) WithDiscoveryLro(info *DiscoveryLro) *Method {
	m.DiscoveryLro = info
	return m
}

// WithSignatures adds method signatures.
//
// A method signature typically maps to an overloads with a subset of
// the request fields.
func (m *Method) WithSignatures(signatures ...*MethodSignature) *Method {
	if m.InputType == nil {
		panic(fmt.Sprintf("missing InputType for method: %s", m.Name))
	}
	for _, s := range signatures {
		s.Method = m
		for _, name := range s.Names {
			for _, f := range m.InputType.Fields {
				if name != f.Name {
					continue
				}
				s.Fields = append(s.Fields, f)
				break
			}
		}
		m.Signatures = append(m.Signatures, s)
	}
	return m
}

// WithDeprecated sets whether the method is deprecated.
func (m *Method) WithDeprecated(deprecated bool) *Method {
	m.Deprecated = deprecated
	return m
}

// NewTestPathBinding creates a PathBinding with the given verb and path template.
func NewTestPathBinding(verb string, pt *PathTemplate) *PathBinding {
	return &PathBinding{
		Verb:         verb,
		PathTemplate: pt,
	}
}

// WithVerb sets the HTTP verb for the binding.
func (b *PathBinding) WithVerb(verb string) *PathBinding {
	b.Verb = verb
	return b
}

// WithPathTemplate sets the path template for the binding.
func (b *PathBinding) WithPathTemplate(pt *PathTemplate) *PathBinding {
	b.PathTemplate = pt
	return b
}

// WithQueryParameters sets the query parameters for the binding.
func (b *PathBinding) WithQueryParameters(params map[string]bool) *PathBinding {
	b.QueryParameters = params
	return b
}

// NewTestOneOf creates a OneOf with defaults for testing.
func NewTestOneOf(name string) *OneOf {
	return &OneOf{
		Name: name,
		ID:   fmt.Sprintf(".test.%s", name),
	}
}

// WithFields adds fields to a oneof group.
//
// When the group is included in a message, all its fields are included into the
// message too.
func (o *OneOf) WithFields(fields ...*Field) *OneOf {
	for _, f := range fields {
		f.IsOneOf = true
		f.Group = o
		o.Fields = append(o.Fields, f)
	}
	return o
}

// WithDocumentation sets the documentation for the oneof group.
func (o *OneOf) WithDocumentation(doc string) *OneOf {
	o.Documentation = doc
	return o
}

// NewTestField creates a field with defaults for testing.
// JSONName is automatically camelCased.
func NewTestField(name string) *Field {
	return &Field{
		Name:     name,
		JSONName: strcase.ToLowerCamel(name),
		ID:       fmt.Sprintf(".test.%s", name),
	}
}

// WithType sets the type of the field.
func (f *Field) WithType(t Typez) *Field {
	f.Typez = t
	return f
}

// WithRepeated marks the field as repeated.
func (f *Field) WithRepeated() *Field {
	f.Repeated = true
	f.Optional = false
	f.Map = false
	return f
}

// WithOptional marks the field as optional.
func (f *Field) WithOptional() *Field {
	f.Optional = true
	f.Repeated = false
	f.Map = false
	return f
}

// WithMap marks the field as a map.
func (f *Field) WithMap() *Field {
	f.Map = true
	f.Repeated = false
	f.Optional = false
	return f
}

// WithRecursive marks the field as recursive.
func (f *Field) WithRecursive() *Field {
	f.Recursive = true
	return f
}

// WithBehavior adds behavior(s) to the field.
func (f *Field) WithBehavior(behaviors ...FieldBehavior) *Field {
	f.Behavior = append(f.Behavior, behaviors...)
	return f
}

// WithMessageType sets the field's message type.
// It sets MessageType, Typez=TypezMessage, and TypezID.
func (f *Field) WithMessageType(msg *Message) *Field {
	f.MessageType = msg
	f.Typez = TypezMessage
	if msg != nil {
		f.TypezID = msg.ID
	}
	return f
}

// WithTypezID sets the field's TypezID.
func (f *Field) WithTypezID(id string) *Field {
	f.TypezID = id
	return f
}

// WithJSONName sets the JSON name of the field.
func (f *Field) WithJSONName(name string) *Field {
	f.JSONName = name
	return f
}

// WithSkipProtoConversion marks the field as skipping proto conversion.
func (f *Field) WithSkipProtoConversion() *Field {
	f.SkipProtoConversion = true
	return f
}

// WithResourceReference sets the resource reference on a field.
func (f *Field) WithResourceReference(refType string) *Field {
	f.ResourceReference = &ResourceReference{Type: refType}
	return f
}

// WithChildTypeReference sets the child type resource reference on a field.
func (f *Field) WithChildTypeReference(childType string) *Field {
	f.ResourceReference = &ResourceReference{ChildType: childType}
	return f
}

// WithDeprecated sets whether the field is deprecated.
func (f *Field) WithDeprecated(deprecated bool) *Field {
	f.Deprecated = deprecated
	return f
}

// WithDocumentation sets the documentation of the field.
func (f *Field) WithDocumentation(documentation string) *Field {
	f.Documentation = documentation
	return f
}

// NewTestResource creates a resource with defaults.
func NewTestResource(typez string) *Resource {
	return &Resource{
		Type: typez,
	}
}

// WithPatterns adds patterns to the resource.
func (r *Resource) WithPatterns(patterns ...ResourcePattern) *Resource {
	r.Patterns = append(r.Patterns, patterns...)
	return r
}

// WithSingular sets the singular name of the resource.
func (r *Resource) WithSingular(name string) *Resource {
	r.Singular = name
	return r
}

// WithPlural sets the plural name of the resource.
func (r *Resource) WithPlural(name string) *Resource {
	r.Plural = name
	return r
}

// NewTestEnum creates an Enum with defaults for testing.
// Default package is "test".
func NewTestEnum(name string) *Enum {
	return (&Enum{Name: name}).WithPackage("test")
}

// WithPackage sets the package for the enum and updates its ID.
func (e *Enum) WithPackage(pkg string) *Enum {
	e.Package = pkg
	e.ID = fmt.Sprintf(".%s.%s", pkg, e.Name)
	return e
}

// WithID overrides the enum's ID.
func (e *Enum) WithID(id string) *Enum {
	e.ID = id
	return e
}

// WithDocumentation sets the documentation for the enum.
func (e *Enum) WithDocumentation(doc string) *Enum {
	e.Documentation = doc
	return e
}

// WithDeprecated sets whether the enum is deprecated.
func (e *Enum) WithDeprecated(deprecated bool) *Enum {
	e.Deprecated = deprecated
	return e
}

// WithParent sets the parent message for the enum and updates its ID.
func (e *Enum) WithParent(parent *Message) *Enum {
	e.Parent = parent
	if parent != nil {
		e.Package = parent.Package
		if strings.HasPrefix(e.ID, ".test.") || e.ID == "" {
			e.ID = fmt.Sprintf("%s.%s", parent.ID, e.Name)
		}
		for _, v := range e.Values {
			v.Parent = e
			if strings.HasPrefix(v.ID, ".test.") || v.ID == "" {
				v.ID = fmt.Sprintf("%s.%s", e.ID, v.Name)
			}
		}
	}
	return e
}

// WithValues adds values to the enum, setting their Parent, ID, and populating UniqueNumberValues.
func (e *Enum) WithValues(values ...*EnumValue) *Enum {
	for _, v := range values {
		v.Parent = e
		if strings.HasPrefix(v.ID, ".test.") || v.ID == "" {
			v.ID = fmt.Sprintf("%s.%s", e.ID, v.Name)
		}
	}
	e.Values = append(e.Values, values...)
	seen := make(map[int32]bool)
	for _, v := range e.UniqueNumberValues {
		seen[v.Number] = true
	}
	for _, v := range values {
		if !seen[v.Number] {
			e.UniqueNumberValues = append(e.UniqueNumberValues, v)
			seen[v.Number] = true
		}
	}
	return e
}

// WithUniqueNumberValues overrides UniqueNumberValues on the enum.
func (e *Enum) WithUniqueNumberValues(values ...*EnumValue) *Enum {
	e.UniqueNumberValues = values
	return e
}

// NewTestEnumValue creates an EnumValue with defaults for testing.
func NewTestEnumValue(name string, number int32) *EnumValue {
	return &EnumValue{
		Name:   name,
		Number: number,
		ID:     fmt.Sprintf(".test.%s", name),
	}
}

// WithID overrides the enum value's ID.
func (ev *EnumValue) WithID(id string) *EnumValue {
	ev.ID = id
	return ev
}

// WithDocumentation sets the documentation for the enum value.
func (ev *EnumValue) WithDocumentation(doc string) *EnumValue {
	ev.Documentation = doc
	return ev
}

// WithDeprecated sets whether the enum value is deprecated.
func (ev *EnumValue) WithDeprecated(deprecated bool) *EnumValue {
	ev.Deprecated = deprecated
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
	segments = append(segments, PathSegment{Literal: host})

	for _, part := range parts[1:] {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			fieldPath := strings.Split(part[1:len(part)-1], ".")
			segments = append(segments, PathSegment{Variable: &PathVariable{FieldPath: fieldPath}})
		} else {
			segments = append(segments, PathSegment{Literal: part})
		}
	}
	return segments
}
