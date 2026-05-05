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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestGenerateField_InitFromDecoder(t *testing.T) {
	outDir := t.TempDir()

	// Field 1: Normal field with JSONName override to trigger CustomSerialization
	field1 := &api.Field{
		Name:          "normal_field",
		Documentation: "A normal field.",
		ID:            ".test.TestMessage.normal_field",
		Typez:         api.TypezString,
		JSONName:      "normal_field", // Differs from camelCase "normalField"
	}

	// Field 2: Optional field with JSONName override
	field2 := &api.Field{
		Name:          "optional_field",
		Documentation: "An optional field.",
		ID:            ".test.TestMessage.optional_field",
		Typez:         api.TypezString,
		Optional:      true,
		JSONName:      "optional_field", // Differs from camelCase "optionalField"
	}

	msg := &api.Message{
		Name:    "TestMessage",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.TestMessage",
		Fields:  []*api.Field{field1, field2},
	}

	model := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filename := filepath.Join(expectedDir, "TestMessage.swift")

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// Verify init(from decoder: Decoder) content
	gotBlock := extractBlock(t, contentStr, "  public init(from decoder: Decoder) throws {", "\n  }")
	wantBlock := `  public init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    self.normalField = try container.decode(String.self, forKey: .normalField)
    self.optionalField = try container.decodeIfPresent(String.self, forKey: .optionalField)
  }`

	if diff := cmp.Diff(wantBlock, gotBlock); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
