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

// Enum defines a message used in request/response handling.
type Enum struct {
	// documentation for the message.
	documentation string
	// name of the attribute.
	name string
	// id is a unique identifier.
	id string
	// Some source specifications allow marking enums as deprecated.
	deprecated bool
	// values associated with the Enum.
	values []*EnumValue
	// The unique integer values, some enums have multiple aliases for the
	// same number (e.g. `enum X { a = 0, b = 0, c = 1 }`).
	uniqueNumberValues []*EnumValue
	// valuesForExamples contains a subset of values suitable for use in generated samples.
	// e.g. non-deprecated, non-zero values.
	valuesForExamples []*SampleValue
	// parent returns the ancestor of this node, if any.
	parent *Message
	// The Protobuf package this enum belongs to.
	packagez string
	// Language specific annotations.
	codec any
}

// Documentation returns the documentation for the enum.
func (e *Enum) Documentation() string { return e.documentation }

// Name returns the name of the enum.
func (e *Enum) Name() string { return e.name }

// ID returns the unique identifier of the enum.
func (e *Enum) ID() string { return e.id }

// Deprecated returns whether the enum is deprecated.
func (e *Enum) Deprecated() bool { return e.deprecated }

// Values returns the values associated with the enum.
func (e *Enum) Values() []*EnumValue { return slices.Clone(e.values) }

// UniqueNumberValues returns the enum values with unique integer numbers.
func (e *Enum) UniqueNumberValues() []*EnumValue { return slices.Clone(e.uniqueNumberValues) }

// ValuesForExamples returns the subset of enum values suitable for generated samples.
func (e *Enum) ValuesForExamples() []*SampleValue { return slices.Clone(e.valuesForExamples) }

// Parent returns the parent [Message] of this enum, if any.
func (e *Enum) Parent() *Message { return e.parent }

// Package returns the package this enum belongs to.
func (e *Enum) Package() string { return e.packagez }

// Codec returns the language-specific annotations.
func (e *Enum) Codec() any { return e.codec }

// SetAnnotations sets the language-specific annotations.
func (e *Enum) SetAnnotations(codec any) { e.codec = codec }

// EnumValue defines a value in an Enum.
type EnumValue struct {
	// documentation for the message.
	documentation string
	// name of the attribute.
	name string
	// id is a unique identifier.
	id string
	// Some source specifications allow marking enum values as deprecated.
	deprecated bool
	// number of the attribute.
	number int32
	// parent returns the ancestor of this node, if any.
	parent *Enum
	// Language specific annotations.
	codec any
}

// Documentation returns the documentation for the enum value.
func (v *EnumValue) Documentation() string { return v.documentation }

// Name returns the name of the enum value.
func (v *EnumValue) Name() string { return v.name }

// ID returns the unique identifier of the enum value.
func (v *EnumValue) ID() string { return v.id }

// Deprecated returns whether the enum value is deprecated.
func (v *EnumValue) Deprecated() bool { return v.deprecated }

// Number returns the integer value of the enum value.
func (v *EnumValue) Number() int32 { return v.number }

// Parent returns the parent [Enum] of this value.
func (v *EnumValue) Parent() *Enum { return v.parent }

// Codec returns the language-specific annotations.
func (v *EnumValue) Codec() any { return v.codec }

// SetAnnotations sets the language-specific annotations.
func (v *EnumValue) SetAnnotations(codec any) { v.codec = codec }

// SampleValue represents a value used in a sample.
type SampleValue struct {
	// The enum value.
	enumValue *EnumValue
	// The index of the value in the sample list (0-based).
	index int
}

// NewSampleValue creates a new [SampleValue].
func NewSampleValue(enumValue *EnumValue, index int) *SampleValue {
	return &SampleValue{
		enumValue: enumValue,
		index:     index,
	}
}

// EnumValue returns the [EnumValue] for this sample value.
func (s *SampleValue) EnumValue() *EnumValue { return s.enumValue }

// Index returns the 0-based index of the value in the sample list.
func (s *SampleValue) Index() int { return s.index }
