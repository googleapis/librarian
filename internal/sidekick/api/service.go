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
	Documentation string
	// Name of the attribute.
	Name string
	// ID is a unique identifier.
	ID string
	// Some source specifications allow marking services as deprecated.
	Deprecated bool
	// Methods associated with the Service.
	Methods []*Method
	// DefaultHost fragment of a URL.
	DefaultHost string
	// The Protobuf package this service belongs to.
	Package string

	// The model this service belongs to, mustache templates use this field to
	// navigate the data structure.
	Model *API
	// QuickstartMethod is the method that will be used to generate the quickstart sample
	// for this service.
	QuickstartMethod *Method
	// Language specific annotations.
	Codec any
}

// HasClientSideStreaming returns true if the service contains any methods
// that support client-side streaming.
func (s *Service) HasClientSideStreaming() bool {
	return slices.ContainsFunc(s.Methods, func(m *Method) bool {
		return m.ClientSideStreaming
	})
}

// HasBidiStreaming returns true if the service contains any methods
// that support bidirectional streaming.
func (s *Service) HasBidiStreaming() bool {
	return slices.ContainsFunc(s.Methods, func(m *Method) bool {
		return m.ClientSideStreaming && m.ServerSideStreaming
	})
}
