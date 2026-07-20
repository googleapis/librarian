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

// Enum defines a message used in request/response handling.
type Enum struct {
	// Documentation for the message.
	Documentation string
	// Name of the attribute.
	Name string
	// ID is a unique identifier.
	ID string
	// Some source specifications allow marking enums as deprecated.
	Deprecated bool
	// Values associated with the Enum.
	Values []*EnumValue
	// The unique integer values, some enums have multiple aliases for the
	// same number (e.g. `enum X { a = 0, b = 0, c = 1 }`).
	UniqueNumberValues []*EnumValue
	// ValuesForExamples contains a subset of values suitable for use in generated samples.
	// e.g. non-deprecated, non-zero values.
	ValuesForExamples []*SampleValue
	// Parent returns the ancestor of this node, if any.
	Parent *Message
	// The Protobuf package this enum belongs to.
	Package string
	// Language specific annotations.
	Codec any
}

// EnumValue defines a value in an Enum.
type EnumValue struct {
	// Documentation for the message.
	Documentation string
	// Name of the attribute.
	Name string
	// ID is a unique identifier.
	ID string
	// Some source specifications allow marking enum values as deprecated.
	Deprecated bool
	// Number of the attribute.
	Number int32
	// Parent returns the ancestor of this node, if any.
	Parent *Enum
	// Language specific annotations.
	Codec any
}

// SampleValue represents a value used in a sample.
type SampleValue struct {
	// The enum value.
	EnumValue *EnumValue
	// The index of the value in the sample list (0-based).
	Index int
}
