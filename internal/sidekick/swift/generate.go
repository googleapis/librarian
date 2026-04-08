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

// Package swift provides a code generator for Swift.
package swift

import (
	"context"
	"embed"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

//go:embed all:templates
var templates embed.FS

// Generate generates code from the model.
func Generate(ctx context.Context, model *api.API, outdir string, cfg *parser.ModelConfig, swiftCfg *config.SwiftPackage) error {
	codec, err := newCodec(model, cfg, swiftCfg, outdir)
	if err != nil {
		return err
	}
	if err := codec.annotateModel(); err != nil {
		return err
	}
	provider := func(name string) (string, error) {
		contents, err := templates.ReadFile(name)
		if err != nil {
			return "", err
		}
		return string(contents), nil
	}
	if err := codec.generateMessages(outdir, model, provider); err != nil {
		return err
	}
	if err := codec.generateServices(outdir, model, provider); err != nil {
		return err
	}
	generatedFiles := language.WalkTemplatesDir(templates, "templates/package")
	return language.GenerateFromModel(outdir, model, provider, generatedFiles)
}

func (c *codec) generateMessages(outdir string, model *api.API, provider language.TemplateProvider) error {
	for _, m := range model.Messages {
		generated := language.GeneratedFile{
			TemplatePath: "templates/common/message.swift.mustache",
			OutputPath:   filepath.Join("Sources", c.PackageName, m.Name+".swift"),
		}
		if err := language.GenerateMessage(outdir, m, provider, generated); err != nil {
			return err
		}
	}
	return nil
}

func (c *codec) generateServices(outdir string, model *api.API, provider language.TemplateProvider) error {
	for _, s := range model.Services {
		generated := language.GeneratedFile{
			TemplatePath: "templates/common/service.swift.mustache",
			OutputPath:   filepath.Join("Sources", c.PackageName, s.Name+".swift"),
		}
		if err := language.GenerateService(outdir, s, provider, generated); err != nil {
			return err
		}
	}
	return nil
}
