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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGenerateConversions_MissingModulePath(t *testing.T) {
	outDir := t.TempDir()
	model := api.NewTestAPI(nil, nil, nil).WithPackageName("google.cloud.test.v1")

	err := GenerateConversions(t.Context(), model, outDir, &config.Library{}, nil)
	if err == nil {
		t.Fatal("GenerateConversions expected error due to missing module-path, got nil")
	}

	wantError := "module-path must be configured for generating conversions"
	if err.Error() != wantError {
		t.Errorf("GenerateConversions returned error %q, want %q", err.Error(), wantError)
	}
}

func TestGenerateConversions_Message(t *testing.T) {
	outDir := t.TempDir()

	field1 := api.NewTestField("name").WithType(api.TypezString)
	field2 := api.NewTestField("metageneration").WithType(api.TypezInt64)
	field3 := api.NewTestField("self").WithType(api.TypezString).WithOptional()
	folder := api.NewTestMessage("Folder").
		WithPackage("google.storage.control.v2").
		WithFields(field1, field2, field3)

	model := api.NewTestAPI([]*api.Message{folder}, nil, nil)

	library := &config.Library{}
	module := &config.SwiftModule{
		ModulePath: "StorageControlProtos",
	}

	if err := GenerateConversions(t.Context(), model, outDir, library, module); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(outDir, "Folder+Convert.swift"))
	if err != nil {
		t.Fatal(err)
	}
	gotContent := string(b)

	// Check output imports
	if !strings.Contains(gotContent, "internal import StorageControlProtos") {
		t.Errorf("expected generated file to import StorageControlProtos")
	}

	// Check conversion logic
	got := extractBlock(t, gotContent, "  internal init(proto: ProtoType) throws {", "\n  }")
	wantInit := `  internal init(proto: ProtoType) throws {
    self.init()
    self.name = proto.name
    self.metageneration = proto.metageneration
    self.self_ = proto.hasSelf_p ? proto.self_p : nil
    self._unknownFields.proto = proto.unknownFields.data
  }`
	if diff := cmp.Diff(wantInit, got); diff != "" {
		t.Errorf("init(proto:) mismatch (-want +got):\n%s", diff)
	}

	got = extractBlock(t, gotContent, "  internal func toProto() throws -> ProtoType {", "\n  }")
	wantToProto := `  internal func toProto() throws -> ProtoType {
    var proto = ProtoType()
    proto.name = self.name
    proto.metageneration = self.metageneration
    if let self_ = self.self_ { proto.self_p = self_ }
    if !self._unknownFields.proto.isEmpty {
      try proto.merge(serializedBytes: self._unknownFields.proto)
    }
    return proto
  }`
	if diff := cmp.Diff(wantToProto, got); diff != "" {
		t.Errorf("toProto() mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateConversions_RecursiveMessage(t *testing.T) {
	outDir := t.TempDir()

	node := api.NewTestMessage("Node").WithPackage("test")

	field1 := api.NewTestField("child_node").
		WithMessageType(node).
		WithDocumentation("Non-optional recursive child.").
		WithRecursive()

	field2 := api.NewTestField("next_node").
		WithMessageType(node).
		WithOptional().
		WithDocumentation("Optional recursive child.").
		WithRecursive()

	node.WithFields(field1, field2)

	model := api.NewTestAPI([]*api.Message{node}, nil, nil)

	library := &config.Library{}
	module := &config.SwiftModule{
		ModulePath: "TestProtos",
	}

	if err := GenerateConversions(t.Context(), model, outDir, library, module); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(outDir, "Node+Convert.swift"))
	if err != nil {
		t.Fatal(err)
	}
	gotContent := string(b)

	got := extractBlock(t, gotContent, "  internal init(proto: ProtoType) throws {", "\n  }")
	wantInit := `  internal init(proto: ProtoType) throws {
    self.init()
    self.childNode = proto.hasChildNode ? GoogleWKT.Recursive(value: try .init(proto: proto.childNode)) : nil
    self.nextNode = proto.hasNextNode ? GoogleWKT.Recursive(value: try .init(proto: proto.nextNode)) : nil
    self._unknownFields.proto = proto.unknownFields.data
  }`
	if diff := cmp.Diff(wantInit, got); diff != "" {
		t.Errorf("init(proto:) mismatch (-want +got):\n%s", diff)
	}

	got = extractBlock(t, gotContent, "  internal func toProto() throws -> ProtoType {", "\n  }")
	wantToProto := `  internal func toProto() throws -> ProtoType {
    var proto = ProtoType()
    if let childNode = self.childNode { proto.childNode = try childNode.value.toProto() }
    if let nextNode = self.nextNode { proto.nextNode = try nextNode.value.toProto() }
    if !self._unknownFields.proto.isEmpty {
      try proto.merge(serializedBytes: self._unknownFields.proto)
    }
    return proto
  }`
	if diff := cmp.Diff(wantToProto, got); diff != "" {
		t.Errorf("toProto() mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateConversions_NoConvertedFields(t *testing.T) {
	outDir := t.TempDir()

	msg := api.NewTestMessage("EmptyMessage").WithPackage("test")
	model := api.NewTestAPI([]*api.Message{msg}, nil, nil)

	library := &config.Library{
		Name: "GoogleCloudStorage",
	}
	module := &config.SwiftModule{
		ModulePath: "StorageControlProtos",
	}

	if err := GenerateConversions(t.Context(), model, outDir, library, module); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(outDir, "EmptyMessage+Convert.swift"))
	if err != nil {
		t.Fatal(err)
	}
	gotContent := string(b)

	got := extractBlock(t, gotContent, "  internal func toProto() throws -> ProtoType {", "\n  }")
	wantToProto := `  internal func toProto() throws -> ProtoType {
    var proto = ProtoType()
    if !self._unknownFields.proto.isEmpty {
      try proto.merge(serializedBytes: self._unknownFields.proto)
    }
    return proto
  }`
	if diff := cmp.Diff(wantToProto, got); diff != "" {
		t.Errorf("toProto() mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateConversions_RepeatedFields(t *testing.T) {
	outDir := t.TempDir()

	enumVal := api.NewTestEnumValue("UNSPECIFIED", 0)
	enum := api.NewTestEnum("Category").
		WithPackage("test").
		WithValues(enumVal)

	item := api.NewTestMessage("Item")

	field1 := api.NewTestField("names").
		WithType(api.TypezString).
		WithRepeated()
	field2 := api.NewTestField("items").
		WithMessageType(item).
		WithRepeated()
	field3 := api.NewTestField("categories").
		WithType(api.TypezEnum).
		WithTypezID(enum.ID).
		WithRepeated()

	container := api.NewTestMessage("Container").
		WithFields(field1, field2, field3)

	model := api.NewTestAPI([]*api.Message{item, container}, []*api.Enum{enum}, nil)

	library := &config.Library{}
	module := &config.SwiftModule{
		ModulePath: "TestProtos",
	}

	if err := GenerateConversions(t.Context(), model, outDir, library, module); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(outDir, "Container+Convert.swift"))
	if err != nil {
		t.Fatal(err)
	}
	gotContent := string(b)

	got := extractBlock(t, gotContent, "  internal init(proto: ProtoType) throws {", "\n  }")
	wantInit := `  internal init(proto: ProtoType) throws {
    self.init()
    self.names = proto.names
    self.items = try proto.items.map { try .init(proto: $0) }
    self.categories = proto.categories.map { .init(proto: $0) }
    self._unknownFields.proto = proto.unknownFields.data
  }`
	if diff := cmp.Diff(wantInit, got); diff != "" {
		t.Errorf("init(proto:) mismatch (-want +got):\n%s", diff)
	}

	got = extractBlock(t, gotContent, "  internal func toProto() throws -> ProtoType {", "\n  }")
	wantToProto := `  internal func toProto() throws -> ProtoType {
    var proto = ProtoType()
    proto.names = self.names
    proto.items = try self.items.map { try $0.toProto() }
    proto.categories = try self.categories.map { try $0.toProto() }
    if !self._unknownFields.proto.isEmpty {
      try proto.merge(serializedBytes: self._unknownFields.proto)
    }
    return proto
  }`
	if diff := cmp.Diff(wantToProto, got); diff != "" {
		t.Errorf("toProto() mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateConversions_OneOf(t *testing.T) {
	outDir := t.TempDir()

	inner := api.NewTestMessage("Inner").WithPackage("test")

	field1 := api.NewTestField("string_field").
		WithType(api.TypezString)
	field2 := api.NewTestField("message_field").
		WithMessageType(inner)

	oneof := api.NewTestOneOf("choice").
		WithFields(field1, field2)

	outer := api.NewTestMessage("Outer").
		WithOneOfs(oneof)

	model := api.NewTestAPI([]*api.Message{inner, outer}, nil, nil)

	library := &config.Library{}
	module := &config.SwiftModule{
		ModulePath: "TestProtos",
	}

	if err := GenerateConversions(t.Context(), model, outDir, library, module); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(outDir, "Outer+Convert.swift"))
	if err != nil {
		t.Fatal(err)
	}
	gotContent := string(b)

	got := extractBlock(t, gotContent, "  internal init(proto: ProtoType) throws {", "\n  }")
	wantInit := `  internal init(proto: ProtoType) throws {
    self.init()
    if let oneof = proto.choice {
      switch oneof {
      case .stringField(let value):
        self.choice = .stringField(value)
      case .messageField(let value):
        self.choice = .messageField(try .init(proto: value))
      }
    }
    self._unknownFields.proto = proto.unknownFields.data
  }`
	if diff := cmp.Diff(wantInit, got); diff != "" {
		t.Errorf("init(proto:) mismatch (-want +got):\n%s", diff)
	}

	got = extractBlock(t, gotContent, "  internal func toProto() throws -> ProtoType {", "\n  }")
	wantToProto := `  internal func toProto() throws -> ProtoType {
    var proto = ProtoType()
    if let oneof = self.choice {
      switch oneof {
      case .stringField(let value):
        proto.choice = .stringField(value)
      case .messageField(let value):
        if let value = value {
          proto.choice = .messageField(try value.toProto())
        }
      }
    }
    if !self._unknownFields.proto.isEmpty {
      try proto.merge(serializedBytes: self._unknownFields.proto)
    }
    return proto
  }`
	if diff := cmp.Diff(wantToProto, got); diff != "" {
		t.Errorf("toProto() mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateConversions_MapFields(t *testing.T) {
	outDir := t.TempDir()

	stringMapEntry := api.NewTestMessage("LabelsEntry").
		WithFields(
			api.NewTestField("key").WithType(api.TypezString),
			api.NewTestField("value").WithType(api.TypezString),
		).
		WithIsMap()

	objectMessage := api.NewTestMessage("RapidCachePolicy")

	objectMapEntry := api.NewTestMessage("PoliciesEntry").
		WithFields(
			api.NewTestField("key").WithType(api.TypezString),
			api.NewTestField("value").WithMessageType(objectMessage),
		).
		WithIsMap()

	enumVal := api.NewTestEnumValue("UNSPECIFIED", 0)
	enumType := api.NewTestEnum("FindingCategory").
		WithPackage("test").
		WithValues(enumVal)

	enumMapEntry := api.NewTestMessage("CategoriesEntry").
		WithFields(
			api.NewTestField("key").WithType(api.TypezString),
			api.NewTestField("value").WithType(api.TypezEnum).WithTypezID(enumType.ID),
		).
		WithIsMap()

	fieldPrimitive := api.NewTestField("labels").
		WithMessageType(stringMapEntry).
		WithMap()

	fieldObject := api.NewTestField("policies").
		WithMessageType(objectMapEntry).
		WithMap()

	fieldEnum := api.NewTestField("categories").
		WithMessageType(enumMapEntry).
		WithMap()

	msg := api.NewTestMessage("ObjectIndex").
		WithFields(fieldPrimitive, fieldObject, fieldEnum)

	model := api.NewTestAPI([]*api.Message{msg, stringMapEntry, objectMessage, objectMapEntry, enumMapEntry}, []*api.Enum{enumType}, nil)

	library := &config.Library{Name: "GoogleCloudStorage"}
	module := &config.SwiftModule{ModulePath: "StorageControlProtos"}

	if err := GenerateConversions(t.Context(), model, outDir, library, module); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(outDir, "ObjectIndex+Convert.swift"))
	if err != nil {
		t.Fatal(err)
	}
	gotContent := string(b)

	got := extractBlock(t, gotContent, "  internal init(proto: ProtoType) throws {", "\n  }")
	wantInit := `  internal init(proto: ProtoType) throws {
    self.init()
    self.labels = proto.labels
    self.policies = try proto.policies.mapValues { try .init(proto: $0) }
    self.categories = proto.categories.mapValues { .init(proto: $0) }
    self._unknownFields.proto = proto.unknownFields.data
  }`
	if diff := cmp.Diff(wantInit, got); diff != "" {
		t.Errorf("init(proto:) mismatch (-want +got):\n%s", diff)
	}

	got = extractBlock(t, gotContent, "  internal func toProto() throws -> ProtoType {", "\n  }")
	wantToProto := `  internal func toProto() throws -> ProtoType {
    var proto = ProtoType()
    proto.labels = self.labels
    proto.policies = try self.policies.mapValues { try $0.toProto() }
    proto.categories = try self.categories.mapValues { try $0.toProto() }
    if !self._unknownFields.proto.isEmpty {
      try proto.merge(serializedBytes: self._unknownFields.proto)
    }
    return proto
  }`
	if diff := cmp.Diff(wantToProto, got); diff != "" {
		t.Errorf("toProto() mismatch (-want +got):\n%s", diff)
	}
}

// TestGenerateConversions_Diagnose covers the conversion members, which name
// every field and every enum value.
func TestGenerateConversions_Diagnose(t *testing.T) {
	for _, test := range []struct {
		name       string
		deprecated bool
	}{
		{name: "deprecated", deprecated: true},
		{name: "not-deprecated", deprecated: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			status := api.NewTestEnum("Status").
				WithPackage("test").
				WithValues(
					api.NewTestEnumValue("STATUS_UNSPECIFIED", 0),
					api.NewTestEnumValue("STATUS_OK", 1).WithDeprecated(test.deprecated),
				)
			folder := api.NewTestMessage("Folder").
				WithPackage("test").
				WithFields(api.NewTestField("name").
					WithType(api.TypezString).
					WithDeprecated(test.deprecated))

			model := api.NewTestAPI([]*api.Message{folder}, []*api.Enum{status}, nil)
			model.PackageName = "test"
			module := &config.SwiftModule{ModulePath: "TestProtos"}
			if err := GenerateConversions(t.Context(), model, outDir, &config.Library{}, module); err != nil {
				t.Fatal(err)
			}

			read := func(basename string) string {
				t.Helper()
				content, err := os.ReadFile(filepath.Join(outDir, basename))
				if err != nil {
					t.Fatal(err)
				}
				return string(content)
			}

			message := read("Folder+Convert.swift")
			checkDiagnose(t, message, "  ", "internal init(proto: ProtoType) throws {", test.deprecated)
			checkDiagnose(t, message, "  ", "internal func toProto() throws -> ProtoType {", test.deprecated)

			// `toProto()` only pattern matches, which does not warn.
			checkDiagnose(t, read("Status+Convert.swift"), "  ",
				"internal init(proto: TestProtos.Test_Status) {", test.deprecated)
		})
	}
}
