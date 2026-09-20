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

// Package python provides a code generator for Python GAPIC client libraries.
package python

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

//go:embed all:templates
var templates embed.FS

var (
	// ErrEscapeOutputDir is returned when an output file path resolves outside the output directory.
	ErrEscapeOutputDir = errors.New("output path escapes output directory")
	// ErrDuplicateOutputPath is returned when multiple generated files target the same output path.
	ErrDuplicateOutputPath = errors.New("duplicate output path")
	// ErrInvalidOutputPath is returned when an output file path is empty or absolute.
	ErrInvalidOutputPath = errors.New("invalid output path")
)

// Generate generates Python GAPIC client code from the api.API model.
func Generate(ctx context.Context, model *api.API, outdir string, library *config.Library) error {
	c, err := newCodec(model, library, outdir)
	if err != nil {
		return err
	}
	if err := c.annotateModel(); err != nil {
		return err
	}
	provider := func(name string) (string, error) {
		contents, err := templates.ReadFile(name)
		if err != nil {
			contents, err = templates.ReadFile(path.Join(path.Dir(name), "partials", path.Base(name)))
		}
		if err != nil {
			contents, err = templates.ReadFile(path.Join("templates", "partials", path.Base(name)))
		}
		if err != nil {
			return "", err
		}
		return string(contents), nil
	}

	pkgDir := c.packageDir()

	modelFiles := []language.GeneratedFile{
		{
			TemplatePath: "templates/gapic_version.py.mustache",
			OutputPath:   filepath.Join(pkgDir, "gapic_version.py"),
		},
		{
			TemplatePath: "templates/py.typed.mustache",
			OutputPath:   filepath.Join(pkgDir, "py.typed"),
		},
	}

	if c.isDefaultVersion() {
		rootDir := c.rootPackageDir()
		if rootDir != "" && rootDir != pkgDir {
			modelFiles = append(modelFiles,
				language.GeneratedFile{
					TemplatePath: "templates/gapic_version.py.mustache",
					OutputPath:   filepath.Join(rootDir, "gapic_version.py"),
				},
				language.GeneratedFile{
					TemplatePath: "templates/py.typed.mustache",
					OutputPath:   filepath.Join(rootDir, "py.typed"),
				},
			)
		}
	}

	if model.HasServices() {
		modelFiles = append(modelFiles, language.GeneratedFile{
			TemplatePath: "templates/services/__init__.py.mustache",
			OutputPath:   filepath.Join(pkgDir, "services", "__init__.py"),
		})
	}

	type serviceFilePair struct {
		service *api.Service
		file    language.GeneratedFile
	}
	var serviceFiles []serviceFilePair

	for _, service := range model.Services {
		ann, ok := service.Codec.(*serviceAnnotations)
		if !ok {
			continue
		}
		serviceDir := filepath.Join(pkgDir, "services", ann.DirectoryName)
		serviceFiles = append(serviceFiles,
			serviceFilePair{
				service: service,
				file: language.GeneratedFile{
					TemplatePath: "templates/services/service/__init__.py.mustache",
					OutputPath:   filepath.Join(serviceDir, "__init__.py"),
				},
			},
			serviceFilePair{
				service: service,
				file: language.GeneratedFile{
					TemplatePath: "templates/services/service/transports/README.rst.mustache",
					OutputPath:   filepath.Join(serviceDir, "transports", "README.rst"),
				},
			},
		)
	}

	allFiles := make([]language.GeneratedFile, 0, len(modelFiles)+len(serviceFiles))
	allFiles = append(allFiles, modelFiles...)
	for _, sf := range serviceFiles {
		allFiles = append(allFiles, sf.file)
	}

	if err := validateOutputContainment(outdir, allFiles); err != nil {
		return err
	}

	if err := language.GenerateFromModel(outdir, model, provider, modelFiles); err != nil {
		return err
	}

	for _, sf := range serviceFiles {
		if err := language.GenerateService(outdir, sf.service, provider, sf.file); err != nil {
			return err
		}
	}

	return nil
}

func validateOutputContainment(outdir string, files []language.GeneratedFile) error {
	absOutdir, err := filepath.Abs(outdir)
	if err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, file := range files {
		if file.OutputPath == "" {
			return fmt.Errorf("%w: output path is empty", ErrInvalidOutputPath)
		}
		if filepath.IsAbs(file.OutputPath) {
			return fmt.Errorf("%w: output path %q is absolute", ErrInvalidOutputPath, file.OutputPath)
		}
		cleanPath := filepath.Clean(file.OutputPath)
		if cleanPath == "." || strings.HasPrefix(cleanPath, "..") || strings.Contains(file.OutputPath, "..") {
			return fmt.Errorf("%w: output path %q escapes output directory", ErrEscapeOutputDir, file.OutputPath)
		}
		target := filepath.Join(absOutdir, cleanPath)
		rel, err := filepath.Rel(absOutdir, target)
		if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
			return fmt.Errorf("%w: output path %q escapes output directory", ErrEscapeOutputDir, file.OutputPath)
		}
		if seen[cleanPath] {
			return fmt.Errorf("%w: duplicate output path %q", ErrDuplicateOutputPath, file.OutputPath)
		}
		seen[cleanPath] = true
	}
	return nil
}
