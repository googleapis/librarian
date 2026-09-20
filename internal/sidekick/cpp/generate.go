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
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

var (
	// ErrEscapesOutputDirectory indicates that a generated file output path escapes the output directory.
	ErrEscapesOutputDirectory = errors.New("output path escapes output directory")

	// ErrDuplicateOutputPath indicates that multiple files would be written to the same output path.
	ErrDuplicateOutputPath = errors.New("duplicate output path")
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
		if err != nil {
			return "", err
		}
		return string(contents), nil
	}

	var allGeneratedFiles []language.GeneratedFile
	cmakeFiles := language.WalkTemplatesDir(templates, "templates/cmake")
	allGeneratedFiles = append(allGeneratedFiles, cmakeFiles...)

	generateGrpc := true
	generateRest := false
	generateRoundRobin := false
	var productPath string
	var forwardingProductPath string
	if libCfg != nil {
		if libCfg.GenerateGrpcTransport != nil {
			generateGrpc = *libCfg.GenerateGrpcTransport
		}
		generateRest = libCfg.GenerateRestTransport
		generateRoundRobin = libCfg.GenerateRoundRobinDecorator
		productPath = libCfg.ProductPath
		forwardingProductPath = libCfg.ForwardingProductPath
	}

	type serviceFileBatch struct {
		service *api.Service
		files   []language.GeneratedFile
	}
	var serviceBatches []serviceFileBatch

	for _, svc := range model.Services {
		if libCfg != nil && slices.Contains(libCfg.OmittedServices, svc.Name) {
			continue
		}
		var svcFiles []language.GeneratedFile
		svcFiles = append(svcFiles, CommonGeneratedFiles(productPath, svc.Name)...)
		if generateGrpc {
			svcFiles = append(svcFiles, GrpcGeneratedFiles(productPath, svc.Name)...)
		}
		if generateRest {
			svcFiles = append(svcFiles, RestGeneratedFiles(productPath, svc.Name)...)
		}
		if generateRoundRobin {
			svcFiles = append(svcFiles, RoundRobinGeneratedFiles(productPath, svc.Name)...)
		}
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
			return fmt.Errorf("%w: %q", ErrEscapesOutputDirectory, gen.OutputPath)
		}
		targetPath := filepath.Join(absOut, gen.OutputPath)
		rel, err := filepath.Rel(absOut, targetPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("%w: %q", ErrEscapesOutputDirectory, gen.OutputPath)
		}
		if seen[rel] {
			return fmt.Errorf("%w: %q", ErrDuplicateOutputPath, gen.OutputPath)
		}
		seen[rel] = true
	}
	return nil
}
