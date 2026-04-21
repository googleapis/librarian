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
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func swiftConfig(t *testing.T, extraDependencies []config.SwiftDependency) *config.SwiftPackage {
	t.Helper()
	deps := []config.SwiftDependency{
		{Name: "GoogleCloudWkt", ApiPackage: wellKnownPackage},
	}
	deps = append(deps, extraDependencies...)
	return &config.SwiftPackage{
		SwiftDefault: config.SwiftDefault{
			Dependencies: deps,
		},
	}
}

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

	if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
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

	if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filename := filepath.Join(expectedDir, "WithNested.swift")
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

	if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filename := filepath.Join(expectedDir, "WithNestedEnum.swift")
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

func TestGenerateMessage_WithExternalImports(t *testing.T) {
	outDir := t.TempDir()

	externalMessage := &api.Message{
		Name:    "ExternalMessage",
		Package: "google.cloud.external.v1",
		ID:      ".google.cloud.external.v1.ExternalMessage",
	}

	message := &api.Message{
		Name:    "LocalMessage",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.LocalMessage",
		Fields: []*api.Field{
			{
				Name:    "ext_field",
				Typez:   api.TypezMessage,
				TypezID: ".google.cloud.external.v1.ExternalMessage",
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"
	model.State = &api.APIState{
		MessageByID: map[string]*api.Message{
			".google.cloud.external.v1.ExternalMessage": externalMessage,
		},
	}

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	swiftCfg := swiftConfig(t, []config.SwiftDependency{
		{
			ApiPackage: "google.cloud.external.v1",
			Name:       "GoogleCloudExternalV1",
		},
		{
			ApiPackage: "google.cloud.unused.v1",
			Name:       "GoogleCloudUnusedV1",
		},
	})

	if err := Generate(t.Context(), model, outDir, cfg, swiftCfg); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filename := filepath.Join(expectedDir, "LocalMessage.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	if !strings.Contains(contentStr, "import GoogleCloudExternalV1") {
		t.Errorf("expected 'import GoogleCloudExternalV1' in %s", filename)
	}
	if strings.Contains(contentStr, "import GoogleCloudUnusedV1") {
		t.Errorf("unexpected 'import GoogleCloudUnusedV1' in %s", filename)
	}
}
