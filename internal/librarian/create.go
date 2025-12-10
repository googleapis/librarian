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
	"log/slog"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/internal/rust"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var (
	errRunningCreateOnNonRustLanguage = errors.New("library creation is only supported for Rust libraries")
	errOutputFlagRequired             = errors.New("output flag is required when default.output is not set in librarian.yaml")
	errServiceConfigAndSpecRequired   = errors.New("both service-config and specification-source flags are required for creating a new library")
	errMissingNameFlag                = errors.New("name flag is required to create a new library")
)

func createCommand() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "create a new client library",
		UsageText: "librarian create --name <name> --specification-source <path> --service-config <path>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "name",
				Usage: "library name",
			},
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
			name := c.String("name")
			specSource := c.String("specification-source")
			serviceConfig := c.String("service-config")
			output := c.String("output")
			specFormat := c.String("specification-format")
			if name == "" {
				return errMissingNameFlag
			}
			return runCreate(ctx, name, specSource, serviceConfig, output, specFormat)
		},
	}
}

func runCreate(ctx context.Context, name, specSource, serviceConfig, output, specFormat string) error {
	cfg, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		return err
	}

	if cfg.Language != "rust" {
		return errRunningCreateOnNonRustLanguage
	}

	if output == "" {
		if cfg.Default == nil || cfg.Default.Output == "" {
			return errOutputFlagRequired
		}
		output = rust.DefaultOutput(specSource, cfg.Default.Output)
	}
	libraryExists := false
	for _, lib := range cfg.Libraries {
		if lib.Name == name {

			libraryExists = true
			break
		}
	}

	if libraryExists {
		return runGenerate(ctx, false, name)
	}

	if (serviceConfig == "") || (specSource == "") {
		return errServiceConfigAndSpecRequired
	}

	// TODO: port over sidekick rustGenerate logic to create a new librarian
	slog.Info("Creating new Rust library (not implemented yet)")
	return nil
}
