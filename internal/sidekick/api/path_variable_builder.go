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

// PathVariableBuilder builds a [PathVariable].
type PathVariableBuilder struct {
	fieldPath     []string
	segments      []string
	allowReserved bool
}

// NewPathVariableBuilder creates a new [PathVariableBuilder] with the given field path.
func NewPathVariableBuilder(fields ...string) *PathVariableBuilder {
	return &PathVariableBuilder{fieldPath: fields}
}

// WithFieldPath sets the field path for the path variable.
func (b *PathVariableBuilder) WithFieldPath(fields ...string) *PathVariableBuilder {
	b.fieldPath = fields
	return b
}

// WithSegments appends segments to the path variable.
func (b *PathVariableBuilder) WithSegments(segments ...string) *PathVariableBuilder {
	b.segments = append(b.segments, segments...)
	return b
}

// WithLiteral adds a literal to the path variable.
func (b *PathVariableBuilder) WithLiteral(l string) *PathVariableBuilder {
	b.segments = append(b.segments, l)
	return b
}

// WithMatchRecursive adds a recursive match to the path variable.
func (b *PathVariableBuilder) WithMatchRecursive() *PathVariableBuilder {
	b.segments = append(b.segments, MultiSegmentWildcard)
	return b
}

// WithMatch adds a match to the path variable.
func (b *PathVariableBuilder) WithMatch() *PathVariableBuilder {
	b.segments = append(b.segments, SingleSegmentWildcard)
	return b
}

// WithAllowReserved marks the variable as allowing reserved characters to remain unescaped.
func (b *PathVariableBuilder) WithAllowReserved() *PathVariableBuilder {
	b.allowReserved = true
	return b
}

// Build constructs a [PathVariable] from the builder.
func (b *PathVariableBuilder) Build() *PathVariable {
	return &PathVariable{
		fieldPath:     b.fieldPath,
		segments:      b.segments,
		allowReserved: b.allowReserved,
	}
}
