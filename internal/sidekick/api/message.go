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

// Message defines a message used in request/response handling.
type Message struct {
	// Documentation for the message.
	Documentation string
	// Name of the attribute.
	Name string
	// ID is a unique identifier.
	ID string
	// Some source specifications allow marking messages as deprecated.
	Deprecated bool
	// Fields associated with the Message.
	Fields []*Field
	// If true, this is a synthetic request message.
	//
	// These messages are created by sidekick when parsing Discovery docs and
	// OpenAPI specifications. The query and request parameters for each method
	// are grouped into a synthetic message.
	SyntheticRequest bool
	// If true, this message is a placeholder / doppelganger for a `api.Service`.
	//
	// These messages are created by sidekick when parsing Discovery docs and
	// OpenAPI specifications. All the synthetic messages for a service need to
	// be grouped under a unique namespace to avoid clashes with similar
	// synthetic messages in other services. Sidekick creates a placeholder
	// message that represents "the service".
	//
	// That is, `service1` and `service2` may both have a synthetic `getRequest`
	// message, with different attributes. We need these to be different
	// messages, with different names. So we create a different parent message
	// for each.
	ServicePlaceholder bool
	// Enums associated with the Message.
	Enums []*Enum
	// Messages associated with the Message. In protobuf these are referred to as
	// nested messages.
	Messages []*Message
	// OneOfs associated with the Message.
	OneOfs []*OneOf
	// Parent returns the ancestor of this message, if any.
	Parent *Message
	// The Protobuf package this message belongs to.
	Package string
	IsMap   bool
	// Indicates that this Message is returned by a standard
	// List RPC and conforms to [AIP-4233](https://google.aip.dev/client-libraries/4233).
	Pagination *PaginationInfo
	// Resource contains the data from the `google.api.resource` annotation.
	Resource *Resource
	// Language specific annotations.
	Codec any
}

// HasFields returns true if the message has fields.
func (m *Message) HasFields() bool {
	return len(m.Fields) != 0
}
