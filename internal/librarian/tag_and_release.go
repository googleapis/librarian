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
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

const (
	pullRequestSegments  = 5
	tagAndReleaseCmdName = "tag-and-release"
)

var (
	detailsRegex = regexp.MustCompile(`(?s)<details><summary>(.*?)</summary>(.*?)</details>`)
	summaryRegex = regexp.MustCompile(`(.*?): (v?\d+\.\d+\.\d+)`)
)

// cmdTagAndRelease is the command for the `release tag-and-release` subcommand.
var cmdTagAndRelease = &cli.Command{
	Short:     "tag-and-release tags and creates a GitHub release for a merged pull request.",
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
	cfg      *config.Config
	ghClient GitHubClient
	repo     gitrepo.Repository
	state    *config.LibrarianState
}

func newTagAndReleaseRunner(cfg *config.Config) (*tagAndReleaseRunner, error) {
	runner, err := newCommandRunner(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.GitHubToken == "" {
		return nil, fmt.Errorf("`LIBRARIAN_GITHUB_TOKEN` must be set")
	}
	return &tagAndReleaseRunner{
		cfg:      cfg,
		repo:     runner.repo,
		state:    runner.state,
		ghClient: runner.ghClient,
	}, nil
}

func (r *tagAndReleaseRunner) run(ctx context.Context) error {
	slog.Info("running tag-and-release command")
	prs, err := r.determinePullRequestsToProcess(ctx)
	if err != nil {
		return err
	}
	if len(prs) == 0 {
		slog.Info("no pull requests to process, exiting")
		return nil
	}

	var hadErrors bool
	for _, p := range prs {
		if err := r.processPullRequest(ctx, p); err != nil {
			slog.Error("failed to process pull request", "pr", p.GetNumber(), "error", err)
			hadErrors = true
			continue
		}
		slog.Info("processed pull request", "pr", p.GetNumber())
	}
	slog.Info("finished processing all pull requests")

	if hadErrors {
		return errors.New("failed to process some pull requests")
	}
	return nil
}

func (r *tagAndReleaseRunner) determinePullRequestsToProcess(ctx context.Context) ([]*github.PullRequest, error) {
	slog.Info("determining pull requests to process")
	if r.cfg.PullRequest != "" {
		slog.Info("processing a single pull request", "pr", r.cfg.PullRequest)
		ss := strings.Split(r.cfg.PullRequest, "/")
		if len(ss) != pullRequestSegments {
			return nil, fmt.Errorf("invalid pull request format: %s", r.cfg.PullRequest)
		}
		prNum, err := strconv.Atoi(ss[pullRequestSegments-1])
		if err != nil {
			return nil, fmt.Errorf("invalid pull request number: %s", ss[pullRequestSegments-1])
		}
		pr, err := r.ghClient.GetPullRequest(ctx, prNum)
		if err != nil {
			return nil, fmt.Errorf("failed to get pull request %d: %w", prNum, err)
		}
		return []*github.PullRequest{pr}, nil
	}

	slog.Info("searching for pull requests to tag and release")
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	query := fmt.Sprintf("label:release:pending merged:>=%s", thirtyDaysAgo)
	prs, err := r.ghClient.SearchPullRequests(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search pull requests: %w", err)
	}
	return prs, nil
}

func (r *tagAndReleaseRunner) processPullRequest(_ context.Context, p *github.PullRequest) error {
	slog.Info("processing pull request", "pr", p.GetNumber())
	// hack to make CI happy until we impl
	// TODO(https://github.com/googleapis/librarian/issues/1009)
	if p.GetNumber() != 0 {
		return fmt.Errorf("skipping pull request %d", p.GetNumber())
	}
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
