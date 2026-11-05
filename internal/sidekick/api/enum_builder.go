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

// EnumBuilder constructs an [Enum].
type EnumBuilder struct {
	documentation      string
	name               string
	id                 string
	deprecated         bool
	values             []*EnumValue
	uniqueNumberValues []*EnumValue
	valuesForExamples  []*SampleValue
	parent             *Message
	packagez           string
}

// NewEnumBuilder creates a new [EnumBuilder].
func NewEnumBuilder() *EnumBuilder {
	return &EnumBuilder{}
}

// WithDocumentation sets the documentation for the enum.
func (b *EnumBuilder) WithDocumentation(documentation string) *EnumBuilder {
	b.documentation = documentation
	return b
}

// WithName sets the name of the enum.
func (b *EnumBuilder) WithName(name string) *EnumBuilder {
	b.name = name
	return b
}

// WithID sets the unique identifier of the enum.
func (b *EnumBuilder) WithID(id string) *EnumBuilder {
	b.id = id
	return b
}

// WithDeprecated sets whether the enum is deprecated.
func (b *EnumBuilder) WithDeprecated(deprecated bool) *EnumBuilder {
	b.deprecated = deprecated
	return b
}

// WithValues appends values to the enum.
func (b *EnumBuilder) WithValues(values ...*EnumValue) *EnumBuilder {
	b.values = append(b.values, values...)
	return b
}

// WithUniqueNumberValues sets the unique-number values of the enum.
func (b *EnumBuilder) WithUniqueNumberValues(values ...*EnumValue) *EnumBuilder {
	b.uniqueNumberValues = append(b.uniqueNumberValues, values...)
	return b
}

// WithValuesForExamples sets the example values of the enum.
func (b *EnumBuilder) WithValuesForExamples(values ...*SampleValue) *EnumBuilder {
	b.valuesForExamples = append(b.valuesForExamples, values...)
	return b
}

// WithParent sets the parent [Message] of the enum.
func (b *EnumBuilder) WithParent(parent *Message) *EnumBuilder {
	b.parent = parent
	return b
}

// WithPackage sets the package of the enum.
func (b *EnumBuilder) WithPackage(packagez string) *EnumBuilder {
	b.packagez = packagez
	return b
}

// Build returns the constructed [Enum].
func (b *EnumBuilder) Build() *Enum {
	e := &Enum{
		documentation:      b.documentation,
		name:               b.name,
		id:                 b.id,
		deprecated:         b.deprecated,
		values:             b.values,
		uniqueNumberValues: b.uniqueNumberValues,
		valuesForExamples:  b.valuesForExamples,
		parent:             b.parent,
		packagez:           b.packagez,
	}
	for _, v := range e.values {
		if v.parent == nil {
			v.parent = e
		}
	}
	return e
}
