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

package language

import (
	"embed"
	"os"
	"path"
	"testing"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

//go:embed all:testTemplates
var templates embed.FS

func TestGenerate(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	outDir := t.TempDir()

	// The list of files to generate, just load them from the embedded templates.
	generatedFiles := WalkTemplatesDir(templates, "testTemplates")
	err := GenerateFromModel(outDir, model, provider, generatedFiles)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"README.md", "test001.txt"} {
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

func TestGenerateService(t *testing.T) {
	service := &api.Service{
		Name: "ExpectedName",
	}
	outDir := t.TempDir()

	gen := GeneratedFile{
		TemplatePath: "testTemplates/test002.mustache",
		OutputPath:   "test002.txt",
	}
	err := GenerateService(outDir, service, provider, gen)
	if err != nil {
		t.Fatal(err)
	}
	verifyElementOutput(t, outDir)
}

func TestGenerateMessage(t *testing.T) {
	message := &api.Message{
		Name: "ExpectedName",
	}
	outDir := t.TempDir()

	gen := GeneratedFile{
		TemplatePath: "testTemplates/test002.mustache",
		OutputPath:   "test002.txt",
	}
	err := GenerateMessage(outDir, message, provider, gen)
	if err != nil {
		t.Fatal(err)
	}
	verifyElementOutput(t, outDir)
}

func TestGenerateEnum(t *testing.T) {
	enum := &api.Enum{
		Name: "ExpectedName",
	}
	outDir := t.TempDir()

	gen := GeneratedFile{
		TemplatePath: "testTemplates/test002.mustache",
		OutputPath:   "test002.txt",
	}
	err := GenerateEnum(outDir, enum, provider, gen)
	if err != nil {
		t.Fatal(err)
	}
	verifyElementOutput(t, outDir)
}

func verifyElementOutput(t *testing.T, outDir string) {
	t.Helper()
	for _, expected := range []string{"test002.txt"} {
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

func provider(name string) (string, error) {
	contents, err := templates.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}
