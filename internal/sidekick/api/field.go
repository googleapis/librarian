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
	"strings"
)

// Field defines a field in a Message.
type Field struct {
	// Documentation for the field.
	documentation string
	// Name of the attribute.
	name string
	// ID is a unique identifier.
	id string
	// Typez is the datatype of the field.
	typez Typez
	// TypezID is the ID of the type the field refers to. This value is populated
	// for message-like types only.
	typezID string
	// JSONName is the name of the field as it appears in JSON. Useful for
	// serializing to JSON.
	jsonName string
	// Optional indicates that the field is marked as optional in proto3.
	optional bool
	// Proto2Optional indicates that the field is a scalar or enum declared
	// optional in a proto2 file. Unlike `Optional`, it does not make the
	// field optional in the generated model.
	proto2Optional bool

	// For a given field, at most one of `Repeated` or `Map` is true.
	//
	// Using booleans (as opposed to an enum) makes it easier to write Go
	// templates.
	//
	// Repeated is true if the field is a repeated field.
	repeated bool
	// Map is true if the field is a map.
	mapz bool
	// Some source specifications allow marking fields as deprecated.
	deprecated bool
	// IsOneOf is true if the field is related to a one-of and not
	// a proto3 optional field.
	isOneOf bool
	// Some fields have a type that refers (sometimes indirectly) to the
	// containing message. That triggers slightly different code generation for
	// some languages.
	recursive bool
	// SkipProtoConversion is true if this field should be omitted from
	// proto/prost conversion (e.g. allowed Any fields in streaming RPCs).
	skipProtoConversion bool
	// AutoPopulated is true if the field is eligible to be auto-populated,
	// per the requirements in AIP-4235.
	//
	// That is:
	// - It has Typez == TypezString
	// - For Protobuf, does not have the `google.api.field_behavior = REQUIRED` annotation
	// - For Protobuf, has the `google.api.field_info.format = UUID4` annotation
	// - For OpenAPI, it is an optional field
	// - For OpenAPI, it has format == "uuid"
	autoPopulated bool
	// FieldBehavior indicates how the field behaves in requests and responses.
	//
	// For example, that a field is required in requests, or given as output
	// but ignored as input.
	behavior []FieldBehavior
	// For fields that are part of a OneOf, the group of fields that makes the
	// OneOf.
	group *OneOf
	// The message that contains this field.
	parent *Message
	// The message type for this field, can be nil.
	messageType *Message
	// The enum type for this field, can be nil.
	enumType *Enum
	// The map type for this field, can be nil.
	mapType *Map
	// ResourceReference contains the data from the `google.api.resource_reference`
	// annotation.
	resourceReference *ResourceReference
	// ResourceNamePattern is a parsed representation of the resource pattern associated
	// with this field.
	resourceNamePattern *ResourceNamePattern
	// Codec is a placeholder to put language specific annotations.
	codec any
}

// Documentation returns the documentation for the field.
func (f *Field) Documentation() string { return f.documentation }

// Name returns the name of the attribute.
func (f *Field) Name() string { return f.name }

// ID returns the unique identifier of the field.
func (f *Field) ID() string { return f.id }

// Typez returns the datatype of the field.
func (f *Field) Typez() Typez { return f.typez }

// TypezID returns the ID of the type the field refers to.
func (f *Field) TypezID() string { return f.typezID }

// JSONName returns the name of the field as it appears in JSON.
func (f *Field) JSONName() string { return f.jsonName }

// Optional returns true if the field is marked as optional in proto3.
func (f *Field) Optional() bool { return f.optional }

// Proto2Optional returns true if the field is a scalar or enum declared optional in a proto2 file.
func (f *Field) Proto2Optional() bool { return f.proto2Optional }

// Repeated returns true if the field is a repeated field.
func (f *Field) Repeated() bool { return f.repeated }

// Map returns true if the field is a map.
func (f *Field) Map() bool { return f.mapz }

// Deprecated returns true if the field is marked as deprecated.
func (f *Field) Deprecated() bool { return f.deprecated }

// IsOneOf returns true if the field is part of a one-of and not a proto3 optional field.
func (f *Field) IsOneOf() bool { return f.isOneOf }

// Recursive returns true if the field type refers to the containing message.
func (f *Field) Recursive() bool { return f.recursive }

// SkipProtoConversion returns true if this field should be omitted from proto/prost conversion.
func (f *Field) SkipProtoConversion() bool { return f.skipProtoConversion }

// AutoPopulated returns true if the field is eligible to be auto-populated per AIP-4235.
func (f *Field) AutoPopulated() bool { return f.autoPopulated }

// Behavior returns how the field behaves in requests and responses.
func (f *Field) Behavior() []FieldBehavior { return slices.Clone(f.behavior) }

