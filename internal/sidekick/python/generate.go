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
	"path"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

//go:embed all:templates
var templates embed.FS

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
			return "", err
		}
		return string(contents), nil
	}

	files := []language.GeneratedFile{
		{
			TemplatePath: "templates/gapic_version.py.mustache",
			OutputPath:   "gapic_version.py",
		},
		{
			TemplatePath: "templates/py.typed.mustache",
			OutputPath:   "py.typed",
		},
	}

	return language.GenerateFromModel(outdir, model, provider, files)
}
