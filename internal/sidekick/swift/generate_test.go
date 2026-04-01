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
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
)

func TestFromProtobuf(t *testing.T) {
	testdataDir, err := filepath.Abs("../../testdata")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("skipping test because protoc is not installed")
	}
	outDir := t.TempDir()

	cfg := &parser.ModelConfig{
		SpecificationFormat: config.SpecProtobuf,
		ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
		SpecificationSource: "google/cloud/secretmanager/v1",
		Source: &sources.SourceConfig{
			Sources: &sources.Sources{
				Googleapis: filepath.Join(testdataDir, "googleapis"),
			},
			ActiveRoots: []string{"googleapis"},
		},
		Codec: map[string]string{
			"copyright-year":      "2026",
			"not-for-publication": "true",
			"version":             "0.1.0",
			"skip-format":         "true",
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(t.Context(), model, outDir, cfg); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(outDir, "README.md")
	stat, err := os.Stat(filename)
	if os.IsNotExist(err) {
		t.Errorf("missing %s: %s", filename, err)
	}
	if stat.Mode().Perm()|0666 != 0666 {
		t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
	}
}
