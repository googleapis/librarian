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

func TestGenerateMessage_Files(t *testing.T) {
	outDir := t.TempDir()

	secret := &api.Message{Name: "Secret", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.Secret"}
	volume := &api.Message{Name: "Volume", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.Volume"}

	model := api.NewTestAPI([]*api.Message{secret, volume}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	for _, expected := range []string{"Secret.swift", "Volume.swift"} {
		filename := filepath.Join(expectedDir, expected)
		if _, err := os.Stat(filename); err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateMessage_WithNestedMessages(t *testing.T) {
	outDir := t.TempDir()

	nested1 := &api.Message{Name: "Nested1", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.WithNested.Nested1"}
	nested2 := &api.Message{Name: "Nested2", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.WithNested.Nested2"}
	withNested := &api.Message{
		Name:     "WithNested",
		Package:  "google.cloud.test.v1",
		ID:       ".google.cloud.test.v1.WithNested",
		Messages: []*api.Message{nested1, nested2},
	}

	model := api.NewTestAPI([]*api.Message{withNested}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filename := filepath.Join(expectedDir, "WithNested.swift")
	if _, err := os.Stat(filename); err != nil {
		t.Error(err)
	}
	for _, unexpected := range []string{"Nested1.swift", "Nested2.swift"} {
		unexpectedFilename := filepath.Join(expectedDir, unexpected)
		if _, err := os.Stat(unexpectedFilename); err == nil {
			t.Errorf("unexpected file generated: %s", unexpectedFilename)
		}
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	gotBlock1 := extractBlock(t, contentStr, "public struct Nested1", "Equatable {")
	wantBlock1 := "public struct Nested1: Codable, Equatable {"
	if diff := cmp.Diff(wantBlock1, gotBlock1); diff != "" {
		t.Errorf("mismatch in Nested1 (-want +got):\n%s", diff)
	}

	gotBlock2 := extractBlock(t, contentStr, "public struct Nested2", "Equatable {")
	wantBlock2 := "public struct Nested2: Codable, Equatable {"
	if diff := cmp.Diff(wantBlock2, gotBlock2); diff != "" {
		t.Errorf("mismatch in Nested2 (-want +got):\n%s", diff)
	}
}

func TestGenerateMessage_WithNestedEnum(t *testing.T) {
	outDir := t.TempDir()

	nestedEnum := &api.Enum{Name: "NestedEnum", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.WithNestedEnum.NestedEnum"}
	nestedEnum.Values = []*api.EnumValue{{Name: "NESTED_ENUM_UNSPECIFIED", Number: 0, Parent: nestedEnum}}
	nestedEnum.UniqueNumberValues = nestedEnum.Values

	withNested := &api.Message{
		Name:    "WithNestedEnum",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.WithNestedEnum",
		Enums:   []*api.Enum{nestedEnum},
	}

	model := api.NewTestAPI([]*api.Message{withNested}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filename := filepath.Join(expectedDir, "WithNestedEnum.swift")
	if _, err := os.Stat(filename); err != nil {
		t.Error(err)
	}
	unexpectedFilename := filepath.Join(expectedDir, "NestedEnum.swift")
	if _, err := os.Stat(unexpectedFilename); err == nil {
		t.Errorf("unexpected file generated: %s", unexpectedFilename)
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	gotBlock := extractBlock(t, contentStr, "public enum NestedEnum", "Equatable {")
	wantBlock := "public enum NestedEnum: Int, Codable, Equatable {"
	if diff := cmp.Diff(wantBlock, gotBlock); diff != "" {
		t.Errorf("mismatch in NestedEnum (-want +got):\n%s", diff)
	}
}
