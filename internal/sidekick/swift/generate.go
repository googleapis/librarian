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
	"strings"

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

func (c *codec) swiftFilename(basename string) string {
	name := basename + ".swift"
	key := strings.ToLower(basename)
	if c.GeneratedFiles == nil {
		c.GeneratedFiles = map[string]int{key: 0}
	} else {
		count, ok := c.GeneratedFiles[key]
		if !ok {
			c.GeneratedFiles[key] = 0
		} else {
			c.GeneratedFiles[key] = count + 1
			// This can only deal with 1000 conflicts on the same name. If we
			// ever have more, we need to have a serious discussion with the
			// API team that produced that many conflicts.
			name = fmt.Sprintf("%s+%03d.swift", basename, count)
		}

	}
	if c.Module {
		return name
	}
	return filepath.Join("Sources", c.LibraryName, name)
}

func (c *codec) generateMessages(outdir string, model *api.API, provider language.TemplateProvider) error {
	for _, m := range model.Messages {
		output := c.swiftFilename(m.Name)
		template := "templates/common/message_file.swift.mustache"
		if m.ServicePlaceholder {
			output = c.swiftFilename(m.Name + "+Requests")
			template = "templates/common/placeholder_file.swift.mustache"
		}
		generated := language.GeneratedFile{
			TemplatePath: template,
			OutputPath:   output,
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
			OutputPath:   c.swiftFilename(e.Name),
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
			OutputPath:   c.swiftFilename(s.Name),
		}
		if err := language.GenerateService(outdir, s, provider, generated); err != nil {
			return err
		}
	}
	return nil
}

func (c *codec) generateStubs(outdir string, model *api.API, provider language.TemplateProvider) error {
	for _, s := range model.Services {
		for _, stub := range []struct {
			suffix   string
			template string
		}{
			{suffix: "+Stub", template: "templates/common/stub.swift.mustache"},
			{suffix: "+Logging", template: "templates/common/logging.swift.mustache"},
			{suffix: "+Retry", template: "templates/common/retry.swift.mustache"},
		} {
			generated := language.GeneratedFile{
				TemplatePath: stub.template,
				OutputPath:   c.swiftFilename(s.Name + stub.suffix),
			}
			if err := language.GenerateService(outdir, s, provider, generated); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *codec) generateSnippets(outdir string, model *api.API, provider language.TemplateProvider) error {
	for _, s := range model.Services {
		// If two services differ only in case (such as `fooService` and
		// `FooService`), then this might generate clashing filenames in
		// filesystems that are case insensitive.
		//
		// This seems unlikely: the services are always in a flat namespace, and
		// they always use consistent naming conventions. We have only found
		// clashes between messages and services or between messages, not
		// between two services.
		//
		// Furthermore, fixing the problem would require changing the generated
		// code, as the name of the snippet file is referenced in the generated
		// comments. The effort to fix that does not seem worthwhile given how
		// unlikely is the problem.
		//
		// If I (coryan@) am wrong, we can fix the generator at that time.
		generated := language.GeneratedFile{
			TemplatePath: "templates/common/client_snippet.swift.mustache",
			OutputPath:   filepath.Join("Snippets", s.Name+"Quickstart.swift"),
		}
		if err := language.GenerateService(outdir, s, provider, generated); err != nil {
			return err
		}
		for _, m := range s.Methods {
			if !isGeneratedMethod(m) || m.IsLroPoller {
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
		OutputPath:   c.swiftFilename("Clients"),
	}
	return language.GenerateFromModel(outdir, model, provider, []language.GeneratedFile{generated})
}
