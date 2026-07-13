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

func TestGenerateEnum_Deprecated(t *testing.T) {
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
			wantEnum:       "/// -- enum marker --\n@available(*, deprecated)\npublic enum Status",
			wantCase:       "/// -- case marker --\n  case unspecified",
		},
		{
			name:           "deprecated-value",
			enumDeprecated: false,
			valDeprecated:  true,
			wantEnum:       "/// -- enum marker --\npublic enum Status",
			wantCase:       "/// -- case marker --\n  @available(*, deprecated)\n  case unspecified",
		},
		{
			name:           "both-deprecated",
			enumDeprecated: true,
			valDeprecated:  true,
			wantEnum:       "/// -- enum marker --\n@available(*, deprecated)\npublic enum Status",
			wantCase:       "/// -- case marker --\n  @available(*, deprecated)\n  case unspecified",
		},
		{
			name:           "not-deprecated",
			enumDeprecated: false,
			valDeprecated:  false,
			wantEnum:       "/// -- enum marker --\npublic enum Status",
			wantCase:       "/// -- case marker --\n  case unspecified",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			enum := &api.Enum{
				Name:          "Status",
				Package:       "google.cloud.test.v1",
				ID:            ".google.cloud.test.v1.Status",
				Deprecated:    test.enumDeprecated,
				Documentation: "-- enum marker --",
			}
			enum.Values = []*api.EnumValue{
				{
					Name:          "STATUS_UNSPECIFIED",
					Number:        0,
					Parent:        enum,
					Deprecated:    test.valDeprecated,
					Documentation: "-- case marker --",
				},
			}
			enum.UniqueNumberValues = enum.Values

			model := api.NewTestAPI(nil, []*api.Enum{enum}, nil)
			model.PackageName = "google.cloud.test.v1"
			cfg := &parser.ModelConfig{}
			if err := Generate(t.Context(), model, outDir, cfg, nil); err != nil {
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
