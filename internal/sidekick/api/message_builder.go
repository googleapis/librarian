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

// MessageBuilder constructs a [Message].
type MessageBuilder struct {
	message Message
}

// NewMessageBuilder creates a new [MessageBuilder].
func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{}
}

// NewMessageBuilderFrom creates a new [MessageBuilder] initialized with a shallow copy of m.
func NewMessageBuilderFrom(m *Message) *MessageBuilder {
	if m == nil {
		return &MessageBuilder{}
	}
	b := &MessageBuilder{message: *m}
	b.message.fields = slices.Clone(m.fields)
	b.message.skippedFields = slices.Clone(m.skippedFields)
	b.message.enums = slices.Clone(m.enums)
	b.message.messages = slices.Clone(m.messages)
	b.message.oneOfs = slices.Clone(m.oneOfs)
	return b
}

// WithDocumentation sets the documentation for the message.
func (b *MessageBuilder) WithDocumentation(doc string) *MessageBuilder {
	b.message.documentation = doc
	return b
}

// WithName sets the name of the message.
func (b *MessageBuilder) WithName(name string) *MessageBuilder {
	b.message.name = name
	return b
}

// WithID sets the unique identifier of the message.
func (b *MessageBuilder) WithID(id string) *MessageBuilder {
	b.message.id = id
	return b
}

// WithDeprecated sets whether the message is deprecated.
func (b *MessageBuilder) WithDeprecated(deprecated bool) *MessageBuilder {
	b.message.deprecated = deprecated
	return b
}

// WithFields sets the fields associated with the message.
func (b *MessageBuilder) WithFields(fields ...*Field) *MessageBuilder {
	b.message.fields = fields
	return b
}

// AddField appends a field to the message.
func (b *MessageBuilder) AddField(field *Field) *MessageBuilder {
	b.message.fields = append(b.message.fields, field)
	return b
}

// WithSyntheticRequest sets whether this is a synthetic request message.
func (b *MessageBuilder) WithSyntheticRequest(synthetic bool) *MessageBuilder {
	b.message.syntheticRequest = synthetic
	return b
}

// WithSkippedFields sets the skipped fields on the message.
func (b *MessageBuilder) WithSkippedFields(fields ...*Field) *MessageBuilder {
	b.message.skippedFields = fields
	return b
}

// WithServicePlaceholder sets whether this message is a placeholder for a service.
func (b *MessageBuilder) WithServicePlaceholder(placeholder bool) *MessageBuilder {
	b.message.servicePlaceholder = placeholder
	return b
}

// WithEnums sets the enums associated with the message.
func (b *MessageBuilder) WithEnums(enums ...*Enum) *MessageBuilder {
	b.message.enums = enums
	return b
}

// AddEnum appends an enum to the message.
func (b *MessageBuilder) AddEnum(enum *Enum) *MessageBuilder {
	b.message.enums = append(b.message.enums, enum)
	return b
}

// WithMessages sets the nested messages associated with the message.
func (b *MessageBuilder) WithMessages(messages ...*Message) *MessageBuilder {
	b.message.messages = messages
	return b
}

// AddMessage appends a nested message to the message.
func (b *MessageBuilder) AddMessage(msg *Message) *MessageBuilder {
	b.message.messages = append(b.message.messages, msg)
	return b
}

// WithOneOfs sets the one-ofs associated with the message.
func (b *MessageBuilder) WithOneOfs(oneOfs ...*OneOf) *MessageBuilder {
	b.message.oneOfs = oneOfs
	return b
}

// AddOneOf appends a one-of to the message.
func (b *MessageBuilder) AddOneOf(oneOf *OneOf) *MessageBuilder {
	b.message.oneOfs = append(b.message.oneOfs, oneOf)
	return b
}

// WithParent sets the parent message.
func (b *MessageBuilder) WithParent(parent *Message) *MessageBuilder {
	b.message.parent = parent
	return b
}

// WithPackage sets the Protobuf package for the message.
func (b *MessageBuilder) WithPackage(pkg string) *MessageBuilder {
	b.message.packagez = pkg
	return b
}

// WithPagination sets the pagination info for the message.
func (b *MessageBuilder) WithPagination(pagination *PaginationInfo) *MessageBuilder {
	b.message.pagination = pagination
	return b
}

// WithResource sets the resource definition on the message.
func (b *MessageBuilder) WithResource(resource *Resource) *MessageBuilder {
	b.message.resource = resource
	return b
}

// WithAnnotations sets the codec annotations on the message.
func (b *MessageBuilder) WithAnnotations(codec any) *MessageBuilder {
	b.message.codec = codec
	return b
}

// Build returns the constructed [Message].
func (b *MessageBuilder) Build() *Message {
	return &b.message
}
