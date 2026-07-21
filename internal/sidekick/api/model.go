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

// Package api defines the data model representing a parsed API surface.
package api

const (
	// ReservedPackageName is a package name reserved for maps and other
	// synthetic messages that do not exist in the input specification.
	//
	// We need a place to put these in the data model without conflicts with the
	// input data model. This symbol is unused in all the IDLs we support.
	ReservedPackageName = "$"

	// StandardFieldNameForResourceRef is the standard name for the resource
	// field.
	//
	// That is, the field that contains the resource name that an RPC operates
	// on. This is one of the AIP requirements / recommendations for standard
	// RPCs.
	StandardFieldNameForResourceRef = "name"

	// StandardFieldNameForParentResourceRef is the standard name for the child
	// resource field.
	//
	// That is, the field that contains the child resource name that an RPC
	// operates on. This is one of the AIP requirements / recommendations for
	// standard RPCs.
	StandardFieldNameForParentResourceRef = "parent"

	// GenericResourceType is a special resource type that may be used by resource references
	// in contexts where the referenced resource may be of any type, as defined by AIPs.
	GenericResourceType = "*"

	// StandardFieldNameForUpdateMask is the standard name for the update mask field
	// in update operations as defined by AIP-134.
	StandardFieldNameForUpdateMask = "update_mask"
)
