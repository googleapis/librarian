// Copyright 2024 Google LLC
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

// MethodSignature defines an alternative signature for the method.
//
// Some methods include annotations to generate additional overloads for the
// corresponding generated code. That is, while most generators emit a single
// function for each RPC, these annotations may define additional overloads for
// the same RPC.
//
// These overloads select a subset of the fields in the request message, and the
// generator emits a function for each list.
type MethodSignature struct {
	// Names define the list of field names from the request message included in
	// this signature.
	Names []string

	// Fields define the list of fields from the request message included in
	// this signature.
	//
	// This is initialized in the cross-reference phase, as the fields may not
	// exist when the method is first parsed.
	Fields []*Field

	// Method cross-references the method containing this signature.
	//
	// This is useful in mustache templates, as the template can access the
	// method and the method's annotations from within a signature.
	Method *Method
}
