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
)

// API represents an API surface.
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
	// ResourceDefinitions contains the data from the `google.api.resource_definition` annotation.
	ResourceDefinitions []*Resource
	// QuickstartService is the service that will be used to generate the quickstart sample
	// at the package level.
	QuickstartService *Service
	// Language specific annotations.
	Codec any

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
	RubyPackage string

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
	PhpNamespace string

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
	CsharpNamespace string
}

// ModelOverride holds configuration overrides for an API model.
type ModelOverride struct {
	Name        string
	Title       string
	Description string
	IncludedIDs []string
	SkippedIDs  []string
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

// Service returns a service that is associated with the API.
func (a *API) Service(id string) *Service {
	return a.serviceByID[id]
}

// AllServices returns an iterator over the services in the API.
func (a *API) AllServices() iter.Seq[*Service] {
	return maps.Values(a.serviceByID)
}

// AddService adds a service to the API.
func (a *API) AddService(s *Service) {
	if a.serviceByID == nil {
		a.serviceByID = make(map[string]*Service)
	}
	a.serviceByID[s.ID] = s
}

// Method returns a method that is associated with the API.
func (a *API) Method(id string) *Method {
	return a.methodByID[id]
}

// AllMethods returns an iterator over the methods in the API.
func (a *API) AllMethods() iter.Seq[*Method] {
	return maps.Values(a.methodByID)
}

// AddMethod adds a method to the API.
func (a *API) AddMethod(m *Method) {
	if a.methodByID == nil {
		a.methodByID = make(map[string]*Method)
	}
	a.methodByID[m.ID] = m
}

// Message returns a message that is associated with the API.
func (a *API) Message(id string) *Message {
	return a.messageByID[id]
}

// AllMessages returns an iterator over the messages in the API.
func (a *API) AllMessages() iter.Seq[*Message] {
	return maps.Values(a.messageByID)
}

// AddMessage adds a message to the API.
func (a *API) AddMessage(m *Message) {
	if a.messageByID == nil {
		a.messageByID = make(map[string]*Message)
	}
	a.messageByID[m.ID] = m
}

// Enum returns a message that is associated with the API.
func (a *API) Enum(id string) *Enum {
	return a.enumByID[id]
}

// AllEnums returns an iterator over the enums in the API.
func (a *API) AllEnums() iter.Seq[*Enum] {
	return maps.Values(a.enumByID)
}

// AddEnum adds an enum to the API.
func (a *API) AddEnum(e *Enum) {
	if a.enumByID == nil {
		a.enumByID = make(map[string]*Enum)
	}
	a.enumByID[e.ID] = e
}

// Resource returns a resource that is associated with the API.
func (a *API) Resource(typ string) *Resource {
	return a.resourceByType[typ]
}

// AllResources returns an iterator over the resources in the API.
func (a *API) AllResources() iter.Seq[*Resource] {
	return maps.Values(a.resourceByType)
}

// AddResource adds a resource to the API.
func (a *API) AddResource(r *Resource) {
	if a.resourceByType == nil {
		a.resourceByType = make(map[string]*Resource)
	}
	a.resourceByType[r.Type] = r
}
