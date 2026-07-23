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
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
)

var (
	testdataDir, _ = filepath.Abs("../testdata")
)

func TestFromProtobuf(t *testing.T) {
	requireProtoc(t)
	outDir := t.TempDir()

	cfg := &parser.ModelConfig{
		SpecificationFormat: config.SpecProtobuf,
		ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
		SpecificationSource: "google/cloud/secretmanager/v1",
		Source: &sources.SourceConfig{
			Sources: &sources.Sources{
				Googleapis: path.Join(testdataDir, "../../testdata/googleapis"),
			},
			ActiveRoots: []string{"googleapis"},
		},
		Codec: map[string]string{
			"api-keys-environment-variables": "GOOGLE_API_KEY,GEMINI_API_KEY",
			"issue-tracker-url":              "http://www.example.com/issues",
			"copyright-year":                 "2025",
			"not-for-publication":            "true",
			"version":                        "0.1.0",
			"skip-format":                    "true",
			"package:google_cloud_rpc":       "^1.2.3",
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
	if err := Generate(t.Context(), model, outDir, cfg.Codec); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"pubspec.yaml", "lib/secretmanager.dart", "README.md", "skills/google_cloud_secretmanager_v1-tests/SKILL.md"} {
		filename := path.Join(outDir, expected)
		stat, err := os.Stat(filename)
		if err != nil {
			t.Errorf("missing or cannot stat %s: %s", filename, err)
			continue
		}
		if stat.Mode().Perm()|0o666 != 0o666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}

	skillContent, err := os.ReadFile(path.Join(outDir, "skills/google_cloud_secretmanager_v1-tests/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(skillContent)
	if !strings.Contains(contentStr, "- Code that uses `SecretManagerService` can be tested by injecting the\n  fake `FakeSecretManagerService`.") {
		t.Errorf("expected skill file to map SecretManagerService to its fake, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "FakeSecretManagerService") {
		t.Errorf("expected skill file to mention FakeSecretManagerService, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "package:google_cloud_secretmanager_v1") {
		t.Errorf("expected skill file to mention package:google_cloud_secretmanager_v1, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "throw TooManyRequestsException(") {
		t.Errorf("expected skill file to throw TooManyRequestsException, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "response: http.Response('', 429),") {
		t.Errorf("expected skill file to fill in TooManyRequestsException parameters, got:\n%s", contentStr)
	}
}

func TestGeneratedFiles_Skills(t *testing.T) {
	// Case 1: No services -> FakeList is empty -> skills file should not be generated.
	{
		model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
		model.PackageName = "test"
		annotate := newAnnotateModel(model)
		options := maps.Clone(requiredConfig)
		if err := annotate.annotateModel(options); err != nil {
			t.Fatal(err)
		}
		files := generatedFiles(model)
		for _, fileInfo := range files {
			if strings.Contains(fileInfo.OutputPath, "skills/") {
				t.Errorf("expected no skill files to be generated when there are no services, got %q", fileInfo.OutputPath)
			}
		}
	}

	// Case 2: Has services -> FakeList is not empty -> skills file should be generated at the correct path.
	{
		service := &api.Service{
			Name: "MyService",
		}
		model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{service})
		model.PackageName = "test"
		annotate := newAnnotateModel(model)
		options := maps.Clone(requiredConfig)
		if err := annotate.annotateModel(options); err != nil {
			t.Fatal(err)
		}
		files := generatedFiles(model)
		foundSkill := false
		expectedPath := filepath.Join("skills", "google_cloud_test-tests", "SKILL.md")
		for _, fileInfo := range files {
			if fileInfo.OutputPath == expectedPath {
				foundSkill = true
				break
			}
		}
		if !foundSkill {
			t.Errorf("expected skill file at %q, but not found", expectedPath)
		}
	}
}

func TestGeneratedFiles(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	annotate := newAnnotateModel(model)

	options := maps.Clone(requiredConfig)
	maps.Copy(options, map[string]string{"package:google_cloud_rpc": "^1.2.3", "package:http": "^4.5.6"})

	annotate.annotateModel(options)
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
