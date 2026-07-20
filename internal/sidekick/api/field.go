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

// Field defines a field in a Message.
type Field struct {
	// Documentation for the field.
	Documentation string
	// Name of the attribute.
	Name string
	// ID is a unique identifier.
	ID string
	// Typez is the datatype of the field.
	Typez Typez
	// TypezID is the ID of the type the field refers to. This value is populated
	// for message-like types only.
	TypezID string
	// JSONName is the name of the field as it appears in JSON. Useful for
	// serializing to JSON.
	JSONName string
	// Optional indicates that the field is marked as optional in proto3.
	Optional bool

	// For a given field, at most one of `Repeated` or `Map` is true.
	//
	// Using booleans (as opposed to an enum) makes it easier to write mustache
	// templates.
	//
	// Repeated is true if the field is a repeated field.
	Repeated bool
	// Map is true if the field is a map.
	Map bool
	// Some source specifications allow marking fields as deprecated.
	Deprecated bool
	// IsOneOf is true if the field is related to a one-of and not
	// a proto3 optional field.
	IsOneOf bool
	// Some fields have a type that refers (sometimes indirectly) to the
	// containing message. That triggers slightly different code generation for
	// some languages.
	Recursive bool
	// AutoPopulated is true if the field is eligible to be auto-populated,
	// per the requirements in AIP-4235.
	//
	// That is:
	// - It has Typez == TypezString
	// - For Protobuf, does not have the `google.api.field_behavior = REQUIRED` annotation
	// - For Protobuf, has the `google.api.field_info.format = UUID4` annotation
	// - For OpenAPI, it is an optional field
	// - For OpenAPI, it has format == "uuid"
	AutoPopulated bool
	// FieldBehavior indicates how the field behaves in requests and responses.
	//
	// For example, that a field is required in requests, or given as output
	// but ignored as input.
	Behavior []FieldBehavior
	// For fields that are part of a OneOf, the group of fields that makes the
	// OneOf.
	Group *OneOf
	// The message that contains this field.
	Parent *Message
	// The message type for this field, can be nil.
	MessageType *Message
	// The enum type for this field, can be nil.
	EnumType *Enum
	// ResourceReference contains the data from the `google.api.resource_reference`
	// annotation.
	ResourceReference *ResourceReference
	// ResourceNamePattern is a parsed representation of the resource pattern associated
	// with this field.
	ResourceNamePattern *ResourceNamePattern
	// Codec is a placeholder to put language specific annotations.
	Codec any
}

// DocumentAsRequired returns true if the field should be documented as required.
func (field *Field) DocumentAsRequired() bool {
	return slices.Contains(field.Behavior, FieldBehaviorRequired)
}

// Singular returns true if the field is not a map or a repeated field.
func (f *Field) Singular() bool {
	return !f.Map && !f.Repeated
}

// NameEqualJSONName returns true if the field's name is the same as its JSON name.
func (f *Field) NameEqualJSONName() bool {
	return f.JSONName == f.Name
}

// IsString returns true if the primitive type of a field is `TypezString`.
//
// This is useful for mustache templates that differ only
// in the broad category of field type involved.
func (f *Field) IsString() bool {
	return f.Typez == TypezString
}

// IsBytes returns true if the primitive type of a field is `TypezBytes`.
//
// This is useful for mustache templates that differ only
// in the broad category of field type involved.
func (f *Field) IsBytes() bool {
	return f.Typez == TypezBytes
}

// IsBool returns true if the primitive type of a field is `TypezBool`.
//
// This is useful for mustache templates that differ only
// in the broad category of field type involved.
func (f *Field) IsBool() bool {
	return f.Typez == TypezBool
}

// IsLikeInt returns true if the primitive type of a field is one of the
// integer types.
//
// This is useful for mustache templates that differ only
// in the broad category of field type involved.
func (f *Field) IsLikeInt() bool {
	switch f.Typez {
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
// This is useful for mustache templates that differ only
// in the broad category of field type involved.
func (f *Field) IsLikeUInt() bool {
	switch f.Typez {
	case TypezUint32, TypezUint64, TypezFixed32, TypezFixed64:
		return true
	default:
		return false
	}
}

// IsLikeFloat returns true if the primitive type of a field is a float or
// double.
//
// This is useful for mustache templates that differ only
// in the broad category of field type involved.
func (f *Field) IsLikeFloat() bool {
	return f.Typez == TypezDouble || f.Typez == TypezFloat
}

// IsEnum returns true if the primitive type of a field is `TypezEnum`.
//
// This is useful for mustache templates that differ only
// in the broad category of field type involved.
func (f *Field) IsEnum() bool {
	return f.Typez == TypezEnum
}

// IsObject returns true if the primitive type of a field is `TypezMessage`.
//
// This is useful for mustache templates that differ only
// in the broad category of field type involved.
//
// The templates *should* first check if the field is singular, as all maps are
// also objects.
func (f *Field) IsObject() bool {
	return f.Typez == TypezMessage
}

// IsResourceReference returns true if the field is annotated with google.api.resource_reference.
func (f *Field) IsResourceReference() bool {
	return f.ResourceReference != nil
}
