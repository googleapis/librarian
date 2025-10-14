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

package dart

import (
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/sidekick/internal/api"
	"github.com/googleapis/librarian/internal/sidekick/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/internal/parser"
)

var (
	testdataDir, _ = filepath.Abs("../../testdata")
)

func TestFromProtobuf(t *testing.T) {
	requireProtoc(t)
	outDir := t.TempDir()

	cfg := &config.Config{
		General: config.GeneralConfig{
			SpecificationFormat: "protobuf",
			ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			SpecificationSource: "google/cloud/secretmanager/v1",
		},
		Source: map[string]string{
			"googleapis-root": path.Join(testdataDir, "googleapis"),
		},
		Codec: map[string]string{
			"api-keys-environment-variables": "GOOGLE_API_KEY,GEMINI_API_KEY",
			"issue-tracker-url":              "http://www.example.com/issues",
			"copyright-year":                 "2025",
			"not-for-publication":            "true",
			"version":                        "0.1.0",
			"skip-format":                    "true",
			"package:googleapis_auth":        "^2.0.0",
			"package:google_cloud_gax":       "^1.2.3",
			"package:http":                   "^4.5.6",
			"package:google_cloud_location":  "^7.8.9",
			"package:google_cloud_protobuf":  "^0.1.2",
			"proto:google.protobuf":          "package:google_cloud_protobuf/protobuf.dart",
			"proto:google.cloud.location":    "package:google_cloud_location/location.dart",
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(model, outDir, cfg); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"pubspec.yaml", "lib/secretmanager.dart", "README.md"} {
		filename := path.Join(outDir, expected)
		stat, err := os.Stat(filename)
		if os.IsNotExist(err) {
			t.Errorf("missing %s: %s", filename, err)
		}
		if stat.Mode().Perm()|0666 != 0666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}
}

func TestGeneratedFiles(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{"package:google_cloud_gax": "^1.2.3", "package:http": "^4.5.6"})
	files := generatedFiles(model)
	if len(files) == 0 {
		t.Errorf("expected a non-empty list of template files from generatedFiles()")
	}

	// Validate that main.dart was replaced with {servicename}.dart.
	for _, fileInfo := range files {
		if filepath.Base(fileInfo.OutputPath) == "main.dart" {
			t.Errorf("expected the main.dart template to be generated as {servicename}.dart")
		}
		if filepath.Base(fileInfo.OutputPath) == "LICENSE.txt" {
			t.Errorf("expected the LICENSE.txt template to be generated as LICENSE")
		}
	}
}

func TestTemplatesAvailable(t *testing.T) {
	var count = 0
	fs.WalkDir(dartTemplates, "templates", func(path string, d fs.DirEntry, err error) error {
		if filepath.Ext(path) != ".mustache" {
			return nil
		}
		if strings.Count(d.Name(), ".") == 1 {
			// skip partials
			return nil
		}
		count++
		return nil
	})

	if count == 0 {
		t.Errorf("no dart templates found")
	}
}

func requireProtoc(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("skipping test because protoc is not installed")
	}
}
