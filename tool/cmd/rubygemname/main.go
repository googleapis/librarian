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

// Command rubygemname validates option ruby_package declarations in googleapis
// proto files and prints the Ruby gem name(s) that must be reserved for an API.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"slices"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/librarian/ruby"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var (
	errAPIAlreadyExists  = errors.New("api already exists in library")
	errWrongAPICount     = errors.New("must provide exactly one API path")
	errNotRubyConfig     = errors.New("config language is not ruby")
	errMissingGoogleapis = errors.New("must specify --googleapis or sources.googleapis in librarian.yaml")
)

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args, os.Stdout); err != nil {
		log.Fatalf("rubygemname: %v", err)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
	cmd := &cli.Command{
		Name:      "rubygemname",
		Usage:     "derive Ruby gem names to reserve for a googleapis API path",
		UsageText: "rubygemname [flags] <api-path>",
		Writer:    stdout,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"librarian-yaml"},
				Value:   config.LibrarianYAML,
				Usage:   "path to google-cloud-ruby librarian.yaml",
			},
			&cli.StringFlag{
				Name:  "googleapis",
				Usage: "path to local googleapis directory (defaults to sources.googleapis in librarian.yaml)",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.Args().Len() != 1 {
				return errWrongAPICount
			}
			return deriveAndPrint(ctx, c.String("config"), c.String("googleapis"), c.Args().First(), stdout)
		},
	}
	return cmd.Run(ctx, args)
}

func deriveAndPrint(ctx context.Context, configPath, googleapisDir, apiPath string, stdout io.Writer) error {
	cfg, err := yaml.Read[config.Config](configPath)
	if err != nil {
		return err
	}
	if cfg.Language != config.LanguageRuby {
		return fmt.Errorf("%w: got %q, want %q", errNotRubyConfig, cfg.Language, config.LanguageRuby)
	}
	if googleapisDir == "" {
		if cfg.Sources == nil || cfg.Sources.Googleapis == nil {
			return errMissingGoogleapis
		}
		srcs, err := librarian.LoadSources(ctx, &config.Sources{Googleapis: cfg.Sources.Googleapis})
		if err != nil {
			return err
		}
		googleapisDir = srcs.Googleapis
	}
	names, err := gemNames(cfg, googleapisDir, apiPath)
	if err != nil {
		return err
	}
	for _, name := range names {
		if _, err := fmt.Fprintln(stdout, name); err != nil {
			return err
		}
	}
	return nil
}

// gemNames returns the gem names to reserve for apiPath, or an error if apiPath
// is already in a library.
func gemNames(cfg *config.Config, googleapisDir, apiPath string) ([]string, error) {
	for _, lib := range cfg.Libraries {
		if slices.ContainsFunc(lib.APIs, func(api *config.API) bool { return api.Path == apiPath }) {
			return nil, fmt.Errorf("%w: %s in library %s", errAPIAlreadyExists, apiPath, lib.Name)
		}
	}
	return ruby.GemNamesToReserve(cfg.Libraries, googleapisDir, apiPath)
}
