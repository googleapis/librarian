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
	Name string
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
	Variants []*RoutingInfoVariant
}

// RoutingInfoVariant represents the routing information stripped of its name.
type RoutingInfoVariant struct {
	// The sequence of field names accessed to get the routing information.
	FieldPath []string
	// A path template that must match the beginning of the field value.
	Prefix RoutingPathSpec
	// A path template that, if matching, is used in the `x-goog-request-params`.
	Matching RoutingPathSpec
	// A path template that must match the end of the field value.
	Suffix RoutingPathSpec
	// Language specific information
	Codec any
}

// FieldName returns the field path as a string.
func (v *RoutingInfoVariant) FieldName() string {
	return strings.Join(v.FieldPath, ".")
}

// TemplateAsString returns the template as a string.
func (v *RoutingInfoVariant) TemplateAsString() string {
	var full []string
	full = append(full, v.Prefix.Segments...)
	full = append(full, v.Matching.Segments...)
	full = append(full, v.Suffix.Segments...)
	return strings.Join(full, "/")
}

// RoutingPathSpec is a specification for a routing path.
type RoutingPathSpec struct {
	// A sequence of matching segments.
	//
	// A template like `projects/*/location/*/**` maps to
	// `["projects", "*", "locations", "*", "**"]`.
	Segments []string
}
