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
	"slices"
)

// Service represents a service in an API.
type Service struct {
	// Documentation for the service.
	documentation string
	// Name of the attribute.
	name string
	// ID is a unique identifier.
	id string
	// Some source specifications allow marking services as deprecated.
	deprecated bool
	// Methods associated with the Service.
	methods []*Method
	// DefaultHost fragment of a URL.
	defaultHost string
	// The Protobuf package this service belongs to.
	packagez string

	// The model this service belongs to, Go templates use this field to
	// navigate the data structure.
	model *API
	// QuickstartMethod is the method that will be used to generate the quickstart sample
	// for this service.
	quickstartMethod *Method
	// Language specific annotations.
	codec any
}

// Documentation returns the documentation for the service.
func (s *Service) Documentation() string { return s.documentation }

// Name returns the name of the service.
func (s *Service) Name() string { return s.name }

// ID returns the unique identifier for the service.
func (s *Service) ID() string { return s.id }

// Deprecated returns whether the service is deprecated.
func (s *Service) Deprecated() bool { return s.deprecated }

// Methods returns the methods associated with the service.
func (s *Service) Methods() []*Method { return s.methods }

// DefaultHost returns the default host fragment of a URL for the service.
func (s *Service) DefaultHost() string { return s.defaultHost }

// Package returns the Protobuf package this service belongs to.
func (s *Service) Package() string { return s.packagez }

// Model returns the model this service belongs to.
func (s *Service) Model() *API { return s.model }

// QuickstartMethod returns the method used to generate the quickstart sample for this service.
func (s *Service) QuickstartMethod() *Method { return s.quickstartMethod }

// Codec returns the language-specific annotations for the service.
func (s *Service) Codec() any { return s.codec }

// SetAnnotations sets the language-specific annotations for the service.
func (s *Service) SetAnnotations(a any) { s.codec = a }

// HasClientSideStreaming returns true if the service contains any methods
// that support client-side streaming.
func (s *Service) HasClientSideStreaming() bool {
	return slices.ContainsFunc(s.methods, func(m *Method) bool {
		return m.clientSideStreaming
	})
}

// HasBidiStreaming returns true if the service contains any methods
// that support bidirectional streaming.
func (s *Service) HasBidiStreaming() bool {
	return slices.ContainsFunc(s.methods, func(m *Method) bool {
		return m.clientSideStreaming && m.serverSideStreaming
	})
}

// HasServerSideStreaming returns true if the service contains any methods
// that support server-side streaming and are not bidirectional.
func (s *Service) HasServerSideStreaming() bool {
	return slices.ContainsFunc(s.methods, func(m *Method) bool {
		return !m.clientSideStreaming && m.serverSideStreaming
	})
}
