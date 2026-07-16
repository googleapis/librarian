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
	folder := &api.Message{
		Name:    "Folder",
		Package: "google.storage.control.v2",
		ID:      ".google.storage.control.v2.Folder",
		Fields:  []*api.Field{field1, field2},
	}
	field1.Parent = folder
	field2.Parent = folder

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
	if !strings.Contains(contentStr, "internal import StorageControlProtos") {
		t.Errorf("expected generated file to import StorageControlProtos")
	}

	// Check conversion logic
	wantInit := "  internal init(proto: ProtoType) throws {\n    self.name = proto.name\n    self.metageneration = proto.metageneration\n  }"
	if !strings.Contains(contentStr, wantInit) {
		t.Errorf("expected generated file to contain init(proto:) implementation:\n%s\nGot:\n%s", wantInit, contentStr)
	}

	wantToProto := "  internal func toProto() throws -> ProtoType {\n    var proto = ProtoType()\n    proto.name = self.name\n    proto.metageneration = self.metageneration\n    return proto\n  }"
	if !strings.Contains(contentStr, wantToProto) {
		t.Errorf("expected generated file to contain toProto() implementation:\n%s\nGot:\n%s", wantToProto, contentStr)
	}
}
