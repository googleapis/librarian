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

// PathInfoBuilder builds a [PathInfo].
type PathInfoBuilder struct {
	bindings      []*PathBinding
	bodyFieldPath string
}

// NewPathInfoBuilder creates a new [PathInfoBuilder].
func NewPathInfoBuilder() *PathInfoBuilder {
	return &PathInfoBuilder{}
}

// WithBindings sets the path bindings on the [PathInfo].
func (b *PathInfoBuilder) WithBindings(bindings ...*PathBinding) *PathInfoBuilder {
	b.bindings = bindings
	return b
}

// WithBinding appends a path binding to the [PathInfo].
func (b *PathInfoBuilder) WithBinding(binding *PathBinding) *PathInfoBuilder {
	b.bindings = append(b.bindings, binding)
	return b
}

// WithBodyFieldPath sets the body field path on the [PathInfo].
func (b *PathInfoBuilder) WithBodyFieldPath(bodyFieldPath string) *PathInfoBuilder {
	b.bodyFieldPath = bodyFieldPath
	return b
}

// Build constructs a [PathInfo] from the builder.
func (b *PathInfoBuilder) Build() *PathInfo {
	return &PathInfo{
		bindings:      b.bindings,
		bodyFieldPath: b.bodyFieldPath,
	}
}
