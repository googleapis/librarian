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

func TestGenerateField_InitFromDecoder(t *testing.T) {
	outDir := t.TempDir()

	// Field 1: Normal field with JSONName override to trigger CustomSerialization
	field1 := api.NewTestField("normal_field").
		WithType(api.TypezString).
		WithJSONName("normal_field"). // Differs from camelCase "normalField"
		WithDocumentation("A normal field.")

	// Field 2: Optional field with JSONName override
	field2 := api.NewTestField("optional_field").
		WithType(api.TypezString).
		WithOptional().
		WithJSONName("optional_field"). // Differs from camelCase "optionalField"
		WithDocumentation("An optional field.")

	msg := api.NewTestMessage("TestMessage").
		WithPackage("google.cloud.test.v1").
		WithFields(field1, field2)

	model := api.NewTestAPI([]*api.Message{msg}, nil, nil)

	library := &config.Library{
		Swift: swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filename := filepath.Join(expectedDir, "TestMessage.swift")

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// Verify init(from decoder: any Decoder) content
	gotBlock := extractBlock(t, contentStr, "  public init(from decoder: any Decoder) throws {", "\n  }")
	wantBlock := `  public init(from decoder: any Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    if let value = try container.decodeIfPresent(Swift.String.self, forKey: .normalField) {
      self.normalField = value
    }
    self.optionalField = try container.decodeIfPresent(Swift.String.self, forKey: .optionalField)
    for key in container.allKeys where !CodingKeys._knownKeys.contains(key.stringValue) {
      self._unknownFields.json[key.stringValue] = try container.decode(
        GoogleWKT.WKTValue.self, forKey: key)
    }
  }`

	if diff := cmp.Diff(wantBlock, gotBlock); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateField_DocComments(t *testing.T) {
	outDir := t.TempDir()

	field1 := api.NewTestField("normal_field").
		WithType(api.TypezString).
		WithDocumentation("Documentation for normal_field.")

	field2 := api.NewTestField("oneof_field").
		WithType(api.TypezString).
		WithDocumentation("Documentation for oneof_field.")

	oneof := api.NewTestOneOf("my_oneof").
		WithFields(field2).
		WithDocumentation("Documentation for my_oneof.")

	msg := api.NewTestMessage("TestMessage").
		WithPackage("google.cloud.test.v1").
		WithFields(field1).
		WithOneOfs(oneof)

	model := api.NewTestAPI([]*api.Message{msg}, nil, nil)

	library := &config.Library{}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "TestMessage.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// Verify normal field documentation
	want := "  /// Documentation for normal_field.\n  public var normalField: Swift.String"
	got := extractBlock(t, contentStr, "  /// Documentation for normal_field.", "public var normalField: Swift.String")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify oneof documentation in the message
	want = "  /// Documentation for my_oneof.\n  public var myOneof: MyOneofOneOf?"
	got = extractBlock(t, contentStr, "  /// Documentation for my_oneof.", "public var myOneof: MyOneofOneOf?")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify field documentation in the oneof enum
	want = "    /// Documentation for oneof_field.\n    case oneofField(Swift.String)"
	got = extractBlock(t, contentStr, "    /// Documentation for oneof_field.", "case oneofField(Swift.String)")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateField_ExplicitNull(t *testing.T) {
	outDir := t.TempDir()

	mapValueType := api.NewTestMapWithFields(
		"$map<string, .google.protobuf.Value>",
		api.NewTestField("key").WithType(api.TypezString),
		api.NewTestField("value").WithType(api.TypezMessage).WithTypezID(api.WktValueID),
	)

	mapNullType := api.NewTestMapWithFields(
		"$map<string, .google.protobuf.NullValue>",
		api.NewTestField("key").WithType(api.TypezString),
		api.NewTestField("value").WithType(api.TypezEnum).WithTypezID(api.WktNullValueID),
	)

	msg := api.NewTestMessage("ExplicitNullMessage").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("singular_value").WithType(api.TypezMessage).WithTypezID(api.WktValueID),
			api.NewTestField("optional_value").WithType(api.TypezMessage).WithTypezID(api.WktValueID).WithOptional(),
			api.NewTestField("repeated_value").WithType(api.TypezMessage).WithTypezID(api.WktValueID).WithRepeated(),
			api.NewTestField("map_value").WithMapType(mapValueType),
			api.NewTestField("singular_null").WithType(api.TypezEnum).WithTypezID(api.WktNullValueID),
			api.NewTestField("optional_null").WithType(api.TypezEnum).WithTypezID(api.WktNullValueID).WithOptional(),
			api.NewTestField("repeated_null").WithType(api.TypezEnum).WithTypezID(api.WktNullValueID).WithRepeated(),
			api.NewTestField("map_null").WithMapType(mapNullType),
			api.NewTestField("optional_struct").WithType(api.TypezMessage).WithTypezID(api.WktStructID).WithOptional(),
		)

	model := api.NewTestAPI([]*api.Message{msg}, nil, nil).
		WithMaps(mapValueType, mapNullType)

	library := &config.Library{
		Swift: swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "ExplicitNullMessage.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	got := extractBlock(t, string(content), "  public init(from decoder: any Decoder) throws {", "\n  }")
	want := `  public init(from decoder: any Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    if container.contains(.singularValue) {
      self.singularValue = try container.decode(GoogleWKT.WKTValue.self, forKey: .singularValue)
    }
    if container.contains(.optionalValue) {
      self.optionalValue = try container.decode(GoogleWKT.WKTValue.self, forKey: .optionalValue)
    }
    if let value = try container.decodeIfPresent([GoogleWKT.WKTValue].self, forKey: .repeatedValue) {
      self.repeatedValue = value
    }
    if let value = try container.decodeIfPresent([Swift.String: GoogleWKT.WKTValue].self, forKey: .mapValue) {
      self.mapValue = value
    }
    if container.contains(.singularNull) {
      self.singularNull = try container.decode(GoogleWKT.WKTNullValue.self, forKey: .singularNull)
    }
    if container.contains(.optionalNull) {
      self.optionalNull = try container.decode(GoogleWKT.WKTNullValue.self, forKey: .optionalNull)
    }
    if let value = try container.decodeIfPresent([GoogleWKT.WKTNullValue].self, forKey: .repeatedNull) {
      self.repeatedNull = value
    }
    if let value = try container.decodeIfPresent([Swift.String: GoogleWKT.WKTNullValue].self, forKey: .mapNull) {
      self.mapNull = value
    }
    self.optionalStruct = try container.decodeIfPresent(GoogleWKT.WKTStruct.self, forKey: .optionalStruct)
    for key in container.allKeys where !CodingKeys._knownKeys.contains(key.stringValue) {
      self._unknownFields.json[key.stringValue] = try container.decode(
        GoogleWKT.WKTValue.self, forKey: key)
    }
  }`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
