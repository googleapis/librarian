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

// ResourceReferenceBuilder builds a [ResourceReference].
type ResourceReferenceBuilder struct {
	r ResourceReference
}

// NewResourceReferenceBuilder creates a new [ResourceReferenceBuilder].
func NewResourceReferenceBuilder() *ResourceReferenceBuilder {
	return &ResourceReferenceBuilder{}
}

// WithType sets the referenced resource type.
func (b *ResourceReferenceBuilder) WithType(typez string) *ResourceReferenceBuilder {
	b.r.typez = typez
	return b
}

// WithChildType sets the referenced child resource type.
func (b *ResourceReferenceBuilder) WithChildType(childType string) *ResourceReferenceBuilder {
	b.r.childType = childType
	return b
}

// Build returns the constructed [ResourceReference].
func (b *ResourceReferenceBuilder) Build() *ResourceReference {
	r := b.r
	return &r
}
