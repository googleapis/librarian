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

// ResourceBuilder constructs a [Resource].
type ResourceBuilder struct {
	typez    string
	patterns []ResourcePattern
	plural   string
	singular string
	self     *Message
}

// NewResourceBuilder creates a new [ResourceBuilder].
func NewResourceBuilder() *ResourceBuilder {
	return &ResourceBuilder{}
}

// WithType sets the resource type.
func (b *ResourceBuilder) WithType(typez string) *ResourceBuilder {
	b.typez = typez
	return b
}

// WithPatterns appends patterns to the resource.
func (b *ResourceBuilder) WithPatterns(patterns ...ResourcePattern) *ResourceBuilder {
	b.patterns = append(b.patterns, patterns...)
	return b
}

// WithPlural sets the plural form of the resource name.
func (b *ResourceBuilder) WithPlural(plural string) *ResourceBuilder {
	b.plural = plural
	return b
}

// WithSingular sets the singular form of the resource name.
func (b *ResourceBuilder) WithSingular(singular string) *ResourceBuilder {
	b.singular = singular
	return b
}

// WithSelf sets the Message that defines this resource.
func (b *ResourceBuilder) WithSelf(self *Message) *ResourceBuilder {
	b.self = self
	return b
}

// Build returns the constructed [Resource].
func (b *ResourceBuilder) Build() *Resource {
	return &Resource{
		typez:    b.typez,
		patterns: b.patterns,
		plural:   b.plural,
		singular: b.singular,
		self:     b.self,
	}
}
