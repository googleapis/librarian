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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var (
	githubAPI      = "https://api.github.com"
	githubDownload = "https://github.com"
	sourceRepos    = map[string]fetch.Repo{
		"conformance": {Org: "protocolbuffers", Repo: "protobuf", Branch: fetch.DefaultBranchMain},
		"discovery":   {Org: "googleapis", Repo: "discovery-artifact-manager", Branch: fetch.DefaultBranchMaster},
		"googleapis":  {Org: "googleapis", Repo: "googleapis", Branch: fetch.DefaultBranchMaster},
		"protobuf":    {Org: "protocolbuffers", Repo: "protobuf", Branch: fetch.DefaultBranchMain},
		"showcase":    {Org: "googleapis", Repo: "gapic-showcase", Branch: fetch.DefaultBranchMain},
	}

	errNoSourcesProvided = errors.New("at least one source must be provided")
	errUnknownSource     = errors.New("unknown source")
)

// updateCommand returns the `update` subcommand.
func updateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "update sources to the latest version",
		Description: `Supported sources are:
  - conformance
  - discovery
  - googleapis
  - protobuf
  - showcase`,
		UsageText: "librarian update <sources...>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) == 0 {
				return errNoSourcesProvided
			}
			for _, arg := range args {
				if _, ok := sourceRepos[arg]; !ok {
					return fmt.Errorf("%w: %s", errUnknownSource, arg)
				}
			}
			cfg, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				return err
			}
			return runUpdate(cfg, args)
		},
	}
}

func runUpdate(cfg *config.Config, sourceNames []string) error {
	if cfg.Sources == nil {
		return errEmptySources
	}

	endpoints := &fetch.Endpoints{
		API:      githubAPI,
		Download: githubDownload,
	}

	sourcesMap := map[string]*config.Source{
		"conformance": cfg.Sources.Conformance,
		"discovery":   cfg.Sources.Discovery,
		"googleapis":  cfg.Sources.Googleapis,
		"protobuf":    cfg.Sources.ProtobufSrc,
		"showcase":    cfg.Sources.Showcase,
	}

	for _, name := range sourceNames {
		source := sourcesMap[name]
		repo := sourceRepos[name]
		if err := updateSource(endpoints, repo, source, cfg); err != nil {
			return err
		}
	}
	return nil
}

func updateSource(endpoints *fetch.Endpoints, repo fetch.Repo, source *config.Source, cfg *config.Config) error {
	if source == nil {
		return nil
	}

	// Source configuration specifically references a branch of the
	// source repository.
	if source.Branch != "" {
		repo.Branch = source.Branch
	}

	oldCommit := source.Commit
	oldSHA256 := source.SHA256

	commit, sha256, err := fetch.LatestCommitAndChecksum(endpoints, &repo)
	if err != nil {
		return err
	}

	if oldCommit != commit || oldSHA256 != sha256 {
		source.Commit = commit
		source.SHA256 = sha256
		if err := yaml.Write(librarianConfigPath, cfg); err != nil {
			return err
		}
	}
	return nil
}
