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

// OneOf is a group of fields that are mutually exclusive. Notably, proto3 optional
// fields are all their own one-of.
type OneOf struct {
	name          string
	id            string
	documentation string
	fields        []*Field
	exampleField  *Field
	codec         any
}

// Name returns the name of the attribute.
func (o *OneOf) Name() string {
	return o.name
}

// ID returns a unique identifier.
func (o *OneOf) ID() string {
	return o.id
}

// Documentation returns the documentation for the field.
func (o *OneOf) Documentation() string {
	return o.documentation
}

// Fields returns the fields associated with the one-of.
func (o *OneOf) Fields() []*Field {
	return o.fields
}

// ExampleField returns the best field to show in a oneof related samples.
// Non deprecated fields are preferred, then scalar, repeated, map fields
// in that order.
func (o *OneOf) ExampleField() *Field {
	return o.exampleField
}

// Codec returns a placeholder to put language specific annotations.
func (o *OneOf) Codec() any {
	return o.codec
}

// SetAnnotations sets the language-specific codec annotations on the one-of.
func (o *OneOf) SetAnnotations(codec any) {
	o.codec = codec
}
