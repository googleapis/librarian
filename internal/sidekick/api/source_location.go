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

// SourceLocation represents the location of an AST element in its source specification file.
type SourceLocation struct {
	// File is the path to the specification file where this element is defined.
	File string
	// Line is the 1-based starting line number in the source file.
	Line int
	// LeadingComments are the comments appearing immediately before the element.
	LeadingComments string
	// TrailingComments are the comments appearing immediately after the element.
	TrailingComments string
	// LeadingDetachedComments are detached comment blocks preceding the element.
	LeadingDetachedComments []string
}
