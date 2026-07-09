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

func TestGenerateField_Deprecated(t *testing.T) {
	for _, test := range []struct {
		name       string
		deprecated bool
		repeated   bool
		want       string
		endStr     string
	}{
		{
			name:       "deprecated",
			deprecated: true,
			want:       "  /// -- field marker --\n  @available(*, deprecated)\n  public var normalField: Swift.String",
			endStr:     "public var normalField: Swift.String",
		},
		{
			name:       "not-deprecated",
			deprecated: false,
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
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			field := &api.Field{
				Name:          "normal_field",
				Documentation: "-- field marker --",
				ID:            ".google.cloud.test.v1.TestMessage.normal_field",
				Typez:         api.TypezString,
				Deprecated:    test.deprecated,
				Repeated:      test.repeated,
			}

			msg := &api.Message{
				Name:    "TestMessage",
				Package: "google.cloud.test.v1",
				ID:      ".google.cloud.test.v1.TestMessage",
				Fields:  []*api.Field{field},
			}

			model := api.NewTestAPI([]*api.Message{msg}, nil, nil)
			model.PackageName = "google.cloud.test.v1"
			cfg := &parser.ModelConfig{}
			if err := Generate(t.Context(), model, outDir, cfg, nil); err != nil {
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
