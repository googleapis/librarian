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
	"strconv"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
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
	cfg      *config.Config
	ghClient GitHubClient
	repo     gitrepo.Repository
	state    *config.LibrarianState
}

func newTagAndReleaseRunner(cfg *config.Config) (*tagAndReleaseRunner, error) {
	if cfg.GitHubToken == "" {
		return nil, fmt.Errorf("`LIBRARIAN_GITHUB_TOKEN` must be set")
	}
	repo, err := cloneOrOpenRepo(cfg.WorkRoot, cfg.Repo, cfg.CI)
	if err != nil {
		return nil, err
	}
	state, err := loadRepoState(repo, "")
	if err != nil {
		return nil, err
	}

	var ghClient GitHubClient
	// TODO(https://github.com/googleapis/librarian/issues/1751) handle if repo is not a URL for client, by checking remotes?
	if isURL(cfg.Repo) {
		languageRepo, err := github.ParseURL(cfg.Repo)
		if err != nil {
			return nil, fmt.Errorf("failed to parse repo url: %w", err)
		}
		ghClient, err = github.NewClient(cfg.GitHubToken, languageRepo)
		if err != nil {
			return nil, fmt.Errorf("failed to create GitHub client: %w", err)
		}
	}
	return &tagAndReleaseRunner{
		cfg:      cfg,
		repo:     repo,
		state:    state,
		ghClient: ghClient,
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
		return fmt.Errorf("failed to process some pull requests")
	}
	return nil
}

func (r *tagAndReleaseRunner) determinePullRequestsToProcess(ctx context.Context) ([]*github.PullRequest, error) {
	slog.Info("determining pull requests to process")
	if r.cfg.PullRequest != "" {
		slog.Info("processing a single pull request", "pr", r.cfg.PullRequest)
		ss := strings.Split(r.cfg.PullRequest, "/")
		if len(ss) != 5 {
			return nil, fmt.Errorf("invalid pull request format: %s", r.cfg.PullRequest)
		}
		prNum, err := strconv.Atoi(ss[4])
		if err != nil {
			return nil, fmt.Errorf("invalid pull request number: %s", ss[4])
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

func (r *tagAndReleaseRunner) processPullRequest(ctx context.Context, p *github.PullRequest) error {
	slog.Info("processing pull request", "pr", p.GetNumber())
	// TODO(https://github.com/googleapis/librarian/issues/1009)
	return nil
}
