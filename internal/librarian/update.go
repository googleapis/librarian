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
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var (
	githubAPI      = "https://api.github.com"
	githubDownload = "https://github.com"
	sourceRepos    = map[string]fetch.RepoRef{
		"sources.conformance": {Org: "protocolbuffers", Name: "protobuf", Branch: config.BranchMain},
		"sources.discovery":   {Org: "googleapis", Name: "discovery-artifact-manager", Branch: fetch.DefaultBranchMaster},
		"sources.googleapis":  {Org: "googleapis", Name: "googleapis", Branch: fetch.DefaultBranchMaster},
		"sources.protobuf":    {Org: "protocolbuffers", Name: "protobuf", Branch: config.BranchMain},
		"sources.showcase":    {Org: "googleapis", Name: "gapic-showcase", Branch: config.BranchMain},
	}
	errNoSourcesProvided = errors.New("at least one source must be provided")
	errUnknownSource     = errors.New("unknown source")
	errEmptySources      = errors.New("sources required in librarian.yaml")
)

// updateCommand returns the `update` subcommand.
func updateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "update sources or version to the latest version",
		Description: `update refreshes the upstream source repositories declared in
librarian.yaml to their latest commits and updates the recorded commit
SHAs in librarian.yaml accordingly. It also supports updating the librarian version.

Supported targets:

  - sources.conformance: protocolbuffers/protobuf conformance tests
  - sources.discovery: googleapis/discovery-artifact-manager
  - sources.googleapis: googleapis/googleapis (the API definitions)
  - sources.protobuf: protocolbuffers/protobuf
  - sources.showcase: googleapis/gapic-showcase
  - version: the librarian tool version

At least one target must be specified.

Examples:

	librarian update sources.googleapis
	librarian update sources.googleapis sources.protobuf
	librarian update version

A typical librarian workflow for regenerating every library against the
latest API definitions is:

	librarian update sources.googleapis
	librarian generate --all`,
		UsageText: "librarian update <version | source>...",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) == 0 {
				return errNoSourcesProvided
			}
			for _, arg := range args {
				if arg == "version" {
					continue
				}
				if _, ok := sourceRepos[arg]; !ok {
					return fmt.Errorf("%w: %s", errUnknownSource, arg)
				}
			}
			cfg, err := yaml.Read[config.Config](config.LibrarianYAML)
			if err != nil {
				return err
			}
			updatedCfg, err := runUpdate(ctx, cfg, args)
			if err != nil {
				return err
			}
			return yaml.Write(config.LibrarianYAML, updatedCfg)
		},
	}
}

// runUpdate refreshes the configured targets in Config.
func runUpdate(ctx context.Context, cfg *config.Config, targets []string) (*config.Config, error) {
	for _, target := range targets {
		if target == "version" {
			env := map[string]string{"GOPROXY": "direct"}
			version, err := command.OutputWithEnv(ctx, env, command.Go, "list", "-m", "-f", "{{.Version}}", "github.com/googleapis/librarian@latest")
			if err != nil {
				return nil, err
			}
			cfg, err = setConfigValue(cfg, "version", strings.TrimSpace(version))
			if err != nil {
				return nil, err
			}
		} else {
			if cfg.Sources == nil {
				return nil, errEmptySources
			}
			repo, ok := sourceRepos[target]
			if !ok {
				return nil, fmt.Errorf("%w: %s", errUnknownSource, target)
			}
			var err error
			cfg, err = setConfigValue(cfg, target+".commit", repo.Branch)
			if err != nil {
				return nil, err
			}
		}
	}
	return cfg, nil
}
