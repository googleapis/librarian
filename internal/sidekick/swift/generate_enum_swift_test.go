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

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestGenerateEnum_Files(t *testing.T) {
	outDir := t.TempDir()

	color := &api.Enum{Name: "Color", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.Color"}
	color.Values = []*api.EnumValue{{Name: "COLOR_UNSPECIFIED", Number: 0, Parent: color}}
	color.UniqueNumberValues = color.Values

	kind := &api.Enum{Name: "Kind", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.Kind"}
	kind.Values = []*api.EnumValue{{Name: "KIND_UNSPECIFIED", Number: 0, Parent: kind}}
	kind.UniqueNumberValues = kind.Values

	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{color, kind}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	for _, expected := range []string{"Color.swift", "Kind.swift"} {
		filename := filepath.Join(expectedDir, expected)
		if _, err := os.Stat(filename); err != nil {
			t.Error(err)
		}
	}
}
