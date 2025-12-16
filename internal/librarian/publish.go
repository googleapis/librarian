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
	"fmt"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

func publishCommand() *cli.Command {
	return &cli.Command{
		Name:      "publish",
		Usage:     "publishes client libraries",
		UsageText: "librarian publish",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "print commands without executing",
			},
			&cli.BoolFlag{
				Name:  "skip-semver-checks",
				Usage: "skip semantic versioning checks",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				return err
			}
			dryRun := cmd.Bool("dry-run")
			skipSemverChecks := cmd.Bool("skip-semver-checks")
			return publish(ctx, cfg, dryRun, skipSemverChecks)
		},
	}
}

func publish(ctx context.Context, cfg *config.Config, dryRun bool, skipSemverChecks bool) error {
	// TODO: Not yet implemented.
	fmt.Printf("publish not implemented. ctx: %v, cfg: %v, dryRun: %v, skipSemverChecks: %v\n", ctx, cfg, dryRun, skipSemverChecks)
	panic("not implemented")
}
