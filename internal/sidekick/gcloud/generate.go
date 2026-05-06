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
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"text/template"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

//go:embed all:templates
var templates embed.FS

// modulePath is the Go module path used to construct surface package
// import paths in the generated cmd/gcloud/main.go. The generator does
// not emit a go.mod today; downstream tooling supplies one.
const modulePath = "cloud.google.com/go/gcloud"

// Generate writes the gcloud binary tree for the given API models. For
// each model it emits internal/generated/<name>/commands.go exposing a
// Command() function, then writes cmd/gcloud/main.go that registers each
// surface under the gcloud root.
func Generate(models []*api.API, outdir string) error {
	if len(models) == 0 {
		return errors.New("gcloud: Generate requires at least one model")
	}
	main := CLIModel{ModulePath: modulePath}
	for _, model := range models {
		surface := constructSurfaceModel(model)
		if err := writeSurface(outdir, surface); err != nil {
			return err
		}
		main.Surfaces = append(main.Surfaces, SurfaceRef{PackageName: surface.PackageName})
	}
	return writeMain(outdir, main)
}

// writeSurface writes internal/generated/<PackageName>/commands.go for a
// single surface.
func writeSurface(outdir string, model SurfaceModel) error {
	contents, err := renderSurface(model)
	if err != nil {
		return err
	}
	dir := filepath.Join(outdir, "internal", "generated", model.PackageName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "commands.go"), []byte(contents), 0o666)
}

// renderSurface renders a surface's commands.go from its template. The
// template output is run through go/format so the golden file is
// gofmt-stable.
func renderSurface(model SurfaceModel) (string, error) {
	t, err := template.ParseFS(templates, "templates/package/surface_commands.go.tmpl")
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, model); err != nil {
		return "", err
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("formatting generated %s/commands.go: %w", model.PackageName, err)
	}
	return string(formatted), nil
}

// writeMain writes cmd/gcloud/main.go registering every surface.
func writeMain(outdir string, model CLIModel) error {
	contents, err := renderMain(model)
	if err != nil {
		return err
	}
	dir := filepath.Join(outdir, "cmd", "gcloud")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "main.go"), []byte(contents), 0o666)
}

// renderMain renders the cmd/gcloud/main.go contents. The template output
// is run through go/format so the golden file is gofmt-stable.
func renderMain(model CLIModel) (string, error) {
	t, err := template.ParseFS(templates, "templates/package/cmd_gcloud_main.go.tmpl")
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, model); err != nil {
		return "", err
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("formatting generated cmd/gcloud/main.go: %w", err)
	}
	return string(formatted), nil
}
