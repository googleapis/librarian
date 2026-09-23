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

package python

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	sidekickpython "github.com/googleapis/librarian/internal/sidekick/python"
	"github.com/googleapis/librarian/internal/sources"
)

func isSidekickGenerator(library *config.Library) bool {
	return library != nil && library.Python != nil && strings.EqualFold(library.Python.Generator, generatorSidekick)
}

func generateSidekick(ctx context.Context, cfg *config.Config, library *config.Library, srcs *sources.Sources) error {
	if library.Output == "" {
		return ErrEmptyOutput
	}
	outdir, err := filepath.Abs(library.Output)
	if err != nil {
		return fmt.Errorf("failed to resolve output directory path: %w", err)
	}

	var model *api.API
	if len(library.APIs) > 0 {
		var pc *config.Protoc
		if cfg != nil && cfg.Tools != nil {
			pc = cfg.Tools.Protoc
		}
		modelConfig, err := toModelConfig(library, library.APIs[0], srcs, pc)
		if err != nil {
			return err
		}
		model, err = parser.CreateModel(modelConfig)
		if err != nil {
			return err
		}
	} else {
		model = &api.API{Name: library.Name}
	}

	if err := os.MkdirAll(outdir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	libCopy := *library
	libCopy.Roots = slices.Clone(library.Roots)
	if srcs != nil && srcs.Googleapis != "" && !slices.Contains(libCopy.Roots, srcs.Googleapis) {
		libCopy.Roots = append(libCopy.Roots, srcs.Googleapis)
	}
	if library.Python != nil {
		pyCopy := *library.Python
		pyCopy.OptArgsByAPI = make(map[string][]string, len(library.Python.OptArgsByAPI))
		for k, v := range library.Python.OptArgsByAPI {
			pyCopy.OptArgsByAPI[k] = slices.Clone(v)
		}
		if srcs != nil {
			root := srcs.Googleapis
			if isPreview(library.Output) {
				root = filepath.Join(root, "preview")
			}
			for _, apiCfg := range library.APIs {
				apiRoot := root
				if apiCfg.Path == repometadata.ShowcasePath {
					apiRoot = srcs.Showcase
				}
				svcConfig, err := serviceconfig.Find(apiRoot, apiCfg.Path, config.LanguagePython)
				if err == nil && svcConfig != nil {
					if svcConfig.HasRESTNumericEnums(config.LanguagePython) {
						if !slices.Contains(pyCopy.OptArgsByAPI[apiCfg.Path], "rest-numeric-enums") {
							pyCopy.OptArgsByAPI[apiCfg.Path] = append(pyCopy.OptArgsByAPI[apiCfg.Path], "rest-numeric-enums")
						}
					}
					transport := string(svcConfig.Transport(config.LanguagePython))
					if transport != "" {
						transportOpt := "transport=" + transport
						if !slices.Contains(pyCopy.OptArgsByAPI[apiCfg.Path], transportOpt) {
							pyCopy.OptArgsByAPI[apiCfg.Path] = append(pyCopy.OptArgsByAPI[apiCfg.Path], transportOpt)
						}
					}
				}
			}
		}
		libCopy.Python = &pyCopy
	}

	if err := sidekickpython.Generate(ctx, model, outdir, &libCopy); err != nil {
		return err
	}
	return Format(ctx, library)
}

func toModelConfig(library *config.Library, apiCfg *config.API, srcs *sources.Sources, pc *config.Protoc) (*parser.ModelConfig, error) {
	if library == nil {
		return nil, ErrNilLibrary
	}
	if apiCfg == nil {
		return nil, ErrNilAPIConfig
	}
	if srcs == nil {
		return nil, ErrNilSources
	}
	sourceConfig := sources.NewSourceConfig(srcs, library.Roots)
	root := srcs.Googleapis
	if isPreview(library.Output) {
		root = filepath.Join(root, "preview")
	}
	if apiCfg.Path == repometadata.ShowcasePath {
		root = srcs.Showcase
	}
	svcConfig, err := serviceconfig.Find(root, apiCfg.Path, config.LanguagePython)
	if err != nil {
		return nil, err
	}
	specFormat := config.SpecProtobuf
	if library.SpecificationFormat != "" {
		specFormat = library.SpecificationFormat
	}
	return &parser.ModelConfig{
		Language:            config.LanguagePython,
		SpecificationFormat: specFormat,
		SpecificationSource: apiCfg.Path,
		Source:              sourceConfig,
		Protoc:              pc,
		ServiceConfig:       svcConfig.ServiceConfig,
	}, nil
}
