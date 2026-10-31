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

// RoutingInfoVariantBuilder builds a RoutingInfoVariant instance.
type RoutingInfoVariantBuilder struct {
	fieldPath []string
	prefix    RoutingPathSpec
	matching  RoutingPathSpec
	suffix    RoutingPathSpec
}

// NewRoutingInfoVariantBuilder creates a new RoutingInfoVariantBuilder.
func NewRoutingInfoVariantBuilder() *RoutingInfoVariantBuilder {
	return &RoutingInfoVariantBuilder{}
}

// WithFieldPath sets the sequence of field names accessed to get the routing information.
func (b *RoutingInfoVariantBuilder) WithFieldPath(fieldPath ...string) *RoutingInfoVariantBuilder {
	b.fieldPath = fieldPath
	return b
}

// WithPrefix sets the prefix path template.
func (b *RoutingInfoVariantBuilder) WithPrefix(prefix RoutingPathSpec) *RoutingInfoVariantBuilder {
	b.prefix = prefix
	return b
}

// WithPrefixSegments sets the prefix path template from segments.
func (b *RoutingInfoVariantBuilder) WithPrefixSegments(segments ...string) *RoutingInfoVariantBuilder {
	b.prefix = NewRoutingPathSpec(segments...)
	return b
}

// WithMatching sets the matching path template.
func (b *RoutingInfoVariantBuilder) WithMatching(matching RoutingPathSpec) *RoutingInfoVariantBuilder {
	b.matching = matching
	return b
}

// WithMatchingSegments sets the matching path template from segments.
func (b *RoutingInfoVariantBuilder) WithMatchingSegments(segments ...string) *RoutingInfoVariantBuilder {
	b.matching = NewRoutingPathSpec(segments...)
	return b
}

// WithSuffix sets the suffix path template.
func (b *RoutingInfoVariantBuilder) WithSuffix(suffix RoutingPathSpec) *RoutingInfoVariantBuilder {
	b.suffix = suffix
	return b
}

// WithSuffixSegments sets the suffix path template from segments.
func (b *RoutingInfoVariantBuilder) WithSuffixSegments(segments ...string) *RoutingInfoVariantBuilder {
	b.suffix = NewRoutingPathSpec(segments...)
	return b
}

// Build constructs the RoutingInfoVariant.
func (b *RoutingInfoVariantBuilder) Build() *RoutingInfoVariant {
	return &RoutingInfoVariant{
		fieldPath: b.fieldPath,
		prefix:    b.prefix,
		matching:  b.matching,
		suffix:    b.suffix,
	}
}
