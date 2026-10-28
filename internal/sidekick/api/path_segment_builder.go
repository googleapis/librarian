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

// PathSegmentBuilder builds a [PathSegment].
type PathSegmentBuilder struct {
	literal  string
	variable *PathVariable
}

// NewPathSegmentBuilder creates a new [PathSegmentBuilder].
func NewPathSegmentBuilder() *PathSegmentBuilder {
	return &PathSegmentBuilder{}
}

// WithLiteral sets the literal on the path segment.
func (b *PathSegmentBuilder) WithLiteral(l string) *PathSegmentBuilder {
	b.literal = l
	return b
}

// WithVariable sets the variable on the path segment.
func (b *PathSegmentBuilder) WithVariable(v *PathVariable) *PathSegmentBuilder {
	b.variable = v
	return b
}

// Build constructs a [PathSegment] from the builder.
func (b *PathSegmentBuilder) Build() *PathSegment {
	return &PathSegment{
		literal:  b.literal,
		variable: b.variable,
	}
}
