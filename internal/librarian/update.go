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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var (
	githubAPI      = "https://api.github.com"
	githubDownload = "https://github.com"
	sourceRepos    = map[string]fetch.RepoRef{
		"conformance": {Org: "protocolbuffers", Name: "protobuf", Branch: config.BranchMain},
		"discovery":   {Org: "googleapis", Name: "discovery-artifact-manager", Branch: fetch.DefaultBranchMaster},
		"googleapis":  {Org: "googleapis", Name: "googleapis", Branch: fetch.DefaultBranchMaster},
		"protobuf":    {Org: "protocolbuffers", Name: "protobuf", Branch: config.BranchMain},
		"showcase":    {Org: "googleapis", Name: "gapic-showcase", Branch: config.BranchMain},
	}

	errNoSourcesProvided = errors.New("at least one source must be provided")
	errUnknownSource     = errors.New("unknown source")
	errEmptySources      = errors.New("sources required in librarian.yaml")
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
  - showcase

Sources use dot notation to refer to subsources.

Examples:
  librarian update googleapis.subsystem
`,
		UsageText: "librarian update <sources...>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) == 0 {
				return errNoSourcesProvided
			}

			// Read librarian.yaml as an unstructured map so we can use dot-separated paths
			// to find the target repositories provided by the user.
			m, err := yaml.Read[map[string]any](config.LibrarianYAML)
			if err != nil {
				// If librarian.yaml doesn't exist, we check if the argument matches one of the 
				// known hardcoded repositories defined in sourceRepos as a fallback.
				for _, arg := range args {
					parts := strings.Split(arg, ".")
					var matchedSource string
					for _, part := range parts {
						if _, ok := sourceRepos[part]; ok {
							matchedSource = part
							break
						}
					}
					if matchedSource == "" {
						return errUnknownSource
					}
				}
				return err
			}
			if _, ok := (*m)["sources"]; !ok {
				return errEmptySources
			}

			var sourcesToUpdate []string
			seen := make(map[string]bool)
			for _, arg := range args {
				// Check if the argument is a valid path in the configuration file.
				// If the user passes "googleapis", we fallback to checking "sources.googleapis".
				path := arg
				if _, err := yaml.Get(*m, path); err != nil {
					path = "sources." + arg
					if _, err := yaml.Get(*m, path); err != nil {
						return fmt.Errorf("%w: %s", errUnknownSource, arg)
					}
				}

				parts := strings.Split(arg, ".")
				var matchedSource string
				for _, part := range parts {
					if _, ok := sourceRepos[part]; ok {
						matchedSource = part
						break
					}
				}
				if !seen[matchedSource] {
					sourcesToUpdate = append(sourcesToUpdate, matchedSource)
					seen[matchedSource] = true
				}
			}
			return runUpdate(*m, sourcesToUpdate)
		},
	}
}

func runUpdate(m map[string]any, sourceNames []string) error {
	endpoints := &fetch.Endpoints{
		API:      githubAPI,
		Download: githubDownload,
	}

	for _, name := range sourceNames {
		repo, exists := sourceRepos[name]
		if !exists {
			return fmt.Errorf("%w: %s", errUnknownSource, name)
		}

		// If the configuration file overrides the default repository branch,
		// extract the overridden property setups.
		if branch, err := yaml.Get(m, "sources."+name+".branch"); err == nil {
			if b, ok := branch.(string); ok && b != "" {
				repo.Branch = b
			}
		}

		commit, sha256, err := fetch.LatestCommitAndChecksum(endpoints, &repo)
		if err != nil {
			return err
		}

		// Update the configuration map with the new commit and checksum.
		updated, err := yaml.Set(m, "sources."+name+".commit", commit)
		if err != nil {
			return err
		}
		updated, err = yaml.Set(updated, "sources."+name+".sha256", sha256)
		if err != nil {
			return err
		}
		m = updated
	}

	return yaml.Write(config.LibrarianYAML, m)
}
