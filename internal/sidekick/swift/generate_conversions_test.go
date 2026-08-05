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
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"

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

	field1 := &api.Field{
		Name:     "name",
		JSONName: "name",
		Typez:    api.TypezString,
	}
	field2 := &api.Field{
		Name:     "metageneration",
		JSONName: "metageneration",
		Typez:    api.TypezInt64,
	}
	field3 := &api.Field{
		Name:     "self",
		JSONName: "self",
		Typez:    api.TypezString,
		Optional: true,
	}
	folder := &api.Message{
		Name:    "Folder",
		Package: "google.storage.control.v2",
		ID:      ".google.storage.control.v2.Folder",
		Fields:  []*api.Field{field1, field2, field3},
	}
	field1.Parent = folder
	field2.Parent = folder
	field3.Parent = folder

	model := api.NewTestAPI([]*api.Message{folder}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "google.storage.control.v2"

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
    return proto
  }`
	if diff := cmp.Diff(wantToProto, got); diff != "" {
		t.Errorf("toProto() mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateConversions_RecursiveMessage(t *testing.T) {
	outDir := t.TempDir()

	field1 := &api.Field{
		Name:          "child_node",
		ID:            ".test.Node.child_node",
		Typez:         api.TypezMessage,
		TypezID:       ".test.Node",
		Documentation: "Non-optional recursive child.",
		Optional:      false,
		Recursive:     true,
	}
	field2 := &api.Field{
		Name:          "next_node",
		ID:            ".test.Node.next_node",
		Typez:         api.TypezMessage,
		TypezID:       ".test.Node",
		Documentation: "Optional recursive child.",
		Optional:      true,
		Recursive:     true,
	}
	node := &api.Message{
		Name:    "Node",
		Package: "test",
		ID:      ".test.Node",
		Fields:  []*api.Field{field1, field2},
	}
	field1.Parent = node
	field2.Parent = node

	model := api.NewTestAPI([]*api.Message{node}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "test"

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
    self.childNode = proto.hasChildNode ? GoogleCloudWkt.Recursive(value: try .init(proto: proto.childNode)) : nil
    self.nextNode = proto.hasNextNode ? GoogleCloudWkt.Recursive(value: try .init(proto: proto.nextNode)) : nil
  }`
	if diff := cmp.Diff(wantInit, got); diff != "" {
		t.Errorf("init(proto:) mismatch (-want +got):\n%s", diff)
	}

	got = extractBlock(t, gotContent, "  internal func toProto() throws -> ProtoType {", "\n  }")
	wantToProto := `  internal func toProto() throws -> ProtoType {
    var proto = ProtoType()
    if let childNode = self.childNode { proto.childNode = try childNode.value.toProto() }
    if let nextNode = self.nextNode { proto.nextNode = try nextNode.value.toProto() }
    return proto
  }`
	if diff := cmp.Diff(wantToProto, got); diff != "" {
		t.Errorf("toProto() mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateConversions_NoConvertedFields(t *testing.T) {
	outDir := t.TempDir()

	field1 := &api.Field{
		Name:  "labels",
		ID:    "1",
		Map:   true,
		Typez: api.TypezString,
	}
	msg := &api.Message{
		Name:    "EmptyOrMapOnly",
		ID:      ".test.EmptyOrMapOnly",
		Fields:  []*api.Field{field1},
		Package: "test",
	}
	field1.Parent = msg
	model := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "test"

	library := &config.Library{
		Name: "GoogleCloudStorage",
	}
	module := &config.SwiftModule{
		ModulePath: "StorageControlProtos",
	}

	if err := GenerateConversions(t.Context(), model, outDir, library, module); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(outDir, "EmptyOrMapOnly+Convert.swift"))
	if err != nil {
		t.Fatal(err)
	}
	gotContent := string(b)

	got := extractBlock(t, gotContent, "  internal func toProto() throws -> ProtoType {", "\n  }")
	wantToProto := `  internal func toProto() throws -> ProtoType {
    let proto = ProtoType()
    return proto
  }`
	if diff := cmp.Diff(wantToProto, got); diff != "" {
		t.Errorf("toProto() mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateConversions_RepeatedFields(t *testing.T) {
	outDir := t.TempDir()

	enumVal := &api.EnumValue{Name: "UNSPECIFIED", Number: 0}
	enum := &api.Enum{
		Name:               "Category",
		ID:                 ".test.Category",
		Package:            "test",
		Values:             []*api.EnumValue{enumVal},
		UniqueNumberValues: []*api.EnumValue{enumVal},
	}
	enumVal.Parent = enum

	item := &api.Message{
		Name:    "Item",
		Package: "test",
		ID:      ".test.Item",
	}

	field1 := &api.Field{
		Name:     "names",
		ID:       ".test.Container.names",
		Typez:    api.TypezString,
		Repeated: true,
	}
	field2 := &api.Field{
		Name:     "items",
		ID:       ".test.Container.items",
		Typez:    api.TypezMessage,
		TypezID:  ".test.Item",
		Repeated: true,
	}
	field3 := &api.Field{
		Name:     "categories",
		ID:       ".test.Container.categories",
		Typez:    api.TypezEnum,
		TypezID:  ".test.Category",
		Repeated: true,
	}
	container := &api.Message{
		Name:    "Container",
		Package: "test",
		ID:      ".test.Container",
		Fields:  []*api.Field{field1, field2, field3},
	}
	field1.Parent = container
	field2.Parent = container
	field3.Parent = container

	model := api.NewTestAPI([]*api.Message{item, container}, []*api.Enum{enum}, []*api.Service{})
	model.PackageName = "test"

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
    return proto
  }`
	if diff := cmp.Diff(wantToProto, got); diff != "" {
		t.Errorf("toProto() mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateConversions_OneOf(t *testing.T) {
	outDir := t.TempDir()

	inner := &api.Message{
		Name:    "Inner",
		Package: "test",
		ID:      ".test.Inner",
	}

	oneof := &api.OneOf{
		Name: "choice",
	}

	field1 := &api.Field{
		Name:    "string_field",
		ID:      ".test.Outer.string_field",
		Typez:   api.TypezString,
		IsOneOf: true,
		Group:   oneof,
	}
	field2 := &api.Field{
		Name:    "message_field",
		ID:      ".test.Outer.message_field",
		Typez:   api.TypezMessage,
		TypezID: ".test.Inner",
		IsOneOf: true,
		Group:   oneof,
	}

	outer := &api.Message{
		Name:    "Outer",
		Package: "test",
		ID:      ".test.Outer",
		Fields:  []*api.Field{field1, field2},
		OneOfs:  []*api.OneOf{oneof},
	}
	oneof.Fields = []*api.Field{field1, field2}
	field1.Parent = outer
	field2.Parent = outer

	model := api.NewTestAPI([]*api.Message{inner, outer}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "test"

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
    return proto
  }`
	if diff := cmp.Diff(wantToProto, got); diff != "" {
		t.Errorf("toProto() mismatch (-want +got):\n%s", diff)
	}
}
