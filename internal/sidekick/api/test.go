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
	"strings"

	"github.com/iancoleman/strcase"
)

// NewTestAPI creates a new test API.
func NewTestAPI(messages []*Message, enums []*Enum, services []*Service) *API {
	packageName := ""
	state := &APIState{
		MessageByID:    make(map[string]*Message),
		MethodByID:     make(map[string]*Method),
		EnumByID:       make(map[string]*Enum),
		ServiceByID:    make(map[string]*Service),
		ResourceByType: make(map[string]*Resource),
	}
	for _, m := range messages {
		packageName = m.Package
		state.MessageByID[m.ID] = m
		if m.Resource != nil {
			state.ResourceByType[m.Resource.Type] = m.Resource
		}
	}
	for _, e := range enums {
		packageName = e.Package
		state.EnumByID[e.ID] = e
	}
	for _, s := range services {
		packageName = s.Package
		state.ServiceByID[s.ID] = s
		for _, m := range s.Methods {
			state.MethodByID[m.ID] = m
		}
	}
	for _, m := range messages {
		parentID := parentName(m.ID)
		parent := state.MessageByID[parentID]
		if parent != nil {
			m.Parent = parent
			parent.Messages = append(parent.Messages, m)
		}
	}
	for _, e := range enums {
		parent := state.MessageByID[parentName(e.ID)]
		if parent != nil {
			e.Parent = parent
			parent.Enums = append(parent.Enums, e)
		}
		for _, ev := range e.Values {
			ev.Parent = e
		}
	}

	model := &API{
		Name:        "Test",
		PackageName: packageName,
		Messages:    messages,
		Enums:       enums,
		Services:    services,
		State:       state,
	}
	model.LoadWellKnownTypes()
	return model
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

// WithResource sets the resource definition on the message.
func (m *Message) WithResource(resource *Resource) *Message {
	m.Resource = resource
	if resource != nil {
		resource.Self = m
	}
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

// WithVerb sets the HTTP verb for the first binding.
func (m *Method) WithVerb(verb string) *Method {
	if len(m.PathInfo.Bindings) > 0 {
		m.PathInfo.Bindings[0].Verb = verb
	}
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
	if len(m.PathInfo.Bindings) > 0 {
		m.PathInfo.Bindings[0].PathTemplate = pt
	}
	return m
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
	return f
}

// WithMap marks the field as a map.
func (f *Field) WithMap() *Field {
	f.Map = true
	return f
}

// WithBehavior adds behavior(s) to the field.
func (f *Field) WithBehavior(behaviors ...FieldBehavior) *Field {
	f.Behavior = append(f.Behavior, behaviors...)
	return f
}

// WithMessageType sets the field's message type.
// It sets MessageType, Typez=MESSAGE_TYPE, and TypezID.
func (f *Field) WithMessageType(msg *Message) *Field {
	f.MessageType = msg
	f.Typez = MESSAGE_TYPE
	if msg != nil {
		f.TypezID = msg.ID
	}
	return f
}

// WithResourceReference sets the resource reference on a field.
func (f *Field) WithResourceReference(refType string) *Field {
	f.ResourceReference = &ResourceReference{Type: refType}
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
func (r *Resource) WithSingular(singular string) *Resource {
	r.Singular = singular
	return r
}
