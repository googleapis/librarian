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

import (
	"iter"
	"maps"
	"slices"
)

// API represents an API surface.
type API struct {
	// Name of the API (e.g. secretmanager).
	name string
	// Name of the package name in the source specification format. For Protobuf
	// this may be `google.cloud.secretmanager.v1`.
	packageName string
	// The API Title (e.g. "Secret Manager API" or "Cloud Spanner API").
	title string
	// The API Description.
	description string
	// The API Revision. In discovery-based services this is the "revision"
	// attribute.
	revision string
	// Services are a collection of services that make up the API.
	services []*Service
	// Messages are a collection of messages used to process request and
	// responses in the API.
	messages []*Message
	// ExternalMessages are top-level messages from external packages (e.g. google.type.LatLng).
	externalMessages []*Message
	// Enums
	enums []*Enum
	// ExternalEnums are top-level enums from external packages.
	externalEnums []*Enum
	// ResourceDefinitions contains the data from the `google.api.resource_definition` annotation.
	resourceDefinitions []*Resource
	// QuickstartService is the service that will be used to generate the quickstart sample
	// at the package level.
	quickstartService *Service
	// Language specific annotations.
	codec any

	// serviceByID returns a service that is associated with the API.
	serviceByID map[string]*Service
	// methodByID returns a method that is associated with the API.
	methodByID map[string]*Method
	// messageByID returns a message that is associated with the API.
	messageByID map[string]*Message
	// enumByID returns a message that is associated with the API.
	enumByID map[string]*Enum
	// resourceByType returns a resource that is associated with the API.
	resourceByType map[string]*Resource
	// mapByID returns a map that is associated with the API.
	mapByID map[string]*Map

	// RubyPackage captures the package name override for Ruby.
	//
	// Normally, the Protobuf package name is capitalized element by element and
	// then used to construct the Ruby package name. For example,
	// `google.cloud.secretmanager.v1` becomes
	// `Google::Cloud::Secretmanager::V1`. In this example,
	// `Google::Cloud::SecretManager::V1` would be more idiomatic. The Protobuf
	// file options can set this override.
	//
	// In sidekick, a codec for Ruby might use this field directly. The codec
	// for Swift uses this to emit warnings if: (1) the Ruby override differs
	// from the standard name, and (2) there is no override in the
	// `librarian.yaml` file.
	rubyPackage string

	// PhpNamespace captures the namespace override for PHP.
	//
	// Normally, the Protobuf package name is capitalized element by element and
	// then used to construct the PHP namespace. For example,
	// `google.cloud.secretmanager.v1` becomes
	// `Google\\Cloud\\Secretmanager\\V1`. In this example,
	// `Google\\Cloud\\SecretManager\\V1` would be more idiomatic. The Protobuf
	// file options can set this override.
	//
	// In sidekick, a codec for PHP might use this field directly. The codec
	// for Swift uses this to emit warnings if: (1) the PHP override differs
	// from the standard name, and (2) there is no override in the
	// `librarian.yaml` file.
	phpNamespace string

	// CsharpNamespace captures the namespace override for C#.
	//
	// Normally, the Protobuf package name is capitalized element by element and
	// then used to construct the C# namespace. For example,
	// `google.cloud.secretmanager.v1` becomes `Google.Cloud.Secretmanager.V1`.
	// In this example, `Google.Cloud.SecretManager.V1` would be more idiomatic.
	// The Protobuf file options can set this override.
	//
	// In sidekick, a codec for C# might use this field directly. The codec
	// for Swift uses this to emit warnings if: (1) the C# override differs
	// from the standard name, and (2) there is no override in the
	// `librarian.yaml` file.
	csharpNamespace string
}

// ModelOverride holds configuration overrides for an API model.
type ModelOverride struct {
	Name        string
	Title       string
	Description string
	IncludedIDs []string
	SkippedIDs  []string
}

// Name returns the name of the API (e.g. secretmanager).
func (a *API) Name() string {
	return a.name
}

// PackageName returns the package name in the source specification format.
func (a *API) PackageName() string {
	return a.packageName
}

// Title returns the API title.
func (a *API) Title() string {
	return a.title
}

// Description returns the API description.
func (a *API) Description() string {
	return a.description
}

// Revision returns the API revision.
func (a *API) Revision() string {
	return a.revision
}

// Services returns the collection of services that make up the API.
func (a *API) Services() []*Service {
	return slices.Clone(a.services)
}

// Messages returns the collection of messages in the API.
func (a *API) Messages() []*Message {
	return slices.Clone(a.messages)
}

// ExternalMessages returns top-level messages from external packages.
func (a *API) ExternalMessages() []*Message {
	return slices.Clone(a.externalMessages)
}

