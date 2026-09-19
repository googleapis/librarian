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
	"slices"
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
		if err == nil {
			return string(contents), nil
		}
		base := filepath.Base(name)
		if partial, err2 := templates.ReadFile(filepath.Join("templates", "partials", base)); err2 == nil {
			return string(partial), nil
		}
		return "", err
	}

	var allGeneratedFiles []language.GeneratedFile
	cmakeFiles := language.WalkTemplatesDir(templates, "templates/cmake")
	allGeneratedFiles = append(allGeneratedFiles, cmakeFiles...)

	generateGrpc := true
	if libCfg != nil && libCfg.GenerateGrpcTransport != nil {
		generateGrpc = *libCfg.GenerateGrpcTransport
	}

	var productPath string
	var forwardingProductPath string
	if libCfg != nil {
		productPath = libCfg.ProductPath
		forwardingProductPath = libCfg.ForwardingProductPath
	}

	type serviceFileBatch struct {
		service *api.Service
		files   []language.GeneratedFile
	}
	var serviceBatches []serviceFileBatch

	if generateGrpc {
		for _, svc := range model.Services {
			if libCfg != nil && slices.Contains(libCfg.OmittedServices, svc.Name) {
				continue
			}
			svcFiles := ServiceGeneratedFiles(productPath, svc.Name)
			allGeneratedFiles = append(allGeneratedFiles, svcFiles...)
			serviceBatches = append(serviceBatches, serviceFileBatch{
				service: svc,
				files:   svcFiles,
			})

			if forwardingProductPath != "" {
				fwdFiles := ForwardingGeneratedFiles(forwardingProductPath, svc.Name)
				allGeneratedFiles = append(allGeneratedFiles, fwdFiles...)
				serviceBatches = append(serviceBatches, serviceFileBatch{
					service: svc,
					files:   fwdFiles,
				})
			}
		}
	}

	if err := validateOutputContainment(outdir, allGeneratedFiles); err != nil {
		return err
	}

	if err := language.GenerateFromModel(outdir, model, provider, cmakeFiles); err != nil {
		return err
	}

	for _, batch := range serviceBatches {
		for _, gen := range batch.files {
			if err := language.GenerateService(outdir, batch.service, provider, gen); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateOutputContainment(outdir string, files []language.GeneratedFile) error {
	absOut, err := filepath.Abs(outdir)
	if err != nil {
		return fmt.Errorf("resolving outdir %q: %w", outdir, err)
	}
	seen := make(map[string]bool, len(files))
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
		if seen[rel] {
			return fmt.Errorf("duplicate output path %q", gen.OutputPath)
		}
		seen[rel] = true
	}
	return nil
}
