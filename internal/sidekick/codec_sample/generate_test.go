// Copyright 2025 Google LLC
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

package codec_sample

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

var (
	testdataDir, _ = filepath.Abs("../testdata")
)

func TestFromProtobuf(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("skipping test because protoc is not installed")
	}
	outDir := t.TempDir()

	cfg := &sidekickconfig.Config{
		General: sidekickconfig.GeneralConfig{
			SpecificationFormat: config.SpecProtobuf,
			ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			SpecificationSource: "google/cloud/secretmanager/v1",
		},
		Source: map[string]string{
			"googleapis-root": path.Join(testdataDir, "../../testdata/googleapis"),
		},
		Codec: map[string]string{
			"copyright-year":              "2025",
			"not-for-publication":         "true",
			"version":                     "0.1.0",
			"skip-format":                 "true",
			"proto:google.protobuf":       "package:google_cloud_protobuf/protobuf.dart",
			"proto:google.cloud.location": "package:google_cloud_location/location.dart",
		},
	}
	model, err := parser.CreateModel(parser.NewModelConfigFromSidekickConfig(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(t.Context(), model, outDir, cfg); err != nil {
		t.Fatal(err)
	}
	filename := path.Join(outDir, "README.md")
	stat, err := os.Stat(filename)
	if os.IsNotExist(err) {
		t.Errorf("missing %s: %s", filename, err)
	}
	if stat.Mode().Perm()|0666 != 0666 {
		t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
	}
}
