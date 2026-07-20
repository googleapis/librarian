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
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestGenerateConversions_MissingModulePath(t *testing.T) {
	outDir := t.TempDir()
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{}

	err := GenerateConversions(t.Context(), model, outDir, cfg, nil)
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

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
			"module-path":    "StorageControlProtos",
			"module":         "true",
		},
	}

	if err := GenerateConversions(t.Context(), model, outDir, cfg, nil); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Convert", "Folder+Convert.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// Check output imports
	if !bytes.Contains(content, []byte("internal import StorageControlProtos")) {
		t.Errorf("expected generated file to import StorageControlProtos")
	}

	// Check conversion logic
	got := extractBlock(t, contentStr, "  internal init(proto: ProtoType) throws {", "\n  }")
	wantInit := "  internal init(proto: ProtoType) throws {\n    self.name = proto.name\n    self.metageneration = proto.metageneration\n    self.self_ = proto.hasSelf_p ? proto.self_p : nil\n  }"
	if diff := cmp.Diff(wantInit, got); diff != "" {
		t.Errorf("init(proto:) mismatch (-want +got):\n%s", diff)
	}

	got = extractBlock(t, contentStr, "  internal func toProto() throws -> ProtoType {", "\n  }")
	wantToProto := "  internal func toProto() throws -> ProtoType {\n    var proto = ProtoType()\n    proto.name = self.name\n    proto.metageneration = self.metageneration\n    if let self_ = self.self_ { proto.self_p = self_ }\n    return proto\n  }"
	if diff := cmp.Diff(wantToProto, got); diff != "" {
		t.Errorf("toProto() mismatch (-want +got):\n%s", diff)
	}
}
