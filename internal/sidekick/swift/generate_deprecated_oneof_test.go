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

func TestGenerateOneOf_Deprecated(t *testing.T) {
	for _, test := range []struct {
		name           string
		deprecated     bool
		typeDeprecated bool
		isObject       bool
		want           string
	}{
		{
			name:       "deprecated-scalar",
			deprecated: true,
			isObject:   false,
			want:       "    /// -- case marker --\n    @available(*, deprecated)\n    case fieldOne(Swift.String)",
		},
		{
			name:       "not-deprecated-scalar",
			deprecated: false,
			isObject:   false,
			want:       "    /// -- case marker --\n    case fieldOne(Swift.String)",
		},
		{
			name:       "deprecated-message",
			deprecated: true,
			isObject:   true,
			want:       "    /// -- case marker --\n    @available(*, deprecated)\n    indirect case fieldOne(Inner)",
		},
		{
			// Naming a deprecated type warns at the declaration, even though
			// the variant itself is not deprecated.
			name:           "deprecated-payload-type",
			typeDeprecated: true,
			isObject:       true,
			want:           "    /// -- case marker --\n    #if hasAttribute(diagnose)\n    @diagnose(DeprecatedDeclaration, as: ignored)\n    #endif\n    indirect case fieldOne(Inner)",
		},
		{
			// Swift does not diagnose deprecated references inside a
			// deprecated declaration, so `@diagnose` would be redundant.
			name:           "deprecated-variant-and-payload-type",
			deprecated:     true,
			typeDeprecated: true,
			isObject:       true,
			want:           "    /// -- case marker --\n    @available(*, deprecated)\n    indirect case fieldOne(Inner)",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			inner := api.NewTestMessage("Inner").
				WithPackage("google.cloud.test.v1").
				WithDeprecated(test.typeDeprecated)

			field := api.NewTestField("field_one").
				WithDocumentation("-- case marker --").
				WithDeprecated(test.deprecated)
			if test.isObject {
				field.WithMessageType(inner)
			} else {
				field.WithType(api.TypezString)
			}

			oneof := api.NewTestOneOf("choice").
				WithFields(field).
				WithDocumentation("-- property marker --")

			msg := api.NewTestMessage("TestMessage").
				WithPackage("google.cloud.test.v1").
				WithOneOfs(oneof)

			model := api.NewTestAPI([]*api.Message{msg, inner}, nil, nil)
			if err := Generate(t.Context(), model, outDir, &config.Library{}, nil); err != nil {
				t.Fatal(err)
			}

			filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "TestMessage.swift")
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			contentStr := string(content)

			endStr := "case fieldOne(Swift.String)"
			if test.isObject {
				endStr = "indirect case fieldOne(Inner)"
			}

			got := extractBlock(t, contentStr, "    /// -- case marker --", endStr)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}

			// Verify the oneof property in the message.
			// It should NOT be deprecated because api.OneOf doesn't have Deprecated field.
			gotProperty := extractBlock(t, contentStr, "  /// -- property marker --", "public var choice: ChoiceOneOf? = nil")
			wantProperty := "  /// -- property marker --\n  public var choice: ChoiceOneOf? = nil"
			if diff := cmp.Diff(wantProperty, gotProperty); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
