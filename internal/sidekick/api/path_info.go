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

// PathInfo contains normalized request path information.
type PathInfo struct {
	// The list of bindings, including the top-level binding.
	bindings []*PathBinding
	// bodyFieldPath is the name of the field that should be used as the body of the
	// request.
	//
	// This is a string that may be "*" which indicates that the entire request
	// should be used as the body.
	//
	// If this is empty then the body is not used.
	bodyFieldPath string
	// Language specific annotations.
	codec any
}

// Bindings returns the list of path bindings, including the top-level binding.
func (p *PathInfo) Bindings() []*PathBinding {
	return slices.Clone(p.bindings)
}

// BodyFieldPath returns the name of the field that should be used as the body of the request.
func (p *PathInfo) BodyFieldPath() string {
	return p.bodyFieldPath
}

// Codec returns the language-specific annotations for the path info.
func (p *PathInfo) Codec() any {
	return p.codec
}

// SetAnnotations sets the language-specific annotations for the path info.
func (p *PathInfo) SetAnnotations(codec any) {
	p.codec = codec
}
