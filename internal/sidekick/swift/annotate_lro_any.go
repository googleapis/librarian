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

package swift

import (
	"cmp"
	"fmt"
	"log/slog"
	"slices"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

const (
	// The message ID for the long-running operation type.
	longRunningOperationID = ".google.longrunning.Operation"
	// The message ID for the `Any` well-known type.
	protobufAnyID = ".google.protobuf.Any"
	// The Swift module that provides the Protobuf types for the well-known
	// types. Unlike the types of an API, they are not compiled into the
	// generated Protobuf module.
	swiftProtobufModule = "SwiftProtobuf"
)

// lroAnyType is one entry in the long-running operation `Any` converter.
//
// Each entry maps a type URL to the pair of types needed to convert a payload
// without consulting SwiftProtobuf's global type registry: the native Swift
// type and the SwiftProtobuf type it converts from.
type lroAnyType struct {
	// TypeURL is the `Any` type URL of the payload (e.g.
	// "type.googleapis.com/google.storage.control.v2.RenameFolderMetadata").
	TypeURL string

	// ParameterTypeName is the native Swift type (e.g. "RenameFolderMetadata").
	ParameterTypeName string

	// ProtoTypeName is the SwiftProtobuf type (e.g.
	// "StorageControlProtos.Google_Storage_Control_V2_RenameFolderMetadata").
	ProtoTypeName string
}

// lroAnyTypes returns the payload types that the package's long-running
// operations can carry, sorted by type URL and deduplicated.
//
// A long-running operation carries its metadata and its response in an `Any`.
// The services declare both types up front, so the set is closed and the
// generator can emit a converter for it. This mirrors `LROTypes` in the Rust
// codec.
//
// The methods are those of every service in the package, not one service:
// there is a single converter, so its table has to cover all of them.
func (c *codec) lroAnyTypes(methods []*api.Method) ([]*lroAnyType, error) {
	seen := map[string]bool{}
	var types []*lroAnyType
	for _, method := range methods {
		if method.OperationInfo == nil {
			continue
		}
		for _, id := range []string{
			method.OperationInfo.MetadataTypeID,
			method.OperationInfo.ResponseTypeID,
		} {
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			message, err := lookupMessage(c.Model, id)
			if err != nil {
				return nil, err
			}
			annotations, ok := message.Codec.(*messageAnnotations)
			if !ok {
				// The message is not generated in this module, so there is no
				// conversion to call. Leaving it out of the table sends the
				// payload down the converter's generic path at runtime, which
				// is the best available outcome: generating a call to a type
				// that does not exist would not compile.
				slog.Warn("omitting a long-running operation payload type from the `Any` converter because it is not generated in this module",
					"type", id, "converter", c.LROAnyConverter)
				continue
			}
			types = append(types, &lroAnyType{
				TypeURL:           annotations.TypeURL,
				ParameterTypeName: annotations.ParameterTypeName,
				ProtoTypeName:     c.lroAnyProtoTypeName(message, annotations),
			})
		}
	}
	slices.SortFunc(types, func(a, b *lroAnyType) int {
		return cmp.Compare(a.TypeURL, b.TypeURL)
	})
	return types, nil
}

// lroAnyProtoTypeName returns the SwiftProtobuf type that a payload decodes
// into.
//
// The message annotations qualify this type with the generated Protobuf module,
// which is correct for the types of the API being generated. The well-known
// types are not compiled into that module, so they keep the names SwiftProtobuf
// ships.
func (c *codec) lroAnyProtoTypeName(m *api.Message, annotations *messageAnnotations) string {
	if m.Package != wellKnownProtobufPackage {
		return annotations.ProtoTypeName
	}
	return fmt.Sprintf("%s.%s%s", swiftProtobufModule, ProtoPackagePrefix(m.Package), pascalCase(m.Name))
}

// annotateLROAnyFields routes the `Any` fields of
// `google.longrunning.Operation` through the configured converter.
//
// Only `Operation` is affected. Other messages with `Any` fields, such as
// `google.rpc.Status`, keep the generic conversion: their payload types are not
// declared anywhere the generator can see, so there is no table to dispatch on.
func (c *codec) annotateLROAnyFields() {
	if c.LROAnyConverter == "" {
		return
	}
	operation := c.Model.Message(longRunningOperationID)
	if operation == nil {
		// This module does not convert `Operation`, so there is nothing to do.
		return
	}
	for _, field := range operation.Fields {
		if field.Typez != api.TypezMessage || field.TypezID != protobufAnyID {
			continue
		}
		annotations, ok := field.Codec.(*fieldAnnotations)
		if !ok {
			continue
		}
		annotations.LROAnyConverter = c.LROAnyConverter
	}
}

// annotateLROAnyConverter collects the payload types for the package's
// long-running operation `Any` converter.
//
// There is one converter per package. The `Operation` fields annotated by
// `annotateLROAnyFields` name a single converter, so the payload types of
// every service have to be unioned into its table; a per-service converter
// would leave the types of all but one service unreachable. The Rust codec
// unions the same way, in its transport template.
func (c *codec) annotateLROAnyConverter(annotations *modelAnnotations, methods []*api.Method) error {
	// Only the libraries that route `Operation` through a converter need the
	// dispatch table. Everywhere else it would be an unused type.
	if c.LROAnyConverter == "" {
		return nil
	}
	types, err := c.lroAnyTypes(methods)
	if err != nil {
		return err
	}
	if len(types) == 0 {
		return nil
	}
	annotations.LROAnyTypes = types
	annotations.LROAnyConverterName = c.LROAnyConverter
	return nil
}
