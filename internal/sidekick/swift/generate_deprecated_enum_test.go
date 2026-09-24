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

func TestGenerateEnum_Deprecated(t *testing.T) {
	const noteDoc = `///
/// - Note: Adding cases to this enumeration is not considered a breaking change.
///   Always include an ` + "`@unknown default:`" + ` case when switching over this type.
///   Do not pattern-match against ` + "`unknownStringValue`" + ` or ` + "`unknownIntValue`" + `
///   expecting specific values to remain unparsed; future releases may promote
///   them to named cases.
`
	for _, test := range []struct {
		name           string
		enumDeprecated bool
		valDeprecated  bool
		wantEnum       string
		wantCase       string
	}{
		{
			name:           "deprecated-enum",
			enumDeprecated: true,
			valDeprecated:  false,
			wantEnum:       "/// -- enum marker --\n" + noteDoc + "@available(*, deprecated)\npublic enum Status",
			wantCase:       "/// -- case marker --\n  case unspecified",
		},
		{
			name:           "deprecated-value",
			enumDeprecated: false,
			valDeprecated:  true,
			wantEnum:       "/// -- enum marker --\n" + noteDoc + "public enum Status",
			wantCase:       "/// -- case marker --\n  @available(*, deprecated)\n  case unspecified",
		},
		{
			name:           "both-deprecated",
			enumDeprecated: true,
			valDeprecated:  true,
			wantEnum:       "/// -- enum marker --\n" + noteDoc + "@available(*, deprecated)\npublic enum Status",
			wantCase:       "/// -- case marker --\n  @available(*, deprecated)\n  case unspecified",
		},
		{
			name:           "not-deprecated",
			enumDeprecated: false,
			valDeprecated:  false,
			wantEnum:       "/// -- enum marker --\n" + noteDoc + "public enum Status",
			wantCase:       "/// -- case marker --\n  case unspecified",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			val := api.NewTestEnumValue("STATUS_UNSPECIFIED", 0).
				WithDeprecated(test.valDeprecated).
				WithDocumentation("-- case marker --")
			enum := api.NewTestEnum("Status").
				WithPackage("google.cloud.test.v1").
				WithDeprecated(test.enumDeprecated).
				WithDocumentation("-- enum marker --").
				WithValues(val)

			model := api.NewTestAPI(nil, []*api.Enum{enum}, nil)
			if err := Generate(t.Context(), model, outDir, &config.Library{}, nil); err != nil {
				t.Fatal(err)
			}

			filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "Status.swift")
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			contentStr := string(content)

			got := extractBlock(t, contentStr, "/// -- enum marker --", "public enum Status")
			if diff := cmp.Diff(test.wantEnum, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			got = extractBlock(t, contentStr, "/// -- case marker --", "case unspecified")
			if diff := cmp.Diff(test.wantCase, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGenerateEnum_Diagnose covers the initializers that assign enum cases.
//
// Only assignment warns. `intValue`, `stringValue` and `encode(to:)` pattern
// match instead, which Swift does not diagnose, so they need no guard.
func TestGenerateEnum_Diagnose(t *testing.T) {
	for _, test := range []struct {
		name              string
		enumDeprecated    bool
		defaultDeprecated bool
		valueDeprecated   bool
		wantDefault       bool
		wantValues        bool
	}{
		{
			name:              "deprecated-default-value",
			defaultDeprecated: true,
			wantDefault:       true,
			wantValues:        true,
		},
		{
			// `init()` only names the default value, so it stays clean.
			name:            "deprecated-other-value",
			valueDeprecated: true,
			wantDefault:     false,
			wantValues:      true,
		},
		{
			// Swift does not diagnose deprecated references inside a
			// deprecated declaration.
			name:              "deprecated-enum",
			enumDeprecated:    true,
			defaultDeprecated: true,
			valueDeprecated:   true,
			wantDefault:       false,
			wantValues:        false,
		},
		{
			name:        "not-deprecated",
			wantDefault: false,
			wantValues:  false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			enum := api.NewTestEnum("Status").
				WithPackage("google.cloud.test.v1").
				WithDeprecated(test.enumDeprecated).
				WithValues(
					api.NewTestEnumValue("STATUS_UNSPECIFIED", 0).
						WithDeprecated(test.defaultDeprecated),
					api.NewTestEnumValue("STATUS_OK", 1).
						WithDeprecated(test.valueDeprecated),
				)

			model := api.NewTestAPI(nil, []*api.Enum{enum}, nil).
				WithPackageName("google.cloud.test.v1")
			if err := Generate(t.Context(), model, outDir, &config.Library{}, nil); err != nil {
				t.Fatal(err)
			}

			filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "Status.swift")
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			contentStr := string(content)

			checkDiagnose(t, contentStr, "  ", "public init() {", test.wantDefault)
			checkDiagnose(t, contentStr, "  ", "public init(stringValue: Swift.String) {", test.wantValues)
			checkDiagnose(t, contentStr, "  ", "public init(intValue: Int) {", test.wantValues)
		})
	}
}