// Group returns the OneOf group for fields that are part of a OneOf.
func (f *Field) Group() *OneOf { return f.group }

// Parent returns the message that contains this field.
func (f *Field) Parent() *Message { return f.parent }

// MessageType returns the message type for this field, or nil.
func (f *Field) MessageType() *Message { return f.messageType }

// EnumType returns the enum type for this field, or nil.
func (f *Field) EnumType() *Enum { return f.enumType }

// MapType returns the map type for this field, or nil.
func (f *Field) MapType() *Map { return f.mapType }

// ResourceReference returns the google.api.resource_reference annotation data.
func (f *Field) ResourceReference() *ResourceReference { return f.resourceReference }

// ResourceNamePattern returns the parsed resource pattern associated with this field.
func (f *Field) ResourceNamePattern() *ResourceNamePattern { return f.resourceNamePattern }

// Codec returns the language-specific annotations for the field.
func (f *Field) Codec() any { return f.codec }

// SetAnnotations sets the language-specific annotations for the field.
func (f *Field) SetAnnotations(codec any) { f.codec = codec }

// DocumentAsRequired returns true if the field should be documented as required.
func (field *Field) DocumentAsRequired() bool {
	return slices.Contains(field.behavior, FieldBehaviorRequired)
}

// Singular returns true if the field is not a map or a repeated field.
func (f *Field) Singular() bool {
	return !f.mapz && !f.repeated
}

// NameEqualJSONName returns true if the field's name is the same as its JSON name.
func (f *Field) NameEqualJSONName() bool {
	return f.jsonName == f.name
}

// IsString returns true if the primitive type of a field is `TypezString`.
//
// This is useful for Go templates that differ only
// in the broad category of field type involved.
func (f *Field) IsString() bool {
	return f.typez == TypezString
}

// IsBytes returns true if the primitive type of a field is `TypezBytes`.
//
// This is useful for Go templates that differ only
// in the broad category of field type involved.
func (f *Field) IsBytes() bool {
	return f.typez == TypezBytes
}

// IsBool returns true if the primitive type of a field is `TypezBool`.
//
// This is useful for Go templates that differ only
// in the broad category of field type involved.
func (f *Field) IsBool() bool {
	return f.typez == TypezBool
}

// IsLikeInt returns true if the primitive type of a field is one of the
// integer types.
//
// This is useful for Go templates that differ only
// in the broad category of field type involved.
func (f *Field) IsLikeInt() bool {
	switch f.typez {
	case TypezInt32, TypezInt64, TypezSint32, TypezSint64:
		return true
	case TypezSfixed32, TypezSfixed64:
		return true
	default:
		return false
	}
}

// IsLikeUInt returns true if the primitive type of a field is one of the
// unsigned integer types.
//
// This is useful for Go templates that differ only
// in the broad category of field type involved.
func (f *Field) IsLikeUInt() bool {
	switch f.typez {
	case TypezUint32, TypezUint64, TypezFixed32, TypezFixed64:
		return true
	default:
		return false
	}
}

// IsLikeFloat returns true if the primitive type of a field is a float or
// double.
//
// This is useful for Go templates that differ only
// in the broad category of field type involved.
func (f *Field) IsLikeFloat() bool {
	return f.typez == TypezDouble || f.typez == TypezFloat
}

// IsEnum returns true if the primitive type of a field is `TypezEnum`.
//
// This is useful for Go templates that differ only
// in the broad category of field type involved.
func (f *Field) IsEnum() bool {
	return f.typez == TypezEnum
}

// IsObject returns true if the primitive type of a field is `TypezMessage`.
//
// This is useful for Go templates that differ only
// in the broad category of field type involved.
func (f *Field) IsObject() bool {
	return f.typez == TypezMessage
}

// IsResourceReference returns true if the field is annotated with google.api.resource_reference.
func (f *Field) IsResourceReference() bool {
	return f.resourceReference != nil
}

// MarkSkipProtoConversionIfAllowed marks f as skipping proto conversion if its
// ID or fieldPath matches any entry in allowGrpcAnyFields, and returns true if
// it matched.
func MarkSkipProtoConversionIfAllowed(f *Field, fieldPath string, allowGrpcAnyFields []string) bool {
	if matchesAllowedAnyField(f.id, allowGrpcAnyFields) || matchesAllowedAnyField(fieldPath, allowGrpcAnyFields) {
		f.skipProtoConversion = true
		return true
	}
	return false
}

func matchesAllowedAnyField(id string, allowGrpcAnyFields []string) bool {
	idTrimmed := strings.TrimPrefix(id, ".")
	for _, allowed := range allowGrpcAnyFields {
		if strings.TrimPrefix(allowed, ".") == idTrimmed {
			return true
		}
	}
	return false
}
