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

// PathVariable is a variable in a path.
type PathVariable struct {
	FieldPath []string
	Segments  []string
	// Allow characters defined as `reserved` by RFC-6570 1.5 to pass through without
	// percent encoding. See RFC-6570 1.2 for examples.
	AllowReserved bool
}

// NewPathVariable creates a new path variable.
func NewPathVariable(fields ...string) *PathVariable {
	return &PathVariable{FieldPath: fields}
}

// WithLiteral adds a literal to the path variable.
func (v *PathVariable) WithLiteral(l string) *PathVariable {
	v.Segments = append(v.Segments, l)
	return v
}

// WithMatchRecursive adds a recursive match to the path variable.
func (v *PathVariable) WithMatchRecursive() *PathVariable {
	v.Segments = append(v.Segments, MultiSegmentWildcard)
	return v
}

// WithMatch adds a match to the path variable.
func (v *PathVariable) WithMatch() *PathVariable {
	v.Segments = append(v.Segments, SingleSegmentWildcard)
	return v
}

// WithAllowReserved marks the variable as allowing reserved characters to remain unescaped.
func (v *PathVariable) WithAllowReserved() *PathVariable {
	v.AllowReserved = true
	return v
}
