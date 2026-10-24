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
	"github.com/googleapis/librarian/internal/config"
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

func TestGenerateField_Encode(t *testing.T) {
	outDir := t.TempDir()

	enum := api.NewTestEnum("Status").
		WithPackage("google.cloud.test.v1").
		WithValues(
			api.NewTestEnumValue("STATUS_UNSPECIFIED", 0),
			api.NewTestEnumValue("ACTIVE", 1),
		)

	strMapEntry := api.NewTestMapMessage("map<string, string>", api.TypezString, api.TypezString).
		WithID("$map<string, string>").
		WithPackage("google.cloud.test.v1")

	intMapEntry := api.NewTestMapMessage("map<int32, string>", api.TypezInt32, api.TypezString).
		WithID("$map<int32, string>").
		WithPackage("google.cloud.test.v1")

	bytesMapEntry := api.NewTestMapMessage("map<string, bytes>", api.TypezString, api.TypezBytes).
		WithID("$map<string, bytes>").
		WithPackage("google.cloud.test.v1")

	msg := api.NewTestMessage("TestMessage").
		WithPackage("google.cloud.test.v1")
	msg.WithFields(
		api.NewTestField("optional_field").WithType(api.TypezString).WithOptional(),
		api.NewTestField("recursive_field").WithMessageType(msg).WithRecursive(),
		api.NewTestField("string_field").WithType(api.TypezString),
		api.NewTestField("bytes_field").WithType(api.TypezBytes),
		api.NewTestField("int32_field").WithType(api.TypezInt32),
		api.NewTestField("uint64_field").WithType(api.TypezUint64),
		api.NewTestField("double_field").WithType(api.TypezDouble),
		api.NewTestField("bool_field").WithType(api.TypezBool),
		api.NewTestField("enum_field").WithEnumType(enum),
		api.NewTestField("repeated_field").WithType(api.TypezString).WithRepeated(),
		api.NewTestField("string_map").WithMessageType(strMapEntry).WithMap(),
		api.NewTestField("int_map").WithMessageType(intMapEntry).WithMap(),
	)

	model := api.NewTestAPI([]*api.Message{msg, strMapEntry, intMapEntry}, []*api.Enum{enum}, nil)
	library := &config.Library{
		Swift: swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "TestMessage.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	gotBlock := extractBlock(t, string(content), "  public func encode(to encoder: any Encoder) throws {", "\n  }")
	wantBlock := `  public func encode(to encoder: any Encoder) throws {
    var container = encoder.container(keyedBy: CodingKeys.self)
    try container.encodeIfPresent(self.optionalField, forKey: .optionalField)
    try container.encodeIfPresent(self.recursiveField, forKey: .recursiveField)
    if !self.stringField.isEmpty {
      try container.encode(self.stringField, forKey: .stringField)
    }
    if !self.bytesField.isEmpty {
      try container.encode(self.bytesField, forKey: .bytesField)
    }
    if self.int32Field != 0 {
      try container.encode(self.int32Field, forKey: .int32Field)
    }
    if self.uint64Field != 0 {
      try container.encode(self.uint64Field, forKey: .uint64Field)
    }
    if self.doubleField != 0 {
      try container.encode(self.doubleField, forKey: .doubleField)
    }
    if self.boolField {
      try container.encode(self.boolField, forKey: .boolField)
    }
    if self.enumField != Status() {
      try container.encode(self.enumField, forKey: .enumField)
    }
    if !self.repeatedField.isEmpty {
      try container.encode(self.repeatedField, forKey: .repeatedField)
    }
    if !self.stringMap.isEmpty {
      try container.encode(self.stringMap, forKey: .stringMap)
    }
    if !self.intMap.isEmpty {
      let stringKeyed = Dictionary(
        uniqueKeysWithValues: self.intMap.lazy.map { (Swift.String($0), $1) }
      )
      try container.encode(stringKeyed, forKey: .intMap)
    }
    for (key, value) in self._unknownFields.json {
      try container.encode(value, forKey: CodingKeys(stringValue: key))
    }
  }`

	if diff := cmp.Diff(wantBlock, gotBlock); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	urlSafeDir := t.TempDir()
	urlSafeMsg := api.NewTestMessage("UrlSafeMessage").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("optional_bytes").WithType(api.TypezBytes).WithOptional(),
			api.NewTestField("singular_bytes").WithType(api.TypezBytes),
			api.NewTestField("repeated_bytes").WithType(api.TypezBytes).WithRepeated(),
			api.NewTestField("bytes_map").WithMessageType(bytesMapEntry).WithMap(),
		)
	urlSafeModel := api.NewTestAPI([]*api.Message{urlSafeMsg, bytesMapEntry}, nil, nil)
	urlSafeLibrary := &config.Library{
		SpecificationFormat: config.SpecDiscovery,
		Swift:               swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), urlSafeModel, urlSafeDir, urlSafeLibrary, nil); err != nil {
		t.Fatal(err)
	}

	urlSafeFilename := filepath.Join(urlSafeDir, "Sources", "GoogleCloudTestV1", "UrlSafeMessage.swift")
	urlSafeContent, err := os.ReadFile(urlSafeFilename)
	if err != nil {
		t.Fatal(err)
	}
	gotUrlSafeBlock := extractBlock(t, string(urlSafeContent), "  public func encode(to encoder: any Encoder) throws {", "\n  }")
	wantUrlSafeBlock := `  public func encode(to encoder: any Encoder) throws {
    var container = encoder.container(keyedBy: CodingKeys.self)
    if let v = self.optionalBytes {
      try container.encode(
        GoogleWKT._DiscoveryBase64.encode(v), forKey: .optionalBytes
      )
    }
    if !self.singularBytes.isEmpty {
      try container.encode(
        GoogleWKT._DiscoveryBase64.encode(self.singularBytes), forKey: .singularBytes
      )
    }
    if !self.repeatedBytes.isEmpty {
      try container.encode(
        self.repeatedBytes.map { GoogleWKT._DiscoveryBase64.encode($0) }, forKey: .repeatedBytes
      )
    }
    if !self.bytesMap.isEmpty {
      let transformed = self.bytesMap.mapValues {
        GoogleWKT._DiscoveryBase64.encode($0)
      }
      try container.encode(transformed, forKey: .bytesMap)
    }
    for (key, value) in self._unknownFields.json {
      try container.encode(value, forKey: CodingKeys(stringValue: key))
    }
  }`

	if diff := cmp.Diff(wantUrlSafeBlock, gotUrlSafeBlock); diff != "" {
		t.Errorf("url-safe mismatch (-want +got):\n%s", diff)
	}
}
