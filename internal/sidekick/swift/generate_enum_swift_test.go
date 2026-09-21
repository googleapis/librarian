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
	"github.com/googleapis/librarian/internal/config"

	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
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
	library := &config.Library{}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
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
	library := &config.Library{}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	contentsB, err := os.ReadFile(filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "Kind.swift"))
	if err != nil {
		t.Fatal(err)
	}
	got := extractBlock(t, string(contentsB), "/// Initialize from an integer value.", "\n  }")
	want := `/// Initialize from an integer value.
  ///
  /// If the value is unknown, this initializes to ` + "[`unknownIntValue`](doc:Kind/unknownIntValue(_:))." + `
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

	gotEncode := extractBlock(t, string(contentsB), "public func encode(to encoder: Encoder) throws {", "\n  }")
	wantEncode := `public func encode(to encoder: Encoder) throws {
    var container = encoder.singleValueContainer()
    switch self {
    case .test: return try container.encode("KIND_TEST")
    case .otherTest: return try container.encode("KIND_OTHER_TEST")
    case .unknownIntValue(let v): return try container.encode(v)
    case .unknownStringValue(let v): return try container.encode(v)
    }
  }`
	if diff := cmp.Diff(wantEncode, gotEncode); diff != "" {
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
	library := &config.Library{}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "Color.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	want := `/// Documentation for the Color enum.
///
/// - Note: Adding cases to this enumeration is not considered a breaking change.
///   Always include an ` + "`@unknown default:`" + ` case when switching over this type.
///   Do not pattern-match against ` + "`unknownStringValue`" + ` or ` + "`unknownIntValue`" + `
///   expecting specific values to remain unparsed; future releases may promote
///   them to named cases.
public enum Color`
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

func TestGenerateEnum_UnknownCaseDocComments(t *testing.T) {
	outDir := t.TempDir()

	kind := api.NewTestEnum("Kind").
		WithPackage("google.cloud.test.v1").
		WithValues(api.NewTestEnumValue("KIND_UNSPECIFIED", 0))

	model := api.NewTestAPI(nil, []*api.Enum{kind}, nil)
	if err := Generate(t.Context(), model, outDir, &config.Library{}, nil); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "Kind.swift"))
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	wantEnum := `import Foundation

/// - Note: Adding cases to this enumeration is not considered a breaking change.
///   Always include an ` + "`@unknown default:`" + ` case when switching over this type.
///   Do not pattern-match against ` + "`unknownStringValue`" + ` or ` + "`unknownIntValue`" + `
///   expecting specific values to remain unparsed; future releases may promote
///   them to named cases.
public enum Kind`
	gotEnum := extractBlock(t, contentStr, "import Foundation", "public enum Kind")
	if diff := cmp.Diff(wantEnum, gotEnum); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	got := extractBlock(t, contentStr, "/// Encodes an unknown integer value.", "case unknownStringValue(String)")
	want := `/// Encodes an unknown integer value.
  ///
  /// The most common cause for an unknown value is for the service to send
  /// a value unknown to the library. We recommend you update your library to
  /// the latest version.
  ///
  /// - Warning: Do not pattern-match specific integer values in this case;
  ///   future releases may promote them to named enum cases.
  case unknownIntValue(Int)
  /// Encodes an unknown string value.
  ///
  /// The most common cause for an unknown value is for the service to send
  /// a value unknown to the library. We recommend you update your library to
  /// the latest version.
  ///
  /// - Warning: Do not pattern-match specific string literals in this case;
  ///   future releases may promote them to named enum cases.
  case unknownStringValue(String)`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
