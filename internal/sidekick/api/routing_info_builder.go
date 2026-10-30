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

// RoutingInfoBuilder builds a RoutingInfo instance.
type RoutingInfoBuilder struct {
	name     string
	variants []*RoutingInfoVariant
}

// NewRoutingInfoBuilder creates a new RoutingInfoBuilder.
func NewRoutingInfoBuilder() *RoutingInfoBuilder {
	return &RoutingInfoBuilder{}
}

// WithName sets the name in `x-goog-request-params`.
func (b *RoutingInfoBuilder) WithName(name string) *RoutingInfoBuilder {
	b.name = name
	return b
}

// WithVariant appends a variant to the routing info.
func (b *RoutingInfoBuilder) WithVariant(variant *RoutingInfoVariant) *RoutingInfoBuilder {
	b.variants = append(b.variants, variant)
	return b
}

// WithVariants sets the variants of the routing info.
func (b *RoutingInfoBuilder) WithVariants(variants ...*RoutingInfoVariant) *RoutingInfoBuilder {
	b.variants = variants
	return b
}

// Build constructs the RoutingInfo.
func (b *RoutingInfoBuilder) Build() *RoutingInfo {
	return &RoutingInfo{
		name:     b.name,
		variants: b.variants,
	}
}
