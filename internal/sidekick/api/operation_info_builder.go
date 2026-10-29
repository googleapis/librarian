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

// OperationInfoBuilder builds an OperationInfo instance.
type OperationInfoBuilder struct {
	metadataTypeID string
	responseTypeID string
}

// NewOperationInfoBuilder creates a new OperationInfoBuilder.
func NewOperationInfoBuilder() *OperationInfoBuilder {
	return &OperationInfoBuilder{}
}

// WithMetadataTypeID sets the metadata type ID.
func (b *OperationInfoBuilder) WithMetadataTypeID(metadataTypeID string) *OperationInfoBuilder {
	b.metadataTypeID = metadataTypeID
	return b
}

// WithResponseTypeID sets the response type ID.
func (b *OperationInfoBuilder) WithResponseTypeID(responseTypeID string) *OperationInfoBuilder {
	b.responseTypeID = responseTypeID
	return b
}

// Build constructs the OperationInfo.
func (b *OperationInfoBuilder) Build() *OperationInfo {
	return &OperationInfo{
		metadataTypeID: b.metadataTypeID,
		responseTypeID: b.responseTypeID,
	}
}
