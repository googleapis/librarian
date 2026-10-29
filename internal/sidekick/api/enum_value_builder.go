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

// EnumValueBuilder constructs an [EnumValue].
type EnumValueBuilder struct {
	documentation string
	name          string
	id            string
	deprecated    bool
	number        int32
	parent        *Enum
}

// NewEnumValueBuilder creates a new [EnumValueBuilder].
func NewEnumValueBuilder() *EnumValueBuilder {
	return &EnumValueBuilder{}
}

// WithDocumentation sets the documentation for the enum value.
func (b *EnumValueBuilder) WithDocumentation(documentation string) *EnumValueBuilder {
	b.documentation = documentation
	return b
}

// WithName sets the name of the enum value.
func (b *EnumValueBuilder) WithName(name string) *EnumValueBuilder {
	b.name = name
	return b
}

// WithID sets the unique identifier of the enum value.
func (b *EnumValueBuilder) WithID(id string) *EnumValueBuilder {
	b.id = id
	return b
}

// WithDeprecated sets whether the enum value is deprecated.
func (b *EnumValueBuilder) WithDeprecated(deprecated bool) *EnumValueBuilder {
	b.deprecated = deprecated
	return b
}

// WithNumber sets the integer value of the enum value.
func (b *EnumValueBuilder) WithNumber(number int32) *EnumValueBuilder {
	b.number = number
	return b
}

// WithParent sets the parent [Enum] of the enum value.
func (b *EnumValueBuilder) WithParent(parent *Enum) *EnumValueBuilder {
	b.parent = parent
	return b
}

// Build returns the constructed [EnumValue].
func (b *EnumValueBuilder) Build() *EnumValue {
	return &EnumValue{
		documentation: b.documentation,
		name:          b.name,
		id:            b.id,
		deprecated:    b.deprecated,
		number:        b.number,
		parent:        b.parent,
	}
}
