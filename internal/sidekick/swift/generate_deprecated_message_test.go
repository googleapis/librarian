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

func TestGenerateMessage_Deprecated(t *testing.T) {
	for _, test := range []struct {
		name             string
		topDeprecated    bool
		nestedDeprecated bool
		wantTop          string
		wantNested       string
	}{
		{
			name:             "deprecated-both",
			topDeprecated:    true,
			nestedDeprecated: true,
			wantTop:          "/// -- top marker --\n@available(*, deprecated)\npublic struct TopMessage",
			wantNested:       "  /// -- nested marker --\n  @available(*, deprecated)\n  public struct NestedMessage",
		},
		{
			name:             "deprecated-top-only",
			topDeprecated:    true,
			nestedDeprecated: false,
			wantTop:          "/// -- top marker --\n@available(*, deprecated)\npublic struct TopMessage",
			wantNested:       "  /// -- nested marker --\n  public struct NestedMessage",
		},
		{
			name:             "deprecated-nested-only",
			topDeprecated:    false,
			nestedDeprecated: true,
			wantTop:          "/// -- top marker --\npublic struct TopMessage",
			wantNested:       "  /// -- nested marker --\n  @available(*, deprecated)\n  public struct NestedMessage",
		},
		{
			name:             "not-deprecated",
			topDeprecated:    false,
			nestedDeprecated: false,
			wantTop:          "/// -- top marker --\npublic struct TopMessage",
			wantNested:       "  /// -- nested marker --\n  public struct NestedMessage",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			nested := &api.Message{
				Name:          "NestedMessage",
				Package:       "google.cloud.test.v1",
				ID:            ".google.cloud.test.v1.TopMessage.NestedMessage",
				Deprecated:    test.nestedDeprecated,
				Documentation: "-- nested marker --",
			}

			top := &api.Message{
				Name:          "TopMessage",
				Package:       "google.cloud.test.v1",
				ID:            ".google.cloud.test.v1.TopMessage",
				Deprecated:    test.topDeprecated,
				Documentation: "-- top marker --",
				Messages:      []*api.Message{nested},
			}

			model := api.NewTestAPI([]*api.Message{top}, nil, nil)
			model.PackageName = "google.cloud.test.v1"
			cfg := &parser.ModelConfig{}
			if err := Generate(t.Context(), model, outDir, cfg, nil); err != nil {
				t.Fatal(err)
			}

			filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "TopMessage.swift")
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			contentStr := string(content)

			gotTop := extractBlock(t, contentStr, "/// -- top marker --", "public struct TopMessage")
			if diff := cmp.Diff(test.wantTop, gotTop); diff != "" {
				t.Errorf("mismatch top (-want +got):\n%s", diff)
			}

			gotNested := extractBlock(t, contentStr, "  /// -- nested marker --", "public struct NestedMessage")
			if diff := cmp.Diff(test.wantNested, gotNested); diff != "" {
				t.Errorf("mismatch nested (-want +got):\n%s", diff)
			}
		})
	}
}
