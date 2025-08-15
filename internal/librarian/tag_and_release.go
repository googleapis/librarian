// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
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
	"log/slog"
	"regexp"
	"strings"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
)

var (
	detailsRegex = regexp.MustCompile(`(?s)<details><summary>(.*?)</summary>(.*?)</details>`)
	summaryRegex = regexp.MustCompile(`(.*?): (v?\d+\.\d+\.\d+)`)
)

// cmdTagAndRelease is the command for the `release tag-and-release` subcommand.
var cmdTagAndRelease = &cli.Command{
	Short:     "release tag-and-release tags and creates a GitHub release for a merged pull request.",
	UsageLine: "librarian release tag-and-release [arguments]",
	Long:      "Tags and creates a GitHub release for a merged pull request.",
	Run: func(ctx context.Context, cfg *config.Config) error {
		runner, err := newTagAndReleaseRunner(cfg)
		if err != nil {
			return err
		}
		return runner.run(ctx)
	},
}

func init() {
	cmdTagAndRelease.Init()
	fs := cmdTagAndRelease.Flags
	cfg := cmdGenerate.Config

	addFlagRepo(fs, cfg)
	addFlagPR(fs, cfg)
}

type tagAndReleaseRunner struct {
	cfg *config.Config
}

func newTagAndReleaseRunner(cfg *config.Config) (*tagAndReleaseRunner, error) {
	if cfg.GitHubToken == "" {
		return nil, fmt.Errorf("`LIBRARIAN_GITHUB_TOKEN` must be set")
	}
	return &tagAndReleaseRunner{
		cfg: cfg,
	}, nil
}

func (r *tagAndReleaseRunner) run(ctx context.Context) error {
	return nil
}

// libraryRelease holds the parsed information from a pull request body.
type libraryRelease struct {
	// Body contains the release notes.
	Body string
	// Library is the library id of the library being released
	Library string
	// Version is the version that is being released
	Version string
}

// parsePullRequestBody parses a string containing release notes and returns a slice of ParsedPullRequestBody.
func parsePullRequestBody(body string) []libraryRelease {
	slog.Info("parsing pull request body")
	var parsedBodies []libraryRelease
	matches := detailsRegex.FindAllStringSubmatch(body, -1)
	for _, match := range matches {
		summary := match[1]
		content := strings.TrimSpace(match[2])

		summaryMatches := summaryRegex.FindStringSubmatch(summary)
		if len(summaryMatches) == 3 {
			slog.Info("parsed pull request body", "library", summaryMatches[1], "version", summaryMatches[2])
			library := strings.TrimSpace(summaryMatches[1])
			version := strings.TrimSpace(summaryMatches[2])
			parsedBodies = append(parsedBodies, libraryRelease{
				Version: version,
				Library: library,
				Body:    content,
			})
		}
		slog.Warn("failed to parse pull request body", "match", strings.Join(match, "\n"))
	}

	return parsedBodies
}
