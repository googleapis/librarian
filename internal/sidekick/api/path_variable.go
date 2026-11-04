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

import "slices"

// PathVariable is a variable in a path.
type PathVariable struct {
	fieldPath []string
	segments  []string
	// Allow characters defined as `reserved` by RFC-6570 1.5 to pass through without
	// percent encoding. See RFC-6570 1.2 for examples.
	allowReserved bool
}

// FieldPath returns the sequence of field names leading to the variable.
func (v *PathVariable) FieldPath() []string {
	return slices.Clone(v.fieldPath)
}

// Segments returns the matching segments of the path variable.
func (v *PathVariable) Segments() []string {
	return slices.Clone(v.segments)
}

// AllowReserved returns whether reserved characters are allowed to pass through without percent encoding.
func (v *PathVariable) AllowReserved() bool {
	return v.allowReserved
}
