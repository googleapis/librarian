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
	// Name of the attribute.
	Name string
	// ID is a unique identifier.
	ID string
	// Documentation for the field.
	Documentation string
	// Fields associated with the one-of.
	Fields []*Field
	// The best field to show in a oneof related samples.
	// Non deprecated fields are preferred, then scalar, repeated, map fields
	// in that order.
	ExampleField *Field
	// Codec is a placeholder to put language specific annotations.
	Codec any
}
