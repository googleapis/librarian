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

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
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
