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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

// GenerateConversions generates the user-facing clean types and conversion mappings.
func GenerateConversions(ctx context.Context, model *api.API, outdir string, library *config.Library, module *config.SwiftModule) error {
	codec, err := newCodec(model, library, module, outdir)
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

	var messages []*api.Message
	for _, m := range model.Messages {
		if m.Parent == nil && !m.IsMap && !m.ServicePlaceholder {
			messages = append(messages, m)
		}
	}

	var enums []*api.Enum
	for _, e := range model.Enums {
		if e.Parent == nil {
			enums = append(enums, e)
		}
	}

	if err := codec.generateEnumConversions(outdir, enums, provider); err != nil {
		return err
	}
	if err := codec.generateMessageConversions(outdir, messages, provider); err != nil {
		return err
	}
	return nil
}

func (c *codec) generateEnumConversions(outdir string, enums []*api.Enum, provider language.TemplateProvider) error {
	for _, e := range enums {
		name := c.enumFileName(e)
		generated := language.GeneratedFile{
			TemplatePath: "templates/convert/convert_enum_file.swift.mustache",
			OutputPath:   conversionFileName(name),
		}
		if err := language.GenerateEnum(outdir, e, provider, generated); err != nil {
			return err
		}
	}
	return nil
}

func (c *codec) generateMessageConversions(outdir string, messages []*api.Message, provider language.TemplateProvider) error {
	for _, m := range messages {
		name := c.messageFileName(m)
		generated := language.GeneratedFile{
			TemplatePath: "templates/convert/convert_message_file.swift.mustache",
			OutputPath:   conversionFileName(name),
		}
		if err := language.GenerateMessage(outdir, m, provider, generated); err != nil {
			return err
		}
	}
	return nil
}

func conversionFileName(typeName string) string {
	return typeName + "+Convert.swift"
}

