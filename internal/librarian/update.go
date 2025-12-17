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
	"maps"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var (
	githubAPI      = "https://api.github.com"
	githubDownload = "https://github.com"
	sourceRepos    = map[string]fetch.Repo{
		"googleapis": {Org: "googleapis", Repo: "googleapis", Branch: "master"},
		"discovery":  {Org: "googleapis", Repo: "discovery-artifact-manager", Branch: "master"},
	}
)

// updateCommand returns the `update` subcommand.
func updateCommand() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Usage:     "update sources to the latest version",
		UsageText: "librarian update [--all | source]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "update all sources",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			all := cmd.Bool("all")
			source := cmd.Args().First()

			if all && source != "" {
				return fmt.Errorf("cannot specify a source when --all is set")
			}
			if !all && source == "" {
				return fmt.Errorf("a source must be specified, or use the --all flag")
			}
			return runUpdate(all, source)
		},
	}
}

func runUpdate(all bool, sourceName string) error {
	cfg, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		return err
	}
	if cfg.Sources == nil {
		return errEmptySources
	}

	endpoints := &fetch.Endpoints{
		API:      githubAPI,
		Download: githubDownload,
	}

	sourcesMap := map[string]*config.Source{
		"googleapis": cfg.Sources.Googleapis,
		"discovery":  cfg.Sources.Discovery,
	}

	var sourceNamesToProcess []string
	if all {
		sourceNamesToProcess = slices.Collect(maps.Keys(sourceRepos))
	} else {
		lowerSourceName := strings.ToLower(sourceName)
		if _, ok := sourceRepos[lowerSourceName]; !ok {
			return fmt.Errorf("unknown source: %s", sourceName)
		}
		sourceNamesToProcess = []string{lowerSourceName}
	}

	for _, name := range sourceNamesToProcess {
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
