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

// Package gcloud provides a code generator for gcloud commands.
package gcloud

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"text/template"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

//go:embed all:templates
var templates embed.FS

// Generate is the package entry point. It builds the model, renders main.go,
// writes it, then renders any other generated files via
// language.GenerateFromModel.
func Generate(model *api.API, outdir string) error {
	cliModel := constructCLIModel(model)
	contents, err := renderMain(cliModel)
	if err != nil {
		return err
	}
	if err := writeMain(outdir, contents); err != nil {
		return err
	}
	return renderReadme(outdir, model)
}

// renderMain renders the main.go contents from the CLI model. The template
// output is run through go/format so the golden file is gofmt-stable.
func renderMain(model CLIModel) (string, error) {
	t, err := template.ParseFS(templates, "templates/package/cli.go.tmpl")
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, model); err != nil {
		return "", err
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("formatting generated main.go: %w", err)
	}
	return string(formatted), nil
}

func writeMain(outdir, contents string) error {
	destination := filepath.Join(outdir, "main.go")
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	return os.WriteFile(destination, []byte(contents), 0666)
}

// renderReadme renders README.md via language.GenerateFromModel.
func renderReadme(outdir string, model *api.API) error {
	provider := func(name string) (string, error) {
		contents, err := templates.ReadFile(name)
		if err != nil {
			return "", err
		}
		return string(contents), nil
	}
	generatedFiles := []language.GeneratedFile{
		{TemplatePath: "templates/package/README.md.mustache", OutputPath: "README.md"},
	}
	return language.GenerateFromModel(outdir, model, provider, generatedFiles)
}
