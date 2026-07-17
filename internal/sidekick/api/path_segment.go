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

const (
	// SingleSegmentWildcard is a special routing path segment which indicates
	// "match anything that does not include a `/`".
	SingleSegmentWildcard = "*"

	// MultiSegmentWildcard is a special routing path segment which indicates
	// "match anything including `/`".
	MultiSegmentWildcard = "**"
)

// PathSegment is a segment of a path.
type PathSegment struct {
	Literal  string
	Variable *PathVariable
}

// WithLiteral adds a literal to the path segment.
func (s *PathSegment) WithLiteral(l string) *PathSegment {
	s.Literal = l
	return s
}

// WithVariable adds a variable to the path segment.
func (s *PathSegment) WithVariable(v *PathVariable) *PathSegment {
	s.Variable = v
	return s
}
