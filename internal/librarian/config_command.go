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
	"io/fs"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

// configCommand returns the CLI command for reading and writing librarian configuration.
func configCommand() *cli.Command {
	return &cli.Command{
		Name:      "config",
		Usage:     "read and write librarian.yaml configuration",
		UsageText: "librarian config [get|set] [path] [value]",
		Commands: []*cli.Command{
			{
				Name:      "get",
				Usage:     "get a configuration value",
				UsageText: "librarian config get [path]",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					path := cmd.Args().First()
					if path == "" {
						return fmt.Errorf("path is required")
					}

					cfg, err := yaml.Read[config.Config](config.LibrarianYAML)
					if err != nil {
						return err
					}

					val, err := getConfigValue(cfg, path)
					if err != nil {
						return err
					}

					fmt.Println(val)
					return nil
				},
			},
			{
				Name:      "set",
				Usage:     "set a configuration value",
				UsageText: "librarian config set [path] [value]",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					path := cmd.Args().Get(0)
					value := cmd.Args().Get(1)
					if path == "" {
						return fmt.Errorf("path is required")
					}
					if value == "" {
						return fmt.Errorf("value is required")
					}

					cfg, err := yaml.Read[config.Config](config.LibrarianYAML)
					if err != nil {
						if !errors.Is(err, fs.ErrNotExist) {
							return err
						}
						cfg = &config.Config{}
					}

					updated, err := setConfigValue(cfg, path, value)
					if err != nil {
						return err
					}

					return yaml.Write(config.LibrarianYAML, updated)
				},
			},
		},
	}
}
