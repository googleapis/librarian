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
	wantInit := "  internal init(proto: ProtoType) throws {\n    self.init()\n    self.name = proto.name\n    self.metageneration = proto.metageneration\n    self.self_ = proto.hasSelf_p ? proto.self_p : nil\n  }"
	if diff := cmp.Diff(wantInit, got); diff != "" {
		t.Errorf("init(proto:) mismatch (-want +got):\n%s", diff)
	}

	got = extractBlock(t, gotContent, "  internal func toProto() throws -> ProtoType {", "\n  }")
	wantToProto := "  internal func toProto() throws -> ProtoType {\n    var proto = ProtoType()\n    proto.name = self.name\n    proto.metageneration = self.metageneration\n    if let self_ = self.self_ { proto.self_p = self_ }\n    return proto\n  }"
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
	wantInit := "  internal init(proto: ProtoType) throws {\n    self.init()\n    self.childNode = proto.hasChildNode ? GoogleCloudWkt.Recursive(value: try Node(proto: proto.childNode)) : nil\n    self.nextNode = proto.hasNextNode ? GoogleCloudWkt.Recursive(value: try Node(proto: proto.nextNode)) : nil\n  }"
	if diff := cmp.Diff(wantInit, got); diff != "" {
		t.Errorf("init(proto:) mismatch (-want +got):\n%s", diff)
	}

	got = extractBlock(t, gotContent, "  internal func toProto() throws -> ProtoType {", "\n  }")
	wantToProto := "  internal func toProto() throws -> ProtoType {\n    var proto = ProtoType()\n    if let childNode = self.childNode { proto.childNode = try childNode.value.toProto() }\n    if let nextNode = self.nextNode { proto.nextNode = try nextNode.value.toProto() }\n    return proto\n  }"
	if diff := cmp.Diff(wantToProto, got); diff != "" {
		t.Errorf("toProto() mismatch (-want +got):\n%s", diff)
	}
}
