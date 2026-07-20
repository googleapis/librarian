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

func TestGenerateEnum_Files(t *testing.T) {
	outDir := t.TempDir()

	color := &api.Enum{Name: "Color", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.Color"}
	color.Values = []*api.EnumValue{{Name: "COLOR_UNSPECIFIED", Number: 0, Parent: color}}
	color.UniqueNumberValues = color.Values

	kind := &api.Enum{Name: "Kind", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.Kind"}
	kind.Values = []*api.EnumValue{{Name: "KIND_UNSPECIFIED", Number: 0, Parent: kind}}
	kind.UniqueNumberValues = kind.Values

	clash0 := &api.Enum{Name: "ClashName", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.ClashName"}
	clash0.Values = []*api.EnumValue{{Name: "CLASH_UNSPECIFIED", Number: 0, Parent: clash0}}
	clash0.UniqueNumberValues = clash0.Values
	clash1 := &api.Enum{Name: "clashName", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.clashName"}
	clash1.Values = []*api.EnumValue{{Name: "CLASH_UNSPECIFIED", Number: 0, Parent: clash1}}
	clash1.UniqueNumberValues = clash1.Values

	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{color, kind, clash0, clash1}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"
	cfg := &parser.ModelConfig{}
	if err := Generate(t.Context(), model, outDir, cfg, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	want := []string{
		"Color.swift",
		"Kind.swift",
		"ClashName.swift",
		"clashName+000.swift",
	}
	for _, expected := range want {
		filename := filepath.Join(expectedDir, expected)
		if _, err := os.Stat(filename); err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateEnum_UniqueNumbers(t *testing.T) {
	outDir := t.TempDir()

	kind := &api.Enum{Name: "Kind", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.Kind"}
	kind.Values = []*api.EnumValue{
		{Name: "KIND_UNSPECIFIED", Number: 0, Parent: kind},
		{Name: "KIND_TEST", Number: 0, Parent: kind},
		{Name: "KIND_OTHER_TEST", Number: 1, Parent: kind},
	}
	kind.UniqueNumberValues = []*api.EnumValue{kind.Values[1], kind.Values[2]}

	model := api.NewTestAPI(nil, []*api.Enum{kind}, nil)
	model.PackageName = "google.cloud.test.v1"
	cfg := &parser.ModelConfig{}
	if err := Generate(t.Context(), model, outDir, cfg, nil); err != nil {
		t.Fatal(err)
	}

	contentsB, err := os.ReadFile(filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "Kind.swift"))
	if err != nil {
		t.Fatal(err)
	}
	got := extractBlock(t, string(contentsB), "/// Initialize from an integer value.", "\n  }")
	want := `/// Initialize from an integer value.
  ///
  /// If the value is unknown, this initializes to ` + "``.unknownIntValue(_:)``." + `
  public init(intValue: Int) {
    switch intValue {
    case 0: self = .test
    case 1: self = .otherTest
    default: self = .unknownIntValue(intValue)
    }
  }`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateEnum_DocComments(t *testing.T) {
	outDir := t.TempDir()

	color := &api.Enum{
		Name:          "Color",
		Package:       "google.cloud.test.v1",
		ID:            ".google.cloud.test.v1.Color",
		Documentation: "Documentation for the Color enum.",
	}
	color.Values = []*api.EnumValue{
		{
			Name:          "COLOR_UNSPECIFIED",
			Number:        0,
			Parent:        color,
			Documentation: "Documentation for the COLOR_UNSPECIFIED value.",
		},
	}
	color.UniqueNumberValues = color.Values

	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{color}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"
	cfg := &parser.ModelConfig{}
	if err := Generate(t.Context(), model, outDir, cfg, nil); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "Color.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	want := "/// Documentation for the Color enum.\npublic enum Color"
	got := extractBlock(t, contentStr, "/// Documentation for the Color enum.", "public enum Color")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	want = "/// Documentation for the COLOR_UNSPECIFIED value.\n  case unspecified"
	got = extractBlock(t, contentStr, "/// Documentation for the COLOR_UNSPECIFIED value.", "case unspecified")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
