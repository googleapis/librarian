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

func TestGenerateField_Deprecated(t *testing.T) {
	for _, test := range []struct {
		name           string
		deprecated     bool
		typeDeprecated bool
		repeated       bool
		want           string
		endStr         string
	}{
		{
			name:       "deprecated",
			deprecated: true,
			repeated:   false,
			want:       "  /// -- field marker --\n  @available(*, deprecated)\n  public var normalField: Swift.String",
			endStr:     "public var normalField: Swift.String",
		},
		{
			name:       "not-deprecated",
			deprecated: false,
			repeated:   false,
			want:       "  /// -- field marker --\n  public var normalField: Swift.String",
			endStr:     "public var normalField: Swift.String",
		},
		{
			name:       "deprecated-repeated",
			deprecated: true,
			repeated:   true,
			want:       "  /// -- field marker --\n  @available(*, deprecated)\n  public var normalField: [Swift.String]",
			endStr:     "public var normalField: [Swift.String]",
		},
		{
			// Naming a deprecated type warns at the declaration, even though
			// the field itself is not deprecated.
			name:           "deprecated-type",
			typeDeprecated: true,
			want:           "  /// -- field marker --\n  #if hasAttribute(diagnose)\n  @diagnose(DeprecatedDeclaration, as: ignored)\n  #endif\n  public var normalField: DeprecatedMessage",
			endStr:         "public var normalField: DeprecatedMessage",
		},
		{
			// Swift does not diagnose deprecated references inside a
			// deprecated declaration, so `@diagnose` would be redundant.
			name:           "deprecated-field-and-type",
			deprecated:     true,
			typeDeprecated: true,
			want:           "  /// -- field marker --\n  @available(*, deprecated)\n  public var normalField: DeprecatedMessage",
			endStr:         "public var normalField: DeprecatedMessage",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			deprecatedMessage := api.NewTestMessage("DeprecatedMessage").
				WithPackage("google.cloud.test.v1").
				WithDeprecated(true)

			field := api.NewTestField("normal_field").
				WithDocumentation("-- field marker --").
				WithType(api.TypezString).
				WithDeprecated(test.deprecated)
			if test.typeDeprecated {
				field = field.WithMessageType(deprecatedMessage)
			}
			if test.repeated {
				field = field.WithRepeated()
			}

			msg := api.NewTestMessage("TestMessage").
				WithPackage("google.cloud.test.v1").
				WithFields(field)

			model := api.NewTestAPI([]*api.Message{msg, deprecatedMessage}, nil, nil)
			if err := Generate(t.Context(), model, outDir, &config.Library{}, nil); err != nil {
				t.Fatal(err)
			}

			filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "TestMessage.swift")
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			contentStr := string(content)

			got := extractBlock(t, contentStr, "  /// -- field marker --", test.endStr)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
