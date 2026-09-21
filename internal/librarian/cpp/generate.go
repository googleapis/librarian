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

// Package cpp provides functionality for generating C++ client libraries.
package cpp

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	sidekickcpp "github.com/googleapis/librarian/internal/sidekick/cpp"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
)

var (
	// ErrMissingCppConfig indicates that a library is missing C++ configuration.
	ErrMissingCppConfig = errors.New("missing C++ configuration")
	// ErrNoAPIs indicates that a library has no configured APIs.
	ErrNoAPIs = errors.New("no configured APIs")
	// ErrMissingGoogleapisSource indicates that the googleapis source is missing.
	ErrMissingGoogleapisSource = errors.New("missing googleapis source")
)

// Generate generates a C++ client library.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library, src *sources.Sources) error {
	if library == nil || library.Cpp == nil {
		libName := "unknown"
		if library != nil {
			libName = library.Name
		}
		return fmt.Errorf("%w for library %s", ErrMissingCppConfig, libName)
	}
	if len(library.APIs) == 0 {
		return fmt.Errorf("%w for library %s", ErrNoAPIs, library.Name)
	}
	if src == nil || src.Googleapis == "" {
		return fmt.Errorf("%w for library %s", ErrMissingGoogleapisSource, library.Name)
	}
	var pc *config.Protoc
	if cfg != nil && cfg.Tools != nil {
		pc = cfg.Tools.Protoc
	}
	modelConfig, err := libraryToModelConfig(library, library.APIs[0], src, pc)
	if err != nil {
		return err
	}
	model, err := parser.CreateModel(modelConfig)
	if err != nil {
		return err
	}
	return sidekickcpp.Generate(ctx, model, library.Output, library.Cpp)
}

// DefaultOutput derives the default output directory from the API path.
func DefaultOutput(api, defaultOut string) string {
	if defaultOut != "" {
		return filepath.Join(defaultOut, api)
	}
	return api
}

func libraryToModelConfig(library *config.Library, apiCfg *config.API, src *sources.Sources, pc *config.Protoc) (*parser.ModelConfig, error) {
	sourceConfig := sources.NewSourceConfig(src, library.Roots)
	serviceConfigPath := ""
	if library.Cpp != nil {
		serviceConfigPath = library.Cpp.OverrideServiceConfigYAMLName
	}
	if serviceConfigPath == "" {
		svcConfig, err := serviceconfig.Find(src.Googleapis, apiCfg.Path, config.LanguageCpp)
		if err != nil {
			return nil, err
		}
		serviceConfigPath = svcConfig.ServiceConfig
	}
	specFormat := config.SpecProtobuf
	if library.SpecificationFormat != "" {
		specFormat = library.SpecificationFormat
	}

	return &parser.ModelConfig{
		Language:            config.LanguageCpp,
		SpecificationFormat: specFormat,
		ServiceConfig:       serviceConfigPath,
		SpecificationSource: apiCfg.Path,
		Source:              sourceConfig,
		Protoc:              pc,
	}, nil
}
