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

import "strings"

// RoutingInfo contains normalized routing info.
//
// The routing information format is documented in:
//
// https://google.aip.dev/client-libraries/4222
//
// At a high level, it consists of a field name (from the request) that is used
// to match a certain path template. If the value of the field matches the
// template, the matching portion is added to `x-goog-request-params`.
//
// An empty `Name` field is used as the special marker to cover this case in
// AIP-4222:
//
//	An empty google.api.routing annotation is acceptable. It means that no
//	routing headers should be generated for the RPC, when they otherwise
//	would be e.g. implicitly from the google.api.http annotation.
type RoutingInfo struct {
	// The name in `x-goog-request-params`.
	name string
	// Group the possible variants for the given name.
	//
	// The variants are parsed into the reverse order of definition. AIP-4222
	// declares:
	//
	//   In cases when multiple routing parameters have the same resource ID
	//   path segment name, thus referencing the same header key, the
	//   "last one wins" rule is used to determine which value to send.
	//
	// Reversing the order allows us to implement "the first match wins". That
	// is easier and more efficient in most languages.
	variants []*RoutingInfoVariant
}

// Name returns the name in `x-goog-request-params`.
func (r *RoutingInfo) Name() string {
	return r.name
}

// Variants returns the possible variants for the given name.
func (r *RoutingInfo) Variants() []*RoutingInfoVariant {
	return r.variants
}

// RoutingInfoVariant represents the routing information stripped of its name.
type RoutingInfoVariant struct {
	// The sequence of field names accessed to get the routing information.
	fieldPath []string
	// A path template that must match the beginning of the field value.
	prefix RoutingPathSpec
	// A path template that, if matching, is used in the `x-goog-request-params`.
	matching RoutingPathSpec
	// A path template that must match the end of the field value.
	suffix RoutingPathSpec
	// Language specific information
	codec any
}

// FieldPath returns the sequence of field names accessed to get the routing information.
func (v *RoutingInfoVariant) FieldPath() []string {
	return v.fieldPath
}

// Prefix returns a path template that must match the beginning of the field value.
func (v *RoutingInfoVariant) Prefix() RoutingPathSpec {
	return v.prefix
}

// Matching returns a path template that, if matching, is used in `x-goog-request-params`.
func (v *RoutingInfoVariant) Matching() RoutingPathSpec {
	return v.matching
}

// Suffix returns a path template that must match the end of the field value.
func (v *RoutingInfoVariant) Suffix() RoutingPathSpec {
	return v.suffix
}

// Codec returns the language-specific annotations.
func (v *RoutingInfoVariant) Codec() any {
	return v.codec
}

// SetAnnotations sets the language-specific annotations.
func (v *RoutingInfoVariant) SetAnnotations(codec any) {
	v.codec = codec
}

// FieldName returns the field path as a string.
func (v *RoutingInfoVariant) FieldName() string {
	return strings.Join(v.fieldPath, ".")
}

// TemplateAsString returns the template as a string.
func (v *RoutingInfoVariant) TemplateAsString() string {
	var full []string
	full = append(full, v.prefix.segments...)
	full = append(full, v.matching.segments...)
	full = append(full, v.suffix.segments...)
	return strings.Join(full, "/")
}

// RoutingPathSpec is a specification for a routing path.
type RoutingPathSpec struct {
	// A sequence of matching segments.
	//
	// A template like `projects/*/location/*/**` maps to
	// `["projects", "*", "locations", "*", "**"]`.
	segments []string
}

// NewRoutingPathSpec creates a new RoutingPathSpec with the given segments.
func NewRoutingPathSpec(segments ...string) RoutingPathSpec {
	return RoutingPathSpec{segments: segments}
}

// Segments returns the sequence of matching segments.
func (s RoutingPathSpec) Segments() []string {
	return s.segments
}
