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

// FieldBuilder constructs a [Field].
type FieldBuilder struct {
	documentation       string
	name                string
	id                  string
	typez               Typez
	typezID             string
	jsonName            string
	optional            bool
	proto2Optional      bool
	repeated            bool
	mapz                bool
	deprecated          bool
	isOneOf             bool
	recursive           bool
	skipProtoConversion bool
	autoPopulated       bool
	behavior            []FieldBehavior
	group               *OneOf
	parent              *Message
	messageType         *Message
	enumType            *Enum
	mapType             *Map
	resourceReference   *ResourceReference
	resourceNamePattern *ResourceNamePattern
}

// NewFieldBuilder creates a new [FieldBuilder].
func NewFieldBuilder() *FieldBuilder {
	return &FieldBuilder{}
}

// NewFieldBuilderFrom creates a [FieldBuilder] initialized with a shallow copy
// of the fields of f (excluding language-specific annotations).
func NewFieldBuilderFrom(f *Field) *FieldBuilder {
	return &FieldBuilder{
		documentation:       f.documentation,
		name:                f.name,
		id:                  f.id,
		typez:               f.typez,
		typezID:             f.typezID,
		jsonName:            f.jsonName,
		optional:            f.optional,
		proto2Optional:      f.proto2Optional,
		repeated:            f.repeated,
		mapz:                f.mapz,
		deprecated:          f.deprecated,
		isOneOf:             f.isOneOf,
		recursive:           f.recursive,
		skipProtoConversion: f.skipProtoConversion,
		autoPopulated:       f.autoPopulated,
		behavior:            slices.Clone(f.behavior),
		group:               f.group,
		parent:              f.parent,
		messageType:         f.messageType,
		enumType:            f.enumType,
		mapType:             f.mapType,
		resourceReference:   f.resourceReference,
		resourceNamePattern: f.resourceNamePattern,
	}
}

// WithDocumentation sets the documentation for the field.
func (b *FieldBuilder) WithDocumentation(documentation string) *FieldBuilder {
	b.documentation = documentation
	return b
}

// WithName sets the name of the attribute.
func (b *FieldBuilder) WithName(name string) *FieldBuilder {
	b.name = name
	return b
}

// WithID sets the unique identifier of the field.
func (b *FieldBuilder) WithID(id string) *FieldBuilder {
	b.id = id
	return b
}

// WithTypez sets the datatype of the field.
func (b *FieldBuilder) WithTypez(typez Typez) *FieldBuilder {
	b.typez = typez
	return b
}

// WithTypezID sets the ID of the type the field refers to.
func (b *FieldBuilder) WithTypezID(typezID string) *FieldBuilder {
	b.typezID = typezID
	return b
}

// WithJSONName sets the JSON name of the field.
func (b *FieldBuilder) WithJSONName(jsonName string) *FieldBuilder {
	b.jsonName = jsonName
	return b
}

// WithOptional sets whether the field is marked as optional in proto3.
func (b *FieldBuilder) WithOptional(optional bool) *FieldBuilder {
	b.optional = optional
	return b
}

// WithProto2Optional sets whether the field is a scalar or enum declared optional in a proto2 file.
func (b *FieldBuilder) WithProto2Optional(proto2Optional bool) *FieldBuilder {
	b.proto2Optional = proto2Optional
	return b
}

// WithRepeated sets whether the field is a repeated field.
func (b *FieldBuilder) WithRepeated(repeated bool) *FieldBuilder {
	b.repeated = repeated
	return b
}

// WithMap sets whether the field is a map.
func (b *FieldBuilder) WithMap(mapz bool) *FieldBuilder {
	b.mapz = mapz
	return b
}

// WithDeprecated sets whether the field is deprecated.
func (b *FieldBuilder) WithDeprecated(deprecated bool) *FieldBuilder {
	b.deprecated = deprecated
	return b
}

// WithIsOneOf sets whether the field is related to a one-of.
func (b *FieldBuilder) WithIsOneOf(isOneOf bool) *FieldBuilder {
	b.isOneOf = isOneOf
	return b
}

// WithRecursive sets whether the field refers to the containing message.
func (b *FieldBuilder) WithRecursive(recursive bool) *FieldBuilder {
	b.recursive = recursive
	return b
}

// WithSkipProtoConversion sets whether the field should be omitted from proto/prost conversion.
func (b *FieldBuilder) WithSkipProtoConversion(skipProtoConversion bool) *FieldBuilder {
	b.skipProtoConversion = skipProtoConversion
	return b
}

// WithAutoPopulated sets whether the field is eligible to be auto-populated.
func (b *FieldBuilder) WithAutoPopulated(autoPopulated bool) *FieldBuilder {
	b.autoPopulated = autoPopulated
	return b
}

// WithBehavior sets the field behaviors.
func (b *FieldBuilder) WithBehavior(behavior ...FieldBehavior) *FieldBuilder {
	b.behavior = append(b.behavior, behavior...)
	return b
}

// WithGroup sets the [OneOf] group for the field.
func (b *FieldBuilder) WithGroup(group *OneOf) *FieldBuilder {
	b.group = group
	return b
}

// WithParent sets the parent [Message] of the field.
func (b *FieldBuilder) WithParent(parent *Message) *FieldBuilder {
	b.parent = parent
	return b
}

// WithMessageType sets the [Message] type for the field.
func (b *FieldBuilder) WithMessageType(messageType *Message) *FieldBuilder {
	b.messageType = messageType
	return b
}

// WithEnumType sets the [Enum] type for the field.
func (b *FieldBuilder) WithEnumType(enumType *Enum) *FieldBuilder {
	b.enumType = enumType
	return b
}

// WithMapType sets the [Map] type for the field.
func (b *FieldBuilder) WithMapType(mapType *Map) *FieldBuilder {
	b.mapType = mapType
	return b
}

// WithResourceReference sets the [ResourceReference] for the field.
func (b *FieldBuilder) WithResourceReference(resourceReference *ResourceReference) *FieldBuilder {
	b.resourceReference = resourceReference
	return b
}

// WithResourceNamePattern sets the [ResourceNamePattern] for the field.
func (b *FieldBuilder) WithResourceNamePattern(resourceNamePattern *ResourceNamePattern) *FieldBuilder {
	b.resourceNamePattern = resourceNamePattern
	return b
}

// Build returns the constructed [Field].
func (b *FieldBuilder) Build() *Field {
	return &Field{
		documentation:       b.documentation,
		name:                b.name,
		id:                  b.id,
		typez:               b.typez,
		typezID:             b.typezID,
		jsonName:            b.jsonName,
		optional:            b.optional,
		proto2Optional:      b.proto2Optional,
		repeated:            b.repeated,
		mapz:                b.mapz,
		deprecated:          b.deprecated,
		isOneOf:             b.isOneOf,
		recursive:           b.recursive,
		skipProtoConversion: b.skipProtoConversion,
		autoPopulated:       b.autoPopulated,
		behavior:            b.behavior,
		group:               b.group,
		parent:              b.parent,
		messageType:         b.messageType,
		enumType:            b.enumType,
		mapType:             b.mapType,
		resourceReference:   b.resourceReference,
		resourceNamePattern: b.resourceNamePattern,
	}
}
