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

// FieldBehavior represents annotations for how the code generator handles a
// field.
//
// Regardless of the underlying data type and whether it is required or optional
// on the wire, some fields must be present for requests to succeed. Or may not
// be included in a request.
type FieldBehavior int

const (
	// FieldBehaviorUnspecified is the default, unspecified field behavior.
	FieldBehaviorUnspecified FieldBehavior = iota

	// FieldBehaviorOptional specifically denotes a field as optional.
	//
	// While Google Cloud uses proto3, where fields are either optional or have
	// a default value, this may be specified for emphasis.
	FieldBehaviorOptional

	// FieldBehaviorRequired denotes a field as required.
	//
	// This indicates that the field **must** be provided as part of the request,
	// and failure to do so will cause an error (usually `INVALID_ARGUMENT`).
	//
	// Code generators may change the generated types to include this field as a
	// parameter necessary to construct the request.
	FieldBehaviorRequired

	// FieldBehaviorOutputOnly denotes a field as output only.
	//
	// Some messages (and their fields) are used in both requests and responses.
	// This indicates that the field is provided in responses, but including the
	// field in a request does nothing (the server *must* ignore it and
	// *must not* throw an error as a result of the field's presence).
	//
	// Code generators that use different builders for "the message as part of a
	// request" vs. "the standalone message" may omit this field in the former.
	FieldBehaviorOutputOnly

	// FieldBehaviorInputOnly denotes a field as input only.
	//
	// This indicates that the field is provided in requests, and the
	// corresponding field is not included in output.
	FieldBehaviorInputOnly

	// FieldBehaviorImmutable denotes a field as immutable.
	//
	// This indicates that the field may be set once in a request to create a
	// resource, but may not be changed thereafter.
	FieldBehaviorImmutable

	// FieldBehaviorUnorderedList denotes that a (repeated) field is an unordered list.
	//
	// This indicates that the service may provide the elements of the list
	// in any arbitrary  order, rather than the order the user originally
	// provided. Additionally, the list's order may or may not be stable.
	FieldBehaviorUnorderedList

	// FieldBehaviorUnorderedNonEmptyDefault denotes that this field returns a non-empty default value if not set.
	//
	// This indicates that if the user provides the empty value in a request,
	// a non-empty value will be returned. The user will not be aware of what
	// non-empty value to expect.
	FieldBehaviorUnorderedNonEmptyDefault

	// FieldBehaviorIdentifier denotes that the field in a resource (a message annotated with
	// google.api.resource) is used in the resource name to uniquely identify the
	// resource.
	//
	// For AIP-compliant APIs, this should only be applied to the
	// `name` field on the resource.
	//
	// This behavior should not be applied to references to other resources within
	// the message.
	//
	// The identifier field of resources often have different field behavior
	// depending on the request it is embedded in (e.g. for Create methods name
	// is optional and unused, while for Update methods it is required). Instead
	// of method-specific annotations, only `IDENTIFIER` is required.
	FieldBehaviorIdentifier
)
