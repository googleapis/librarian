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
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestGenerateOneOf(t *testing.T) {
	outDir := t.TempDir()

	inner := &api.Message{
		Name:    "Inner",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.Inner",
	}

	oneof := &api.OneOf{
		Name:          "choice",
		Documentation: "A group of fields where only one is set.",
	}

	outer := &api.Message{
		Name:    "Outer",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.Outer",
		Fields: []*api.Field{
			{
				Name:          "string_field",
				ID:            ".google.cloud.test.v1.Outer.string_field",
				Documentation: "A string field that is part of the oneof.",
				Typez:         api.STRING_TYPE,
				IsOneOf:       true,
				Group:         oneof,
			},
			{
				Name:          "message_field",
				ID:            ".google.cloud.test.v1.Outer.message_field",
				Documentation: "A message field that is part of the oneof.",
				Typez:         api.MESSAGE_TYPE,
				TypezID:       ".google.cloud.test.v1.Inner",
				IsOneOf:       true,
				Group:         oneof,
			},
			{
				Name:          "regular_int32",
				ID:            ".google.cloud.test.v1.Outer.regular_int32",
				Documentation: "A regular field.",
				Typez:         api.INT32_TYPE,
			},
			{
				Name:          "regular_string",
				ID:            ".google.cloud.test.v1.Outer.regular_string",
				Documentation: "Another regular field.",
				Typez:         api.STRING_TYPE,
			},
		},
		OneOfs: []*api.OneOf{oneof},
	}
	oneof.Fields = []*api.Field{outer.Fields[0], outer.Fields[1]}

	model := api.NewTestAPI([]*api.Message{outer, inner}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"
	cfg := &parser.ModelConfig{}
	if err := Generate(t.Context(), model, outDir, cfg, nil); err != nil {
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
	// detector test. On the other hand, checking that the oneof fields are defined propertly, and
	// that the constructor has the right arguments is more tedious and also becomes a change detector
	// test.
	//
	// To verify the code compile, use something like: https://godbolt.org/z/EE9G7KTr8
	want := `public struct Outer: Codable, Equatable {

  /// A regular field.
  public var regularInt32: Int32

  /// Another regular field.
  public var regularString: String

  /// A group of fields where only one is set.
  public var choice: OneOf_Choice?

  /// Initialize a new instance of ` + "`Outer`" + `.
  public init(
    regularInt32: Int32 = Int32(),
    regularString: String = String(),
    choice: OneOf_Choice? = nil,
  ) {
    self.regularInt32 = regularInt32
    self.regularString = regularString
    self.choice = choice
  }

  /// A group of fields where only one is set.
  public enum OneOf_Choice: Codable, Equatable {
    /// A string field that is part of the oneof.
    case stringField(String)
    /// A message field that is part of the oneof.
    indirect case messageField(Inner)
  }
}
`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
