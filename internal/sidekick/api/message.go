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
	documentation      string
	name               string
	id                 string
	deprecated         bool
	fields             []*Field
	syntheticRequest   bool
	skippedFields      []*Field
	servicePlaceholder bool
	enums              []*Enum
	messages           []*Message
	oneOfs             []*OneOf
	parent             *Message
	packagez           string
	pagination         *PaginationInfo
	resource           *Resource
	codec              any
}

// Documentation returns the documentation for the message.
func (m *Message) Documentation() string {
	return m.documentation
}

// Name returns the name of the attribute.
func (m *Message) Name() string {
	return m.name
}

// ID returns a unique identifier.
func (m *Message) ID() string {
	return m.id
}

// Deprecated returns whether the message is marked as deprecated.
func (m *Message) Deprecated() bool {
	return m.deprecated
}

// Fields returns the fields associated with the Message.
func (m *Message) Fields() []*Field {
	return m.fields
}

// SyntheticRequest returns true if this is a synthetic request message.
//
// These messages are created by sidekick when parsing Discovery docs and
// OpenAPI specifications. The query and request parameters for each method
// are grouped into a synthetic message.
func (m *Message) SyntheticRequest() bool {
	return m.syntheticRequest
}

// SkippedFields returns the fields that were removed.
//
// When generating Sidekick-gencode <-> Protobuf-gencode converters this is useful to know because
// the emitted code for the protos may need additional initializers.
func (m *Message) SkippedFields() []*Field {
	return m.skippedFields
}

// ServicePlaceholder returns true if this message is a placeholder / doppelganger for an [Service].
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
func (m *Message) ServicePlaceholder() bool {
	return m.servicePlaceholder
}

// Enums returns the enums associated with the Message.
func (m *Message) Enums() []*Enum {
	return m.enums
}

// Messages returns the nested messages associated with the Message.
func (m *Message) Messages() []*Message {
	return m.messages
}

// OneOfs returns the one-ofs associated with the Message.
func (m *Message) OneOfs() []*OneOf {
	return m.oneOfs
}

// Parent returns the ancestor of this message, if any.
func (m *Message) Parent() *Message {
	return m.parent
}

// Package returns the Protobuf package this message belongs to.
func (m *Message) Package() string {
	return m.packagez
}

// Pagination returns pagination metadata if this Message is returned by a standard
// List RPC and conforms to [AIP-4233](https://google.aip.dev/client-libraries/4233).
func (m *Message) Pagination() *PaginationInfo {
	return m.pagination
}

// Resource returns the data from the `google.api.resource` annotation.
func (m *Message) Resource() *Resource {
	return m.resource
}

// Codec returns the language-specific annotations.
func (m *Message) Codec() any {
	return m.codec
}

// SetAnnotations sets the language-specific codec annotations on the message.
func (m *Message) SetAnnotations(codec any) {
	m.codec = codec
}

// HasFields returns true if the message has fields.
func (m *Message) HasFields() bool {
	return len(m.fields) != 0
}

// HasSkippedFields returns true if the message has fields affected by the `skipped_ids` configuration.
func (m *Message) HasSkippedFields() bool {
	return len(m.skippedFields) != 0
}

// HasSkippedProtoConversionFields returns true if any field in the message has SkipProtoConversion = true.
func (m *Message) HasSkippedProtoConversionFields() bool {
	for _, f := range m.fields {
		if f.skipProtoConversion {
			return true
		}
	}
	return false
}
