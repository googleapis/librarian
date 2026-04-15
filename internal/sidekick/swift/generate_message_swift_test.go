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

func TestGenerateMessage_Files(t *testing.T) {
	outDir := t.TempDir()

	secret := &api.Message{Name: "Secret", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.Secret"}
	volume := &api.Message{Name: "Volume", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.Volume"}

	model := api.NewTestAPI([]*api.Message{secret, volume}, []*api.Enum{}, []*api.Service{})
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
	for _, expected := range []string{"Secret.swift", "Volume.swift"} {
		filename := filepath.Join(expectedDir, expected)
		if _, err := os.Stat(filename); err != nil {
			t.Error(err)
		}
	}
}
