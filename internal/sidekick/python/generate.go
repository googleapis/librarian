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
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cbroglie/mustache"
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
		cleanName := path.Clean(filepath.ToSlash(name))
		baseName := strings.TrimSuffix(path.Base(cleanName), ".mustache")
		candidates := []string{
			cleanName,
			cleanName + ".mustache",
			path.Join("templates", "partials", path.Base(cleanName)+".mustache"),
			path.Join("templates", "partials", baseName+".mustache"),
			path.Join("templates", "partials", path.Base(cleanName)),
		}
		for _, candidate := range candidates {
			if contents, err := templates.ReadFile(candidate); err == nil {
				return string(contents), nil
			}
		}
		return "", fmt.Errorf("template %q not found", name)
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

	modelAnn, _ := model.Codec.(*modelAnnotations)
	if modelAnn != nil && len(modelAnn.TypeFiles) > 0 {
		modelFiles = append(modelFiles, language.GeneratedFile{
			TemplatePath: "templates/types/__init__.py.mustache",
			OutputPath:   filepath.Join(pkgDir, "types", "__init__.py"),
		})
	}

	type typeFilePair struct {
		fileAnn *fileAnnotations
		file    language.GeneratedFile
	}
	var typeFiles []typeFilePair

	if modelAnn != nil {
		for _, fAnn := range modelAnn.AllTypeFiles {
			typeFiles = append(typeFiles, typeFilePair{
				fileAnn: fAnn,
				file: language.GeneratedFile{
					TemplatePath: "templates/types/proto.py.mustache",
					OutputPath:   filepath.Join(pkgDir, "types", fAnn.Stem+".py"),
				},
			})
		}
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
					TemplatePath: "templates/services/service/client.py.mustache",
					OutputPath:   filepath.Join(serviceDir, "client.py"),
				},
			},
			serviceFilePair{
				service: service,
				file: language.GeneratedFile{
					TemplatePath: "templates/services/service/transports/README.rst.mustache",
					OutputPath:   filepath.Join(serviceDir, "transports", "README.rst"),
				},
			},
			serviceFilePair{
				service: service,
				file: language.GeneratedFile{
					TemplatePath: "templates/services/service/transports/base.py.mustache",
					OutputPath:   filepath.Join(serviceDir, "transports", "base.py"),
				},
			},
			serviceFilePair{
				service: service,
				file: language.GeneratedFile{
					TemplatePath: "templates/services/service/transports/grpc.py.mustache",
					OutputPath:   filepath.Join(serviceDir, "transports", "grpc.py"),
				},
			},
			serviceFilePair{
				service: service,
				file: language.GeneratedFile{
					TemplatePath: "templates/services/service/transports/grpc_asyncio.py.mustache",
					OutputPath:   filepath.Join(serviceDir, "transports", "grpc_asyncio.py"),
				},
			},
			serviceFilePair{
				service: service,
				file: language.GeneratedFile{
					TemplatePath: "templates/services/service/transports/__init__.py.mustache",
					OutputPath:   filepath.Join(serviceDir, "transports", "__init__.py"),
				},
			},
		)
		if ann.HasPagers {
			serviceFiles = append(serviceFiles,
				serviceFilePair{
					service: service,
					file: language.GeneratedFile{
						TemplatePath: "templates/services/service/pagers.py.mustache",
						OutputPath:   filepath.Join(serviceDir, "pagers.py"),
					},
				},
			)
		}
	}

	allFiles := make([]language.GeneratedFile, 0, len(modelFiles)+len(serviceFiles)+len(typeFiles))
	allFiles = append(allFiles, modelFiles...)
	for _, sf := range serviceFiles {
		allFiles = append(allFiles, sf.file)
	}
	for _, tf := range typeFiles {
		allFiles = append(allFiles, tf.file)
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

	for _, tf := range typeFiles {
		if err := generateTypeFile(outdir, tf.fileAnn, provider, tf.file); err != nil {
			return err
		}
	}

	return nil
}

type mustacheProvider struct {
	impl    language.TemplateProvider
	dirname string
}

func (p *mustacheProvider) Get(name string) (string, error) {
	if suffix, ok := strings.CutPrefix(name, "/"); ok {
		return p.impl(suffix + ".mustache")
	}
	return p.impl(path.Join(p.dirname, name) + ".mustache")
}

func generateTypeFile(outDir string, fileAnn *fileAnnotations, provider language.TemplateProvider, gen language.GeneratedFile) error {
	templateContents, err := provider(gen.TemplatePath)
	if err != nil {
		return err
	}
	destination := filepath.Join(outDir, gen.OutputPath)
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	nestedProvider := &mustacheProvider{
		impl:    provider,
		dirname: path.Dir(filepath.ToSlash(gen.TemplatePath)),
	}
	s, err := mustache.RenderPartials(templateContents, nestedProvider, fileAnn)
	if err != nil {
		return err
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[i] = ""
		}
	}
	s = strings.Join(lines, "\n")
	return os.WriteFile(destination, []byte(s), 0o666)
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
