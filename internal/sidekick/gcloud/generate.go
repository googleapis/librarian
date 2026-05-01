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

package gcloud

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/surfer"
)

//go:embed all:templates
var templates embed.FS

// Generate emits a self-contained Go CLI module under outDir for the given
// API model. The generated module imports modulePath as its Go module path.
func Generate(model *api.API, outDir, modulePath string) error {
	root, err := surfer.BuildCommandTree(model, nil)
	if err != nil {
		return fmt.Errorf("gcloud: build command tree: %w", err)
	}
	data, err := annotate(root, model, modulePath)
	if err != nil {
		return fmt.Errorf("gcloud: annotate: %w", err)
	}
	return render(data, outDir)
}

func render(data *templateData, outDir string) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(outDir, "cmd"), 0755); err != nil {
		return err
	}
	if err := renderTemplate("templates/go.mod.tmpl", data, filepath.Join(outDir, "go.mod"), false); err != nil {
		return err
	}
	if err := renderTemplate("templates/main.go.tmpl", data, filepath.Join(outDir, "main.go"), true); err != nil {
		return err
	}
	for _, sg := range data.Subgroups {
		path := filepath.Join(outDir, "cmd", sg.Name+".go")
		if err := renderGroup(sg, data, path); err != nil {
			return err
		}
	}
	return nil
}

func renderTemplate(name string, data *templateData, outPath string, formatGo bool) error {
	tmpl, err := template.New(filepath.Base(name)).Funcs(funcs()).ParseFS(templates, name)
	if err != nil {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute %s: %w", name, err)
	}
	out := buf.Bytes()
	if formatGo {
		formatted, err := format.Source(out)
		if err != nil {
			return fmt.Errorf("format %s: %w\n--- generated source ---\n%s", outPath, err, out)
		}
		out = formatted
	}
	return os.WriteFile(outPath, out, 0644)
}

func renderGroup(sg *subgroup, data *templateData, outPath string) error {
	tmpl, err := template.New("group.go.tmpl").Funcs(funcs()).ParseFS(templates, "templates/cmd/group.go.tmpl")
	if err != nil {
		return fmt.Errorf("parse group template: %w", err)
	}
	payload := struct {
		LicenseHeader string
		ModulePath    string
		RootName      string
		Group         *subgroup
	}{
		LicenseHeader: data.LicenseHeader,
		ModulePath:    data.ModulePath,
		RootName:      data.RootName,
		Group:         sg,
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, payload); err != nil {
		return fmt.Errorf("execute group template: %w", err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("format %s: %w\n--- generated source ---\n%s", outPath, err, buf.String())
	}
	return os.WriteFile(outPath, formatted, 0644)
}

func funcs() template.FuncMap {
	return template.FuncMap{
		"goString": goStringLiteral,
		"join":     strings.Join,
	}
}

func goStringLiteral(s string) string {
	return fmt.Sprintf("%q", s)
}
