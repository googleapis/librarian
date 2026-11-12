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

// OneOfBuilder constructs a [OneOf].
type OneOfBuilder struct {
	name          string
	id            string
	documentation string
	fields        []*Field
	exampleField  *Field
	codec         any
}

// NewOneOfBuilder creates a new [OneOfBuilder].
func NewOneOfBuilder() *OneOfBuilder {
	return &OneOfBuilder{}
}

// WithName sets the name of the one-of.
func (b *OneOfBuilder) WithName(name string) *OneOfBuilder {
	b.name = name
	return b
}

// WithID sets the unique identifier of the one-of.
func (b *OneOfBuilder) WithID(id string) *OneOfBuilder {
	b.id = id
	return b
}

// WithDocumentation sets the documentation for the one-of.
func (b *OneOfBuilder) WithDocumentation(doc string) *OneOfBuilder {
	b.documentation = doc
	return b
}

// WithFields sets the fields associated with the one-of.
func (b *OneOfBuilder) WithFields(fields ...*Field) *OneOfBuilder {
	b.fields = slices.Clone(fields)
	return b
}

// AddField appends a field to the one-of.
func (b *OneOfBuilder) AddField(field *Field) *OneOfBuilder {
	b.fields = append(b.fields, field)
	return b
}

// WithExampleField sets the example field for the one-of.
func (b *OneOfBuilder) WithExampleField(field *Field) *OneOfBuilder {
	b.exampleField = field
	return b
}

// WithAnnotations sets the codec annotations on the one-of.
func (b *OneOfBuilder) WithAnnotations(codec any) *OneOfBuilder {
	b.codec = codec
	return b
}

// Build returns the constructed [OneOf] and links its non-nil fields to it.
func (b *OneOfBuilder) Build() *OneOf {
	o := &OneOf{
		name:          b.name,
		id:            b.id,
		documentation: b.documentation,
		fields:        slices.Clone(b.fields),
		exampleField:  b.exampleField,
		codec:         b.codec,
	}
	for _, f := range o.fields {
		if f != nil {
			f.isOneOf = true
			f.group = o
		}
	}
	return o
}
