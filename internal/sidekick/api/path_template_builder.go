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

// PathTemplateBuilder builds a [PathTemplate].
type PathTemplateBuilder struct {
	segments []PathSegment
	verb     string
}

// NewPathTemplateBuilder creates a new [PathTemplateBuilder].
func NewPathTemplateBuilder() *PathTemplateBuilder {
	return &PathTemplateBuilder{}
}

// WithSegments sets the segments on the path template.
func (b *PathTemplateBuilder) WithSegments(segments ...PathSegment) *PathTemplateBuilder {
	b.segments = segments
	return b
}

// WithSegment appends a segment to the path template.
func (b *PathTemplateBuilder) WithSegment(segment PathSegment) *PathTemplateBuilder {
	b.segments = append(b.segments, segment)
	return b
}

// WithLiteral adds a literal to the path template.
func (b *PathTemplateBuilder) WithLiteral(l string) *PathTemplateBuilder {
	b.segments = append(b.segments, *NewPathSegmentBuilder().WithLiteral(l).Build())
	return b
}

// WithVariable adds a variable to the path template.
func (b *PathTemplateBuilder) WithVariable(v *PathVariable) *PathTemplateBuilder {
	b.segments = append(b.segments, *NewPathSegmentBuilder().WithVariable(v).Build())
	return b
}

// WithVariableNamed adds a variable with the given name to the path template.
func (b *PathTemplateBuilder) WithVariableNamed(fields ...string) *PathTemplateBuilder {
	v := NewPathVariableBuilder(fields...).WithMatch().Build()
	b.segments = append(b.segments, *NewPathSegmentBuilder().WithVariable(v).Build())
	return b
}

// WithVerb sets the verb on the path template.
func (b *PathTemplateBuilder) WithVerb(v string) *PathTemplateBuilder {
	b.verb = v
	return b
}

// Build constructs a [PathTemplate] from the builder.
func (b *PathTemplateBuilder) Build() *PathTemplate {
	return &PathTemplate{
		segments: b.segments,
		verb:     b.verb,
	}
}
