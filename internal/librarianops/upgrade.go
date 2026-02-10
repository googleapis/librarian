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

package librarianops

import (
	"context"
	"fmt"

	"github.com/googleapis/librarian/internal/command"
	"github.com/urfave/cli/v3"
)

func upgradeCommand() *cli.Command {
	return &cli.Command{
		Name:      "upgrade",
		Usage:     "upgrade librarian version in librarian.yaml",
		UsageText: "librarianops upgrade [<repo> | -C <dir> ]",
		Description: `Examples:
  librarianops upgrade google-cloud-rust
  librarianops upgrade -C ~/workspace/google-cloud-rust

For each repository, librarianops will:
  1. Get the latest librarian version from @main.
  2. Update the version field in librarian.yaml.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "C",
				Usage: "work in `directory` (repo name inferred from basename)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_, workDir, verbose, err := ParseRepoFlags(cmd)
			if err != nil {
				return err
			}
			command.Verbose = verbose
			_, err = runUpgrade(ctx, workDir)
			if err != nil {
				return err
			}
			return nil
		},
	}
}

// Upgrade consist in getting the latest librarian version and updates the librarian.yaml file.
// It returns the new version, and an error if one occurred.
func runUpgrade(ctx context.Context, repoDir string) (string, error) {
	version, err := GetLatestLibrarianVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get latest librarian version: %w", err)
	}

	configPath := GenerateLibrarianConfigPath(repoDir)
	if err := UpdateLibrarianVersionInConfigFile(version, configPath); err != nil {
		return "", fmt.Errorf("failed to update librarian version: %w", err)
	}

	return version, nil
}
