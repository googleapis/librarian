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
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var (
	errLibraryAlreadyExists = errors.New("library requested to add already exists in config")
	errMissingLibraryName   = errors.New("must provide library name as argument to add a new library")
	errNoYaml               = errors.New("unable to read librarian.yaml")
)

func addCommand() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "add a new client library to librarian.yaml",
		UsageText: "librarian add <library> [apis...] [flags]",
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			name := args.First()
			if name == "" {
				return errMissingLibraryName
			}
			var channels []string
			if len(args.Slice()) > 1 {
				channels = args.Slice()[1:]
			}
			return runAdd(ctx, name, channels...)
		},
	}
}

func runAdd(ctx context.Context, name string, channel ...string) error {
	cfg, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		return fmt.Errorf("%w: %v", errNoYaml, err)
	}
	// check for existing libraries, if it exists return an error
	exists := slices.ContainsFunc(cfg.Libraries, func(lib *config.Library) bool {
		return lib.Name == name
	})
	if exists {
		return fmt.Errorf("%w: %s", errLibraryAlreadyExists, name)
	}

	if err := addLibraryToLibrarianConfig(cfg, name, channel...); err != nil {
		return err
	}
	if err := RunTidyOnConfig(ctx, cfg); err != nil {
		return err
	}
	return nil
}

func addLibraryToLibrarianConfig(cfg *config.Config, name string, channel ...string) error {
	lib := &config.Library{
		Name:          name,
		CopyrightYear: strconv.Itoa(time.Now().Year()),
	}

	for _, c := range channel {
		lib.Channels = append(lib.Channels, &config.Channel{
			Path: c,
		})
	}
	cfg.Libraries = append(cfg.Libraries, lib)
	sort.Slice(cfg.Libraries, func(i, j int) bool {
		return cfg.Libraries[i].Name < cfg.Libraries[j].Name
	})
	return yaml.Write(librarianConfigPath, cfg)
}
