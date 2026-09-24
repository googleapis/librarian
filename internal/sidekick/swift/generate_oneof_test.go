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

func TestGenerateOneOf(t *testing.T) {
	outDir := t.TempDir()

	inner := api.NewTestMessage("Inner").
		WithPackage("google.cloud.test.v1")

	oneofField1 := api.NewTestField("string_field").
		WithType(api.TypezString).
		WithDocumentation("A string field that is part of the oneof.")

	oneofField2 := api.NewTestField("message_field").
		WithMessageType(inner).
		WithDocumentation("A message field that is part of the oneof.")

	oneof := api.NewTestOneOf("choice").
		WithFields(oneofField1, oneofField2).
		WithDocumentation("A group of fields where only one is set.")

	regField1 := api.NewTestField("regular_int32").
		WithType(api.TypezInt32).
		WithDocumentation("A regular field.")

	regField2 := api.NewTestField("regular_string").
		WithType(api.TypezString).
		WithJSONName("regularStringSpecial").
		WithDocumentation("Another regular field.")

	outer := api.NewTestMessage("Outer").
		WithPackage("google.cloud.test.v1").
		WithOneOfs(oneof).
		WithFields(regField1, regField2)

	model := api.NewTestAPI([]*api.Message{outer, inner}, nil, nil)
	library := &config.Library{
		Swift: swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filename := filepath.Join(expectedDir, "Outer.swift")

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// Extract content from "public struct Outer" to the end
	startIdx := strings.Index(contentStr, "public struct Outer")
	if startIdx == -1 {
		t.Fatal("file does not contain 'public struct Outer'")
	}
	got := contentStr[startIdx:]

	// I (coryan@) don't particularly like testing a big string like this. It is a bit of a change
	// detector test. On the other hand, checking that the oneof fields are defined properly, and
	// that the constructor has the right arguments is more tedious and also becomes a change detector
	// test.
	//
	// To verify the code compile, use something like: https://godbolt.org/z/EE9G7KTr8
	want := `public struct Outer: Codable, Equatable, GoogleWKT._AnyPackable,
  Sendable {

  /// A regular field.
  public var regularInt32: Swift.Int32 = Swift.Int32()

  /// Another regular field.
  public var regularString: Swift.String = Swift.String()

  /// A group of fields where only one is set.
  public var choice: ChoiceOneOf? = nil

  @_spi(GoogleCloudInternal) public var _unknownFields: GoogleWKT._UnknownFields = .init()

  /// Initialize a new instance of ` + "`Outer`" + `.
  public init() {}

  /// Use ` + "`config`" + ` to return a new instance of this object, with some fields updated.
  ///
  /// Commonly used to initialize the value, for example:
  ///
  /// ` + "```" + `
  /// let value = Outer().with { $0.stringField = ... }
  /// ` + "```" + `
  public func with(_ config: (inout Self) throws -> Swift.Void) rethrows -> Self {
    var copy = self
    try config(&copy)
    return copy
  }

  private struct CodingKeys: CodingKey {
    var stringValue: Swift.String
    var intValue: Swift.Int? { nil }
    init(stringValue: Swift.String) { self.stringValue = stringValue }
    init?(intValue: Swift.Int) { nil }

    static let stringField = CodingKeys(stringValue: "stringField")
    static let messageField = CodingKeys(stringValue: "messageField")
    static let regularInt32 = CodingKeys(stringValue: "regularInt32")
    static let regularString = CodingKeys(stringValue: "regularStringSpecial")

    static let _knownKeys: Set<Swift.String> = [
      "stringField",
      "messageField",
      "regularInt32",
      "regularStringSpecial",
    ]
  }

  public init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    if let value = try container.decodeIfPresent(Swift.Int32.self, forKey: .regularInt32) {
      self.regularInt32 = value
    }
    if let value = try container.decodeIfPresent(Swift.String.self, forKey: .regularString) {
      self.regularString = value
    }

    var choice: ChoiceOneOf? = nil
    let choiceCheckAndSet = {
      if choice != nil {
        throw DecodingError.dataCorrupted(DecodingError.Context(codingPath: decoder.codingPath, debugDescription: "Multiple values set for oneof 'choice'"))
      }
      choice = $0
    }
    if let stringField = try container.decodeIfPresent(Swift.String.self, forKey: .stringField) {
      try choiceCheckAndSet(.stringField(stringField))
    }
    if let messageField = try container.decodeIfPresent(Inner.self, forKey: .messageField) {
      try choiceCheckAndSet(.messageField(messageField))
    }
    self.choice = choice
    for key in container.allKeys where !CodingKeys._knownKeys.contains(key.stringValue) {
      self._unknownFields.json[key.stringValue] = try container.decode(
        GoogleWKT.WKTValue.self, forKey: key)
    }
  }

  public func encode(to encoder: Encoder) throws {
    var container = encoder.container(keyedBy: CodingKeys.self)
    try container.encode(self.regularInt32, forKey: .regularInt32)
    try container.encode(self.regularString, forKey: .regularString)

    if let choice = self.choice {
      switch choice {
      case .stringField(let value):
        try container.encode(value, forKey: .stringField)
      case .messageField(let value):
        try container.encode(value, forKey: .messageField)
      }
    }
    for (key, value) in self._unknownFields.json {
      try container.encode(value, forKey: CodingKeys(stringValue: key))
    }
  }


  /// A group of fields where only one is set.
  public enum ChoiceOneOf: Codable, Equatable, Sendable {
    /// A string field that is part of the oneof.
    case stringField(Swift.String)
    /// A message field that is part of the oneof.
    indirect case messageField(Inner)
  }

  public static var _anyTypeUrl: Swift.String {
    return "type.googleapis.com/google.cloud.test.v1.Outer"
  }
  public init(fromAny any: GoogleWKT.WKTAny) throws {
    self = try GoogleWKT._slowAnyDeserialize(Self.self, from: any)
  }
  public func _pack() throws -> GoogleWKT.WKTStruct {
    return try GoogleWKT._slowAnySerialize(message: self)
  }
}
`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateOneOfWithKeyword(t *testing.T) {
	outDir := t.TempDir()

	headerField := api.NewTestField("header").
		WithType(api.TypezString).
		WithDocumentation("Specifies HTTP header name to extract JWT token.")

	queryField := api.NewTestField("query").
		WithType(api.TypezString).
		WithDocumentation("Specifies URL query parameter name to extract JWT token.")

	cookieField := api.NewTestField("cookie").
		WithType(api.TypezString).
		WithDocumentation("Specifies cookie name to extract JWT token.")

	oneof := api.NewTestOneOf("in").
		WithFields(headerField, queryField, cookieField)

	jwtLocation := api.NewTestMessage("JwtLocation").
		WithPackage("google.api").
		WithOneOfs(oneof)

	model := api.NewTestAPI([]*api.Message{jwtLocation}, nil, nil)
	library := &config.Library{
		Swift: swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleApi")
	filename := filepath.Join(expectedDir, "JwtLocation.swift")

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)
	got := extractBlock(t, contentStr, "public struct JwtLocation", "public enum InOneOf")

	// I (coryan@) don't particularly like testing a big string like this. It is a bit of a change
	// detector test. On the other hand, checking that the oneof fields are defined properly, and
	// that the constructor has the right arguments is more tedious and also becomes a change detector
	// test.
	//
	// To verify the code compile, use something like: https://godbolt.org/z/EE9G7KTr8
	want := `public struct JwtLocation: Codable, Equatable, GoogleWKT._AnyPackable,
  Sendable {

  public var ` + "`in`" + `: InOneOf? = nil

  @_spi(GoogleCloudInternal) public var _unknownFields: GoogleWKT._UnknownFields = .init()

  /// Initialize a new instance of ` + "`JwtLocation`" + `.
  public init() {}

  /// Use ` + "`config`" + ` to return a new instance of this object, with some fields updated.
  ///
  /// Commonly used to initialize the value, for example:
  ///
  /// ` + "```" + `
  /// let value = JwtLocation().with { $0.header = ... }
  /// ` + "```" + `
  public func with(_ config: (inout Self) throws -> Swift.Void) rethrows -> Self {
    var copy = self
    try config(&copy)
    return copy
  }

  private struct CodingKeys: CodingKey {
    var stringValue: Swift.String
    var intValue: Swift.Int? { nil }
    init(stringValue: Swift.String) { self.stringValue = stringValue }
    init?(intValue: Swift.Int) { nil }

    static let header = CodingKeys(stringValue: "header")
    static let query = CodingKeys(stringValue: "query")
    static let cookie = CodingKeys(stringValue: "cookie")

    static let _knownKeys: Set<Swift.String> = [
      "header",
      "query",
      "cookie",
    ]
  }

  public init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)

    var ` + "`in`" + `: InOneOf? = nil
    let inCheckAndSet = {
      if ` + "`in`" + ` != nil {
        throw DecodingError.dataCorrupted(DecodingError.Context(codingPath: decoder.codingPath, debugDescription: "Multiple values set for oneof '` + "`in`" + `'"))
      }
      ` + "`in`" + ` = $0
    }
    if let header = try container.decodeIfPresent(Swift.String.self, forKey: .header) {
      try inCheckAndSet(.header(header))
    }
    if let query = try container.decodeIfPresent(Swift.String.self, forKey: .query) {
      try inCheckAndSet(.query(query))
    }
    if let cookie = try container.decodeIfPresent(Swift.String.self, forKey: .cookie) {
      try inCheckAndSet(.cookie(cookie))
    }
    self.` + "`in` = `in`" + `
    for key in container.allKeys where !CodingKeys._knownKeys.contains(key.stringValue) {
      self._unknownFields.json[key.stringValue] = try container.decode(
        GoogleWKT.WKTValue.self, forKey: key)
    }
  }

  public func encode(to encoder: Encoder) throws {
    var container = encoder.container(keyedBy: CodingKeys.self)

    if let choice = self.` + "`in`" + ` {
      switch choice {
      case .header(let value):
        try container.encode(value, forKey: .header)
      case .query(let value):
        try container.encode(value, forKey: .query)
      case .cookie(let value):
        try container.encode(value, forKey: .cookie)
      }
    }
    for (key, value) in self._unknownFields.json {
      try container.encode(value, forKey: CodingKeys(stringValue: key))
    }
  }


  public enum InOneOf`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
