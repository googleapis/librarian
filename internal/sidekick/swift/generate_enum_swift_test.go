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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGenerateEnum_Files(t *testing.T) {
	outDir := t.TempDir()

	color := api.NewTestEnum("Color").
		WithPackage("google.cloud.test.v1").
		WithValues(api.NewTestEnumValue("COLOR_UNSPECIFIED", 0))

	kind := api.NewTestEnum("Kind").
		WithPackage("google.cloud.test.v1").
		WithValues(api.NewTestEnumValue("KIND_UNSPECIFIED", 0))

	clash0 := api.NewTestEnum("ClashName").
		WithPackage("google.cloud.test.v1").
		WithValues(api.NewTestEnumValue("CLASH_UNSPECIFIED", 0))

	clash1 := api.NewTestEnum("clashName").
		WithPackage("google.cloud.test.v1").
		WithValues(api.NewTestEnumValue("CLASH_UNSPECIFIED", 0))

	model := api.NewTestAPI(nil, []*api.Enum{color, kind, clash0, clash1}, nil)
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

	val0 := api.NewTestEnumValue("KIND_UNSPECIFIED", 0)
	val1 := api.NewTestEnumValue("KIND_TEST", 0)
	val2 := api.NewTestEnumValue("KIND_OTHER_TEST", 1)

	kind := api.NewTestEnum("Kind").
		WithPackage("google.cloud.test.v1").
		WithValues(val0, val1, val2).
		WithUniqueNumberValues(val1, val2)

	model := api.NewTestAPI(nil, []*api.Enum{kind}, nil)
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

	val := api.NewTestEnumValue("COLOR_UNSPECIFIED", 0).
		WithDocumentation("Documentation for the COLOR_UNSPECIFIED value.")
	color := api.NewTestEnum("Color").
		WithPackage("google.cloud.test.v1").
		WithDocumentation("Documentation for the Color enum.").
		WithValues(val)

	model := api.NewTestAPI(nil, []*api.Enum{color}, nil)
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
public enum Color: Codable, Equatable, Hashable, Sendable {`
	got := extractBlock(t, contentStr, "/// Documentation for the Color enum.", "public enum Color: Codable, Equatable, Hashable, Sendable {")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	want = "/// Documentation for the COLOR_UNSPECIFIED value.\n  case unspecified"
	got = extractBlock(t, contentStr, "/// Documentation for the COLOR_UNSPECIFIED value.", "case unspecified")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateEnum_DefaultDocComments(t *testing.T) {
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

	want := `import Foundation

/// - Note: Adding cases to this enumeration is not considered a breaking change.
///   Always include an ` + "`@unknown default:`" + ` case when switching over this type.
///   Do not pattern-match against ` + "`unknownStringValue`" + ` or ` + "`unknownIntValue`" + `
///   expecting specific values to remain unparsed; future releases may promote
///   them to named cases.
public enum Kind`
	got := extractBlock(t, contentStr, "import Foundation", "public enum Kind")
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
	got := extractBlock(t, contentStr, "/// Encodes an unknown integer value.", "case unknownStringValue(String)")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateEnum_Discovery(t *testing.T) {
	outDir := t.TempDir()

	status := api.NewTestEnum("Status").
		WithPackage("google.cloud.compute.v1").
		WithValues(
			api.NewTestEnumValue("DONE", 0),
			api.NewTestEnumValue("PENDING", 1),
			api.NewTestEnumValue("RUNNING", 2),
		)

	model := api.NewTestAPI(nil, []*api.Enum{status}, nil)
	library := &config.Library{
		SpecificationFormat: config.SpecDiscovery,
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	contentB, err := os.ReadFile(filepath.Join(outDir, "Sources", "GoogleCloudComputeV1", "Status.swift"))
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(contentB)

	// Verify no integer or default init artifacts
	if strings.Contains(contentStr, "unknownIntValue") {
		t.Errorf("expected no unknownIntValue in discovery enum, got:\n%s", contentStr)
	}
	if strings.Contains(contentStr, "init(intValue:") {
		t.Errorf("expected no init(intValue:) in discovery enum, got:\n%s", contentStr)
	}
	if strings.Contains(contentStr, "var intValue:") {
		t.Errorf("expected no intValue in discovery enum, got:\n%s", contentStr)
	}
	if strings.Contains(contentStr, "public init() {") {
		t.Errorf("expected no public init() in discovery enum, got:\n%s", contentStr)
	}

	// Verify cases
	wantCases := `  case done
  case pending
  case running
  /// Encodes an unknown string value.
  ///
  /// The most common cause for an unknown value is for the service to send
  /// a value unknown to the library. We recommend you update your library to
  /// the latest version.
  ///
  /// - Warning: Do not pattern-match specific string literals in this case;
  ///   future releases may promote them to named enum cases.
  case unknownStringValue(Swift.String)`
	gotCases := extractBlock(t, contentStr, "  case done", "case unknownStringValue(Swift.String)")
	if diff := cmp.Diff(wantCases, gotCases); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify stringValue
	wantStringValue := `  public var stringValue: Swift.String? {
    switch self {
    case .done: return "DONE"
    case .pending: return "PENDING"
    case .running: return "RUNNING"
    case .unknownStringValue(let v): return v
    }
  }`
	gotStringValue := extractBlock(t, contentStr, "  public var stringValue: Swift.String? {", "\n  }")
	if diff := cmp.Diff(wantStringValue, gotStringValue); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify init(stringValue:)
	wantInitString := `  public init(stringValue: Swift.String) {
    switch stringValue {
    case "DONE": self = .done
    case "PENDING": self = .pending
    case "RUNNING": self = .running
    default: self = .unknownStringValue(stringValue)
    }
  }`
	gotInitString := extractBlock(t, contentStr, "  public init(stringValue: Swift.String) {", "\n  }")
	if diff := cmp.Diff(wantInitString, gotInitString); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify decoder
	wantDecoder := `  public init(from decoder: Decoder) throws {
    let container = try decoder.singleValueContainer()
    let s = try container.decode(Swift.String.self)
    self.init(stringValue: s)
  }`
	gotDecoder := extractBlock(t, contentStr, "  public init(from decoder: Decoder) throws {", "\n  }")
	if diff := cmp.Diff(wantDecoder, gotDecoder); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify encoder
	wantEncoder := `  public func encode(to encoder: Encoder) throws {
    var container = encoder.singleValueContainer()
    switch self {
    case .done: return try container.encode("DONE")
    case .pending: return try container.encode("PENDING")
    case .running: return try container.encode("RUNNING")
    case .unknownStringValue(let v): return try container.encode(v)
    }
  }`
	gotEncoder := extractBlock(t, contentStr, "  public func encode(to encoder: Encoder) throws {", "\n  }")
	if diff := cmp.Diff(wantEncoder, gotEncoder); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateMessageEnum_Discovery(t *testing.T) {
	outDir := t.TempDir()

	status := api.NewTestEnum("Status").
		WithPackage("google.cloud.compute.v1").
		WithValues(
			api.NewTestEnumValue("DONE", 0),
			api.NewTestEnumValue("PENDING", 1),
		)

	msg := api.NewTestMessage("Operation").
		WithPackage("google.cloud.compute.v1").
		WithEnums(status)

	model := api.NewTestAPI([]*api.Message{msg}, nil, nil)
	library := &config.Library{
		SpecificationFormat: config.SpecDiscovery,
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	contentB, err := os.ReadFile(filepath.Join(outDir, "Sources", "GoogleCloudComputeV1", "Operation.swift"))
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(contentB)

	enumBlock := extractBlock(t, contentStr, "public enum Status: Codable, Equatable, Hashable, Sendable {", "public static var _anyTypeUrl:")

	// Verify nested enum in message does not contain integer conversions or default init
	if strings.Contains(enumBlock, "unknownIntValue") {
		t.Errorf("expected no unknownIntValue in nested discovery enum, got:\n%s", enumBlock)
	}
	if strings.Contains(enumBlock, "init(intValue:") {
		t.Errorf("expected no init(intValue:) in nested discovery enum, got:\n%s", enumBlock)
	}
	if strings.Contains(enumBlock, "var intValue:") {
		t.Errorf("expected no intValue in nested discovery enum, got:\n%s", enumBlock)
	}
	if strings.Contains(enumBlock, "public init() {") {
		t.Errorf("expected no public init() in nested discovery enum, got:\n%s", enumBlock)
	}
	if !strings.Contains(contentStr, "public enum Status: Codable, Equatable, Hashable, Sendable {") {
		t.Errorf("expected Status enum in Operation.swift, got:\n%s", contentStr)
	}
}
