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

package rust

import (
	"context"
	"embed"
	"path/filepath"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

//go:embed all:templates
var templates embed.FS

// Generate generates Rust code from the model.
func Generate(ctx context.Context, model *api.API, outdir string, cfg *parser.ModelConfig) error {
	c, err := newCodec(cfg.SpecificationFormat, cfg.Codec)
	if err != nil {
		return err
	}
	annotations, err := annotateModel(model, c)
	if err != nil {
		return err
	}
	provider := templatesProvider()
	generatedFiles := c.generatedFiles(annotations.HasServices())
	return language.GenerateFromModel(outdir, model, provider, generatedFiles)
}

// GenerateStorage generates Rust code for the storage service.
func GenerateStorage(ctx context.Context, outdir string, storageModel *api.API, storageConfig *parser.ModelConfig, controlModel *api.API, controlConfig *parser.ModelConfig) error {
	storageCodec, err := newCodec(storageConfig.SpecificationFormat, storageConfig.Codec)
	if err != nil {
		return err
	}
	if _, err := annotateModel(storageModel, storageCodec); err != nil {
		return err
	}
	controlCodec, err := newCodec(controlConfig.SpecificationFormat, controlConfig.Codec)
	if err != nil {
		return err
	}
	if _, err := annotateModel(controlModel, controlCodec); err != nil {
		return err
	}

	model := &api.API{
		Codec: &storageAnnotations{
			Storage: storageModel,
			Control: controlModel,
		},
	}
	provider := templatesProvider()
	generatedFiles := language.WalkTemplatesDir(templates, "templates/storage")
	return language.GenerateFromModel(outdir, model, provider, generatedFiles)
}

type storageAnnotations struct {
	Storage *api.API
	Control *api.API
}

func templatesProvider() language.TemplateProvider {
	return func(name string) (string, error) {
		contents, err := templates.ReadFile(filepath.ToSlash(name))
		if err != nil {
			return "", err
		}
		return string(contents), nil
	}
}

func (c *codec) generatedFiles(hasServices bool) []language.GeneratedFile {
	if c.templateOverride != "" {
		return language.WalkTemplatesDir(templates, c.templateOverride)
	}
	var root string
	switch {
	case !hasServices:
		root = "templates/nosvc"
	default:
		root = "templates/crate"
	}
	return language.WalkTemplatesDir(templates, root)
}
