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

// OperationInfo contains normalized long running operation info.
type OperationInfo struct {
	// The metadata type. If there is no metadata, this is set to
	// `.google.protobuf.Empty`.
	metadataTypeID string
	// The result type. This is the expected type when the long running
	// operation completes successfully.
	responseTypeID string
	// The method.
	method *Method
	// Language specific annotations.
	codec any
}

// MetadataTypeID returns the metadata type ID. If there is no metadata, this is
// set to `.google.protobuf.Empty`.
func (o *OperationInfo) MetadataTypeID() string {
	return o.metadataTypeID
}

// ResponseTypeID returns the result type ID expected when the long running
// operation completes successfully.
func (o *OperationInfo) ResponseTypeID() string {
	return o.responseTypeID
}

// Method returns the parent method.
func (o *OperationInfo) Method() *Method {
	return o.method
}

// Codec returns the language-specific annotations.
func (o *OperationInfo) Codec() any {
	return o.codec
}

// SetAnnotations sets the language-specific annotations.
func (o *OperationInfo) SetAnnotations(codec any) {
	o.codec = codec
}