// Enums returns the collection of enums in the API.
func (a *API) Enums() []*Enum {
	return slices.Clone(a.enums)
}

// ExternalEnums returns top-level enums from external packages.
func (a *API) ExternalEnums() []*Enum {
	return slices.Clone(a.externalEnums)
}

// ResourceDefinitions returns the resource definitions from google.api.resource_definition.
func (a *API) ResourceDefinitions() []*Resource {
	return slices.Clone(a.resourceDefinitions)
}

// QuickstartService returns the service used to generate the package-level quickstart sample.
func (a *API) QuickstartService() *Service {
	return a.quickstartService
}

// Codec returns the language-specific annotations.
func (a *API) Codec() any {
	return a.codec
}

// SetAnnotations sets the language-specific annotations on the API.
func (a *API) SetAnnotations(codec any) {
	a.codec = codec
}

// RubyPackage returns the package name override for Ruby.
func (a *API) RubyPackage() string {
	return a.rubyPackage
}

// PhpNamespace returns the namespace override for PHP.
func (a *API) PhpNamespace() string {
	return a.phpNamespace
}

// CsharpNamespace returns the namespace override for C#.
func (a *API) CsharpNamespace() string {
	return a.csharpNamespace
}

// HasMessages returns true if the API contains messages (most do).
//
// This is useful in Go templates to skip code that only makes sense
// when per-message code follows.
func (a *API) HasMessages() bool {
	return len(a.messages) != 0
}

// HasServices returns true if the API contains services (most do).
//
// This is useful in Go templates to skip code that only makes sense
// when per-service code follows.
func (a *API) HasServices() bool {
	return len(a.services) != 0
}

// ModelCodec returns the Codec field with an alternative name.
//
// In some Go templates we want to access the annotations for the
// enclosing model.
func (a *API) ModelCodec() any {
	return a.codec
}

// Service returns a service that is associated with the API.
func (a *API) Service(id string) *Service {
	return a.serviceByID[id]
}

// AllServices returns an iterator over the services in the API.
func (a *API) AllServices() iter.Seq[*Service] {
	return maps.Values(a.serviceByID)
}

func (a *API) addService(s *Service) {
	if a.serviceByID == nil {
		a.serviceByID = make(map[string]*Service)
	}
	a.serviceByID[s.id] = s
}

// Method returns a method that is associated with the API.
func (a *API) Method(id string) *Method {
	return a.methodByID[id]
}

// AllMethods returns an iterator over the methods in the API.
func (a *API) AllMethods() iter.Seq[*Method] {
	return maps.Values(a.methodByID)
}

func (a *API) addMethod(m *Method) {
	if a.methodByID == nil {
		a.methodByID = make(map[string]*Method)
	}
	a.methodByID[m.id] = m
}

// Message returns a message that is associated with the API.
func (a *API) Message(id string) *Message {
	return a.messageByID[id]
}

// AllMessages returns an iterator over the messages in the API.
func (a *API) AllMessages() iter.Seq[*Message] {
	return maps.Values(a.messageByID)
}

func (a *API) addMessage(m *Message) {
	if a.messageByID == nil {
		a.messageByID = make(map[string]*Message)
	}
	a.messageByID[m.id] = m
}

// Enum returns a message that is associated with the API.
func (a *API) Enum(id string) *Enum {
	return a.enumByID[id]
}

// AllEnums returns an iterator over the enums in the API.
func (a *API) AllEnums() iter.Seq[*Enum] {
	return maps.Values(a.enumByID)
}

func (a *API) addEnum(e *Enum) {
	if a.enumByID == nil {
		a.enumByID = make(map[string]*Enum)
	}
	a.enumByID[e.id] = e
}

// Resource returns a resource that is associated with the API.
func (a *API) Resource(typ string) *Resource {
	return a.resourceByType[typ]
}

// AllResources returns an iterator over the resources in the API.
func (a *API) AllResources() iter.Seq[*Resource] {
	return maps.Values(a.resourceByType)
}

func (a *API) addResource(r *Resource) {
	if a.resourceByType == nil {
		a.resourceByType = make(map[string]*Resource)
	}
	a.resourceByType[r.typez] = r
}

// Map returns a map that is associated with the API.
func (a *API) Map(id string) *Map {
	return a.mapByID[id]
}

// AllMaps returns an iterator over the maps in the API.
func (a *API) AllMaps() iter.Seq[*Map] {
	return maps.Values(a.mapByID)
}

func (a *API) addMap(m *Map) {
	if a.mapByID == nil {
		a.mapByID = make(map[string]*Map)
	}
	a.mapByID[m.id] = m
}
