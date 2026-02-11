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
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/dart"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/googleapis/librarian/internal/librarian/python"
	"github.com/googleapis/librarian/internal/librarian/rust"
	"github.com/urfave/cli/v3"
)

var (
	errLibraryAlreadyExists = errors.New("library already exists in config")
	errMissingAPI           = errors.New("must provide at least one API")
)

func addCommand() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "add a new client library to librarian.yaml",
		UsageText: "librarian add <apis...> [flags]",
		Action: func(ctx context.Context, c *cli.Command) error {
			apis := c.Args().Slice()
			if len(apis) == 0 {
				return errMissingAPI
			}
			cfg, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				return err
			}
			return runAdd(ctx, cfg, apis...)
		},
	}
}

func runAdd(ctx context.Context, cfg *config.Config, apis ...string) error {
	cfg, err := addLibrary(cfg, apis...)
	if err != nil {
		return err
	}
	if err := RunTidyOnConfig(ctx, cfg); err != nil {
		return err
	}
	return nil
}

// deriveLibraryName derives a library name from an API path.
// The derivation is language-specific.
func deriveLibraryName(language, api string) string {
	switch language {
	case languageDart:
		return dart.DefaultLibraryName(api)
	case languageFake:
		return fakeDefaultLibraryName(api)
	case languagePython:
		return python.DefaultLibraryName(api)
	case languageRust:
		return rust.DefaultLibraryName(api)
	default:
		return strings.ReplaceAll(api, "/", "-")
	}
}

func addLibrary(cfg *config.Config, apis ...string) (*config.Config, error) {
	name := deriveLibraryName(cfg.Language, apis[0])
	exists := slices.ContainsFunc(cfg.Libraries, func(lib *config.Library) bool {
		return lib.Name == name
	})
	if exists {
		return nil, fmt.Errorf("%w: %s", errLibraryAlreadyExists, name)
	}

	lib := &config.Library{
		Name:          name,
		CopyrightYear: strconv.Itoa(time.Now().Year()),
	}
	for _, a := range apis {
		lib.APIs = append(lib.APIs, &config.API{
			Path: a,
		})
	}
	cfg.Libraries = append(cfg.Libraries, lib)
	sort.Slice(cfg.Libraries, func(i, j int) bool {
		return cfg.Libraries[i].Name < cfg.Libraries[j].Name
	})
	return cfg, nil
}
