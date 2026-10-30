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

package rust

import (
	"fmt"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

// ProstWktType maps a Protobuf well-known type ID to its Prost extern_path
// expression in build.rs and its Rust type name in convert-prost ToProto calls.
type ProstWktType struct {
	ProtoPath   string
	ExternPath  string
	ToProtoType string
}

// ProstWktTypes returns the well-known type mappings shared between rust_prost
// (for prost_build extern_path configuration in build.rs) and rust (for
// convert-prost ToProto type qualification).
//
// This replicates prost_build's default well-known type mappings, except
// BytesValue is mapped to bytes::Bytes to match wkt::BytesValue. See
// https://github.com/tokio-rs/prost/blob/13646cde7eab75c81b3047767aa0a86e7dbecf12/prost-build/src/extern_paths.rs#L39
func ProstWktTypes() []ProstWktType {
	return []ProstWktType{
		{ProtoPath: api.WktBoolValueID, ExternPath: `"bool"`, ToProtoType: "bool"},
		{ProtoPath: api.WktBytesValueID, ExternPath: `format!("{prost_path}::bytes::Bytes")`, ToProtoType: "::bytes::Bytes"},
		{ProtoPath: api.WktDoubleValueID, ExternPath: `"f64"`, ToProtoType: "f64"},
		{ProtoPath: api.WktEmptyID, ExternPath: `"()"`, ToProtoType: "gaxi::prost::Empty"},
		{ProtoPath: api.WktFloatValueID, ExternPath: `"f32"`, ToProtoType: "f32"},
		{ProtoPath: api.WktInt32ValueID, ExternPath: `"i32"`, ToProtoType: "i32"},
		{ProtoPath: api.WktInt64ValueID, ExternPath: `"i64"`, ToProtoType: "i64"},
		{ProtoPath: api.WktStringValueID, ExternPath: `format!("{prost_path}::alloc::string::String")`, ToProtoType: "std::string::String"},
		{ProtoPath: api.WktUInt32ValueID, ExternPath: `"u32"`, ToProtoType: "u32"},
		{ProtoPath: api.WktUInt64ValueID, ExternPath: `"u64"`, ToProtoType: "u64"},
	}
}

// prostFieldType returns the Rust type name of the field in the prost-generated
// module, used to qualify `gaxi::prost::ToProto::<ProstType>::to_proto` calls in
// convert-prost templates.
func (c *codec) prostFieldType(f *api.Field, model *api.API) (string, error) {
	switch f.Typez() {
	case api.TypezMessage:
		for _, wkt := range ProstWktTypes() {
			if f.TypezID() == wkt.ProtoPath {
				return wkt.ToProtoType, nil
			}
		}
		m := model.Message(f.TypezID())
		if m == nil {
			return "", fmt.Errorf("unable to lookup field type (%q) for field %s", f.TypezID(), f.ID())
		}
		return c.qualifyProstPath(m.Package, prostMessageRelativePath(m), model.PackageName), nil
	case api.TypezEnum:
		e := model.Enum(f.TypezID())
		if e == nil {
			return "", fmt.Errorf("unable to lookup field type (%q) for field %s", f.TypezID(), f.ID())
		}
		return c.qualifyProstPath(e.Package(), prostEnumRelativePath(e), model.PackageName), nil
	case api.TypezGroup, api.TypezMap:
		return "", nil
	default:
		return scalarFieldType(f)
	}
}

// prostTypeName returns the ProstRelativeName for a message, enum, or oneof.
// In convert-prost mode (or when prost-path is set), external types are
// package-qualified via qualifyProstPath; otherwise, the package-relative path
// is returned for transport and mock templates that prefix PackageModuleName.
func (c *codec) prostTypeName(pkg, rel, modelPackage string) string {
	if c.templateOverride == "templates/convert-prost" || c.prostPath != "" {
		return c.qualifyProstPath(pkg, rel, modelPackage)
	}
	return rel
}

// qualifyProstPath returns the Rust path to a prost-generated type with relative
// path rel in Protobuf package pkg when referenced from a model in modelPackage.
func (c *codec) qualifyProstPath(pkg, rel, modelPackage string) string {
	// Check pkg == modelPackage before pkg == "google.protobuf" so that when
	// generating conversions for the google.protobuf package itself (e.g.,
	// google-cloud-wkt), same-package types resolve to their local relative path
	// rather than prost_types::<rel>.
	if pkg == "" || pkg == modelPackage || pkg == api.ReservedPackageName {
		return rel
	}
	if pkg == "google.protobuf" {
		return "prost_types::" + rel
	}
	prostPrefix := "crate::"
	if c.prostPath != "" {
		prostPrefix = c.prostPath + "::"
	}
	return prostPrefix + packageToModuleName(pkg) + "::" + rel
}
