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

// ServiceBuilder constructs a [Service] instance.
type ServiceBuilder struct {
	s Service
}

// NewServiceBuilder returns a new [ServiceBuilder].
func NewServiceBuilder() *ServiceBuilder {
	return &ServiceBuilder{}
}

// Build returns the constructed [Service].
func (b *ServiceBuilder) Build() *Service {
	return &b.s
}

// WithDocumentation sets the documentation for the service.
func (b *ServiceBuilder) WithDocumentation(doc string) *ServiceBuilder {
	b.s.documentation = doc
	return b
}

// WithName sets the name of the service.
func (b *ServiceBuilder) WithName(name string) *ServiceBuilder {
	b.s.name = name
	return b
}

// WithID sets the unique identifier for the service.
func (b *ServiceBuilder) WithID(id string) *ServiceBuilder {
	b.s.id = id
	return b
}

// WithDeprecated sets whether the service is deprecated.
func (b *ServiceBuilder) WithDeprecated(deprecated bool) *ServiceBuilder {
	b.s.deprecated = deprecated
	return b
}

// WithMethods sets the methods associated with the service.
func (b *ServiceBuilder) WithMethods(methods ...*Method) *ServiceBuilder {
	b.s.methods = methods
	return b
}

// AddMethod appends a method to the service.
func (b *ServiceBuilder) AddMethod(method *Method) *ServiceBuilder {
	b.s.methods = append(b.s.methods, method)
	return b
}

// Methods returns the methods currently added to the builder.
func (b *ServiceBuilder) Methods() []*Method {
	return slices.Clone(b.s.methods)
}

// ID returns the ID currently set on the builder.
func (b *ServiceBuilder) ID() string {
	return b.s.id
}

// Name returns the name currently set on the builder.
func (b *ServiceBuilder) Name() string {
	return b.s.name
}

// Package returns the package currently set on the builder.
func (b *ServiceBuilder) Package() string {
	return b.s.packagez
}

// DefaultHost returns the default host currently set on the builder.
func (b *ServiceBuilder) DefaultHost() string {
	return b.s.defaultHost
}

// Documentation returns the documentation currently set on the builder.
func (b *ServiceBuilder) Documentation() string {
	return b.s.documentation
}

// WithDefaultHost sets the default host fragment of a URL for the service.
func (b *ServiceBuilder) WithDefaultHost(host string) *ServiceBuilder {
	b.s.defaultHost = host
	return b
}

// WithPackage sets the Protobuf package this service belongs to.
func (b *ServiceBuilder) WithPackage(pkg string) *ServiceBuilder {
	b.s.packagez = pkg
	return b
}

// WithModel sets the model this service belongs to.
func (b *ServiceBuilder) WithModel(model *API) *ServiceBuilder {
	b.s.model = model
	return b
}

// WithQuickstartMethod sets the quickstart method for the service.
func (b *ServiceBuilder) WithQuickstartMethod(method *Method) *ServiceBuilder {
	b.s.quickstartMethod = method
	return b
}
