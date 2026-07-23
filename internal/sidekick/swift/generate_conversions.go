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

package swift

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

// GenerateConversions generates the user-facing clean types and conversion mappings.
func GenerateConversions(ctx context.Context, model *api.API, outdir string, cfg *parser.ModelConfig, swiftCfg *config.SwiftPackage) error {
	codec, err := newCodec(model, cfg, swiftCfg, outdir)
	if err != nil {
		return err
	}
	if codec.ModulePath == "" {
		return fmt.Errorf("module-path must be configured for generating conversions")
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
	if err := codec.generateEnumConversions(outdir, model, provider); err != nil {
		return err
	}
	if err := codec.generateMessageConversions(outdir, model, provider); err != nil {
		return err
	}
	return nil
}

func (c *codec) generateEnumConversions(outdir string, model *api.API, provider language.TemplateProvider) error {
	for _, e := range model.Enums {
		name := c.enumFileName(e)
		output := filepath.Join("Convert", name+"+Convert.swift")
		if !c.Module {
			output = filepath.Join("Sources", c.LibraryName, "Convert", name+"+Convert.swift")
		}
		generated := language.GeneratedFile{
			TemplatePath: "templates/convert/convert_enum_file.swift.mustache",
			OutputPath:   output,
		}
		if err := language.GenerateEnum(outdir, e, provider, generated); err != nil {
			return err
		}
	}
	return nil
}

func (c *codec) generateMessageConversions(outdir string, model *api.API, provider language.TemplateProvider) error {
	for _, m := range model.Messages {
		if m.IsMap {
			continue
		}
		if m.ServicePlaceholder {
			continue
		}
		name := c.messageFileName(m)
		output := filepath.Join("Convert", name+"+Convert.swift")
		if !c.Module {
			output = filepath.Join("Sources", c.LibraryName, "Convert", name+"+Convert.swift")
		}
		generated := language.GeneratedFile{
			TemplatePath: "templates/convert/convert_message_file.swift.mustache",
			OutputPath:   output,
		}
		if err := language.GenerateMessage(outdir, m, provider, generated); err != nil {
			return err
		}
	}
	return nil
}
