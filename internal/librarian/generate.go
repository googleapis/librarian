// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/language"
	"github.com/urfave/cli/v3"
)

var (
	errMissingLibraryOrAllFlag = errors.New("must specify library name or use --all flag")
	errBothLibraryAndAllFlag   = errors.New("cannot specify both library name and --all flag")
)

func generateCommand() *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Usage:     "generate a client library",
		UsageText: "librarian generate [library] [--all]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "generate all libraries",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runGenerate(ctx, cmd)
		},
	}
}

func runGenerate(ctx context.Context, cmd *cli.Command) error {
	all := cmd.Bool("all")
	libraryName := cmd.Args().First()

	if !all && libraryName == "" {
		return errMissingLibraryOrAllFlag
	}
	if all && libraryName != "" {
		return errBothLibraryAndAllFlag
	}

	cfg, err := config.Read(librarianConfigPath)
	if err != nil {
		return err
	}

	if all {
		for _, lib := range cfg.Libraries {
			if err := language.Generate(ctx, cfg.Language, lib); err != nil {
				return err
			}
		}
	} else {
		var library *config.Library
		for _, lib := range cfg.Libraries {
			if lib.Name == libraryName {
				library = lib
				break
			}
		}
		if library == nil {
			return fmt.Errorf("library %q not found", libraryName)
		}
		if err := language.Generate(ctx, cfg.Language, library); err != nil {
			return err
		}
	}
	return cfg.Write(librarianConfigPath)
}
