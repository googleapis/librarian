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

package librarian

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/internal/rust"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var (
	errUnsupportedLanguage = errors.New("library creation is not supported for the specified language")
	errOutputFlagRequired  = errors.New("output flag is required when default.output is not set in librarian.yaml")
	errMissingLibraryName  = errors.New("must provide library name as argument to create a new library")
	errNoYaml              = errors.New("unable to read librarian.yaml")
)

func createCommand() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "create a new client library",
		UsageText: "librarian create [library] --specification-source [path] --service-config [path]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "specification-source",
				Usage: "path to the specification source (e.g., google/cloud/secretmanager/v1)",
			},
			&cli.StringFlag{
				Name:  "service-config",
				Usage: "path to the service config",
			},
			&cli.StringFlag{
				Name:  "output",
				Usage: "output directory (optional, will be derived if not provided)",
			},
			&cli.StringFlag{
				Name:  "specification-format",
				Usage: "specification format (e.g., protobuf, discovery)",
				Value: "protobuf",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			libraryName := c.Args().First()
			if libraryName == "" {
				return errMissingLibraryName
			}
			specSource := c.String("specification-source")
			serviceConfig := c.String("service-config")
			output := c.String("output")
			specFormat := c.String("specification-format")
			return runCreate(ctx, libraryName, specSource, serviceConfig, output, specFormat)
		},
	}
}

func runCreate(ctx context.Context, name, specSource, serviceConfig, output, specFormat string) error {
	return create(ctx, name, specSource, serviceConfig, output, specFormat, &Generate{}, &rust.RustCreate{})
}

func create(ctx context.Context, libraryName, specSource, serviceConfig, output, specFormat string, gen Generator, rustCreator rust.RustCreator) error {
	cfg, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		return fmt.Errorf("%w: %v", errNoYaml, err)
	}
	// check for existing libraries, if it exists just run generate
	for _, lib := range cfg.Libraries {
		if lib.Name == libraryName {
			return gen.Run(ctx, false, libraryName)
		}
	}
	specSource = deriveSpecSource(specSource, serviceConfig, cfg.Language)
	if output, err = deriveOutput(output, cfg, libraryName, specSource, cfg.Language); err != nil {
		return err
	}
	if err := addLibraryToLibrarianConfig(cfg, libraryName, output, specSource, serviceConfig, specFormat); err != nil {
		return err
	}
	switch cfg.Language {
	case "rust":
		if err := rustCreator.PrepareCargoWorkspace(ctx, output); err != nil {
			return err
		}
		if err := gen.Run(ctx, false, libraryName); err != nil {
			return err
		}
		return rustCreator.FormatAndValidateLibrary(ctx, output)
	default:
		return errUnsupportedLanguage
	}
}

func deriveSpecSource(specSource string, serviceConfig string, language string) string {
	switch language {
	case "rust":
		if specSource == "" && serviceConfig != "" {
			return path.Dir(serviceConfig)
		}
	}
	return specSource
}

func deriveOutput(output string, cfg *config.Config, libraryName string, specSource string, language string) (string, error) {
	if output == "" && (cfg.Default == nil || cfg.Default.Output == "") {
		return "", errOutputFlagRequired
	}
	switch language {
	case "rust":
		if output == "" {
			if cfg.Default == nil || cfg.Default.Output == "" {
				return "", errOutputFlagRequired
			}
			if specSource != "" {
				return defaultOutput(language, specSource, cfg.Default.Output), nil
			}
			libOutputDir := strings.ReplaceAll(libraryName, "-", "/")
			return defaultOutput(language, libOutputDir, cfg.Default.Output), nil
		}
	default:
		return defaultOutput(language, specSource, cfg.Default.Output), nil
	}

	return output, nil
}

func addLibraryToLibrarianConfig(rootConfig *config.Config, name, output, specificationSource, serviceConfig, specificationFormat string) error {
	lib := &config.Library{
		Name:                name,
		Output:              output,
		Version:             "0.1.0",
		SpecificationFormat: specificationFormat,
		CopyrightYear:       strconv.Itoa(time.Now().Year()),
	}
	if serviceConfig != "" || specificationSource != "" {
		lib.Channels = []*config.Channel{
			{
				Path:          specificationSource,
				ServiceConfig: serviceConfig,
			},
		}
	}
	rootConfig.Libraries = append(rootConfig.Libraries, lib)
	data, err := yaml.Marshal(rootConfig)
	if err != nil {
		return fmt.Errorf("error marshaling librarian config: %w", err)
	}

	if err := os.WriteFile(librarianConfigPath, data, 0o644); err != nil {
		return fmt.Errorf("error writing librarian.yaml: %w", err)
	}
	return nil
}
