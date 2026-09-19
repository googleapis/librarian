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

// Package cpp provides a C++ code generator for Librarian.
package cpp

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

//go:embed all:templates
var templates embed.FS

// Generate generates C++ code from the model into outdir.
func Generate(_ context.Context, model *api.API, outdir string, libCfg *config.CppLibrary) error {
	c := newCodec(libCfg)
	if err := c.annotateModel(model); err != nil {
		return err
	}
	provider := func(name string) (string, error) {
		contents, err := templates.ReadFile(name)
		if err != nil {
			return "", err
		}
		return string(contents), nil
	}
	generatedFiles := language.WalkTemplatesDir(templates, "templates/cmake")
	if err := validateOutputContainment(outdir, generatedFiles); err != nil {
		return err
	}
	return language.GenerateFromModel(outdir, model, provider, generatedFiles)
}

func validateOutputContainment(outdir string, files []language.GeneratedFile) error {
	absOut, err := filepath.Abs(outdir)
	if err != nil {
		return fmt.Errorf("resolving outdir %q: %w", outdir, err)
	}
	for _, gen := range files {
		cleanPath := filepath.Clean(gen.OutputPath)
		if cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
			return fmt.Errorf("output path %q escapes output directory %q", gen.OutputPath, outdir)
		}
		targetPath := filepath.Join(absOut, gen.OutputPath)
		rel, err := filepath.Rel(absOut, targetPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("output path %q escapes output directory %q", gen.OutputPath, outdir)
		}
	}
	return nil
}
