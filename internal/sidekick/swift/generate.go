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
	"fmt"
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
	if err := codec.generateEnums(outdir, model, provider); err != nil {
		return err
	}
	if err := codec.generateServices(outdir, model, provider); err != nil {
		return err
	}
	if err := codec.generateClients(outdir, model, provider); err != nil {
		return err
	}
	if err := codec.generateStubs(outdir, model, provider); err != nil {
		return err
	}
	if codec.Module {
		// Modules only get the top-level messages, enums, and services generated.
		return nil
	}
	if err := codec.generateSnippets(outdir, model, provider); err != nil {
		return err
	}
	generatedFiles := language.WalkTemplatesDir(templates, "templates/package")
	return language.GenerateFromModel(outdir, model, provider, generatedFiles)
}

func (c *codec) outputPath(name string) string {
	if c.Module {
		return name
	}
	return filepath.Join("Sources", c.PackageName, name)
}

func (c *codec) generateMessages(outdir string, model *api.API, provider language.TemplateProvider) error {
	for _, m := range model.Messages {
		generated := language.GeneratedFile{
			TemplatePath: "templates/common/message_file.swift.mustache",
			OutputPath:   c.outputPath(m.Name + ".swift"),
		}
		if err := language.GenerateMessage(outdir, m, provider, generated); err != nil {
			return err
		}
	}
	return nil
}

func (c *codec) generateEnums(outdir string, model *api.API, provider language.TemplateProvider) error {
	for _, e := range model.Enums {
		generated := language.GeneratedFile{
			TemplatePath: "templates/common/enum_file.swift.mustache",
			OutputPath:   c.outputPath(e.Name + ".swift"),
		}
		if err := language.GenerateEnum(outdir, e, provider, generated); err != nil {
			return err
		}
	}
	return nil
}

func (c *codec) generateServices(outdir string, model *api.API, provider language.TemplateProvider) error {
	for _, s := range model.Services {
		generated := language.GeneratedFile{
			TemplatePath: "templates/common/service.swift.mustache",
			OutputPath:   c.outputPath(s.Name + ".swift"),
		}
		if err := language.GenerateService(outdir, s, provider, generated); err != nil {
			return err
		}
	}
	return nil
}

func (c *codec) generateStubs(outdir string, model *api.API, provider language.TemplateProvider) error {
	for _, s := range model.Services {
		generated := language.GeneratedFile{
			TemplatePath: "templates/common/stub.swift.mustache",
			OutputPath:   c.outputPath(s.Name + "Stub.swift"),
		}
		if err := language.GenerateService(outdir, s, provider, generated); err != nil {
			return err
		}
	}
	return nil
}

func (c *codec) generateSnippets(outdir string, model *api.API, provider language.TemplateProvider) error {
	for _, s := range model.Services {
		generated := language.GeneratedFile{
			TemplatePath: "templates/common/client_snippet.swift.mustache",
			OutputPath:   filepath.Join("Snippets", s.Name+"Quickstart.swift"),
		}
		if err := language.GenerateService(outdir, s, provider, generated); err != nil {
			return err
		}
		for _, m := range s.Methods {
			if !isGeneratedMethod(m) {
				continue
			}
			mGenerated := language.GeneratedFile{
				TemplatePath: "templates/common/method_snippet.swift.mustache",
				OutputPath:   filepath.Join("Snippets", fmt.Sprintf("%s_%s.swift", s.Name, m.Name)),
			}
			if err := language.GenerateMethod(outdir, m, provider, mGenerated); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *codec) generateClients(outdir string, model *api.API, provider language.TemplateProvider) error {
	if len(model.Services) == 0 {
		return nil
	}
	generated := language.GeneratedFile{
		TemplatePath: "templates/common/clients.swift.mustache",
		OutputPath:   c.outputPath("Clients.swift"),
	}
	return language.GenerateFromModel(outdir, model, provider, []language.GeneratedFile{generated})
}
